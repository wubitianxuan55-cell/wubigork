package schedule

// cpm_test.go — ComputeCpm 用例与前端 cpm.test.ts 镜像（口径改动必须两侧同步）。

import (
	"strings"
	"testing"
)

func tsk(id string, dur int, extra func(*Task)) Task {
	t := Task{ID: id, Name: id, Duration: dur, Level: 1, Progress: 0}
	if extra != nil {
		extra(&t)
	}
	return t
}

func lnk(from, to string, typ LinkType, lag int) Link {
	return Link{From: from, To: to, Type: typ, Lag: lag}
}

func TestCpmChainAllCritical(t *testing.T) {
	r := ComputeCpm(
		[]Task{tsk("A", 3, nil), tsk("B", 2, nil), tsk("C", 4, nil)},
		[]Link{lnk("A", "B", FS, 0), lnk("B", "C", FS, 0)},
	)
	if !r.OK || r.Duration != 9 {
		t.Fatalf("ok=%v dur=%d", r.OK, r.Duration)
	}
	if got := r.Rows["A"]; got != (TaskCpm{ES: 0, EF: 3, LS: 0, LF: 3, TF: 0, FF: 0, Critical: true}) {
		t.Fatalf("A = %+v", got)
	}
	if got := r.Rows["B"]; got.EF != 5 || !got.Critical {
		t.Fatalf("B = %+v", got)
	}
	if got := r.Rows["C"]; got.EF != 9 || !got.Critical {
		t.Fatalf("C = %+v", got)
	}
}

func TestCpmParallelFloat(t *testing.T) {
	r := ComputeCpm(
		[]Task{tsk("A", 2, nil), tsk("B", 6, nil), tsk("C", 1, nil)},
		[]Link{lnk("A", "B", FS, 0), lnk("A", "C", FS, 0)},
	)
	if r.Duration != 8 || !r.Rows["B"].Critical {
		t.Fatalf("dur=%d B=%+v", r.Duration, r.Rows["B"])
	}
	if c := r.Rows["C"]; c.ES != 2 || c.EF != 3 || c.TF != 5 || c.FF != 5 {
		t.Fatalf("C = %+v", c)
	}
	if r.Rows["A"].FF != 0 {
		t.Fatalf("A.ff = %d", r.Rows["A"].FF)
	}
	// 逆推：C 的 LS=7/LF=8（镜像「非关键任务 LF 由后继约束」用例）
	if c := r.Rows["C"]; c.LS != 7 || c.LF != 8 {
		t.Fatalf("C ls/lf = %d/%d", c.LS, c.LF)
	}
}

func TestCpmEmpty(t *testing.T) {
	r := ComputeCpm(nil, nil)
	if !r.OK || r.Duration != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestCpmNoLinks(t *testing.T) {
	r := ComputeCpm([]Task{tsk("A", 5, nil), tsk("B", 9, nil), tsk("C", 2, nil)}, nil)
	if r.Duration != 9 || r.Rows["A"].TF != 4 || r.Rows["C"].Critical {
		t.Fatalf("%+v", r)
	}
}

func TestCpmLinkTypes(t *testing.T) {
	cases := []struct {
		name     string
		tasks    []Task
		links    []Link
		checkID  string
		es, ef   int
		duration int
	}{
		{"SS+时距", []Task{tsk("A", 10, nil), tsk("B", 5, nil)}, []Link{lnk("A", "B", SS, 4)}, "B", 4, 9, 10},
		{"FF", []Task{tsk("A", 10, nil), tsk("B", 2, nil)}, []Link{lnk("A", "B", FF, 0)}, "B", 8, 10, 10},
		{"SF里程碑", []Task{tsk("A", 5, nil), tsk("M", 0, func(x *Task) { x.IsMilestone = true })}, []Link{lnk("A", "M", SF, 5)}, "M", 5, 5, 5},
		{"FS正时距", []Task{tsk("A", 3, nil), tsk("B", 1, nil)}, []Link{lnk("A", "B", FS, 2)}, "B", 5, 6, 6},
		{"FS负时距", []Task{tsk("A", 6, nil), tsk("B", 4, nil)}, []Link{lnk("A", "B", FS, -2)}, "B", 4, 8, 8},
		{"混合SS+FF", []Task{tsk("A", 10, nil), tsk("B", 6, nil)}, []Link{lnk("A", "B", SS, 4), lnk("A", "B", FF, 2)}, "B", 6, 12, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := ComputeCpm(c.tasks, c.links)
			if !r.OK || r.Duration != c.duration {
				t.Fatalf("ok=%v dur=%d want %d", r.OK, r.Duration, c.duration)
			}
			if got := r.Rows[c.checkID]; got.ES != c.es || got.EF != c.ef {
				t.Fatalf("%s = %+v", c.checkID, got)
			}
		})
	}
}

func TestCpmMiddleFreeFloat(t *testing.T) {
	// A(1)→C(1)→D(5)；A→B(6)→D：D.es=max(2,7)=7；C.ef=2 → C.ff=5
	r := ComputeCpm(
		[]Task{tsk("A", 1, nil), tsk("B", 6, nil), tsk("C", 1, nil), tsk("D", 5, nil)},
		[]Link{lnk("A", "C", FS, 0), lnk("C", "D", FS, 0), lnk("A", "B", FS, 0), lnk("B", "D", FS, 0)},
	)
	if c := r.Rows["C"]; c.ES != 1 || c.EF != 2 || c.FF != 5 {
		t.Fatalf("C = %+v", c)
	}
	if !r.Rows["D"].Critical {
		t.Fatalf("D 应关键：%+v", r.Rows["D"])
	}
}

func TestCpmManualModes(t *testing.T) {
	// manual 锁定开始：入边不推它，后继以它的 EF 为约束
	manual := tsk("M", 3, func(x *Task) { x.Mode = ModeManual; x.ManualStart = 2 })
	r := ComputeCpm(
		[]Task{tsk("A", 10, nil), manual, tsk("B", 1, nil)},
		[]Link{lnk("A", "M", FS, 0), lnk("M", "B", FS, 0)},
	)
	if m := r.Rows["M"]; m.ES != 2 || m.EF != 5 || m.Critical {
		t.Fatalf("M = %+v", m)
	}
	if b := r.Rows["B"]; b.ES != 5 || b.EF != 6 {
		t.Fatalf("B = %+v", b)
	}
	if !r.Rows["A"].Critical || r.Duration != 10 {
		t.Fatalf("A=%+v dur=%d", r.Rows["A"], r.Duration)
	}

	// manual 不回传约束：P 的 LF 不被 M 的 LS 收小
	m2 := tsk("M", 1, func(x *Task) { x.Mode = ModeManual; x.ManualStart = 5 })
	r2 := ComputeCpm([]Task{tsk("P", 2, nil), m2}, []Link{lnk("P", "M", FS, 0)})
	if m := r2.Rows["M"]; m.ES != 5 || m.EF != 6 {
		t.Fatalf("M = %+v", m)
	}
	if p := r2.Rows["P"]; p.LF != 6 || p.TF != 4 || p.FF != 4 || p.Critical {
		t.Fatalf("P = %+v", p)
	}

	// manual 缺省 manualStart=0
	m3 := tsk("M", 4, func(x *Task) { x.Mode = ModeManual })
	r3 := ComputeCpm([]Task{m3}, nil)
	if m := r3.Rows["M"]; m.ES != 0 || m.EF != 4 || m.Critical {
		t.Fatalf("M = %+v", m)
	}
}

func TestCpmCycleFailClosed(t *testing.T) {
	r := ComputeCpm(
		[]Task{tsk("A", 1, nil), tsk("B", 1, nil), tsk("C", 1, nil)},
		[]Link{lnk("A", "B", FS, 0), lnk("B", "C", FS, 0), lnk("C", "A", FS, 0)},
	)
	if r.OK {
		t.Fatal("环应 fail-closed")
	}
	if !strings.Contains(r.Error, "循环依赖") {
		t.Fatalf("error = %q", r.Error)
	}
	if len(r.Cycle) != 3 {
		t.Fatalf("cycle = %v", r.Cycle)
	}
}

func TestCpmIgnoresSelfAndDangling(t *testing.T) {
	r := ComputeCpm(
		[]Task{tsk("A", 3, nil), tsk("B", 2, nil)},
		[]Link{lnk("A", "A", FS, 0), lnk("X", "B", FS, 0), lnk("A", "B", FS, 0)},
	)
	if !r.OK || r.Rows["B"].ES != 3 || r.Duration != 5 {
		t.Fatalf("%+v", r)
	}
}

func TestCpmMilestoneChain(t *testing.T) {
	r := ComputeCpm(
		[]Task{tsk("A", 4, nil), tsk("M", 0, func(x *Task) { x.IsMilestone = true }), tsk("B", 2, nil)},
		[]Link{lnk("A", "M", FS, 0), lnk("M", "B", FS, 0)},
	)
	if m := r.Rows["M"]; m.ES != 4 || m.EF != 4 || !m.Critical {
		t.Fatalf("M = %+v", m)
	}
	if r.Duration != 6 {
		t.Fatalf("dur = %d", r.Duration)
	}
}
