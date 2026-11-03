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
