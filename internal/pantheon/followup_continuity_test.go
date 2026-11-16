package pantheon

import (
	"strings"
	"testing"
	"time"
)

// 续议会话承接（2026-11-16）：续议（基于小结续议，ParentSessionID 非空）的
// 同席席位继承父场登记的引擎侧会话 ID——首调即 resume 父场会话，防丢上下文。
// 降级链（会话过期→错误+零输出→丢弃→回落新建）由既有机制承担，此处锁承接
// 主链与说明注入。

func TestFollowUpInheritsEngineSessions(t *testing.T) {
	oldBudget, oldStall := seatToolBudget, seatStallLimit
	seatToolBudget, seatStallLimit = 100, time.Minute
	t.Cleanup(func() { seatToolBudget, seatStallLimit = oldBudget, oldStall })

	runner := &dshSeatFake{}
	m := newTestManager(t, runner, fakeAdapter("alpha"), newDSHFakeAdapter(runner))
	defer m.Close()

	parent, err := m.Start(StartRequest{
		Topic: "上一场", AgentIDs: []string{"alpha", "dsh"}, ChairmanAgentID: "alpha",
		Rounds: 1, Protocol: ProtocolIndependent,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitSessionSettled(t, m, parent.ID)
	if done.Status != StatusCompleted {
		t.Fatalf("parent status = %s, error = %s", done.Status, done.Error)
	}
	if got := participantEngineSession(done, "dsh"); got != "session-first" {
		t.Fatalf("parent dsh engine session = %q, want session-first", got)
	}
	if got := participantEngineSession(done, "alpha"); got != "" {
		t.Fatalf("parent alpha (no buildResume) must not record engine session, got %q", got)
	}

	follow, err := m.Start(StartRequest{
		Topic: "续议一场", AgentIDs: []string{"alpha", "dsh"}, ChairmanAgentID: "alpha",
		Rounds: 1, Protocol: ProtocolIndependent, ParentSessionID: parent.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	doneFollow := waitSessionSettled(t, m, follow.ID)
	if doneFollow.Status != StatusCompleted {
		t.Fatalf("follow status = %s, error = %s", doneFollow.Status, doneFollow.Error)
	}
	// 承接主链：续议首调必须带 --session-id session-first（父场 ID）；调完
	// 后新上报的会话 ID 落回本会话参与者（session-resumed）。
	resumed := false
	continuityNote := false
	runner.mu.Lock()
	for _, spec := range runner.specs {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, "--session-id session-first") {
			resumed = true
		}
		if strings.Contains(spec.Stdin, "开场陈述") && strings.Contains(spec.Stdin, "【续议承接】") {
			continuityNote = true
		}
	}
	runner.mu.Unlock()
	if !resumed {
		t.Fatal("follow-up first seat call must resume the inherited parent engine session")
	}
	if !continuityNote {
		t.Fatal("follow-up opening prompt must carry the continuity brief section")
	}
	if got := participantEngineSession(doneFollow, "dsh"); got != "session-resumed" {
		t.Fatalf("follow-up dsh engine session = %q, want session-resumed (new run overwrites)", got)
	}
	if got := participantEngineSession(doneFollow, "alpha"); got != "" {
		t.Fatalf("follow-up alpha must stay without engine session, got %q", got)
	}
}

func TestParentEngineSessionFor(t *testing.T) {
	parent := &Session{Participants: []Participant{
		{AgentID: "dsh", EngineSessionID: "  sess-9  "},
		{AgentID: "alpha"},
	}}
	if got := parentEngineSessionFor(parent, "dsh"); got != "sess-9" {
		t.Fatalf("inherited id = %q, want trimmed sess-9", got)
	}
	if got := parentEngineSessionFor(parent, "zcode"); got != "" {
		t.Fatalf("unlisted seat must inherit nothing, got %q", got)
	}
	if got := parentEngineSessionFor(nil, "dsh"); got != "" {
		t.Fatalf("nil parent must be safe, got %q", got)
	}
}

func TestFollowUpBriefCarriesContinuityNote(t *testing.T) {
	with := buildBrief(StartRequest{Topic: "续议", ParentSessionID: "pan_parent"}, nil)
	for _, want := range []string{"【续议承接】", "续议", "以上方【议题】为准"} {
		if !strings.Contains(with, want) {
			t.Fatalf("continuity brief misses %q:\n%s", want, with)
		}
	}
	plain := buildBrief(StartRequest{Topic: "新一场"}, nil)
	if strings.Contains(plain, "【续议承接】") {
		t.Fatalf("plain session must not carry continuity note:\n%s", plain)
	}
}

func participantEngineSession(session *Session, agentID string) string {
	for _, p := range session.Participants {
		if p.AgentID == agentID {
			return strings.TrimSpace(p.EngineSessionID)
		}
	}
	return ""
}
