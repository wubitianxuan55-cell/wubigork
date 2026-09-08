// Package cost — LLM 拆解结果合理性校验(v4.158.0 AI 组价复核闭环)。
//
// CheckComposeComponents 是纯函数:无 IO、无状态,只依赖入参。在
// GaeaCostCompose 拆解后立即计算,结论随 CostComposeView.Checks 一并展示与
// 留痕,让「AI 拆出来的含量×单价是否自洽」在确认前就摆在人眼前。
package cost

import (
	"fmt"
	"math"
)

// ComposeCheck 合理性校验结论一条。
//   - Level:"warn"=数据可疑需人工修正;"info"=口径提醒(不阻断确认)。
//   - Row:组件行序号(Components 下标);-1=全局结论。
type ComposeCheck struct {
	Level string `json:"level"` // "warn" | "info"
	Row   int    `json:"row"`   // 组件行序号;-1=全局
	Msg   string `json:"msg"`
}

// CheckComposeComponents 校验人材机组件金额合理性,规则逐条钉死:
//
//	R1 金额一致性:|amount − quantity×price| > max(0.01, |amount|×0.01)
//	  → warn(该行)「金额 %.2f ≠ 含量×单价 %.2f」
//	R2 非正值:quantity<=0 或 price<=0 → warn(该行)「含量/单价须为正」;
//	  该行视为非法:跳过 R1(负含量下 quantity×price 无意义),且不计入 R3 合计
//	R3 全局合计(Σamount,过滤非法行后):
//	  Σ > recommended×1.05 → warn(-1)「人材机合计 %.2f 已超推荐价 %.2f」
//	  Σ < recommended×0.5  → info(-1)「人材机合计仅为推荐价 %.0f%%,注意管理费/利润/税金口径」
//	  recommended<=0 时推荐价本身无效,跳过全局两档(只留行级结论)
//
// 顺序稳定:先行级规则按行序(每行先 R2 后 R1),再全局。comps 空返回 nil。
func CheckComposeComponents(comps []Component, recommended float64) []ComposeCheck {
	if len(comps) == 0 {
		return nil
	}
	var checks []ComposeCheck
	var total float64
	for i, c := range comps {
		if c.Quantity <= 0 || c.Price <= 0 {
			checks = append(checks, ComposeCheck{Level: "warn", Row: i, Msg: "含量/单价须为正"})
			continue // 非法行:跳过 R1,不计入合计
		}
		// R1 容差:max(0.01, |amount|×0.01)——小额看绝对差,大额看 1% 相对差。
		tol := math.Max(0.01, math.Abs(c.Amount)*0.01)
		if diff := math.Abs(c.Amount - c.Quantity*c.Price); diff > tol {
			checks = append(checks, ComposeCheck{Level: "warn", Row: i, Msg: fmt.Sprintf("金额 %.2f ≠ 含量×单价 %.2f", c.Amount, c.Quantity*c.Price)})
		}
		total += c.Amount
	}
	if recommended > 0 {
		if total > recommended*1.05 {
			checks = append(checks, ComposeCheck{Level: "warn", Row: -1, Msg: fmt.Sprintf("人材机合计 %.2f 已超推荐价 %.2f", total, recommended)})
		} else if total < recommended*0.5 {
			checks = append(checks, ComposeCheck{Level: "info", Row: -1, Msg: fmt.Sprintf("人材机合计仅为推荐价 %.0f%%,注意管理费/利润/税金口径", total/recommended*100)})
		}
	}
	return checks
}
