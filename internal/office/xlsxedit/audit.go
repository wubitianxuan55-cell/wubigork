package xlsxedit

// audit.go — xlsx 只读审计引擎（灵犀 optimize plan P1-2，docs/gaea-lingxi-
// optimize-plan-2026-11.md）：错误值扫描 + 循环引用最小面 + 未重算清单。
//
// 定位：xlsx_apply 应用后已有自动重算；本引擎给「交付门」提供复核依据——
// 交付前公式错误必须为零。深循环（A→B→A 多跳）交给 LibreOffice 重算链
// （Recalc 的 recalc.py 会报 Err:522），这里只做无需外部进程的结构面扫描。

import (
	"fmt"
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

	// ② 未重算：有公式但缓存值为空——openpyxl/excelize 写出的文件常态。
	v, verr := f.GetCellValue(sheet, cell)
	if verr == nil && strings.TrimSpace(v) == "" {
		r.Findings = append(r.Findings, AuditFinding{
			Sheet: sheet, Cell: cell, Kind: "missing_cache", Formula: formula,
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
