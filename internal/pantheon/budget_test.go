package pantheon

import (
	"strings"
	"testing"
	"time"
)

// grokRoundPrompt 从模型席捕获的提示词里取第 2 轮讨论 prompt。
func grokRoundPrompt(t *testing.T, backend *fakeGrokBackend) string {
	t.Helper()
	backend.mu.Lock()
	defer backend.mu.Unlock()
	for i := len(backend.prompts) - 1; i >= 0; i-- {
		if strings.Contains(backend.prompts[i], "补充论证") {
			return backend.prompts[i]
		}
	}
	t.Fatal("grok round-2 discussion prompt missing")
	return ""
}

// 端到端（补 6.4 如实缺口）：B4 降档档对模型席真实生效——模型席默认全量
// 窗口（sharedHistory=fullRecordBlock，带「你此前的发言」全文标注），高水位
// 后第 2 轮必须退回梯度（标注消失）。对照腿证明断言有判别力。
func TestBudgetDowngradeSwitchesModelSeatToGradient(t *testing.T) {
	newCouncilBackend := func(t *testing.T, budgetMinutes int, elapsed time.Duration) *fakeGrokBackend {
		t.Helper()
		backend := &fakeGrokBackend{}
		m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), newGrokAdapter(backend))
		defer m.Close()
		if budgetMinutes > 0 {
			m.budgetElapsed = func(*Session) time.Duration { return elapsed }
		}
		session, err := m.Start(StartRequest{
			Topic: "降档对照", AgentIDs: []string{"alpha", "grok"}, ChairmanAgentID: "alpha",
			Rounds: 2, TimeBudgetMinutes: budgetMinutes,
		})
		if err != nil {
			t.Fatal(err)
		}
		done := waitForStatus(t, m, session.ID, StatusCompleted)
		if done.Verdict == nil {
			t.Fatalf("no verdict: %+v", done.Error)
		}
		return backend
	}

	// 对照腿：无预算 → 模型席第 2 轮保持全量窗口形态。
	controlPrompt := grokRoundPrompt(t, newCouncilBackend(t, 0, 0))
	if !strings.Contains(controlPrompt, "（你此前的发言") {
		t.Fatalf("control round-2 prompt must keep the full-window form:\n%s", controlPrompt)
	}

	// 预算腿：85% 高水位 → 第 2 轮退回梯度，全文标注消失。
	downgraded := grokRoundPrompt(t, newCouncilBackend(t, 100, 85*time.Minute))
	if strings.Contains(downgraded, "（你此前的发言") {
		t.Fatalf("round-2 prompt must be downgraded from full window at ≥80%%:\n%s", downgraded)
	}
}

func TestBudgetPercentAndTier(t *testing.T) {
	if got := budgetPercent(0, time.Minute); got != 0 {
		t.Fatalf("zero elapsed percent = %d, want 0", got)
	}
	if got := budgetPercent(time.Minute, 0); got != 0 {
		t.Fatalf("no budget percent = %d, want 0", got)
	}
	if got := budgetPercent(30*time.Minute, 90*time.Minute); got != 33 {
		t.Fatalf("percent = %d, want 33", got)
	}
	if got := budgetPercent(2*time.Hour, time.Hour); got != 200 {
		t.Fatalf("overrun percent = %d, want 200 (unclamped on purpose)", got)
	}

	cases := []struct {
		elapsed time.Duration
		budget  time.Duration
		want    int
	}{
		{79 * time.Minute, 100 * time.Minute, 0},
		{80 * time.Minute, 100 * time.Minute, budgetHistoryTier},
		{89 * time.Minute, 100 * time.Minute, budgetHistoryTier},
		{90 * time.Minute, 100 * time.Minute, budgetRoundTier},
		{150 * time.Minute, 100 * time.Minute, budgetRoundTier},
		{time.Hour, 0, 0},
	}
	for _, tc := range cases {
		if got := budgetTier(tc.elapsed, tc.budget); got != tc.want {
			t.Fatalf("budgetTier(%v, %v) = %d, want %d", tc.elapsed, tc.budget, got, tc.want)
		}
	}
}

func TestBudgetDowngradeMode(t *testing.T) {
	cases := []struct {
		mode seatHistoryMode
		tier int
		want seatHistoryMode
	}{
		{historySeeded, budgetHistoryTier, historyGradient},
		{historyFullWindow, budgetHistoryTier, historyGradient},
		{historyFullWindow, budgetHistoryTier - 1, historyFullWindow},
		{historyDelta, budgetRoundTier, historyDelta},
		{historyGradient, budgetRoundTier, historyGradient},
	}
	for _, tc := range cases {
		if got := budgetDowngradeMode(tc.mode, tc.tier); got != tc.want {
			t.Fatalf("budgetDowngradeMode(%v, %d) = %v, want %v", tc.mode, tc.tier, got, tc.want)
		}
	}
}

func TestBudgetSignalsExport(t *testing.T) {
	var b strings.Builder
	noBudget := &Session{Request: StartRequest{Title: "无预算"}}
	writeBudgetSignals(&b, noBudget)
	if b.Len() != 0 {
		t.Fatalf("no-budget session must render no budget section, got %q", b.String())
	}

	b.Reset()
	session := &Session{
		Request:   StartRequest{Title: "预算会", TimeBudgetMinutes: 100},
		CreatedAt: 1_000, UpdatedAt: 30 * 60 * 1_000,
	}
	session.Events = append(session.Events, Event{Kind: EventBudgetStop, Round: 2, Text: budgetStopText(2, 91*time.Minute, 100*time.Minute)})
	writeBudgetSignals(&b, session)
	out := b.String()
	for _, want := range []string{"## 预算", "时间预算：100 分钟", "30.0 分钟", "收口：第 2 轮后触发预算收口"} {
		if !strings.Contains(out, want) {
			t.Fatalf("budget section missing %q:\n%s", want, out)
		}
	}
	if budgetStopRound(session) != 2 {
		t.Fatalf("budgetStopRound = %d, want 2", budgetStopRound(session))
	}
}

// 端到端：≥90% 档在轮间收口——第 1 轮后停，跳过轮间小结直接汇总，
// 事件与警示都在档，导出带预算节与收口停止信号。
func TestBudgetStopEndsCouncilEarly(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	m.budgetElapsed = func(*Session) time.Duration { return 95 * time.Minute }
	session, err := m.Start(StartRequest{
		Topic: "预算收口", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 4, TimeBudgetMinutes: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	// 第 1 轮是独立开场（Phase=opening），后续轮才是 discussion——按轮次判，
	// 不按相位名判；汇总（Round=0）不计。
	spoke := 0
	for _, contribution := range done.Contributions {
		if contribution.Phase == PhaseSynthesis {
			continue
		}
		if contribution.Round >= 2 {
			t.Fatalf("round %d (%s) must not run after budget stop", contribution.Round, contribution.Phase)
		}
		spoke++
	}
	if spoke == 0 {
		t.Fatal("round 1 contributions missing")
	}
	if budgetStopRound(done) != 1 {
		t.Fatalf("budget stop round = %d, want 1", budgetStopRound(done))
	}
	foundEvent := false
	for _, event := range done.Events {
		if event.Kind == EventBudgetStop && strings.Contains(event.Text, "预算收口") {
			foundEvent = true
		}
	}
	if !foundEvent {
		t.Fatalf("budget_stop event missing from events: %+v", done.Events)
	}
	foundWarning := false
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "预算收口") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("budget warning missing: %+v", done.Warnings)
	}
	exportDoc, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## 预算", "时间预算：100 分钟", "触发整场预算收口"} {
		if !strings.Contains(exportDoc, want) {
			t.Fatalf("export missing %q:\n%s", want, exportDoc)
		}
	}
}

// 预算内不变量：预算为 0 = 不限，无预算事件、无预算节，行为与落地前一致。
func TestZeroBudgetLeavesBehaviorUntouched(t *testing.T) {
	runner := consensusRunner("一致")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	m.budgetElapsed = func(*Session) time.Duration { return 10 * time.Hour }
	session, err := m.Start(StartRequest{
		Topic: "无预算", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if budgetStopRound(done) != 0 {
		t.Fatalf("zero budget must never emit budget_stop, got round %d", budgetStopRound(done))
	}
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "预算") {
			t.Fatalf("zero budget must not warn about budget: %q", warning)
		}
	}
	exportDoc, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(exportDoc, "## 预算") {
		t.Fatalf("zero budget session must not render a budget section:\n%s", exportDoc)
	}
}
