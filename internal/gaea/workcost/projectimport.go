// projectimport.go —— 实测项目工作簿（五表模版）解析器。
//
// 模版形态（旺平矿业 2026-10-07 / 什邡 2026-10-06，均为 gaea 项目空间导出）：
//
//	封面       —— 项目名/地点/工期/组价口径（自述「综合单价=工序消耗量×基础资源价」）
//	费用汇总   —— 序号|分部|编码|项目名称|工序特征|单位|工程量|综合单价|合价|单价位置|依据
//	              + 下半段取费（管理费/利润/规费/税/控制价）
//	综合单价   —— 序号|清单编码|项目名称|单位|行类|资源编码|资源名称|消耗单位|消耗量|单价|合价
//	              （汇总行 P=Σ明细；明细行 单价=VLOOKUP 工料机价格）
//	工料机价格 —— 编码|类别|名称|规格/计算式|单位|不含税单价|来源|备注
//	工程量计算 —— 编码|名称|数值|单位|公式/口径|来源|去向|备注
//
// 本解析器把前三张表映射到工料法三层：资源库 / 消耗量（工序） / 综合单价，
// 并把工程量与取费参数一并带出。**只解析、不落库**——落库由调用方在用户确认后执行。
//
// 公式单元格（台班单价 =900+F4*90+F5、数量 =1/3/2）：优先取公式串保留口径，
// 同时取缓存值作为数值。无缓存值时数值为 0 并记入 Warnings（不静默当 0 用）。
package workcost

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Sheet 名常量（模版固定命名；容错见 sheetByName）。
const (
	sheetCover      = "封面"
	sheetSummary    = "费用汇总"
	sheetUnitPrice  = "综合单价"
	sheetResourcePr = "工料机价格"
	sheetQuantity   = "工程量计算"
)

// 备选 sheet 名（什邡工程量表名为「工程量核算」，清单表名为「清单测算」）。
var sheetAliases = map[string][]string{
	sheetSummary:    {"费用汇总", "清单测算"},
	sheetQuantity:   {"工程量计算", "工程量核算"},
	sheetUnitPrice:  {"综合单价"},
	sheetResourcePr: {"工料机价格"},
}

// ProjectResource 解析出的工料机资源（工料机价格表一行）。
type ProjectResource struct {
	Code         string  `json:"code"`
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	Spec         string  `json:"spec"`
	Unit         string  `json:"unit"`
	Price        float64 `json:"price"`
	PriceFormula string  `json:"priceFormula"` // 台班公式原文（留口径追溯）
	Source       string  `json:"source"`
	Note         string  `json:"note"`
}

// ProjectConsumption 解析出的消耗量行（综合单价表明细行 = 工序消耗）。
type ProjectConsumption struct {
	Seq           string  `json:"seq"` // 1.1 / 2.3 明细序号
	ItemCode      string  `json:"itemCode"`
	ItemTitle     string  `json:"itemTitle"`
	ItemUnit      string  `json:"itemUnit"`
	Kind          string  `json:"kind"` // 行类：人工/材料/机械/外委
	ResourceCode  string  `json:"resourceCode"`
	ResourceTitle string  `json:"resourceTitle"`
	Unit          string  `json:"unit"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	PriceFormula  string  `json:"priceFormula"`
	Amount        float64 `json:"amount"`
}

// ProjectItem 解析出的清单项（综合单价表汇总行 + 费用汇总的工程量）。
type ProjectItem struct {
	Code         string  `json:"code"` // 清单编码（WP01/BJ01…）
	Title        string  `json:"title"`
	Unit         string  `json:"unit"`
	Division     string  `json:"division"` // 分部（A临建/B支护降水…）
	Feature      string  `json:"feature"`  // 工序特征
	Quantity     float64 `json:"quantity"` // 工程量（费用汇总引用工程量表）
	QuantityExpr string  `json:"quantityExpr"`
	CategoryPath string  `json:"categoryPath"` // 综合单价/分部
}

// ProjectFeeParams 解析出的取费参数（费用汇总下半段）。
type ProjectFeeParams struct {
	ManagementRate    float64 `json:"managementRate"`
	RegulatoryRate    float64 `json:"regulatoryRate"`
	ProfitRate        float64 `json:"profitRate"`
	TaxRate           float64 `json:"taxRate"`
	ControlPrice      float64 `json:"controlPrice"`
	ManagementFormula string  `json:"managementFormula"`
	TaxNote           string  `json:"taxNote"`
}

// ProjectQuantity 解析出的工程量计算条目（工程量计算表）。
type ProjectQuantity struct {
	Code  string  `json:"code"`
	Title string  `json:"title"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Note  string  `json:"note"`
}

// SkippedRow 解析时跳过的行（显形而非静默丢弃）。
type SkippedRow struct {
	Sheet  string `json:"sheet"`
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ProjectBundle 一份项目工作簿的完整解析结果（待用户确认后落库）。
type ProjectBundle struct {
	Path       string               `json:"path"`
	FileName   string               `json:"fileName"`
	Project    string               `json:"project"`
	Location   string               `json:"location"`
	Duration   string               `json:"duration"`
	Pricing    string               `json:"pricing"` // 封面的组价口径自述
	Resources  []ProjectResource    `json:"resources"`
	Items      []ProjectItem        `json:"items"`
	Lines      []ProjectConsumption `json:"lines"`
	Fee        ProjectFeeParams     `json:"fee"`
	Quantities []ProjectQuantity    `json:"quantities"`
	Skipped    []SkippedRow         `json:"skipped"`
	Warnings   []string             `json:"warnings"`
	Sheets     []string             `json:"sheets"`
}

// DetectedFiveSheet 报告该工作簿是否为新五表模版（供导入入口分流）。
func DetectedFiveSheet(sheets []string) bool {
	hasResource, hasUnitPrice := false, false
	for _, s := range sheets {
		n := strings.TrimSpace(s)
		for _, a := range sheetAliases[sheetResourcePr] {
			if n == a {
				hasResource = true
			}
		}
		for _, a := range sheetAliases[sheetUnitPrice] {
			if n == a {
				hasUnitPrice = true
			}
		}
	}
	return hasResource && hasUnitPrice
}

// ParseProjectWorkbook 解析一份五表项目工作簿。
func ParseProjectWorkbook(path string) (*ProjectBundle, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, fmt.Errorf("打开工作簿失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	b := &ProjectBundle{
		Path:     filepath.ToSlash(path),
		FileName: filepath.Base(path),
		Sheets:   f.GetSheetList(),
		// 切片构造即非 nil：nil 经 Wails 序列化成 JSON null，前端
		// `bundle.warnings.length` / `bundle.items.length` 会崩（与 Compose 同根）。
		Resources:  []ProjectResource{},
		Items:      []ProjectItem{},
		Lines:      []ProjectConsumption{},
		Quantities: []ProjectQuantity{},
		Skipped:    []SkippedRow{},
		Warnings:   []string{},
	}
	if !DetectedFiveSheet(b.Sheets) {
		return nil, fmt.Errorf("不是五表模版（需要「%s」与「%s」两张表），实得表：%v",
			sheetResourcePr, sheetUnitPrice, b.Sheets)
	}

	if sn := sheetByName(f, sheetCover); sn != "" {
		parseCover(f, sn, b)
	}
	if sn := sheetByName(f, sheetResourcePr); sn != "" {
		parseResourceSheet(f, sn, b)
	}
	if sn := sheetByName(f, sheetUnitPrice); sn != "" {
		parseUnitPriceSheet(f, sn, b)
	}
	if sn := sheetByName(f, sheetSummary); sn != "" {
		parseSummarySheet(f, sn, b)
	}
	if sn := sheetByName(f, sheetQuantity); sn != "" {
		parseQuantitySheet(f, sn, b)
	}

	b.Warnings = append(b.Warnings, crossCheck(b)...)
	return b, nil
}

// sheetByName 按名称或别名找 sheet（返回实际 sheet 名，找不到返回空串）。
func sheetByName(f *excelize.File, canonical string) string {
	list := f.GetSheetList()
	names := sheetAliases[canonical]
	if len(names) == 0 {
		names = []string{canonical}
	}
	for _, want := range names {
		for _, sn := range list {
			if strings.TrimSpace(sn) == want {
				return sn
			}
		}
	}
	return ""
}

// ── 单元格读取 ──────────────────────────────────────────────────────

// cellText 取单元格文本（公式取公式原文，否则取显示值）。
func cellText(f *excelize.File, sheet, cell string) string {
	if formula, err := f.GetCellFormula(sheet, cell); err == nil && strings.TrimSpace(formula) != "" {
		return "=" + strings.TrimSpace(formula)
	}
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

// cellValue 取单元格数值：有缓存值时用它；无缓存但公式指向另一单元格
// （如 =工料机价格!F44）则递归取一次目标值；否则 0。
func cellValue(f *excelize.File, sheet, cell string) float64 {
	if v, err := f.GetCellValue(sheet, cell); err == nil {
		if n, ok := parseNum(v); ok {
			return n
		}
	}
	// 公式：尝试解析简单引用（='表'!F44 或 =F44）取目标缓存的公式或值。
	formula, err := f.GetCellFormula(sheet, cell)
	if err != nil || strings.TrimSpace(formula) == "" {
		return 0
	}
	return resolveFormulaRef(f, sheet, formula)
}

// resolveFormulaRef 解析形如 =工料机价格!F44 / ='工料机价格'!F44 / =F44 的引用。
// 仅支持单层引用（模版里的跨表取价就是这一形态）；算术公式（台班费）不做
// 求值——那需要电子表格引擎，且原表已存缓存值，走 cellValue 第一段即可。
func resolveFormulaRef(f *excelize.File, sheet, formula string) float64 {
	expr := strings.TrimPrefix(strings.TrimSpace(formula), "=")
	expr = strings.ReplaceAll(expr, "$", "")
	target := expr
	if i := strings.Index(expr, "!"); i >= 0 {
		refSheet := strings.Trim(strings.TrimSpace(expr[:i]), "'")
		target = strings.TrimSpace(expr[i+1:])
		if refSheet == "" || !isCellRef(target) {
			return 0
		}
		if v, err := f.GetCellValue(refSheet, target); err == nil {
			if n, ok := parseNum(v); ok {
				return n
			}
		}
		// 目标本身也是公式 → 递归一次（更深层的算术公式不求值）。
		if fo, err := f.GetCellFormula(refSheet, target); err == nil && strings.TrimSpace(fo) != "" {
			return resolveFormulaRef(f, refSheet, fo)
		}
		return 0
	}
	if !isCellRef(target) {
		return 0
	}
	return 0
}

// isCellRef 判断是否为纯单元格引用（列字母+行号）。
func isCellRef(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	for i < len(s) && (s[i] >= 'A' && s[i] <= 'Z' || s[i] >= 'a' && s[i] <= 'z') {
		i++
	}
	if i == 0 || i == len(s) {
		return false
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseNum 宽松解析数字（含千分位、百分号、全角空格）。
func parseNum(s string) (float64, bool) {
	t := strings.TrimSpace(s)
	t = strings.ReplaceAll(t, ",", "")
	t = strings.ReplaceAll(t, "，", "")
	t = strings.ReplaceAll(t, "\u00a0", "")
	if t == "" {
		return 0, false
	}
	if strings.HasSuffix(t, "%") {
		if v, err := strconv.ParseFloat(strings.TrimSuffix(t, "%"), 64); err == nil {
			return v / 100, true
		}
		return 0, false
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// rowValues 读一行显示值（1-based 行号，maxCol 列）。
//
// ⚠️ 合并单元格：excelize 对合并区域内的**每个**格子都返回同一文本（实测
// B2..G2 全等），直接拼接会把标题重复 N 次、把合并的说明串误当「标签-值」对。
// 故这里对**相邻重复**去重：合并区一定连续，去重即还原合并语义（保留列位）。
func rowValues(f *excelize.File, sheet string, row, maxCol int) []string {
	out := make([]string, 0, maxCol)
	prev := ""
	for c := 1; c <= maxCol; c++ {
		cell, err := excelize.CoordinatesToCellName(c, row)
		if err != nil {
			out = append(out, "")
			continue
		}
		v := cellText(f, sheet, cell)
		if v != "" && v == prev {
			out = append(out, "")
			continue
		}
		out = append(out, v)
		if v != "" {
			prev = v
		}
	}
	return out
}

// rowIsMergedUniform 判断该行是否为「水平合并的整段文本」（非空格子文本全同）。
// 标题行/整段说明行属此形态，其中的「标签-值」配对扫描必须跳过。
func rowIsMergedUniform(vals []string) bool {
	first := ""
	n := 0
	for _, v := range vals {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if n == 0 {
			first = v
		} else if v != first {
			return false
		}
		n++
	}
	return n >= 2
}

// findHeaderRow 在前 limit 行内查找包含全部 keys 的行（返回 1-based 行号，0=未找到）。
func findHeaderRow(f *excelize.File, sheet string, maxCol, limit int, keys ...string) int {
	for r := 1; r <= limit; r++ {
		vals := rowValues(f, sheet, r, maxCol)
		joined := strings.Join(vals, "|")
		ok := true
		for _, k := range keys {
			if !strings.Contains(joined, k) {
				ok = false
				break
			}
		}
		if ok {
			return r
		}
	}
	return 0
}

// colIndex 在表头行里定位列（1-based；0=未找到）。按 keys 依次匹配，
// exact=true 时要求去空格后完全相等。
func colIndex(header []string, exact bool, keys ...string) int {
	for _, k := range keys {
		k = strings.TrimSpace(k)
		for i, h := range header {
			hn := strings.TrimSpace(h)
			if exact {
				if hn == k {
					return i + 1
				}
				continue
			}
			if strings.Contains(hn, k) {
				return i + 1
			}
		}
	}
	return 0
}
