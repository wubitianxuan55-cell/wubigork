package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/office/docmd"
	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/gaea/gaea/internal/office/xlsxpreview"
)

// PreviewResult 是文件预览负载：kind 决定前端渲染方式。
//   - image       → dataUrl 提供图片
//   - docx        → dataUrl 提供原始 docx（前端 docx-preview 保真渲染）
//   - xlsx        → body 为结构化单元格 JSON（值/公式/样式，前端表格渲染）
//   - markdown    → body 为 Markdown（.md 原文或 .doc/.xls/pdf 转换结果）
//   - pdf         → .pptx 转换的逐页缩略（Pages）或整本 PDF dataUrl（gaea_pptx.go）
//   - text        → body 为纯文本
//   - unsupported → 无法内联预览，前端提供"外部打开"
//   - error       → error 描述原因
type PreviewResult struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Ext     string `json:"ext"`
	Size    int64  `json:"size"`
	Kind    string `json:"kind"`
	Body    string `json:"body"`
	DataURL string `json:"dataUrl"`
	Error   string `json:"error"`
	// Truncated/TotalPages 标记 PDF 预览页数截断（前端展示明确提示）。
	Truncated  bool `json:"truncated,omitempty"`
	TotalPages int  `json:"totalPages,omitempty"`
	// Pages 是 .pptx 预览的逐页缩略（Page 为 1-based 页码，DataURL 为该页
	// PNG，data:image/png;base64）。仅在 pdftoppm 可用时填充；为空且带
	// DataURL 时前端回退内嵌 PDF 查看器（实现见 gaea_pptx.go previewPptx）。
	Pages []PreviewPage `json:"pages,omitempty"`
	// Hint 是给前端的扩展能力提示："outline" = 可调 GaeaPptxOutline 拉取
	// pptx 结构化大纲卡。
	Hint string `json:"hint,omitempty"`
}

var textExts = map[string]bool{
	".txt": true, ".log": true, ".ini": true, ".cfg": true, ".conf": true,
	".json": true, ".jsonl": true, ".toml": true, ".yaml": true, ".yml": true,
	".csv": true, ".tsv": true, ".xml": true, ".html": true, ".css": true, ".js": true,
	".ts": true, ".tsx": true, ".jsx": true, ".go": true, ".py": true, ".java": true,
	".c": true, ".h": true, ".cpp": true, ".hpp": true, ".rs": true, ".rb": true,
	".php": true, ".sh": true, ".bat": true, ".ps1": true, ".sql": true, ".mdx": true,
}

// maxPreviewBytes 是 md/文本类预览的正文上限：超过时截断并带可见标记，
// 避免超大文件把整个正文塞进前端（对标 xlsx 既有 Truncated 标记）。
const maxPreviewBytes = 2 << 20 // 2MB

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".bmp": true, ".svg": true, ".ico": true,
}

// previewSearchDirs 裸文件名预览的常见输出目录（按优先级）。
// S4 产物路径分区：.gaea/play/exports 与 .gaea/exports 并列可检索。
var previewSearchDirs = []string{
	"exports", ".gaea/exports", ".gaea/play/exports", "outputs", "reports", "docs",
	".gaea/uploads", "uploads", "attachments", ".gaea/attachments",
	"templates", ".gaea/templates",
}

// resolvePreviewPath 把工作区相对路径解析为绝对路径并返回展示用相对路径；
// 裸文件名（无目录分隔符）直接找不到时，在常见输出目录里查找同名文件，
// 使“输出文件：成本测算.xlsx”这类纯文件名引用也能直接预览。
func resolvePreviewPath(rel string) (path, displayRel string) {
	root := gaeaCwd()
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel), rel
	}
	display := filepath.ToSlash(rel)
	joined := filepath.Join(root, rel)
	if fileExists(joined) {
		return joined, display
	}
	if !strings.ContainsAny(rel, `/\`) {
		if found := findInOutputDirs(root, rel); found != "" {
			if r, err := filepath.Rel(root, found); err == nil {
				display = filepath.ToSlash(r)
			}
			return found, display
		}
	}
	return joined, display
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func findInOutputDirs(root, name string) string {
	best := ""
	var bestMod time.Time
	for _, dir := range previewSearchDirs {
		p := filepath.Join(root, dir, name)
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best, bestMod = p, info.ModTime()
		}
	}
	return best
}

// GaeaPreview 读取工作区相对路径并返回结构化预览。
func (a *App) GaeaPreview(rel string) PreviewResult {
	path, displayRel := resolvePreviewPath(rel)
	info, err := os.Stat(path)
	if err != nil {
		return PreviewResult{Path: displayRel, Kind: "error", Error: "文件不存在"}
	}
	if info.IsDir() {
		return PreviewResult{Path: displayRel, Kind: "error", Error: "目录无法预览"}
	}

	ext := strings.ToLower(filepath.Ext(rel))
	name := filepath.Base(rel)
	base := PreviewResult{Path: displayRel, Name: name, Ext: ext, Size: info.Size()}

	if imageExts[ext] {
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "image"
		base.DataURL = "data:" + mimeFor(ext) + ";base64," + base64.StdEncoding.EncodeToString(b)
		return base
	}

	switch ext {
	case ".md", ".markdown":
		body, truncated, err := readPreviewCapped(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = previewTruncateNote(body, truncated)
		base.Truncated = truncated
		return base
	case ".mmd", ".mermaid":
		// Mermaid 图表源码：按 markdown 渲染，聊天/预览中的 ```mermaid
		// 代码块会直接渲染成图。
		body, truncated, err := readPreviewCapped(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = "```mermaid\n" + body + "\n```" + previewTruncateNote("", truncated)
		base.Truncated = truncated
		return base
	case ".docx":
		// 原始 docx 交给前端 docx-preview 保真渲染（版式/表格/页眉页脚/修订）；
		// Body 同时附带轻量 Markdown 文本（截断头部），供交付卡片缩略图
		// 显示"看得见的文件内容"，完整版式仍走 dataUrl。
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "docx"
		base.DataURL = "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64," +
			base64.StdEncoding.EncodeToString(b)
		if md, mdErr := docmd.Convert(path, ""); mdErr == nil {
			base.Body = previewThumbText(md)
		}
		return base
	case ".xlsx":
		// 公式无缓存值时先补一次 LibreOffice 重算，保证预览显示计算结果
		if need, _ := xlsxpreview.NeedsRecalc(path); need {
			_, _ = xlsxedit.Recalc(path, gaeaCwd())
		}
		j, err := xlsxpreview.Render(path)
		if err != nil {
			base.Kind = "unsupported"
			base.Error = err.Error()
			return base
		}
		base.Kind = "xlsx"
		base.Body = j
		return base
	case ".pptx":
		// v4.28 B2：pptx → soffice PDF（落 .gaea/cache 缓存）→ poppler 逐页
		// 缩略；kind=pdf + Pages + Hint="outline"（前端叠大纲卡）。实现见
		// gaea_pptx.go previewPptx。
		return previewPptx(path, info, base)
	case ".doc", ".xls", ".pdf":
		// 扫描件 OCR 逐页进度经事件通道回传前端（仅 Wails/HTTP 桥接环境真正发布）。
		md, total, truncated, err := docmd.ConvertLimitProgress(path, "", docmd.DefaultMaxPDFPages,
			func(done, n int) {
				if n <= 0 {
					return
				}
				a.emit("gaea-event", map[string]interface{}{
					"kind": "preview_progress",
					"path": rel,
					"progress": map[string]interface{}{
						"path":  rel,
						"done":  done,
						"total": n,
					},
				})
			})
		if err != nil {
			base.Kind = "unsupported"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = md
		base.TotalPages = total
		base.Truncated = truncated
		if truncated {
			base.Body += fmt.Sprintf("\n\n> ⚠️ 预览已截断：PDF 共 %d 页，仅显示前 %d 页。可让 AI 使用 summarize_file 获取全文摘要，或拆分文件后处理。",
				total, docmd.DefaultMaxPDFPages)
		}
		return base
	}

	if textExts[ext] {
		body, truncated, err := readPreviewCapped(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "text"
		base.Body = previewTruncateNote(body, truncated)
		base.Truncated = truncated
		return base
	}

	base.Kind = "unsupported"
	base.Error = "该格式暂不支持内联预览，可点击右上角在外部程序中打开"
	return base
}

// previewThumbText 截取预览正文头部作为交付卡片缩略图文本：
// 取前 maxThumbBytes 字节（UTF-8 安全截断），去掉可能残留的半行。
const maxThumbBytes = 4 << 10 // 4KB，足够展示前几行内容

func previewThumbText(md string) string {
	if len(md) <= maxThumbBytes {
		return md
	}
	head := md[:maxThumbBytes]
	// 回退到 UTF-8 字符边界，避免截断处出现乱码。
	i := len(head)
	for i > 0 && head[i-1]&0xC0 == 0x80 {
		i--
	}
	return head[:i]
}

// readPreviewCapped 读取预览正文：超过 maxPreviewBytes 时截断（不切断 UTF-8
// 字符），返回截断后的正文与 truncated 标记（标记文案由 previewTruncateNote 追加）。
func readPreviewCapped(path string) (body string, truncated bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	if len(b) <= maxPreviewBytes {
		return string(b), false, nil
	}
	head := b[:maxPreviewBytes]
	// 回退到 UTF-8 字符边界，避免截断处出现乱码。
	i := len(head)
	for i > 0 && head[i-1]&0xC0 == 0x80 {
		i--
	}
	return string(head[:i]), true, nil
}

// previewTruncateNote 在截断时追加可见标记（md/text 在正文后；mermaid 传空正文，
// 标记追加在围栏外）。
func previewTruncateNote(body string, truncated bool) string {
	if !truncated {
		return body
	}
	if body != "" {
		body += "\n\n"
	}
	return body + "> ⚠️ 预览已截断：文件过大（>2MB），仅显示前 2MB。可让 AI 使用 summarize_file 获取全文摘要。"
}

func mimeFor(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
