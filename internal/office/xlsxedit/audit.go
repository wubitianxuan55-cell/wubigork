package xlsxedit

// audit.go — xlsx 只读审计引擎（灵犀 optimize plan P1-2，docs/gaea-lingxi-
// optimize-plan-2026-11.md）：错误值扫描 + 循环引用最小面 + 未重算清单。
//
// 定位：xlsx_apply 应用后已有自动重算；本引擎给「交付门」提供复核依据——
// 交付前公式错误必须为零。深循环（A→B→A 多跳）交给 LibreOffice 重算链
// （Recalc 的 recalc.py 会报 Err:522），这里只做无需外部进程的结构面扫描。

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// AuditMaxCells 是单文件审计的单元格上限（防超大表拖死 agent 回合）。
// 超限时报告如实标注 truncated，而不是静默扫完。
const AuditMaxCells = 200000

// AuditFinding 是一条审计发现。Kind 取值：
//   - error_value   缓存值是错误（#REF!/#DIV/0!/#VALUE!/#N/A/#NAME?/#NULL!/#NUM!）
//   - missing_cache 有公式但无缓存值（文件由 openpyxl/excelize 写出未重算，
//     或缓存被清）——交付前必须 xlsx_recalc
//   - empty_result  公式重算过、结果就是空字符串（IFERROR(VLOOKUP(空键),"")
//     一类）：不是「没重算」，不阻塞交付门（2026-11-26 造价反馈三批）
//   - self_ref      公式直接引用自身单元格（循环引用最小面）
type AuditFinding struct {
	Sheet   string `json:"sheet"`
	Cell    string `json:"cell"`
	Kind    string `json:"kind"`
	Value   string `json:"value,omitempty"`
	Formula string `json:"formula,omitempty"`
}

// AuditReport 是 Audit 的结构化结果。
type AuditReport struct {
	Sheets        int            `json:"sheets"`
	TotalFormulas int            `json:"total_formulas"`
	Findings      []AuditFinding `json:"findings"`
	Truncated     bool           `json:"truncated"`

	// shapes 每张表里公式格的原始 XML 形态（表名小写 → 单元格 → 形态）。
	// 读不出来时为 nil：此时空值一律按未重算报（保守），不会把没重算的放过去。
	shapes map[string]map[string]formulaShape
}

// formulaShape 一个公式格在原始 XML 里的形态。excelize 的读值接口把「没有 <v>」
// 和「<v></v>」都读成空串，单看读值分不出「没重算」与「重算了、结果就是空串」
// ——实测（2026-11-26）三种写法的区别只在原始 XML：
//
//	excelize 写出未重算：  <c r="C1" t="str"><f>…</f></c>          （没有 <v>）
//	openpyxl 写出未重算：  <c r="C1"><f>…</f><v></v></c>           （没有 t）
//	LibreOffice 重算空串：<c r="C1" t="str"><f>…</f><v></v></c>
type formulaShape struct {
	typedStr        bool // 类型标记 t="str"
	hasValueElement bool // 有 <v> 元素（内容可为空）
}

// isEmptyStringResult 报告该公式格是不是「重算过、结果就是空串」。
func (r *AuditReport) isEmptyStringResult(sheet, cell string) bool {
	if r.shapes == nil {
		return false
	}
	s, ok := r.shapes[strings.ToLower(sheet)][cell]
	return ok && s.typedStr && s.hasValueElement
}

// rawFormulaShapes 直接读 xlsx 包里每张工作表的 XML，记下公式格的形态。
func rawFormulaShapes(path string) (map[string]map[string]formulaShape, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	parts, err := sheetParts(&zr.Reader)
	if err != nil {
		return nil, err
	}
	files := map[string]*zip.File{}
	for _, zf := range zr.File {
		files[zf.Name] = zf
	}
	out := make(map[string]map[string]formulaShape, len(parts))
	for name, part := range parts {
		zf, ok := files[part]
		if !ok {
			continue
		}
		rc, oerr := zf.Open()
		if oerr != nil {
			return nil, oerr
		}
		shapes, serr := scanFormulaShapes(rc)
		_ = rc.Close()
		if serr != nil {
			return nil, serr
		}
		out[strings.ToLower(name)] = shapes
	}
	return out, nil
}

// sheetParts 工作表名 → 包内部件路径（xl/workbook.xml 的 r:id 经 rels 解到 Target）。
func sheetParts(zr *zip.Reader) (map[string]string, error) {
	readPart := func(name string) ([]byte, error) {
		for _, zf := range zr.File {
			if zf.Name == name {
				rc, err := zf.Open()
				if err != nil {
					return nil, err
				}
				defer func() { _ = rc.Close() }()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("包里没有 %s", name)
	}
	wb, err := readPart("xl/workbook.xml")
	if err != nil {
		return nil, err
	}
	relsXML, err := readPart("xl/_rels/workbook.xml.rels")
	if err != nil {
		return nil, err
	}
	var wbSheets struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := xml.Unmarshal(wb, &wbSheets); err != nil {
		return nil, err
	}
	var rels struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.Unmarshal(relsXML, &rels); err != nil {
		return nil, err
	}
	target := map[string]string{}
	for _, r := range rels.Rel {
		t := strings.ReplaceAll(r.Target, "\\", "/")
		switch {
		case strings.HasPrefix(t, "/xl/"):
		case strings.HasPrefix(t, "/"):
			t = "xl" + t
		default:
			t = "xl/" + strings.TrimPrefix(t, "./")
		}
		target[r.ID] = t
	}
	out := map[string]string{}
	for _, s := range wbSheets.Sheets {
		if p, ok := target[s.RID]; ok {
			out[s.Name] = p
		}
	}
	return out, nil
}

// scanFormulaShapes 流式读一张工作表 XML，收公式格的形态。
func scanFormulaShapes(r io.Reader) (map[string]formulaShape, error) {
	dec := xml.NewDecoder(r)
	shapes := map[string]formulaShape{}
	var (
		ref, ctype string
		hasF, hasV bool
		inCell     bool
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case t.Name.Local == "c" && !inCell:
				inCell, hasF, hasV, ref, ctype = true, false, false, "", ""
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "r":
						ref = a.Value
					case "t":
						ctype = a.Value
					}
				}
			case inCell && t.Name.Local == "f":
				hasF = true
			case inCell && t.Name.Local == "v":
				hasV = true
			}
		case xml.EndElement:
			if t.Name.Local == "c" && inCell {
				if hasF && ref != "" {
					shapes[ref] = formulaShape{typedStr: ctype == "str", hasValueElement: hasV}
				}
				inCell = false
			}
		}
	}
	return shapes, nil
}

// auditCellRefRe 抽取公式里的裸单元格引用（LOG10 这类函数名同形，只在
// 与自身地址全等时才判 self_ref，见 scanCell）。
var auditCellRefRe = regexp.MustCompile(`\$?([A-Za-z]{1,3})\$?([0-9]{1,7})`)

// Audit 对 .xlsx 做只读审计（不改文件）。
// 范围口径 = 「sheetData 实际内容范围（Rows 迭代器）∪ dimension 声明范围」：
// 不单信 dimension——excelize/openpyxl 写出的文件该属性常为陈旧的 "A1"
// （实测）；也不单信 Rows——行尾无缓存值的公式格会被尾随裁剪。两者取并。
// 截断语义：单表声明面积超过 AuditMaxCells 时跳过扫描并如实标 truncated。
func Audit(path string) (AuditReport, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return AuditReport{}, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	rep := AuditReport{Findings: []AuditFinding{}}
	// 形态读不出来不拦审计：只是空值公式一律按未重算报（保守）。
	rep.shapes, _ = rawFormulaShapes(path)
	for _, sheet := range f.GetSheetList() {
		rep.Sheets++
		maxRow, maxCol := rowsExtent(f, sheet)
		if dim, derr := f.GetSheetDimension(sheet); derr == nil && dim != "" {
			if _, _, dc, dr, perr := parseDimension(dim); perr == nil {
				if dc > maxCol && dc <= 16384 {
					maxCol = dc // 列上限 XFD
				}
				if dr > maxRow && dr <= 1048576 {
					maxRow = dr // 行上限
				}
			}
		}
		if int64(maxCol)*int64(maxRow) > AuditMaxCells {
			rep.Truncated = true
			continue
		}
		for row := 1; row <= maxRow; row++ {
			for col := 1; col <= maxCol; col++ {
				cell, cnerr := excelize.CoordinatesToCellName(col, row)
				if cnerr != nil {
					continue
				}
				rep.scanCell(f, sheet, cell)
			}
		}
	}
	return rep, nil
}

// rowsExtent 用 Rows 迭代器取实际内容范围（行数取已见行号，列数取各行
// 非尾随裁剪后的最大列数——两者都是下界）。
func rowsExtent(f *excelize.File, sheet string) (maxRow, maxCol int) {
	rows, rerr := f.Rows(sheet)
	if rerr != nil {
		return 0, 0
	}
	defer func() { _ = rows.Close() }()
	rowIdx := 0
	for rows.Next() {
		rowIdx++
		if cols, cerr := rows.Columns(); cerr == nil && len(cols) > maxCol {
			maxCol = len(cols)
		}
	}
	return rowIdx, maxCol
}

// scanCell 检查单个单元格的三类问题并追加进报告。
func (r *AuditReport) scanCell(f *excelize.File, sheet, cell string) {
	formula, ferr := f.GetCellFormula(sheet, cell)
	if ferr != nil || formula == "" {
		return // 非公式单元格不在审计面
	}
	r.TotalFormulas++

	// ① 缓存错误值：单元格类型是 error（LibreOffice/Excel 重算后写回的缓存）。
	ct, terr := f.GetCellType(sheet, cell)
	if terr == nil && ct == excelize.CellTypeError {
		v, _ := f.GetCellValue(sheet, cell)
		r.Findings = append(r.Findings, AuditFinding{
			Sheet: sheet, Cell: cell, Kind: "error_value", Value: v, Formula: formula,
		})
		return
	}

	// ② 缓存值为空，分两种（判定依据是原始 XML 形态，见 rawFormulaShapes）：
	//   - t="str" 且有 <v> 元素（空的）——重算引擎写过这个格，结果就是空串
	//     （IFERROR(VLOOKUP(空键),"") 一类）。未用槽位的取价公式全是这种：测算
	//     样表 480 格，交付门此前一律判「未重算→未达」，回执又说「结果本应为空的
	//     不算问题」，自相矛盾（2026-11-26 造价反馈三批，三场实测 414/278/114 条）。
	//   - 其余空值——openpyxl/excelize 写出的公式格常态，才是真的没重算。
	v, verr := f.GetCellValue(sheet, cell)
	if verr == nil && strings.TrimSpace(v) == "" {
		kind := "missing_cache"
		if r.isEmptyStringResult(sheet, cell) {
			kind = "empty_result"
		}
		r.Findings = append(r.Findings, AuditFinding{
			Sheet: sheet, Cell: cell, Kind: kind, Formula: formula,
		})
	}

	// ③ 自引用（循环引用最小面）：公式裸引用自身地址。
	r.scanSelfRef(sheet, cell, formula)
}

// scanSelfRef 在公式里找与自身地址全等的裸引用（$ 与大小写归一）。
func (r *AuditReport) scanSelfRef(sheet, cell, formula string) {
	for _, m := range auditCellRefRe.FindAllStringSubmatch(formula, -1) {
		ref, rcerr := excelize.CoordinatesToCellName(
			ColumnNameToNumberSafe(m[1]), atoiSafe(m[2]))
		if rcerr != nil {
			continue
		}
		if strings.EqualFold(ref, cell) {
			r.Findings = append(r.Findings, AuditFinding{
				Sheet: sheet, Cell: cell, Kind: "self_ref", Formula: formula,
			})
			return
		}
	}
}

// parseDimension 把 "A1:F200" 解析成四角坐标（不合法时报错由调用方跳过）。
func parseDimension(dim string) (minCol, minRow, maxCol, maxRow int, err error) {
	parts := strings.Split(strings.ToUpper(dim), ":")
	start := parts[0]
	end := start
	if len(parts) == 2 {
		end = parts[1]
	}
	minCol, minRow, err = excelize.CellNameToCoordinates(start)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	maxCol, maxRow, err = excelize.CellNameToCoordinates(end)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return minCol, minRow, maxCol, maxRow, nil
}

// ColumnNameToNumberSafe 列字母 → 序号（非法输入返回 0，让调用方跳过）。
func ColumnNameToNumberSafe(name string) int {
	n, err := excelize.ColumnNameToNumber(strings.ToUpper(strings.ReplaceAll(name, "$", "")))
	if err != nil {
		return 0
	}
	return n
}

// atoiSafe 行号字符串 → int（非法输入返回 0）。
func atoiSafe(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// AuditFileExists 是工具层的存在性预检（错误文案要带原始路径）。
func AuditFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
