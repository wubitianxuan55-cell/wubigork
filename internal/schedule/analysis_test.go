// schedule/analysis_test.go — 成本叙事（CostNarrative）用例（v4.122.0 资源成本刀2）。
//
// 口径（设计 §3.2）：「已分配未定价」是 AI 建议层 finding（work 无费率且无每次
// 使用 / cost 缺金额），非引擎拒绝；material 无单价不在该 finding 范围（拍板口径）。
// CPM 未过 → OK=false 且其余为零值（成本服从同一裁决）。
package schedule

import "testing"

func narrativeProject() *Project {
	return &Project{
		Name: "叙事", StartDate: "2026-01-05",
		Tasks: []Task{
			{ID: "G", Name: "分组", Level: 0},
			{ID: "A", Name: "挖土", Duration: 2, Level: 1},
			{ID: "B", Name: "吊装", Duration: 3, Level: 1},
			{ID: "C", Name: "运输", Duration: 1, Level: 1},
		},
		Links: []Link{lnk("A", "B", FS, 0)},
		Resources: []Resource{
			{ID: "r1", Name: "普工", Type: ResWork},                    // 未定价（无费率无每次使用）
			{ID: "r2", Name: "塔吊", Type: ResWork, StandardRate: 500}, // 已定价
			{ID: "m1", Name: "石材", Type: ResMaterial},                // 材料无单价：不在未定价 finding 范围
			{ID: "c1", Name: "差旅", Type: ResCost},                    // 成本资源缺金额 → 未定价
		},
		Assignments: []Assignment{
			{TaskID: "A", ResourceID: "r1"},
			{TaskID: "B", ResourceID: "r2", Units: floatPtr(2)},
			{TaskID: "B", ResourceID: "m1", Quantity: floatPtr(4)},
			{TaskID: "C", ResourceID: "c1"},
		},
	}
}

func TestCostNarrativeUnpriced(t *testing.T) {
	p := narrativeProject()
	n := p.CostNarrative(ComputeCpm(p.Tasks, p.Links))
	if !n.OK {
		t.Fatalf("CPM 通过时叙事应 ok：%+v", n)
	}
	// B = 3×2×500 = 3000，其余行成本 0 → Top 只收非零行
	if len(n.TopTasks) != 1 || n.TopTasks[0].ID != "B" || n.TopTasks[0].Name != "吊装" || n.TopTasks[0].Total != 3000 {
		t.Fatalf("TopTasks = %+v", n.TopTasks)
	}
	if n.Total != 3000 {
		t.Errorf("Total = %v, want 3000", n.Total)
	}
	// byResource 按资源表序，且只含在册分配（四条分配各产出一条，含零值行）
	if len(n.ByResource) != 4 || n.ByResource[0].ID != "r1" || n.ByResource[0].Total != 0 ||
		n.ByResource[1].ID != "r2" || n.ByResource[1].Total != 3000 || n.ByResource[2].ID != "m1" || n.ByResource[3].ID != "c1" {
		t.Errorf("ByResource = %+v", n.ByResource)
	}
	// 未定价：work_norate + cost_noamount；材料无单价不报；分组行分配不在场
	if len(n.Unpriced) != 2 {
		t.Fatalf("Unpriced = %+v, want 2 条", n.Unpriced)
	}
	if n.Unpriced[0].Kind != UnpricedWorkNoRate || n.Unpriced[0].TaskID != "A" || n.Unpriced[0].TaskName != "挖土" || n.Unpriced[0].ResourceName != "普工" {
		t.Errorf("Unpriced[0] = %+v", n.Unpriced[0])
	}
	if n.Unpriced[1].Kind != UnpricedCostNoAmount || n.Unpriced[1].TaskID != "C" || n.Unpriced[1].ResourceID != "c1" {
		t.Errorf("Unpriced[1] = %+v", n.Unpriced[1])
	}
	// 补价后未定价清零（费率×工期生效；成本资源补金额）
	p.Resources[0].StandardRate = 100
	p.Assignments[3].Amount = floatPtr(250)
	n = p.CostNarrative(ComputeCpm(p.Tasks, p.Links))
	if len(n.Unpriced) != 0 || n.Total != 3450 { // 2×100 + 3000 + 250
		t.Errorf("补价后 Unpriced/Total = %+v/%v, want 0/3450", n.Unpriced, n.Total)
	}
}

func TestCostNarrativeTopFiveAndStableOrder(t *testing.T) {
	p := &Project{Name: "P", StartDate: "2026-01-05"}
	tasks := []Task{}
	for i := 0; i < 7; i++ {
		tasks = append(tasks, Task{ID: string(rune('a' + i)), Name: string(rune('a' + i)), Duration: 1, Level: 1})
	}
	p.Tasks = tasks
	p.Resources = []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100}}
	assigns := []Assignment{}
	for _, ts := range p.Tasks {
		assigns = append(assigns, Assignment{TaskID: ts.ID, ResourceID: "r1"})
	}
	p.Assignments = assigns
	n := p.CostNarrative(ComputeCpm(p.Tasks, p.Links))
	if len(n.TopTasks) != 5 {
		t.Fatalf("Top 应截取 5 条，got %d", len(n.TopTasks))
	}
	// 全部等值（各 100）→ 稳定序=任务表序
	for i, tt := range n.TopTasks {
		if tt.ID != string(rune('a'+i)) || tt.Total != 100 {
			t.Fatalf("Top[%d] = %+v, want 稳定任务表序", i, tt)
		}
	}
}

func TestCostNarrativeFailClosedOnCycle(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks:       []Task{{ID: "a", Name: "a", Duration: 2, Level: 1}, {ID: "b", Name: "b", Duration: 2, Level: 1}},
		Links:       []Link{lnk("a", "b", FS, 0), lnk("b", "a", FS, 0)},
		Resources:   []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100}},
		Assignments: []Assignment{{TaskID: "a", ResourceID: "r1"}},
	}
	n := p.CostNarrative(ComputeCpm(p.Tasks, p.Links))
	if n.OK || n.Total != 0 || len(n.TopTasks) != 0 || len(n.ByResource) != 0 || len(n.Unpriced) != 0 {
		t.Fatalf("环依赖应 fail-closed：%+v", n)
	}
	if n.Error == "" {
		t.Fatal("应带错误原文")
	}
}

func TestCostNarrativeLegacyPlan(t *testing.T) {
	p := &Project{Name: "旧计划", StartDate: "2026-01-05", Tasks: []Task{{ID: "a", Name: "a", Duration: 2, Level: 1}}}
	n := p.CostNarrative(ComputeCpm(p.Tasks, p.Links))
	if !n.OK || n.Total != 0 || len(n.TopTasks) != 0 || len(n.ByResource) != 0 || len(n.Unpriced) != 0 {
		t.Fatalf("无资源维度旧计划叙事应全空：%+v", n)
	}
}

// TestAnalyzeCdTasks 双工期刀2（v4.151）：混排计划的日历天任务清单
// （自然日数/等效工作日跨度/锚点日期；显式 wd 任务不入清单）。
func TestAnalyzeCdTasks(t *testing.T) {
	p := Project{
		Name:      "混排样板",
		StartDate: "2026-09-07",
		Tasks: []Task{
			tsk("A", 5, nil),
			cdTask("养护", 28, nil),
			tsk("回填", 5, nil),
			tsk("wd标记", 3, func(x *Task) { x.DurationUnit = UnitWd }),
		},
		Links: []Link{lnk("A", "养护", FS, 0), lnk("养护", "回填", FS, 0)},
	}
	cpm, a := p.Analyze()
	if !cpm.OK || cpm.Duration != 30 {
		t.Fatalf("ok=%v dur=%d, want 30", cpm.OK, cpm.Duration)
	}
	if len(a.CdTasks) != 1 {
		t.Fatalf("cdTasks = %+v", a.CdTasks)
	}
	cd := a.CdTasks[0]
	if cd.ID != "养护" || cd.Duration != 28 || cd.Span != 20 || cd.Anchor != "2026-09-14" {
		t.Fatalf("cd 行 = %+v", cd)
	}
}

// ── 执行期治理叙事（斑马复刻 S2/S5：检视+预警，plan-ask 口径 Go 镜像）────────────

func execNarrativeProject() *Project {
	// A(3)→B(2) FS：A 完成日=EF−1=2；B es=3、EF=5 → 完成日=4
	return &Project{
		Name: "执行治理", StartDate: "2026-01-05",
		Tasks: []Task{
			{ID: "A", Name: "挖土", Duration: 3, Level: 1, AskFinish: intPtr(0)},  // 完成日 2，要求 0 → 红灯 2
			{ID: "B", Name: "吊装", Duration: 2, Level: 1, AskFinish: intPtr(9)},  // 完成日 4，要求 9 → 绿灯 -5
			{ID: "C", Name: "运输", Duration: 1, Level: 1},                         // 未设置 → unassigned
			{ID: "G", Name: "分组", Level: 0, AskFinish: intPtr(0)},               // 分组行不参与
		},
		Links: []Link{lnk("A", "B", FS, 0)},
	}
}

func TestExecutionNarrativeWarnCounts(t *testing.T) {
	p := execNarrativeProject()
	cpm, _ := p.Analyze()
	if !cpm.OK {
		t.Fatalf("cpm 应通过: %+v", cpm)
	}
	n := p.ExecutionNarrative(cpm)
	if n == nil {
		t.Fatal("有信号应出叙事")
	}
	if n.Warn.Red != 1 || n.Warn.Green != 1 || n.Warn.Unassigned != 1 {
		t.Fatalf("计数不齐: %+v", n.Warn)
	}
	if len(n.RedRows) != 1 || n.RedRows[0].TaskID != "A" {
		t.Fatalf("红灯行不齐: %+v", n.RedRows)
	}
	if n.RedRows[0].Overdue != 2 || n.RedRows[0].Band != "0-15天" {
		t.Fatalf("红灯口径不齐（完成日=EF−1=2，要求 0 → 2 天）: %+v", n.RedRows[0])
	}
}

func TestExecutionNarrativeBandTableMirror(t *testing.T) {
	// 五档边界与前端 warn.ts warnBand 同表：(0,15]/(15,30]/(30,60]/(60,100]/100+
	cases := map[int]string{1: "0-15天", 15: "0-15天", 16: "15-30天", 30: "15-30天", 31: "30-60天", 60: "30-60天", 61: "60-100天", 100: "60-100天", 101: "100天以上"}
	for days, want := range cases {
		if got := execWarnBand(days); got != want {
			t.Fatalf("execWarnBand(%d)=%s， want %s", days, got, want)
		}
	}
	if execWarnBand(0) != "" || execWarnBand(-3) != "" {
		t.Fatal("压线/提前无档")
	}
}

func TestExecutionNarrativeMilestoneFinishDay(t *testing.T) {
	// 里程碑/零工期完成日=EF（=ES，当天完成），不是 EF−1：
	// A(3)→M FS → M es=3、EF=3，完成日=3；要求 0 → 红灯 3 天。
	// 若公式误用 EF−1=2 会得 2 天——用例钉死这个口径。
	p := &Project{
		Name: "里程碑", StartDate: "2026-01-05",
		Tasks: []Task{
			{ID: "A", Name: "挖土", Duration: 3, Level: 1},
			{ID: "M", Name: "节点", Level: 1, IsMilestone: true, AskFinish: intPtr(0)},
		},
		Links: []Link{lnk("A", "M", FS, 0)},
	}
	cpm, _ := p.Analyze()
	n := p.ExecutionNarrative(cpm)
	if n == nil || n.Warn.Red != 1 {
		t.Fatalf("里程碑要求早于计划应红灯: %+v", n)
	}
	if n.RedRows[0].Overdue != 3 || n.RedRows[0].Band != "0-15天" || !n.RedRows[0].IsMilestone {
		t.Fatalf("里程碑完成日=EF=3 → 超期 3 天: %+v", n.RedRows)
	}
}

func TestExecutionNarrativeNilWithoutSignal(t *testing.T) {
	// 零检视且无人设要求完成 → nil（不造假零值）；CPM 未过 → nil
	p := sampleProject()
	cpm, _ := p.Analyze()
	if n := p.ExecutionNarrative(cpm); n != nil {
		t.Fatalf("无信号应返回 nil: %+v", n)
	}
	bad := sampleProject()
	bad.Links = append(bad.Links, lnk("B", "A", FS, 0)) // 循环
	bcpm, _ := bad.Analyze()
	if bcpm.OK {
		t.Fatal("构造循环失败")
	}
	if n := bad.ExecutionNarrative(bcpm); n != nil {
		t.Fatal("CPM 未过应返回 nil")
	}
}

func TestExecutionNarrativeLatestReview(t *testing.T) {
	p := execNarrativeProject()
	p.Reviews = []Review{
		{At: 1, Date: "2026-01-06", Actuals: map[string]ReviewActual{"A": {Percent: intPtr(100)}}},
		{At: 4, Date: "2026-01-09", Frozen: true, Actuals: map[string]ReviewActual{"A": {Percent: intPtr(100)}, "B": {Percent: intPtr(50)}}},
	}
	cpm, _ := p.Analyze()
	n := p.ExecutionNarrative(cpm)
	if n == nil || n.ReviewCount != 2 || n.Latest == nil {
		t.Fatalf("叙事不齐: %+v", n)
	}
	if n.Latest.At != 4 || !n.Latest.Frozen {
		t.Fatalf("应取最新一期: %+v", n.Latest)
	}
	// 加权完成量（reviews.projectPercent 口径，分母含全部叶）：A(3)100%+B(2)50%+C(1)0%
	// → (300+100+0)/(3+2+1)=67%
	if n.Latest.Percent != 67 {
		t.Fatalf("加权完成量应为 67: %d", n.Latest.Percent)
	}
	// 计划应完成：A es=0 全跨 3；B es=3 到 at=4 已跨 2；C es=0 已跨 1 → 6/6=100%
	if n.Latest.PlannedPercent != 100 {
		t.Fatalf("计划应完成应为 100: %d", n.Latest.PlannedPercent)
	}
	if n.Latest.GapPercent != -33 {
		t.Fatalf("偏差应为 -33: %d", n.Latest.GapPercent)
	}
}

func TestExecutionNarrativeRedRowsSortedAndCapped(t *testing.T) {
	ask := func(v int) *int { return &v }
	p := &Project{Name: "排序", StartDate: "2026-01-05"}
	for i := 0; i < 12; i++ {
		p.Tasks = append(p.Tasks, Task{ID: string(rune('A' + i)), Name: "T", Duration: 2, Level: 1, AskFinish: ask(0)})
	}
	cpm, _ := p.Analyze()
	n := p.ExecutionNarrative(cpm)
	if n.Warn.Red != 12 {
		t.Fatalf("12 项全红灯: %+v", n.Warn)
	}
	if len(n.RedRows) != 10 {
		t.Fatalf("明细应截到 10: %d", len(n.RedRows))
	}
	for i := 1; i < len(n.RedRows); i++ {
		if n.RedRows[i-1].Overdue < n.RedRows[i].Overdue {
			t.Fatalf("应 overdue 降序: %+v", n.RedRows)
		}
	}
}
