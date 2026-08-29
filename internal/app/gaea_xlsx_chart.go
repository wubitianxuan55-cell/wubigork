package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// XlsxChartInput 是「表格选中区域 → 一键图表」入参。
type XlsxChartInput struct {
	Rel       string `json:"rel"`       // xlsx 工作区相对路径
	Sheet     string `json:"sheet"`     // 工作表名；空 = 第一个工作表
	Refs      string `json:"refs"`      // 区域 "A1:B6" 或单单元格 "B2"；空 = 自动取前两列数据行
	ChartType string `json:"chartType"` // bar | line | pie | scatter
	Title     string `json:"title"`     // 图表标题；空 = 文件名
}

// XlsxChartResult 是图表产物：原生图表对象已嵌入工作簿（Excel/WPS 打开即可见、
// 可继续编辑），并带回数据供前端渲染迷你预览。
type XlsxChartResult struct {
	Path      string    `json:"path"`      // xlsx 工作区相对路径（图表已嵌入该文件）
	Name      string    `json:"name"`      // 文件名
	Sheet     string    `json:"sheet"`     // 嵌入的工作表
	Anchor    string    `json:"anchor"`    // 图表左上角锚点单元格（如 D1）
	Labels    int       `json:"labels"`    // 数据点数量
	LabelList []string  `json:"labelList"` // 类别（迷你图预览）
	Values    []float64 `json:"values"`    // 数值（迷你图预览）
	ChartType string    `json:"chartType"`
	Title     string    `json:"title"`
}

// GaeaXlsxChart 表格「选中区域 → 一键图表」：从 xlsx 的选中区域提取
// 「标签列 + 数值列」→ excelize 在工作簿内嵌入原生图表对象（Excel 原生可编辑，
// 非图片截图）→ 返回锚点与数据供前端迷你预览。
// 对标 Gemini in Sheets / Copilot in Excel 的「原生对象而非截图」交付标准。
func (a *App) GaeaXlsxChart(in XlsxChartInput) (XlsxChartResult, error) {
	if in.Rel == "" {
		return XlsxChartResult{}, fmt.Errorf("缺少 xlsx 文件路径")
	}
	xlsxPath := in.Rel
	if !filepath.IsAbs(in.Rel) {
		xlsxPath = filepath.Join(gaeaCwd(), in.Rel)
	}
	if _, err := os.Stat(xlsxPath); err != nil {
		return XlsxChartResult{}, fmt.Errorf("xlsx 不存在：%s", in.Rel)
	}
	if in.ChartType == "" {
		in.ChartType = "bar"
	}
	switch in.ChartType {
	case "bar", "line", "pie", "scatter":
	default:
		return XlsxChartResult{}, fmt.Errorf("不支持的图表类型 %q（bar/line/pie/scatter）", in.ChartType)
	}

	region, labels, values, err := extractChartRegion(xlsxPath, in.Sheet, in.Refs)
	if err != nil {
		return XlsxChartResult{}, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(in.Rel), filepath.Ext(in.Rel))
	}

	anchor, err := embedNativeChart(xlsxPath, region, in.ChartType, title)
	if err != nil {
		return XlsxChartResult{}, err
	}
	rel, _ := filepath.Rel(gaeaCwd(), xlsxPath)
	return XlsxChartResult{
		Path:      filepath.ToSlash(rel),
		Name:      filepath.Base(xlsxPath),
		Sheet:     region.Sheet,
		Anchor:    anchor,
		Labels:    len(labels),
		LabelList: labels,
		Values:    values,
		ChartType: in.ChartType,
		Title:     title,
	}, nil
}

// chartRegion 描述图表数据在工作表中的位置（生成原生图表的系列引用用）。
type chartRegion struct {
	Sheet    string
	LabelCol int // 0 = 无标签列（单列数据）
	ValueCol int
	FirstRow int // 首个数据行（跳过表头后）
	LastRow  int
}

// extractChartRegion 从 xlsx 选中区域提取「标签列 + 数值列」的位置与数据。
// refs 规则：空 = 自动取前两列数据行（首行作标签列头则跳过表头）；
// "A1:B6" = 显式区域，取区域第一列为标签、第二列为数值；
// 单单元格 "B2" = 从 A1 到该单元格的矩形（表头到选中行）。
func extractChartRegion(path, sheet, refs string) (region chartRegion, labels []string, values []float64, err error) {
	region = chartRegion{LabelCol: 1, ValueCol: 2, FirstRow: 2}
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return region, nil, nil, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer f.Close()
	if sheet == "" {
		if list := f.GetSheetList(); len(list) > 0 {
			sheet = list[0]
		}
	}
	region.Sheet = sheet
	all, err := f.GetRows(sheet)
	if err != nil {
		return region, nil, nil, fmt.Errorf("读取工作表 %q 失败: %w", sheet, err)
	}
	if len(all) == 0 {
		return region, nil, nil, fmt.Errorf("工作表 %q 为空", sheet)
	}

	r1, c1, r2, c2 := 1, 1, len(all), 2 // 默认：全表前两列
	if refs != "" {
		var parseErr error
		if strings.Contains(refs, ":") {
			r1, c1, r2, c2, parseErr = parseChartRange(refs)
		} else {
			// 单单元格 → A1 到该单元格（表头到选中行），标签列 A、数值列取选中列
			col, row, cerr := excelize.CellNameToCoordinates(strings.ToUpper(refs))
			if cerr != nil {
				return region, nil, nil, fmt.Errorf("无效单元格引用：%s", refs)
			}
			r1, c1, r2, c2 = 1, 1, row, col
		}
		if parseErr != nil {
			return region, nil, nil, parseErr
		}
		if r2 > len(all) {
			r2 = len(all)
		}
		if r2 < r1 || c2 < c1 {
			return region, nil, nil, fmt.Errorf("数据区域无效：%s", refs)
		}
	}
	// 数值列取区域第二列；区域只有一列时用第一列（单列数据默认无标签）
	labelCol, valueCol := c1, c1+1
	if c2 == c1 {
		labelCol, valueCol = 0, c1 // 无标签列，数值列即区域唯一列
	}
	region.LabelCol, region.ValueCol = labelCol, valueCol
	slice := func(row []string, c int) string {
		if c <= 0 || c-1 >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[c-1])
	}

	firstDataRow := r1
	if refs == "" || !strings.Contains(refs, ":") && r1 == 1 {
		// 自动模式或单单元格从 A1 开始：首行视为表头（标签列头），跳过
		firstDataRow = r1 + 1
	}
	region.FirstRow, region.LastRow = firstDataRow, r2
	rows := all
	for i := firstDataRow; i <= r2 && i-1 < len(rows); i++ {
		row := rows[i-1]
		lbl := slice(row, labelCol)
		if lbl == "" {
			lbl = strconv.Itoa(len(labels) + 1)
		}
		v := 0.0
		clean := strings.ReplaceAll(strings.ReplaceAll(slice(row, valueCol), ",", ""), " ", "")
		if parsed, e := strconv.ParseFloat(clean, 64); e == nil {
			v = parsed
		}
		labels = append(labels, lbl)
		values = append(values, v)
	}
	if len(values) == 0 {
		return region, nil, nil, fmt.Errorf("数据区域没有数值（请选中含数据的区域）")
	}
	return region, labels, values, nil
}

// embedNativeChart 用 excelize 在工作簿内嵌入原生图表对象并保存，
// 返回锚点单元格。图表锚定在数值列右侧两列、数据区首行处（浮于空白，
// 不遮挡数据），在 Excel/WPS 中可继续拖拽与编辑。
func embedNativeChart(path string, region chartRegion, chartType, title string) (string, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return "", fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer f.Close()

	typeByName := map[string]excelize.ChartType{
		"bar":     excelize.Col,
		"line":    excelize.Line,
		"pie":     excelize.Pie,
		"scatter": excelize.Scatter,
	}
	series := excelize.ChartSeries{Name: title}
	if region.LabelCol > 0 {
		labelColName, err := excelize.ColumnNumberToName(region.LabelCol)
		if err != nil {
			return "", fmt.Errorf("标签列无效：%w", err)
		}
		series.Categories = seriesRef(region.Sheet, labelColName, region.FirstRow, region.LastRow)
	}
	valueColName, err := excelize.ColumnNumberToName(region.ValueCol)
	if err != nil {
		return "", fmt.Errorf("数值列无效：%w", err)
	}
	series.Values = seriesRef(region.Sheet, valueColName, region.FirstRow, region.LastRow)

	chart := &excelize.Chart{
		Type:      typeByName[chartType],
		Series:    []excelize.ChartSeries{series},
		Title:     excelize.ChartTitle{Paragraph: []excelize.RichTextRun{{Text: title}}},
		Legend:    excelize.ChartLegend{Position: "bottom"},
		Dimension: excelize.ChartDimension{Width: 520, Height: 300},
		PlotArea:  excelize.ChartPlotArea{ShowPercent: chartType == "pie"},
	}
	anchorCol, err := excelize.ColumnNumberToName(region.ValueCol + 2)
	if err != nil {
		return "", fmt.Errorf("图表锚点列无效：%w", err)
	}
	anchor := anchorCol + strconv.Itoa(max(1, region.FirstRow-1))
	if err := f.AddChart(region.Sheet, anchor, chart); err != nil {
		return "", fmt.Errorf("嵌入图表失败: %w", err)
	}
	if err := f.Save(); err != nil {
		return "", fmt.Errorf("保存 xlsx 失败: %w", err)
	}
	return anchor, nil
}

// seriesRef 生成图表系列的绝对区域引用，如 "Sheet1!$B$2:$B$6"。
func seriesRef(sheet, col string, firstRow, lastRow int) string {
	return fmt.Sprintf("%s!$%s$%d:$%s$%d", sheet, col, firstRow, col, lastRow)
}

// parseChartRange 解析 "A1:B6" 区域为行列坐标。
func parseChartRange(r string) (r1, c1, r2, c2 int, err error) {
	parts := strings.Split(strings.ToUpper(strings.ReplaceAll(r, " ", "")), ":")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("区域格式应为 A1:B6，得到 %q", r)
	}
	c1, r1, err = excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("区域起点无效 %q", parts[0])
	}
	c2, r2, err = excelize.CellNameToCoordinates(parts[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("区域终点无效 %q", parts[1])
	}
	return r1, c1, r2, c2, nil
}
