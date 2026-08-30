package app

// v4.9.1 Verifier 通道 A 引用级深化（审计欠账收口）：xlsx_apply 在「公式重算
// 零错误」之上增加「声明 ↔ 实况」引用级比对——证据卡 v4.9.1 起随卡落盘
// opsJson（机器可读 op 载荷），复核时逐条回读目标工作簿：
//   - set_value：单元格实际值与声明值一致（数值按浮点等值比较，容差 1e-9）；
//   - set_formula：单元格实际公式与声明公式一致（去空白后精确比较）；
//   - replace：替换单元格包含 Replace 且不再包含 Find。
// 其余 op 类型（fill_range/transform/split_column/set_style 等批量或样式类）
// 不可单格核对，计入 skipped（诚实降级，宁漏勿误）。旧证据卡无 opsJson →
// 声明比对不适用。全簿公式评估错误（含断链 #REF! 的求值结果）由既有 recalc
// 重算承担，本层不做全簿公式扫描（避免字符串级误报）。

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/xuri/excelize/v2"
)

// verifyClaimMaxOps 声明比对逐条核对的 op 上限（防超卡拖慢复核）。
const verifyClaimMaxOps = 32

// verifyXlsxChannelADeep 重算零错误后的引用级「声明↔实况」比对。返回追加在
// 通道 A 文案之后的复核结论（fail 前缀会升级整条通道 A 结论）。
func verifyXlsxChannelADeep(f *excelize.File, rec evidence.ChangeRecord) string {
	if strings.TrimSpace(rec.OpsJSON) == "" {
		return "；旧证据卡无 opsJson，声明比对不适用"
	}
	var ops []xlsxedit.Op
	if err := json.Unmarshal([]byte(rec.OpsJSON), &ops); err != nil {
		return "；warn: opsJson 解析失败，跳过声明比对"
	}
	if len(ops) > verifyClaimMaxOps {
		ops = ops[:verifyClaimMaxOps]
	}
	checked, mismatched, skipped := 0, 0, 0
	var examples []string
	noteMismatch := func(s string) {
		if len(examples) < 3 {
			examples = append(examples, s)
		}
	}

	for _, op := range ops {
		if op.Sheet == "" {
			skipped++
			continue
		}
		switch op.Type {
		case "set_value":
			actual, err := f.GetCellValue(op.Sheet, op.Target)
			if err != nil {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 不可读（%v）", op.Sheet, op.Target, err))
				continue
			}
			if !cellValueEquals(fmt.Sprint(op.Value), actual) {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 预期「%s」实际「%s」", op.Sheet, op.Target, fmt.Sprint(op.Value), actual))
				continue
			}
			checked++
		case "set_formula":
			actual, err := f.GetCellFormula(op.Sheet, op.Target)
			if err != nil {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 公式不可读（%v）", op.Sheet, op.Target, err))
				continue
			}
			if compactFormula(actual) != compactFormula(fmt.Sprint(op.Formula)) {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 公式预期「%s」实际「%s」", op.Sheet, op.Target, op.Formula, actual))
				continue
			}
			checked++
		case "replace":
			if op.Target == "" || op.Find == "" {
				skipped++
				continue
			}
			actual, err := f.GetCellValue(op.Sheet, op.Target)
			if err != nil {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 不可读（%v）", op.Sheet, op.Target, err))
				continue
			}
			if !strings.Contains(actual, op.Replace) || strings.Contains(actual, op.Find) {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 替换未如实落盘（实际「%s」）", op.Sheet, op.Target, actual))
				continue
			}
			checked++
		default:
			skipped++
		}
	}

	switch {
	case mismatched > 0:
		return fmt.Sprintf("；fail: 引用级复核发现 %d 项声明与实况不符（%s）",
			mismatched, strings.Join(examples, "；"))
	case checked == 0:
		return fmt.Sprintf("；引用级声明比对无可核对项（跳过 %d 项批量/样式类）", skipped)
	default:
		suffix := ""
		if skipped > 0 {
			suffix = fmt.Sprintf("，跳过 %d 项批量/样式类", skipped)
		}
		return fmt.Sprintf("；引用级复核 %d 项声明与实况一致%s", checked, suffix)
	}
}

// cellValueEquals 声明值与单元格实际值等值比较：两侧都能解析为数值时按浮点
// 等值（42 ≈ 42.0）；否则按去空白字符串精确比较。
func cellValueEquals(expected, actual string) bool {
	e, eerr := strconv.ParseFloat(strings.TrimSpace(expected), 64)
	a, aerr := strconv.ParseFloat(strings.TrimSpace(actual), 64)
	if eerr == nil && aerr == nil {
		return math.Abs(e-a) <= 1e-9*math.Max(1, math.Abs(e))
	}
	return strings.TrimSpace(expected) == strings.TrimSpace(actual)
}

// compactFormula 公式归一（去前导 = 与空白）——applyOne 落盘时剥掉 "=" 前缀，
// excelize 存储规范化（空格、参数间隔）也可能与声明原文有无害差异。
func compactFormula(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "=")
	return strings.ReplaceAll(s, " ", "")
}
