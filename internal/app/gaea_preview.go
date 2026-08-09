package app

import (
	"encoding/base64"
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
}

var textExts = map[string]bool{
	".txt": true, ".log": true, ".ini": true, ".cfg": true, ".conf": true,
	".json": true, ".jsonl": true, ".toml": true, ".yaml": true, ".yml": true,
	".csv": true, ".tsv": true, ".xml": true, ".html": true, ".css": true, ".js": true,
	".ts": true, ".tsx": true, ".jsx": true, ".go": true, ".py": true, ".java": true,
	".c": true, ".h": true, ".cpp": true, ".hpp": true, ".rs": true, ".rb": true,
	".php": true, ".sh": true, ".bat": true, ".ps1": true, ".sql": true, ".mdx": true,
}

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".bmp": true, ".svg": true, ".ico": true,
}

// previewSearchDirs 裸文件名预览的常见输出目录（按优先级）。
var previewSearchDirs = []string{
	"exports", ".gaea/exports", "outputs", "reports", "docs",
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
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = string(b)
		return base
	case ".mmd", ".mermaid":
		// Mermaid 图表源码：按 markdown 渲染，聊天/预览中的 ```mermaid
		// 代码块会直接渲染成图。
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = "```mermaid\n" + string(b) + "\n```"
		return base
	case ".docx":
		// 原始 docx 交给前端 docx-preview 保真渲染（版式/表格/页眉页脚/修订）。
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "docx"
		base.DataURL = "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64," +
			base64.StdEncoding.EncodeToString(b)
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
	case ".doc", ".xls", ".pdf":
		md, err := docmd.Convert(path, "")
		if err != nil {
			base.Kind = "unsupported"
			base.Error = err.Error()
			return base
		}
		base.Kind = "markdown"
		base.Body = md
		return base
	}

	if textExts[ext] {
		b, err := os.ReadFile(path)
		if err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		base.Kind = "text"
		base.Body = string(b)
		return base
	}

	base.Kind = "unsupported"
	base.Error = "该格式暂不支持内联预览，可点击右上角在外部程序中打开"
	return base
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
