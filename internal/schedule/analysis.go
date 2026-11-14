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

// CdTask 日历天（cd）任务行（v4.151 双工期刀2：混排计划自检——哪几项按
// 自然日定时、占多少工作日跨度）。
type CdTask struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Duration 自然日数（durationUnit=cd 下的 duration 原值）。
	Duration int `json:"duration"`
	// Span 等效工作日跨度（cpm 行 ef−es；成本按此计费，元/工日不因日历天放大）。
	Span int `json:"span"`
	// Anchor 锚点日期（es 所在工作日，YYYY-MM-DD）。
	Anchor string `json:"anchor"`
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
	// CdTasks 日历天任务清单（v4.151 双工期刀2；缺省=无 cd 任务）。
	CdTasks []CdTask `json:"cdTasks,omitempty"`
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
		case res.Type == ResCost && (a.Amount == nil || *a.Amount == 0):
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

// ── 执行期治理叙事（斑马复刻 S2/S5：检视 + 预警，schedule_analyze 回执 execution 段）──────
//
// 口径源=前端 reviews.ts / warn.ts（plan-ask「计划-要求」算法，斑马 cn4:262/:293/:276-277）。
// Go 侧只做**叙事投影**（agent 读数），不是第二套预警引擎：档位表/完成日口径与前端逐字
// 同源，两侧注释互为镜像锚——改任何一侧必须同步另一侧并跑两侧测试。

// WarnCounts 预警计数（红灯=突破上级下达的要求完成；压线 0 天=绿灯）。
type WarnCounts struct {
	Red   int `json:"red"`
	Green int `json:"green"`
	// Unassigned 未设置要求完成的叶任务数（斑马「无级别」，不计入红绿灯）。
	Unassigned int `json:"unassigned"`
}

// WarnRow 红灯工作明细行。
type WarnRow struct {
	TaskID      string `json:"taskId"`
	Name        string `json:"name"`
	// Overdue 超期工作日（计划完成日 − 要求完成日，正=突破）。
	Overdue int    `json:"overdue"`
	// Band 滞后五档文本（与前端 warn.ts warnBand 同表：(0,15]/(15,30]/(30,60]/(60,100]/100+）。
	Band        string `json:"band"`
	IsMilestone bool   `json:"isMilestone,omitempty"`
}

// ReviewSnapshot 最新一期检视快照（斑马「当期/累计」两列口径）。
type ReviewSnapshot struct {
	// At 检视时间（工作日序号，相对开工日）。
	At int `json:"at"`
	// Date ISO 日期（展示与人工校核）。
	Date string `json:"date,omitempty"`
	// Percent 整份计划加权完成量（reviews.projectPercent 口径：按有效工期加权，里程碑不进分母）。
	Percent int `json:"percent"`
	// PlannedPercent 到该工作日计划应完成（reviews.reviewProgress 口径：按 CPM 行算已跨工期）。
	PlannedPercent int `json:"plannedPercent"`
	// GapPercent 实际 − 计划（正=超前，负=滞后）。
	GapPercent int `json:"gapPercent"`
	// Frozen 拉直冻结（前端 reviews.straighten 语义）。
	Frozen bool `json:"frozen,omitempty"`
}

// ExecutionNarrative 执行期治理叙事。**无任何信号**（零检视且全部叶任务未设要求完成）
// 时返回 nil——回执不造假零值；CPM 未过时同 nil（一切日期比较服从引擎裁决，同 CostNarrative）。
type ExecutionNarrative struct {
	ReviewCount int             `json:"reviewCount"`
	Latest      *ReviewSnapshot `json:"latest,omitempty"`
	Warn        WarnCounts      `json:"warn"`
	// RedRows 红灯明细（overdue 降序、并列保任务表序；上限 10——叙事只给最痛的，
	// 全量列由前端 warn.ts 承载）。
	RedRows []WarnRow `json:"redRows,omitempty"`
}

// execWarnBand 滞后五档（前端 warn.ts warnBand 镜像：只有 >0 的超期有档）。
func execWarnBand(overdue int) string {
	switch {
	case overdue <= 0:
		return ""
	case overdue <= 15:
		return "0-15天"
	case overdue <= 30:
		return "15-30天"
	case overdue <= 60:
		return "30-60天"
	case overdue <= 100:
		return "60-100天"
	default:
		return "100天以上"
	}
}

// execPlanFinishDay 计划完成日（最后工作日口径，前端 warn.ts planFinishDay 镜像）：
// cpm 行 EF 是独占边界（ES+工期），有工期 → EF−1；里程碑/零工期 → EF（完成发生在当天）。
func execPlanFinishDay(t Task, row TaskCpm) int {
	d := 0
	if t.Level != 0 && !t.IsMilestone && t.Duration > 0 {
		d = t.Duration
	}
	if d > 0 {
		return row.EF - 1
	}
	return row.EF
}

// execReviewPercent 某期的任务完成百分比（前端 reviews.reviewPercent 镜像）：
// 该期回填优先，缺省回落任务当前进度。
func execReviewPercent(r *Review, t Task) int {
	if r != nil {
		if a, ok := r.Actuals[t.ID]; ok && a.Percent != nil {
			v := *a.Percent
			if v < 0 { v = 0 }
			if v > 100 { v = 100 }
			return v
		}
	}
	v := t.Progress
	if v < 0 { v = 0 }
	if v > 100 { v = 100 }
	return v
}

// execProjectPercent 整份计划加权完成量（前端 reviews.projectPercent 镜像）：
// 分母不含里程碑与分组行；全是里程碑时退化为达成率。
func execProjectPercent(p Project, r *Review) int {
	weight := 0
	done := 0
	milestones := 0
	milestoneDone := 0
	for i := range p.Tasks {
		t := &p.Tasks[i]
		if t.Level == 0 {
			continue
		}
		if t.IsMilestone {
			milestones++
			if execReviewPercent(r, *t) >= 100 {
				milestoneDone++
			}
			continue
		}
		d := t.Duration
		if d < 0 { d = 0 }
		if d == 0 { continue }
		weight += d
		done += d * execReviewPercent(r, *t)
	}
	if weight == 0 {
		if milestones == 0 { return 0 }
		return int(float64(milestoneDone) / float64(milestones) * 100)
	}
	return int(float64(done)/float64(weight) + 0.5)
}

// ExecutionNarrative 执行期治理叙事主计算（cpm 为 ComputeCpm 结果）。
func (p Project) ExecutionNarrative(cpm CpmResult) *ExecutionNarrative {
	if !cpm.OK {
		return nil
	}
	n := &ExecutionNarrative{ReviewCount: len(p.Reviews)}
	hasSignal := len(p.Reviews) > 0
	for i := range p.Tasks {
		t := &p.Tasks[i]
		if t.Level == 0 {
			continue // 分组行不参与（前端 leavesOf 口径）
		}
		if t.AskFinish == nil || *t.AskFinish < 0 {
			n.Warn.Unassigned++
			continue
		}
		hasSignal = true
		row, ok := cpm.Rows[t.ID]
		if !ok {
			n.Warn.Unassigned++ // 有要求完成但缺排程行 → 前端 level 'none' 口径
			continue
		}
		overdue := execPlanFinishDay(*t, row) - *t.AskFinish
		if overdue > 0 {
			n.Warn.Red++
			n.RedRows = append(n.RedRows, WarnRow{
				TaskID: t.ID, Name: t.Name, Overdue: overdue,
				Band: execWarnBand(overdue), IsMilestone: t.IsMilestone,
			})
		} else {
			n.Warn.Green++
		}
	}
	if len(n.RedRows) > 0 {
		// overdue 降序、并列保任务表序（前端 sortWarns desc 稳定口径）
		rows := n.RedRows
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Overdue > rows[j].Overdue })
		if len(rows) > 10 {
			rows = rows[:10]
		}
		n.RedRows = rows
	}
	if len(p.Reviews) > 0 {
		latest := p.Reviews[0]
		for _, r := range p.Reviews[1:] {
			if r.At > latest.At {
				latest = r
			}
		}
		// 当期计划应完成 vs 实际（前端 reviews.reviewProgress 口径：按 CPM 行算已跨工期）
		total, planned, actual := 0, 0, 0.0
		for i := range p.Tasks {
			t := &p.Tasks[i]
			if t.Level == 0 || t.IsMilestone {
				continue
			}
			d := t.Duration
			if d < 0 { d = 0 }
			if d == 0 { continue }
			total += d
			if row, ok := cpm.Rows[t.ID]; ok {
				elapsed := latest.At - row.ES + 1
				if elapsed < 0 { elapsed = 0 }
				if elapsed > d { elapsed = d }
				planned += elapsed
			}
			actual += float64(d) * float64(execReviewPercent(&latest, *t)) / 100
		}
		plannedPct, actualPct := 0, 0
		if total > 0 {
			plannedPct = int(float64(planned)/float64(total)*100 + 0.5)
			actualPct = int(actual/float64(total)*100 + 0.5)
		}
		n.Latest = &ReviewSnapshot{
			At:             latest.At,
			Date:           latest.Date,
			Percent:        execProjectPercent(p, &latest),
			PlannedPercent: plannedPct,
			GapPercent:     actualPct - plannedPct,
			Frozen:         latest.Frozen,
		}
	}
	if !hasSignal {
		return nil
	}
	return n
}
