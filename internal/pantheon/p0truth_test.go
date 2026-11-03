package pantheon

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// 2026-11-03 V2 P0 真相单源化验收（pan_1793681319746 r6 擦除窗口清单驱动）：
//   - 用户插话写入活循环共享对象，不被下一个保存点擦除，且进入下一席 prompt；
//   - 活循环在场时 ResolveGate 拒绝（防双活循环并发写同一会话）。

func TestUserNoteSurvivesActiveLoopAndReachesNextSeat(t *testing.T) {
	block := make(chan struct{})
	var once sync.Once
	runner := defaultFakeRunner()
	// alpha 首席阻塞，给插话留窗口；放行后 alpha 完成，beta 的 prompt 应见插话。
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "第 1 轮开场陈述") {
			once.Do(func() { <-block })
		}
		return "alpha 开场观点。", nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		return "beta 吸收了插话。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "插话存活", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 等 alpha 真正开跑再插话（此时活循环持有 session 指针并高频保存）。
	deadline := time.Now().Add(2 * time.Second)
	noted := false
	for time.Now().Before(deadline) {
		if _, err := m.AddUserNote(session.ID, "请重点比较两案的运维成本"); err == nil {
			noted = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !noted {
		t.Fatal("user note never landed while alpha blocked")
	}
	close(block)
	done := waitForStatus(t, m, session.ID, StatusCompleted)

	// 存活：终态会话里插话仍在（未被保存点擦除）。
	survived := false
	for _, c := range done.Contributions {
		if c.AgentID == "user" && strings.Contains(c.Content, "运维成本") {
			survived = true
		}
	}
	if !survived {
		t.Fatal("user note was erased by the active loop (2026-11-03 P0 regression)")
	}

	// 交付：beta 的第 1 轮 prompt 含插话全文（drain 生效）。
	delivered := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "「BETA」") &&
			strings.Contains(call.Stdin, "请重点比较两案的运维成本") {
			delivered = true
		}
	}
	if !delivered {
		t.Fatal("user note never reached the next seat's prompt")
	}
}

func TestResolveGateRejectsWhileRunning(t *testing.T) {
	block := make(chan struct{})
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer close(block)
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "活跑解门拒绝", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForRunning := func() bool {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			m.mu.Lock()
			_, running := m.runs[session.ID]
			m.mu.Unlock()
			if running {
				return true
			}
			time.Sleep(10 * time.Millisecond)
		}
		return false
	}
	if !waitForRunning() {
		t.Fatal("session never started running")
	}
	if _, err := m.ResolveGate(session.ID, "resolve", ""); err == nil {
		t.Fatal("ResolveGate must reject while the run loop is alive (double-loop guard)")
	}
}

// 末轮覆盖投影（P0-C）：早轮成功+末轮失败的席位必须进签注缺口，
// Partial 必须为真——旧口径读累计 successful 会谎报全员在场。
func TestFinalRoundCoverageDrivesGapsAndPartial(t *testing.T) {
	runner := defaultFakeRunner()
	round := 0
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		round++
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论\nrecommendation: 建议", nil
		}
		if round >= 3 { // 末轮（第 2 轮）失败
			return "", nil
		}
		return "alpha 观点。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "末轮缺席", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil || !done.Verdict.Partial {
		t.Fatalf("verdict.Partial must be true when a seat fails the final round: %+v", done.Verdict)
	}
	foundGap := false
	for _, gap := range done.Gaps {
		if strings.Contains(gap, "末轮调用失败") {
			foundGap = true
		}
	}
	if !foundGap {
		t.Fatalf("final-round failure must enter signoff gaps: %v", done.Gaps)
	}
}

// 轮边界具名动作（P0-D）：第二轮席位状态不再冻结——running 写入当前 attempt，
// 不再产生「状态不能从 finished 变为 running」警告洪水。
func TestSeatUnfreezesAcrossRounds(t *testing.T) {
	runner := defaultFakeRunner()
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "跨轮解冻", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "状态未能更新") && strings.Contains(warning, "finished") {
			t.Fatalf("seat status must not be frozen across rounds anymore: %s", warning)
		}
	}
	for _, participant := range done.Participants {
		if participant.Status == SeatRunning {
			t.Fatalf("participant should end in a terminal state, got running: %+v", participant)
		}
	}
}
