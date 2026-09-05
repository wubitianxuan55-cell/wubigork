package xlsxpreview

// 条件格式提取（办公 U4 渲染缺口清单 #2 的 Go 前半，零绑定）：
// 把 xlsx 条件格式中「静态可判定」的 CellIs 子集（range + 比较符 + 常量阈值 +
// dxf 样式）提取进预览 JSON，由前端 XlsxPreview 按单元格数值套色渲染。
// expression/colorScale/dataBar/iconSet/top10 等无法静态判定的类型诚实跳过并
// 计数（CondSkipped），绝不猜测近似渲染。
//
// excelize v2.11 没有公开的条件格式读 API（stylesReader/getSheetXMLPath 均为
// 私有），故经 f.Pkg（公开的原始 zip 条目表）直接解析 workbook.xml（sheet 名 →
// r:id → 文件路径）、worksheet XML（conditionalFormatting/cfRule）与
// styles.xml（dxfs 差异样式）三处。

import (
	"encoding/xml"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// CondRule 是一条静态可判定的条件格式规则（CellIs 子集）。
type CondRule struct {
	Range     string   `json:"range"`               // "D2:D100"（sqref 按段拆分）
	Op        string   `json:"op"`                  // greaterThan/lessThan/.../between
	Formulas  []string `json:"formulas"`            // 常量阈值（数字或带引号文本）
	Priority  int      `json:"priority,omitempty"`  // Excel 优先级（小者先行）
	Fill      string   `json:"fill,omitempty"`      // 6 位 hex（同 CellStyle.Fill 口径）
	FontColor string   `json:"fontColor,omitempty"` // 6 位 hex
	Bold      bool     `json:"bold,omitempty"`
}

// condSupportedOps 静态可判定的比较符（与前端 xlsxCondFmt.ts 判定同源锚定）。
var condSupportedOps = map[string]bool{
	"greaterThan":        true,
	"lessThan":           true,
	"greaterThanOrEqual": true,
	"lessThanOrEqual":    true,
	"equal":              true,
	"notEqual":           true,
	"between":            true,
	"notBetween":         true,
}

// ── worksheet XML（conditionalFormatting 段） ──────────────

type cfRuleXML struct {
	Type     string   `xml:"type,attr"`
	DxfID    *int     `xml:"dxfId,attr"`
	Priority int      `xml:"priority,attr"`
	Operator string   `xml:"operator,attr"`
	Formula  []string `xml:"formula"`
}

type condFmtXML struct {
	SQRef  string      `xml:"sqref,attr"`
	CfRule []*cfRuleXML `xml:"cfRule"`
}

type worksheetCfXML struct {
	ConditionalFormatting []*condFmtXML `xml:"conditionalFormatting"`
}

// ── styles.xml（dxfs 差异样式） ────────────────────────────

type cfColorXML struct {
	RGB   string `xml:"rgb,attr"`
	Theme *int   `xml:"theme,attr"` // 主题色无法静态解析 → 该样式放弃
}

type cfFontXML struct {
	B     *struct{}    `xml:"b"`
	Color *cfColorXML  `xml:"color"`
}

type cfPatternFillXML struct {
	PatternType string       `xml:"patternType,attr"`
	FgColor     *cfColorXML  `xml:"fgColor"`
	BgColor     *cfColorXML  `xml:"bgColor"`
}

// dxf 的 fill 元素内还有一层 patternFill（ECMA-376 §18.8.21）
type cfFillXML struct {
	PatternFill *cfPatternFillXML `xml:"patternFill"`
}

type cfDxfXML struct {
	Font *cfFontXML  `xml:"font"`
	Fill *cfFillXML  `xml:"fill"`
}

type cfDxfsXML struct {
	Dxfs []*cfDxfXML `xml:"dxf"`
}

type cfStyleSheetXML struct {
	Dxfs *cfDxfsXML `xml:"dxfs"`
}

// ── workbook.xml + rels（sheet 名 → 文件路径） ─────────────

type cfWorkbookSheet struct {
	Name string `xml:"name,attr"`
	// r:id 属性：Go encoding/xml 对带 URL 的属性命名空间匹配不可靠（实测
	// URL:id 匹配不到），sheet 元素的 id 属性只有 r:id 一个，按本地名匹配。
	RID string `xml:"id,attr"`
}

type cfWorkbookXML struct {
	Sheets []cfWorkbookSheet `xml:"sheets>sheet"`
}

type cfRelXML struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type cfRelsXML struct {
	Relationship []cfRelXML `xml:"Relationship"`
}

// pkgBytes 取 zip 原始条目（f.Pkg 是 excelize 公开的原始包表）。
func pkgBytes(f *excelize.File, path string) ([]byte, bool) {
	v, ok := f.Pkg.Load(path)
	if !ok {
		return nil, false
	}
	b, ok := v.([]byte)
	return b, ok
}

// sheetXMLPath 解析 workbook.xml + rels 得到 sheet 名 → worksheet 内部路径。
func sheetXMLPath(f *excelize.File, sheet string) (string, bool) {
	wbRaw, ok := pkgBytes(f, "xl/workbook.xml")
	if !ok {
		return "", false
	}
	var wb cfWorkbookXML
	if err := xml.Unmarshal(wbRaw, &wb); err != nil {
		return "", false
	}
	rid := ""
	for _, s := range wb.Sheets {
		if strings.EqualFold(s.Name, sheet) {
			rid = s.RID
			break
		}
	}
	if rid == "" {
		return "", false
	}
	relsRaw, ok := pkgBytes(f, "xl/_rels/workbook.xml.rels")
	if !ok {
		return "", false
	}
	var rels cfRelsXML
	if err := xml.Unmarshal(relsRaw, &rels); err != nil {
		return "", false
	}
	for _, r := range rels.Relationship {
		if r.ID != rid {
			continue
		}
		target := strings.ReplaceAll(r.Target, "\\", "/")
		switch {
		case strings.HasPrefix(target, "/"):
			return strings.TrimPrefix(target, "/"), true
		case strings.HasPrefix(target, "xl/"):
			return target, true
		default:
			return "xl/" + target, true
		}
	}
	return "", false
}

// dxfLookup 解析 styles.xml 的 dxfs 表，返回按 dxfId 取 {fill, fontColor, bold} 的函数。
func dxfLookup(f *excelize.File) func(dxfID int) (fill, fontColor string, bold bool) {
	empty := func(int) (string, string, bool) { return "", "", false }
	raw, ok := pkgBytes(f, "xl/styles.xml")
	if !ok {
		return empty
	}
	var ss cfStyleSheetXML
	if err := xml.Unmarshal(raw, &ss); err != nil || ss.Dxfs == nil {
		return empty
	}
	return func(id int) (string, string, bool) {
		if id < 0 || id >= len(ss.Dxfs.Dxfs) {
			return "", "", false
		}
		d := ss.Dxfs.Dxfs[id]
		fill, fontColor, bold := "", "", false
		if d.Fill != nil && d.Fill.PatternFill != nil && d.Fill.PatternFill.PatternType != "none" {
			// dxf fill 的底色在 bgColor（与普通样式 fgColor 语义相反，ECMA-376 §18.8.21）
			if c := d.Fill.PatternFill.BgColor; c != nil && c.Theme == nil && c.RGB != "" {
				fill = normalizeColor(c.RGB)
			}
		}
		if d.Font != nil {
			if d.Font.B != nil {
				bold = true
			}
			if d.Font.Color != nil && d.Font.Color.Theme == nil && d.Font.Color.RGB != "" {
				fontColor = normalizeColor(d.Font.Color.RGB)
			}
		}
		return fill, fontColor, bold
	}
}

// scalarFormula 判定公式元素是否静态常量（数字或带引号文本），原样返回供前端判定。
func scalarFormula(s string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", false
	}
	if strings.HasPrefix(t, "\"") && strings.HasSuffix(t, "\"") && len(t) >= 2 {
		return t, true
	}
	if _, err := strconv.ParseFloat(t, 64); err == nil {
		return t, true
	}
	return "", false
}

// extractCondRules 提取单个工作表的静态可判定条件格式规则。
// 返回（规则按 priority 升序=Excel 优先级，跳过数）。
func extractCondRules(f *excelize.File, sheet string) ([]CondRule, int) {
	path, ok := sheetXMLPath(f, sheet)
	if !ok {
		return nil, 0
	}
	wsRaw, ok := pkgBytes(f, path)
	if !ok {
		return nil, 0
	}
	var ws worksheetCfXML
	if err := xml.Unmarshal(wsRaw, &ws); err != nil {
		return nil, 0
	}
	rules := []CondRule{}
	skipped := 0
	dxfAt := dxfLookup(f)
	for _, cf := range ws.ConditionalFormatting {
		for _, r := range cf.CfRule {
			if r.Type != "cellIs" || !condSupportedOps[r.Operator] || r.DxfID == nil {
				skipped++
				continue
			}
			need := 2
			if r.Operator != "between" && r.Operator != "notBetween" {
				need = 1
			}
			if len(r.Formula) < need {
				skipped++
				continue
			}
			formulas := make([]string, 0, need)
			static := true
			for i := 0; i < need; i++ {
				s, ok := scalarFormula(r.Formula[i])
				if !ok {
					static = false
					break
				}
				formulas = append(formulas, s)
			}
			if !static {
				skipped++ // 阈值引用单元格/函数，无法静态判定
				continue
			}
			fill, fontColor, bold := dxfAt(*r.DxfID)
			if fill == "" && fontColor == "" && !bold {
				skipped++ // 样式不可解析（主题色等），渲染无意义
				continue
			}
			for _, seg := range strings.Fields(cf.SQRef) {
				rules = append(rules, CondRule{
					Range: seg, Op: r.Operator, Formulas: formulas,
					Priority: r.Priority, Fill: fill, FontColor: fontColor, Bold: bold,
				})
			}
		}
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })
	if len(rules) == 0 {
		return nil, skipped
	}
	return rules, skipped
}
