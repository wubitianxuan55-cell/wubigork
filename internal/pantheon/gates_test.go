package pantheon

import (
	"strings"
	"testing"
)

func gateSession(status string, participants ...Participant) *Session {
	return &Session{
		ID: "pan_gates", Title: "决策门", Status: status, Phase: PhaseDiscussion,
		CurrentRound: 1,
		Request:      StartRequest{Topic: "决策门", Rounds: 3, AgentIDs: []string{"alpha", "beta"}},
		Participants: participants,
	}
}

// M14：没被处理的门必须一直挡路，第二道门也不许叠着开。
func TestOpenGateBlocksAndRefusesSecondGate(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	session := gateSession(StatusRunning, Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatWaiting})
	if err := m.store.save(session); err != nil {
		t.Fatal(err)
	}

	got, err := m.OpenGate(session.ID, "")
	if err != nil {
		t.Fatalf("开门失败：%v", err)
	}
	if got.Status != StatusWaitingInput {
		t.Fatalf("开门后状态 = %q，应为 waiting_input", got.Status)
	}
	if !hasOpenGateLocked(got) {
		t.Fatal("门没有记在事件里：重启后会凭空消失")
	}
	if !m.paused(got) {
		t.Fatal("门开着却没判为暂停")
	}
	if _, err := m.OpenGate(session.ID, "再来一道"); err == nil {
		t.Fatal("第二道门应被拒绝")
	} else if !strings.Contains(err.Error(), "决策门") {
		t.Fatalf("拒绝理由不可读：%v", err)
	}
}

// 回答是 exactly-once：同一个问题答第二次必须被拒，门的记录也不能被悄悄改写。
func TestAnswerSeatIsExactlyOnce(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	session := gateSession(StatusRunning, Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatRunning})
	if err := m.store.save(session); err != nil {
		t.Fatal(err)
	}

	asked, err := m.AskSeat(session.ID, "alpha", "要不要覆盖旧结论？", []string{"覆盖", "保留"})
	if err != nil {
		t.Fatalf("提问失败：%v", err)
	}
	if asked.Status != StatusWaitingInput {
		t.Fatalf("提问后整场状态 = %q", asked.Status)
	}
	if got := asked.Participants[0].Status; got != SeatNeedsInput {
		t.Fatalf("席位状态 = %q，应为 needs_input", got)
	}
	pending := pendingQuestionsLocked(asked)
	if len(pending) != 1 || pending[0].QuestionID == "" {
		t.Fatalf("待答问题 = %+v", pending)
	}
	questionID := pending[0].QuestionID

	answered, err := m.AnswerSeat(session.ID, questionID, "保留")
	if err != nil {
		t.Fatalf("回答失败：%v", err)
	}
	if len(pendingQuestionsLocked(answered)) != 0 {
		t.Fatal("回答后仍然有待答问题")
	}
	if got := answered.Participants[0].Status; got != SeatWaiting {
		t.Fatalf("答完后席位 = %q，应回到 waiting 等一次显式续跑", got)
	}
	if _, err := m.AnswerSeat(session.ID, questionID, "再答一次"); err == nil {
		t.Fatal("重复回答必须被拒")
	}
	if _, err := m.AnswerSeat(session.ID, "pan_gates-q999", "无中生有"); err == nil {
		t.Fatal("不存在的提问必须被拒")
	}
}

// 回答只解除阻塞，不偷偷续跑：整场仍是 waiting_input，等用户点「继续议事」。
func TestAnswerSeatDoesNotAutoResume(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	session := gateSession(StatusRunning, Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatRunning})
	if err := m.store.save(session); err != nil {
		t.Fatal(err)
	}
	asked, err := m.AskSeat(session.ID, "alpha", "先做哪一半？", nil)
	if err != nil {
		t.Fatal(err)
	}
	questionID := pendingQuestionsLocked(asked)[0].QuestionID
	after, err := m.AnswerSeat(session.ID, questionID, "先做减法")
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != StatusWaitingInput {
		t.Fatalf("回答后整场该继续停着，实际 %q", after.Status)
	}
}

// 有未答问题或没解的门时，续跑必须被拒——否则 run 会立刻又停回去，看起来
// 像"点继续没反应"。
func TestResumeRefusesPendingQuestionsAndOpenGate(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()

	withQuestion := gateSession(StatusWaitingInput, Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatNeedsInput})
	withQuestion.Contributions = []Contribution{{ID: "c1", AgentID: "alpha", Content: "已有观点"}}
	withQuestion.Events = []Event{{
		Seq: 1, Kind: EventQuestion, AgentID: "alpha",
		Question: &QuestionPayload{QuestionID: "pan_gates-q1", AskedBy: "alpha", At: 1},
	}}
	if err := m.store.save(withQuestion); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Resume(withQuestion.ID); err == nil || !strings.Contains(err.Error(), "没有回答") {
		t.Fatalf("未答问题时续跑应被拒，实际 err = %v", err)
	}

	withGate := gateSession(StatusWaitingInput, Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatWaiting})
	withGate.Contributions = []Contribution{{ID: "c2", AgentID: "alpha", Content: "已有观点"}}
	withGate.Events = []Event{{Seq: 1, Kind: EventGate, Gate: &GatePayload{Action: GateActionOpen}}}
	if err := m.store.save(withGate); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Resume(withGate.ID); err == nil || !strings.Contains(err.Error(), "决策门") {
		t.Fatalf("门没解时续跑应被拒，实际 err = %v", err)
	}
}

// M14 的跨重启部分：waiting_input 不是"被中断"，重启原样保留阻塞与未答的问题。
func TestRecoverInterruptedKeepsWaitingInputSession(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	session := gateSession(StatusWaitingInput, Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatNeedsInput})
	session.Events = []Event{{
		Seq: 1, Kind: EventQuestion, AgentID: "alpha",
		Question: &QuestionPayload{QuestionID: "pan_gates-q1", AskedBy: "alpha", At: 1},
	}}
	if err := m.store.save(session); err != nil {
		t.Fatal(err)
	}

	if _, err := m.RecoverInterrupted(); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusWaitingInput {
		t.Fatalf("重启把等人的议事改成了 %q", got.Status)
	}
	if got.Participants[0].Status != SeatNeedsInput {
		t.Fatalf("重启改了席位状态：%q", got.Participants[0].Status)
	}
	if len(pendingQuestionsLocked(got)) != 1 {
		t.Fatal("重启弄丢了未答的问题")
	}
}

// 结案缺口要具名：没交出结论的席位留在缺口里，不能算成弃权。
func TestSeatGapsNamesEveryUnfinishedSeat(t *testing.T) {
	session := gateSession(StatusCompleted,
		Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatFinished},
		Participant{AgentID: "beta", Name: "BETA", Status: SeatFinishedUnverified},
		Participant{AgentID: "gamma", Name: "GAMMA", Status: SeatSkipped},
		Participant{AgentID: "delta", Name: "DELTA", Status: SeatFailed},
	)
	gaps := seatGaps(session)
	if len(gaps) != 2 {
		t.Fatalf("缺口 = %v，应为两席", gaps)
	}
	joined := strings.Join(gaps, "、")
	if !strings.Contains(joined, "BETA（未验证返回）") || !strings.Contains(joined, "GAMMA（已跳过）") {
		t.Fatalf("缺口名字/原因不可读：%s", joined)
	}
}

// 同样的小结、不同的缺口，签注指纹必须不同；否则转发件会声称一场比实际更整齐的议事。
func TestSignoffFingerprintDistinguishesGaps(t *testing.T) {
	verdict := &Verdict{Recommendation: "按方案 A 推进"}
	full := signoffFingerprint(verdict, nil)
	gapped := signoffFingerprint(verdict, []string{"BETA（未验证返回）"})
	if full != verdictFingerprint(verdict) {
		t.Fatal("无缺口时指纹口径应保持原样")
	}
	if full == gapped || gapped == "" {
		t.Fatalf("缺口没进指纹：full=%s gapped=%s", full, gapped)
	}
}

// 门只在 running 时能开：终态会话不该被"开一道门"变成等人。
func TestOpenGateRefusesNonRunningSession(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	session := gateSession(StatusCompleted)
	if err := m.store.save(session); err != nil {
		t.Fatal(err)
	}
	if _, err := m.OpenGate(session.ID, "收尾时停一下"); err == nil {
		t.Fatal("已完成的议事不该能开门")
	}
	if m.openGateInSession(session, "收尾时停一下") {
		t.Fatal("内部开门同样要认 running")
	}
}
