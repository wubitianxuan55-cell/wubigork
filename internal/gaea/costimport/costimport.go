// Package costimport 成本库文件导入解析：读取 xlsx/csv 报价单/测算表，
// 自动映射列（名称/规格/单位/单价/来源/分类）、归一化价格、并对既有
// 成本条目做同名匹配（标记「新增」或「将覆盖更新」），供前端预览确认后
// 批量入库。遵循「无确认不落库」：本包只产出候选，写库由调用方执行。
package costimport

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

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
	// 综合单价架构（用户定调）：人材机二级 = 三个金额合计 + 组成明细行。
	LaborFee      float64
	MaterialFee   float64
	MachineFee    float64
	ManagementFee float64 // 管理费（元，仅展示追溯）
	ProfitFee     float64 // 利润（元，仅展示追溯）
	AdvanceFee    float64 // 垫资（元，仅展示追溯）
	TaxRate       float64 // 税率（%，仅展示追溯）
	Components    []cost.Component
	Body          string
	Spec         string
	Source       string
	SourceRow    int     // 原始工作表物理行号（1-based；0=无法确定，如纵向参数表/AI 解析）
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
	fieldLabor
	fieldMaterial
	fieldMachine
	fieldAnalysis
	fieldMgmt
	fieldProfit
	fieldAdvance
)

var fieldKeywords = map[columnField][]string{
	fieldTitle:    {"材料名称", "项目名称", "名称规格", "品名", "材料", "物资", "项目", "科目", "设备", "机械", "名称", "内容", "子目"},
	fieldSpec:     {"规格型号", "规格", "型号", "材质", "参数"},
	fieldUnit:     {"单位"},
	fieldPrice:    {"含税单价", "不含税单价", "市场价", "信息价", "报价", "单价", "价格", "金额"},
	fieldSource:   {"供应商", "来源", "品牌", "产地"},
	fieldCategory: {"分类", "类别"},
	fieldLabor:    {"人工费"},
	fieldMaterial: {"材料费"},
	fieldMachine:  {"机械费"},
	fieldAnalysis: {"综合单价分析", "单价分析"},
	fieldMgmt:     {"管理费", "管理"},
	fieldProfit:   {"利润"},
	fieldAdvance:  {"垫资"},
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
		// 《市政成本测算手册》专有格式：任何一张 sheet 出现「综合单价分析」+
		// 人工费/材料费/机械费 表头即按手册多表解析（首张可能是地区系数表）。
		if detectManualFormat(abs) {
			pv := &Preview{
				Path:     filepath.ToSlash(abs),
				FileName: filepath.Base(abs),
				Message:  "识别到《市政成本测算手册》格式：按 综合单价/专业/分部 提取子目，并解析「综合单价分析」为人材机组成。",
			}
			pv.Rows = MatchRows(parseManualSheets(abs), store)
			return pv, nil
		}
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
	// 物理行号：表头之后的第一条数据行在原始表中的 1-based 行号。
	firstDataRow := 2
	if header := findHeaderRow(rows); header >= 0 {
		firstDataRow = header + 2
	}
	pv.Rows = MatchRows(buildRows(dataRows, colMap, firstDataRow), store)
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

func buildRows(rows [][]string, colMap map[int]columnField, firstDataRow int) []Row {
	out := make([]Row, 0, len(rows))
	for i, r := range rows {
		row := buildRow(r, colMap)
		row.SourceRow = firstDataRow + i
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
		case fieldLabor:
			if vv, ok := parseZeroPrice(v); ok {
				row.LaborFee = vv
			}
		case fieldMaterial:
			if vv, ok := parseZeroPrice(v); ok {
				row.MaterialFee = vv
			}
		case fieldMachine:
			if vv, ok := parseZeroPrice(v); ok {
				row.MachineFee = vv
			}
		case fieldMgmt:
			if vv, ok := parseZeroPrice(v); ok {
				row.ManagementFee = vv
			}
		case fieldProfit:
			if vv, ok := parseZeroPrice(v); ok {
				row.ProfitFee = vv
			}
		case fieldAdvance:
			if vv, ok := parseZeroPrice(v); ok {
				row.AdvanceFee = vv
			}
		case fieldAnalysis:
			if row.Components == nil && strings.TrimSpace(v) != "" {
				row.Components = parseAnalysisComponents(v)
			}
		}
	}
	return row
}

// parseZeroPrice 解析数值（允许 0：人工费/材料费/机械费列常为 0）。
func parseZeroPrice(s string) (float64, bool) {
	clean := strings.NewReplacer(",", "", "，", "", "¥", "", "￥", "", "元", "", " ", "", "\u00a0", "").Replace(strings.TrimSpace(s))
	if clean == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, false
	}
	return v, true
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

// ── 《市政成本测算手册》专有格式解析（综合单价架构）──────────────
//
// 手册结构：每张专业 sheet（道路/交通/绿化/电力/给水/暖气/与雨污/照明）
// = 表头（序号/工程名称/项目特征及工作内容/计量单位/含税单价/综合单价分析/
// 人工费/材料费/机械费/管理(3%)/利润(10%)/垫资(3%)）+ 分部行 + 子目行。
// 「综合单价分析」列是人工费/材料费/机械费 明细文本，解析为组成行（二级）。

// isManualFormat 判断是否《市政成本测算手册》专有格式：前几行同时出现
// 「综合单价分析」表头与 人工费/材料费/机械费 三列。
func isManualFormat(rows [][]string) bool {
	hasAnalysis, hasLabor, hasMaterial, hasMachine := false, false, false, false
	limit := 4
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		for _, c := range rows[i] {
			v := normCell(c)
			if strings.Contains(v, "综合单价分析") {
				hasAnalysis = true
			}
			if strings.Contains(v, "人工费") {
				hasLabor = true
			}
			if strings.Contains(v, "材料费") {
				hasMaterial = true
			}
			if strings.Contains(v, "机械费") {
				hasMachine = true
			}
		}
	}
	return hasAnalysis && hasLabor && hasMaterial && hasMachine
}

// detectManualFormat 遍历全部 sheet 探测手册格式（首张常为地区系数表，
// 不能只读第一张）。
func detectManualFormat(path string) bool {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return false
	}
	defer f.Close()
	for _, sn := range f.GetSheetList() {
		raw, err := f.GetRows(sn)
		if err != nil {
			continue
		}
		if isManualFormat(raw) {
			return true
		}
	}
	return false
}

// parseManualSheets 解析手册全部专业 sheet，产出综合单价子目候选行
// （分类路径=综合单价/专业/分部，含人材机合计、费率与组成明细）。
func parseManualSheets(path string) []Row {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []Row
	used := map[string]bool{}
	for _, sn := range f.GetSheetList() {
		name := strings.TrimSpace(sn)
		if name == "原始" || name == "人工系数" {
			continue // 地区人工折算系数表，非成本子目
		}
		spec := manualSpecialty(name)
		raw, err := f.GetRows(sn)
		if err != nil {
			continue
		}
		rows := manualRows(raw)
		header := findManualHeaderRow(rows)
		if header < 0 {
			continue
		}
		cols := manualHeaderCols(rows, header)
		section := ""
		for i := header + 1; i < len(rows); i++ {
			row := buildManualRow(rows[i], cols, spec, &section, i-header)
			if row == nil {
				continue
			}
			row.Title = manualDistinctTitle(row.Title, row.Body, strconv.Itoa(row.SourceRow), used)
			out = append(out, *row)
		}
	}
	return out
}

// manualSpecialty sheet 名 → 专业名（对齐默认分类树二级）。
func manualSpecialty(sheet string) string {
	switch strings.TrimSpace(sheet) {
	case "道路":
		return "道路工程"
	case "交通":
		return "交通工程"
	case "绿化":
		return "绿化工程"
	case "电力":
		return "电力工程"
	case "给水":
		return "给水工程"
	case "暖气":
		return "暖气工程"
	case "与雨污", "雨污":
		return "雨污工程"
	case "照明":
		return "照明工程"
	}
	return strings.TrimSpace(sheet)
}

// manualRows 裁剪/去空行（保留首行：手册表头可能跨多行）。
func manualRows(raw [][]string) [][]string {
	maxCols := 0
	var out [][]string
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
	for i := range out {
		for len(out[i]) < maxCols {
			out[i] = append(out[i], "")
		}
	}
	return out
}

// findManualHeaderRow 在前几行里找表头行（含 序号/工程名称/计量单位/
// 综合单价分析 等关键词最多的行；照明等表头在第二行）。
func findManualHeaderRow(rows [][]string) int {
	limit := 5
	if len(rows) < limit {
		limit = len(rows)
	}
	best, bestScore := -1, 0
	for i := 0; i < limit; i++ {
		score := 0
		for _, c := range rows[i] {
			v := normCell(c)
			for _, kw := range []string{"序号", "工程名称", "项目名称", "计量单位", "综合单价分析", "人工费", "材料费", "机械费", "报价", "管理", "利润", "垫资"} {
				if strings.Contains(v, kw) {
					score++
					break
				}
			}
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if bestScore < 3 {
		return -1
	}
	return best
}

// manualHeaderCols 定位手册各字段列：在表头行 ±1 行内按关键词找列；
// 找不到时回退手册固定列位。
func manualHeaderCols(rows [][]string, header int) map[string]int {
	cols := map[string]int{}
	from, to := header-1, header+1
	if from < 0 {
		from = 0
	}
	if to >= len(rows) {
		to = len(rows) - 1
	}
	fields := []struct {
		key string
		kws []string
	}{
		{"seq", []string{"序号"}},
		{"name", []string{"工程名称", "项目名称"}},
		{"feature", []string{"项目特征及工作内容", "项目特征"}},
		{"unit", []string{"计量单位"}},
		{"price", []string{"含税单价", "报价"}},
		{"analysis", []string{"综合单价分析"}},
		{"labor", []string{"人工费"}},
		{"material", []string{"材料费"}},
		{"machine", []string{"机械费"}},
		{"mgmt", []string{"管理"}},
		{"profit", []string{"利润"}},
		{"advance", []string{"垫资"}},
	}
	for _, f := range fields {
		for i := from; i <= to; i++ {
			found := false
			for c, cell := range rows[i] {
				v := normCell(cell)
				for _, kw := range f.kws {
					if strings.Contains(v, kw) {
						cols[f.key] = c
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
	}
	defaults := map[string]int{
		"seq": 0, "name": 1, "feature": 2, "unit": 3, "price": 5,
		"analysis": 7, "labor": 8, "material": 9, "machine": 10,
		"mgmt": 11, "profit": 12, "advance": 13,
	}
	for k, v := range defaults {
		if _, ok := cols[k]; !ok {
			cols[k] = v
		}
	}
	return cols
}

// buildManualRow 把手册一行转为候选成本条目；分部行更新 section 并返回 nil。
func buildManualRow(r []string, cols map[string]int, specialty string, section *string, seqNo int) *Row {
	seq := strings.TrimSpace(cellAt(r, cols["seq"]))
	name := normCell(cellAt(r, cols["name"]))
	price, hasPrice := parseZeroPrice(cellAt(r, cols["price"]))
	// 分部行：序号列有文字但非数字，名称列为空（如 土方工程 / 3.机动车道）。
	if seq != "" && !isNumeric(seq) && name == "" {
		if clean := cleanSection(seq); clean != "" {
			*section = clean
		}
		return nil
	}
	hasSeq := seq != "" && isNumeric(seq)
	if name == "" || (!hasSeq && !hasPrice) {
		return nil
	}
	row := &Row{
		Title:      name,
		Unit:       normCell(cellAt(r, cols["unit"])),
		Price:      price,
		Source:     "市政成本测算手册/" + specialty,
		SourceRow:  seqNo + 1, // 表头行+1 为物理行号（近似，1-based）
		Components: parseAnalysisComponents(cellAt(r, cols["analysis"])),
	}
	if labor, ok := parseZeroPrice(cellAt(r, cols["labor"])); ok {
		row.LaborFee = labor
	}
	if material, ok := parseZeroPrice(cellAt(r, cols["material"])); ok {
		row.MaterialFee = material
	}
	if machine, ok := parseZeroPrice(cellAt(r, cols["machine"])); ok {
		row.MachineFee = machine
	}
	if v, ok := parseZeroPrice(cellAt(r, cols["mgmt"])); ok {
		row.ManagementFee = v
	}
	if v, ok := parseZeroPrice(cellAt(r, cols["profit"])); ok {
		row.ProfitFee = v
	}
	if v, ok := parseZeroPrice(cellAt(r, cols["advance"])); ok {
		row.AdvanceFee = v
	}
	row.TaxRate = 9 // 手册表头统一 含税单价(9%)
	if feature := strings.TrimSpace(cellAt(r, cols["feature"])); feature != "" {
		row.Body = "项目特征及工作内容：\n" + truncateRunes(feature, 3000)
	}
	cat := "综合单价/" + specialty
	if *section != "" {
		cat += "/" + *section
	}
	row.Category = cat
	row.Raw = strings.Join(r, " | ")
	return row
}

// manualDistinctTitle 同批次内同名子目追加特征片段去重（手册同一子目常因
// 深度/规格不同出现多行，直接同名会互相覆盖）。
func manualDistinctTitle(base, feature, seq string, used map[string]bool) string {
	candidate := base
	if !used[candidate] {
		used[candidate] = true
		return candidate
	}
	if frag := extractShortFeature(feature); frag != "" {
		candidate = base + "（" + frag + "）"
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
	candidate = base + "（序号" + seq + "）"
	used[candidate] = true
	return candidate
}

// extractShortFeature 从项目特征文本提取简短判别片段（编号行去前缀拼接）。
var manualFeatureLine = regexp.MustCompile(`^\s*\d+[\.、．]`)

func extractShortFeature(feature string) string {
	var parts []string
	for _, line := range strings.Split(feature, "\n") {
		t := strings.TrimSpace(line)
		if manualFeatureLine.MatchString(t) {
			t = manualFeatureLine.ReplaceAllString(t, "")
			t = strings.TrimSpace(t)
			if t != "" {
				parts = append(parts, t)
			}
			if len(parts) >= 3 {
				break
			}
		}
	}
	frag := strings.Join(parts, "；")
	frag = cleanCJKSpaces(frag)
	if frag == "" {
		for _, line := range strings.Split(feature, "\n") {
			if t := strings.TrimSpace(line); t != "" {
				frag = t
				break
			}
		}
	}
	return truncateRunes(frag, 40)
}

// cleanSection 清理分部行文本：剥序号、去括号备注、压缩汉字间空格。
var sectionPrefix = regexp.MustCompile(`^[\s\d一二三四五六七八九十]*[\.、．·]?[\s]*`)
var cjkSpace = regexp.MustCompile(`([\p{Han}])\s+([\p{Han}])`)

func cleanSection(s string) string {
	s = sectionPrefix.ReplaceAllString(strings.TrimSpace(s), "")
	if i := strings.IndexAny(s, "（("); i >= 0 {
		s = s[:i]
	}
	s = cleanCJKSpaces(s)
	return truncateRunes(s, 30)
}

func cleanCJKSpaces(s string) string {
	return cjkSpace.ReplaceAllString(strings.TrimSpace(s), "$1$2")
}

func cellAt(r []string, idx int) string {
	if idx < 0 || idx >= len(r) {
		return ""
	}
	return r[idx]
}

// normCell 归一化单元格：换行/制表符折叠为空格，压缩连续空白，去汉字间空格。
func normCell(s string) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	return cleanCJKSpaces(s)
}

// ── 「综合单价分析」文本 → 人材机组成行 ──────────────────────────

// parseAnalysisComponents 解析综合单价分析文本为组成明细行：
// 人工费/材料费/机械费 段头切换 kind；明细行提取 名称/金额/单价/含量，
// 原始行保留在 Note（损耗系数等表达式不丢）。
func parseAnalysisComponents(text string) []cost.Component {
	var out []cost.Component
	kind := ""
	pendingTitle := "" // 无金额断行暂存（OCR 换行断句），等下一行带金额的明细认领
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if h, ok := analysisSectionHeader(line); ok {
			kind = h
			pendingTitle = ""
			continue
		}
		stripped := stripListMarker(line)
		if stripped == "" {
			continue
		}
		if !strings.Contains(stripped, "元") {
			// 无金额标注的续行/断行（OCR 换行断句）：暂存为待认领标题，
			// 下一行带金额的明细若缺标题则认领它。
			pendingTitle = truncateRunes(stripped, 80)
			continue
		}
		amount, hasAmount := trailingAmount(stripped)
		if !hasAmount {
			continue // 纯说明/续行：金额缺失无法成行
		}
		title := componentTitle(stripped)
		if !hasHan(title) && pendingTitle != "" {
			title = strings.TrimRightFunc(pendingTitle, func(r rune) bool {
				return unicode.IsDigit(r) || r == ' ' || r == '\t' || r == '.' || r == '（' || r == '('
			})
			title = strings.TrimSpace(title)
		}
		if title == "" {
			continue
		}
		pendingTitle = ""
		c := cost.Component{
			Kind:   kind,
			Title:  title,
			Amount: amount,
			Note:   truncateRunes(stripped, 160),
		}
		if c.Kind == "" {
			c.Kind = kindByKeywords(stripped)
		}
		if p, ok := firstNumber(stripped); ok && p > 0 {
			c.Price = p
		}
		if c.Price <= 0 {
			c.Price = amount
		}
		if c.Amount > 0 && c.Price > 0 && math.Abs(c.Amount-c.Price) > 0.001 {
			c.Quantity = round3(c.Amount / c.Price)
		}
		out = append(out, c)
	}
	return out
}

// hasHan 判断字符串是否含汉字（纯表达式行如「170Kg/m³*0.2m」无汉字）。
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// analysisSectionHeader 判断行是否为 人工费/材料费/机械费 段头（含合并段）。
func analysisSectionHeader(line string) (string, bool) {
	for _, kw := range []string{"人工费", "材料费", "机械费"} {
		if !strings.HasPrefix(line, kw) {
			continue
		}
		kind := normalizeKind(kw)
		rest := strings.TrimSpace(line[len(kw):])
		if strings.HasPrefix(rest, "+") {
			for _, k2 := range []string{"人工费", "材料费", "机械费"} {
				if strings.HasPrefix(strings.TrimSpace(rest[1:]), k2) {
					kind = normalizeKind(kw) + "+" + normalizeKind(k2)
					break
				}
			}
		}
		return kind, true
	}
	return "", false
}

func normalizeKind(k string) string {
	return strings.TrimSuffix(strings.TrimSpace(k), "费")
}

// stripListMarker 剥列表序号（1. / 2、 / 3）等；「3.9元」这类价格前缀不误剥。
func stripListMarker(s string) string {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return s
	}
	if i < len(s) && strings.ContainsRune(".、．)）", rune(s[i])) {
		rest := strings.TrimLeft(s[i+1:], " \t")
		if rest == "" || (rest[0] >= '0' && rest[0] <= '9') {
			return s
		}
		return rest
	}
	return s
}

// componentTitle 提取明细行名称：截到首个「数字+元」价格表达式，剥残余算术尾。
func componentTitle(line string) string {
	s := strings.TrimSpace(line)
	if idx := firstPriceExprStart(s); idx > 0 {
		s = s[:idx]
	}
	s = strings.TrimRightFunc(s, func(r rune) bool {
		return unicode.IsDigit(r) || r == '*' || r == '×' || r == '=' || r == '.' ||
			r == ' ' || r == '\t' || r == '（' || r == '('
	})
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "：:，,;；")
	s = stripListMarker(s)
	s = strings.TrimSpace(s)
	return truncateRunes(s, 80)
}

// firstPriceExprStart 返回首个「数字紧邻/间隔空格+元」的位置（价格表达式起点）。
func firstPriceExprStart(s string) int {
	rs := []rune(s)
	for i := 0; i+1 < len(rs); i++ {
		if rs[i] >= '0' && rs[i] <= '9' && rs[i+1] == '元' {
			return byteIndexOfRune(s, i)
		}
	}
	for i := 0; i+2 < len(rs); i++ {
		if rs[i] >= '0' && rs[i] <= '9' && rs[i+1] == ' ' && rs[i+2] == '元' {
			return byteIndexOfRune(s, i)
		}
	}
	return -1
}

func byteIndexOfRune(s string, runeIdx int) int {
	return len(string([]rune(s)[:runeIdx]))
}

// trailingAmount 提取行尾金额：优先「最后一个元」前紧邻数字，否则行末数字。
func trailingAmount(line string) (float64, bool) {
	// 1) 末尾「=数字」优先（计算式结果值，如 …*1.02=53.86）。
	if i := strings.LastIndex(line, "="); i >= 0 {
		if v, ok := trailingNumber(line[i+1:]); ok {
			return v, true
		}
	}
	// 2) 最后一个「元」前紧邻数字。
	if i := strings.LastIndex(line, "元"); i > 0 {
		if v, ok := trailingNumber(line[:i]); ok {
			return v, true
		}
	}
	// 3) 行末数字兜底。
	return trailingNumber(line)
}

// trailingNumber 从字符串末尾向前提取数字（跳过尾部空格）。
func trailingNumber(s string) (float64, bool) {
	s = strings.TrimRight(s, " \t")
	end := len(s)
	start := end
	for start > 0 {
		c := s[start-1]
		if c >= '0' && c <= '9' || c == '.' {
			start--
			continue
		}
		break
	}
	if start == end {
		return 0, false
	}
	v, err := strconv.ParseFloat(s[start:end], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

var numberRe = regexp.MustCompile(`\d+(?:\.\d+)?`)

func firstNumber(s string) (float64, bool) {
	m := numberRe.FindString(s)
	if m == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func kindByKeywords(s string) string {
	if strings.Contains(s, "机械") || strings.Contains(s, "台班") {
		return "机械"
	}
	if strings.Contains(s, "材料") || strings.Contains(s, "砼") || strings.Contains(s, "混凝土") ||
		strings.Contains(s, "钢筋") || strings.Contains(s, "管") {
		return "材料"
	}
	return "人工"
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

func truncateRunes(s string, n int) string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= n {
		return string(rs)
	}
	return string(rs[:n])
}
