// Package costimport 成本库文件导入解析：读取 xlsx/csv 报价单/测算表，
// 自动映射列（名称/规格/单位/单价/来源/分类）、归一化价格、并对既有
// 成本条目做同名匹配（标记「新增」或「将覆盖更新」），供前端预览确认后
// 批量入库。遵循「无确认不落库」：本包只产出候选，写库由调用方执行。
package costimport

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/cost"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

// MaxRows 导入预览上限：超大表格截断并提示，避免 IPC/上下文失控。
const MaxRows = 500

// Row 是导入预览中的一条候选成本条目（含与既有库的匹配信息）。
type Row struct {
	Name         string
	Title        string
	Category     string
	Unit         string
	Price        float64
	Spec         string
	Source       string
	Status       string
	ExistingName string  // 匹配到的既有条目名称（空=新增）
	ExistingPrice float64 // 既有条目现价（覆盖提示用）
	MatchNote    string  // "新增" / "将覆盖更新（现价 ¥3200）"
	Raw          string  // 原始行摘要，供前端核对
	Skip         bool
	SkipReason   string
}

// Preview 是导入解析结果视图。
type Preview struct {
	Path     string
	FileName string
	Columns  []string // 识别出的表头
	Unmapped []string // 未映射列（如序号/数量/备注）
	Rows     []Row
	Message  string
}

// columnField 表示表头列映射到的成本字段。
type columnField int

const (
	fieldNone columnField = iota
	fieldTitle
	fieldSpec
	fieldUnit
	fieldPrice
	fieldSource
	fieldCategory
)

var fieldKeywords = map[columnField][]string{
	fieldTitle:    {"材料名称", "项目名称", "名称规格", "品名", "材料", "物资", "项目", "科目", "设备", "机械", "名称", "内容", "子目"},
	fieldSpec:     {"规格型号", "规格", "型号", "材质", "参数"},
	fieldUnit:     {"单位"},
	fieldPrice:    {"含税单价", "单价", "市场价", "信息价", "报价", "价格", "金额"},
	fieldSource:   {"供应商", "来源", "品牌", "产地"},
	fieldCategory: {"分类", "类别"},
}

// Parse 解析 xlsx/csv 文件并产出候选成本条目。
// store 可为 nil（跳过既有匹配）；path 支持绝对路径或工作区相对路径。
func Parse(path string, store *cost.Store) (*Preview, error) {
	abs := path
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		abs = filepath.Join(wd, path)
	}
	ext := strings.ToLower(filepath.Ext(abs))

	var columns []string
	var rows [][]string
	var err error
	switch ext {
	case ".xlsx", ".xlsm", ".et", ".ods":
		columns, rows, err = readSheet(abs)
	case ".csv", ".tsv":
		columns, rows, err = readCSV(abs, ext == ".tsv")
	default:
		return nil, fmt.Errorf("暂不支持 %s 格式导入，请使用 xlsx/csv（PDF 可用 AI 解析）", ext)
	}
	if err != nil {
		return nil, err
	}

	pv := &Preview{
		Path:     filepath.ToSlash(abs),
		FileName: filepath.Base(abs),
		Columns:  append([]string(nil), columns...),
	}
	if len(rows) == 0 {
		pv.Message = "文件没有数据行。"
		return pv, nil
	}

	// 表头识别：首行包含单价/名称等关键词且多列非空视为表头；否则在后续
	// 前几行里找真正的表头（报价单/测算表常在表头前带标题行、说明行）。
	colMap := map[int]columnField{}
	dataRows := rows
	if likelyHeader(columns) {
		colMap = mapColumns(columns)
	} else if header := findHeaderRow(rows); header >= 0 {
		columns = rows[header]
		dataRows = rows[header+1:]
		pv.Columns = append([]string(nil), columns...)
		pv.Message = fmt.Sprintf("已跳过前 %d 行标题/说明，识别到表头。", header+1)
		colMap = mapColumns(columns)
	} else if vertical := buildVerticalRows(rows); len(vertical) > 0 {
		pv.Message = fmt.Sprintf("未识别到横向表头，已按纵向参数表提取 %d 条单价类条目（其余参数行已跳过）。", len(vertical))
		pv.Rows = MatchRows(vertical, store)
		return pv, nil
	} else {
		pv.Message = "未识别到表头（缺少名称/单价等列），可尝试 AI 智能解析。"
	}
	// 未映射列（供前端提示）：跳过序号/数量/合计/备注等常见非成本列。
	for c, h := range columns {
		if colMap[c] == fieldNone && !isNoiseHeader(h) {
			pv.Unmapped = append(pv.Unmapped, h)
		}
	}

	if len(dataRows) > MaxRows {
		dataRows = dataRows[:MaxRows]
		pv.Message = strings.TrimSpace(pv.Message + " ") + fmt.Sprintf("仅展示前 %d 行，其余请分批导入。", MaxRows)
	}
	pv.Rows = MatchRows(buildRows(dataRows, colMap), store)
	return pv, nil
}

// findHeaderRow 在前几行数据中查找真正的表头行（报价单常在表头前带
// 项目标题/说明行），找不到返回 -1。
func findHeaderRow(rows [][]string) int {
	limit := 5
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		if likelyHeader(rows[i]) {
			return i
		}
	}
	return -1
}

// buildVerticalRows 纵向参数表兜底：无横向表头时，若行结构为
// 「参数名 | 数值 | 说明 | 单位」的竖排参数表（成本测算表/报价参数表常见），
// 提取名称/单位含“元”的单价类条目，其余几何/费率等参数行跳过。
// 识别失败返回空（调用方回退到 AI 智能解析提示）。
func buildVerticalRows(rows [][]string) []Row {
	matches := 0
	for i, r := range rows {
		if i >= 8 {
			break
		}
		if isVerticalParamRow(r) {
			matches++
		}
	}
	if matches < 4 {
		return nil
	}

	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		if len(r) < 2 || strings.TrimSpace(r[0]) == "" {
			continue
		}
		price, ok := parsePrice(r[1])
		if !ok {
			continue
		}
		title := strings.TrimSpace(r[0])
		unit := ""
		if len(r) >= 4 {
			unit = strings.TrimSpace(r[3])
		} else if len(r) == 3 {
			unit = strings.TrimSpace(r[2])
		}
		if !strings.Contains(title, "元") && !strings.Contains(unit, "元") {
			continue
		}
		row := Row{Title: title, Price: price, Unit: unit, Category: normalizeCategory(title)}
		if len(r) >= 3 {
			row.Source = strings.TrimSpace(r[2])
		}
		row.Raw = strings.Join(r, " | ")
		out = append(out, row)
	}
	return out
}

// isVerticalParamRow 判断一行是否符合「名称 | 数值 | 说明 | 单位」竖排结构，
// 避免把横向数据表（首列为序号）误判成参数表。
func isVerticalParamRow(r []string) bool {
	if len(r) < 4 {
		return false
	}
	name := strings.TrimSpace(r[0])
	if name == "" || isNumeric(name) {
		return false
	}
	if _, ok := parsePrice(r[1]); !ok {
		return false
	}
	if strings.TrimSpace(r[2]) == "" {
		return false
	}
	unit := strings.TrimSpace(r[3])
	return unit != "" && !isNumeric(unit)
}

func isNumeric(s string) bool {
	clean := strings.NewReplacer(",", "", "，", "", " ", "").Replace(strings.TrimSpace(s))
	if clean == "" {
		return false
	}
	_, err := strconv.ParseFloat(clean, 64)
	return err == nil
}

// MatchRows 对候选行做既有条目匹配（按标题/名称精确匹配），补全
// Name/Status/MatchNote/Existing* 字段；缺少名称或有效单价的行标记 Skip。
func MatchRows(rows []Row, store *cost.Store) []Row {
	byTitle := map[string]cost.Summary{}
	byName := map[string]cost.Summary{}
	if store != nil && store.Available() {
		for _, s := range store.List() {
			if t := strings.ToLower(strings.TrimSpace(s.Title)); t != "" {
				byTitle[t] = s
			}
			if n := strings.TrimSpace(s.Name); n != "" {
				byName[n] = s
			}
		}
	}

	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		if row.Title == "" && row.Price <= 0 {
			row.Skip = true
			row.SkipReason = "缺少名称与单价"
		} else if row.Title == "" {
			row.Skip = true
			row.SkipReason = "缺少名称"
		} else if row.Price <= 0 {
			row.Skip = true
			row.SkipReason = "缺少有效单价"
		} else {
			row.Name = cost.SlugName(row.Title)
			row.Status = "现行"
			if existing, ok := byTitle[strings.ToLower(strings.TrimSpace(row.Title))]; ok {
				row.ExistingName = existing.Name
				row.ExistingPrice = existing.Price
				row.MatchNote = fmt.Sprintf("将覆盖更新（现价 ¥%s）", fmtPrice(existing.Price))
			} else if existing, ok := byName[row.Name]; ok {
				row.ExistingName = existing.Name
				row.ExistingPrice = existing.Price
				row.MatchNote = fmt.Sprintf("将覆盖更新（现价 ¥%s）", fmtPrice(existing.Price))
			} else {
				row.MatchNote = "新增"
			}
		}
		out = append(out, row)
	}
	return out
}

func buildRows(rows [][]string, colMap map[int]columnField) []Row {
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		row := buildRow(r, colMap)
		row.Raw = strings.Join(r, " | ")
		out = append(out, row)
	}
	return out
}

// RawTable 返回文件原始表格（表头 + 行），供 AI 解析提示词使用。
func RawTable(path string) (columns []string, rows [][]string, err error) {
	abs := path
	if !filepath.IsAbs(path) {
		wd, werr := os.Getwd()
		if werr != nil {
			return nil, nil, werr
		}
		abs = filepath.Join(wd, path)
	}
	ext := strings.ToLower(filepath.Ext(abs))
	switch ext {
	case ".xlsx", ".xlsm", ".et", ".ods":
		return readSheet(abs)
	case ".csv", ".tsv":
		return readCSV(abs, ext == ".tsv")
	default:
		return nil, nil, fmt.Errorf("暂不支持 %s 格式", ext)
	}
}

func readSheet(path string) ([]string, [][]string, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, nil, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("xlsx 无工作表")
	}
	raw, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, fmt.Errorf("读取工作表失败: %w", err)
	}
	cols, rows := normalizeTable(raw)
	return cols, rows, nil
}

func readCSV(path string, tsv bool) ([]string, [][]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	enc, _ := fileenc.Detect(b)
	text := string(fileenc.Decode(b, enc))
	sep := ','
	if tsv {
		sep = '\t'
	}
	rd := csv.NewReader(strings.NewReader(text))
	rd.Comma = sep
	rd.LazyQuotes = true
	rd.FieldsPerRecord = -1
	var rows [][]string
	for {
		rec, rerr := rd.Read()
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			break // 容错：跳过坏行
		}
		rows = append(rows, rec)
	}
	cols, data := normalizeTable(rows)
	return cols, data, nil
}

// normalizeTable 裁剪每行到 maxCols 列、去掉全空行。
func normalizeTable(rows [][]string) (cols []string, data [][]string) {
	maxCols := 0
	var out [][]string
	for _, r := range rows {
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
	// 统一列宽：不足补空串，便于按列定位。
	for i := range out {
		for len(out[i]) < maxCols {
			out[i] = append(out[i], "")
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out[0], out[1:]
}

func isHeader(row []string) bool {
	for _, c := range row {
		low := strings.ToLower(strings.TrimSpace(c))
		if low == "" {
			continue
		}
		if strings.Contains(low, "单价") || strings.Contains(low, "价格") ||
			strings.Contains(low, "名称") || strings.Contains(low, "品名") ||
			strings.Contains(low, "单位") || strings.Contains(low, "材料") {
			return true
		}
	}
	return false
}

// likelyHeader 判断一行是否更像真正的表头：含成本字段关键词且至少两列
// 非空，避免把单格标题行（如「材料报价清单」「XX 项目测算表」）误当表头，
// 导致后续把真实表头当数据行解析。
func likelyHeader(row []string) bool {
	if !isHeader(row) {
		return false
	}
	nonEmpty := 0
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			nonEmpty++
		}
	}
	return nonEmpty >= 2
}

func mapColumns(header []string) map[int]columnField {
	out := map[int]columnField{}
	for c, h := range header {
		low := strings.ToLower(strings.TrimSpace(h))
		if low == "" {
			continue
		}
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

// isNoiseHeader 判断表头是否为可忽略的非成本列（序号/数量/合计/备注等）。
func isNoiseHeader(h string) bool {
	low := strings.ToLower(strings.TrimSpace(h))
	for _, kw := range []string{"序号", "编号", "数量", "合计", "小计", "备注", "说明", "日期", "时间"} {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}

func buildRow(r []string, colMap map[int]columnField) Row {
	row := Row{}
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
		case fieldSpec:
			if row.Spec == "" {
				row.Spec = v
			}
		case fieldUnit:
			if row.Unit == "" {
				row.Unit = v
			}
		case fieldPrice:
			if p, ok := parsePrice(v); ok && row.Price <= 0 {
				row.Price = p
			}
		case fieldSource:
			if row.Source == "" {
				row.Source = v
			}
		case fieldCategory:
			if row.Category == "" {
				row.Category = normalizeCategory(v)
			}
		}
	}
	return row
}

// parsePrice 归一化价格：去掉 ¥/元/逗号/空格，支持 "3,200.00"、"3200 元"。
func parsePrice(s string) (float64, bool) {
	clean := strings.NewReplacer(",", "", "，", "", "¥", "", "￥", "", "元", "", " ", "", "\u00a0", "").Replace(strings.TrimSpace(s))
	if clean == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func normalizeCategory(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "机械", "材料", "人工", "运输", "检测", "综合单价", "其他":
		return s
	}
	if strings.Contains(s, "综合单价") {
		return "综合单价"
	}
	for _, kw := range []string{"材料", "水泥", "钢筋", "砂石", "钢材"} {
		if strings.Contains(s, kw) {
			return "材料"
		}
	}
	if strings.Contains(s, "人工") || strings.Contains(s, "工日") || strings.Contains(s, "工资") {
		return "人工"
	}
	for _, kw := range []string{"机械", "设备", "台班", "租赁"} {
		if strings.Contains(s, kw) {
			return "机械"
		}
	}
	if strings.Contains(s, "运输") || strings.Contains(s, "运费") || strings.Contains(s, "外运") {
		return "运输"
	}
	if strings.Contains(s, "检测") || strings.Contains(s, "试验") {
		return "检测"
	}
	return "其他"
}

func fmtPrice(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
