// schedule/cost_test.go — 成本 rollup 用例（v4.122.0 资源成本刀1）。
//
// 与前端 frontend/src/schedule/cost.test.ts 互为镜像：同一批场景
// 同一批期望值，两侧引擎口径必须逐字段一致。
package schedule

import (
	"math"
	"testing"
)

func costTask(id string, dur int, extra func(*Task)) Task {
	t := Task{ID: id, Name: id, Duration: dur, Level: 1}
	if extra != nil {
		extra(&t)
	}
	return t
}

func floatPtr(v float64) *float64 { return &v }

// 镜像 TS costProject：A(3,work 人力)→B(2,material 钢材+cost 差旅) + 里程碑 + 分组行。
func costProject() *Project {
	units2 := floatPtr(2)
	return &Project{
		Name:      "成本样板",
		StartDate: "2026-01-05",
		Tasks: []Task{
			costTask("A", 3, nil),
			costTask("B", 2, func(t *Task) { t.FixedCost = 50 }),
			{ID: "G1", Name: "G1", Level: 0},
			costTask("C", 4, nil),
			costTask("M", 5, func(t *Task) { t.IsMilestone = true }),
		},
		Links: []Link{{From: "A", To: "B", Type: FS}},
		Resources: []Resource{
			{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 300, CostPerUse: 200},
			{ID: "r2", Name: "钢材", Type: ResMaterial, StandardRate: 55, Unit: "t"},
			{ID: "r3", Name: "差旅", Type: ResCost},
		},
		Assignments: []Assignment{
			{TaskID: "A", ResourceID: "r1", Units: units2},
			{TaskID: "B", ResourceID: "r2", Quantity: 10},
			{TaskID: "B", ResourceID: "r3", Amount: 300},
			{TaskID: "M", ResourceID: "r1"},
			{TaskID: "G1", ResourceID: "r1"}, // 分组行分配：引擎跳过（闸在 Validate）
		},
	}
}

func TestComputeCostsWorkFormula(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks:       []Task{costTask("A", 3, nil), costTask("B", 2, nil)},
		Resources:   []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100, CostPerUse: 200}},
		Assignments: []Assignment{{TaskID: "A", ResourceID: "r1", Units: floatPtr(2)}},
	}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if !r.OK {
		t.Fatal("CPM 通过时成本应 ok")
	}
	got := r.Rows["A"]
	want := TaskCost{Fixed: 0, Assigned: 800, Total: 800} // 3×2×100+200
	if got != want {
		t.Errorf("Rows[A] = %+v, want %+v", got, want)
	}
	if r.Rows["B"] != (TaskCost{}) {
		t.Errorf("Rows[B] = %+v, want 零值", r.Rows["B"])
	}
	if r.ByResource["r1"] != 800 || r.Total != 800 {
		t.Errorf("byResource/total = %v/%v, want 800/800", r.ByResource["r1"], r.Total)
	}
}

func TestComputeCostsMaterialAndCost(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks:       []Task{costTask("A", 3, nil), costTask("B", 2, nil)},
		Resources:   []Resource{{ID: "m", Name: "钢材", Type: ResMaterial, StandardRate: 55}, {ID: "c", Name: "差旅", Type: ResCost}},
		Assignments: []Assignment{{TaskID: "A", ResourceID: "m", Quantity: 10}, {TaskID: "B", ResourceID: "c", Amount: 300}},
	}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if r.Rows["A"].Assigned != 550 || r.Rows["B"].Assigned != 300 {
		t.Errorf("material/cost = %v/%v, want 550/300", r.Rows["A"].Assigned, r.Rows["B"].Assigned)
	}
	if r.ByResource["m"] != 550 || r.ByResource["c"] != 300 || r.Total != 850 {
		t.Errorf("byResource/total = %v, want m:550 c:300 total:850", r.ByResource)
	}
}

func TestComputeCostsFixedAndMilestonePerUseOnly(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks: []Task{
			costTask("A", 3, func(t *Task) { t.FixedCost = 120.5 }),
			costTask("M", 5, func(t *Task) { t.IsMilestone = true }),
		},
		Resources:   []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 300, CostPerUse: 200}},
		Assignments: []Assignment{{TaskID: "M", ResourceID: "r1"}},
	}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if r.Rows["A"] != (TaskCost{Fixed: 120.5, Assigned: 0, Total: 120.5}) {
		t.Errorf("Rows[A] = %+v", r.Rows["A"])
	}
	if r.Rows["M"] != (TaskCost{Fixed: 0, Assigned: 200, Total: 200}) {
		t.Errorf("Rows[M] = %+v, want 里程碑 effDur=0 只剩每次使用 200", r.Rows["M"])
	}
	if r.Total != 320.5 {
		t.Errorf("Total = %v, want 320.5", r.Total)
	}
}

func TestComputeCostsMirrorScenario(t *testing.T) {
	p := costProject()
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if len(r.Rows) != 4 {
		t.Fatalf("rows 只含叶任务，got %d 项", len(r.Rows))
	}
	if r.Rows["A"].Assigned != 2000 { // 3×2×300+200
		t.Errorf("Rows[A].Assigned = %v, want 2000", r.Rows["A"].Assigned)
	}
	if r.Rows["B"] != (TaskCost{Fixed: 50, Assigned: 850, Total: 900}) { // 10×55+300
		t.Errorf("Rows[B] = %+v", r.Rows["B"])
	}
	if r.ByResource["r1"] != 2200 || r.ByResource["r2"] != 550 || r.ByResource["r3"] != 300 {
		t.Errorf("byResource = %v, want r1:2200 r2:550 r3:300", r.ByResource)
	}
	if r.Total != 3100 { // 2000+900+0(C)+200(M)
		t.Errorf("Total = %v, want 3100", r.Total)
	}
}

func TestComputeCostsLegacyPlanZeroTotal(t *testing.T) {
	p := &Project{Name: "P", StartDate: "2026-01-05", Tasks: []Task{costTask("A", 3, nil)}}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if !r.OK || r.Total != 0 || len(r.ByResource) != 0 {
		t.Errorf("无资源维度旧计划应 ok/0/空，got %+v", r)
	}
}

func TestComputeCostsFailClosedOnCycle(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks:       []Task{costTask("A", 3, nil), costTask("B", 2, nil)},
		Links:       []Link{{From: "A", To: "B", Type: FS}, {From: "B", To: "A", Type: FS}},
		Resources:   []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100}},
		Assignments: []Assignment{{TaskID: "A", ResourceID: "r1"}},
	}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if r.OK || r.Total != 0 || len(r.Rows) != 0 {
		t.Errorf("循环依赖应 fail-closed，got %+v", r)
	}
}

func TestComputeCostsSkipsDanglingRefs(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks:       []Task{costTask("A", 3, nil)},
		Resources:   []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100}},
		Assignments: []Assignment{{TaskID: "A", ResourceID: "r1"}, {TaskID: "A", ResourceID: "ghost"}, {TaskID: "ghostTask", ResourceID: "r1"}},
	}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if r.Rows["A"].Assigned != 300 || len(r.ByResource) != 1 || r.ByResource["r1"] != 300 {
		t.Errorf("悬空引用应跳过，got rows:%+v byResource:%v", r.Rows["A"], r.ByResource)
	}
}

func TestComputeCostsRoundsToCent(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks:       []Task{costTask("A", 3, nil)},
		Resources:   []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 33.335}},
		Assignments: []Assignment{{TaskID: "A", ResourceID: "r1"}},
	}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if r.Rows["A"].Assigned != 100.01 || r.Total != 100.01 { // 3×33.335=100.005 → 100.01
		t.Errorf("Rows[A].Assigned/Total = %v/%v, want 100.01/100.01", r.Rows["A"].Assigned, r.Total)
	}
}

func TestComputeCostsZeroUnitsAndManualDuration(t *testing.T) {
	p := &Project{
		Name: "P", StartDate: "2026-01-05",
		Tasks:       []Task{costTask("A", 3, nil), costTask("B", 4, func(t *Task) { t.Mode = ModeManual; t.ManualStart = 2 })},
		Resources:   []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100}},
		Assignments: []Assignment{{TaskID: "A", ResourceID: "r1", Units: floatPtr(0)}, {TaskID: "B", ResourceID: "r1"}},
	}
	r := ComputeCosts(*p, ComputeCpm(p.Tasks, p.Links))
	if r.Rows["A"].Assigned != 0 {
		t.Errorf("units=0 应归零，got %v", r.Rows["A"].Assigned)
	}
	if r.Rows["B"].Assigned != 400 {
		t.Errorf("manual 工期照常计成本，got %v, want 400", r.Rows["B"].Assigned)
	}
	if math.Abs(r.Total-400) > 1e-9 {
		t.Errorf("Total = %v, want 400", r.Total)
	}
}

// ── 双工期口径（v4.150 刀1）：等效工作日跨度计费（镜像 TS cost.test.ts 双工期 describe）──

func TestComputeCostsCdSpan(t *testing.T) {
	// 养护 28cd 挂工时资源：按等效跨度 20 计费（周末不记工日，非 28）
	cdP := func(extra func(*Task)) Project {
		tk := costTask("养护", 28, func(x *Task) { x.DurationUnit = UnitCd })
		if extra != nil {
			extra(&tk)
		}
		return Project{
			Name:        "养护样板",
			StartDate:   "2026-09-07",
			Tasks:       []Task{tk},
			Resources:   []Resource{{ID: "r1", Name: "r1", Type: ResWork, StandardRate: 100}},
			Assignments: []Assignment{{TaskID: "养护", ResourceID: "r1"}},
		}
	}
	manual := func(x *Task) { x.Mode = ModeManual; x.ManualStart = 2 }
	auto := cdP(nil)
	r := ComputeCosts(auto, ComputeCpmCal(auto.Tasks, auto.Links, nil, auto.StartDate))
	if !r.OK || r.Rows["养护"].Assigned != 2000 {
		t.Errorf("28cd assigned = %v, want 2000（20 工日×100）", r.Rows["养护"].Assigned)
	}
	// wd 任务逐位不变：5wd×1×100=500（换源 ef−es 与 effDur 相等 pin）
	wdP := Project{
		Name:        "wd 样板",
		StartDate:   "2026-09-07",
		Tasks:       []Task{costTask("A", 5, nil)},
		Resources:   []Resource{{ID: "r1", Name: "r1", Type: ResWork, StandardRate: 100}},
		Assignments: []Assignment{{TaskID: "A", ResourceID: "r1"}},
	}
	rw := ComputeCosts(wdP, ComputeCpmCal(wdP.Tasks, wdP.Links, nil, wdP.StartDate))
	if !rw.OK || rw.Rows["A"].Assigned != 500 {
		t.Errorf("5wd assigned = %v, want 500", rw.Rows["A"].Assigned)
	}
	// manual cd：按锁定区间的等效跨度计费（manualStart=2，ef=22，跨度 20）
	man := cdP(manual)
	rm := ComputeCosts(man, ComputeCpmCal(man.Tasks, man.Links, nil, man.StartDate))
	if !rm.OK || rm.Rows["养护"].Assigned != 2000 {
		t.Errorf("manual cd assigned = %v, want 2000", rm.Rows["养护"].Assigned)
	}
}
