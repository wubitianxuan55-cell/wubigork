package app

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/office/docmd"
)

// PreviewResult 是文件预览负载：kind 决定前端渲染方式。
//   - image       → dataUrl 提供图片
//   - markdown    → body 为 Markdown（.md 原文或 docx/xlsx/pdf 转换结果）
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

// GaeaPreview 读取工作区相对路径并返回结构化预览。
func (a *App) GaeaPreview(rel string) PreviewResult {
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	info, err := os.Stat(path)
	if err != nil {
		return PreviewResult{Path: rel, Kind: "error", Error: "文件不存在"}
	}
	if info.IsDir() {
		return PreviewResult{Path: rel, Kind: "error", Error: "目录无法预览"}
	}

	ext := strings.ToLower(filepath.Ext(rel))
	name := filepath.Base(rel)
	base := PreviewResult{Path: rel, Name: name, Ext: ext, Size: info.Size()}

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
	case ".docx", ".doc", ".xlsx", ".xls", ".pdf":
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
