package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(xlsxRead{}) }

type xlsxRead struct{}

func (xlsxRead) Name() string { return "xlsx_read" }

func (xlsxRead) Description() string {
	return "读取 Excel (.xlsx) 文件内容，返回表格数据（JSON格式，含表头和数据行）。支持读取第一个工作表。"
}

func (xlsxRead) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"xlsx文件路径"},
  "all_sheets":{"type":"boolean","description":"是否返回所有工作表"}, 
  "sheet_index":{"type":"integer","description":"工作表索引（从0开始）","default":0}
},
"required":["path"]
}`)
}

func (xlsxRead) ReadOnly() bool { return true }

func (xlsxRead) CompactDescription() string     { return compactDesc["xlsx_read"] }
func (xlsxRead) CompactSchema() json.RawMessage { return compactSchema["xlsx_read"] }

// xlsx内部XML结构（只取需要的部分）
type xlsxSharedStrings struct {
	Items []xlsxSI `xml:"si"`
}
type xlsxSI struct {
	Text string `xml:"t"`
}

type xlsxSheetData struct {
	Rows []xlsxRow `xml:"row"`
}
type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}
type xlsxCell struct {
	Reference string `xml:"r,attr"`
	Type      string `xml:"t,attr"` // s=shared string
	Value     string `xml:"v"`
}

type xlsxWorkbook struct {
	Sheets []xlsxSheetRef `xml:"sheets>sheet"`
}
type xlsxSheetRef struct {
	Name string `xml:"name,attr"`
}

type xlsxResult struct {
	SheetName string     `json:"sheet_name"`
	Headers   []string   `json:"headers,omitempty"`
	Rows      [][]string `json:"rows"`
	RowCount  int        `json:"row_count"`
	ColCount  int        `json:"col_count"`
}

func (xlsxRead) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path       string `json:"path"`
		SheetIndex int    `json:"sheet_index"`
		AllSheets  bool   `json:"all_sheets,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path 不能为空")
	}

	data, err := readZipFile(p.Path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 解析 sharedStrings
	ssContent, err := readFileFromZip(data, "xl/sharedStrings.xml")
	ssMap := make(map[int]string)
	if err == nil {
		var ss xlsxSharedStrings
		if xml.Unmarshal(ssContent, &ss) == nil {
			for i, si := range ss.Items {
				ssMap[i] = si.Text
			}
		}
	}

	// 获取 sheet 名称
	sheetName := fmt.Sprintf("Sheet%d", p.SheetIndex+1)
	wbContent, err := readFileFromZip(data, "xl/workbook.xml")
	if err == nil {
		var wb xlsxWorkbook
		if xml.Unmarshal(wbContent, &wb) == nil {
			if p.SheetIndex >= 0 && p.SheetIndex < len(wb.Sheets) {
				sheetName = wb.Sheets[p.SheetIndex].Name
			}
		}
	}

	// 读取 sheet 数据
	sheetFile := fmt.Sprintf("xl/worksheets/sheet%d.xml", p.SheetIndex+1)
	sheetContent, err := readFileFromZip(data, sheetFile)
	if err != nil {
		return "", fmt.Errorf("读取工作表 %d 失败（文件内可能没有该sheet）: %w", p.SheetIndex, err)
	}

	var sd xlsxSheetData
	if err := xml.Unmarshal(sheetContent, &sd); err != nil {
		return "", fmt.Errorf("解析sheet XML失败: %w", err)
	}

	// 提取数据
	var rows [][]string
	maxCols := 0
	for _, row := range sd.Rows {
		var rowData []string
		// 按列引用排序（A1, B1, C1...）
		cells := make(map[string]string)
		colOrder := []string{}
		for _, cell := range row.Cells {
			colRef := extractColRef(cell.Reference)
			if colRef == "" {
				continue
			}
			colOrder = append(colOrder, colRef)
			val := cell.Value
			if cell.Type == "s" {
				// shared string
				var idx int
				var ssi int
				if _, serr := fmt.Sscanf(val, "%d", &ssi); serr == nil {
					idx = ssi
					if s, ok := ssMap[idx]; ok {
						val = s
					}
				}
			}
			cells[colRef] = val
		}
		// 按列字母顺序输出
		cols := sortColRefs(colOrder)
		for _, c := range cols {
			rowData = append(rowData, cells[c])
		}
		if len(rowData) > maxCols {
			maxCols = len(rowData)
		}
		rows = append(rows, rowData)
	}

	res := xlsxResult{
		SheetName: sheetName,
		Rows:      rows,
		RowCount:  len(rows),
		ColCount:  maxCols,
	}
	if len(rows) > 0 {
		res.Headers = rows[0]
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	return string(out), nil
}

func readZipFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func readFileFromZip(data []byte, name string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		if strings.EqualFold(f.Name, name) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("文件 %s 未找到", name)
}

func extractColRef(ref string) string {
	col := ""
	for _, c := range ref {
		if c >= 'A' && c <= 'Z' {
			col += string(c)
		} else {
			break
		}
	}
	return col
}

func sortColRefs(refs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range refs {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	// 简单冒泡按字母序
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
