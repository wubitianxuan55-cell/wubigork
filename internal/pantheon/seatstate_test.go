package pantheon

import (
	"strings"
	"testing"
)

// 席位状态常量必须与 manager.go 里既有的裸字符串逐字一致，否则转移表通过的
// 状态和实际写进 session.json 的状态会是两套词表（2026-11-02 落地前逐条对过
// manager.go:410/412/694/761/1439/1546/1576）。
func TestSeatStatusConstantsMatchLegacyBareStrings(t *testing.T) {
	legacy := map[string]string{
		"waiting":     SeatWaiting,
		"running":     SeatRunning,
		"finished":    SeatFinished,
		"failed":      SeatFailed,
		"skipped":     SeatSkipped,
		"retrying":    SeatRetrying,
		"unavailable": SeatUnavailable,
		"cancelled":   SeatCancelled,
	}
	for bare, constant := range legacy {
		if constant != bare {
			t.Fatalf("席位状态常量漂移：%q != %q", constant, bare)
		}
	}
	if SeatNeedsInput != "needs_input" || SeatFinishedUnverified != "finished_unverified" {
		t.Fatalf("新增席位状态取值不符：%q / %q", SeatNeedsInput, SeatFinishedUnverified)
	}
	if StatusWaitingInput != "waiting_input" || StatusInterrupted != "interrupted" {
		t.Fatalf("新增会话状态取值不符：%q / %q", StatusWaitingInput, StatusInterrupted)
	}
	if EventQuestion != "question" || EventAnswer != "answer" || EventGate != "gate" {
		t.Fatalf("决策门事件 Kind 取值不符：%q / %q / %q", EventQuestion, EventAnswer, EventGate)
	}
	if GateActionOpen != "open" || GateActionResolve != "resolve" || GateActionDismiss != "dismiss" {
		t.Fatalf("决策门动作取值不符：%q / %q / %q", GateActionOpen, GateActionResolve, GateActionDismiss)
	}
}

// 转移表逐条覆盖：每条 allow 都要真允许，每条 forbid 都要真拒绝。
func TestCanTransitionSeat(t *testing.T) {
	cases := []struct {
		from, to string
		allow    bool
		why      string
	}{
		// waiting：开场前的可用性检查落这里。
		{SeatWaiting, SeatRunning, true, "可以开始"},
		{SeatWaiting, SeatSkipped, true, "开场前跳过"},
		{SeatWaiting, SeatUnavailable, true, "开场前不可用"},
		{SeatWaiting, SeatFinished, false, "没跑过不算完成"},
		{SeatWaiting, SeatWaiting, false, "非终态不自环"},
		{SeatWaiting, SeatFailed, false, "没跑过不算失败"},

		// running：收声、未验证收声、失败、需要输入、取消。
		{SeatRunning, SeatFinished, true, "正常收声"},
		{SeatRunning, SeatFinishedUnverified, true, "重启后未观测到收声"},
		{SeatRunning, SeatFailed, true, "调用失败"},
		{SeatRunning, SeatNeedsInput, true, "席位提问"},
		{SeatRunning, SeatCancelled, true, "用户取消"},
		{SeatRunning, SeatWaiting, false, "不许退回开场"},
		{SeatRunning, SeatRetrying, false, "重试要经 failed 或显式重建"},
		{SeatRunning, SeatRunning, false, "重复 started 不算新证据"},

		// needs_input：回答到达才回 running。
		{SeatNeedsInput, SeatRunning, true, "回答到达"},
		{SeatNeedsInput, SeatSkipped, true, "问题被弃置"},
		{SeatNeedsInput, SeatCancelled, true, "用户取消"},
		{SeatNeedsInput, SeatFinished, false, "问题没答完不许判完成"},
		{SeatNeedsInput, SeatFailed, false, "挂起不是失败"},
		{SeatNeedsInput, SeatNeedsInput, false, "重复提问不算新证据"},

		// finished_unverified：只有一个出口，且是唯一的升级通道。
		{SeatFinishedUnverified, SeatFinished, true, "同一 attempt 陈词被接受"},
		{SeatFinishedUnverified, SeatFailed, false, "没人看见不等于失败"},
		{SeatFinishedUnverified, SeatRunning, false, "不许原地复活"},
		{SeatFinishedUnverified, SeatFinishedUnverified, false, "非终态不自环（见 seatstate.go 注释）"},

		// retrying：重试也是席位状态，不是自由字符串。
		{SeatRetrying, SeatRunning, true, "重试开始"},
		{SeatRetrying, SeatFailed, true, "重试失败"},
		{SeatRetrying, SeatSkipped, true, "重试次数用尽后跳过"},
		{SeatRetrying, SeatFinished, false, "没跑完不许完成"},
		{SeatRetrying, SeatCancelled, true, "重试期间用户终止整场议事，席位跟着收口"},
		{SeatRetrying, SeatWaiting, false, "不许退回开场"},

		// 终态自环：幂等重放允许，出终态一律不允许。
		{SeatFinished, SeatFinished, true, "终态自环"},
		{SeatFailed, SeatFailed, true, "终态自环"},
		{SeatSkipped, SeatSkipped, true, "终态自环"},
		{SeatUnavailable, SeatUnavailable, true, "终态自环"},
		{SeatCancelled, SeatCancelled, true, "终态自环"},
		{SeatFinished, SeatRunning, false, "终态不许回到运行"},
		{SeatFinished, SeatFinishedUnverified, false, "已是确定完成"},
		{SeatFailed, SeatRunning, false, "终态不许回到运行"},
		{SeatSkipped, SeatRunning, false, "终态不许回到运行"},
		{SeatUnavailable, SeatRunning, false, "终态不许回到运行"},
		{SeatCancelled, SeatRunning, false, "终态不许回到运行"},

		// fail closed：未知状态不在表里，一律不允许。
		{"bogus", SeatRunning, false, "未知来源状态"},
		{SeatRunning, "bogus", false, "未知目标状态"},
		{"", SeatRunning, false, "空来源状态"},
	}
	for _, c := range cases {
		if got := CanTransitionSeat(c.from, c.to); got != c.allow {
			t.Fatalf("%s → %s 期望 allow=%v 实际 %v（%s）", c.from, c.to, c.allow, got, c.why)
		}
	}
}

func TestSeatTransitionError(t *testing.T) {
	if err := SeatTransitionError(SeatFinishedUnverified, SeatFinished); err != nil {
		t.Fatalf("唯一升级通道不该报错：%v", err)
	}
	if err := SeatTransitionError(SeatFinished, SeatFinished); err != nil {
		t.Fatalf("终态自环不该报错：%v", err)
	}

	err := SeatTransitionError(SeatFinished, SeatRunning)
	if err == nil {
		t.Fatal("finished → running 必须报错")
	}
	if want := "席位状态不能从 finished 变为 running"; err.Error() != want {
		t.Fatalf("错误信息应为人能读的中文：want %q got %q", want, err.Error())
	}

	for _, pair := range [][2]string{
		{SeatFinishedUnverified, SeatFailed},
		{SeatNeedsInput, SeatFinished},
		{SeatRunning, SeatRunning},
	} {
		if err := SeatTransitionError(pair[0], pair[1]); err == nil {
			t.Fatalf("%s → %s 必须报错", pair[0], pair[1])
		}
	}
	if err := SeatTransitionError(SeatFinishedUnverified, SeatFinishedUnverified); err == nil || !strings.Contains(err.Error(), "自环") {
		t.Fatalf("非终态自环应报「只有终态允许自环」，got %v", err)
	}
}

func TestCanTransitionSession(t *testing.T) {
	cases := []struct {
		from, to string
		allow    bool
		why      string
	}{
		{StatusRunning, StatusWaitingInput, true, "席位提问，议事挂起"},
		{StatusRunning, StatusCompleted, true, "全体结案"},
		{StatusRunning, StatusFailed, true, "议事失败"},
		{StatusRunning, StatusCancelled, true, "用户终止"},
		{StatusRunning, StatusInterrupted, true, "重启时没观测到收声"},
		{StatusRunning, StatusRunning, false, "非终态不自环"},
		{StatusWaitingInput, StatusRunning, true, "回答到达，议事继续"},
		{StatusWaitingInput, StatusCancelled, true, "用户终止"},
		{StatusWaitingInput, StatusInterrupted, true, "带问题重启"},
		{StatusWaitingInput, StatusCompleted, false, "问题没答完不许结案"},
		{StatusWaitingInput, StatusFailed, false, "挂起不是失败"},
		{StatusWaitingInput, StatusWaitingInput, false, "非终态不自环"},
		{StatusCompleted, StatusCompleted, true, "终态自环"},
		{StatusFailed, StatusFailed, true, "终态自环"},
		{StatusCancelled, StatusCancelled, true, "终态自环"},
		{StatusInterrupted, StatusInterrupted, true, "终态自环"},
		{StatusCompleted, StatusRunning, false, "终态不许重开"},
		{StatusCancelled, StatusRunning, false, "终态不许重开"},
		{StatusInterrupted, StatusRunning, false, "终态不许重开"},
		{StatusFailed, StatusRunning, false, "终态不许重开（恢复另走显式动作）"},
		{"bogus", StatusRunning, false, "未知来源状态"},
		{StatusRunning, "bogus", false, "未知目标状态"},
	}
	for _, c := range cases {
		if got := CanTransitionSession(c.from, c.to); got != c.allow {
			t.Fatalf("%s → %s 期望 allow=%v 实际 %v（%s）", c.from, c.to, c.allow, got, c.why)
		}
	}
}

func TestSessionTransitionError(t *testing.T) {
	if err := SessionTransitionError(StatusRunning, StatusWaitingInput); err != nil {
		t.Fatalf("running → waiting_input 不该报错：%v", err)
	}
	err := SessionTransitionError(StatusCompleted, StatusRunning)
	if err == nil {
		t.Fatal("completed → running 必须报错")
	}
	if want := "议事状态不能从 completed 变为 running"; err.Error() != want {
		t.Fatalf("错误信息应为人能读的中文：want %q got %q", want, err.Error())
	}
	if err := SessionTransitionError(StatusWaitingInput, StatusWaitingInput); err == nil || !strings.Contains(err.Error(), "自环") {
		t.Fatalf("非终态自环应报「只有终态允许自环」，got %v", err)
	}
}
