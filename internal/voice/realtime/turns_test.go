// turns_test.go — SpeculativeTurnTracker 语义固化（对齐 sidecar speculative_turns.py）。
package realtime

import (
	"testing"
	"time"
)

func TestStartAndListenContinuation(t *testing.T) {
	tr := NewSpeculativeTurns()
	id, rev := tr.StartTurn()
	if id != "turn_1" || rev != 0 {
		t.Fatalf("首轮=%s/%d want turn_1/0", id, rev)
	}
	if tr.Phase() != PhaseListening {
		t.Fatalf("相位=%s want listening", tr.Phase())
	}
	// LISTENING 中再次确认语音：原轮原 rev 继续
	id2, rev2, reopened := tr.SpeechStarted(1000)
	if id2 != "turn_1" || rev2 != 0 || reopened {
		t.Fatalf("LISTENING 续轮=%s/%d/%v", id2, rev2, reopened)
	}
}

func TestSoftEndGraceAndCommit(t *testing.T) {
	tr := NewSpeculativeTurns()
	tr.StartTurn()
	tr.SegmentFinalized(2000, 800, 0)
	if tr.Phase() != PhaseSoftEnded {
		t.Fatalf("相位=%s want soft_ended", tr.Phase())
	}
	// 宽限期内输出被 HOLD（hold_for_grace=true）
	g := tr.Gate("turn_1", 0, true, time.Time{}, false)
	if g.Action != GateHold {
		t.Fatalf("宽限期内应 HOLD got %s", g.Action)
	}
	// 宽限期内 commit=true 仍 HOLD（提交只在无悬挂时发生——python 语义）
	if g = tr.Gate("turn_1", 0, true, time.Time{}, true); g.Action != GateHold {
		t.Fatalf("宽限期内 commit 应仍 HOLD got %s", g.Action)
	}
	// 宽限过后 commit=true 提交并 ACCEPT
	time.Sleep(900 * time.Millisecond)
	if g = tr.Gate("turn_1", 0, true, time.Time{}, true); g.Action != GateAccept {
		t.Fatalf("宽限后 commit 应 ACCEPT got %s", g.Action)
	}
	if tr.Phase() != PhaseAnswering {
		t.Fatalf("提交后相位=%s want answering", tr.Phase())
	}
	// 提交后不可重开
	if tr.CanReopen(3000) {
		t.Fatal("已提交轮 can_reopen 应 false")
	}
	// close：仍 current 但 committed 移除 → phase CLOSED，is_latest 仍 true（current 语义）
	tr.Close("turn_1", 0)
	if tr.Phase() != PhaseClosed {
		t.Fatalf("关闭后相位=%s want closed", tr.Phase())
	}
	if !tr.IsLatest("turn_1", 0) {
		t.Fatal("关闭后的 current 轮 is_latest 应 true")
	}
	// 下一轮开始后旧轮变 stale
	tr.StartTurn()
	if tr.IsLatest("turn_1", 0) {
		t.Fatal("新轮开始后旧轮 is_latest 应 false")
	}
}

func TestReopenRevision(t *testing.T) {
	tr := NewSpeculativeTurns()
	tr.StartTurn()
	tr.SegmentFinalized(1000, 800, 0)
	// 候选持有（2s）
	if !tr.SpeechCandidateStarted(1100) {
		t.Fatal("候选应成立")
	}
	if !tr.HasSpeechCandidate() {
		t.Fatal("应持有候选")
	}
	// 候选存在 → gate（非宽限）HOLD
	if g := tr.Gate("turn_1", 0, false, time.Time{}, false); g.Action != GateHold {
		t.Fatalf("候选持有应 HOLD got %s", g.Action)
	}
	// 确认重开：同轮 rev+1
	id, rev, reopened := tr.SpeechStarted(1500)
	if id != "turn_1" || rev != 1 || !reopened {
		t.Fatalf("重开=%s/%d/%v want turn_1/1/true", id, rev, reopened)
	}
	// 重开后 grace/pending 清空，gate 直接 ACCEPT
	if g := tr.Gate("turn_1", 1, false, time.Time{}, false); g.Action != GateAccept {
		t.Fatalf("重开后应 ACCEPT got %s", g.Action)
	}
	// 7000ms 音频时钟上限：final 游标 1000ms，audioMs=9000 时不可重开
	tr.SegmentFinalized(1000, 800, 0)
	if tr.CanReopen(9000) {
		t.Fatal("超 7000ms 音频时钟 can_reopen 应 false")
	}
	if !tr.CanReopen(8000) {
		t.Fatal("边界 8000ms（差 7000）can_reopen 应 true")
	}
	// 超限后 speech_started 开新轮
	id3, _, reopened3 := tr.SpeechStarted(9000)
	if id3 != "turn_2" || reopened3 {
		t.Fatalf("超限应开新轮 got %s/%v", id3, reopened3)
	}
}

func TestGraceExtendOnlyAndProcessingDeadline(t *testing.T) {
	tr := NewSpeculativeTurns()
	tr.StartTurn()
	tr.SegmentFinalized(500, 800, 400) // incomplete → delay 400ms
	d1, ok := tr.ProcessingDeadline("turn_1", 0)
	if !ok || d1.Before(time.Now()) {
		t.Fatalf("处理死线异常: ok=%v d1=%v", ok, d1)
	}
	if _, ok := tr.ProcessingDeadline("turn_9", 0); ok {
		t.Fatal("非 current 不应有处理死线")
	}
	// 宽限延长-only：第二次 finalized 更短宽限不缩短死线
	grace1 := tr.Gate("turn_1", 0, true, time.Time{}, false)
	tr.SegmentFinalized(600, 100, 0) // 再 finalized（grace 800→但仅延长语义按 deadline 比较）
	grace2 := tr.Gate("turn_1", 0, true, time.Time{}, false)
	if grace2.Action == GateHold && grace1.Action == GateHold && grace2.Deadline.Before(grace1.Deadline) {
		t.Fatal("宽限死线只允许延长")
	}
}

func TestWaitForGateUnblocksOnCommit(t *testing.T) {
	tr := NewSpeculativeTurns()
	tr.StartTurn()
	tr.SegmentFinalized(0, 300, 0) // 300ms 宽限
	done := make(chan bool, 1)
	go func() {
		done <- tr.WaitForGate("turn_1", 0, true, time.Time{})
	}()
	time.Sleep(50 * time.Millisecond)
	tr.Gate("turn_1", 0, true, time.Time{}, true) // 提交 → 广播
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("提交后 WaitForGate 应 ACCEPT")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForGate 未被提交唤醒")
	}
}

func TestStaleDrop(t *testing.T) {
	tr := NewSpeculativeTurns()
	tr.StartTurn()
	tr.SegmentFinalized(0, 800, 0)
	tr.StartTurn() // turn_2
	if g := tr.Gate("turn_1", 0, false, time.Time{}, false); g.Action != GateDrop {
		t.Fatalf("旧轮 gate 应 DROP got %s", g.Action)
	}
	if !tr.IsLatest("turn_2", 0) {
		t.Fatal("current 应 latest")
	}
}
