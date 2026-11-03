package pantheon

import (
	"regexp"
	"strings"
	"testing"
)

// 2026-11-03 用户中途评论（用户拍板）验收基线：
//   - running/waiting_input 可插话；结束后拒绝；
//   - 评论以 Phase=user_note 贡献入档 + user_note 事件；
//   - 分发全文永不截断，标注「用户插话」。

func TestAddUserNoteAppendsContributionAndEvent(t *testing.T) {
	runner := defaultFakeRunner()
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "插话入档", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusRunning)
	updated, err := m.AddUserNote(session.ID, "请重点比较两案的运维成本，其他维度从简。")
	if err != nil {
		t.Fatal(err)
	}
	var note *Contribution
	for i := range updated.Contributions {
		if updated.Contributions[i].Phase == PhaseUserNote {
			note = &updated.Contributions[i]
		}
	}
	if note == nil {
		t.Fatalf("user note contribution missing:\n%+v", updated.Contributions)
	}
	if note.AgentID != "user" || note.Content == "" {
		t.Fatalf("user note fields wrong: %+v", note)
	}
	if len(updated.Issues) != 1 {
		t.Fatalf("user note must enter the shared issue ledger: %+v", updated.Issues)
	}
	issue := updated.Issues[0]
	if issue.Origin != "user" ||
		issue.TargetContributionID != note.ID ||
		issue.Evidence != note.ID ||
		issue.Status != "open" ||
		!issue.Blocking {
		t.Fatalf("user-note issue fields wrong: %+v", issue)
	}
	hasEvent := false
	for _, event := range updated.Events {
		if event.Kind == EventUserNote && event.Text == "请重点比较两案的运维成本，其他维度从简。" {
			hasEvent = true
		}
	}
	if !hasEvent {
		t.Fatal("user_note event missing")
	}
	if _, err := m.AddUserNote(session.ID, "   "); err == nil {
		t.Fatal("blank note must be rejected")
	}
}

func TestUserNoteAcceptResponseResolvesSharedIssue(t *testing.T) {
	runner := defaultFakeRunner()
	userIDPattern := regexp.MustCompile(`pan_[A-Za-z0-9_-]+-user-\d+`)
	for _, seat := range []string{"alpha", "beta"} {
		original := runner.outputs[seat]
		runner.outputs[seat] = func(spec commandSpec) (string, error) {
			text, err := original(spec)
			if err != nil || !strings.Contains(spec.Stdin, "当前是第 2 轮补充论证") {
				return text, err
			}
			userID := userIDPattern.FindString(spec.Stdin)
			if userID == "" {
				return text, nil
			}
			return text + `
动作：回应
目标贡献：` + userID + `
回应结论：接受
证据句柄：` + userID, nil
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "用户插话回应", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusRunning)
	updated, err := m.AddUserNote(session.ID, "后续判断必须先比较运维成本。")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Issues) != 1 || updated.Issues[0].Origin != "user" {
		t.Fatalf("user note issue missing: %+v", updated.Issues)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	found := false
	for _, issue := range done.Issues {
		if issue.Origin != "user" {
			continue
		}
		found = true
		if issue.Status != "resolved" ||
			issue.ResponseVerdict != ResponseAccept ||
			issue.ResolvedIn < 2 ||
			issue.ResolutionEvidence == "" {
			t.Fatalf("accepted user note must resolve with evidence: %+v", issue)
		}
	}
	if !found {
		t.Fatalf("user issue missing after completion: %+v", done.Issues)
	}
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "回应未命中目标贡献") {
			t.Fatalf("additional responses to a resolved user note must stay attached, not miss: %s", warning)
		}
	}
}

func TestAddUserNoteRejectsAfterCompletion(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "结束后拒绝", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if _, err := m.AddUserNote(done.ID, "补一句"); err == nil {
		t.Fatal("note after completion must be rejected")
	}
}

func TestUserNoteDispatchedInFullNeverTruncated(t *testing.T) {
	// 超长插话全文分发：立场卡/归档分层不适用于用户插话。
	item := Contribution{
		ID: "pan_s-user-42", Phase: PhaseUserNote, Round: 2,
		AgentID: "user", AgentName: "用户",
		Content: longStatement("插话") + "插话独有尾部标记USNTAIL。",
	}
	got := formatContributionBody(item, 3, nil)
	if !strings.Contains(got, "插话独有尾部标记USNTAIL") {
		t.Fatalf("user note must dispatch in full regardless of length:\n%s", got)
	}
	if !strings.Contains(got, "（用户插话 · 第 2 轮）") {
		t.Fatalf("user note must carry the interjection label:\n%s", got)
	}
}

func TestDiscussionPromptCarriesUserNoteGuidance(t *testing.T) {
	note := Contribution{
		ID: "pan_s-user-7", Phase: PhaseUserNote, Round: 1,
		AgentID: "user", AgentName: "用户", Content: "请聚焦第二方案的落地风险。",
	}
	prompt := discussionPrompt(
		StartRequest{Topic: "插话指引"}, AgentStatus{ID: "beta", Name: "BETA"},
		[]Contribution{note}, 2, 1, 2, nil, nil,
	)
	if !strings.Contains(prompt, "请聚焦第二方案的落地风险。") {
		t.Fatalf("prompt must carry the user note verbatim:\n%s", prompt)
	}
	if !strings.Contains(prompt, "用户插话") || !strings.Contains(prompt, "优先级最高") {
		t.Fatalf("prompt must instruct seats to answer interjections first:\n%s", prompt)
	}
}
