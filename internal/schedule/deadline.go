// schedule/deadline.go — 倒排校核（v4.117.0 刀8）。
//
// 与前端 frontend/src/schedule/deadline.ts 互为镜像（同批场景同批期望值）：
// 目标是引擎裁决合同/定额竣工约束的可行性，不自动改排程（自动锁死任务违背
// manual 语义；压缩方案由 AI 建议、用户拍板）。目标总工期=DeadlineWorkdays
// 换算（工作日口径）；可行性=当前总工期 ≤ 目标。
package schedule

// DeadlineCheck 倒排校核结果。
type DeadlineCheck struct {
	// Deadline 目标竣工日期（YYYY-MM-DD）。
	Deadline string `json:"deadline"`
	// TargetWorkdays 目标总工期（工作日，1-based：第 N 个工作日竣工）。
	TargetWorkdays  int `json:"targetWorkdays"`
	CurrentDuration int `json:"currentDuration"`
	// Feasible 可行：当前总工期 ≤ 目标。
	Feasible bool `json:"feasible"`
	// Overrun 超期量（正=超期工作日，负=富余，0=压线达成）。
	Overrun int `json:"overrun"`
	// CriticalTasks 关键工作名清单（超期时的压缩对象；手动不计）。
	CriticalTasks []string `json:"criticalTasks"`
}

// CheckDeadline 倒排校核。未设目标竣工或计划未通过 CPM 返回 nil。
func CheckDeadline(p *Project, cpm CpmResult) *DeadlineCheck {
	if p.Deadline == "" || !cpm.OK {
		return nil
	}
	target := DeadlineWorkdays(p.StartDate, p.Deadline, p.Calendar)
	crit := make([]string, 0, len(p.Tasks))
	for _, t := range p.Tasks {
		if t.Level > 0 && t.Mode != ModeManual {
			if row, ok := cpm.Rows[t.ID]; ok && row.Critical {
				crit = append(crit, t.Name)
			}
		}
	}
	return &DeadlineCheck{
		Deadline:        p.Deadline,
		TargetWorkdays:  target,
		CurrentDuration: cpm.Duration,
		Feasible:        cpm.Duration <= target,
		Overrun:         cpm.Duration - target,
		CriticalTasks:   crit,
	}
}
