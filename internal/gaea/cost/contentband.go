// Package cost — 同类条目含量对照(v4.209.0 造价刀路池 §6 收官)。
//
// CheckContentBaseline 是纯函数:把 AI 拆解(或手录)的人材机组件含量,与同类
// 条目池的同一资源含量分布对照,带外给 warn。基线源=用户自己的成本库(私人
// 记忆哲学):行业含量区间等外部数据源不引入,同类样本不足时诚实不比对。
// 与 CheckComposeComponents 互补:那边管「金额自洽」,这边管「含量与同类相比
// 是否离谱」——结论同用 ComposeCheck 承载,组价链零新结构零新绑定。
package cost

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// MinContentSamples 同一资源参与对照的最少同类样本数;低于该值不比对
// (两三例凑不出规律,宁缺勿误)。
const MinContentSamples = 3

// normCompTitle 组件标题匹配键:全角→半角、去全部空白、转小写。
// 「C32.5 水泥」「ｃ３２．５水泥」「c32.5  水泥」归一后同键;空串原样返回。
func normCompTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ' || r == '\u00a0' || r == '\u3000' || r == '\t':
			continue
		case r >= '！' && r <= '～': // 全角 ASCII 区 → 半角
			r -= 0xFEE0
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// normCompUnit 组件单位匹配键:去空白+小写(「m³/M³/m3」的语义差异不在本刀
// 范围,先做粗归一;双方都为空视为可比——导入件单位缺失常见,不因缺失放弃对照)。
func normCompUnit(s string) string {
	return normCompTitle(s)
}

// CheckContentBaseline 含量对照:目标组件逐行与池内同标题同单位资源的含量
// 分位带比较,带外产出 warn(带内不产生结论,降噪)。规则钉死:
//
//	匹配键:标题归一精确相等 + 单位归一相等(含双方都空);一方缺失一律不比
//	  ——kg 与 t、未知与已知不可比,宁缺勿误(与导入链「带码未命中宁新增勿
//	  误配」同一哲学);
//	分位带:同类样本 P25/中位/P75(R-7,percentile 同口径);
//	带外:q > P75 → 偏高 warn;q < P25 → 偏低 warn;P25==P75(样本同值)时
//	  按 ±5% 容差判带外,避免完全同值样本把任何微小差异标异常;
//	目标行 Quantity<=0 跳过(非法行归 CheckComposeComponents R2 管);
//	同标题同单位样本 < MinContentSamples 不比对(诚实不足,与「无对照」同为
//	  静默——本函数只在有把握时开口)。
//
// 顺序稳定:按目标组件行序输出。pool 为空/comps 为空返回 nil。
func CheckContentBaseline(comps []Component, pool [][]Component) []ComposeCheck {
	if len(comps) == 0 || len(pool) == 0 {
		return nil
	}
	// 池内按归一(标题,单位)聚合含量样本。
	type key struct{ title, unit string }
	samples := make(map[key][]float64)
	for _, comps2 := range pool {
		for _, c := range comps2 {
			if c.Quantity <= 0 {
				continue
			}
			t := normCompTitle(c.Title)
			if t == "" {
				continue
			}
			k := key{t, normCompUnit(c.Unit)}
			samples[k] = append(samples[k], c.Quantity)
		}
	}
	var out []ComposeCheck
	for i, c := range comps {
		if c.Quantity <= 0 {
			continue
		}
		t := normCompTitle(c.Title)
		if t == "" {
			continue
		}
		qs := samples[key{t, normCompUnit(c.Unit)}]
		if len(qs) < MinContentSamples {
			continue
		}
		sort.Float64s(qs)
		p25, p75 := percentile(qs, 0.25), percentile(qs, 0.75)
		outside := c.Quantity > p75 || c.Quantity < p25
		if p25 == p75 { // 同值退化带:按 ±5% 容差判带外
			outside = c.Quantity > p75*1.05 || c.Quantity < p25*0.95
		}
		if !outside {
			continue
		}
		dir, edge := "高", p75
		if c.Quantity < p25 {
			dir, edge = "低", p25
		}
		pct := (c.Quantity - edge) / edge * 100
		out = append(out, ComposeCheck{
			Level: "warn", Row: i,
			Msg: fmt.Sprintf("含量 %.4g %s于同类 %d 例边界 %.4g（%+.1f%%），核对工艺口径或损耗",
				c.Quantity, dir, len(qs), edge, pct),
		})
	}
	return out
}
