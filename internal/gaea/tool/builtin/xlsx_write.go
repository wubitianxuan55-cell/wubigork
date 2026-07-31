package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(xlsxWrite{}) }

type xlsxWrite struct{}

func (xlsxWrite) Name() string { return "xlsx_write" }

func (xlsxWrite) Description() string {
	return "创建 Excel (.xlsx) 文件：支持表头/数据行、多工作表、公式(=开头的单元格自动识别)、数值类型自动检测、内置图表(bar/line/pie)。兼容 Excel 和 WPS。"
}

func (xlsxWrite) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"输出文件路径（.xlsx）"},
  "sheet_name":{"type":"string","description":"工作表名称（默认 Sheet1，仅单表时生效）"},
  "headers":{"type":"array","items":{"type":"string"},"description":"表头行"},
  "rows":{"type":"array","items":{"type":"array","items":{"type":"string"}},"description":"数据行（=开头的值自动识别为公式）"},
  "sheets":{"type":"array","items":{"type":"object","properties":{
    "name":{"type":"string","description":"工作表名称"},
    "headers":{"type":"array","items":{"type":"string"},"description":"表头行"},
    "rows":{"type":"array","items":{"type":"array","items":{"type":"string"}},"description":"数据行"},
    "chart":{"type":"object","description":"图表配置","properties":{
      "type":{"type":"string","description":"图表类型：bar/column/line/pie"},
      "categories":{"type":"string","description":"分类轴数据范围，如 A1:A5"},
      "values":{"type":"string","description":"数值轴数据范围，如 B1:B5"},
      "title":{"type":"string","description":"图表标题"},
      "position":{"type":"string","description":"图表放置位置，如 D1"}
    }}
  }},"description":"多工作表（替代单表参数）"}
},
"anyOf":[{"required":["path","rows"]},{"required":["path","sheets"]}]
}`)
}

func (xlsxWrite) ReadOnly() bool { return false }

func (xlsxWrite) CompactDescription() string     { return compactDesc["xlsx_write"] }
func (xlsxWrite) CompactSchema() json.RawMessage { return compactSchema["xlsx_write"] }

// xlsxSheet 定义单个工作表
type xlsxSheet struct {
	Name    string     `json:"name"`
	Headers []string   `json:"headers,omitempty"`
	Rows    [][]string `json:"rows"`
	Chart   *xlsxChart `json:"chart,omitempty"`
}

// xlsxChart 定义图表配置
type xlsxChart struct {
	Type       string `json:"type"`       // bar / column / line / pie
	Categories string `json:"categories"` // 如 "A1:A5"
	Values     string `json:"values"`     // 如 "B1:B5"
	Title      string `json:"title,omitempty"`
	Position   string `json:"position"` // 如 "D1"
}

func (xlsxWrite) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path      string      `json:"path"`
		SheetName string      `json:"sheet_name,omitempty"`
		Headers   []string    `json:"headers,omitempty"`
		Rows      [][]string  `json:"rows,omitempty"`
		Sheets    []xlsxSheet `json:"sheets,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	if !strings.HasSuffix(strings.ToLower(p.Path), ".xlsx") {
		return "", fmt.Errorf("path 必须以 .xlsx 结尾")
	}

	// 构建 sheets 列表
	var sheets []xlsxSheet
	if len(p.Sheets) > 0 {
		sheets = p.Sheets
		for i := range sheets {
			if sheets[i].Name == "" {
				sheets[i].Name = fmt.Sprintf("Sheet%d", i+1)
			}
		}
	} else if len(p.Rows) > 0 {
		name := p.SheetName
		if name == "" {
			name = "Sheet1"
		}
		sheets = []xlsxSheet{{Name: name, Headers: p.Headers, Rows: p.Rows}}
	} else {
		return "", fmt.Errorf("必须提供 rows 或 sheets 参数")
	}

	// 统计需要图表的 sheet
	type chartInfo struct {
		sheetIdx   int
		chartIdx   int
		drawingIdx int
		chart      xlsxChart
	}
	var chartInfos []chartInfo
	for si, s := range sheets {
		if s.Chart != nil {
			chartInfos = append(chartInfos, chartInfo{
				sheetIdx:   si,
				chartIdx:   len(chartInfos) + 1,
				drawingIdx: len(chartInfos) + 1,
				chart:      *s.Chart,
			})
		}
	}
	hasCharts := len(chartInfos) > 0

	// 生成 ZIP
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	n := len(sheets)

	// [Content_Types].xml
	var ct strings.Builder
	ct.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	for i := 0; i < n; i++ {
		ct.WriteString(fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i+1))
	}
	for _, ci := range chartInfos {
		ct.WriteString(fmt.Sprintf(`<Override PartName="/xl/charts/chart%d.xml" ContentType="application/vnd.openxmlformats-officedocument.drawingml.chart+xml"/>`, ci.chartIdx))
		ct.WriteString(fmt.Sprintf(`<Override PartName="/xl/drawings/drawing%d.xml" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/>`, ci.drawingIdx))
	}
	ct.WriteString(`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`)
	writeZipEntry(w, "[Content_Types].xml", ct.String())

	// _rels/.rels
	writeZipEntry(w, "_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`)

	// xl/_rels/workbook.xml.rels
	rid := 1
	var rels strings.Builder
	rels.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 0; i < n; i++ {
		rels.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, rid, i+1))
		rid += 2
	}
	// 重置 rid 为 2 开始（偶数为样式/图表）
	rid = 2
	for range chartInfos {
		rid += 2
	}
	rid = (n*2 + 1) // 样式 RID
	rels.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, rid))
	rels.WriteString("\n</Relationships>")
	writeZipEntry(w, "xl/_rels/workbook.xml.rels", rels.String())

	// xl/workbook.xml
	var wb strings.Builder
	wb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets>`)
	for i, s := range sheets {
		wb.WriteString(fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(s.Name), i+1, i*2+1))
	}
	wb.WriteString("</sheets>\n</workbook>")
	writeZipEntry(w, "xl/workbook.xml", wb.String())

	// xl/styles.xml
	writeZipEntry(w, "xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>
<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>
<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/></cellXfs>
</styleSheet>`)

	// 写入每个工作表
	for si, sheet := range sheets {
		allRows := sheet.Rows
		if len(sheet.Headers) > 0 {
			allRows = append([][]string{sheet.Headers}, allRows...)
		}

		maxCols := 0
		for _, row := range allRows {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}
		if maxCols == 0 {
			continue
		}

		// 补全短行
		for i, row := range allRows {
			for len(row) < maxCols {
				row = append(row, "")
			}
			allRows[i] = row
		}

		// 生成 sheet XML
		var sx strings.Builder
		sx.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
		sx.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` + "\n")

		// 列宽
		sx.WriteString("<cols>\n")
		for ci := 0; ci < maxCols; ci++ {
			sx.WriteString(fmt.Sprintf(`<col min="%d" max="%d" width="%d" customWidth="1"/>`, ci+1, ci+1, 12))
		}
		sx.WriteString("</cols>\n")

		sx.WriteString("<sheetData>\n")
		for ri, row := range allRows {
			sx.WriteString(fmt.Sprintf(`<row r="%d">`, ri+1))
			for ci, val := range row {
				ref := fmt.Sprintf("%s%d", colLetter(ci), ri+1)
				val = strings.TrimSpace(val)
				if val == "" {
					sx.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t></t></is></c>`, ref))
				} else if strings.HasPrefix(val, "=") {
					formula := val[1:]
					sx.WriteString(fmt.Sprintf(`<c r="%s"><f>%s</f></c>`, ref, xmlEscape(formula)))
				} else if isNumeric(val) {
					sx.WriteString(fmt.Sprintf(`<c r="%s" s="1"><v>%s</v></c>`, ref, val))
				} else {
					sx.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, xmlEscape(val)))
				}
			}
			sx.WriteString("</row>\n")
		}
		sx.WriteString("</sheetData>\n")

		// 如果有图表，添加 drawing 引用
		for _, ci := range chartInfos {
			if ci.sheetIdx == si {
				sx.WriteString(fmt.Sprintf(`<drawing r:id="rId1"/>` + "\n"))
				break
			}
		}

		sx.WriteString("</worksheet>\n")
		writeZipEntry(w, fmt.Sprintf("xl/worksheets/sheet%d.xml", si+1), sx.String())

		// 写 sheet 的 rels 文件（仅在有图表时）
		for _, ci := range chartInfos {
			if ci.sheetIdx == si {
				sr := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/drawing%d.xml"/>
</Relationships>`, ci.drawingIdx)
				writeZipEntry(w, fmt.Sprintf("xl/worksheets/_rels/sheet%d.xml.rels", si+1), sr)
				break
			}
		}
	}

	// 写图表 XML 和 drawing XML
	for _, ci := range chartInfos {
		// 解析 position (如 "D1" → col=3, row=0)
		posCol, posRow := parseCellRef(ci.chart.Position)
		if ci.chart.Position == "" {
			posCol = maxCols(sheets[ci.sheetIdx], len(sheets[ci.sheetIdx].Headers) > 0)
			posRow = len(sheets[ci.sheetIdx].Rows) + 1
		}

		// 图表生成
		chartName := fmt.Sprintf("Chart%d", ci.chartIdx)
		categoriesRange := fmt.Sprintf("'%s'!$%s", xmlEscape(sheets[ci.sheetIdx].Name), ci.chart.Categories)
		valuesRange := fmt.Sprintf("'%s'!$%s", xmlEscape(sheets[ci.sheetIdx].Name), ci.chart.Values)
		chartXML := buildXlsxChartXML(ci.chart.Type, chartName, ci.chart.Title, categoriesRange, valuesRange)
		writeZipEntry(w, fmt.Sprintf("xl/charts/chart%d.xml", ci.chartIdx), chartXML)

		writeZipEntry(w, fmt.Sprintf("xl/charts/_rels/chart%d.xml.rels", ci.chartIdx), `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/package" Target=""/>
</Relationships>`)

		// drawing XML
		drawingXML := buildXlsxDrawingXML(ci.chartIdx, posCol, posRow)
		writeZipEntry(w, fmt.Sprintf("xl/drawings/drawing%d.xml", ci.drawingIdx), drawingXML)

		// drawing rels
		dr := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart" Target="../charts/chart%d.xml"/>
</Relationships>`, ci.chartIdx)
		writeZipEntry(w, fmt.Sprintf("xl/drawings/_rels/drawing%d.xml.rels", ci.drawingIdx), dr)
	}

	w.Close()

	if err := os.WriteFile(p.Path, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	sheetLabel := fmt.Sprintf("%d 个工作表", n)
	if hasCharts {
		sheetLabel += fmt.Sprintf("，%d 个图表", len(chartInfos))
	}
	return tool.WrapText(fmt.Sprintf("已创建 xlsx 文件：%s（%s）", p.Path, sheetLabel)), nil
}

// maxCols 返回工作表的最大列数
func maxCols(sheet xlsxSheet, hasHeader bool) int {
	max := 0
	for _, row := range sheet.Rows {
		if len(row) > max {
			max = len(row)
		}
	}
	if hasHeader && len(sheet.Headers) > max {
		max = len(sheet.Headers)
	}
	return max
}

// parseCellRef 解析单元格引用如 "D1" → (col=3, row=0)
func parseCellRef(ref string) (int, int) {
	ref = strings.ToUpper(strings.TrimSpace(ref))
	if ref == "" {
		return 0, 0
	}
	col := 0
	i := 0
	for i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z' {
		col = col*26 + int(ref[i]-'A'+1)
		i++
	}
	row := 0
	for i < len(ref) && ref[i] >= '0' && ref[i] <= '9' {
		row = row*10 + int(ref[i]-'0')
		i++
	}
	if col > 0 {
		col-- // 0-based
	}
	if row > 0 {
		row-- // 0-based
	}
	return col, row
}

// buildXlsxChartXML 生成 OOXML 图表 XML
func buildXlsxChartXML(chartType, name, title, categories, values string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<c:chartSpace xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<c:chart>
<c:title>`)
	if title != "" {
		b.WriteString(`<c:tx><c:rich><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr sz="1400" b="1"/><a:t>`)
		b.WriteString(xmlEscape(title))
		b.WriteString(`</a:t></a:r></a:p></c:rich></c:tx>`)
	}
	b.WriteString(`<c:overlay val="0"/>
</c:title>
<c:autoTitleDeleted val="0"/>
<c:plotArea>
<c:layout/>`)

	switch chartType {
	case "bar":
		b.WriteString(buildXlsxChartSeries("bar", name, categories, values))
	case "column":
		b.WriteString(buildXlsxChartSeries("col", name, categories, values))
	case "line":
		b.WriteString(buildXlsxChartSeries("line", name, categories, values))
	case "pie":
		b.WriteString(buildXlsxChartSeries("pie", name, categories, values))
	default:
		b.WriteString(buildXlsxChartSeries("col", name, categories, values))
	}

	// 对于非饼图，添加坐标轴
	if chartType != "pie" {
		b.WriteString(`<c:catAx><c:axId val="1"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="b"/><c:crossAx val="2"/></c:catAx>
<c:valAx><c:axId val="2"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="l"/><c:crossAx val="1"/></c:valAx>`)
	}

	b.WriteString(`</c:plotArea>
<c:legend><c:legendPos val="r"/><c:overlay val="0"/></c:legend>
<c:plotVisOnly val="1"/>
</c:chart>
</c:chartSpace>`)
	return b.String()
}

// buildXlsxChartSeries 生成图表系列 XML
func buildXlsxChartSeries(chartType, name, categories, values string) string {
	var b strings.Builder
	switch chartType {
	case "bar":
		b.WriteString(`<c:barChart><c:barDir val="bar"/><c:grouping val="clustered">`)
	case "col":
		b.WriteString(`<c:barChart><c:barDir val="col"/><c:grouping val="clustered">`)
	case "line":
		b.WriteString(`<c:lineChart><c:grouping val="standard">`)
	case "pie":
		b.WriteString(`<c:pieChart>`)
	}

	b.WriteString(fmt.Sprintf(`<c:ser>
<c:idx val="0"/>
<c:order val="0"/>
<c:tx><c:v>%s</c:v></c:tx>
<c:cat><c:strRef><c:f>%s</c:f></c:strRef></c:cat>
<c:val><c:numRef><c:f>%s</c:f></c:numRef></c:val>
</c:ser>`, xmlEscape(name), categories, values))

	switch chartType {
	case "bar", "col":
		b.WriteString(`<c:gapWidth val="100"/>
</c:barChart>`)
	case "line":
		b.WriteString(`</c:lineChart>`)
	case "pie":
		b.WriteString(`</c:pieChart>`)
	}
	return b.String()
}

// buildXlsxDrawingXML 生成 drawing XML，用于放置图表
func buildXlsxDrawingXML(chartIdx int, colFrom, rowFrom int) string {
	colTo := colFrom + 8
	rowTo := rowFrom + 15
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<xdr:twoCellAnchor>
<xdr:from><xdr:col>%d</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from>
<xdr:to><xdr:col>%d</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to>
<xdr:graphicFrame macro="">
<xdr:nvGraphicFramePr><xdr:cNvPr id="2" name="Chart %d"/><xdr:cNvGraphicFramePr/><xdr:nvPr/></xdr:nvGraphicFramePr>
<xdr:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/></xdr:xfrm>
<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/chart"><c:chart r:id="rId1" xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart"/></a:graphicData></a:graphic>
</xdr:graphicFrame>
<xdr:clientData/>
</xdr:twoCellAnchor>
</xdr:wsDr>`, colFrom, rowFrom, colTo, rowTo, chartIdx)
}

// isNumeric 判断字符串是否为数值
func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// 去除千分位逗号
	s = strings.ReplaceAll(s, ",", "")
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func colLetter(i int) string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if i < 26 {
		return string(letters[i])
	}
	return colLetter(i/26-1) + string(letters[i%26])
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func writeZipEntry(w *zip.Writer, name, content string) {
	f, _ := w.Create(name)
	f.Write([]byte(content))
}
