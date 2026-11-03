package pantheon

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// 2026-11-03 并行开场刀（用户拍板「第一轮互相看不到，可不可以同时进行」）验收基线：
//   - 独立开场第 1 轮多席执行区间重叠（并行实证，比 wall-time 断言稳健）；
//   - 全部贡献入库且落盘序=完成序（与直播一致）；
//   - 并行腿中 skip 按席定位（多槽），被跳席无贡献、其余席正常；
//   - 圆桌接力第 1 轮保持串行（区间不重叠）。

type seatSpan struct {
	start int64
	end   int64
}

func TestIndependentOpeningRunsSeatsInParallel(t *testing.T) {
	runner := defaultFakeRunner()
	var mu sync.Mutex
	spans := map[string]seatSpan{}
	for _, seat := range []string{"alpha", "beta"} {
		runner.outputs[seat] = func(spec commandSpec) (string, error) {
			// 汇总席复用同一 fn：只对讨论腿记录区间，否则第二次调用覆盖 opening span。
			if strings.Contains(spec.Stdin, "汇总人") {
				return seat + " 汇总完成。", nil
			}
			mu.Lock()
			spans[seat] = seatSpan{start: time.Now().UnixMilli()}
			mu.Unlock()
			time.Sleep(220 * time.Millisecond)
			mu.Lock()
			current := spans[seat]
			current.end = time.Now().UnixMilli()
			spans[seat] = current
			mu.Unlock()
			return seat + " 的独立开场观点。", nil
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "并行开场", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	mu.Lock()
	alpha, beta := spans["alpha"], spans["beta"]
	mu.Unlock()
	if alpha.start == 0 || beta.start == 0 || alpha.end == 0 || beta.end == 0 {
		t.Fatalf("seat spans incomplete: %+v", spans)
	}
	// 区间重叠证明：两席的执行窗口必须互相覆盖对方的中点附近——串行下
	// 后位 start ≥ 前位 end，永不重叠（220ms sleep，串行总时长 ≥440ms）。
	overlap := alpha.start < beta.end && beta.start < alpha.end
	if !overlap {
		t.Fatalf("seats did not run in parallel: alpha=[%d,%d] beta=[%d,%d]", alpha.start, alpha.end, beta.start, beta.end)
	}
	notes := 0
	for _, contribution := range done.Contributions {
		if contribution.Phase == PhaseOpening {
			notes++
		}
	}
	if notes != 2 {
		t.Fatalf("opening contributions = %d, want 2 (all seats must land)", notes)
	}
}

func TestParallelOpeningSkipTargetsSeat(t *testing.T) {
	runner := defaultFakeRunner()
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		time.Sleep(400 * time.Millisecond) // 给测试留出 skip 窗口
		return "alpha 不该被入库。", nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		return "beta 并行完成。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "并行跳过", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 等 alpha 真正启动（participant running / agent_started 事件），再按席跳过。
	deadline := time.Now().Add(2 * time.Second)
	skipped := false
	for time.Now().Before(deadline) {
		if err := m.SkipAgent(session.ID, "alpha"); err == nil {
			skipped = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !skipped {
		t.Fatal("skip on parallel seat never succeeded")
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	for _, contribution := range done.Contributions {
		if contribution.AgentID == "alpha" && contribution.Phase == PhaseOpening {
			t.Fatalf("skipped seat must not land a contribution: %+v", contribution)
		}
	}
	betaOK := false
	for _, contribution := range done.Contributions {
		if contribution.AgentID == "beta" && contribution.Phase == PhaseOpening {
			betaOK = true
		}
	}
	if !betaOK {
		t.Fatal("parallel sibling contribution missing after skip")
	}
}

func TestCouncilOpeningRemainsSequential(t *testing.T) {
	runner := defaultFakeRunner()
	var mu sync.Mutex
	spans := map[string]seatSpan{}
	for _, seat := range []string{"alpha", "beta"} {
		runner.outputs[seat] = func(spec commandSpec) (string, error) {
			if strings.Contains(spec.Stdin, "汇总人") {
				return seat + " 汇总完成。", nil
			}
			mu.Lock()
			spans[seat] = seatSpan{start: time.Now().UnixMilli()}
			mu.Unlock()
			time.Sleep(80 * time.Millisecond)
			mu.Lock()
			current := spans[seat]
			current.end = time.Now().UnixMilli()
			spans[seat] = current
			mu.Unlock()
			return seat + " 圆桌接力发言。", nil
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "接力串行", Protocol: ProtocolCouncil,
		AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	mu.Lock()
	alpha, beta := spans["alpha"], spans["beta"]
	mu.Unlock()
	// 串行判据（完整区间重叠）：beta 启动于 alpha 结束前且 alpha 启动于 beta
	// 结束前才构成重叠；接力下后位 start ≥ 前位 end，两条件不可能同时成立。
	overlapped := alpha.start < beta.end && beta.start < alpha.end
	if overlapped {
		t.Fatalf("council opening overlapped: alpha=[%d,%d] beta=[%d,%d]", alpha.start, alpha.end, beta.start, beta.end)
	}
	if len(done.Contributions) == 0 {
		t.Fatal("no contributions")
	}
}

// 空结果自动重试（2026-11-03 并行实弹）：zcode 中途夭折零正文=11-02 在册老病,
// 偶发——自动重试一次须留痕且成功入库;连续两次空才判失败。
func TestEmptyResultAutoRetriesOnce(t *testing.T) {
	runner := defaultFakeRunner()
	calls := map[string]int{}
	var mu sync.Mutex
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "alpha 汇总完成。", nil
		}
		mu.Lock()
		calls["alpha"]++
		n := calls["alpha"]
		mu.Unlock()
		if n == 1 {
			return "", nil // 第一遍夭折：空正文 → 「Agent 返回空结果」
		}
		return "alpha 重试后正常陈词。", nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		return "beta 正常陈词。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "空结果自愈", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	mu.Lock()
	n := calls["alpha"]
	mu.Unlock()
	if n != 2 {
		t.Fatalf("alpha calls = %d, want 2 (auto retry once)", n)
	}
	retryEvent := false
	for _, event := range done.Events {
		if event.Kind == EventAgentRetry && event.AgentID == "alpha" {
			retryEvent = true
		}
	}
	if !retryEvent {
		t.Fatal("auto retry must be visible as agent_retry event")
	}
	landed := false
	for _, contribution := range done.Contributions {
		if contribution.AgentID == "alpha" && strings.Contains(contribution.Content, "重试后正常陈词") {
			landed = true
		}
	}
	if !landed {
		t.Fatal("retried contribution must land")
	}
}

func TestEmptyResultFailsAfterSecondBlank(t *testing.T) {
	runner := defaultFakeRunner()
	runner.outputs["alpha"] = func(commandSpec) (string, error) {
		return "", nil // 两遍都空：重试后仍失败，走 agent_error
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "连续空失败", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	blank := 0
	for _, contribution := range done.Contributions {
		if contribution.AgentID == "alpha" && contribution.Phase == PhaseOpening {
			if strings.TrimSpace(contribution.Content) == "" {
				blank++
			}
		}
	}
	if blank != 0 {
		t.Fatalf("empty contributions must not land: %d", blank)
	}
	retries := 0
	for _, event := range done.Events {
		if event.Kind == EventAgentRetry && event.AgentID == "alpha" {
			retries++
		}
	}
	if retries != 1 {
		t.Fatalf("auto retries = %d, want exactly 1", retries)
	}
}
