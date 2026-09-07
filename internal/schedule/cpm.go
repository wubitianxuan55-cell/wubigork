// schedule/cpm.go — 关键路径法（CPM）计算引擎（前端 cpm.ts 的忠实移植，v4.110/111 口径）。
//
// 输入任务表 + 搭接关系（FS/SS/FF/SF + 时距），输出正推（ES/EF）、逆推（LS/LF）、
// 总时差（TF）、自由时差（FF）与关键工作标记。检测到循环依赖时 fail-closed 返回
// OK=false。手动/自动双模式：manual 锁定开始、忽略入边、逆推不回传约束、不标关键。
package schedule

import (
	"fmt"
	"sort"
	"strings"
)

// effDur 有效工期：里程碑为 0，其余取非负整数。
func effDur(t Task) int {
	if t.IsMilestone {
		return 0
	}
	return maxInt(0, t.Duration)
}

// forwardBound 单条搭接对后续任务 ES 的正推下界。
func forwardBound(l Link, from TaskCpm, durTo int) int {
	switch l.Type {
	case FS:
		return from.EF + l.Lag
	case SS:
		return from.ES + l.Lag
	case FF:
		return from.EF + l.Lag - durTo
	case SF:
		return from.ES + l.Lag - durTo
	}
	return 0
}

// backwardBound 单条搭接对前置任务 LF 的逆推上界。
func backwardBound(l Link, to TaskCpm, durFrom, durTo int) int {
	switch l.Type {
	case FS:
		return to.LF - durTo - l.Lag
	case SS:
		return to.LS - l.Lag + durFrom
	case FF:
		return to.LF - l.Lag
	case SF:
		return to.LF - l.Lag + durFrom
	}
	return 0
}

// freeFloatPart 单条搭接对前置任务自由时差的贡献（搭接口径 LAG_i-j，
// v4.129 刀G 修 E1）：FF/FS 锚点=前置 EF；SS/SF 锚点=前置 ES
// （旧版漏减 ES_from 致 SS/SF 下 FF 虚高，违反「TF=0 ⇒ FF=0」定理）。
func freeFloatPart(l Link, to TaskCpm, efFrom, esFrom int) int {
	switch l.Type {
	case FS:
		return to.ES - l.Lag - efFrom
	case SS:
		return to.ES - l.Lag - esFrom
	case FF:
		return to.EF - l.Lag - efFrom
	case SF:
		return to.EF - l.Lag - esFrom
	}
	return 0
}

// topoOrder Kahn 拓扑排序；cycle 非空表示存在环（环上任务 id）。
func topoOrder(ids []string, out map[string][]string, indeg map[string]int) (order, cycle []string) {
	deg := make(map[string]int, len(indeg))
	for k, v := range indeg {
		deg[k] = v
	}
	queue := make([]string, 0)
	for _, id := range ids {
		if deg[id] == 0 {
			queue = append(queue, id)
		}
	}
	order = make([]string, 0, len(ids))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, nxt := range out[id] {
			deg[nxt]--
			if deg[nxt] == 0 {
				queue = append(queue, nxt)
			}
		}
	}
	if len(order) != len(ids) {
		inOrder := make(map[string]bool, len(order))
		for _, id := range order {
			inOrder[id] = true
		}
		for _, id := range ids {
			if !inOrder[id] {
				cycle = append(cycle, id)
			}
		}
	}
	return order, cycle
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ComputeCpm CPM 主计算（无目标竣工锚点，按计算工期逆推；无日历换算上下文，
// 含 cd 任务时 fail-closed——调用方持有 Project 时应走 ComputeCpmCal）。
func ComputeCpm(tasks []Task, links []Link) CpmResult {
	return computeCpmFull(tasks, links, 0, nil, "")
}

// ComputeCpmPlan 带计划工期锚点的 CPM（v4.129 刀G G3）：planFinish=目标竣工换算的
// 工作日边界（第 N 工作日竣工）。规程口径：计划工期 Tp 小于计算工期 Tc 时逆推从
// Tp 起——绑定链出现负时差，关键工作=总时差最小者（不再恒为 TF=0）；
// planFinish<=0 或 ≥Tc 时与 ComputeCpm 完全一致。
func ComputeCpmPlan(tasks []Task, links []Link, planFinish int) CpmResult {
	return computeCpmFull(tasks, links, planFinish, nil, "")
}

// ComputeCpmCal 带日历换算上下文的 CPM（v4.150 双工期刀1）：存在 cd（日历天）
// 任务时以 cal/start 为换算锚点（cal 缺省回落周一~五，与 WdToDate 全部读点
// 同口径；start 缺失 → OK=false fail-closed）。无 cd 任务时上下文不被读取，
// 与 ComputeCpm 逐位一致。
func ComputeCpmCal(tasks []Task, links []Link, cal *Calendar, start string) CpmResult {
	return computeCpmFull(tasks, links, 0, cal, start)
}

// ComputeCpmPlanCal ComputeCpmPlan 的日历上下文变体（Go 无默认参，先例式双入口）。
func ComputeCpmPlanCal(tasks []Task, links []Link, planFinish int, cal *Calendar, start string) CpmResult {
	return computeCpmFull(tasks, links, planFinish, cal, start)
}

func computeCpmFull(tasks []Task, links []Link, planFinish int, calInput *Calendar, start string) CpmResult {
	byID := make(map[string]Task, len(tasks))
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
		ids = append(ids, t.ID)
	}
	valid := make([]Link, 0, len(links))
	for _, l := range links {
		if l.From != l.To {
			if _, ok := byID[l.From]; ok {
				if _, ok2 := byID[l.To]; ok2 {
					valid = append(valid, l)
				}
			}
		}
	}

	out := make(map[string][]string, len(ids))
	outLinks := make(map[string][]Link, len(ids))
	inLinks := make(map[string][]Link, len(ids))
	indeg := make(map[string]int, len(ids))
	for _, id := range ids {
		out[id] = nil
		outLinks[id] = nil
		inLinks[id] = nil
		indeg[id] = 0
	}
	for _, l := range valid {
		out[l.From] = append(out[l.From], l.To)
		outLinks[l.From] = append(outLinks[l.From], l)
		inLinks[l.To] = append(inLinks[l.To], l)
		indeg[l.To]++
	}

	rows := make(map[string]TaskCpm, len(tasks))
	for _, t := range tasks {
		rows[t.ID] = TaskCpm{}
	}
	res := CpmResult{Rows: rows}
	if len(ids) == 0 {
		res.OK = true
		return res
	}

	order, cycle := topoOrder(ids, out, indeg)
	if len(cycle) > 0 {
		// 与前端同口径：按任务表顺序拼接环上任务名
		res.Cycle = cycle
		res.Error = fmt.Sprintf("存在循环依赖：%s", joinNames(byID, cycle))
		return res
	}

	// 双工期口径（v4.150 刀1）：存在 cd（日历天）任务时启用日历换算分支；
	// 无 cd 任务走快路径（下方分支全部短路，结果与旧口径逐位一致）。
	// fail-closed：缺开工日期无法换算；cd 涉非 FS 搭接会引入对 dur 的不动点
	// 迭代，v1 引擎拒绝（闸在 Validate，此处防御直接调用）。
	hasCd := false
	for _, t := range tasks {
		if t.DurationUnit == UnitCd {
			hasCd = true
			break
		}
	}
	cal := Calendar{}
	startISO := ""
	if hasCd {
		if strings.TrimSpace(start) == "" {
			res.Error = "计划缺开工日期，无法计算日历天（cd）任务"
			return res
		}
		cal = NormalizeCalendar(calInput)
		startISO = start
		for _, l := range valid {
			if (byID[l.From].DurationUnit == UnitCd || byID[l.To].DurationUnit == UnitCd) && l.Type != FS {
				res.Error = fmt.Sprintf("日历天（cd）任务仅支持 FS 搭接（%s → %s 为 %s）", byID[l.From].Name, byID[l.To].Name, l.Type)
				return res
			}
		}
	}

	// 正推：ES = max(各搭接下界)，EF = ES + 工期；manual 任务锁定开始、忽略入边。
	for _, id := range order {
		t := byID[id]
		dur := effDur(t)
		row := rows[id]
		if t.Mode == ModeManual {
			fixed := maxInt(0, t.ManualStart)
			row.ES = fixed
			if hasCd && t.DurationUnit == UnitCd {
				row.EF = CdToEf(startISO, fixed, dur, &cal)
			} else {
				row.EF = fixed + dur
			}
			rows[id] = row
			continue
		}
		es := 0
		for _, l := range inLinks[id] {
			es = maxInt(es, forwardBound(l, rows[l.From], dur))
		}
		row.ES = es
		if hasCd && t.DurationUnit == UnitCd {
			row.EF = CdToEf(startISO, es, dur, &cal)
		} else {
			row.EF = es + dur
		}
		rows[id] = row
	}
	duration := 0
	for _, t := range tasks {
		duration = maxInt(duration, rows[t.ID].EF)
	}

	// 逆推锚点（G3）：计划工期 < 计算工期时从计划工期逆推（负时差诚实呈现）。
	anchor := duration
	if planFinish > 0 && planFinish < duration {
		anchor = planFinish
	}

	// 逆推：LF = min(各搭接上界 / 计划工期)；manual 任务 LF=EF（锁定，不回传约束）。
	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		t := byID[id]
		dur := effDur(t)
		row := rows[id]
		if t.Mode == ModeManual {
			row.LF = row.EF
			row.LS = row.ES
			rows[id] = row
			continue
		}
		lf := anchor
		for _, l := range outLinks[id] {
			if byID[l.To].Mode == ModeManual {
				continue // 手动后继不约束前置
			}
			// FS 上界=后继最迟开始−lag：wd 后继 to.ls=to.lf−durTo 与旧式逐位
			// 一致；cd 后继的 ls 已按日历换算（≠lf−dur），必须用 to.ls 才诚实。
			if hasCd && l.Type == FS {
				lf = minInt(lf, rows[l.To].LS-l.Lag)
			} else {
				lf = minInt(lf, backwardBound(l, rows[l.To], dur, effDur(byID[l.To])))
			}
		}
		row.LF = lf
		if hasCd && t.DurationUnit == UnitCd {
			row.LS = CdLatestStart(startISO, lf, dur, &cal)
		} else {
			row.LS = lf - dur
		}
		rows[id] = row
	}

	// 时差与关键标记（manual 任务不标关键）：关键工作=总时差最小者
	// （规程口径：Tp=Tc 时最小值为 0，Tp<Tc 时为负——不再恒为 TF==0）。
	// 仅在 deadline 真正收紧（anchor<duration）时启用 minTF：manual 驱动总工期、
	// 非手动链无 TF=0 的场景维持经典口径（避免把带浮动的工作误标关键）。
	minTF := -1
	for _, t := range tasks {
		if t.Mode == ModeManual {
			continue
		}
		tf := rows[t.ID].LS - rows[t.ID].ES
		if minTF == -1 || tf < minTF {
			minTF = tf
		}
	}
	critTF := 0
	if anchor < duration {
		critTF = minTF
	}
	for _, t := range tasks {
		row := rows[t.ID]
		row.TF = row.LS - row.ES
		ff := anchor - row.EF
		for _, l := range outLinks[t.ID] {
			if byID[l.To].Mode == ModeManual {
				continue
			}
			ff = minInt(ff, freeFloatPart(l, rows[l.To], row.EF, row.ES))
		}
		row.FF = ff
		row.Critical = t.Mode != ModeManual && row.TF == critTF
		rows[t.ID] = row
	}

	res.OK = true
	res.Duration = duration
	return res
}

func joinNames(byID map[string]Task, ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += " → "
		}
		out += byID[id].Name
	}
	return out
}

// CriticalChain 关键工作名序列（按 ES 再按任务表顺序；叙事用途——CPM 关键
// 工作集合必构成一条以上 s→t 链，本方法给「链上都有谁」的稳定输出）。
func (r CpmResult) CriticalChain(tasks []Task) []string {
	type item struct {
		name string
		es   int
		idx  int
	}
	items := make([]item, 0)
	for i, t := range tasks {
		if row, ok := r.Rows[t.ID]; ok && row.Critical {
			items = append(items, item{name: t.Name, es: row.ES, idx: i})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].es != items[j].es {
			return items[i].es < items[j].es
		}
		return items[i].idx < items[j].idx
	})
	names := make([]string, 0, len(items))
	for _, it := range items {
		names = append(names, it.name)
	}
	return names
}
