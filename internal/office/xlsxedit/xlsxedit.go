// Package xlsxedit 提供 Excel 单元格级编辑：AI 生成的 JSON 操作集在 excelize
// 上执行（公式/常量填充/列变换/清洗/拆分/替换），再用 LibreOffice 重算公式，
// 供预览「结果可检查、公式可编辑」。
package xlsxedit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/proc"
	"github.com/xuri/excelize/v2"
)

// Op 是 AI 规划的一个单元格操作（字段按类型复用）。
type Op struct {
	Type    string      `json:"type"`
	Sheet   string      `json:"sheet,omitempty"`
	Target  string      `json:"target,omitempty"` // "B4"
	Range   string      `json:"range,omitempty"`  // "A1:B10"
	Value   interface{} `json:"value,omitempty"`
	With    interface{} `json:"with,omitempty"` // fill_blanks 填充值（值或 above/mean/median/mode）
	Formula string      `json:"formula,omitempty"`
	Find    string      `json:"find,omitempty"`
	Replace string      `json:"replace,omitempty"`
	Col     string      `json:"col,omitempty"`     // "A"
	Sep     string      `json:"sep,omitempty"`     // 拆分分隔符
	NewCols []string    `json:"newCols,omitempty"` // ["B","C"]
	Headers []string    `json:"headers,omitempty"`
	Trim    bool        `json:"trim,omitempty"`
	Upper   bool        `json:"upper,omitempty"`
	Lower   bool        `json:"lower,omitempty"`
	Style   *Style      `json:"style,omitempty"` // set_style 样式载荷
	Width   float64     `json:"width,omitempty"` // set_col_width 列宽

	// ── 波2 扩面（dsh-office「Excel: data work」差异化能力移植；全部
	// omitempty 向后兼容，旧 ops JSON 不受影响）──
	Remove       []string          `json:"remove,omitempty"`       // clean：删除指定字样
	NoSpaces     bool              `json:"noSpaces,omitempty"`     // clean：去全部空格
	Map          map[string]string `json:"map,omitempty"`          // clean：取值改名（旧值→新值）
	To           string            `json:"to,omitempty"`           // clean："number"|"date"|"text"；copy_range：目标 "清洗后!A1"
	Columns      []string          `json:"columns,omitempty"`      // remove_duplicates：判重列（缺省整行）
	Where        *Where            `json:"where,omitempty"`        // delete_rows：条件删行（与 blank/row 互斥）
	Blank        bool              `json:"blank,omitempty"`        // delete_rows：整行全空才删
	Row          int               `json:"row,omitempty"`          // insert_rows/delete_rows：起始行号
	Count        int               `json:"count,omitempty"`        // 行/列数量（缺省 1）
	By           []SortKey         `json:"by,omitempty"`           // sort：多键
	Header       string            `json:"header,omitempty"`       // add_column：新列列名
	After        string            `json:"after,omitempty"`        // add_column：插在该列后（缺省数据区右侧）
	NumberFormat string            `json:"numberFormat,omitempty"` // add_column：新列数字格式
	Values       bool              `json:"values,omitempty"`       // copy_range：按显示值复制；add_column：true 即报错（见 applyAddColumn）
}

// Where 是 delete_rows 的删行条件：col 列满足 is 条件即删该行。
type Where struct {
	Col   string      `json:"col"`          // "B"
	Is    string      `json:"is,omitempty"` // blank|=|<>|>|<|>=|<=|contains|text（缺省 blank；text=写成文本的数字或日期）
	Value interface{} `json:"value,omitempty"`
}

// SortKey 是 sort 的单键：按 column 排序，order 为 asc（缺省）或 desc。
type SortKey struct {
	Column string `json:"column"`
	Order  string `json:"order,omitempty"`
}

// Style 是 set_style 的样式载荷：指针/空串字段表示「不改」，叠加到单元格
// 现有样式上（例如只加粗不会丢掉已有填充色）。
type Style struct {
	Bold      *bool    `json:"bold,omitempty"`
	Italic    *bool    `json:"italic,omitempty"`
	Underline *bool    `json:"underline,omitempty"`
	FontSize  *float64 `json:"fontSize,omitempty"`
	FontColor string   `json:"fontColor,omitempty"` // "RRGGBB"（可带 #）
	Fill      string   `json:"fill,omitempty"`      // 纯色填充 "RRGGBB"
	NumFmt    string   `json:"numFmt,omitempty"`    // 自定义数字格式，如 "0.00%"
	Align     string   `json:"align,omitempty"`     // left | center | right
	Wrap      *bool    `json:"wrap,omitempty"`
}

// Context 是发送给 AI 的表格上下文（含表头与抽样数据）。
type Context struct {
	Sheets  []string   `json:"sheets"`
	Active  string     `json:"active"`
	Dims    string     `json:"dims"`
	Headers []string   `json:"headers"`
	Sample  [][]string `json:"sample"`
}

// BuildContext 用 excelize 打开文件并生成指定 sheet 的上下文 JSON。
func BuildContext(path, sheet string) (string, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return "", fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	ctx := Context{Sheets: sheets, Active: sheet}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return "", fmt.Errorf("读取工作表 %q 失败: %w", sheet, err)
	}
	if dim, err := f.GetSheetDimension(sheet); err == nil {
		ctx.Dims = dim
	}
	if len(rows) > 0 {
		ctx.Headers = trimCells(rows[0])
	}
	for i, r := range rows {
		if i == 0 || len(ctx.Sample) >= sampleRows {
			continue
		}
		ctx.Sample = append(ctx.Sample, trimCells(r))
	}
	b, err := json.Marshal(ctx)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func trimCells(row []string) []string {
	out := make([]string, 0, len(row))
	for _, v := range row {
		if len(v) > 60 {
			v = v[:60] + "…"
		}
		out = append(out, v)
	}
	return out
}

// ApplyOps 在 path 上执行操作集并保存。返回逐条应用摘要。
func ApplyOps(path string, ops []Op) ([]string, error) {
	if len(ops) == 0 {
		return nil, fmt.Errorf("没有可执行的操作")
	}
	if len(ops) > maxOps {
		return nil, fmt.Errorf("操作数量超限（最多 %d 条）", maxOps)
	}
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	validSheets := map[string]bool{}
	for _, name := range f.GetSheetList() {
		validSheets[name] = true
	}

	var summary []string
	for i, op := range ops {
		if op.Sheet == "" {
			op.Sheet = f.GetSheetList()[0]
		}
		if !validSheets[op.Sheet] {
			return nil, fmt.Errorf("操作 %d：工作表 %q 不存在", i+1, op.Sheet)
		}
		desc, err := applyOne(f, op)
		if err != nil {
			return nil, fmt.Errorf("操作 %d（%s）: %w", i+1, op.Type, err)
		}
		summary = append(summary, desc)
	}
	if err := f.SaveAs(path); err != nil {
		return nil, fmt.Errorf("保存 xlsx 失败: %w", err)
	}
	return summary, nil
}

// ── 先规划后应用：临时副本试运行 + 单元格级 diff ──────────────

// Change 是规划 diff 中的一处单元格变更（值或公式与原文件不同）。
type Change struct {
	Sheet   string `json:"sheet"`
	Cell    string `json:"cell"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Formula string `json:"formula,omitempty"` // 变更后为公式时给出（前端显示 fx）
}

const (
	maxOps          = 20
	maxCellsPerOp   = 10000
	sampleRows      = 15
	maxPlanChanges  = 200   // 变更清单条数上限（超出仍计数）
	maxPlanDiffRead = 20000 // diff 单元格读取预算（超出按受影响格计数并截断）
)

// PlanOps 在原文件的临时副本上试运行操作集：不触碰原文件，返回逐条操作
// 描述（与真实执行一致）与单元格级变更清单。副本执行失败即规划失败
// （操作集非法，如工作表不存在），原文件保持原样。
func PlanOps(path string, ops []Op) (summary []string, changes []Change, total int, truncated bool, err error) {
	if len(ops) == 0 {
		return nil, nil, 0, false, fmt.Errorf("没有可执行的操作")
	}
	if len(ops) > maxOps {
		return nil, nil, 0, false, fmt.Errorf("操作数量超限（最多 %d 条）", maxOps)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gaea-plan-*.xlsx")
	if err != nil {
		return nil, nil, 0, false, fmt.Errorf("创建规划副本失败: %w", err)
	}
	tmpName := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpName) }()
	if err := copyFile(path, tmpName); err != nil {
		return nil, nil, 0, false, fmt.Errorf("复制规划副本失败: %w", err)
	}
	summary, aerr := ApplyOps(tmpName, ops)
	if aerr != nil {
		return nil, nil, 0, false, aerr
	}
	changes, total, truncated, derr := diffOps(path, tmpName, ops)
	if derr != nil {
		return nil, nil, 0, false, derr
	}
	return summary, changes, total, truncated, nil
}

// diffOps 对比原文件与试运行副本，产出操作集影响范围内的单元格级变更清单。
func diffOps(origPath, tmpPath string, ops []Op) ([]Change, int, bool, error) {
	orig, err := excelize.OpenFile(origPath, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, 0, false, fmt.Errorf("打开原文件失败: %w", err)
	}
	defer func() { _ = orig.Close() }()
	tmp, err := excelize.OpenFile(tmpPath, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, 0, false, fmt.Errorf("打开副本失败: %w", err)
	}
	defer func() { _ = tmp.Close() }()

	sheetList := tmp.GetSheetList()
	first := ""
	if len(sheetList) > 0 {
		first = sheetList[0]
	}
	var changes []Change
	total := 0
	truncated := false
	reads := 0
	for _, op := range ops {
		sheet := op.Sheet
		if sheet == "" {
			sheet = first
		}
		for _, cell := range opCells(tmp, sheet, op) {
			if len(changes) >= maxPlanChanges || reads >= maxPlanDiffRead {
				// 超出清单/读取预算：停止读取，total 停留在已确认的变更数（下界）
				truncated = true
				continue
			}
			reads++
			before, _ := orig.GetCellValue(sheet, cell)
			after, _ := tmp.GetCellValue(sheet, cell)
			origFormula, _ := orig.GetCellFormula(sheet, cell)
			formula, _ := tmp.GetCellFormula(sheet, cell)
			if before == after && formula == origFormula {
				continue
			}
			total++
			changes = append(changes, Change{
				Sheet: sheet, Cell: cell, Before: before, After: after, Formula: formula,
			})
		}
	}
	return changes, total, truncated, nil
}

// opCells 列出操作影响「存储值/公式」的单元格（与 applyOne 的写入范围一致；
// 纯样式/列宽不影响值，返回 nil —— diff 只呈现值与公式变更，样式由预览呈现）。
func opCells(f *excelize.File, sheet string, op Op) []string {
	switch op.Type {
	case "set_formula", "set_value":
		if op.Target == "" {
			return nil
		}
		return []string{op.Target}
	case "fill_range", "transform", "replace", "clean":
		if op.Range == "" {
			return nil
		}
		coords, err := parseRange(op.Range)
		if err != nil {
			return nil
		}
		var cells []string
		for r := coords[0]; r <= coords[2]; r++ {
			for c := coords[1]; c <= coords[3]; c++ {
				axis, _ := excelize.CoordinatesToCellName(c, r)
				cells = append(cells, axis)
			}
		}
		return cells
	case "split_column":
		if len(op.NewCols) == 0 {
			return nil
		}
		rows, err := f.GetRows(sheet)
		if err != nil {
			return nil
		}
		last := len(rows)
		if last < 1 {
			last = 1
		}
		var cells []string
		for _, col := range op.NewCols {
			for r := 1; r <= last; r++ {
				cells = append(cells, col+strconv.Itoa(r))
			}
		}
		return cells
	case "merge_cells", "unmerge_cells":
		// 合并会清掉被覆盖格的存储值，纳入 diff 检查
		if op.Range == "" {
			return nil
		}
		coords, err := parseRange(op.Range)
		if err != nil {
			return nil
		}
		var cells []string
		for r := coords[0]; r <= coords[2]; r++ {
			for c := coords[1]; c <= coords[3]; c++ {
				axis, _ := excelize.CoordinatesToCellName(c, r)
				cells = append(cells, axis)
			}
		}
		return cells
	case "fill_blanks":
		// 单列区域（col 或 range）的数据行
		colNum := 0
		r1, r2 := 0, 0
		if op.Col != "" {
			n, err := excelize.ColumnNameToNumber(op.Col)
			if err != nil {
				return nil
			}
			rows, err := f.GetRows(sheet)
			if err != nil {
				return nil
			}
			colNum, r1, r2 = n, 2, len(rows)
		} else if op.Range != "" {
			coords, err := parseRange(op.Range)
			if err != nil {
				return nil
			}
			colNum, r1, r2 = coords[1], coords[0], coords[2]
		} else {
			return nil
		}
		var cells []string
		for r := r1; r <= r2; r++ {
			axis, _ := excelize.CoordinatesToCellName(colNum, r)
			cells = append(cells, axis)
		}
		return cells
	case "remove_duplicates", "delete_rows", "insert_rows", "sort",
		"insert_columns", "delete_columns":
		// 行列位移类：整块 used range 纳入 diff 检查（内存与读取预算双封顶：
		// 追加到 maxPlanDiffRead 即停，超出部分由 diffOps 计数截断）
		coords, err := usedRangeCoords(f, sheet)
		if err != nil {
			return nil
		}
		var cells []string
		for r := coords[0]; r <= coords[2]; r++ {
			for c := coords[1]; c <= coords[3]; c++ {
				axis, _ := excelize.CoordinatesToCellName(c, r)
				cells = append(cells, axis)
				if len(cells) >= maxPlanDiffRead {
					return cells
				}
			}
		}
		return cells
	case "add_column":
		// 新列整体（表头+数据行）
		rows, err := f.GetRows(sheet)
		if err != nil {
			return nil
		}
		newColNum := maxRowsWidth(rows) + 1
		if op.After != "" {
			if an, err := excelize.ColumnNameToNumber(op.After); err == nil {
				newColNum = an + 1
			}
		}
		last := len(rows)
		if last < 1 {
			last = 1
		}
		var cells []string
		for r := 1; r <= last; r++ {
			axis, _ := excelize.CoordinatesToCellName(newColNum, r)
			cells = append(cells, axis)
		}
		return cells
	case "copy_range":
		// 目标区域（仅同表目标可纳入 diff——diffOps 以 op 表为准逐格读，
		// 跨表目标的变更清单暂缺，由 summary/预览呈现实况）
		if op.To == "" || op.Range == "" {
			return nil
		}
		dst := strings.TrimSpace(op.To)
		if i := strings.LastIndex(dst, "!"); i >= 0 {
			if strings.Trim(strings.TrimSpace(dst[:i]), "'") != sheet {
				return nil
			}
			dst = strings.TrimSpace(dst[i+1:])
		}
		coords, err := parseRange(op.Range)
		if err != nil {
			return nil
		}
		c1, r1, err := excelize.CellNameToCoordinates(dst)
		if err != nil {
			return nil
		}
		var cells []string
		for r := 0; r <= coords[2]-coords[0]; r++ {
			for c := 0; c <= coords[3]-coords[1]; c++ {
				axis, _ := excelize.CoordinatesToCellName(c1+c, r1+r)
				cells = append(cells, axis)
				if len(cells) >= maxPlanDiffRead {
					return cells
				}
			}
		}
		return cells
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// applyOne 分派单个 op（实现体按 Op.Type 拆在 apply_one.go 的 opHandlers
// 注册表，键集由 apply_one_test.go 冻结；unknown 类型 fail-closed）。
func applyOne(f *excelize.File, op Op) (string, error) {
	if h, ok := opHandlers[op.Type]; ok {
		return h(f, op)
	}
	return "", fmt.Errorf("不支持的操作类型 %q", op.Type)
}

// mergeStyle 把请求的样式字段叠加到现有样式上（未给定的字段保留原值），
// 生成新样式 ID。existingID 为 0 表示无现有样式。
func mergeStyle(f *excelize.File, existingID int, s *Style) (int, error) {
	st := &excelize.Style{}
	if existingID > 0 {
		if old, err := f.GetStyle(existingID); err == nil && old != nil {
			cp := *old // 只覆盖标量/替换整块字段，Border 等原样保留
			st = &cp
		}
	}
	needFont := s.Bold != nil || s.Italic != nil || s.Underline != nil || s.FontSize != nil || s.FontColor != ""
	if needFont {
		font := st.Font
		if font == nil {
			font = &excelize.Font{}
		}
		if s.Bold != nil {
			font.Bold = *s.Bold
		}
		if s.Italic != nil {
			font.Italic = *s.Italic
		}
		if s.Underline != nil {
			if *s.Underline {
				font.Underline = "single"
			} else {
				font.Underline = ""
			}
		}
		if s.FontSize != nil {
			font.Size = *s.FontSize
		}
		if s.FontColor != "" {
			c, err := normalizeHex(s.FontColor)
			if err != nil {
				return 0, fmt.Errorf("fontColor 无效: %w", err)
			}
			font.Color = c
		}
		st.Font = font
	}
	if s.Fill != "" {
		c, err := normalizeHex(s.Fill)
		if err != nil {
			return 0, fmt.Errorf("fill 无效: %w", err)
		}
		st.Fill = excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{c}}
	}
	if s.NumFmt != "" {
		nf := s.NumFmt
		st.CustomNumFmt = &nf
	}
	if s.Align != "" || s.Wrap != nil {
		al := st.Alignment
		if al == nil {
			al = &excelize.Alignment{}
		}
		if s.Align != "" {
			al.Horizontal = s.Align
		}
		if s.Wrap != nil {
			al.WrapText = *s.Wrap
		}
		st.Alignment = al
	}
	return f.NewStyle(st)
}

// normalizeHex 归一颜色："#rrggbb" / "RRGGBB" → "RRGGBB"（也接受 8 位 ARGB）。
func normalizeHex(c string) (string, error) {
	c = strings.ToUpper(strings.TrimPrefix(strings.TrimSpace(c), "#"))
	if len(c) != 6 && len(c) != 8 {
		return "", fmt.Errorf("应为 RRGGBB，得到 %q", c)
	}
	for _, ch := range c {
		if !(ch >= '0' && ch <= '9' || ch >= 'A' && ch <= 'F') {
			return "", fmt.Errorf("非十六进制字符 %q", ch)
		}
	}
	return c, nil
}

// parseRange 解析 "A1:B10" 为 (r1,c1,r2,c2)。
func parseRange(r string) ([4]int, error) {
	parts := strings.Split(r, ":")
	if len(parts) != 2 {
		return [4]int{}, fmt.Errorf("区域格式应为 A1:B10，得到 %q", r)
	}
	c1, r1, err := excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return [4]int{}, fmt.Errorf("区域起点无效 %q", parts[0])
	}
	c2, r2, err := excelize.CellNameToCoordinates(parts[1])
	if err != nil {
		return [4]int{}, fmt.Errorf("区域终点无效 %q", parts[1])
	}
	return [4]int{r1, c1, r2, c2}, nil
}

var cellRefRe = regexp.MustCompile(`(?i)([A-Z]{1,3})(\d+)`)

// adjustRowRefs 把公式模板中的相对行引用按 delta 行平移（列不变，$ 绝对引用不动）。
func adjustRowRefs(formula string, delta int) string {
	var b strings.Builder
	last := 0
	for _, loc := range cellRefRe.FindAllStringSubmatchIndex(formula, -1) {
		start, end := loc[0], loc[1]
		// 跳过 $ 绝对引用（$B2 / B$2 / $B$2）
		if start > 0 && formula[start-1] == '$' {
			continue
		}
		// 跳过前面紧跟字母的匹配（函数名如 LOG10 / ROUND 等误判）
		if start > 0 {
			prev := formula[start-1]
			if (prev >= 'A' && prev <= 'Z') || (prev >= 'a' && prev <= 'z') {
				continue
			}
		}
		// 后面紧跟 ( 的是函数名，不是单元格引用
		if end < len(formula) && formula[end] == '(' {
			continue
		}
		row, err := strconv.Atoi(formula[loc[4]:loc[5]])
		if err != nil {
			continue
		}
		b.WriteString(formula[last:start])
		b.WriteString(formula[loc[2]:loc[3]]) // 列字母
		b.WriteString(strconv.Itoa(row + delta))
		last = end
	}
	b.WriteString(formula[last:])
	return b.String()
}

// ── LibreOffice 重算 ────────────────────────────────────────

// RecalcReport 是 recalc.py 的 JSON 输出。
type RecalcReport struct {
	Status        string                 `json:"status"`
	TotalErrors   int                    `json:"total_errors"`
	TotalFormulas int                    `json:"total_formulas"`
	ErrorSummary  map[string]interface{} `json:"error_summary"`
	Error         string                 `json:"error"`
}

// Recalc 调用技能环境里的 recalc.py（LibreOffice 宏重算公式并扫描错误）。
// workspaceRoot 用于定位 .gaea/skills/xlsx/scripts/recalc.py。
func Recalc(path, workspaceRoot string) (RecalcReport, error) {
	script := findRecalcScript(workspaceRoot)
	if script == "" {
		return RecalcReport{}, fmt.Errorf("未找到 recalc.py（.gaea/skills/xlsx/scripts）")
	}
	cmd := exec.Command("python", script, path, "60")
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	out, err := cmd.CombinedOutput()
	if err != nil {
		return RecalcReport{}, fmt.Errorf("重算失败: %v（%s）", err, truncate(string(out), 300))
	}
	var rep RecalcReport
	if err := json.Unmarshal(out, &rep); err != nil {
		return RecalcReport{}, fmt.Errorf("重算输出解析失败: %v（%s）", err, truncate(string(out), 300))
	}
	if rep.Error != "" {
		return rep, fmt.Errorf("重算出错: %s", rep.Error)
	}
	return rep, nil
}

func findRecalcScript(workspaceRoot string) string {
	candidates := []string{
		filepath.Join(workspaceRoot, ".gaea", "skills", "xlsx", "scripts", "recalc.py"),
		filepath.Join(workspaceRoot, "skills", "xlsx", "scripts", "recalc.py"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".codex", "skills", "xlsx", "scripts", "recalc.py"),
			filepath.Join(home, ".gaea", "skills", "xlsx", "scripts", "recalc.py"),
		)
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
