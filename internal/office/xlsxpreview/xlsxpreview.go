// Package xlsxpreview 把 xlsx 提取为结构化单元格视图（值/公式/样式/合并/列宽），
// 供前端渲染「单元格级保真预览」：sheet 切换、公式标识、样式近似还原。
// 数据模型同时是 P0-③ Excel 单元格编辑的底座（选中区域 → 指令 → 写回）。
package xlsxpreview

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// 预览上限：超大表格截断并在 Truncated 标记，避免 IPC 负载失控。
const (
	MaxRows = 2000
	MaxCols = 100
)

// Preview 是整个工作簿的预览视图。
type Preview struct {
	Sheets []Sheet `json:"sheets"`
}

// Sheet 是单个工作表的预览。
type Sheet struct {
	Name      string             `json:"name"`
	Rows      [][]Cell           `json:"rows"`
	Merged    []string           `json:"merged,omitempty"` // "A1:B2"
	ColWidths map[string]float64 `json:"colWidths,omitempty"`
	Freeze    *Freeze            `json:"freeze,omitempty"` // 冻结窗格（表头）
	Truncated bool               `json:"truncated,omitempty"`
}

// Freeze 描述冻结窗格：Row/Col 为从首行/首列起冻结的行列数。
type Freeze struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// Cell 是单元格渲染信息。
type Cell struct {
	Ref     string    `json:"ref"` // "A1"
	Value   string    `json:"value"`
	Formula string    `json:"formula,omitempty"`
	Type    string    `json:"type,omitempty"` // number|string|bool|date|formula|error
	Style   *CellStyle `json:"style,omitempty"`
}

// CellStyle 是简化的单元格样式（供前端近似还原）。
type CellStyle struct {
	Bold      bool   `json:"bold,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Underline bool   `json:"underline,omitempty"`
	Strike    bool   `json:"strike,omitempty"`
	FontColor string `json:"fontColor,omitempty"` // "FF0000"
	Fill      string `json:"fill,omitempty"`      // "FFFF00"
	Align     string `json:"align,omitempty"`     // left|center|right
	Wrap      bool   `json:"wrap,omitempty"`
	NumFmt    string `json:"numFmt,omitempty"`
	Border    bool   `json:"border,omitempty"`
}

// Render 解析 xlsx 为预览 JSON。
func Render(path string) (string, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return "", fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer f.Close()

	pr := Preview{}
	for _, name := range f.GetSheetList() {
		sh, err := renderSheet(f, name)
		if err != nil {
			// 图表/宏表等非工作表跳过，不阻断整体预览
			continue
		}
		pr.Sheets = append(pr.Sheets, sh)
	}
	if len(pr.Sheets) == 0 {
		return "", fmt.Errorf("工作簿中没有可预览的工作表")
	}
	b, err := json.Marshal(pr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// NeedsRecalc 检测工作簿是否存在无缓存值的公式单元格。
// 这类单元格（openpyxl 等直接写公式未重算的文件）需要 LibreOffice
// 重算后才带计算结果，否则预览只能显示 fx 标记。
func NeedsRecalc(path string) (bool, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return false, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer f.Close()
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			continue
		}
		for r := 0; r < len(rows) && r < MaxRows; r++ {
			for c := 0; c < len(rows[r]) && c < MaxCols; c++ {
				axis, err := excelize.CoordinatesToCellName(c+1, r+1)
				if err != nil {
					continue
				}
				formula, _ := f.GetCellFormula(name, axis)
				if formula == "" {
					continue
				}
				v, _ := f.GetCellValue(name, axis)
				if v == "" {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func renderSheet(f *excelize.File, name string) (Sheet, error) {
	sh := Sheet{Name: name, ColWidths: map[string]float64{}}

	rows, err := f.GetRows(name)
	if err != nil {
		return sh, err
	}
	rowCount := len(rows)
	colCount := 0
	for _, r := range rows {
		if len(r) > colCount {
			colCount = len(r)
		}
	}
	if rowCount > MaxRows {
		rowCount = MaxRows
		sh.Truncated = true
	}
	if colCount > MaxCols {
		colCount = MaxCols
		sh.Truncated = true
	}

	styleCache := map[int]*CellStyle{}
	getStyle := func(styleID int) *CellStyle {
		if s, ok := styleCache[styleID]; ok {
			return s
		}
		st, err := f.GetStyle(styleID)
		if err != nil || st == nil {
			styleCache[styleID] = nil
			return nil
		}
		cs := &CellStyle{}
		if st.Font != nil {
			cs.Bold = st.Font.Bold
			cs.Italic = st.Font.Italic
			cs.Underline = st.Font.Underline != ""
			cs.Strike = st.Font.Strike
			cs.FontColor = normalizeColor(st.Font.Color)
		}
		if len(st.Fill.Color) > 0 {
			cs.Fill = normalizeColor(st.Fill.Color[0])
		}
		if st.Alignment != nil {
			cs.Align = st.Alignment.Horizontal
			cs.Wrap = st.Alignment.WrapText
		}
		if st.CustomNumFmt != nil && *st.CustomNumFmt != "" {
			cs.NumFmt = *st.CustomNumFmt
		} else if st.NumFmt != 0 {
			cs.NumFmt = strconv.Itoa(st.NumFmt)
		}
		if len(st.Border) > 0 {
			cs.Border = true
		}
		styleCache[styleID] = cs
		return cs
	}

	for r := 0; r < rowCount; r++ {
		rowCells := []Cell{}
		row := rows[r]
		for c := 0; c < colCount && c < len(row); c++ {
			val := row[c]
			axis, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				continue
			}
			formula, _ := f.GetCellFormula(name, axis)
			if val == "" && formula == "" {
				continue
			}
			cell := Cell{Ref: axis, Value: val, Formula: formula}
			if t, err := f.GetCellType(name, axis); err == nil {
				cell.Type = cellTypeName(t)
			}
			if styleID, err := f.GetCellStyle(name, axis); err == nil && styleID > 0 {
				if cs := getStyle(styleID); cs != nil && hasStyle(*cs) {
					cell.Style = cs
				}
			}
			rowCells = append(rowCells, cell)
		}
		sh.Rows = append(sh.Rows, rowCells)
	}

	if merges, err := f.GetMergeCells(name); err == nil {
		for _, m := range merges {
			if len(m) > 0 {
				sh.Merged = append(sh.Merged, m[0])
			}
		}
	}
	if p, err := f.GetPanes(name); err == nil && (p.YSplit > 0 || p.XSplit > 0) {
		if p.YSplit > 0 || p.XSplit > 0 {
			sh.Freeze = &Freeze{Row: p.YSplit, Col: p.XSplit}
		}
	}
	for c := 1; c <= colCount; c++ {
		col, err := excelize.ColumnNumberToName(c)
		if err != nil {
			continue
		}
		if w, err := f.GetColWidth(name, col); err == nil {
			sh.ColWidths[col] = w
		}
	}
	return sh, nil
}

func cellTypeName(t excelize.CellType) string {
	switch t {
	case excelize.CellTypeNumber:
		return "number"
	case excelize.CellTypeBool:
		return "bool"
	case excelize.CellTypeDate:
		return "date"
	case excelize.CellTypeError:
		return "error"
	case excelize.CellTypeFormula, excelize.CellTypeSharedString, excelize.CellTypeInlineString:
		return "string"
	default:
		return ""
	}
}

// normalizeColor 把 ARGB 色值截断为 RRGGBB（去透明度前缀）。
func normalizeColor(c string) string {
	if len(c) == 8 {
		return c[2:]
	}
	return c
}

func hasStyle(cs CellStyle) bool {
	return cs.Bold || cs.Italic || cs.Underline || cs.Strike || cs.FontColor != "" ||
		cs.Fill != "" || cs.Align != "" || cs.Wrap || cs.NumFmt != "" || cs.Border
}
