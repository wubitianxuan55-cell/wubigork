package pantheon

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type promptScriptRunner struct {
	mu       sync.Mutex
	calls    []commandSpec
	decision string
	detail   string
	sent     bool
	question bool
}

func (r *promptScriptRunner) Run(_ context.Context, spec commandSpec) (string, error) {
	r.mu.Lock()
	r.calls = append(r.calls, spec)
	decision, detail, sent, question := r.decision, r.detail, r.sent, r.question
	if decision != "" && strings.Contains(spec.Stdin, "轮间小结") {
		r.sent = true
	}
	r.mu.Unlock()

	switch {
	case strings.Contains(spec.Stdin, "轮间小结"):
		if decision != "" && !sent {
			return "小结：先停一下。\n" + chairEnvelopeOpen + `{"decision":"` + decision + `","detail":"` + detail + `"}` + chairEnvelopeClose, nil
		}
		return "小结：继续。", nil
	case strings.Contains(spec.Stdin, "你的职责是协助整理"):
		return fakeOutputFor("alpha", spec.Stdin), nil
	case strings.Contains(spec.Stdin, "开场陈述"):
		if question && spec.Executable == "beta" {
			return "我的初步判断：先做只读验证。\nNEEDS INPUT: 要确认缓存口径吗？\nDefault: 先确认", nil
		}
		return fakeOutputFor(spec.Executable, spec.Stdin), nil
	case strings.Contains(spec.Stdin, "补充论证"):
		return "补充论证：承接前文，本轮不新增分歧。\n剩余分歧：无\n置信度：中", nil
	default:
		return fakeOutputFor(spec.Executable, spec.Stdin), nil
	}
}

func (r *promptScriptRunner) prompts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.calls))
	for _, call := range r.calls {
		out = append(out, call.Stdin)
	}
	return out
}

func TestChairBlockedOpensGate(t *testing.T) {
	runner := &promptScriptRunner{decision: "blocked", detail: "缺关键证据"}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "主席阻塞", Topic: "验证 E3", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 2, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusWaitingInput)
	if !hasOpenGateLocked(done) {
		t.Fatalf("blocked 必须开一道门，events=%+v", done.Events)
	}
	var decision *ChairDecision
	for _, contribution := range done.Contributions {
		if contribution.Phase == PhaseModeration {
			decision = contribution.ChairDecision
		}
	}
	if decision == nil || decision.Status != chairDecisionBlocked || decision.Detail == "" {
		t.Fatalf("moderation decision = %+v, want blocked with detail", decision)
	}
}

func TestChairNeedsInputAsksHostAndAnswerResumes(t *testing.T) {
	runner := &promptScriptRunner{decision: "needs-input", detail: "需要你选 A/B"}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "主席问人", Topic: "验证 E3 needs-input", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 2, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	waiting := waitForStatus(t, m, session.ID, StatusWaitingInput)
	var question *QuestionPayload
	for _, event := range waiting.Events {
		if event.Kind == EventQuestion && event.Question != nil {
			question = event.Question
		}
	}
	if question == nil || question.Origin != QuestionOriginChair || question.AskedBy != "" {
		t.Fatalf("host question = %+v, want origin=chair askedBy empty", question)
	}
	if _, err := m.AnswerSeat(session.ID, question.QuestionID, "选 A"); err != nil {
		t.Fatal(err)
	}
	var resumed bool
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if _, err := m.Resume(session.ID); err == nil {
			resumed = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !resumed {
		t.Fatal("Resume never succeeded after host answer")
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	var answer *Contribution
	for index := range done.Contributions {
		if done.Contributions[index].Phase == PhaseUserAnswer {
			answer = &done.Contributions[index]
		}
	}
	if answer == nil || answer.QuestionID != question.QuestionID || answer.Content != "选 A" {
		t.Fatalf("host answer contribution = %+v", answer)
	}
}

func TestSeatQuestionPausesAtRoundEndAndAnswerReachesNextRound(t *testing.T) {
	runner := &promptScriptRunner{question: true}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "席位提问", Topic: "验证 E4", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 2, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	waiting := waitForStatus(t, m, session.ID, StatusWaitingInput)
	var question *QuestionPayload
	for _, event := range waiting.Events {
		if event.Kind == EventQuestion && event.Question != nil {
			question = event.Question
		}
	}
	if question == nil || question.Origin != QuestionOriginSeat || question.AskedBy != "beta" {
		t.Fatalf("seat question = %+v", question)
	}
	if question.Default != "先确认" {
		t.Fatalf("default = %q, want 先确认", question.Default)
	}
	if _, err := m.AnswerSeat(session.ID, question.QuestionID, "确认缓存口径"); err != nil {
		t.Fatal(err)
	}
	// waitForStatus 返回时 run 的 defer 可能还没删掉 runs 注册；Resume 自带
	// 二次复查，短暂重试即可。
	var resumed bool
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if _, err := m.Resume(session.ID); err == nil {
			resumed = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !resumed {
		t.Fatal("Resume never succeeded after answer")
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	var answer *Contribution
	for index := range done.Contributions {
		if done.Contributions[index].Phase == PhaseUserAnswer {
			answer = &done.Contributions[index]
		}
	}
	if answer == nil || answer.QuestionID != question.QuestionID || answer.Content != "确认缓存口径" {
		t.Fatalf("user answer contribution = %+v", answer)
	}
	found := false
	for _, prompt := range runner.prompts() {
		if strings.Contains(prompt, "用户回答") && strings.Contains(prompt, "确认缓存口径") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("next-round prompt must carry 【用户回答】 and the answer text")
	}
}
