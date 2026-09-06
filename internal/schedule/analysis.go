// schedule/analysis.go — CPM 分析叙事结构（schedule_analyze / apply 回执共用）。
package schedule

import "sort"

// NearCriticalTask 近关键工作（总时差 ≤ 2 个工作日）。
type NearCriticalTask struct {
	Name string `json:"name"`
	TF   int    `json:"tf"`
}

// MilestoneInfo 里程碑及其计划完成工作日序号。
type MilestoneInfo struct {
	Name    string `json:"name"`
	Workday int    `json:"workday"`
}

// Analysis CPM 之上的叙事数据（agent 据此组织自然语言解读）。
type Analysis struct {
	Cpm CpmResult `json:"cpm"`
	// Critical 关键工作名序列（按 ES、任务表序稳定）。
	Critical []string `json:"critical"`
	// NearCritical 近关键工作（TF ≤ 2，风险缓冲小）。
	NearCritical []NearCriticalTask `json:"nearCritical"`
	// Milestones 里程碑清单（含计划完成工作日序号）。
	Milestones []MilestoneInfo `json:"milestones"`
	// LeafCount 叶任务数（不含分组行）。
	LeafCount int `json:"leafCount"`
}

// ── 成本叙事（v4.122 资源成本刀2，schedule_analyze 回执 costs 段） ──────────

// 「已分配未定价」kind（AI 建议层 finding，非引擎拒绝；设计 §3.2）。
const (
	// UnpricedWorkNoRate 工时分配：资源未设 standardRate 且未设 costPerUse（成本=0）。
	UnpricedWorkNoRate = "work_norate"
	// UnpricedCostNoAmount 成本分配：缺 amount（成本=0）。
	UnpricedCostNoAmount = "cost_noamount"
)

// CostTopTask 成本 Top 任务行（叶任务，按行成本降序）。
type CostTopTask struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Total float64 `json:"total"`
}

// ResourceCost 资源维度成本汇总行。
type ResourceCost struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Type  ResourceType `json:"type"`
	Total float64      `json:"total"`
}

// UnpricedAssignment 已分配未定价条目（成本按 0 计，建议补费率/金额）。
type UnpricedAssignment struct {
	TaskID       string `json:"taskId"`
	TaskName     string `json:"taskName"`
	ResourceID   string `json:"resourceId"`
	ResourceName string `json:"resourceName"`
	// Kind 未定价原因：work_norate（工时资源未设费率与每次使用）/ cost_noamount（成本资源缺金额）。
	Kind string `json:"kind"`
}

// CostNarrative 成本叙事数据：在 ComputeCosts 之上组织总成本、成本 Top5 任务、
// 按资源汇总与「已分配未定价」清单。CPM 未过时 OK=false 且其余为零值
// （成本服从同一裁决，fail-closed）。
type CostNarrative struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	// Total 项目总成本（元）。
	Total float64 `json:"total"`
	// TopTasks 成本 Top5 叶任务（行成本降序，并列按任务表序稳定）。
	TopTasks []CostTopTask `json:"topTasks,omitempty"`
	// ByResource 资源维度汇总（元，按资源表序）。
	ByResource []ResourceCost `json:"byResource,omitempty"`
	// Unpriced 已分配未定价清单（AI 建议层，非引擎拒绝）。
	Unpriced []UnpricedAssignment `json:"unpriced,omitempty"`
}

// CostNarrative 成本叙事主计算（cpm 为 ComputeCpm 结果，成本服从同一裁决）。
func (p Project) CostNarrative(cpm CpmResult) CostNarrative {
	costs := ComputeCosts(p, cpm)
	n := CostNarrative{OK: costs.OK, Error: costs.Error, Total: costs.Total}
	if !costs.OK {
		return n
	}
	resByID := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resByID[r.ID] = r
	}
	taskByID := make(map[string]Task, len(p.Tasks))
	for _, t := range p.Tasks {
		taskByID[t.ID] = t
	}

	// 成本 Top5 叶任务：并列按任务表序稳定（sort.SliceStable + 表序入列）。
	type ranked struct {
		id string
		c  TaskCost
	}
	order := make([]ranked, 0, len(p.Tasks))
	for _, t := range p.Tasks {
		if t.Level == 0 {
			continue
		}
		order = append(order, ranked{id: t.ID, c: costs.Rows[t.ID]})
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].c.Total > order[j].c.Total })
	for i, r := range order {
		if i >= 5 || r.c.Total == 0 {
			break
		}
		n.TopTasks = append(n.TopTasks, CostTopTask{ID: r.id, Name: taskByID[r.id].Name, Total: r.c.Total})
	}

	// 资源维度汇总：按资源表序（确定性，不暴露 map 迭代序）。
	for _, r := range p.Resources {
		if total, ok := costs.ByResource[r.ID]; ok {
			n.ByResource = append(n.ByResource, ResourceCost{ID: r.ID, Name: r.Name, Type: r.Type, Total: total})
		}
	}

	// 已分配未定价：work 无费率且无每次使用 / cost 缺金额（口径 §3.2，
	// 只在「分配存在」时提示——资源在册但未挂分配不算）。
	for _, a := range p.Assignments {
		task, taskOK := taskByID[a.TaskID]
		res, resOK := resByID[a.ResourceID]
		if !taskOK || !resOK || task.Level == 0 {
			continue // 悬空/分组行：闸在 Validate，不在此重复报
		}
		var kind string
		switch {
		case res.Type == ResWork && res.StandardRate == 0 && res.CostPerUse == 0:
			kind = UnpricedWorkNoRate
		case res.Type == ResCost && a.Amount == 0:
			kind = UnpricedCostNoAmount
		}
		if kind != "" {
			n.Unpriced = append(n.Unpriced, UnpricedAssignment{
				TaskID: a.TaskID, TaskName: task.Name,
				ResourceID: a.ResourceID, ResourceName: res.Name, Kind: kind,
			})
		}
	}
	return n
}
