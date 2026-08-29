// Package wssearch 提供工作区轻量全文搜索：扫描工作区文本/办公文件，
// 提取正文后用 TF-IDF（中文 bigram 分词）打分，并返回命中片段。
// 对标 Cursor / Cherry Studio 的本地索引：先做关键词/分词，向量检索按需再上。
// 文件正文按 (绝对路径, mtime, size) 缓存，重复搜索不重复解析。
package wssearch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/search"
	"github.com/gaea/gaea/internal/office/docmd"
)

// Hit 是工作区全文搜索的一条命中。
type Hit struct {
	Path    string  // 工作区相对路径（/ 分隔）
	Name    string  // 文件名
	Size    int64   // 字节数
	ModTime int64   // unix 毫秒
	Score   float64 // 相关度
	Snippet string  // 命中片段（含省略号，无高亮标记）
	// Truncated 标记正文被 20 万字符上限截断（索引不全，用户可见）。
	Truncated bool
	// Skipped 非空表示文件未索引（如原始文本 >5MB），给出可见原因与替代入口。
	Skipped string
}

// 索引收录的文本类扩展名（直接读取正文）。
var textExts = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".csv": true,
	".json": true, ".yml": true, ".yaml": true, ".toml": true,
	".ini": true, ".log": true,
}

// 索引收录的办公类扩展名（docmd 转 Markdown 提取正文）。
var officeExts = map[string]bool{
	".docx": true, ".doc": true, ".xlsx": true, ".xls": true, ".pdf": true,
}

// skipDirs 是工作区搜索跳过的噪音目录（与 app 层资料概览保持一致）。
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "dist": true, "build": true,
	".cache": true, ".codegraph": true, ".tianxuan": true, ".reasonix": true,
}

// isNoiseRel 判断路径是否属于搜索噪音：.gaea 下只跳过会话/归档/缓存与 play
// 产物分区（.gaea/play/exports——S1.2 双空间：play 交付物属乐园产物，不进
// 共享关键词检索面，工位搜索不可见；对应验收红线「工位搜索搜不到乐园记忆/
// 产物」），exports（work 交付产物）等仍可被索引。
func isNoiseRel(rel string) bool {
	first, rest, _ := strings.Cut(rel, "/")
	if first != ".gaea" {
		return false
	}
	if rest == "play/exports" || strings.HasPrefix(rest, "play/exports/") {
		return true
	}
	seg, _, _ := strings.Cut(rest, "/")
	return seg == "sessions" || seg == "archive" || seg == "cache"
}

const (
	maxDepth     = 5
	maxFileSize  = 5 << 20 // 原始文本文件超过 5MB 不索引
	maxDocRunes  = 200_000 // 单个文档最多索引 20 万字符
	defaultLimit = 20
	maxLimit     = 50
)

// ─── 正文提取缓存 ────────────────────────────────────────────────

type cachedText struct {
	modTime   int64
	size      int64
	text      string
	truncated bool
}

var (
	textCache = map[string]cachedText{}
	cacheMu   sync.Mutex
)

// extractText 读取/转换一个文件的正文；失败返回 ("", false)。
func extractText(abs string) (string, bool, bool) {
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return "", false, false
	}
	key := filepath.ToSlash(abs)
	cacheMu.Lock()
	if c, ok := textCache[key]; ok && c.modTime == info.ModTime().UnixMilli() && c.size == info.Size() {
		text := c.text
		truncated := c.truncated
		cacheMu.Unlock()
		return text, true, truncated
	}
	cacheMu.Unlock()

	ext := strings.ToLower(filepath.Ext(abs))
	var text string
	var truncated bool
	switch {
	case textExts[ext]:
		if info.Size() > maxFileSize {
			return "", false, false
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			return "", false, false
		}
		text = toValidUTF8(string(b))
	case officeExts[ext]:
		md, _, _, err := docmd.ConvertLimit(abs, "", docmd.DefaultMaxPDFPages)
		if err != nil {
			return "", false, false
		}
		text = toValidUTF8(md)
	default:
		return "", false, false
	}
	truncated = runeCount(text) > maxDocRunes
	if truncated {
		text = truncateRunes(text, maxDocRunes)
	}
	cacheMu.Lock()
	textCache[key] = cachedText{modTime: info.ModTime().UnixMilli(), size: info.Size(), text: text, truncated: truncated}
	cacheMu.Unlock()
	return text, true, truncated
}

func runeCount(s string) int { return len([]rune(s)) }

// toValidUTF8 修复非 UTF-8 字节（BOM 去除 + 非法字节替换），保证后续 rune 操作安全。
func toValidUTF8(s string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "\ufffd")
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// ─── 扫描 ────────────────────────────────────────────────────────

type doc struct {
	path      string
	name      string
	size      int64
	mod       int64
	text      string
	truncated bool
	skipped   string
}

// scan 遍历工作区收集可索引文档（跳过噪音目录，限制深度）。
func scan(root string) []doc {
	var docs []doc
	var walk func(dir, rel string, depth int)
	walk = func(dir, rel string, depth int) {
		if depth > maxDepth {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				dirRel := name
				if rel != "" {
					dirRel = rel + "/" + name
				}
				if skipDirs[name] || strings.HasPrefix(name, ".tmp") || isNoiseRel(dirRel) {
					continue
				}
				walk(filepath.Join(dir, name), dirRel, depth+1)
				continue
			}
			ext := strings.ToLower(filepath.Ext(name))
			if !textExts[ext] && !officeExts[ext] {
				continue
			}
			relPath := name
			if rel != "" {
				relPath = rel + "/" + name
			}
			if isNoiseRel(relPath) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			docs = append(docs, doc{path: relPath, name: name, size: info.Size(), mod: info.ModTime().UnixMilli()})
		}
	}
	walk(root, "", 0)
	return docs
}

// ─── 搜索 ────────────────────────────────────────────────────────

// Search 在工作区全文检索 query，按相关度倒序返回最多 limit 条命中。
// 空查询或无可索引文档返回空切片。
func Search(cwd, query string, limit int) []Hit {
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}

	docs := scan(cwd)
	if len(docs) == 0 {
		return nil
	}

	// 提取正文（缓存命中则跳过解析）；全部失败时返回空。
	searchDocs := make([]search.Doc, 0, len(docs))
	for i := range docs {
		abs := filepath.Join(cwd, filepath.FromSlash(docs[i].path))
		ext := strings.ToLower(filepath.Ext(docs[i].path))
		if textExts[ext] && docs[i].size > maxFileSize {
			// 超大文本不索引，但保留为「文件名命中 + 跳过原因」提示。
			docs[i].skipped = fmt.Sprintf("文件过大（%s）未索引：可让 AI 用 summarize_file 获取全文摘要，或拆分文件",
				humanSize(docs[i].size))
			continue
		}
		text, ok, truncated := extractText(abs)
		if !ok || strings.TrimSpace(text) == "" {
			continue
		}
		docs[i].text = text
		docs[i].truncated = truncated
		searchDocs = append(searchDocs, search.Doc{ID: docs[i].path, Text: text})
	}
	if len(searchDocs) == 0 {
		return nil
	}

	idx := search.NewTfidfIndex()
	idx.Build(searchDocs)
	scored := idx.Search(q, limit, 0.03)

	// 标题/文件名命中补底：正文无分数但文件名含关键词时也返回（0.05 保底分）。
	filenameBoost := map[string]float64{}
	if len(scored) < limit {
		for _, d := range docs {
			if strings.Contains(strings.ToLower(d.name), strings.ToLower(q)) {
				filenameBoost[d.path] = 0.08
			}
		}
	}

	out := make([]Hit, 0, len(scored))
	seen := map[string]bool{}
	for _, s := range scored {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		var d *doc
		for i := range docs {
			if docs[i].path == s.ID {
				d = &docs[i]
				break
			}
		}
		if d == nil {
			continue
		}
		score := s.Score + filenameBoost[s.ID]
		out = append(out, Hit{
			Path:      d.path,
			Name:      d.name,
			Size:      d.size,
			ModTime:   d.mod,
			Score:     score,
			Snippet:   makeSnippet(d.text, q),
			Truncated: d.truncated,
			Skipped:   d.skipped,
		})
	}
	// 文件名补底命中
	for path, boost := range filenameBoost {
		if seen[path] {
			continue
		}
		seen[path] = true
		for i := range docs {
			if docs[i].path != path {
				continue
			}
			out = append(out, Hit{
				Path:      docs[i].path,
				Name:      docs[i].name,
				Size:      docs[i].size,
				ModTime:   docs[i].mod,
				Score:     boost,
				Snippet:   makeSnippet(docs[i].text, q),
				Truncated: docs[i].truncated,
				Skipped:   docs[i].skipped,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// humanSize 把字节数格式化为可读大小。
func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// makeSnippet 取第一个命中的上下文片段（前后各约 60 字符），并压缩换行。
func makeSnippet(text, query string) string {
	terms := search.Tokenize(query)
	// 优先用最长的整词定位（中文 bigram 最短），避免先用噪声 bigram 命中。
	sort.Slice(terms, func(i, j int) bool {
		return utf8.RuneCountInString(terms[i]) > utf8.RuneCountInString(terms[j])
	})
	runes := []rune(text)
	lower := []rune(strings.ToLower(text))
	pos := -1
	for _, t := range terms {
		tr := []rune(strings.ToLower(t))
		if len(tr) == 0 {
			continue
		}
		for i := 0; i+len(tr) <= len(lower); i++ {
			match := true
			for j := range tr {
				if lower[i+j] != tr[j] {
					match = false
					break
				}
			}
			if match {
				pos = i
				break
			}
		}
		if pos >= 0 {
			break
		}
	}
	if pos < 0 {
		return collapseWhitespace(truncateRunes(text, 120))
	}

	const half = 60
	start := pos - half
	if start < 0 {
		start = 0
	}
	end := pos + half
	if end > len(runes) {
		end = len(runes)
	}
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(runes) {
		snippet += "…"
	}
	return collapseWhitespace(snippet)
}

// collapseWhitespace 把连续的空白/换行压成单个空格，避免片段跨行撑爆 UI。
func collapseWhitespace(s string) string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t'
	})
	return strings.Join(fields, " ")
}
