package docmd

// xlsx_inspect.go — 核对 wpscli 产出的 xlsx 并统计回执要的数（见 wps_excel.go）。
//
// 直接流式读工作表 XML，不走 excelize：逐格调 GetCellType 是平方级开销（839 页、18 万
// 合并区的工作簿实测 5 分半），而且行迭代进行中它查不到格子、一律报 Unset，会把文本格
// 当成数值（2026-11-26 实弹：4 页里 149 个文本格被数成了数值）。

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	// xlsxMaskCols 行掩码能表示的最大列数。更宽的工作表照常计数，只是不参与疑似缺格判断。
	xlsxMaskCols = 64
	// xlsxSuspectSamples 回执里最多列出几条疑似缺格的行位置。
	xlsxSuspectSamples = 8
	// xlsxRelNS workbook.xml 里 r:id 所在的命名空间。
	xlsxRelNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
)

// xlsxNumberTextRe 整格就是一个数字的文本（定额编码 040205017001、多级序号 1.1.1）。
var xlsxNumberTextRe = regexp.MustCompile(`^[+-]?\d+(?:[.,]\d+)*$`)

// xlsxInspect 打开产物做核对与统计。一个非空格子都没有视为转换失败——wpscli 的
// 「success」不可信（pdf2md 实测报成功却产出 0 字节）。
func xlsxInspect(path string) (ExcelReceipt, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return ExcelReceipt{}, fmt.Errorf("WPS 产出的 Excel 打不开（产物损坏）: %w", err)
	}
	defer func() { _ = r.Close() }()
	sheets := xlsxSheets(&r.Reader)
	if len(sheets) == 0 {
		return ExcelReceipt{}, errors.New("WPS 产出的 Excel 打不开（产物损坏）: 工作簿里没有工作表")
	}
	sst, err := xlsxSharedStrings(&r.Reader)
	if err != nil {
		return ExcelReceipt{}, fmt.Errorf("WPS 产出的 Excel 打不开（产物损坏）: %w", err)
	}
	var rec ExcelReceipt
	for _, sheet := range sheets {
		found, err := xlsxReadEntry(&r.Reader, sheet.part, func(rc io.Reader) error {
			return xlsxInspectSheet(rc, sst, sheet.name, &rec)
		})
		if err != nil {
			return ExcelReceipt{}, fmt.Errorf("读取工作表 %s: %w", sheet.name, err)
		}
		if !found {
			return ExcelReceipt{}, fmt.Errorf("WPS 产出的 Excel 缺少工作表 %s 的数据（%s）", sheet.name, sheet.part)
		}
	}
	if rec.Cells == 0 {
		return ExcelReceipt{}, errors.New("WPS 转换结果没有内容（产物里一个非空格子都没有）")
	}
	return rec, nil
}

// ── 工作簿结构 ───────────────────────────────────────────────────────

// xlsxSheet 工作簿里的一张工作表。
type xlsxSheet struct {
	name string
	part string // zip 条目名，如 xl/worksheets/sheet1.xml
}

// xlsxSheets 按工作簿里的顺序列出工作表（workbook.xml 的 <sheet> 经 rels 找到数据部件）。
// 读不到时退回按部件名自然序列出 xl/worksheets/*.xml。
func xlsxSheets(r *zip.Reader) []xlsxSheet {
	rels := map[string]string{}
	_, _ = xlsxReadEntry(r, "xl/_rels/workbook.xml.rels", func(rc io.Reader) error {
		return xlsxWalk(rc, func(se xml.StartElement) {
			if se.Name.Local == "Relationship" {
				rels[xlsxAttr(se, "", "Id")] = xlsxAttr(se, "", "Target")
			}
		}, nil, nil)
	})
	var sheets []xlsxSheet
	_, _ = xlsxReadEntry(r, "xl/workbook.xml", func(rc io.Reader) error {
		return xlsxWalk(rc, func(se xml.StartElement) {
			if se.Name.Local != "sheet" {
				return
			}
			target := rels[xlsxAttr(se, xlsxRelNS, "id")]
			if target == "" {
				return
			}
			target = strings.TrimPrefix(target, "/") // 绝对写法 /xl/… 与相对 xl/ 的写法都认
			if !strings.HasPrefix(target, "xl/") {
				target = "xl/" + target
			}
			sheets = append(sheets, xlsxSheet{name: xlsxAttr(se, "", "name"), part: path.Clean(target)})
		}, nil, nil)
	})
	if len(sheets) > 0 {
		return sheets
	}
	for _, f := range r.File {
		if d, base := path.Split(f.Name); d == "xl/worksheets/" && strings.HasSuffix(base, ".xml") {
			sheets = append(sheets, xlsxSheet{name: strings.TrimSuffix(base, ".xml"), part: f.Name})
		}
	}
	sort.SliceStable(sheets, func(i, j int) bool { return naturalLess(sheets[i].part, sheets[j].part) })
	return sheets
}

// xlsxSharedStrings 读共享字符串表（下标 = 单元格里 <v> 的值）。富文本把各段 <t> 拼起来；
// 注音 <rPh> 里的 <t> 不算。
func xlsxSharedStrings(r *zip.Reader) ([]string, error) {
	var sst []string
	found, err := xlsxReadEntry(r, "xl/sharedStrings.xml", func(rc io.Reader) error {
		var cur strings.Builder
		inSI, inT, inPh := false, false, false
		return xlsxWalk(rc, func(se xml.StartElement) {
			switch se.Name.Local {
			case "si":
				inSI, cur = true, strings.Builder{}
			case "rPh":
				inPh = true
			case "t":
				inT = inSI && !inPh
			}
		}, func(s string) {
			if inT {
				cur.WriteString(s)
			}
		}, func(ee xml.EndElement) {
			switch ee.Name.Local {
			case "t":
				inT = false
			case "rPh":
				inPh = false
			case "si":
				inSI = false
				sst = append(sst, cur.String())
			}
		})
	})
	if !found || err != nil {
		return nil, err
	}
	return sst, nil
}

// ── 工作表 ───────────────────────────────────────────────────────────

// xlsxRowMask 一行里哪些列有值、哪些列是数字（按位，第 c 列=第 c 位）。
type xlsxRowMask struct{ nonEmpty, numberLike uint64 }

// 格子内容的类型（按 <c> 的 t 属性与 <v> 判）。
const (
	xlsxCellEmpty = iota
	xlsxCellNumber
	xlsxCellText
)

// xlsxInspectSheet 流式扫一遍工作表：数行、列、合并单元格、非空格子与数字格，并记下每行
// 的列掩码供疑似缺格判断。
func xlsxInspectSheet(rc io.Reader, sst []string, name string, rec *ExcelReceipt) error {
	rec.Sheets++
	var (
		masks                             []xlsxRowMask
		cur                               xlsxRowMask
		rowNum, maxRow, maxCol, colSeq    int
		cellCol                           int
		cellT, cellV                      string
		inCell, inValue, inInline, inText bool
		inline                            strings.Builder
	)
	err := xlsxWalk(rc, func(se xml.StartElement) {
		switch se.Name.Local {
		case "row":
			if n, _ := strconv.Atoi(xlsxAttr(se, "", "r")); n > 0 {
				rowNum = n
			} else {
				rowNum++
			}
			if rowNum > maxRow {
				maxRow = rowNum
			}
			cur, colSeq = xlsxRowMask{}, 0
		case "c":
			inCell = true
			cellT, cellV = xlsxAttr(se, "", "t"), ""
			if cellCol = xlsxColIndex(xlsxAttr(se, "", "r")); cellCol == 0 { // 没有 r 属性：按行内顺序
				colSeq++
				cellCol = colSeq
			} else {
				colSeq = cellCol
			}
		case "v":
			inValue = inCell
		case "is":
			if inCell {
				inInline = true
				inline.Reset()
			}
		case "t":
			inText = inInline
		case "mergeCell":
			rec.Merged++
		}
	}, func(s string) {
		switch {
		case inValue:
			cellV += s
		case inText:
			inline.WriteString(s)
		}
	}, func(ee xml.EndElement) {
		switch ee.Name.Local {
		case "v":
			inValue = false
		case "t":
			inText = false
		case "is":
			inInline = false
			cellV = inline.String() // 内联字符串的值在 </is> 就取定，</c> 时标志已不清
		case "c":
			if !inCell {
				return
			}
			inCell = false
			xlsxCountCell(xlsxCell{col: cellCol, t: cellT, v: cellV}, sst, rec, &cur, &maxCol)
		case "row":
			masks = append(masks, cur)
		}
	})
	if err != nil {
		return err
	}
	rec.Rows += maxRow
	if maxCol > rec.MaxCols {
		rec.MaxCols = maxCol
	}
	for _, n := range xlsxSuspectRows(masks) {
		rec.SuspectCount++
		if len(rec.Suspects) < xlsxSuspectSamples {
			rec.Suspects = append(rec.Suspects, name+"!"+strconv.Itoa(n))
		}
	}
	return nil
}

// xlsxCell 一个格子的原始信息。
type xlsxCell struct {
	col int    // 1 起的列号
	t   string // <c> 的 t 属性
	v   string // <v> 原始值；inlineStr 时是 <is> 里各段 <t> 的拼接
}

// xlsxCountCell 把一个格子记进统计与行掩码。空白格（只有换行 / 空格的也算）不算非空。
func xlsxCountCell(cell xlsxCell, sst []string, rec *ExcelReceipt, row *xlsxRowMask, maxCol *int) {
	value, kind := xlsxCellValue(cell, sst)
	if kind == xlsxCellEmpty || strings.TrimSpace(value) == "" {
		return
	}
	rec.Cells++
	if cell.col > *maxCol {
		*maxCol = cell.col
	}
	numberLike := kind == xlsxCellNumber
	if kind == xlsxCellNumber {
		rec.Numbers++
	} else if xlsxNumberTextRe.MatchString(strings.TrimSpace(value)) {
		numberLike = true
		rec.NumberText++
	}
	if cell.col-1 >= xlsxMaskCols {
		return
	}
	bit := uint64(1) << (cell.col - 1)
	row.nonEmpty |= bit
	if numberLike {
		row.numberLike |= bit
	}
}

// xlsxCellValue 还原格子的显示值与类型。布尔 / 错误值不参与统计（当空格）。
func xlsxCellValue(cell xlsxCell, sst []string) (string, int) {
	switch cell.t {
	case "s": // 共享字符串：<v> 是下标
		i, err := strconv.Atoi(strings.TrimSpace(cell.v))
		if err != nil || i < 0 || i >= len(sst) {
			return "", xlsxCellEmpty
		}
		return sst[i], xlsxCellText
	case "inlineStr", "str":
		return cell.v, xlsxCellText
	case "b", "e":
		return "", xlsxCellEmpty
	}
	// 没写 t（OOXML 的默认类型就是数值）/ n / d
	return cell.v, xlsxCellNumber
}

// xlsxSuspectRows 找出「疑似缺格」的数据行，返回它们的行号。
//
// 2026-11-26 实测：WPS 在线识别（photo2excel 同）会把工程量为「1」的格子整格漏掉，
// 而且没有别的信号可依。口径：主键列=数字格最多的一列（并列取最左；清单表里就是序号
// 或编码列）；数据行=主键列里是数字的行（表题、表头、合计行这些文字行不算）；数据行
// 里填充率过半的其余数字列，哪个数据行缺了它，这一行就疑似缺格。数据行不足 3 行、或
// 除主键列外没有别的数字列时不判。
func xlsxSuspectRows(masks []xlsxRowMask) []int {
	width := 0
	for _, m := range masks {
		for c := width; c < xlsxMaskCols; c++ {
			if m.nonEmpty&(uint64(1)<<c) != 0 {
				width = c + 1
			}
		}
	}
	var numCols [xlsxMaskCols]int
	for _, m := range masks {
		for c := 0; c < width; c++ {
			if m.numberLike&(uint64(1)<<c) != 0 {
				numCols[c]++
			}
		}
	}
	key, keyCount := -1, 1 // 至少 2 个数字格才算数字列
	for c := 0; c < width; c++ {
		if numCols[c] > keyCount {
			key, keyCount = c, numCols[c]
		}
	}
	if key < 0 {
		return nil
	}
	keyBit := uint64(1) << key
	var data []xlsxRowMask
	var nums []int // 与 data 对齐的行号
	for i, m := range masks {
		if m.numberLike&keyBit != 0 {
			data, nums = append(data, m), append(nums, i+1)
		}
	}
	if len(data) < 3 {
		return nil
	}
	var filled [xlsxMaskCols]int
	for _, m := range data {
		for c := 0; c < width; c++ {
			if c != key && m.numberLike&(uint64(1)<<c) != 0 {
				filled[c]++
			}
		}
	}
	var out []int
	for i, m := range data {
		for c := 0; c < width; c++ {
			// 至少 3 行、且过半数据行有值：稀稀拉拉的可选数字列不算
			if c != key && filled[c] >= 3 && filled[c]*2 >= len(data) && m.nonEmpty&(uint64(1)<<c) == 0 {
				out = append(out, nums[i])
				break
			}
		}
	}
	return out
}

// ── 小工具 ───────────────────────────────────────────────────────────

// xlsxEntry 按名找 zip 条目。
func xlsxEntry(r *zip.Reader, name string) *zip.File {
	for _, f := range r.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// xlsxReadEntry 读一个 zip 条目并交给 fn；条目不存在时 found=false。
func xlsxReadEntry(r *zip.Reader, name string, fn func(io.Reader) error) (found bool, err error) {
	f := xlsxEntry(r, name)
	if f == nil {
		return false, nil
	}
	rc, err := f.Open()
	if err != nil {
		return true, err
	}
	err = fn(rc)
	if closeErr := rc.Close(); err == nil {
		err = closeErr
	}
	return true, err
}

// xlsxWalk 用 xml.Decoder 流式扫完 rc，逐令牌调回调（nil 表示不关心）。
func xlsxWalk(rc io.Reader, onStart func(xml.StartElement), onText func(string), onEnd func(xml.EndElement)) error {
	dec := xml.NewDecoder(rc)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if onStart != nil {
				onStart(t)
			}
		case xml.CharData:
			if onText != nil {
				onText(string(t))
			}
		case xml.EndElement:
			if onEnd != nil {
				onEnd(t)
			}
		}
	}
}

// xlsxAttr 取属性值；space 非空时要求命名空间也匹配。
func xlsxAttr(se xml.StartElement, space, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local && (space == "" || a.Name.Space == space) {
			return a.Value
		}
	}
	return ""
}

// xlsxColIndex 单元格坐标里的列号（"AB12" → 28）；没有列字母时返回 0。
func xlsxColIndex(coord string) int {
	n := 0
	for i := 0; i < len(coord); i++ {
		c := coord[i]
		switch {
		case c >= 'A' && c <= 'Z':
			n = n*26 + int(c-'A'+1)
		case c >= 'a' && c <= 'z':
			n = n*26 + int(c-'a'+1)
		default:
			return n
		}
	}
	return n
}
