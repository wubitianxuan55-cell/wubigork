// Package weixin — 真机抓包基建（v4.8.2）：把 iLink 协议探明窗口需要的第一手
// 原始负载逐行追加到 JSONL 抓包文件（docs/ilink-non-text-protocol.md §3）。
// 三类 kind：
//   - qr_status      扫码登录成功响应原文（含 baseurl/redirect_host —— 上传域
//     最有价值线索）
//   - inbound_media  getUpdates 批次含 image_item/file_item/未知 type（整批原文）
//   - upload_probe   SendFileCard 上传探针每次尝试（请求路径+响应+errcode）
//
// 铁律：抓包绝不影响主流程——路径未注册时整体 no-op，任何读写失败静默吞掉。
package weixin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// captureRecord 抓包文件单行结构。
type captureRecord struct {
	Ts   int64       `json:"ts"`   // UnixMilli
	Kind string      `json:"kind"` // qr_status | inbound_media | upload_probe
	Data interface{} `json:"data"` // 结构化数据或 json.RawMessage 原文
}

var captureMu sync.Mutex
var capturePath string // 当前抓包文件路径；""=未注册（capture 整体 no-op）

// defaultCapturePath 抓包默认路径：UserCacheDir/gaea/wx_capture.jsonl
// （Windows 为 %LocalAppData%\gaea\wx_capture.jsonl），UserCacheDir 不可用
// 时退 TempDir。app 层未配置 Config.CapturePath 时的兜底；dataRoot 在 app 层
// 持有，weixin 包不反向依赖，需要落 dataRoot 时 app 层一行显式传入即可。
func defaultCapturePath() string {
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		return filepath.Join(dir, "gaea", "wx_capture.jsonl")
	}
	return filepath.Join(os.TempDir(), "gaea-wx_capture.jsonl")
}

// setCapturePath 注册抓包路径（Server 构造时调用）。多助手多 Server 共用同
// 一默认路径，后注册覆盖前者（同进程路径一致，覆盖无实际影响）。
func setCapturePath(p string) {
	captureMu.Lock()
	capturePath = p
	captureMu.Unlock()
}

// capture 追加一行 {"ts":...,"kind":...,"data":...} 到抓包文件。任何失败
// 静默（抓包不能影响主流程）；data 传 json.RawMessage 时原样嵌入原始 JSON。
func capture(kind string, data interface{}) {
	captureMu.Lock()
	defer captureMu.Unlock()
	if capturePath == "" {
		return
	}
	line, err := json.Marshal(captureRecord{Ts: time.Now().UnixMilli(), Kind: kind, Data: data})
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(capturePath), 0o755)
	f, err := os.OpenFile(capturePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = f.Write(append(line, '\n'))
	_ = f.Close()
}

// ─── 扫码登录响应钩子 ────────────────────────────────────────

// recordQRStatus 扫码登录响应钩子（qrlogin.go 两个 Poll 函数解析成功后调用）：
// 响应携带 baseurl/redirect_host（即登录成功/近成功态）时抓整响应原文
// （kind="qr_status"）。v4.8.3 起上传走 getuploadurl + 固定 CDN 域，不再依赖
// 这两个字段——抓包仅作协议证据留存。无媒体域的中间轮询态不抓，避免刷屏。
func recordQRStatus(raw []byte, resp *QRStatusResp) {
	if resp == nil || (resp.BaseURL == "" && resp.RedirectHost == "") {
		return
	}
	capture("qr_status", json.RawMessage(raw))
}
