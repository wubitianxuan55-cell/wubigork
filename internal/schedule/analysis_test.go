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
			{TaskID: "B", ResourceID: "m1", Quantity: 4},
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
	p.Assignments[3].Amount = 250
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
