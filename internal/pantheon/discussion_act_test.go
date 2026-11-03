package pantheon

import (
	"regexp"
	"strings"
	"testing"
)

func TestExtractDiscussionActDetails(t *testing.T) {
	raw := `成本边界还缺少实测。
动作：质疑
目标贡献：pan_s-beta-r1-123
被挑战命题：两轮讨论一定优于单席直答
证据句柄：README.md 第 12 行
回应结论：部分接受
目标分歧：pan_s-issue-4
剩余分歧：成本未核验`

	details := extractDiscussionActDetails(raw)
	if details.DiscussionAct != DiscussionActChallenge {
		t.Fatalf("act = %q, want challenge", details.DiscussionAct)
	}
	if details.TargetContributionID != "pan_s-beta-r1-123" {
		t.Fatalf("target contribution = %q", details.TargetContributionID)
	}
	if details.TargetIssueID != "pan_s-issue-4" {
		t.Fatalf("target issue = %q", details.TargetIssueID)
	}
	if details.ChallengedClaim != "两轮讨论一定优于单席直答" {
		t.Fatalf("challenged claim = %q", details.ChallengedClaim)
	}
	if details.EvidenceHandle != "README.md 第 12 行" {
		t.Fatalf("evidence handle = %q", details.EvidenceHandle)
	}
	if details.ResponseVerdict != ResponsePartiallyAccept {
		t.Fatalf("response = %q, want partial", details.ResponseVerdict)
	}
	if strings.Contains(details.Text, "动作：") || strings.Contains(details.Text, "目标贡献：") {
		t.Fatalf("protocol fields must be stripped from prose:\n%s", details.Text)
	}
	if !strings.Contains(details.Text, "剩余分歧：成本未核验") {
		t.Fatalf("non-action protocol lines must remain for P1 parser:\n%s", details.Text)
	}
}

func TestDiscussionActLabelsNormalize(t *testing.T) {
	tests := map[string]string{
		"主张":        DiscussionActClaim,
		"challenge": DiscussionActChallenge,
		"回应":        DiscussionActRespond,
		"修订":        DiscussionActRevise,
		"补强":        DiscussionActReinforce,
		"拓宽":        DiscussionActBroaden,
		"乱写":        "",
	}
	for raw, want := range tests {
		if got := normalizeDiscussionAct(raw); got != want {
			t.Fatalf("normalizeDiscussionAct(%q) = %q, want %q", raw, got, want)
		}
	}
	if got := normalizeResponseVerdict("有条件接受"); got != ResponsePartiallyAccept {
		t.Fatalf("conditional acceptance = %q", got)
	}
}

func TestChallengeAndResponseUpdateIssueLedger(t *testing.T) {
	m := &Manager{}
	target := Contribution{
		ID:        "pan_s-alpha-r1-1",
		Round:     1,
		AgentID:   "alpha",
		AgentName: "Alpha",
		Content:   "建议先做两轮讨论。",
	}
	challenge := Contribution{
		ID:                    "pan_s-beta-r2-2",
		Round:                 2,
		AgentID:               "beta",
		AgentName:             "Beta",
		DiscussionAct:         DiscussionActChallenge,
		TargetContributionID:  target.ID,
		ChallengedClaim:       "两轮讨论一定优于单席直答",
		EvidenceHandle:        "缺少同任务对照【未核验】",
		ResidualFlipCondition: "出现 8 组同任务对照且多席反例更多",
		ResidualVerification:  "由 Beta 记录单席与四席同题结果",
		Content:               "成本边界缺少实测。",
	}
	session := &Session{ID: "pan_s", Contributions: []Contribution{target}}
	if warnings := m.recordIssueLedger(session, challenge, "质疑"); len(warnings) != 0 {
		t.Fatalf("challenge warnings = %v", warnings)
	}
	if len(session.Issues) != 1 {
		t.Fatalf("issues = %+v, want one challenge issue", session.Issues)
	}
	issue := session.Issues[0]
	if issue.Origin != "challenge" ||
		issue.TargetContributionID != target.ID ||
		issue.ChallengedClaim == "" ||
		issue.Evidence == "" ||
		issue.AgentID != "beta" {
		t.Fatalf("challenge relationship incomplete: %+v", issue)
	}
	if len(issue.ClaimAgentIDs) != 1 || issue.ClaimAgentIDs[0] != "alpha" {
		t.Fatalf("claim attribution = %+v, want alpha only", issue.ClaimAgentIDs)
	}

	session.Contributions = append(session.Contributions, challenge)
	response := Contribution{
		ID:                   "pan_s-alpha-r3-3",
		Round:                3,
		AgentID:              "alpha",
		AgentName:            "Alpha",
		DiscussionAct:        DiscussionActRespond,
		TargetIssueID:        issue.ID,
		TargetContributionID: target.ID,
		ResponseVerdict:      ResponsePartiallyAccept,
		EvidenceHandle:       "两轮结果只在反例发现率上领先，成本仍高",
		Content:              "部分接受，适用条件需要收窄。",
	}
	if warnings := m.recordIssueLedger(session, response, "回应"); len(warnings) != 0 {
		t.Fatalf("response warnings = %v", warnings)
	}
	issue = session.Issues[0]
	if issue.ResponseVerdict != ResponsePartiallyAccept ||
		issue.ResponseEvidence == "" ||
		issue.Status != "open" ||
		!issue.Blocking ||
		issue.LastSeenIn != 2 {
		t.Fatalf("response must update but not withdraw issue: %+v", issue)
	}
	if len(issue.ResponseAgentIDs) != 1 || issue.ResponseAgentIDs[0] != "alpha" {
		t.Fatalf("response attribution = %+v", issue.ResponseAgentIDs)
	}

	session.Contributions = append(session.Contributions, response)
	withdraw := Contribution{
		ID:             "pan_s-beta-r4-4",
		Round:          4,
		AgentID:        "beta",
		AgentName:      "Beta",
		DiscussionAct:  DiscussionActClaim,
		Consensus:      "yes",
		Residual:       "",
		EvidenceHandle: "pan_s-alpha-r3-3 的对照结果",
		Content:        "接受修订后的适用范围。",
	}
	m.recordIssueLedger(session, withdraw, "主张")
	if session.Issues[0].Status != "resolved" || session.Issues[0].Blocking {
		t.Fatalf("only dissenter's explicit withdrawal resolves issue: %+v", session.Issues[0])
	}
	if session.Issues[0].ResolutionEvidence == "" {
		t.Fatalf("withdrawal must preserve evidence: %+v", session.Issues[0])
	}
}

func TestChallengeUnknownTargetWarnsWithoutCreatingIssue(t *testing.T) {
	m := &Manager{}
	session := &Session{ID: "pan_s"}
	challenge := Contribution{
		ID:                   "pan_s-beta-r2-2",
		Round:                2,
		AgentID:              "beta",
		AgentName:            "Beta",
		DiscussionAct:        DiscussionActChallenge,
		TargetContributionID: "pan_s-missing-r1-1",
		ChallengedClaim:      "不存在的命题",
		Content:              "质疑。",
	}
	warnings := m.recordIssueLedger(session, challenge, "质疑")
	if len(warnings) != 1 || !strings.Contains(warnings[0], "不存在") {
		t.Fatalf("warnings = %v, want missing-target warning", warnings)
	}
	if len(session.Issues) != 0 {
		t.Fatalf("unknown target must not create a guessed issue: %+v", session.Issues)
	}
}

func TestActionTargetAndResponseFlowThroughManager(t *testing.T) {
	runner := defaultFakeRunner()
	alphaIDPattern := regexp.MustCompile(`pan_[A-Za-z0-9_-]+-alpha-r1-\d+`)
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") || strings.Contains(spec.Stdin, "轮的主持人") {
			return "summary: 结论\nrecommendation: 先补核验", nil
		}
		if strings.Contains(spec.Stdin, "当前是第 1 轮开场陈述") {
			return "建议采用方案 A。\n动作：主张", nil
		}
		return `部分接受成本质疑，但范围仍需收窄。
动作：回应
目标贡献：` + alphaIDPattern.FindString(spec.Stdin) + `
回应结论：部分接受
证据句柄：pan_s-alpha-r3-3 的对照记录`, nil
	}
	runner.outputs["beta"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "汇总人") || strings.Contains(spec.Stdin, "轮的主持人") {
			return "summary: 结论", nil
		}
		if strings.Contains(spec.Stdin, "当前是第 2 轮补充论证") {
			target := alphaIDPattern.FindString(spec.Stdin)
			if target == "" {
				t.Fatalf("round-2 prompt must expose alpha contribution ID:\n%s", spec.Stdin)
			}
			return `成本边界缺少同任务实测。
动作：质疑
目标贡献：` + target + `
被挑战命题：方案 A 的总成本一定可控
证据句柄：当前只有单席估算【未核验】
翻转条件：拿到 8 组同任务实测
下一步核验：由 Beta 记录单席与四席成本`, nil
		}
		return "补充一个适用范围。\n动作：拓宽", nil
	}

	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "动作链路", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if len(done.Issues) != 1 {
		t.Fatalf("issues = %+v, want one challenge-derived issue", done.Issues)
	}
	issue := done.Issues[0]
	if issue.Origin != "challenge" || issue.ResponseVerdict != ResponsePartiallyAccept {
		t.Fatalf("challenge/response lifecycle = %+v", issue)
	}
	var sawChallenge, sawResponse bool
	for _, contribution := range done.Contributions {
		if contribution.DiscussionAct == DiscussionActChallenge {
			sawChallenge = contribution.TargetContributionID != "" && contribution.ChallengedClaim != ""
		}
		if contribution.DiscussionAct == DiscussionActRespond {
			sawResponse = contribution.TargetContributionID != "" &&
				contribution.ResponseVerdict == ResponsePartiallyAccept
		}
	}
	if !sawChallenge || !sawResponse {
		t.Fatalf("action metadata missing from contributions: challenge=%v response=%v\n%+v",
			sawChallenge, sawResponse, done.Contributions)
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"动作：质疑", "被挑战命题：", "回应结论：部分接受", "来源：定向质疑"} {
		if !strings.Contains(export, want) {
			t.Fatalf("export missing %q:\n%s", want, export)
		}
	}
}
