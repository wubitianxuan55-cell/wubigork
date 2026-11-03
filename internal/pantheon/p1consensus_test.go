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
				text += "\n共识状态：一致\n剩余分歧：成本边界未收口" +
					"\n分歧依据：当前只有单席估算【未核验】" +
					"\n翻转条件：完成同任务对照" +
					"\n下一步核验：由成员记录 8 组同任务成本"
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
				return `观点。
共识状态：仍有分歧
剩余分歧：接口契约未冻结
分歧依据：pan_stream-beta-r1-123 的接口差异
翻转条件：接口差异低于 3 项且迁移验证通过
下一步核验：由 alpha 用 v2 客户端与 schema 对拍`, nil
			}
			return `已解决。
被 pan_stream-beta-r2-456 的迁移结果说服。
共识状态：一致
剩余分歧：无
撤回依据：pan_stream-beta-r2-456 的迁移结果`, nil
		}
		return "alpha 观点。", nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论", nil
		}
		if strings.Contains(spec.Stdin, "全员一致检查") {
			return `同意，但保留一条。
共识状态：一致
剩余分歧：成本边界
分歧依据：当前仅有单席估算【未核验】
翻转条件：完成 8 组同任务报价对照
下一步核验：由 beta 记录四席与单席的调用成本`, nil
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
			if issue.ResolutionEvidence == "" || issue.Disposition == "" || issue.Blocking {
				t.Fatalf("alpha's resolution must preserve evidence and disposition: %+v", issue)
			}
		case "dissent":
			dissent++
			if issue.Evidence == "" || issue.FlipCondition == "" || issue.Verification == "" {
				t.Fatalf("dissent must preserve its structured audit fields: %+v", issue)
			}
			if !issue.Blocking || issue.Disposition != "讨论结束仍保留" {
				t.Fatalf("dissent must remain blocking and carry final disposition: %+v", issue)
			}
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
	injectedDetails := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "【分歧台账（open）】") &&
			strings.Contains(call.Stdin, "接口契约未冻结") {
			injected = true
			if strings.Contains(call.Stdin, "翻转条件=接口差异低于 3 项且迁移验证通过") &&
				strings.Contains(call.Stdin, "下一步核验=由 alpha 用 v2 客户端与 schema 对拍") {
				injectedDetails = true
			}
		}
	}
	if !injected {
		t.Fatal("open issues must be injected into later-round prompts")
	}
	if !injectedDetails {
		t.Fatal("open issue prompts must carry flip condition and verification path")
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
	if !strings.Contains(export, "- 翻转条件：完成 8 组同任务报价对照") ||
		!strings.Contains(export, "- 下一步核验：由 beta 记录四席与单席的调用成本") {
		t.Fatalf("export must carry structured issue fields:\n%.800s", export)
	}

	// 档案栏摘要只带计数，不复制台账正文：打开议题前也能判断是否还有待收口项。
	summaries, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %d, want 1", len(summaries))
	}
	if got := summaries[0]; got.IssueCount != 2 || got.OpenIssueCount != 0 || got.DissentCount != 1 {
		t.Fatalf("issue summary = %+v, want total=2 open=0 dissent=1", got)
	}
}

func TestIssueLedgerCapturesStructuredDisagreementInFixedRounds(t *testing.T) {
	runner := defaultFakeRunner()
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") {
			return "summary: 结论\nrecommendation: 建议", nil
		}
		return `固定轮次仍保留一个覆盖缺口。
剩余分歧：固定轮次可能提前收口
分歧依据：pan_stream-beta-r1-123 的缺席反例【他人核验】
翻转条件：补充一轮反例测试且无新反例
下一步核验：由 alpha 在独立工作区重跑 10 次`, nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "固定轮次台账", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "beta", Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if len(done.Issues) != 1 {
		t.Fatalf("fixed-round ledger issues = %+v, want one", done.Issues)
	}
	issue := done.Issues[0]
	if issue.Status != "dissent" || issue.LastSeenIn != 2 {
		t.Fatalf("issue lifecycle = %+v", issue)
	}
	if issue.Origin != "residual" {
		t.Fatalf("fixed-round issue origin = %q, want residual", issue.Origin)
	}
	if issue.Evidence == "" || issue.FlipCondition == "" || issue.Verification == "" {
		t.Fatalf("structured fields missing: %+v", issue)
	}
}

func TestContributionContractRequiresIssueAuditFields(t *testing.T) {
	contract := contributionContract()
	for _, want := range []string{
		"剩余分歧：无",
		"分歧依据：",
		"分歧对象：",
		"翻转条件：",
		"下一步核验：",
		"每轮另写一行讨论动作",
		"目标贡献：",
		"被挑战命题：",
		"回应结论：",
	} {
		if !strings.Contains(contract, want) {
			t.Fatalf("contribution contract missing %q:\n%s", want, contract)
		}
	}
}
