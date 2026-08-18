package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/office/crosslink"
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

// XlsxChartResult 是图表产物：PNG 相对路径 + dataURL（前端直接预览）。
type XlsxChartResult struct {
	Path      string `json:"path"`      // 工作区相对路径
	Name      string `json:"name"`      // 文件名
	DataURL   string `json:"dataUrl"`   // base64 PNG dataURL
	Labels    int    `json:"labels"`    // 数据点数量
	ChartType string `json:"chartType"`
}

// GaeaXlsxChart 表格「选中区域 → 一键图表」：从 xlsx 的选中区域提取
// 「标签列 + 数值列」→ matplotlib 生成 PNG → 返回可预览的 dataURL。
// 对标千问表格 Agent / ChatExcel 的「可交付」表格体验：选中数据即出图，
// 不离开预览，产物落 .gaea/exports/ 供后续引用。
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

	labels, values, err := extractRangeChartData(xlsxPath, in.Sheet, in.Refs)
	if err != nil {
		return XlsxChartResult{}, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(in.Rel), filepath.Ext(in.Rel))
	}

	exportsDir := filepath.Join(gaeaCwd(), ".gaea", "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return XlsxChartResult{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	base := safeDeliverableName(title)
	pngPath := filepath.Join(exportsDir, base+"-chart-"+stamp+".png")
	if err := crosslink.GenerateChartPNG(labels, values, in.ChartType, title, pngPath); err != nil {
		return XlsxChartResult{}, err
	}

	raw, err := os.ReadFile(pngPath)
	if err != nil {
		return XlsxChartResult{}, err
	}
	rel, _ := filepath.Rel(gaeaCwd(), pngPath)
	return XlsxChartResult{
		Path:      filepath.ToSlash(rel),
		Name:      filepath.Base(pngPath),
		DataURL:   "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw),
		Labels:    len(labels),
		ChartType: in.ChartType,
	}, nil
}

// extractRangeChartData 从 xlsx 选中区域提取「标签列 + 数值列」。
// refs 规则：空 = 自动取前两列数据行（首行作标签列头则跳过表头）；
// "A1:B6" = 显式区域，取区域第一列为标签、第二列为数值；
// 单单元格 "B2" = 从 A1 到该单元格的矩形（表头到选中行）。
func extractRangeChartData(path, sheet, refs string) (labels []string, values []float64, err error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, nil, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer f.Close()
	if sheet == "" {
		if list := f.GetSheetList(); len(list) > 0 {
			sheet = list[0]
		}
	}
	all, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("读取工作表 %q 失败: %w", sheet, err)
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("工作表 %q 为空", sheet)
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
				return nil, nil, fmt.Errorf("无效单元格引用：%s", refs)
			}
			r1, c1, r2, c2 = 1, 1, row, col
		}
		if parseErr != nil {
			return nil, nil, parseErr
		}
		if r2 > len(all) {
			r2 = len(all)
		}
		if r2 < r1 || c2 < c1 {
			return nil, nil, fmt.Errorf("数据区域无效：%s", refs)
		}
	}
	// 数值列取区域第二列；区域只有一列时用第一列（单列数据默认无标签）
	labelCol, valueCol := c1, c1+1
	if c2 == c1 {
		labelCol, valueCol = 0, c1 // 无标签列，数值列即区域唯一列
	}
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
		return nil, nil, fmt.Errorf("数据区域没有数值（请选中含数据的区域）")
	}
	return labels, values, nil
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
