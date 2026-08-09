// Package xlsxedit 提供 Excel 单元格级编辑：AI 生成的 JSON 操作集在 excelize
// 上执行（公式/常量填充/列变换/清洗/拆分/替换），再用 LibreOffice 重算公式，
// 供预览「结果可检查、公式可编辑」。
package xlsxedit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Op 是 AI 规划的一个单元格操作（字段按类型复用）。
type Op struct {
	Type    string      `json:"type"`
	Sheet   string      `json:"sheet,omitempty"`
	Target  string      `json:"target,omitempty"` // "B4"
	Range   string      `json:"range,omitempty"`  // "A1:B10"
	Value   interface{} `json:"value,omitempty"`
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
}

const (
	maxOps          = 20
	maxCellsPerOp   = 10000
	sampleRows      = 15
)

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
	defer f.Close()

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
	defer f.Close()

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

func applyOne(f *excelize.File, op Op) (string, error) {
	switch op.Type {
	case "set_formula":
		if op.Target == "" || op.Formula == "" {
			return "", fmt.Errorf("target/formula 缺失")
		}
		if err := f.SetCellFormula(op.Sheet, op.Target, strings.TrimPrefix(op.Formula, "=")); err != nil {
			return "", err
		}
		return fmt.Sprintf("写入公式 %s=%s", op.Target, op.Formula), nil

	case "set_value":
		if op.Target == "" {
			return "", fmt.Errorf("target 缺失")
		}
		if err := f.SetCellValue(op.Sheet, op.Target, op.Value); err != nil {
			return "", err
		}
		return fmt.Sprintf("写入值 %s=%v", op.Target, op.Value), nil

	case "fill_range":
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

	case "transform":
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

	case "replace":
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

	case "split_column":
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

	case "clean":
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
				orig := val
				if op.Trim {
					val = strings.TrimSpace(val)
				}
				if op.Upper {
					val = strings.ToUpper(val)
				}
				if op.Lower {
					val = strings.ToLower(val)
				}
				if val != orig {
					if err := f.SetCellValue(op.Sheet, axis, val); err != nil {
						return "", err
					}
					count++
				}
			}
		}
		return fmt.Sprintf("清洗 %s（%d 格）", op.Range, count), nil

	default:
		return "", fmt.Errorf("不支持的操作类型 %q", op.Type)
	}
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
	out, err := exec.Command("python", script, path, "60").CombinedOutput()
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
