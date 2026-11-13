package xlsxedit

// profile.go — 列概况（波2 扩面：移植上游 dsh-office office_read 的 Excel
// profile 语义进无头管线）。对无头 xlsx 出模型可读文本：每列取值种类计数、
// 数值列 min/median/max、低基数列各取值+次数、可疑项（首尾空格/写成文本的
// 数字和日期/类型混杂/离群值/完全空行/整行重复）。只读不改文件——是模型
// 规划清洗 ops（xlsx_apply）的输入，不是清洗本身。
//
// 判定口径与 apply_one.go 的 clean/delete_rows 共用 parse.go 纯函数：
// GetCellValue（显示值）+ GetCellType（原始类型）双路——显示值能解析成数字
// 但存储类型是字符串 →「写成文本的数字」；日期形态同理。

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// maxProfileCells 概况扫描预算（单元格数）：超出拒绝并提示用 range 缩小
// 范围——概况是大表规划输入，不该把整本百万行表读进内存。
const maxProfileCells = 100000

// colStat 单列的概况统计（扫描期累积，渲染期消费）。
type colStat struct {
	colName   string
	header    string
	numbers   int
	dates     int
	texts     int
	blanks    int
	spaced    int // 首尾空格
	textNums  int // 写成文本的数字
	textDates int // 写成文本的日期
	values    []float64
	valueRows []int // 与 values 一一对应的行号（离群值定位）
	dateMin   float64
	dateMax   float64
	distinct  map[string]int
}

// Profile 对 path 的 sheet（缺省第一个表）的区块（缺省 used range，首行=表头）
// 生成列概况文本（模型可读，非 JSON）。
func Profile(path, sheet, rangeStr string) (string, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return "", fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("工作簿没有工作表")
	}
	if sheet == "" {
		sheet = sheets[0]
	}
	known := false
	for _, s := range sheets {
		if s == sheet {
			known = true
			break
		}
	}
	if !known {
		return "", fmt.Errorf("工作表 %q 不存在", sheet)
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return "", fmt.Errorf("读取工作表 %q 失败: %w", sheet, err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("表 %q 为空", sheet)
	}
	// 区块范围：给 range 用 range（越界裁到实际行数）；否则整表 used range。
	r1, c1, r2, c2 := 1, 1, len(rows), maxRowsWidth(rows)
	if rangeStr != "" {
		co, perr := parseRange(rangeStr)
		if perr != nil {
			return "", perr
		}
		r1, c1, r2, c2 = co[0], co[1], co[2], co[3]
		if r2 > len(rows) {
			r2 = len(rows)
		}
	}
	if r2-r1 < 1 {
		return "", fmt.Errorf("区块 %s 没有数据行（首行按表头处理）", rangeStr)
	}
	if (r2-r1+1)*(c2-c1+1) > maxProfileCells {
		return "", fmt.Errorf("区块过大（超过 %d 格）：先用 range 参数缩小范围", maxProfileCells)
	}

	// ── 整块扫描 ──────────────────────────────────────────────
	cols := make([]colStat, c2-c1+1)
	for i := range cols {
		name, _ := excelize.ColumnNumberToName(c1 + i)
		cols[i] = colStat{colName: name, distinct: map[string]int{}}
		if r1-1 < len(rows) {
			row := rows[r1-1]
			if (c1-1)+i < len(row) {
				cols[i].header = strings.TrimSpace(row[(c1-1)+i])
			}
		}
	}
	var emptyRows []int
	wholeDup := 0
	seenWhole := map[string]bool{}
	for r := r1 + 1; r <= r2; r++ {
		row := []string(nil)
		if r-1 < len(rows) {
			row = rows[r-1]
		}
		// 整行空判定与重复行键（区块宽度内）
		key := strings.Builder{}
		any := false
		for i := range cols {
			v := ""
			if (c1-1)+i < len(row) {
				v = row[(c1-1)+i]
			}
			key.WriteString(v)
			key.WriteByte(1)
			if tidyText(v) != "" {
				any = true
			}
		}
		if !any {
			emptyRows = append(emptyRows, r)
			continue
		}
		if seenWhole[key.String()] {
			wholeDup++
		} else {
			seenWhole[key.String()] = true
		}
		// 逐列分类
		for i := range cols {
			v := ""
			if (c1-1)+i < len(row) {
				v = row[(c1-1)+i]
			}
			st := &cols[i]
			disp := v
			if disp != "" {
				axis, _ := excelize.CoordinatesToCellName(c1+i, r)
				if got, gerr := f.GetCellValue(sheet, axis); gerr == nil {
					disp = got
				}
			}
			if tidyText(disp) == "" {
				st.blanks++
				continue
			}
			axis, _ := excelize.CoordinatesToCellName(c1+i, r)
			t, terr := f.GetCellType(sheet, axis)
			if terr != nil {
				t = excelize.CellTypeUnset
			}
			isString := t == excelize.CellTypeSharedString || t == excelize.CellTypeInlineString
			trimmed := strings.Trim(disp, " \t\u3000\u00A0")
			if disp != trimmed {
				st.spaced++
			}
			st.distinct[disp]++
			// 日期判定先行（上游同序）：显示值是日期形态且能解析 → 日期
			if !isString && displayDateShaped(disp) {
				if d, ok := parseFlexibleDate(disp, 0); ok {
					st.dates++
					serial, _ := dateToSerial(d)
					if st.dates == 1 {
						st.dateMin, st.dateMax = serial, serial
					} else {
						if serial < st.dateMin {
							st.dateMin = serial
						}
						if serial > st.dateMax {
							st.dateMax = serial
						}
					}
					continue
				}
			}
			if isString && looseDateShaped(disp) {
				if _, ok := parseFlexibleDate(disp, 0); ok {
					st.texts++
					st.textDates++
					continue
				}
			}
			if n, ok := parseLooseNumber(disp); ok {
				if isString {
					st.texts++
					st.textNums++
					continue
				}
				st.numbers++
				st.values = append(st.values, n)
				st.valueRows = append(st.valueRows, r)
				continue
			}
			st.texts++
		}
	}

	// ── 渲染 ─────────────────────────────────────────────────
	var b strings.Builder
	fmt.Fprintf(&b, "概况 %q：区块 %s，%d 列 × %d 数据行（首行按表头）",
		sheet, blockRangeText(c1, r1, c2, r2), c2-c1+1, r2-r1)
	if len(emptyRows) > 0 {
		fmt.Fprintf(&b, "；完全空行 %d（行 %s）", len(emptyRows), rowListText(emptyRows))
	}
	if wholeDup > 0 {
		fmt.Fprintf(&b, "；整行重复 %d 行", wholeDup)
	}
	b.WriteString("。\n")

	for i := range cols {
		st := &cols[i]
		head := st.header
		if head == "" {
			head = "(无表头)"
		}
		b.WriteString("\n" + st.colName + " \"" + head + "\"：")
		var kinds []string
		dateCol := st.dates > 0 && st.dates*2 > st.numbers+st.dates
		if st.numbers > 0 && !dateCol {
			kinds = append(kinds, fmt.Sprintf("数值 %d", st.numbers))
		}
		if st.dates > 0 {
			if st.numbers > 0 {
				kinds = append(kinds, fmt.Sprintf("日期 %d", st.dates), fmt.Sprintf("数值 %d", st.numbers))
			} else {
				kinds = append(kinds, fmt.Sprintf("日期 %d", st.dates))
			}
		}
		if st.texts > 0 {
			kinds = append(kinds, fmt.Sprintf("文本 %d", st.texts))
		}
		if st.blanks > 0 {
			kinds = append(kinds, fmt.Sprintf("空白 %d", st.blanks))
		}
		if len(kinds) == 0 {
			b.WriteString("空列")
		} else {
			b.WriteString(strings.Join(kinds, "，"))
		}
		// 数值列：min/median/max；日期列：日期范围
		if st.numbers > 0 && !dateCol {
			sorted := append([]float64(nil), st.values...)
			sort.Float64s(sorted)
			med := median(sorted)
			fmt.Fprintf(&b, "（min %s，median %s，max %s）",
				plainNumber(sorted[0]), plainNumber(med), plainNumber(sorted[len(sorted)-1]))
		}
		if st.dates > 0 && st.dateMax >= st.dateMin && st.dateMin > 0 {
			dmin, _ := serialToDate(st.dateMin)
			dmax, _ := serialToDate(st.dateMax)
			fmt.Fprintf(&b, "（%s ~ %s）", dmin.Format("2006-01-02"), dmax.Format("2006-01-02"))
		}
		// 低基数列：distinct ≤12 列各取值+次数（按次数降序，最多 24 项）
		if len(st.distinct) > 0 && len(st.distinct) <= 12 {
			keys := make([]string, 0, len(st.distinct))
			for k := range st.distinct {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(a, j int) bool {
				if st.distinct[keys[a]] != st.distinct[keys[j]] {
					return st.distinct[keys[a]] > st.distinct[keys[j]]
				}
				return keys[a] < keys[j]
			})
			const capList = 24
			parts := make([]string, 0, capList)
			for ki, k := range keys {
				if ki >= capList {
					break
				}
				parts = append(parts, strconv.Quote(clip(k, 24))+" "+strconv.Itoa(st.distinct[k]))
			}
			fmt.Fprintf(&b, "；%d 种取值：%s", len(st.distinct), strings.Join(parts, "，"))
		}
		// 可疑项
		var odd []string
		if st.spaced > 0 {
			odd = append(odd, fmt.Sprintf("%d 格首尾空格", st.spaced))
		}
		if st.textNums > 0 {
			odd = append(odd, fmt.Sprintf("%d 格数字写成文本", st.textNums))
		}
		if st.textDates > 0 {
			odd = append(odd, fmt.Sprintf("%d 格日期写成文本", st.textDates))
		}
		if st.numbers > 0 && st.texts > 0 {
			odd = append(odd, "数值文本混杂")
		}
		if outlier := outlierText(st); outlier != "" {
			odd = append(odd, outlier)
		}
		if len(odd) > 0 {
			b.WriteString("。可疑：" + strings.Join(odd, "，"))
		}
		b.WriteString("。")
	}
	return b.String(), nil
}

// outlierText 数值列离群值（>Q3+1.5IQR 或 <Q1−1.5IQR），列出行号 ≤5。
func outlierText(st *colStat) string {
	n := len(st.values)
	if n < 4 {
		return ""
	}
	sorted := append([]float64(nil), st.values...)
	sort.Float64s(sorted)
	q1 := sorted[n/4]
	q3 := sorted[n*3/4]
	iqr := q3 - q1
	low := q1 - 1.5*iqr
	high := q3 + 1.5*iqr
	var rows []int
	for i, v := range st.values {
		if v > high || v < low {
			rows = append(rows, st.valueRows[i])
		}
	}
	if len(rows) == 0 {
		return ""
	}
	return fmt.Sprintf("离群值 %d 个（>%s 或 <%s；行 %s）",
		len(rows), plainNumber(high), plainNumber(low), rowListText(rows))
}

// median 中位数（入参须已排序；偶数个取中间两值均值）。
func median(sorted []float64) float64 {
	m := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[m]
	}
	return (sorted[m-1] + sorted[m]) / 2
}

// blockRangeText 区块的 A1 形态范围文本。
func blockRangeText(c1, r1, c2, r2 int) string {
	tl, _ := excelize.CoordinatesToCellName(c1, r1)
	br, _ := excelize.CoordinatesToCellName(c2, r2)
	return tl + ":" + br
}

// clip 截断长文本（概况回执里的取值样本）。
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// dateToSerial/serialToDate Excel 日期序列值与 time.Time 的互转（1900 日期
// 系统：序列 1 = 1899-12-31；与 excelize 的换算口径一致）。
var excelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

func dateToSerial(d time.Time) (float64, error) {
	utc := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	days := utc.Sub(excelEpoch).Hours() / 24
	return days, nil
}

func serialToDate(serial float64) (time.Time, error) {
	return excelEpoch.AddDate(0, 0, int(math.Round(serial))), nil
}
