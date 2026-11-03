package pantheon

import (
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
		[]Contribution{note}, 2, 1, 2, nil,
	)
	if !strings.Contains(prompt, "请聚焦第二方案的落地风险。") {
		t.Fatalf("prompt must carry the user note verbatim:\n%s", prompt)
	}
	if !strings.Contains(prompt, "用户插话") || !strings.Contains(prompt, "优先级最高") {
		t.Fatalf("prompt must instruct seats to answer interjections first:\n%s", prompt)
	}
}
