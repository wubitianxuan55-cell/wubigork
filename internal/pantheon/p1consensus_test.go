package pantheon

import (
	"strings"
	"testing"
)

// 2026-11-03 V2 P1 停机判据+翻转举证验收（grok r2 实证洞的根修）：
//   - 自报「一致」但剩余分歧非「无」→ 不得停机；
//   - 同席 no→yes 翻转：正文带贡献 ID 证据句柄 → 静默；无句柄 → 从众嫌疑警告留痕。

func TestResidualDisagreementBlocksEarlyStop(t *testing.T) {
	runner := defaultFakeRunner()
	for id, original := range runner.outputs {
		output := original
		runner.outputs[id] = func(spec commandSpec) (string, error) {
			text, err := output(spec)
			if err == nil && strings.Contains(spec.Stdin, "全员一致检查") {
				// 全员自报一致，但都保留同一条实质分歧——旧门会停，新门必须读 Residual。
				text += "\n共识状态：一致\n剩余分歧：成本边界未收口"
			}
			return text, err
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "假一致", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.CurrentRound < 3 {
		t.Fatalf("residual disagreement must block early stop: stopped at round %d", done.CurrentRound)
	}
	for _, c := range done.Contributions {
		if c.Consensus == "yes" && c.Residual == "" {
			t.Fatalf("residual line must be captured on the contribution: %+v", c)
		}
	}
}

func TestFlipWithoutEvidenceIsFlagged(t *testing.T) {
	runner := defaultFakeRunner()
	round := 0
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论\nrecommendation: 建议", nil
		}
		round++
		if strings.Contains(spec.Stdin, "全员一致检查") {
			if round <= 1 {
				return "观点。\n共识状态：仍有分歧\n剩余分歧：无", nil
			}
			// 翻转：无任何贡献 ID 引用——从众嫌疑。
			return "我改主意了，同意。\n共识状态：一致\n剩余分歧：无", nil
		}
		return "alpha 观点。", nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论", nil
		}
		if strings.Contains(spec.Stdin, "全员一致检查") {
			return "同意。\n共识状态：一致\n剩余分歧：无", nil
		}
		return "beta 观点。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "翻转举证", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "beta",
		UntilConsensus: true, MaxRounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	flagged := false
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "从众嫌疑") || strings.Contains(warning, "未引用说服它的证据") {
			flagged = true
		}
	}
	if !flagged {
		t.Fatalf("unevidenced flip must be flagged in warnings: %v", done.Warnings)
	}
}

func TestFlipWithEvidenceStaysSilent(t *testing.T) {
	runner := defaultFakeRunner()
	round := 0
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论", nil
		}
		round++
		if strings.Contains(spec.Stdin, "全员一致检查") {
			if round <= 1 {
				return "观点。\n共识状态：仍有分歧\n剩余分歧：无", nil
			}
			return "被 pan_stream-beta-r1-123 说服，同意。\n共识状态：一致\n剩余分歧：无", nil
		}
		return "alpha 观点。", nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论", nil
		}
		if strings.Contains(spec.Stdin, "全员一致检查") {
			return "同意。\n共识状态：一致\n剩余分歧：无", nil
		}
		return "beta 观点。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "翻转有举证", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "beta",
		UntilConsensus: true, MaxRounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "从众嫌疑") {
			t.Fatalf("evidenced flip must not be flagged: %s", warning)
		}
	}
}

// 分歧台账全生命周期（2026-11-03 V2 P1）：Residual 自动开条→下轮 prompt 注入→
// 撤回闭合（resolved 带轮次）→未撤回的结案转 dissent 永久入档。
func TestIssueLedgerLifecycle(t *testing.T) {
	runner := defaultFakeRunner()
	calls := 0
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论\nrecommendation: 建议", nil
		}
		calls++
		if strings.Contains(spec.Stdin, "全员一致检查") {
			if calls <= 1 {
				return "观点。\n共识状态：仍有分歧\n剩余分歧：接口契约未冻结", nil
			}
			return "已解决。\n共识状态：一致\n剩余分歧：无", nil
		}
		return "alpha 观点。", nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论", nil
		}
		if strings.Contains(spec.Stdin, "全员一致检查") {
			return "同意，但保留一条。\n共识状态：一致\n剩余分歧：成本边界", nil
		}
		return "beta 观点。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "台账生命周期", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "beta",
		UntilConsensus: true, MaxRounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)

	var resolved, dissent int
	for _, issue := range done.Issues {
		switch issue.Status {
		case "resolved":
			resolved++
			if issue.AgentID != "alpha" || issue.ResolvedIn < 2 {
				t.Fatalf("alpha's issue must resolve in a later round: %+v", issue)
			}
		case "dissent":
			dissent++
		}
	}
	if resolved != 1 || dissent != 1 {
		t.Fatalf("ledger = resolved %d / dissent %d, want 1/1:\n%+v", resolved, dissent, done.Issues)
	}

	// beta 的分歧保留 → 假一致门拦截 → 没有提前停在共识轮。
	if done.CurrentRound < 3 {
		t.Fatalf("beta's residual must block early stop: stopped at %d", done.CurrentRound)
	}

	// 第 2 轮起的 prompt 注入了 alpha 的 open 分歧（撤回前可见）。
	injected := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "【分歧台账（open）】") &&
			strings.Contains(call.Stdin, "接口契约未冻结") {
			injected = true
		}
	}
	if !injected {
		t.Fatal("open issues must be injected into later-round prompts")
	}

	// 汇总席看到台账去向要求；导出带台账段。
	sawLedgerInSynthesis := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "分歧台账（按条交代去向）") {
			sawLedgerInSynthesis = true
		}
	}
	if !sawLedgerInSynthesis {
		t.Fatal("synthesis prompt must carry the issue ledger")
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(export, "### 分歧台账") || !strings.Contains(export, "保留异议至收口") {
		t.Fatalf("export must carry the ledger:\n%.400s", export)
	}
}
