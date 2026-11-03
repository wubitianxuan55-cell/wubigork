package pantheon

import (
	"strings"
	"testing"
)

func TestParseAttestation(t *testing.T) {
	attestation := parseAttestation(`立场：有条件支持
理由：方向正确，但成本证据不足
条件：先完成 8 组对照；保留成本上限
相关分歧：pan_s-issue-1、pan_s-issue-2`, "beta", "BETA", "hash")
	if attestation.Position != AttestationConditional ||
		attestation.Reason == "" ||
		len(attestation.Conditions) != 2 ||
		len(attestation.IssueIDs) != 2 ||
		attestation.VerdictSHA256 != "hash" {
		t.Fatalf("attestation = %+v", attestation)
	}
}

func TestRedTeamRunsIndependentAttestationAndRevision(t *testing.T) {
	runner := defaultFakeRunner()
	for _, seat := range []string{"alpha", "beta"} {
		seatID := seat
		runner.outputs[seatID] = func(spec commandSpec) (string, error) {
			switch {
			case strings.Contains(spec.Stdin, "独立复核已出现反对意见"):
				return `{
  "summary":"修订结论：先补成本证据",
  "recommendation":"先完成成本对照，再决定",
  "revisionReason":"吸收 Beta 的成本证据缺口",
  "proposedActions":[{"action":"完成成本对照","acceptanceEvidence":"8 组原始记录"}]
}`, nil
			case strings.Contains(spec.Stdin, "你是众神殿的独立复核席"):
				if seatID == "beta" {
					return "立场：反对\n理由：成本证据不足\n条件：先补对照\n相关分歧：issue-1", nil
				}
				return "立场：支持\n理由：结论与现有证据一致\n条件：无\n相关分歧：无", nil
			case strings.Contains(spec.Stdin, "汇总人"):
				return `{
  "summary":"原结论",
  "recommendation":"推进试点",
  "proposedActions":[{"action":"小范围试点","acceptanceEvidence":"试点记录"}]
}`, nil
			default:
				return "参与讨论。\n动作：主张\n剩余分歧：无\n置信度：中", nil
			}
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic:           "红队评审",
		Preset:          PresetRedTeam,
		AgentIDs:        []string{"alpha", "beta"},
		ChairmanAgentID: "alpha",
		Rounds:          2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if len(done.Attestations) != 2 {
		t.Fatalf("attestations = %+v, want two independent reviews", done.Attestations)
	}
	var support, oppose int
	for _, attestation := range done.Attestations {
		switch attestation.Position {
		case AttestationSupport:
			support++
		case AttestationOppose:
			oppose++
		}
	}
	if support != 1 || oppose != 1 {
		t.Fatalf("positions = support %d / oppose %d: %+v", support, oppose, done.Attestations)
	}
	if done.Verdict == nil ||
		done.Verdict.RevisedFromSHA256 == "" ||
		!strings.Contains(done.Verdict.Recommendation, "成本对照") {
		t.Fatalf("revised verdict missing: %+v", done.Verdict)
	}
	if len(done.VerdictRevisions) != 1 ||
		done.VerdictRevisions[0].SHA256 != done.Verdict.RevisedFromSHA256 {
		t.Fatalf("revision history = %+v", done.VerdictRevisions)
	}
	signed, err := m.SignOff(session.ID, "accepted", "")
	if err != nil {
		t.Fatal(err)
	}
	if signed.SignOff == nil || signed.SignOff.VerdictSHA256 == "" {
		t.Fatalf("signoff hash missing: %+v", signed.SignOff)
	}
	if signed.SignOff.VerdictSHA256 == verdictFingerprint(signed.Verdict) {
		t.Fatal("signoff hash must include attestations and revision history")
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"### 独立确认", "### 修订历史", "成本证据不足"} {
		if !strings.Contains(export, want) {
			t.Fatalf("export missing %q:\n%s", want, export)
		}
	}
}
