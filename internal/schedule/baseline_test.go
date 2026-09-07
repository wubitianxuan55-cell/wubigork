// schedule/baseline_test.go — 基线快照与漂移对比用例（v4.116.0 刀7）。
//
// 与前端 frontend/src/schedule/baseline.test.ts 互为镜像：同一批场景
// 同一批期望值，两侧引擎口径必须逐字段一致。
package schedule

import (
	"strings"
	"testing"
)

// 三任务串联样板：A(3) → B(2) → C(4)，总工期 9，全关键（镜像 TS chainProject）。
func chainProject() *Project {
	return &Project{
		Name:      "链式样板",
		StartDate: "2026-09-07",
		Tasks: []Task{
			{ID: "A", Name: "挖土", Duration: 3, Level: 1},
			{ID: "B", Name: "垫层", Duration: 2, Level: 1},
			{ID: "C", Name: "浇筑", Duration: 4, Level: 1},
		},
		Links: []Link{
			{From: "A", To: "B", Type: FS},
			{From: "B", To: "C", Type: FS},
		},
	}
}

func mustSnapshot(t *testing.T, p *Project, savedAt, name string) *Baseline {
	t.Helper()
	b, err := SnapshotBaseline(p, savedAt, name)
	if err != nil {
		t.Fatalf("SnapshotBaseline 意外失败：%v", err)
	}
	return b
}

func TestSnapshotBaselineLeafRows(t *testing.T) {
	p := chainProject()
	p.Tasks = append(p.Tasks, Task{ID: "M", Name: "验收", Duration: 1, Level: 1, IsMilestone: true})
	p.Links = append(p.Links, Link{From: "C", To: "M", Type: FS})
	b := mustSnapshot(t, p, "2026-09-06 10:00", "")
	if b.Name != "基线" {
		t.Errorf("基线名 = %q, want 基线", b.Name)
	}
	if b.SavedAt != "2026-09-06 10:00" || b.Duration != 9 {
		t.Errorf("SavedAt/Duration = %q/%d, want 2026-09-06 10:00/9", b.SavedAt, b.Duration)
	}
	if got := b.Rows["A"]; got != (BaselineRow{Name: "挖土", ES: 0, EF: 3, Dur: 3, Critical: true}) {
		t.Errorf("Rows[A] = %+v", got)
	}
	if got := b.Rows["C"]; got != (BaselineRow{Name: "浇筑", ES: 5, EF: 9, Dur: 4, Critical: true}) {
		t.Errorf("Rows[C] = %+v", got)
	}
	if got := b.Rows["M"]; got != (BaselineRow{Name: "验收", ES: 9, EF: 9, Dur: 0, Critical: true}) {
		t.Errorf("里程碑行 = %+v, want dur=0 es=ef=9", got)
	}
}

func TestSnapshotBaselineNameTrim(t *testing.T) {
	p := chainProject()
	if b := mustSnapshot(t, p, "2026-09-06 10:00", "  开工版  "); b.Name != "开工版" {
		t.Errorf("带空白基线名 = %q, want 开工版", b.Name)
	}
	if b := mustSnapshot(t, p, "2026-09-06 10:00", "   "); b.Name != "基线" {
		t.Errorf("空白基线名 = %q, want 基线", b.Name)
	}
}

func TestSnapshotBaselineCycleRejected(t *testing.T) {
	p := chainProject()
	p.Links = append(p.Links, Link{From: "C", To: "A", Type: FS})
	if _, err := SnapshotBaseline(p, "2026-09-06 10:00", ""); err == nil {
		t.Fatal("循环依赖应拒绝保存基线")
	} else if !strings.Contains(err.Error(), "循环依赖") {
		t.Errorf("错误应含「循环依赖」：%v", err)
	}
}

func TestSnapshotBaselineEmptyRejected(t *testing.T) {
	p := &Project{Name: "空", StartDate: "2026-09-07"}
	if _, err := SnapshotBaseline(p, "2026-09-06 10:00", ""); err == nil {
		t.Fatal("空计划应拒绝保存基线")
	} else if !strings.Contains(err.Error(), "没有叶任务") {
		t.Errorf("错误应含「没有叶任务」：%v", err)
	}
}

func TestDriftNoBaseline(t *testing.T) {
	p := chainProject()
	if d := ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links)); d != nil {
		t.Fatalf("无基线应返回 nil，got %+v", d)
	}
}

func TestDriftIdentical(t *testing.T) {
	p := chainProject()
	b := mustSnapshot(t, p, "2026-09-06 10:00", "")
	p.Baseline = b
	d := ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links))
	if d == nil {
		t.Fatal("有基线不应返回 nil")
	}
	if d.DurationDrift != 0 || d.SameCount != 3 || d.ShiftedCount != 0 || len(d.Rows) != 0 {
		t.Errorf("一致计划漂移应清零：dur=%d same=%d shifted=%d rows=%d", d.DurationDrift, d.SameCount, d.ShiftedCount, len(d.Rows))
	}
	if len(d.CriticalGained) != 0 || len(d.CriticalLost) != 0 {
		t.Errorf("关键链不应有变化：gained=%v lost=%v", d.CriticalGained, d.CriticalLost)
	}
}

func TestDriftCriticalLengthen(t *testing.T) {
	p := chainProject()
	p.Baseline = mustSnapshot(t, p, "2026-09-06 10:00", "")
	for i := range p.Tasks {
		if p.Tasks[i].ID == "A" {
			p.Tasks[i].Duration = 5
		}
	}
	d := ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links))
	if d.BaselineDuration != 9 || d.CurrentDuration != 11 || d.DurationDrift != 2 {
		t.Errorf("总工期漂移 = %d→%d (%d), want 9→11 (+2)", d.BaselineDuration, d.CurrentDuration, d.DurationDrift)
	}
	if d.ShiftedCount != 3 || d.AddedCount != 0 || d.RemovedCount != 0 {
		t.Errorf("计数 shifted/added/removed = %d/%d/%d, want 3/0/0", d.ShiftedCount, d.AddedCount, d.RemovedCount)
	}
	a := findRow(t, d, "A")
	if a.Kind != "shifted" || a.ESDrift != 0 || a.EFDrift != 2 || a.DurDrift != 2 {
		t.Errorf("A 行 = %+v, want shifted es+0 ef+2 dur+2", a)
	}
	if r := findRow(t, d, "C"); r.ESDrift != 2 {
		t.Errorf("C 推移 = %d, want 2", r.ESDrift)
	}
}

func TestDriftCriticalGained(t *testing.T) {
	// A(4)→C(4) 关键链总工期 8，B(4) 与 C 并行（SS0）：基线里 B 有 4 天
	// 总时差、非关键。压 A、C 到 2 天后总工期 4，B 追平总工期变关键
	// （es/ef/工期全没变，仅关键标记翻转——单标记变化也计为推移）。
	p := &Project{
		Name:      "并行样板",
		StartDate: "2026-09-07",
		Tasks: []Task{
			{ID: "A", Name: "挖土", Duration: 4, Level: 1},
			{ID: "B", Name: "预埋", Duration: 4, Level: 1},
			{ID: "C", Name: "浇筑", Duration: 4, Level: 1},
		},
		Links: []Link{
			{From: "A", To: "C", Type: FS},
			{From: "A", To: "B", Type: SS},
		},
	}
	base := ComputeCpm(p.Tasks, p.Links)
	if base.Duration != 8 || base.Rows["B"].Critical {
		t.Fatalf("样板前置条件破坏：基线总工期=%d B关键=%v", base.Duration, base.Rows["B"].Critical)
	}
	p.Baseline = mustSnapshot(t, p, "2026-09-06 10:00", "")
	for i := range p.Tasks {
		if p.Tasks[i].ID == "A" || p.Tasks[i].ID == "C" {
			p.Tasks[i].Duration = 2
		}
	}
	d := ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links))
	if d.DurationDrift != -4 {
		t.Errorf("总工期漂移 = %d, want -4", d.DurationDrift)
	}
	if len(d.CriticalGained) != 1 || d.CriticalGained[0] != "预埋" {
		t.Errorf("CriticalGained = %v, want [预埋]", d.CriticalGained)
	}
	r := findRow(t, d, "B")
	if r.Kind != "shifted" || r.ESDrift != 0 || r.EFDrift != 0 || r.DurDrift != 0 || !r.CriticalNow || r.CriticalBase {
		t.Errorf("B 行 = %+v, want shifted 仅关键标记翻转", r)
	}
}

func TestDriftAddedAndRemoved(t *testing.T) {
	p := chainProject()
	p.Baseline = mustSnapshot(t, p, "2026-09-06 10:00", "")
	// 新增 D：C→FS→D(2)
	p.Tasks = append(p.Tasks, Task{ID: "D", Name: "养护", Duration: 2, Level: 1})
	p.Links = append(p.Links, Link{From: "C", To: "D", Type: FS})
	d := ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links))
	if d.AddedCount != 1 || d.DurationDrift != 2 {
		t.Errorf("新增后 added/dur = %d/%d, want 1/2", d.AddedCount, d.DurationDrift)
	}
	r := findRow(t, d, "D")
	if r.Kind != "added" || r.Base != nil || r.ESDrift != 9 || r.Now == nil || r.Now.EF != 11 {
		t.Errorf("D 行 = %+v", r)
	}
	// 再移除 B：A→C 直连
	p.Tasks = []Task{p.Tasks[0], p.Tasks[2], p.Tasks[3]}
	p.Links = []Link{{From: "A", To: "C", Type: FS}, {From: "C", To: "D", Type: FS}}
	d = ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links))
	if d.RemovedCount != 1 || d.DurationDrift != 0 {
		t.Errorf("移除后 removed/dur = %d/%d, want 1/0", d.RemovedCount, d.DurationDrift)
	}
	r = findRow(t, d, "B")
	if r.Kind != "removed" || r.Name != "垫层" || r.Base == nil || r.Base.EF != 5 || r.Now != nil {
		t.Errorf("B 行 = %+v, want removed 名=垫层 base.ef=5 now=nil", r)
	}
}

func TestDriftRowsOnlyDrifted(t *testing.T) {
	p := chainProject()
	p.Baseline = mustSnapshot(t, p, "2026-09-06 10:00", "")
	// 只改 C 工期：A、B 排程不变（变化不回传），只有 C 进 Rows
	for i := range p.Tasks {
		if p.Tasks[i].ID == "C" {
			p.Tasks[i].Duration = 6
		}
	}
	d := ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links))
	if len(d.Rows) != 1 || d.Rows[0].ID != "C" || d.SameCount != 2 {
		t.Errorf("Rows = %+v same=%d, want 仅 C、same=2", d.Rows, d.SameCount)
	}
}

func TestApplyOpsBaselineChannel(t *testing.T) {
	p := chainProject()
	// 缺 savedAt 拒绝（引擎纯函数不取时钟）
	if _, err := ApplyOps(p, []Op{{Type: "set_baseline", BaselineName: "开工版"}}); err == nil {
		t.Fatal("set_baseline 缺 savedAt 应拒绝")
	}
	sums, err := ApplyOps(p, []Op{
		{Type: "set_baseline", BaselineName: "开工版", SavedAt: "2026-09-06 10:00"},
		{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: intPtr(5)}},
	})
	if err != nil {
		t.Fatalf("ops 意外失败：%v", err)
	}
	if !strings.Contains(sums[0], "保存基线「开工版」") || !strings.Contains(sums[0], "总工期 9 天") {
		t.Errorf("set_baseline 摘要 = %q", sums[0])
	}
	if p.Baseline == nil || p.Baseline.Name != "开工版" || p.Baseline.Duration != 9 {
		t.Fatalf("基线未按 ops 落盘：%+v", p.Baseline)
	}
	d := ComputeBaselineDrift(p, ComputeCpm(p.Tasks, p.Links))
	if d == nil || d.DurationDrift != 2 {
		t.Errorf("保存基线后再调整应能对比漂移：%+v", d)
	}
	// 清除
	if _, err := ApplyOps(p, []Op{{Type: "clear_baseline"}}); err != nil {
		t.Fatalf("clear_baseline 意外失败：%v", err)
	}
	if p.Baseline != nil {
		t.Fatal("清除后基线应为 nil")
	}
	if _, err := ApplyOps(p, []Op{{Type: "clear_baseline"}}); err == nil {
		t.Fatal("无基线再清除应报错（无事可做≠静默成功）")
	}
}

func findRow(t *testing.T, d *Drift, id string) DriftRow {
	t.Helper()
	for _, r := range d.Rows {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("漂移行 %q 不存在", id)
	return DriftRow{}
}

func intPtr(v int) *int { return &v }

// ── 双工期口径（v4.150 刀1）：基线 dur=等效工作日跨度 EF−ES（镜像 TS baseline.test.ts 双工期 describe）──

func cdCureProject() *Project {
	return &Project{
		Name:      "养护样板",
		StartDate: "2026-09-07",
		Tasks:     []Task{{ID: "HY", Name: "养护", Duration: 28, Level: 1, Progress: 0, DurationUnit: UnitCd}},
	}
}

func TestBaselineCdSpan(t *testing.T) {
	p := cdCureProject()
	b := mustSnapshot(t, p, "2026-09-07 10:00", "")
	row := b.Rows["HY"]
	if row.ES != 0 || row.EF != 20 || row.Dur != 20 || !row.Critical {
		t.Fatalf("快照行 = %+v, want dur=20（等效跨度，非自然日数 28）", row)
	}

	// 平移（manualStart 0→10）等效跨度不变：durDrift=0，shifted 仅因 es/ef
	moved := cdCureProject()
	moved.Tasks[0].Mode = ModeManual
	moved.Tasks[0].ManualStart = 10
	moved.Baseline = b
	d := ComputeBaselineDrift(moved, ComputeCpmCal(moved.Tasks, moved.Links, nil, moved.StartDate))
	if d == nil || len(d.Rows) != 1 {
		t.Fatalf("drift = %+v", d)
	}
	if got := d.Rows[0]; got.Kind != "shifted" || got.ESDrift != 10 || got.EFDrift != 10 || got.DurDrift != 0 {
		t.Fatalf("平移行 = %+v", got)
	}

	// 日历变化改等效跨度：durDrift 如实呈现（边界日期不变、工作日索引收缩 20→19）
	holiday := cdCureProject()
	holiday.Calendar = &Calendar{Workweek: []int{1, 2, 3, 4, 5}, Holidays: []string{"2026-09-16"}}
	holiday.Baseline = b
	dh := ComputeBaselineDrift(holiday, ComputeCpmCal(holiday.Tasks, holiday.Links, holiday.Calendar, holiday.StartDate))
	if dh == nil || len(dh.Rows) != 1 {
		t.Fatalf("holiday drift = %+v", dh)
	}
	if got := dh.Rows[0]; got.Kind != "shifted" || got.DurDrift != -1 {
		t.Fatalf("日历漂移行 = %+v, want durDrift=-1", got)
	}
}
