// diag_tap.go — 播放面诊断落盘端点（2026-11-07 重叠线复开第一动作）。
//
// 背景：决定性解析证明服务端线缆音频干净，「越到后面越混杂/加速/重叠」的污染
// 进入点在客户端播放运行时（泵/ring/worklet/设备）。本端点接收前端 worklet
// tap 回传的**实际外放样本**（设备率 Int16 LE），原样落盘，供分窗声纹解析
// （scripts/voice_v2/playback_tap_analyze.py）定位杂音/重叠/加速的确切时间窗。
//
// 协议：POST /v1/diag/playback，body=原始 Int16 LE 字节，header 携带元数据——
//
//	X-Gaea-Diag-File     会话文件标识（前端 connect 时生成，同会话共用）
//	X-Gaea-Diag-Rate     采样率（设备实际 sampleRate）
//	X-Gaea-Diag-Truncate "1"=清空重开（新会话首轮）
//	X-Gaea-Diag-Seq      块序号（乱序/丢块观测，不参与写入）
//
// 文件：os.UserConfigDir()/gaea/diag/voice-playback-tap-<file>.pcm；保留最近
// 5 个（诊断刀，不进发布纪律面——修复定形后整刀可撤）。
package realtime

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// diagTapKeep 诊断 tap 文件保留个数（超出按修改时间淘汰最旧）。
const diagTapKeep = 5

// diagFileID 文件标识合法字符（防路径穿越：仅 [0-9a-z-]，≤64 字符）。
var diagFileID = regexp.MustCompile(`^[0-9a-z-]{1,64}$`)

// diagTapDir 诊断落盘根目录（<UserConfigDir>/gaea/diag）。测试可注入。
func diagTapDir() (string, error) {
	root, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "gaea", "diag"), nil
}

// userConfigDir 独立小函数：os.UserConfigDir 的可测替身锚点。
var userConfigDir = func() (string, error) { return os.UserConfigDir() }

// handleDiagPlayback 播放诊断样本落盘。
func (srv *Server) handleDiagPlayback(w http.ResponseWriter, r *http.Request) {
	// 诊断端点跨源：前端页面 origin（wails.localhost）≠ 127.0.0.1:8791，且带
	// X-Gaea-Diag-* 自定义头=非简单请求，浏览器必先 OPTIONS 预检——不回预检
	// 则 fetch 全灭（2026-11-07 实弹二层根修，与 URL 拼接 404 叠加）。
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Gaea-Diag-File, X-Gaea-Diag-Rate, X-Gaea-Diag-Truncate, X-Gaea-Diag-Seq")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	file := r.Header.Get("X-Gaea-Diag-File")
	if !diagFileID.MatchString(file) {
		http.Error(w, "bad file id", http.StatusBadRequest)
		return
	}
	rate := r.Header.Get("X-Gaea-Diag-Rate")
	if _, err := strconv.Atoi(rate); err != nil {
		http.Error(w, "bad rate", http.StatusBadRequest)
		return
	}
	body, err := readDiagBody(r)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(body)%2 != 0 { // Int16 LE 对齐守卫
		body = body[:len(body)-len(body)%2]
	}
	dir, err := diagTapDir()
	if err != nil {
		http.Error(w, "config dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, "mkdir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	path := filepath.Join(dir, "voice-playback-tap-"+file+".pcm")
	truncate := r.Header.Get("X-Gaea-Diag-Truncate") == "1"
	flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if truncate {
		flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
		// sidecar 元数据：采样率（解析脚本 playback_tap_analyze.py 的 rate 单源）
		meta := fmt.Sprintf(`{"rate":%s,"file":%q,"created":"%s"}`, rate, file, time.Now().Format(time.RFC3339))
		if err := os.WriteFile(filepath.Join(dir, "voice-playback-tap-"+file+".meta.json"), []byte(meta), 0o644); err != nil {
			slog.Warn("diag tap meta 写入失败", "error", err)
		}
	}
	f, err := os.OpenFile(path, flag, 0o644)
	if err != nil {
		http.Error(w, "open: "+err.Error(), http.StatusInternalServerError)
		return
	}
	n, err := f.Write(body)
	closeErr := f.Close()
	if err != nil {
		http.Error(w, "write: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if closeErr != nil {
		http.Error(w, "close: "+closeErr.Error(), http.StatusInternalServerError)
		return
	}
	pruneDiagTapFiles(dir)
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"ok":true,"file":%q,"bytes":%d}`, filepath.Base(path), n)
}

// readDiagBody 限长读取（4MiB/请求；前端 1s@48k Int16≈96KB，余量充足）。
func readDiagBody(r *http.Request) ([]byte, error) {
	const max = 4 << 20
	return io.ReadAll(http.MaxBytesReader(nil, r.Body, max))
}

// pruneDiagTapFiles 保留最近 diagTapKeep 个 tap 文件。
func pruneDiagTapFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type tapFile struct {
		path string
		mod  time.Time
	}
	var files []tapFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "voice-playback-tap-") || strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, tapFile{filepath.Join(dir, e.Name()), info.ModTime()})
	}
	if len(files) <= diagTapKeep {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	for _, f := range files[diagTapKeep:] {
		if err := os.Remove(f.path); err != nil {
			slog.Warn("diag tap 淘汰失败", "file", f.path, "error", err)
		}
	}
}
