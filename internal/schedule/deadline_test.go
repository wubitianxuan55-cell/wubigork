// schedule/deadline_test.go — 倒排校核用例（v4.117.0 刀8）。
//
// 与前端 frontend/src/schedule/deadline.test.ts 互为镜像：同一批场景
// 同一批期望值，两侧引擎口径必须逐字段一致。
package schedule

import (
	"testing"
)

// 三任务串联样板：A(3) → B(2) → C(4)，总工期 9，全关键（镜像 TS chainProject）。
func chainProjectDeadline() *Project {
	return &Project{
		Name:      "链式样板",
		StartDate: "2026-09-07", // 周一
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

func TestCheckDeadlineNilWithoutDeadline(t *testing.T) {
	p := chainProjectDeadline()
	if d := CheckDeadline(p, ComputeCpm(p.Tasks, p.Links)); d != nil {
		t.Fatalf("未设目标竣工应返回 nil，got %+v", d)
	}
}

func TestCheckDeadlineFeasible(t *testing.T) {
	p := chainProjectDeadline()
	p.Deadline = "2026-09-21" // 周一开工起第 11 个工作日（含两个周末）
	d := CheckDeadline(p, ComputeCpm(p.Tasks, p.Links))
	if d == nil {
		t.Fatal("有目标竣工不应返回 nil")
	}
	if d.TargetWorkdays != 11 || d.CurrentDuration != 9 || !d.Feasible || d.Overrun != -2 {
		t.Errorf("校核 = target:%d dur:%d feasible:%v overrun:%d, want 11/9/true/-2", d.TargetWorkdays, d.CurrentDuration, d.Feasible, d.Overrun)
	}
	want := []string{"挖土", "垫层", "浇筑"}
	if len(d.CriticalTasks) != 3 || d.CriticalTasks[0] != want[0] || d.CriticalTasks[2] != want[2] {
		t.Errorf("CriticalTasks = %v, want %v", d.CriticalTasks, want)
	}
}

func TestCheckDeadlineExactHit(t *testing.T) {
	p := chainProjectDeadline()
	p.Deadline = "2026-09-17" // 周四=第 9 个工作日
	d := CheckDeadline(p, ComputeCpm(p.Tasks, p.Links))
	if d.TargetWorkdays != 9 || !d.Feasible || d.Overrun != 0 {
		t.Errorf("压线校核 = target:%d feasible:%v overrun:%d, want 9/true/0", d.TargetWorkdays, d.Feasible, d.Overrun)
	}
}

func TestCheckDeadlineInfeasible(t *testing.T) {
	p := chainProjectDeadline()
	p.Deadline = "2026-09-15" // 第 7 个工作日
	d := CheckDeadline(p, ComputeCpm(p.Tasks, p.Links))
	if d.TargetWorkdays != 7 || d.Feasible || d.Overrun != 2 {
		t.Errorf("不可达校核 = target:%d feasible:%v overrun:%d, want 7/false/2", d.TargetWorkdays, d.Feasible, d.Overrun)
	}
}

func TestCheckDeadlineManualExcluded(t *testing.T) {
	p := chainProjectDeadline()
	p.Tasks[1].Mode = ModeManual
	p.Deadline = "2026-09-15"
	d := CheckDeadline(p, ComputeCpm(p.Tasks, p.Links))
	// B 手动锁定 0 起：忽略入边不回传约束 → A 也有时差，关键仅剩 C
	if len(d.CriticalTasks) != 1 || d.CriticalTasks[0] != "浇筑" {
		t.Errorf("手动任务应排除且不约束前置：%v", d.CriticalTasks)
	}
}

func TestCheckDeadlineBeforeStart(t *testing.T) {
	p := chainProjectDeadline()
	p.Deadline = "2026-09-06" // 开工前一日
	d := CheckDeadline(p, ComputeCpm(p.Tasks, p.Links))
	if d.TargetWorkdays != 0 || d.Feasible || d.Overrun != 9 {
		t.Errorf("早于开工 = target:%d feasible:%v overrun:%d, want 0/false/9", d.TargetWorkdays, d.Feasible, d.Overrun)
	}
}

func TestDeadlineWorkdaysCalendar(t *testing.T) {
	// 工作日竣工含当日；非工作日竣工回落；节假日不计
	if got := DeadlineWorkdays("2026-09-07", "2026-09-11", nil); got != 5 {
		t.Errorf("周五竣工 = %d, want 5", got)
	}
	if got := DeadlineWorkdays("2026-09-07", "2026-09-12", nil); got != 5 {
		t.Errorf("周六竣工回落周五 = %d, want 5", got)
	}
	holiday := DefaultCalendar()
	holiday.Holidays = []string{"2026-09-11"}
	if got := DeadlineWorkdays("2026-09-07", "2026-09-18", &holiday); got != 9 {
		t.Errorf("含节假日 = %d, want 9", got)
	}
	if got := DeadlineWorkdays("2026-09-10", "2026-09-09", nil); got != 0 {
		t.Errorf("早于开工 = %d, want 0", got)
	}
}

func TestApplyOpsSetMetaDeadline(t *testing.T) {
	p := chainProjectDeadline()
	// 非法格式（长度≠10）拒绝
	bad := "2026-9-15"
	if _, err := ApplyOps(p, []Op{{Type: "set_meta", Deadline: &bad}}); err == nil {
		t.Fatal("非法日期口径应拒绝")
	}
	sums, err := ApplyOps(p, []Op{{Type: "set_meta", Deadline: strPtr("2026-09-15")}})
	if err != nil {
		t.Fatalf("set_meta deadline 意外失败：%v", err)
	}
	if p.Deadline != "2026-09-15" || sums[0] != "目标竣工→2026-09-15" {
		t.Errorf("落盘/摘要 = %q/%q", p.Deadline, sums[0])
	}
	// 空串清除
	if _, err := ApplyOps(p, []Op{{Type: "set_meta", Deadline: strPtr("")}}); err != nil {
		t.Fatalf("清除意外失败：%v", err)
	}
	if p.Deadline != "" {
		t.Errorf("清除后应为空串，got %q", p.Deadline)
	}
	// nil 不动
	if _, err := ApplyOps(p, []Op{{Type: "set_meta", Name: "改名"}}); err != nil {
		t.Fatalf("nil deadline 意外失败：%v", err)
	}
	// Validate 非法落盘拒绝
	p.Deadline = "20260915"
	if err := Validate(p); err == nil {
		t.Fatal("Validate 应拒绝非法目标竣工")
	}
}

func strPtr(s string) *string { return &s }
