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

// ── v4.129.0 刀G：E1 四型 FF 口径 + G3 计划工期锚点（与 cpm.test.ts 镜像）──

func TestFreeFloatPartFourTypes(t *testing.T) {
	to := TaskCpm{ES: 21, EF: 25}
	cases := []struct {
		typ            LinkType
		lag            int
		efFrom, esFrom int
		want           int
	}{
		{FS, 2, 15, 10, 4},  // 21-2-15
		{SS, 6, 15, 10, 5},  // 21-6-10（旧版漏减 esFrom 得 15）
		{FF, 1, 15, 10, 9},  // 25-1-15
		{SF, 3, 15, 10, 12}, // 25-3-10（旧版漏减 esFrom 得 22）
	}
	for _, c := range cases {
		if got := freeFloatPart(Link{From: "a", To: "b", Type: c.typ, Lag: c.lag}, to, c.efFrom, c.esFrom); got != c.want {
			t.Fatalf("%s: got %d want %d", c.typ, got, c.want)
		}
	}
}

func TestCpmSSTaskFloatE1(t *testing.T) {
	// A(5)→P(10)FS；Q(20) 根；S2 前置=P SS+2 与 Q FS：ES(S2)=max(7,20)=20
	// FF(P)=min(32-15=17, 20-2-5=13)=13（旧版漏减 ES(P) 得 17）
	r := ComputeCpm(
		[]Task{tsk("A", 5, nil), tsk("P", 10, nil), tsk("Q", 20, nil), tsk("S2", 12, nil)},
		[]Link{lnk("A", "P", FS, 0), lnk("P", "S2", SS, 2), lnk("Q", "S2", FS, 0)},
	)
	if got := r.Rows["P"]; got.ES != 5 || got.EF != 15 || got.FF != 13 {
		t.Fatalf("P = %+v", got)
	}
	if r.Rows["P"].FF > r.Rows["P"].TF {
		t.Fatalf("定理 TF=0⇒FF=0 / FF≤TF 破坏: %+v", r.Rows["P"])
	}
}

func TestCpmSFTaskFloatE1(t *testing.T) {
	// A(5)→P(10)FS；Q(16) 根；S 前置=P SF+3 与 Q FS：ES(S)=16，EF(S)=18
	// T(20)←S：duration=38；FF(P)=min(38-15=23, 18-3-5=10)=10（旧版得 15）
	r := ComputeCpm(
		[]Task{tsk("A", 5, nil), tsk("P", 10, nil), tsk("Q", 16, nil), tsk("S", 2, nil), tsk("T", 20, nil)},
		[]Link{lnk("A", "P", FS, 0), lnk("P", "S", SF, 3), lnk("Q", "S", FS, 0), lnk("S", "T", FS, 0)},
	)
	if got := r.Rows["P"]; got.ES != 5 || got.EF != 15 || got.FF != 10 {
		t.Fatalf("P = %+v", got)
	}
}

func TestCpmPlanFinishAnchor(t *testing.T) {
	tasks := []Task{tsk("A", 3, nil), tsk("B", 4, nil), tsk("C", 5, nil)}
	links := []Link{lnk("A", "B", FS, 0), lnk("B", "C", FS, 0)}
	// planFinish=10 < Tc=12：绑定链负时差、关键=TF 最小集
	r := ComputeCpmPlan(tasks, links, 10)
	if r.Duration != 12 {
		t.Fatalf("dur = %d", r.Duration)
	}
	for _, id := range []string{"A", "B", "C"} {
		got := r.Rows[id]
		if got.TF != -2 || !got.Critical {
			t.Fatalf("%s = %+v", id, got)
		}
	}
	if r.Rows["C"].FF != -2 {
		t.Fatalf("C.FF = %d", r.Rows["C"].FF)
	}
	// planFinish=Tc：与无锚点一致（TF=0）
	r = ComputeCpmPlan(tasks, links, 12)
	if !r.Rows["A"].Critical || r.Rows["A"].TF != 0 || r.Rows["C"].FF != 0 {
		t.Fatalf("planFinish=Tc: A=%+v C=%+v", r.Rows["A"], r.Rows["C"])
	}
	// planFinish>Tc：不拉伸（锚点回落计算工期）
	r = ComputeCpmPlan(tasks, links, 15)
	if !r.Rows["A"].Critical || r.Rows["A"].TF != 0 {
		t.Fatalf("planFinish>Tc: A=%+v", r.Rows["A"])
	}
	// 无锚点：与旧口径完全一致
	r = ComputeCpm(tasks, links)
	if !r.Rows["B"].Critical || r.Rows["B"].ES != 3 || r.Rows["B"].EF != 7 {
		t.Fatalf("no anchor: B=%+v", r.Rows["B"])
	}
}

// ── 双工期口径（v4.150 刀1）：cd 任务 CPM（镜像 TS cpm.test.ts 双工期 describe）──

func cdTask(id string, dur int, extra func(*Task)) Task {
	return tsk(id, dur, func(x *Task) {
		x.DurationUnit = UnitCd
		if extra != nil {
			extra(x)
		}
	})
}

const cdMon = "2026-09-07" // 周一开工锚点（镜像 TS START）

func TestCpmCdCuring(t *testing.T) {
	r := ComputeCpmCal([]Task{cdTask("养护", 28, nil)}, nil, nil, cdMon)
	if !r.OK || r.Duration != 20 {
		t.Fatalf("ok=%v dur=%d, want 20", r.OK, r.Duration)
	}
	want := TaskCpm{ES: 0, EF: 20, LS: 0, LF: 20, TF: 0, FF: 0, Critical: true}
	if got := r.Rows["养护"]; got != want {
		t.Fatalf("养护 = %+v", got)
	}
}

func TestCpmCdMixedChain(t *testing.T) {
	r := ComputeCpmCal(
		[]Task{tsk("挖土", 5, nil), cdTask("养护", 28, nil), tsk("回填", 5, nil)},
		[]Link{lnk("挖土", "养护", FS, 0), lnk("养护", "回填", FS, 0)},
		nil, cdMon)
	if !r.OK || r.Duration != 30 {
		t.Fatalf("ok=%v dur=%d, want 30", r.OK, r.Duration)
	}
	if got := r.Rows["养护"]; got != (TaskCpm{ES: 5, EF: 25, LS: 5, LF: 25, TF: 0, FF: 0, Critical: true}) {
		t.Fatalf("养护 = %+v", got)
	}
	if got := r.Rows["挖土"]; got.EF != 5 || !got.Critical {
		t.Fatalf("挖土 = %+v", got)
	}
	if got := r.Rows["回填"]; got != (TaskCpm{ES: 25, EF: 30, LS: 25, LF: 30, TF: 0, FF: 0, Critical: true}) {
		t.Fatalf("回填 = %+v", got)
	}
}

func TestCpmCdSnapFold(t *testing.T) {
	r := ComputeCpmCal([]Task{cdTask("a", 26, nil), cdTask("b", 27, nil), cdTask("c", 28, nil)}, nil, nil, cdMon)
	if !r.OK || r.Duration != 20 {
		t.Fatalf("ok=%v dur=%d, want 20", r.OK, r.Duration)
	}
	for _, id := range []string{"a", "b", "c"} {
		if r.Rows[id].EF != 20 {
			t.Fatalf("%s ef = %d, want 20", id, r.Rows[id].EF)
		}
	}
}

func TestCpmCdHolidayCalendar(t *testing.T) {
	cal := &Calendar{Workweek: []int{1, 2, 3, 4, 5}, Holidays: []string{"2026-09-16"}}
	r := ComputeCpmCal([]Task{cdTask("养护", 9, nil)}, nil, cal, cdMon)
	if !r.OK || r.Rows["养护"].EF != 7 {
		t.Fatalf("ok=%v ef=%d, want 7", r.OK, r.Rows["养护"].EF)
	}
}

func TestCpmCdFsLag(t *testing.T) {
	r := ComputeCpmCal(
		[]Task{cdTask("养护", 28, nil), tsk("验收", 2, nil)},
		[]Link{lnk("养护", "验收", FS, 2)},
		nil, cdMon)
	if !r.OK || r.Duration != 24 {
		t.Fatalf("ok=%v dur=%d, want 24", r.OK, r.Duration)
	}
	if got := r.Rows["验收"]; got != (TaskCpm{ES: 22, EF: 24, LS: 22, LF: 24, TF: 0, FF: 0, Critical: true}) {
		t.Fatalf("验收 = %+v", got)
	}
	if got := r.Rows["养护"]; got.EF != 20 || got.LS != 0 || !got.Critical {
		t.Fatalf("养护 = %+v", got)
	}
}

func TestCpmCdFloatBranch(t *testing.T) {
	r := ComputeCpmCal([]Task{tsk("主线", 30, nil), cdTask("养护", 28, nil)}, nil, nil, cdMon)
	if !r.OK || r.Duration != 30 {
		t.Fatalf("ok=%v dur=%d, want 30", r.OK, r.Duration)
	}
	if !r.Rows["主线"].Critical {
		t.Fatalf("主线 = %+v", r.Rows["主线"])
	}
	if got := r.Rows["养护"]; got != (TaskCpm{ES: 0, EF: 20, LS: 10, LF: 30, TF: 10, FF: 10, Critical: false}) {
		t.Fatalf("养护 = %+v", got)
	}
}

func TestCpmCdBackwardFlat(t *testing.T) {
	r := ComputeCpmCal(
		[]Task{tsk("主线", 20, nil), cdTask("养护", 26, nil), tsk("收尾", 4, nil)},
		[]Link{lnk("主线", "收尾", FS, 0), lnk("养护", "收尾", FS, 0)},
		nil, cdMon)
	if !r.OK {
		t.Fatalf("ok=%v", r.OK)
	}
	// 养护 lf=20，26cd 平段 fwd(0..2)=20 → ls=2（非 0）
	if got := r.Rows["养护"]; got.ES != 0 || got.EF != 20 || got.LS != 2 || got.TF != 2 || got.Critical {
		t.Fatalf("养护 = %+v", got)
	}
	if got := r.Rows["主线"]; got.TF != 0 || !got.Critical {
		t.Fatalf("主线 = %+v", got)
	}
}

func TestCpmCdManual(t *testing.T) {
	r := ComputeCpmCal(
		[]Task{cdTask("养护", 28, func(x *Task) { x.Mode = ModeManual; x.ManualStart = 2 }), tsk("验收", 3, nil)},
		[]Link{lnk("养护", "验收", FS, 0)},
		nil, cdMon)
	if !r.OK || r.Duration != 25 {
		t.Fatalf("ok=%v dur=%d, want 25", r.OK, r.Duration)
	}
	if got := r.Rows["养护"]; got != (TaskCpm{ES: 2, EF: 22, LS: 2, LF: 22, TF: 0, FF: 0, Critical: false}) {
		t.Fatalf("养护 = %+v", got)
	}
	if got := r.Rows["验收"]; got.ES != 22 || got.EF != 25 || !got.Critical {
		t.Fatalf("验收 = %+v", got)
	}
}

func TestCpmCdFastPathBitwise(t *testing.T) {
	tasks := []Task{tsk("A", 3, nil), tsk("B", 2, nil), tsk("C", 4, nil)}
	links := []Link{lnk("A", "B", FS, 0), lnk("B", "C", FS, 0)}
	plain := ComputeCpm(tasks, links)
	holiday := &Calendar{Workweek: []int{1, 2, 3, 4, 5}, Holidays: []string{"2026-09-08"}}
	withCtx := ComputeCpmCal(tasks, links, holiday, cdMon)
	if withCtx.OK != plain.OK || withCtx.Duration != plain.Duration {
		t.Fatalf("快路径口径漂移：%+v vs %+v", withCtx, plain)
	}
	for id, want := range plain.Rows {
		if withCtx.Rows[id] != want {
			t.Fatalf("%s 快路径漂移：%+v vs %+v", id, withCtx.Rows[id], want)
		}
	}
	// 显式 wd 单位同样走快路径
	wdMarked := ComputeCpmCal([]Task{tsk("A", 3, func(x *Task) { x.DurationUnit = UnitWd })}, nil, nil, cdMon)
	plainWd := ComputeCpm([]Task{tsk("A", 3, nil)}, nil)
	if !wdMarked.OK || wdMarked.Duration != plainWd.Duration || wdMarked.Rows["A"] != plainWd.Rows["A"] {
		t.Fatalf("显式 wd 漂移：%+v vs %+v", wdMarked.Rows["A"], plainWd.Rows["A"])
	}
}

func TestCpmCdFailClosed(t *testing.T) {
	r := ComputeCpmCal([]Task{cdTask("养护", 28, nil)}, nil, nil, "")
	if r.OK || !strings.Contains(r.Error, "缺开工日期") {
		t.Fatalf("ok=%v err=%q", r.OK, r.Error)
	}
	ss := ComputeCpmCal([]Task{cdTask("养护", 28, nil), tsk("后续", 3, nil)}, []Link{lnk("养护", "后续", SS, 0)}, nil, cdMon)
	if ss.OK || !strings.Contains(ss.Error, "FS") {
		t.Fatalf("SS ok=%v err=%q", ss.OK, ss.Error)
	}
	ff := ComputeCpmCal([]Task{tsk("A", 3, nil), cdTask("养护", 28, nil)}, []Link{lnk("A", "养护", FF, 0)}, nil, cdMon)
	if ff.OK || !strings.Contains(ff.Error, "FS") {
		t.Fatalf("FF ok=%v err=%q", ff.OK, ff.Error)
	}
}

func TestCpmCdPlanFinish(t *testing.T) {
	r := ComputeCpmPlanCal(
		[]Task{tsk("挖土", 5, nil), cdTask("养护", 28, nil), tsk("回填", 5, nil)},
		[]Link{lnk("挖土", "养护", FS, 0), lnk("养护", "回填", FS, 0)},
		25, nil, cdMon)
	if !r.OK || r.Duration != 30 {
		t.Fatalf("ok=%v dur=%d, want 30", r.OK, r.Duration)
	}
	if got := r.Rows["养护"]; got.LS != 0 || got.TF != -5 || !got.Critical {
		t.Fatalf("养护 = %+v", got)
	}
	if got := r.Rows["挖土"]; got.TF != -5 || !got.Critical {
		t.Fatalf("挖土 = %+v", got)
	}
}
