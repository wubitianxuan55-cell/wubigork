package xlsxedit

// apply_one.go — applyOne 的表驱动实现体（2026-10-04 自 253 行 switch 逐字
// 搬迁，配方照 internal/schedule/ops.go opHandlers 先例：契约先行→表驱动→
// 零编辑复跑；summary 回执文案由 apply_one_test.go 冻结）。

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// opHandlers 把 Op.Type 映射到对应 apply 函数：键集由
// TestOpHandlersKeySetLocked 冻结，新增 op 类型须注册于此并补 summary 契约例。
var opHandlers = map[string]func(*excelize.File, Op) (string, error){
	"set_formula":       applySetFormula,
	"set_value":         applySetValue,
	"fill_range":        applyFillRange,
	"transform":         applyTransform,
	"replace":           applyReplace,
	"split_column":      applySplitColumn,
	"clean":             applyClean,
	"set_style":         applySetStyle,
	"merge_cells":       applyMergeCells,
	"unmerge_cells":     applyUnmergeCells,
	"set_col_width":     applySetColWidth,
	"fill_blanks":       applyFillBlanks,
	"remove_duplicates": applyRemoveDuplicates,
	"delete_rows":       applyDeleteRows,
	"insert_rows":       applyInsertRows,
	"insert_columns":    applyInsertColumns,
	"delete_columns":    applyDeleteColumns,
	"sort":              applySort,
	"add_column":        applyAddColumn,
	"copy_range":        applyCopyRange,
}

func applySetFormula(f *excelize.File, op Op) (string, error) {
	if op.Target == "" || op.Formula == "" {
		return "", fmt.Errorf("target/formula 缺失")
	}
	if err := f.SetCellFormula(op.Sheet, op.Target, strings.TrimPrefix(op.Formula, "=")); err != nil {
		return "", err
	}
	return fmt.Sprintf("写入公式 %s=%s", op.Target, op.Formula), nil
}

func applySetValue(f *excelize.File, op Op) (string, error) {
	if op.Target == "" {
		return "", fmt.Errorf("target 缺失")
	}
	if err := f.SetCellValue(op.Sheet, op.Target, op.Value); err != nil {
		return "", err
	}
	return fmt.Sprintf("写入值 %s=%v", op.Target, op.Value), nil
}

func applyFillRange(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	n := 0
	for r := coords[0]; r <= coords[2]; r++ {
		for c := coords[1]; c <= coords[3]; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			if err := f.SetCellValue(op.Sheet, axis, op.Value); err != nil {
				return "", err
			}
			if n++; n > maxCellsPerOp {
				return "", fmt.Errorf("填充区域过大（超过 %d 格）", maxCellsPerOp)
			}
		}
	}
	return fmt.Sprintf("填充 %s = %v", op.Range, op.Value), nil
}

func applyTransform(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	if op.Formula == "" {
		return "", fmt.Errorf("formula 缺失")
	}
	if coords[1] != coords[3] {
		return "", fmt.Errorf("transform 仅支持单列区域（列方向公式调整暂不支持）")
	}
	n := 0
	for r := coords[0]; r <= coords[2]; r++ {
		axis, _ := excelize.CoordinatesToCellName(coords[1], r)
		formula := adjustRowRefs(op.Formula, r-coords[0])
		if err := f.SetCellFormula(op.Sheet, axis, strings.TrimPrefix(formula, "=")); err != nil {
			return "", err
		}
		if n++; n > maxCellsPerOp {
			return "", fmt.Errorf("变换区域过大（超过 %d 格）", maxCellsPerOp)
		}
	}
	return fmt.Sprintf("逐行公式 %s（%d 行）", op.Range, coords[2]-coords[0]+1), nil
}

func applyReplace(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	count := 0
	for r := coords[0]; r <= coords[2]; r++ {
		for c := coords[1]; c <= coords[3]; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			val, err := f.GetCellValue(op.Sheet, axis)
			if err != nil || val == "" {
				continue
			}
			if strings.Contains(val, op.Find) {
				if err := f.SetCellValue(op.Sheet, axis, strings.ReplaceAll(val, op.Find, op.Replace)); err != nil {
					return "", err
				}
				count++
			}
		}
	}
	return fmt.Sprintf("替换 %s：%q → %q（%d 格）", op.Range, op.Find, op.Replace, count), nil
}

func applySplitColumn(f *excelize.File, op Op) (string, error) {
	if op.Col == "" || op.Sep == "" || len(op.NewCols) == 0 {
		return "", fmt.Errorf("col/sep/newCols 缺失")
	}
	colNum, err := excelize.ColumnNameToNumber(op.Col)
	if err != nil {
		return "", err
	}
	if len(op.Headers) > 0 {
		for i, h := range op.Headers {
			if i >= len(op.NewCols) {
				break
			}
			if err := f.SetCellValue(op.Sheet, op.NewCols[i]+"1", h); err != nil {
				return "", err
			}
		}
	}
	rows, err := f.GetRows(op.Sheet)
	if err != nil {
		return "", err
	}
	count := 0
	for ri := 1; ri < len(rows); ri++ {
		row := rows[ri]
		if colNum-1 >= len(row) {
			continue
		}
		parts := strings.Split(row[colNum-1], op.Sep)
		for i, part := range parts {
			if i >= len(op.NewCols) {
				break
			}
			axis := op.NewCols[i] + strconv.Itoa(ri+1)
			if err := f.SetCellValue(op.Sheet, axis, strings.TrimSpace(part)); err != nil {
				return "", err
			}
			count++
		}
	}
	return fmt.Sprintf("拆分 %s 列 → %s（%d 格）", op.Col, strings.Join(op.NewCols, ","), count), nil
}

// applyClean 清洗区域（波2 扩面）：在既有 trim/upper/lower 之上支持
// Remove（删除字样）/NoSpaces（去全部空格）/Map（取值改名）/To（转
// number/date/text）。To 转换按显示值解析（parseLooseNumber/parseFlexibleDate），
// 解析失败的原样保留并计数报告；公式格一律不动（清洗针对数据）。
func applyClean(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	to := strings.ToLower(strings.TrimSpace(op.To))
	switch to {
	case "", "number", "date", "text":
	default:
		return "", fmt.Errorf("clean 的 to 仅支持 number/date/text，得到 %q", op.To)
	}
	// Map 键过同一套归一（tidyText），值原样替换。
	lookup := make(map[string]string, len(op.Map))
	for k, v := range op.Map {
		lookup[tidyText(k)] = v
	}
	count := 0
	failed := 0
	for r := coords[0]; r <= coords[2]; r++ {
		for c := coords[1]; c <= coords[3]; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			val, err := f.GetCellValue(op.Sheet, axis)
			if err != nil || val == "" {
				continue
			}
			// 公式格不清洗（会被字符串覆盖掉公式）。
			if formula, ferr := f.GetCellFormula(op.Sheet, axis); ferr == nil && formula != "" {
				continue
			}
			if to != "" {
				done, ok := cleanConvert(f, op, axis, val, to)
				if ok {
					if done {
						count++
					}
				} else {
					failed++
				}
				continue
			}
			orig := val
			if op.Trim {
				val = tidyText(val)
			}
			if op.NoSpaces {
				val = strings.ReplaceAll(val, " ", "")
				val = strings.ReplaceAll(val, "\u3000", "")
				val = strings.ReplaceAll(val, "\t", "")
			}
			for _, piece := range op.Remove {
				if piece != "" {
					val = strings.ReplaceAll(val, piece, "")
				}
			}
			if op.Trim {
				val = strings.TrimSpace(val)
			}
			if op.Upper {
				val = strings.ToUpper(val)
			}
			if op.Lower {
				val = strings.ToLower(val)
			}
			if mapped, ok := lookup[val]; ok {
				val = mapped
			}
			if val != orig {
				if err := f.SetCellValue(op.Sheet, axis, val); err != nil {
					return "", err
				}
				count++
			}
		}
	}
	if to != "" {
		if failed > 0 {
			return fmt.Sprintf("清洗 %s（%d 格，转 %s；%d 格解析失败，原样保留）", op.Range, count, to, failed), nil
		}
		return fmt.Sprintf("清洗 %s（%d 格，转 %s）", op.Range, count, to), nil
	}
	return fmt.Sprintf("清洗 %s（%d 格）", op.Range, count), nil
}

// cleanConvert 对单格做 To 转换。返回（是否写了格子, 解析是否成功）；
// 本来就是目标形态的格子不写但算成功。
func cleanConvert(f *excelize.File, op Op, axis, val, to string) (bool, bool) {
	t, terr := f.GetCellType(op.Sheet, axis)
	if terr != nil {
		t = excelize.CellTypeUnset
	}
	isString := t == excelize.CellTypeSharedString || t == excelize.CellTypeInlineString
	switch to {
	case "number":
		if !isString {
			// 数字格/公式结果已按数值存储，无需转换
			return false, true
		}
		n, ok := parseLooseNumber(val)
		if !ok {
			return false, false
		}
		if err := f.SetCellValue(op.Sheet, axis, n); err != nil {
			return false, false
		}
		return true, true
	case "date":
		if !isString && displayDateShaped(val) {
			// 已格式化的日期格，无需转换
			return false, true
		}
		// 优先按文本解析；纯 8 位数字（20250301，含存储为数字的）按日期换算。
		if t2, ok := parseFlexibleDate(val, time.Now().Year()); ok {
			if err := writeDateCell(f, op.Sheet, axis, t2); err != nil {
				return false, false
			}
			return true, true
		}
		if n, ok := parseLooseNumber(val); ok && n == float64(int64(n)) && n >= 19000101 && n <= 22001231 {
			if t2, ok := parseFlexibleDate(strconv.FormatInt(int64(n), 10), time.Now().Year()); ok {
				if err := writeDateCell(f, op.Sheet, axis, t2); err != nil {
					return false, false
				}
				return true, true
			}
		}
		return false, false
	default: // "text"
		if isString {
			return false, true
		}
		// 数字/日期格 → 显示值写成文本
		if err := f.SetCellValue(op.Sheet, axis, val); err != nil {
			return false, false
		}
		return true, true
	}
}

// writeDateCell 写入日期并叠加 yyyy-mm-dd 显示格式（excelize 的 time.Time
// 缺省格式是 m/d/yy h:mm，落库统一日期观感）。
func writeDateCell(f *excelize.File, sheet, axis string, d time.Time) error {
	if err := f.SetCellValue(sheet, axis, d); err != nil {
		return err
	}
	return applyNumFmt(f, sheet, axis, "yyyy-mm-dd")
}

// applyNumFmt 给单元格叠加数字格式（mergeStyle 只动 NumFmt 字段，其余样式保留）。
func applyNumFmt(f *excelize.File, sheet, axis, numFmt string) error {
	oldID, err := f.GetCellStyle(sheet, axis)
	if err != nil {
		oldID = 0
	}
	nid, err := mergeStyle(f, oldID, &Style{NumFmt: numFmt})
	if err != nil {
		return err
	}
	return f.SetCellStyle(sheet, axis, axis, nid)
}

func applySetStyle(f *excelize.File, op Op) (string, error) {
	if op.Style == nil {
		return "", fmt.Errorf("style 缺失")
	}
	var cells []string
	if op.Target != "" {
		cells = []string{op.Target}
	} else if op.Range != "" {
		coords, err := parseRange(op.Range)
		if err != nil {
			return "", err
		}
		for r := coords[0]; r <= coords[2]; r++ {
			for c := coords[1]; c <= coords[3]; c++ {
				axis, _ := excelize.CoordinatesToCellName(c, r)
				cells = append(cells, axis)
			}
		}
		if len(cells) > maxCellsPerOp {
			return "", fmt.Errorf("样式区域过大（超过 %d 格）", maxCellsPerOp)
		}
	} else {
		return "", fmt.Errorf("target/range 缺失")
	}
	// 旧样式ID → 叠加后样式ID 的缓存：区域里相同起点的样式只新建一次
	styleCache := map[int]int{}
	for _, axis := range cells {
		oldID, err := f.GetCellStyle(op.Sheet, axis)
		if err != nil {
			oldID = 0
		}
		nid, ok := styleCache[oldID]
		if !ok {
			nid, err = mergeStyle(f, oldID, op.Style)
			if err != nil {
				return "", err
			}
			styleCache[oldID] = nid
		}
		if err := f.SetCellStyle(op.Sheet, axis, axis, nid); err != nil {
			return "", err
		}
	}
	scope := op.Range
	if op.Target != "" {
		scope = op.Target
	}
	return fmt.Sprintf("设置样式 %s（%d 格）", scope, len(cells)), nil
}

func applyMergeCells(f *excelize.File, op Op) (string, error) {
	if op.Range == "" {
		return "", fmt.Errorf("range 缺失")
	}
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	tl, _ := excelize.CoordinatesToCellName(coords[1], coords[0])
	br, _ := excelize.CoordinatesToCellName(coords[3], coords[2])
	if err := f.MergeCell(op.Sheet, tl, br); err != nil {
		return "", err
	}
	return fmt.Sprintf("合并 %s", op.Range), nil
}

func applyUnmergeCells(f *excelize.File, op Op) (string, error) {
	if op.Range == "" {
		return "", fmt.Errorf("range 缺失")
	}
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	tl, _ := excelize.CoordinatesToCellName(coords[1], coords[0])
	br, _ := excelize.CoordinatesToCellName(coords[3], coords[2])
	if err := f.UnmergeCell(op.Sheet, tl, br); err != nil {
		return "", err
	}
	return fmt.Sprintf("取消合并 %s", op.Range), nil
}

func applySetColWidth(f *excelize.File, op Op) (string, error) {
	if op.Col == "" || op.Width <= 0 {
		return "", fmt.Errorf("col/width 缺失或非法")
	}
	if err := f.SetColWidth(op.Sheet, op.Col, op.Col, op.Width); err != nil {
		return "", err
	}
	return fmt.Sprintf("列宽 %s = %.1f", op.Col, op.Width), nil
}

// ── 波2 扩面：数据作业 ops（dsh-office「Excel: data work」差异化能力移植）──

// usedRangeCoords 工作表实际内容范围（GetRows 的宽高；首行=表头）。
// 空表返回 1:1。
func usedRangeCoords(f *excelize.File, sheet string) ([4]int, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return [4]int{}, err
	}
	last, width := len(rows), 0
	for _, r := range rows {
		if len(r) > width {
			width = len(r)
		}
	}
	if last < 1 {
		last = 1
	}
	if width < 1 {
		width = 1
	}
	return [4]int{1, 1, last, width}, nil
}

// maxRowsWidth 各行中的最大列数。
func maxRowsWidth(rows [][]string) int {
	width := 0
	for _, r := range rows {
		if len(r) > width {
			width = len(r)
		}
	}
	if width < 1 {
		width = 1
	}
	return width
}

// rowListText 行号清单回执（≤12 个，超出截断加 ".."——诚实摘要不冒充全量）。
func rowListText(rows []int) string {
	parts := make([]string, 0, len(rows))
	const limit = 12
	for i, r := range rows {
		if i >= limit {
			parts = append(parts, "..")
			break
		}
		parts = append(parts, strconv.Itoa(r))
	}
	return strings.Join(parts, ", ")
}

// plainNumber 数值的可读形态：整值不带小数，其余最多 4 位小数并去尾零
// （上游 Plain 同语义）。
func plainNumber(n float64) string {
	if math.Abs(n-math.Round(n)) < 1e-9 && math.Abs(n) < 1e15 {
		return strconv.FormatFloat(math.Round(n), 'f', -1, 64)
	}
	s := strconv.FormatFloat(n, 'f', 4, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

// applyFillBlanks 填充空值：col 或单列 range 区域内，空白格按 with 填充。
// with = 值 | "above"（向上填充）| "mean" | "median" | "mode"（按该列现有
// 显示值解析数字统计；解析不出的跳过）。表头行不参与填充。
func applyFillBlanks(f *excelize.File, op Op) (string, error) {
	with := op.With
	if with == nil {
		with = op.Value
	}
	if with == nil {
		return "", fmt.Errorf("fill_blanks 缺 with：值或 above/mean/median/mode")
	}
	var colNum, rowStart, rowEnd int
	var scope string
	if op.Col != "" {
		n, err := excelize.ColumnNameToNumber(strings.TrimSpace(op.Col))
		if err != nil {
			return "", fmt.Errorf("fill_blanks 列 %q 无效: %w", op.Col, err)
		}
		rows, err := f.GetRows(op.Sheet)
		if err != nil {
			return "", err
		}
		colNum, rowStart, rowEnd = n, 2, len(rows)
		scope = strings.TrimSpace(op.Col) + " 列"
	} else if op.Range != "" {
		coords, err := parseRange(op.Range)
		if err != nil {
			return "", err
		}
		if coords[1] != coords[3] {
			return "", fmt.Errorf("fill_blanks 区域应为单列（如 B2:B9），得到 %q", op.Range)
		}
		colNum, rowStart, rowEnd = coords[1], coords[0], coords[2]
		scope = op.Range
	} else {
		return "", fmt.Errorf("fill_blanks 缺 col/range")
	}
	mode := ""
	if s, ok := with.(string); ok {
		mode = strings.ToLower(tidyText(s))
	}
	// 统计填充源（均值/中位数/众数按现有显示值算）。
	var numbers []float64
	seenCount := map[string]int{}
	var commonest string
	for r := rowStart; r <= rowEnd; r++ {
		axis, _ := excelize.CoordinatesToCellName(colNum, r)
		disp, _ := f.GetCellValue(op.Sheet, axis)
		if tidyText(disp) == "" {
			continue
		}
		if n, ok := parseLooseNumber(disp); ok {
			numbers = append(numbers, n)
		}
		seenCount[disp]++
		if seenCount[disp] > seenCount[commonest] {
			commonest = disp
		}
	}
	if seenCount[commonest] == 0 {
		commonest = ""
	}
	// 解析填充值。
	fillText := ""
	literal := interface{}(nil)
	switch mode {
	case "above":
		// 逐格沿上方非空值向下填充；值在循环里取。
	case "mean":
		if len(numbers) == 0 {
			return "", fmt.Errorf("fill_blanks：%s 没有可解析的数字，算不出均值", scope)
		}
		sum := 0.0
		for _, n := range numbers {
			sum += n
		}
		literal = math.Round(sum/float64(len(numbers))*10000) / 10000
		fillText = "填均值 " + plainNumber(literal.(float64))
	case "median":
		if len(numbers) == 0 {
			return "", fmt.Errorf("fill_blanks：%s 没有可解析的数字，算不出中位数", scope)
		}
		sort.Float64s(numbers)
		m := len(numbers) / 2
		var med float64
		if len(numbers)%2 == 1 {
			med = numbers[m]
		} else {
			med = (numbers[m-1] + numbers[m]) / 2
		}
		literal = med
		fillText = "填中位数 " + plainNumber(med)
	case "mode":
		if commonest == "" {
			return "", fmt.Errorf("fill_blanks：%s 没有非空值，取不出众数", scope)
		}
		literal = commonest
		fillText = "填众数 " + strconv.Quote(commonest)
	default:
		literal = with
		fillText = "填 " + fmt.Sprint(with)
	}
	filled := 0
	var last string
	lastRowNum := 0
	for r := rowStart; r <= rowEnd; r++ {
		axis, _ := excelize.CoordinatesToCellName(colNum, r)
		disp, _ := f.GetCellValue(op.Sheet, axis)
		if tidyText(disp) != "" {
			last = disp
			lastRowNum = r
			continue
		}
		var value interface{}
		if mode == "above" {
			if last == "" {
				continue // 上方还没有值：保持空白
			}
			// 延续来源格的存储形态：数值格填数值，文本格填文本。
			if n, ok := parseLooseNumber(last); ok && isNumericCell(f, op.Sheet, axisOf(colNum, lastRowNum)) {
				value = n
			} else {
				value = last
			}
		} else {
			value = literal
		}
		if err := f.SetCellValue(op.Sheet, axis, value); err != nil {
			return "", err
		}
		if filled++; filled > maxCellsPerOp {
			return "", fmt.Errorf("填充区域过大（超过 %d 格）", maxCellsPerOp)
		}
	}
	if mode == "above" {
		fillText = "向上填充"
	}
	return fmt.Sprintf("填充空值 %s：%d 格（%s）", scope, filled, fillText), nil
}

// axisOf 单元格坐标小工具（列号+行号 → "B3"）。
func axisOf(col, row int) string {
	axis, _ := excelize.CoordinatesToCellName(col, row)
	return axis
}

// isNumericCell 判断格子当前是否按数值存储（above 填充延续数值形态用）。
func isNumericCell(f *excelize.File, sheet, axis string) bool {
	t, err := f.GetCellType(sheet, axis)
	if err != nil {
		return false
	}
	return t == excelize.CellTypeNumber || t == excelize.CellTypeDate || t == excelize.CellTypeUnset
}

// applyRemoveDuplicates 按指定列判重（缺省整行）：保留首条、删后续。
// 键做 tidy+大小写归一（与上游一致）；整行全空的行不参与判重。
// 从底往上删行，避免索引位移。
func applyRemoveDuplicates(f *excelize.File, op Op) (string, error) {
	rows, err := f.GetRows(op.Sheet)
	if err != nil {
		return "", err
	}
	if len(rows) < 2 {
		return "", fmt.Errorf("remove_duplicates：没有数据行")
	}
	width := maxRowsWidth(rows)
	var keyCols []int
	for _, name := range op.Columns {
		n, err := excelize.ColumnNameToNumber(strings.TrimSpace(name))
		if err != nil {
			return "", fmt.Errorf("remove_duplicates 列 %q 无效: %w", name, err)
		}
		keyCols = append(keyCols, n)
	}
	seen := make(map[string]bool)
	var gone []int
	for r := 2; r <= len(rows); r++ {
		row := rows[r-1]
		cols := keyCols
		if len(cols) == 0 {
			cols = make([]int, 0, width)
			for c := 1; c <= width; c++ {
				cols = append(cols, c)
			}
		}
		key := strings.Builder{}
		any := false
		for _, c := range cols {
			v := ""
			if c-1 < len(row) {
				v = row[c-1]
			}
			t := tidyText(v)
			if t != "" {
				any = true
			}
			key.WriteString(strings.ToUpper(t))
			key.WriteByte(1)
		}
		if !any {
			continue
		}
		k := key.String()
		if seen[k] {
			gone = append(gone, r)
			continue
		}
		seen[k] = true
	}
	what := "整行"
	if len(op.Columns) > 0 {
		what = "列 " + strings.Join(op.Columns, ",")
	}
	if len(gone) == 0 {
		return fmt.Sprintf("去重 %s：无重复行", what), nil
	}
	for i := len(gone) - 1; i >= 0; i-- {
		if err := f.RemoveRow(op.Sheet, gone[i]); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("去重 %s：删 %d 行（保留首条；行 %s）", what, len(gone), rowListText(gone)), nil
}

// applyDeleteRows 三种形态（字段互斥）：blank=true 删整行全空的行；
// where 按 col 列条件删行；row(+count) 直接删指定行。统一从底往上删。
func applyDeleteRows(f *excelize.File, op Op) (string, error) {
	rows, err := f.GetRows(op.Sheet)
	if err != nil {
		return "", err
	}
	last := len(rows)
	width := maxRowsWidth(rows)
	var gone []int
	cond := ""
	switch {
	case op.Where != nil:
		wh := op.Where
		colNum, err := excelize.ColumnNameToNumber(strings.TrimSpace(wh.Col))
		if err != nil {
			return "", fmt.Errorf("delete_rows 列 %q 无效: %w", wh.Col, err)
		}
		how := strings.ToLower(strings.TrimSpace(wh.Is))
		if how == "" {
			how = "blank"
		}
		switch how {
		case "blank", "=", "==", "<>", "!=", ">", "<", ">=", "<=", "contains", "text":
		default:
			return "", fmt.Errorf("delete_rows 的 is 仅支持 blank/=/<>/>/</>=/<=/contains/text，得到 %q", wh.Is)
		}
		cond = fmt.Sprintf("%s 列 %s", strings.TrimSpace(wh.Col), how)
		if wh.Value != nil && how != "blank" && how != "text" {
			cond += " " + fmt.Sprint(wh.Value)
		}
		for r := 2; r <= last; r++ {
			if rowMatches(f, op.Sheet, r, colNum, how, wh.Value) {
				gone = append(gone, r)
			}
		}
	case op.Blank:
		cond = "整行空白"
		for r := 2; r <= last; r++ {
			row := rows[r-1]
			any := false
			for c := 1; c <= width; c++ {
				if c-1 < len(row) && tidyText(row[c-1]) != "" {
					any = true
					break
				}
			}
			if !any {
				gone = append(gone, r)
			}
		}
	default:
		if op.Row < 1 {
			return "", fmt.Errorf("delete_rows 需 blank/where/row 之一")
		}
		count := op.Count
		if count <= 0 {
			count = 1
		}
		for r := op.Row; r < op.Row+count && r <= last; r++ {
			gone = append(gone, r)
		}
		if count == 1 {
			cond = fmt.Sprintf("第 %d 行", op.Row)
		} else {
			cond = fmt.Sprintf("第 %d-%d 行", op.Row, op.Row+count-1)
		}
	}
	if len(gone) == 0 {
		return fmt.Sprintf("删行 %s：无符合条件的行", cond), nil
	}
	for i := len(gone) - 1; i >= 0; i-- {
		if err := f.RemoveRow(op.Sheet, gone[i]); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("删行 %s：%d 行（行 %s）", cond, len(gone), rowListText(gone)), nil
}

// rowMatches 条件删行的单行判定。数值比较（>/</>=/<=）两侧都要能解析成数字；
// =/<> 优先数字比较，否则大小写不敏感文本比较；text = 写成文本的数字或日期。
func rowMatches(f *excelize.File, sheet string, row, colNum int, how string, against interface{}) bool {
	disp, _ := f.GetCellValue(sheet, axisOf(colNum, row))
	blank := tidyText(disp) == ""
	switch how {
	case "blank":
		return blank
	case "contains":
		if blank {
			return false
		}
		return strings.Contains(strings.ToLower(tidyText(disp)), strings.ToLower(tidyText(fmt.Sprint(against))))
	case "text":
		if blank {
			return false
		}
		t, err := f.GetCellType(sheet, axisOf(colNum, row))
		if err != nil {
			return false
		}
		if t != excelize.CellTypeSharedString && t != excelize.CellTypeInlineString {
			return false
		}
		if _, ok := parseLooseNumber(disp); ok {
			return true
		}
		return looseDateShaped(disp) && func() bool { _, ok := parseFlexibleDate(disp, 0); return ok }()
	}
	// 数值比较族 + 等于/不等
	var left float64
	var leftOK bool
	if !blank {
		left, leftOK = parseLooseNumber(disp)
	}
	var right float64
	var rightOK bool
	if against != nil {
		right, rightOK = againstNumber(against)
	}
	switch how {
	case ">", "<", ">=", "<=":
		if !leftOK || !rightOK {
			return false
		}
	case "=", "==", "<>", "!=":
		if !rightOK {
			return false
		}
		if !leftOK {
			if blank {
				return false
			}
			eq := strings.EqualFold(tidyText(disp), tidyText(fmt.Sprint(against)))
			if how == "=" || how == "==" {
				return eq
			}
			return !eq
		}
	}
	if !leftOK || !rightOK {
		return false
	}
	switch how {
	case "=", "==":
		return left == right
	case "<>", "!=":
		return left != right
	case ">":
		return left > right
	case "<":
		return left < right
	case ">=":
		return left >= right
	case "<=":
		return left <= right
	}
	return false
}

// againstNumber 条件值 → 数字（JSON 数字直取；字符串走宽松解析）。
func againstNumber(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		return parseLooseNumber(x)
	}
	return 0, false
}

// applyInsertRows 在 row 行前插入 count 行（缺省 1）。
func applyInsertRows(f *excelize.File, op Op) (string, error) {
	if op.Row < 1 {
		return "", fmt.Errorf("insert_rows 缺 row")
	}
	n := op.Count
	if n <= 0 {
		n = 1
	}
	if err := f.InsertRows(op.Sheet, op.Row, n); err != nil {
		return "", err
	}
	return fmt.Sprintf("插入 %d 行（第 %d 行前）", n, op.Row), nil
}

// applyInsertColumns 在 col 列前插入 count 列（缺省 1）。
func applyInsertColumns(f *excelize.File, op Op) (string, error) {
	if op.Col == "" {
		return "", fmt.Errorf("insert_columns 缺 col")
	}
	n := op.Count
	if n <= 0 {
		n = 1
	}
	if err := f.InsertCols(op.Sheet, op.Col, n); err != nil {
		return "", err
	}
	return fmt.Sprintf("插入 %d 列（%s 列前）", n, op.Col), nil
}

// applyDeleteColumns 从 col 列起删除 count 列（缺省 1；逐列删，删一次右移一次）。
func applyDeleteColumns(f *excelize.File, op Op) (string, error) {
	if op.Col == "" {
		return "", fmt.Errorf("delete_columns 缺 col")
	}
	n := op.Count
	if n <= 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		if err := f.RemoveCol(op.Sheet, op.Col); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("删除 %d 列（%s 列起）", n, op.Col), nil
}

// applySort 多键排序：读数据区（range 或整表 used range，首行默认表头不动），
// 按键列比较（两侧都能解析数字按数值比，否则大小写不敏感文本比）后整体回写。
// 数值格按原始值回写保持类型；公式随行位移平移相对引用。
func applySort(f *excelize.File, op Op) (string, error) {
	if len(op.By) == 0 {
		return "", fmt.Errorf("sort 缺 by")
	}
	coords, err := parseRange(op.Range)
	if err != nil {
		if op.Range != "" {
			return "", err
		}
		if coords, err = usedRangeCoords(f, op.Sheet); err != nil {
			return "", err
		}
	}
	if coords[2]-coords[0] < 1 {
		return fmt.Sprintf("排序 %s：无数据行", sortScope(op)), nil
	}
	// 读数据区：显示值 + 原始值 + 类型 + 公式。
	type cellVal struct {
		disp    string
		raw     string
		typ     excelize.CellType
		formula string
	}
	grid := make([][]cellVal, coords[2]-coords[0]+1)
	for r := coords[0]; r <= coords[2]; r++ {
		row := make([]cellVal, coords[3]-coords[1]+1)
		for c := coords[1]; c <= coords[3]; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			disp, _ := f.GetCellValue(op.Sheet, axis)
			raw, _ := f.GetCellValue(op.Sheet, axis, excelize.Options{RawCellValue: true})
			formula, _ := f.GetCellFormula(op.Sheet, axis)
			t, _ := f.GetCellType(op.Sheet, axis)
			row[c-coords[1]] = cellVal{disp: disp, raw: raw, typ: t, formula: formula}
		}
		grid[r-coords[0]] = row
	}
	type rowRef struct {
		cells []cellVal
		orig  int
	}
	data := make([]rowRef, coords[2]-coords[0])
	for i := range data {
		data[i] = rowRef{cells: grid[i+1], orig: i}
	}
	keys := make([]int, 0, len(op.By))
	downs := make([]bool, 0, len(op.By))
	said := make([]string, 0, len(op.By))
	for _, k := range op.By {
		num, err := excelize.ColumnNameToNumber(strings.TrimSpace(k.Column))
		if err != nil {
			return "", fmt.Errorf("sort 键列 %q 无效: %w", k.Column, err)
		}
		idx := num - coords[1]
		if idx < 0 || idx >= len(grid[0]) {
			return "", fmt.Errorf("sort 键列 %q 在数据区之外", k.Column)
		}
		down := strings.HasPrefix(strings.ToLower(strings.TrimSpace(k.Order)), "d")
		keys = append(keys, idx)
		downs = append(downs, down)
		arrow := " ↑"
		if down {
			arrow = " ↓"
		}
		said = append(said, strings.TrimSpace(k.Column)+arrow)
	}
	sort.SliceStable(data, func(a, b int) bool {
		for ki := range keys {
			i := keys[ki]
			cmp := compareDisplay(data[a].cells[i].disp, data[b].cells[i].disp)
			if downs[ki] {
				cmp = -cmp
			}
			if cmp != 0 {
				return cmp < 0
			}
		}
		return false
	})
	writes := 0
	for i, rr := range data {
		newRow := coords[0] + 1 + i
		delta := newRow - (coords[0] + 1 + rr.orig)
		for ci, cv := range rr.cells {
			axis, _ := excelize.CoordinatesToCellName(coords[1]+ci, newRow)
			if cv.formula != "" {
				if err := f.SetCellFormula(op.Sheet, axis, strings.TrimPrefix(adjustRowRefs(cv.formula, delta), "=")); err != nil {
					return "", err
				}
			} else if n, err := strconv.ParseFloat(cv.raw, 64); err == nil && cv.raw != "" {
				if err := f.SetCellValue(op.Sheet, axis, n); err != nil {
					return "", err
				}
			} else if err := f.SetCellValue(op.Sheet, axis, cv.raw); err != nil {
				return "", err
			}
			if writes++; writes > maxCellsPerOp {
				return "", fmt.Errorf("排序区域过大（超过 %d 格）", maxCellsPerOp)
			}
		}
	}
	return fmt.Sprintf("排序 %d 行（%s）", len(data), strings.Join(said, "，")), nil
}

// sortScope 回执里的范围描述。
func sortScope(op Op) string {
	if op.Range != "" {
		return op.Range
	}
	return "整表"
}

// compareDisplay 显示值比较：两侧都能解析数字按数值比，否则文本比。
func compareDisplay(x, y string) int {
	nx, xok := parseLooseNumber(x)
	ny, yok := parseLooseNumber(y)
	if xok && yok {
		switch {
		case nx < ny:
			return -1
		case nx > ny:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(strings.ToLower(tidyText(x)), strings.ToLower(tidyText(y)))
}

// applyAddColumn 在数据区右侧（或 after 列后）加新列：首行写 header，数据行
// 逐行写 formula（为第 2 行而写、按行偏移，transform 同款）或 value。
// values:true 直接报错——excelize 无公式计算器，公式重算走保存后的 Recalc 链。
func applyAddColumn(f *excelize.File, op Op) (string, error) {
	if op.Header == "" {
		return "", fmt.Errorf("add_column 缺 header")
	}
	if op.Formula == "" && op.Value == nil {
		return "", fmt.Errorf("add_column 缺 formula/value")
	}
	if op.Values {
		return "", fmt.Errorf("values:true 需要公式重算，请用 formula 留公式，保存时自动重算")
	}
	rows, err := f.GetRows(op.Sheet)
	if err != nil {
		return "", err
	}
	lastRow := len(rows)
	newColNum := maxRowsWidth(rows) + 1
	inserted := ""
	if op.After != "" {
		an, err := excelize.ColumnNameToNumber(strings.TrimSpace(op.After))
		if err != nil {
			return "", fmt.Errorf("add_column after 列 %q 无效: %w", op.After, err)
		}
		next, err := excelize.ColumnNumberToName(an + 1)
		if err != nil {
			return "", err
		}
		if err := f.InsertCols(op.Sheet, next, 1); err != nil {
			return "", err
		}
		newColNum = an + 1
		inserted = "，插在 " + strings.TrimSpace(op.After) + " 后"
	}
	colName, err := excelize.ColumnNumberToName(newColNum)
	if err != nil {
		return "", err
	}
	if err := f.SetCellValue(op.Sheet, axisOf(newColNum, 1), op.Header); err != nil {
		return "", err
	}
	if lastRow < 2 {
		return fmt.Sprintf("加列 %s（%s 列%s，无数据行）", op.Header, colName, inserted), nil
	}
	for r := 2; r <= lastRow; r++ {
		axis, _ := excelize.CoordinatesToCellName(newColNum, r)
		if op.Formula != "" {
			formula := strings.TrimPrefix(adjustRowRefs(op.Formula, r-2), "=")
			if err := f.SetCellFormula(op.Sheet, axis, formula); err != nil {
				return "", err
			}
		} else {
			if err := f.SetCellValue(op.Sheet, axis, op.Value); err != nil {
				return "", err
			}
		}
		if op.NumberFormat != "" {
			if err := applyNumFmt(f, op.Sheet, axis, op.NumberFormat); err != nil {
				return "", err
			}
		}
		if r-1 > maxCellsPerOp {
			return "", fmt.Errorf("加列行数过多（超过 %d 格）", maxCellsPerOp)
		}
	}
	// 错误值检测：显示值以 # 开头（#NAME?/#REF! 等缓存错误）即报；新写公式
	// 尚无缓存值，重算链（Recalc）兜底再报。
	errN := 0
	var errSamples []string
	for r := 2; r <= lastRow; r++ {
		v, verr := f.GetCellValue(op.Sheet, axisOf(newColNum, r))
		if verr != nil || !strings.HasPrefix(v, "#") {
			continue
		}
		errN++
		if len(errSamples) < 3 {
			errSamples = append(errSamples, axisOf(newColNum, r)+" "+v)
		}
	}
	mode := "值"
	if op.Formula != "" {
		mode = "公式"
	}
	summary := fmt.Sprintf("加列 %s（%s 列%s，%d 行%s）", op.Header, colName, inserted, lastRow-1, mode)
	if errN > 0 {
		summary += fmt.Sprintf("，出错 %d 格（%s）", errN, strings.Join(errSamples, ", "))
	}
	return summary, nil
}

// applyCopyRange 跨表复制区域（保留原表、在副本上清洗场景）：逐格复制值。
// 缺省按原始值复制（数值保持数值类型，公式格取缓存值）；values:true 按
// 显示值复制。样式不复制——summary 如实注明「复制了值」。
func applyCopyRange(f *excelize.File, op Op) (string, error) {
	if op.Range == "" || op.To == "" {
		return "", fmt.Errorf("copy_range 缺 range/to")
	}
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	srcSheet := op.Sheet
	dstSheet := srcSheet
	dst := strings.TrimSpace(op.To)
	if i := strings.LastIndex(dst, "!"); i >= 0 {
		name := strings.Trim(strings.TrimSpace(dst[:i]), "'")
		dstSheet = name
		dst = strings.TrimSpace(dst[i+1:])
	}
	valid := false
	for _, s := range f.GetSheetList() {
		if s == dstSheet {
			valid = true
			break
		}
	}
	if !valid {
		return "", fmt.Errorf("copy_range 目标工作表 %q 不存在", dstSheet)
	}
	c1, r1, err := excelize.CellNameToCoordinates(dst)
	if err != nil {
		return "", fmt.Errorf("copy_range 目标单元格 %q 无效: %w", dst, err)
	}
	cells := (coords[2] - coords[0] + 1) * (coords[3] - coords[1] + 1)
	if cells > maxCellsPerOp {
		return "", fmt.Errorf("复制区域过大（超过 %d 格）", maxCellsPerOp)
	}
	for r := coords[0]; r <= coords[2]; r++ {
		for c := coords[1]; c <= coords[3]; c++ {
			srcAxis, _ := excelize.CoordinatesToCellName(c, r)
			dstAxis, _ := excelize.CoordinatesToCellName(c1+c-coords[1], r1+r-coords[0])
			value, err := copyCellValue(f, srcSheet, srcAxis, op.Values)
			if err != nil {
				return "", err
			}
			if err := f.SetCellValue(dstSheet, dstAxis, value); err != nil {
				return "", err
			}
		}
	}
	br, _ := excelize.CoordinatesToCellName(c1+coords[3]-coords[1], r1+coords[2]-coords[0])
	mode := "原始值"
	if op.Values {
		mode = "显示值"
	}
	return fmt.Sprintf("复制 %s!%s → %s!%s（%d 格，复制了值/%s；不含公式与样式）",
		srcSheet, op.Range, dstSheet, dst+":"+br, cells, mode), nil
}

// copyCellValue 读源格要复制的值：缺省按原始值保类型（数值→float64）；
// 公式格无独立原始值，取显示值（数值形态保数值）。values=true 按显示值。
func copyCellValue(f *excelize.File, sheet, axis string, display bool) (interface{}, error) {
	if display {
		disp, err := f.GetCellValue(sheet, axis)
		if err != nil {
			return nil, err
		}
		if n, perr := strconv.ParseFloat(strings.TrimSpace(disp), 64); perr == nil && disp != "" {
			return n, nil
		}
		return disp, nil
	}
	formula, _ := f.GetCellFormula(sheet, axis)
	if formula != "" {
		disp, err := f.GetCellValue(sheet, axis)
		if err != nil {
			return nil, err
		}
		if n, perr := strconv.ParseFloat(strings.TrimSpace(disp), 64); perr == nil && disp != "" {
			return n, nil
		}
		return disp, nil
	}
	raw, err := f.GetCellValue(sheet, axis, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, err
	}
	if n, perr := strconv.ParseFloat(raw, 64); perr == nil && raw != "" {
		return n, nil
	}
	return raw, nil
}
