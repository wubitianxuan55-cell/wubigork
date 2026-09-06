// schedule/cost.go — 成本 rollup 引擎（纯函数，v4.122.0 资源成本刀1）。
//
// 与前端 frontend/src/schedule/cost.ts 互为镜像（测试同批场景同批期望值）。
// 设计口径（docs/gaea-schedule-resource-cost-design-2026-09.md §3.2）：
//   - work：effDur(工作日) × units(默认1) × StandardRate + CostPerUse
//   - material：Quantity × StandardRate + CostPerUse
//   - cost：Amount
//   - 任务成本 = FixedCost(默认0) + Σ 分配成本；rows 只含叶任务（分组行汇总
//     =子孙叶求和，视图层滚动口径，不经独立存储）；
//   - CPM 未过（循环依赖）→ OK=false，成本不出（fail-closed）；
//   - 悬空引用的分配跳过（容错读由前端 normalize 承担、落盘拒绝由 Validate
//     承担，引擎只对在册数据计算）；
//   - 金额 rollup 后四舍五入到分（0.01）。输入经校验非负，正值下
//     math.Round 与 JS Math.round 同律。
package schedule

import "math"

// TaskCost 单任务成本行（元）。
type TaskCost struct {
	Fixed    float64 `json:"fixed"`
	Assigned float64 `json:"assigned"`
	Total    float64 `json:"total"`
}

// CostResult 成本 rollup 结果。
type CostResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	// Rows 叶任务明细（分组行不入 Rows）。
	Rows map[string]TaskCost `json:"rows"`
	// ByResource 资源维度汇总（元）。
	ByResource map[string]float64 `json:"byResource"`
	// Total 项目总成本 = Σ 叶任务 total（元）。
	Total float64 `json:"total"`
}

func round2(n float64) float64 {
	return math.Round(n*100) / 100
}

// assignmentCost 单条分配的原始成本（元，未舍入）。资源类型未知返回 false（跳过）。
func assignmentCost(a Assignment, res Resource, dur int) (float64, bool) {
	units := 1.0
	if a.Units != nil {
		units = *a.Units
	}
	switch res.Type {
	case ResWork:
		return float64(dur)*units*res.StandardRate + res.CostPerUse, true
	case ResMaterial:
		return a.Quantity*res.StandardRate + res.CostPerUse, true
	case ResCost:
		return a.Amount, true
	default:
		return 0, false
	}
}

// ComputeCosts 成本 rollup 主计算。cpm 为 ComputeCpm 结果（成本服从同一裁决）；
// 悬空引用的分配跳过；全无资源/成本数据的计划 → OK 且 Total=0。
func ComputeCosts(p Project, cpm CpmResult) CostResult {
	if !cpm.OK {
		return CostResult{OK: false, Error: cpm.Error, Rows: map[string]TaskCost{}, ByResource: map[string]float64{}}
	}
	resources := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resources[r.ID] = r
	}
	tasksByID := make(map[string]Task, len(p.Tasks))
	for _, t := range p.Tasks {
		tasksByID[t.ID] = t
	}
	assignedRaw := make(map[string]float64)
	byResourceRaw := make(map[string]float64)
	for _, a := range p.Assignments {
		res, resOK := resources[a.ResourceID]
		task, taskOK := tasksByID[a.TaskID]
		if !resOK || !taskOK || task.Level == 0 {
			continue // 悬空引用/分组行分配：跳过（闸在 Validate）
		}
		c, ok := assignmentCost(a, res, effDur(task))
		if !ok {
			continue
		}
		assignedRaw[a.TaskID] += c
		byResourceRaw[a.ResourceID] += c
	}

	rows := make(map[string]TaskCost, len(p.Tasks))
	var totalRaw float64
	for _, t := range p.Tasks {
		if t.Level == 0 {
			continue
		}
		fixed := t.FixedCost
		assigned := assignedRaw[t.ID]
		rows[t.ID] = TaskCost{Fixed: round2(fixed), Assigned: round2(assigned), Total: round2(fixed + assigned)}
		totalRaw += fixed + assigned
	}
	byResource := make(map[string]float64, len(byResourceRaw))
	for rid, raw := range byResourceRaw {
		byResource[rid] = round2(raw)
	}
	return CostResult{OK: true, Rows: rows, ByResource: byResource, Total: round2(totalRaw)}
}
