package app

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// NovelImportResult 导入成品小说的结果摘要。
type NovelImportResult struct {
	Path         string `json:"path"`
	Title        string `json:"title"`
	ChapterCount int    `json:"chapter_count"`
	TotalWords   int    `json:"total_words"`
}

// importChapter 解析出的章节（标题 + 正文）。
type importChapter struct {
	Title   string
	Content string
}

// ImportNovelBook 导入成品小说（TXT / Markdown / EPUB）到书架：
// 解析章节 → 新建项目（outline + chapters/）→ 返回项目路径（不自动打开）。
func (a *writingState) ImportNovelBook(filePath, title, genre, style string) (NovelImportResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".txt" && ext != ".md" && ext != ".markdown" && ext != ".epub" {
		return NovelImportResult{}, fmt.Errorf("暂支持 TXT / Markdown / EPUB 格式")
	}
	if _, err := os.Stat(filePath); err != nil {
		return NovelImportResult{}, fmt.Errorf("文件不存在：%s", filePath)
	}
	chapters, err := parseNovelFile(filePath)
	if err != nil {
		return NovelImportResult{}, err
	}
	if len(chapters) == 0 {
		return NovelImportResult{}, fmt.Errorf("未解析到任何章节内容")
	}
	if strings.TrimSpace(title) == "" {
		title = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}
	if strings.TrimSpace(genre) == "" {
		genre = "未分类"
	}
	if strings.TrimSpace(style) == "" {
		style = "默认"
	}

	dir, err := uniqueProjectDir(a.cfg.NovelsDir, title)
	if err != nil {
		return NovelImportResult{}, err
	}
	pm, err := project.Create(dir, title, genre, style, "")
	if err != nil {
		return NovelImportResult{}, err
	}

	nodes := make([]types.OutlineNode, 0, len(chapters))
	totalWords := 0
	for i, ch := range chapters {
		num := i + 1
		content := strings.TrimSpace(ch.Content)
		if content == "" {
			content = "（本章暂无内容）"
		}
		if err := pm.WriteChapter(num, content); err != nil {
			return NovelImportResult{}, fmt.Errorf("写章节 %d 失败: %w", num, err)
		}
		totalWords += utf8.RuneCountInString(content)
		nodes = append(nodes, types.OutlineNode{
			ID:          fmt.Sprintf("imp-%03d", num),
			Title:       strings.TrimSpace(ch.Title),
			OrderIndex:  num,
			ChapterFile: fmt.Sprintf("%03d.md", num),
			Status:      types.OutlineDone,
		})
	}
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nodes}); err != nil {
		return NovelImportResult{}, err
	}
	return NovelImportResult{
		Path: dir, Title: title, ChapterCount: len(chapters), TotalWords: totalWords,
	}, nil
}

// uniqueProjectDir 在书架下生成不冲突的项目目录（同名追加序号）。
func uniqueProjectDir(novelsDir, title string) (string, error) {
	base := sanitizeDirName(title)
	dir := filepath.Join(novelsDir, base)
	for i := 2; ; i++ {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			break
		}
		dir = filepath.Join(novelsDir, fmt.Sprintf("%s (%d)", base, i))
	}
	return dir, nil
}

// sanitizeDirName 去除 Windows 非法文件名字符。
func sanitizeDirName(s string) string {
	re := regexp.MustCompile(`[\\/:*?"<>|\r\n\t]+`)
	out := strings.TrimSpace(re.ReplaceAllString(s, "_"))
	if out == "" || out == "." || out == ".." {
		return "导入小说"
	}
	return out
}

// ── 章节解析 ──────────────────────────────────────────────

func parseNovelFile(filePath string) ([]importChapter, error) {
	if strings.EqualFold(filepath.Ext(filePath), ".epub") {
		return parseEpubChapters(filePath)
	}
	return parseTextChapters(filePath)
}

var chapterHeadingRe = regexp.MustCompile(`(?i)^\s*(?:第\s*[0-9０-９一二三四五六七八九十百千万零两〇]+\s*[章回卷节篇部集]|chapter\s+\d+|序章|楔子|引子|前言|序言|尾声|后记|番外|外传|终章|大结局)`)
var markdownHeadingRe = regexp.MustCompile(`^\s*#{1,6}\s+\S`)
var markdownStripRe = regexp.MustCompile(`^#{1,6}\s*`)

func isChapterHeading(line string) bool {
	return chapterHeadingRe.MatchString(line) || markdownHeadingRe.MatchString(line)
}

func parseTextChapters(filePath string) ([]importChapter, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	text := decodeText(raw)
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	var chapters []importChapter
	var cur *importChapter
	var pending []string // 首个章节标题出现前的内容（并入第一章）
	flush := func() {
		if cur == nil {
			return
		}
		body := strings.TrimSpace(cur.Content)
		if body != "" {
			chapters = append(chapters, importChapter{Title: cur.Title, Content: body})
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isChapterHeading(trimmed) {
			flush()
			title := strings.TrimSpace(markdownStripRe.ReplaceAllString(trimmed, ""))
			if title == "" {
				title = trimmed
			}
			cur = &importChapter{Title: title}
			if len(pending) > 0 {
				cur.Content = strings.Join(pending, "\n") + "\n\n"
				pending = nil
			}
			continue
		}
		if cur == nil {
			if strings.TrimSpace(trimmed) != "" {
				pending = append(pending, trimmed)
			}
			continue
		}
		cur.Content += line + "\n"
	}
	flush()
	if len(chapters) == 0 && strings.TrimSpace(text) != "" {
		chapters = []importChapter{{Title: "全文", Content: strings.TrimSpace(text)}}
	}
	return chapters, nil
}

// decodeText 解码文本：UTF-8 优先，GB18030/GBK 兜底（常见中文 TXT）。
func decodeText(raw []byte) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	for _, dec := range []encoding.Encoding{
		simplifiedchinese.GB18030, simplifiedchinese.GBK,
	} {
		if out, err := dec.NewDecoder().Bytes(raw); err == nil && utf8.Valid(out) {
			return string(out)
		}
	}
	return string(raw)
}

// ── EPUB 解析（archive/zip 直接读取 spine 顺序章节） ──────────────

type opfFile struct {
	Metadata struct {
		Titles []string `xml:"title"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Itemrefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

func parseEpubChapters(filePath string) ([]importChapter, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 EPUB 失败: %w", err)
	}
	defer zr.Close()

	opfName, err := findEpubOpf(zr)
	if err != nil {
		return nil, err
	}
	opfData, err := readZipEntry(zr, opfName)
	if err != nil {
		return nil, fmt.Errorf("读取 EPUB 元数据失败: %w", err)
	}
	var opf opfFile
	if err := xml.Unmarshal(opfData, &opf); err != nil {
		return nil, fmt.Errorf("解析 EPUB 元数据失败: %w", err)
	}
	hrefByID := make(map[string]string, len(opf.Manifest.Items))
	for _, it := range opf.Manifest.Items {
		hrefByID[it.ID] = it.Href
	}
	baseDir := filepath.ToSlash(filepath.Dir(opfName))

	var chapters []importChapter
	for i, ir := range opf.Spine.Itemrefs {
		href := hrefByID[ir.IDRef]
		if href == "" {
			continue
		}
		entryName := href
		if !strings.HasPrefix(entryName, "/") && baseDir != "." {
			entryName = baseDir + "/" + entryName
		}
		raw, err := readZipEntry(zr, entryName)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(stripHTML(string(raw)))
		if content == "" {
			continue
		}
		title := extractEpubTitle(string(raw))
		if title == "" {
			title = fmt.Sprintf("第%d章", i+1)
		}
		chapters = append(chapters, importChapter{Title: title, Content: content})
	}
	if len(chapters) == 0 {
		return nil, fmt.Errorf("EPUB 内未解析到章节内容")
	}
	return chapters, nil
}

// findEpubOpf 通过 META-INF/container.xml 定位 content.opf；缺失时扫描 *.opf。
func findEpubOpf(zr *zip.ReadCloser) (string, error) {
	if data, err := readZipEntry(zr, "META-INF/container.xml"); err == nil {
		var c struct {
			Rootfiles struct {
				Rootfile []struct {
					FullPath string `xml:"full-path,attr"`
				} `xml:"rootfile"`
			} `xml:"rootfiles"`
		}
		if xml.Unmarshal(data, &c) == nil && len(c.Rootfiles.Rootfile) > 0 && c.Rootfiles.Rootfile[0].FullPath != "" {
			return c.Rootfiles.Rootfile[0].FullPath, nil
		}
	}
	for _, f := range zr.File {
		if strings.EqualFold(filepath.Ext(f.Name), ".opf") {
			return f.Name, nil
		}
	}
	return "", fmt.Errorf("EPUB 缺少 content.opf")
}

func readZipEntry(zr *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range zr.File {
		if filepath.ToSlash(f.Name) != filepath.ToSlash(name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("zip 内找不到 %s", name)
}

var (
	htmlBlockRe   = regexp.MustCompile(`(?i)<(?:/p|/div|/h[1-6]|br\s*/?|/li|/tr)>`)
	htmlTagRe     = regexp.MustCompile(`<[^>]+>`)
	htmlHeadRe    = regexp.MustCompile(`(?is)<head>.*?</head>`)
	titleRe       = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	epubHeadingRe = regexp.MustCompile(`(?is)<h[12][^>]*>(.*?)</h[12]>`)
)

func stripHTML(s string) string {
	s = htmlHeadRe.ReplaceAllString(s, "")
	s = htmlBlockRe.ReplaceAllString(s, "\n")
	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	// 折叠连续空行（保留段落分隔）
	var b strings.Builder
	blank := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) == "" {
			blank++
			if blank > 2 {
				continue
			}
		} else {
			blank = 0
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String())
}

func extractEpubTitle(xhtml string) string {
	if m := epubHeadingRe.FindStringSubmatch(xhtml); len(m) == 2 {
		if t := strings.TrimSpace(stripHTML(m[1])); t != "" {
			return t
		}
	}
	if m := titleRe.FindStringSubmatch(xhtml); len(m) == 2 {
		return strings.TrimSpace(stripHTML(m[1]))
	}
	return ""
}
