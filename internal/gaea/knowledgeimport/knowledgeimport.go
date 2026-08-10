// Package knowledgeimport 知识库文件导入：支持 md/txt（直接入库）、
// docx/pdf（转 Markdown 提取正文）、xlsx/csv（表头自动映射：标题/分类/
// 阶段/标签/正文/来源）。产出候选条目预览（与既有条目按标题匹配标
// 「新增/将覆盖」），供前端确认后批量入库。遵循「无确认不落库」。
package knowledgeimport

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/textsim"
	"github.com/gaea/gaea/internal/office/docmd"
)

const (
	// MaxBodyChars 单条正文上限（超长截断，避免 IPC/上下文失控）。
	MaxBodyChars = 80000
	// MaxRows 导入预览行数上限。
	MaxRows = 300
)

// Row 是导入预览中的一条候选知识条目。
type Row struct {
	Name         string
	Title        string
	Category     string
	Phase        string
	Discipline   string
	Tags         []string
	Status       string
	Source       string
	Body         string
	ExistingName string // 匹配到的既有条目（空=新增）
	MatchNote    string // 新增 / 将覆盖更新
	SimilarName  string // 相似（非同名）条目建议合并
	SimilarNote  string // 如 "与「xxx」相似 87%，建议合并"
	Raw          string
	Skip         bool
	SkipReason   string
}

// Preview 是导入解析结果视图。
type Preview struct {
	Path     string
	FileName string
	Columns  []string
	Unmapped []string
	Rows     []Row
	Message  string
}

// ExtractText 提取文件文本（md/txt 原文；docx/pdf/xlsx/csv 转 Markdown）。
func ExtractText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt", ".text":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case ".docx", ".xlsx", ".xlsm", ".pdf", ".et", ".ods":
		md, _, _, err := docmd.ConvertLimit(path, "", 50)
		if err != nil {
			return "", err
		}
		return md, nil
	case ".csv", ".tsv":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("暂不支持 %s 格式导入", ext)
	}
}

// Parse 解析文件为候选知识条目（store 可为 nil 跳过既有匹配）。
func Parse(path string, store *knowledge.Store) (*Preview, error) {
	abs := path
	if !filepath.IsAbs(path) {
		if wd, err := os.Getwd(); err == nil {
			abs = filepath.Join(wd, path)
		}
	}
	ext := strings.ToLower(filepath.Ext(abs))
	pv := &Preview{
		Path:     filepath.ToSlash(abs),
		FileName: filepath.Base(abs),
	}
	switch ext {
	case ".md", ".markdown", ".txt", ".text":
		text, err := ExtractText(abs)
		if err != nil {
			return nil, err
		}
		stem := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
		pv.Rows = MatchRows([]Row{{
			Title:    stem,
			Category: guessCategory(text),
			Status:   "现行",
			Source:   filepath.Base(abs),
			Body:     capText(text),
		}}, store)
		pv.Message = "md/txt 直接入库，可在预览中修改分类/标题。"
	case ".docx", ".pdf":
		text, err := ExtractText(abs)
		if err != nil {
			return nil, err
		}
		stem := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
		pv.Rows = MatchRows([]Row{{
			Title:    stem,
			Category: guessCategory(text),
			Status:   "现行",
			Source:   filepath.Base(abs),
			Body:     capText(text),
		}}, store)
		pv.Message = "已提取文档正文（可能含多主题，可用 AI 智能解析拆分）。"
	case ".xlsx", ".xlsm", ".csv", ".tsv", ".et", ".ods":
		cols, rows, err := tableRows(abs, ext == ".tsv")
		if err != nil {
			return nil, err
		}
		pv.Columns = cols
		if len(rows) == 0 {
			pv.Message = "表格没有数据行。"
			return pv, nil
		}
		colMap := mapColumns(cols)
		for c, h := range cols {
			if colMap[c] == fieldNone && !isNoiseHeader(h) {
				pv.Unmapped = append(pv.Unmapped, h)
			}
		}
		if len(rows) > MaxRows {
			rows = rows[:MaxRows]
			pv.Message = "仅展示前 300 行，其余请分批导入。"
		}
		pv.Rows = MatchRows(buildRows(rows, colMap, filepath.Base(abs)), store)
		if len(pv.Columns) == 0 {
			pv.Message = strings.TrimSpace(pv.Message + " ") + "未识别到表头（缺少标题/正文等列）。"
		}
	default:
		return nil, fmt.Errorf("暂不支持 %s 格式导入", ext)
	}
	return pv, nil
}

// MatchRows 补全 Name/Status/MatchNote/Existing*，并按标题做既有匹配。
func MatchRows(rows []Row, store *knowledge.Store) []Row {
	byTitle := map[string]knowledge.EntrySummary{}
	var allTitles []knowledge.EntrySummary
	if store != nil {
		for _, s := range store.List() {
			if t := strings.ToLower(strings.TrimSpace(s.Title)); t != "" {
				byTitle[t] = s
			}
			allTitles = append(allTitles, s)
		}
	}
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		if r.Title == "" {
			r.Skip = true
			r.SkipReason = "缺少标题"
		} else if strings.TrimSpace(r.Body) == "" {
			r.Skip = true
			r.SkipReason = "缺少正文"
		} else {
			r.Name = slugName(r.Title)
			r.Status = "现行"
			if e, ok := byTitle[strings.ToLower(strings.TrimSpace(r.Title))]; ok {
				r.ExistingName = e.Name
				r.MatchNote = "将覆盖更新"
			} else {
				r.MatchNote = "新增"
			}
			// 相似（非同名）条目提示：标题模糊匹配 ≥0.65 建议合并。
			if r.ExistingName == "" {
				best := ""
				bestScore := 0.0
				for _, e := range allTitles {
					if strings.EqualFold(strings.TrimSpace(e.Title), strings.TrimSpace(r.Title)) {
						continue
					}
					if s := Similarity(r.Title, e.Title); s > bestScore {
						bestScore = s
						best = e.Title
					}
				}
			if bestScore >= 0.55 {
					r.SimilarName = best
					r.SimilarNote = fmt.Sprintf("与「%s」相似 %d%%，建议合并", best, int(bestScore*100))
				}
			}
		}
		out = append(out, r)
	}
	return out
}

// SimilarHit 是查重命中的相似条目。
type SimilarHit struct {
	Name  string
	Title string
	Score float64
}

// FindSimilar 返回与 title 模糊相似（≥min）的既有条目，按相似度降序。
func FindSimilar(store *knowledge.Store, title string, min float64) []SimilarHit {
	if store == nil || strings.TrimSpace(title) == "" {
		return nil
	}
	var out []SimilarHit
	for _, e := range store.List() {
		if strings.EqualFold(strings.TrimSpace(e.Title), strings.TrimSpace(title)) {
			continue
		}
		if s := Similarity(title, e.Title); s >= min {
			out = append(out, SimilarHit{Name: e.Name, Title: e.Title, Score: s})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// Similarity 标题相似度（0~1）：委托 textsim（CJK 二元组集合 Dice）。
func Similarity(a, b string) float64 {
	return textsim.Similarity(a, b)
}

// ── 表格解析（xlsx/csv）────────────────────────────────────────

type columnField int

const (
	fieldNone columnField = iota
	fieldTitle
	fieldCategory
	fieldPhase
	fieldDiscipline
	fieldTags
	fieldSource
	fieldBody
)

var fieldKeywords = map[columnField][]string{
	fieldTitle:      {"标题", "名称", "题目", "主题", "条目"},
	fieldCategory:   {"分类", "类别"},
	fieldPhase:      {"阶段"},
	fieldDiscipline: {"专业", "领域"},
	fieldTags:       {"标签", "关键词"},
	fieldSource:     {"来源", "出处"},
	fieldBody:       {"正文", "内容", "摘要", "说明", "条文"},
}

func tableRows(path string, tsv bool) ([]string, [][]string, error) {
	var raw [][]string
	if strings.EqualFold(filepath.Ext(path), ".csv") || strings.EqualFold(filepath.Ext(path), ".tsv") {
		rows, err := readCSV(path, tsv)
		if err != nil {
			return nil, nil, err
		}
		raw = rows
	} else {
		rows, err := readXLSX(path)
		if err != nil {
			return nil, nil, err
		}
		raw = rows
	}
	// 去空行、统一列宽、首行=表头。
	var out [][]string
	maxCols := 0
	for _, r := range raw {
		line := make([]string, len(r))
		empty := true
		for i, c := range r {
			line[i] = strings.TrimSpace(c)
			if line[i] != "" {
				empty = false
			}
		}
		if empty {
			continue
		}
		if len(line) > maxCols {
			maxCols = len(line)
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil, nil, nil
	}
	for i := range out {
		for len(out[i]) < maxCols {
			out[i] = append(out[i], "")
		}
	}
	return out[0], out[1:], nil
}

func mapColumns(header []string) map[int]columnField {
	out := map[int]columnField{}
	for c, h := range header {
		low := strings.ToLower(strings.TrimSpace(h))
		best := fieldNone
		bestLen := 0
		for f, kws := range fieldKeywords {
			for _, kw := range kws {
				if strings.Contains(low, strings.ToLower(kw)) && len(kw) > bestLen {
					best = f
					bestLen = len(kw)
				}
			}
		}
		if best != fieldNone {
			out[c] = best
		}
	}
	return out
}

func isNoiseHeader(h string) bool {
	low := strings.ToLower(strings.TrimSpace(h))
	for _, kw := range []string{"序号", "编号", "日期", "时间", "备注"} {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}

func buildRows(rows [][]string, colMap map[int]columnField, source string) []Row {
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		row := Row{Status: "现行", Source: source}
		for c, f := range colMap {
			if c >= len(r) {
				continue
			}
			v := strings.TrimSpace(r[c])
			switch f {
			case fieldTitle:
				if row.Title == "" {
					row.Title = v
				}
			case fieldCategory:
				if row.Category == "" {
					row.Category = v
				}
			case fieldPhase:
				if row.Phase == "" {
					row.Phase = v
				}
			case fieldDiscipline:
				if row.Discipline == "" {
					row.Discipline = v
				}
			case fieldTags:
				if len(row.Tags) == 0 {
					for _, t := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == '，' || r == '、' || r == ';' || r == '；' }) {
						if t = strings.TrimSpace(t); t != "" {
							row.Tags = append(row.Tags, t)
						}
					}
				}
			case fieldSource:
				if row.Source == "" || row.Source == source {
					row.Source = v
				}
			case fieldBody:
				if row.Body == "" {
					row.Body = capText(v)
				}
			}
		}
		row.Raw = strings.Join(r, " | ")
		out = append(out, row)
	}
	return out
}

func readCSV(path string, tsv bool) ([][]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sep := ','
	if tsv {
		sep = '\t'
	}
	var rows [][]string
	var cur []string
	var sb strings.Builder
	text := string(b)
	inQuote := false
	for _, ch := range text {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == rune(sep) && !inQuote:
			cur = append(cur, strings.TrimSpace(sb.String()))
			sb.Reset()
		case (ch == '\n' || ch == '\r') && !inQuote:
			if sb.Len() > 0 || len(cur) > 0 {
				cur = append(cur, strings.TrimSpace(sb.String()))
				sb.Reset()
				rows = append(rows, cur)
				cur = nil
			}
		default:
			sb.WriteRune(ch)
		}
	}
	if sb.Len() > 0 || len(cur) > 0 {
		cur = append(cur, strings.TrimSpace(sb.String()))
		rows = append(rows, cur)
	}
	return rows, nil
}

func readXLSX(path string) ([][]string, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx 无工作表")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}
	return rows, nil
}

func capText(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= MaxBodyChars {
		return string(r)
	}
	return string(r[:MaxBodyChars]) + "\n…（已截断）"
}

// guessCategory 按内容关键词猜测分类（默认其他）。
func guessCategory(text string) string {
	for _, kw := range []string{"GB ", "GB/", "HJ ", "HJ/", "规范", "标准", "规程"} {
		if strings.Contains(text, kw) {
			return knowledge.CatStandard
		}
	}
	if strings.Contains(text, "案例") || strings.Contains(text, "项目") || strings.Contains(text, "工程实例") {
		return knowledge.CatCase
	}
	if strings.Contains(text, "经验") || strings.Contains(text, "总结") || strings.Contains(text, "教训") {
		return knowledge.CatExperience
	}
	if strings.Contains(text, "材料") || strings.Contains(text, "工艺") {
		return knowledge.CatMaterial
	}
	if strings.Contains(text, "法规") || strings.Contains(text, "政策") || strings.Contains(text, "条例") {
		return knowledge.CatRegulation
	}
	if strings.Contains(text, "调查") || strings.Contains(text, "报告") {
		return knowledge.CatSurvey
	}
	if strings.Contains(text, "方案") || strings.Contains(text, "设计") {
		return knowledge.CatDesign
	}
	return knowledge.CatOther
}

// slugName 由标题确定性生成唯一键（与 cost.SlugName 同算法，保证同名覆盖）。
func slugName(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "entry"
	}
	if runes := []rune(name); len(runes) > 60 {
		name = string(runes[:60])
	}
	return name
}
