package export

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmaupin/go-epub"
	"github.com/gaea/gaea/internal/project"
)

// Manager 导出管理器
type Manager struct {
	pm *project.Manager
}

// New 创建导出管理器
func New(pm *project.Manager) *Manager {
	return &Manager{pm: pm}
}

// ExportTXT 导出为纯文本
func (m *Manager) ExportTXT(outPath string) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", m.pm.Meta.Title))
	sb.WriteString(fmt.Sprintf("题材: %s  文风: %s\n", m.pm.Meta.Genre, m.pm.Meta.Style))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	worldview, err := m.pm.ReadWorldview()
	if err != nil {
		slog.Warn("exportTXT: 读取世界观失败", "error", err)
	}
	if worldview != "" {
		sb.WriteString("【世界观】\n")
		sb.WriteString(worldview)
		sb.WriteString("\n\n" + strings.Repeat("=", 50) + "\n\n")
	}

	for i := 1; ; i++ {
		content, err := m.pm.ReadChapter(i)
		if err != nil {
			break
		}
		sb.WriteString(fmt.Sprintf("第 %d 章\n\n", i))
		sb.WriteString(content)
		sb.WriteString("\n\n" + strings.Repeat("-", 30) + "\n\n")
	}

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("写入 TXT 失败: %w", err)
	}
	return outPath, nil
}

// ExportMarkdown 导出为 Markdown 合集
func (m *Manager) ExportMarkdown(outPath string) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", m.pm.Meta.Title))
	sb.WriteString(fmt.Sprintf("> 题材: %s | 文风: %s\n\n", m.pm.Meta.Genre, m.pm.Meta.Style))

	for i := 1; ; i++ {
		content, err := m.pm.ReadChapter(i)
		if err != nil {
			break
		}
		sb.WriteString(fmt.Sprintf("## 第 %d 章\n\n", i))
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("写入 Markdown 失败: %w", err)
	}
	return outPath, nil
}

// ExportEPUB 导出为 EPUB 电子书
func (m *Manager) ExportEPUB(outPath string) (string, error) {
	e := epub.NewEpub(m.pm.Meta.Title)
	e.SetAuthor("gaea")

	// 封面页
	coverHTML := fmt.Sprintf(`<html><body>
		<h1>%s</h1>
		<p>题材: %s | 文风: %s</p>
		<p>由 gaea AI 辅助创作</p>
	</body></html>`, m.pm.Meta.Title, m.pm.Meta.Genre, m.pm.Meta.Style)
	e.AddSection(coverHTML, "封面", "cover.xhtml", "")

	// 世界观
	worldview, err := m.pm.ReadWorldview()
	if err != nil {
		slog.Warn("exportEPUB: 读取世界观失败", "error", err)
	}
	if worldview != "" {
		wvHTML := fmt.Sprintf("<html><body><h2>世界观</h2><pre>%s</pre></body></html>",
			escapeHTML(worldview))
		e.AddSection(wvHTML, "世界观", "worldview.xhtml", "")
	}

	// 各章节
	for i := 1; ; i++ {
		content, err := m.pm.ReadChapter(i)
		if err != nil {
			break
		}
		chHTML := fmt.Sprintf("<html><body><h2>第 %d 章</h2>%s</body></html>",
			i, chapterToHTML(content))
		e.AddSection(chHTML, fmt.Sprintf("第 %d 章", i),
			fmt.Sprintf("chapter_%03d.xhtml", i), "")
	}

	if err := e.Write(outPath); err != nil {
		return "", fmt.Errorf("写入 EPUB 失败: %w", err)
	}
	return outPath, nil
}

// ExportAll 一键导出所有格式到项目目录
func (m *Manager) ExportAll() (map[string]string, error) {
	dir := filepath.Join(m.pm.Dir, "export")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建导出目录失败: %w", err)
	}

	result := make(map[string]string)

	formats := []struct {
		ext string
		fn  func(string) (string, error)
	}{
		{".txt", m.ExportTXT},
		{".md", m.ExportMarkdown},
		{".epub", m.ExportEPUB},
	}

	for _, f := range formats {
		path := filepath.Join(dir, sanitizeFilename(m.pm.Meta.Title)+f.ext)
		out, err := f.fn(path)
		if err != nil {
			result[f.ext] = fmt.Sprintf("失败: %v", err)
		} else {
			result[f.ext] = out
		}
	}

	return result, nil
}

// ── 辅助 ────────────────────────────────────────────────────

func chapterToHTML(content string) string {
	// 简单转换：段落间空行 → <p>
	paragraphs := strings.Split(content, "\n\n")
	var sb strings.Builder
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			sb.WriteString("<p>")
			sb.WriteString(escapeHTML(p))
			sb.WriteString("</p>\n")
		}
	}
	return sb.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return replacer.Replace(s)
}
