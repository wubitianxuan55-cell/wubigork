package export

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bmaupin/go-epub"
	"github.com/gaea/gaea/internal/project"
)

// Manager 导出管理器
type Manager struct {
	pm *project.Manager

	// mainlineOnly 仅导出主线章节（跳过分支章节文件 NNN[a-z].md）。
	// 零值 false = 含分支，与历史行为一致；由 SetMainlineOnly 显式开启。
	mainlineOnly bool

	// FailedChapters 最近一次导出中读取失败的章节数（部分失败可见性）。
	// 导出入口均为前台顺序调用，无需并发保护；连续导出多种格式时保留最后一种格式的计数。
	FailedChapters int
}

type chapterEntry struct {
	num    int
	branch string
}

// New 创建导出管理器
func New(pm *project.Manager) *Manager {
	return &Manager{pm: pm}
}

// SetMainlineOnly 设置是否仅导出主线章节（跳过 NNN[a-z].md 分支章节文件）。
// 返回自身便于链式调用；不调用时保持默认（含分支，历史行为）。
func (m *Manager) SetMainlineOnly(v bool) *Manager {
	m.mainlineOnly = v
	return m
}

// author 返回导出用作者名：优先取项目元数据 Author 字段；
// 项目未配置作者时回退固定品牌名 "gaea"（与历史导出行为保持一致）。
func (m *Manager) author() string {
	if m.pm != nil && m.pm.Meta != nil && strings.TrimSpace(m.pm.Meta.Author) != "" {
		return strings.TrimSpace(m.pm.Meta.Author)
	}
	return "gaea"
}

// ensureParentDir 确保导出目标文件的父目录存在（WriteFile / EPUB 写盘不会自动建目录）。
func ensureParentDir(outPath string) error {
	return os.MkdirAll(filepath.Dir(outPath), 0755)
}

// listChapters 扫描 chapters/ 下的主线和分支章节，主线在前，分支按 a/b/c 顺序。
// mainlineOnly 模式下跳过分支章节文件（正则与 internal/graph/consistency.go 一致）。
func (m *Manager) listChapters() []chapterEntry {
	dir := filepath.Join(m.pm.Dir, "chapters")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`^(\d{3})([a-z]?)\.md$`)
	var out []chapterEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		match := re.FindStringSubmatch(e.Name())
		if len(match) != 3 {
			continue
		}
		var num int
		if _, err := fmt.Sscanf(match[1], "%d", &num); err != nil {
			continue
		}
		if m.mainlineOnly && match[2] != "" {
			continue
		}
		out = append(out, chapterEntry{num: num, branch: match[2]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].num != out[j].num {
			return out[i].num < out[j].num
		}
		return out[i].branch < out[j].branch
	})
	return out
}

// ExportTXT 导出为纯文本
func (m *Manager) ExportTXT(outPath string) (string, error) {
	if err := ensureParentDir(outPath); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
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

	failed := 0
	for _, ch := range m.listChapters() {
		content, err := m.pm.ReadChapterBranch(ch.num, ch.branch)
		if err != nil {
			failed++
			slog.Warn("exportTXT: 读取章节失败，跳过", "num", ch.num, "branch", ch.branch, "error", err)
			continue
		}
		label := fmt.Sprintf("第 %d 章", ch.num)
		if ch.branch != "" {
			label += " " + ch.branch
		}
		sb.WriteString(label + "\n\n")
		sb.WriteString(content)
		sb.WriteString("\n\n" + strings.Repeat("-", 30) + "\n\n")
	}
	m.FailedChapters = failed

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("写入 TXT 失败: %w", err)
	}
	return outPath, nil
}

// ExportMarkdown 导出为 Markdown 合集
func (m *Manager) ExportMarkdown(outPath string) (string, error) {
	if err := ensureParentDir(outPath); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", m.pm.Meta.Title))
	sb.WriteString(fmt.Sprintf("> 题材: %s | 文风: %s\n\n", m.pm.Meta.Genre, m.pm.Meta.Style))

	// 与 TXT 导出对齐：世界观同样进入 Markdown 合集
	worldview, err := m.pm.ReadWorldview()
	if err != nil {
		slog.Warn("exportMarkdown: 读取世界观失败", "error", err)
	}
	if worldview != "" {
		sb.WriteString("## 世界观\n\n")
		sb.WriteString(worldview)
		sb.WriteString("\n\n---\n\n")
	}

	failed := 0
	for _, ch := range m.listChapters() {
		content, err := m.pm.ReadChapterBranch(ch.num, ch.branch)
		if err != nil {
			failed++
			slog.Warn("exportMarkdown: 读取章节失败，跳过", "num", ch.num, "branch", ch.branch, "error", err)
			continue
		}
		label := fmt.Sprintf("第 %d 章", ch.num)
		if ch.branch != "" {
			label += " " + ch.branch
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n", label))
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}
	m.FailedChapters = failed

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("写入 Markdown 失败: %w", err)
	}
	return outPath, nil
}

// findCoverImage 探测项目书封：play 空间 <pm.Dir>/.gaea/play/exports/cover-*.png，
// 多个命中时取 mtime 最新（与 novel_bookcover.go 的落盘约定对应，仅读取已存在文件，
// 不触发任何生图逻辑）。未找到返回空串，调用方回退默认封面页。
func (m *Manager) findCoverImage() string {
	dir := filepath.Join(m.pm.Dir, ".gaea", "play", "exports")
	matches, err := filepath.Glob(filepath.Join(dir, "cover-*.png"))
	if err != nil {
		return ""
	}
	best, bestMod := "", time.Time{}
	for _, p := range matches {
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			continue
		}
		if best == "" || fi.ModTime().After(bestMod) {
			best, bestMod = p, fi.ModTime()
		}
	}
	return best
}

// ExportEPUB 导出为 EPUB 电子书
func (m *Manager) ExportEPUB(outPath string) (string, error) {
	e := epub.NewEpub(m.pm.Meta.Title)
	e.SetAuthor(m.author())

	// 真封面：项目已生成书封（play 空间 exports/cover-*.png）时用 SetCover 嵌入，
	// go-epub 会生成 image 全屏 cover.xhtml（不进 TOC）并写入 cover 元数据；
	// 未生成书封或嵌入失败则维持历史硬编码封面页，行为逐字节不变。
	realCover := false
	if coverPath := m.findCoverImage(); coverPath != "" {
		imgRef, err := e.AddImage(coverPath, "cover.png")
		if err != nil {
			slog.Warn("exportEPUB: 嵌入书封失败，回退默认封面页", "path", coverPath, "error", err)
		} else {
			e.SetCover(imgRef, "")
			realCover = true
		}
	}
	if !realCover {
		// 封面页
		coverHTML := fmt.Sprintf(`<html><body>
		<h1>%s</h1>
		<p>题材: %s | 文风: %s</p>
		<p>由 gaea AI 辅助创作</p>
	</body></html>`, m.pm.Meta.Title, m.pm.Meta.Genre, m.pm.Meta.Style)
		if _, err := e.AddSection(coverHTML, "封面", "cover.xhtml", ""); err != nil {
			return "", fmt.Errorf("添加封面章节失败: %w", err)
		}
	}

	// 世界观
	worldview, err := m.pm.ReadWorldview()
	if err != nil {
		slog.Warn("exportEPUB: 读取世界观失败", "error", err)
	}
	if worldview != "" {
		wvHTML := fmt.Sprintf("<html><body><h2>世界观</h2><pre>%s</pre></body></html>",
			escapeHTML(worldview))
		if _, err := e.AddSection(wvHTML, "世界观", "worldview.xhtml", ""); err != nil {
			return "", fmt.Errorf("添加世界观章节失败: %w", err)
		}
	}

	// 各章节（与 HTML 导出共用 markdownToHTML 分段器，保证两格式段落行为一致）
	failed := 0
	for _, ch := range m.listChapters() {
		content, err := m.pm.ReadChapterBranch(ch.num, ch.branch)
		if err != nil {
			failed++
			slog.Warn("exportEPUB: 读取章节失败，跳过", "num", ch.num, "branch", ch.branch, "error", err)
			continue
		}
		label := fmt.Sprintf("第 %d 章", ch.num)
		filename := fmt.Sprintf("chapter_%03d", ch.num)
		if ch.branch != "" {
			label += " " + ch.branch
			filename += ch.branch
		}
		chHTML := fmt.Sprintf("<html><body><h2>%s</h2>%s</body></html>",
			label, markdownToHTML(content))
		if _, err := e.AddSection(chHTML, label, filename+".xhtml", ""); err != nil {
			return "", fmt.Errorf("添加章节 %s 失败: %w", label, err)
		}
	}
	m.FailedChapters = failed

	if err := ensureParentDir(outPath); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
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
		{".docx", m.ExportDOCX},
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

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// windowsReservedNames Windows 保留设备名（带扩展名同样非法，如 CON.txt）。
var windowsReservedNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {},
	"COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {},
	"LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	s = replacer.Replace(s)
	// Windows 文件名不允许以点或空格结尾
	s = strings.TrimRight(s, ". ")
	if s == "" {
		return s
	}
	// Windows 保留名按基名（第一个点之前）匹配，命中则加前缀 "_" 规避设备名冲突
	base := s
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	if _, reserved := windowsReservedNames[strings.ToUpper(base)]; reserved {
		s = "_" + s
	}
	return s
}
