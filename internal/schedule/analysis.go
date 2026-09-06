// schedule/analysis.go — CPM 分析叙事结构（schedule_analyze / apply 回执共用）。
package schedule

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
