package app

// wx_file_handler.go — 微信入站文件自持 + 内容提取注入（「微信文件收发」刀，
// app 接线线）。跨线契约（通道线 clawbot.Server.FileHandler，签名逐字）：
//
//	type InboundFileHandler func(localPath, fileName string, sizeBytes int64, md5sum string) string
//
// 通道线在入站文件消息下载成功后回调本处理器；返回值字符串作为注入 handle
// 的文本行（nil / panic / 空串由 clawbot 回退占位行——下载失败根本不会回调）。
//
// 职责三段：
//  1. 自持：localPath 是通道线解密落盘的临时文件（识别完即删），这里复制进
//     whisperDataRoot/wx_files/（FNV 哈希名防路径注入 + 纳秒序号防同名覆盖，
//     保留扩展名），单文件 50MiB 二次防线（通道线自有上限之外的最后闸）。
//  2. 提取：内容文本提取尽量复用仓内现有解析器——docx/doc/xlsx/xls/pptx/ppt/
//     pdf 走 internal/office/docmd.Convert（format_convert 工具与桌面预览面板
//     同源管线），txt/md/csv 等纯文本直读；不支持的格式诚实告知路径。
//  3. 注入行：格式「[用户发来文件 X（424 KB）]\n<提取内容前 6000 字符>」，
//     超限标注截断；提取失败降级为带自持路径的诚实文案——任何一路失败都不
//     影响消息主流程（外层契约兜底占位行）。
//
// 清理策略（有意持久化，不设 TTL）：wx_files 供「把文件发我」回推
// （ActionSendLatestFile）与桌面端打开消费，故 stopAssistantWx 不清理；防洪
// 用「数量 + 总量」双阈值——超过 50 个文件或总量 500MiB 时按修改时间从旧到新
// 删除，直到回到限额内（最近收到的文件永远优先保留）。

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/office/docmd"
)

// 自持目录参数：单文件上限 / 注入截断字符数 / 清理双阈值。
const (
	wxFileMaxBytes    = 50 << 20  // 50 MiB：单文件复制二次防线
	wxFileInjectChars = 6000      // 注入行提取内容截断（字符，非字节）
	wxFileMaxCount    = 50        // 清理阈值：文件数
	wxFileMaxTotal    = 500 << 20 // 清理阈值：目录总字节
)

// newWxFileHandler 构造入站文件处理器（whisperState.startAssistantWx 注入
// srv.FileHandler；dataRoot = whisperDataRoot，自持目录固定其下 wx_files/）。
func newWxFileHandler(dataRoot string) weixin.InboundFileHandler {
	s := &wxFileStore{dir: filepath.Join(dataRoot, "wx_files"), maxFile: wxFileMaxBytes}
	return s.Ingest
}

// wxFileStore 入站文件自持仓库（dir + 单文件上限可注入，测试用小上限/临时
// 目录直构）。无内部状态，Ingest 并发安全（依赖原子语义：哈希唯一名 + 目录
// 限额尽力而为）。
type wxFileStore struct {
	dir     string
	maxFile int64
}

// Ingest 契约入口：自持复制 → 内容提取 → 拼注入行。任何失败返回诚实降级
// 文案（绝不返回空串——空串会让外层回退占位行，丢失文件名/路径信息）。
// md5sum 为通道线校验和（已用于下载完整性），此处不再复验，保留签名对齐。
func (s *wxFileStore) Ingest(localPath, fileName string, sizeBytes int64, md5sum string) string {
	_ = md5sum
	if fileName == "" {
		fileName = filepath.Base(localPath)
	}
	dst, err := s.persist(localPath, fileName)
	if err != nil {
		slog.Warn("[wx-file] 入站文件自持失败", "file", fileName, "size", sizeBytes, "err", err)
		if errors.Is(err, errWxFileTooLarge) {
			return fmt.Sprintf("收到文件 %s（%s），但超出自持上限 %d MB，未能保存。", fileName, formatWxFileSize(sizeBytes), s.maxFile>>20)
		}
		return fmt.Sprintf("收到文件 %s，但保存失败，暂无法读取它的内容。", fileName)
	}
	return s.injectLine(dst, fileName, sizeBytes)
}

// errWxFileTooLarge 源文件超过自持上限（Ingest 据此区分文案）。
var errWxFileTooLarge = errors.New("文件超过自持上限")

// persist 复制自持：Stat 预检上限 → 建目录 → FNV 哈希名落盘（复制期间同样
// 限流，源文件被增长/置换逃不掉）→ 限额清理。返回自持路径。
func (s *wxFileStore) persist(src, fileName string) (string, error) {
	if src == "" {
		return "", errors.New("空源路径")
	}
	st, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if st.Size() > s.maxFile {
		return "", errWxFileTooLarge
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(s.dir, wxFileSelfName(fileName, time.Now()))
	if err := copyFileBounded(src, dst, s.maxFile); err != nil {
		return "", err
	}
	s.enforceQuota()
	return dst, nil
}

// wxFileSelfName 自持文件名：原文件名 FNV 哈希（防路径注入——微信文件名用户
// 可控，绝不直接拼路径）+ 纳秒序号（同名文件重复发送不覆盖）+ 白名单扩展名
// （只保留字母数字，防「.docx\u0000.exe」类花式扩展名；提取不出留 .bin）。
func wxFileSelfName(fileName string, at time.Time) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fileName))
	ext := wxFileSafeExt(fileName)
	return fmt.Sprintf("wx-file-%08x-%d%s", h.Sum32(), at.UnixNano(), ext)
}

// wxFileSafeExt 归一化扩展名：小写、只保留字母数字；空/超长/含非法字符一律
// 回退 .bin（诚实降级为「格式暂不支持」而非误路由解析器）。
func wxFileSafeExt(fileName string) string {
	raw := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	if raw == "" || len(raw) > 16 {
		return ".bin"
	}
	var b strings.Builder
	b.WriteByte('.')
	for _, r := range raw[1:] {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() <= 1 {
		return ".bin"
	}
	return b.String()
}

// enforceQuota 自持目录限额清理（见文件头「清理策略」）：文件数 > 50 或总量
// > 500MiB 时按修改时间从旧到新删除，直到回到限额内。尽力而为：枚举/删除
// 失败只记日志（下次 Ingest 再收口），绝不影响本次注入。
func (s *wxFileStore) enforceQuota() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	type fileEnt struct {
		path string
		size int64
		mod  time.Time
	}
	var files []fileEnt
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		files = append(files, fileEnt{path: filepath.Join(s.dir, e.Name()), size: info.Size(), mod: info.ModTime()})
		total += info.Size()
	}
	if len(files) <= wxFileMaxCount && total <= wxFileMaxTotal {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) }) // 旧→新
	for i, f := range files {
		if len(files)-i <= wxFileMaxCount && total <= wxFileMaxTotal {
			break
		}
		if rmErr := os.Remove(f.path); rmErr != nil {
			// 删除失败：剩余计数与总量都不减（尽力而为，下次 Ingest 再收口）。
			slog.Warn("[wx-file] 自持目录清理失败", "file", f.path, "err", rmErr)
			continue
		}
		total -= f.size
	}
}

// injectLine 拼注入行：文件头 + 提取内容（截断 wxFileInjectChars 字符）。
// 提取失败/内容为空 → 诚实降级带自持路径（模型至少能告诉用户文件在哪）。
func (s *wxFileStore) injectLine(selfPath, fileName string, sizeBytes int64) string {
	header := fmt.Sprintf("[用户发来文件 %s（%s）]", fileName, formatWxFileSize(sizeBytes))
	text, err := extractWxFileText(selfPath)
	if err != nil {
		slog.Warn("[wx-file] 入站文件内容提取失败", "file", fileName, "err", err)
		if errors.Is(err, errWxFileUnsupported) {
			return fmt.Sprintf("收到文件 %s（%s）。该格式暂不支持内容读取，可在桌面端打开：%s",
				fileName, formatWxFileSize(sizeBytes), filepath.ToSlash(selfPath))
		}
		return fmt.Sprintf("%s（内容读取失败，可在桌面端打开：%s）", header, filepath.ToSlash(selfPath))
	}
	body := strings.TrimSpace(text)
	if body == "" {
		return fmt.Sprintf("%s（文件内容为空，可在桌面端打开：%s）", header, filepath.ToSlash(selfPath))
	}
	if rs := []rune(body); len(rs) > wxFileInjectChars {
		body = string(rs[:wxFileInjectChars]) + "…(内容过长已截断)"
	}
	return header + "\n" + body
}

// errWxFileUnsupported 不在支持矩阵内的格式（injectLine 据此给诚实文案）。
var errWxFileUnsupported = errors.New("格式暂不支持内容读取")

// extractWxFileText 内容提取（复用仓内现有解析器，支持矩阵见文件头）：
//   - docx/doc/xlsx/xls/pptx/ppt/pdf → docmd.Convert（format_convert 工具与
//     桌面预览面板同源管线；pptx 走 markitdown，缺失时诚实报错）；
//   - txt/md/markdown/csv/log/json → 纯文本直读（utf-8 原样，不转码）；
//   - 其余 → errWxFileUnsupported。
func extractWxFileText(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt", ".pdf":
		return docmd.Convert(path, "")
	case ".txt", ".md", ".markdown", ".csv", ".log", ".json":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", errWxFileUnsupported
	}
}

// formatWxFileSize 人类可读大小（注入行文案口径）：<1KiB 用 B，<1MiB 用整数
// KB（「424 KB」），其余用一位小数 MB（「4.2 MB」）。
func formatWxFileSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%d KB", n>>10)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
