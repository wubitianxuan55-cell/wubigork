package pantheon

import (
	"strings"
	"testing"
	"time"
)

// waitForRunningSeat 等到有一席真的开跑，返回席位 ID 与它这次的 attempt。
func waitForRunningSeat(t *testing.T, m *Manager, sessionID string) (string, CallRef) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		session, err := m.Get(sessionID)
		if err != nil {
			t.Fatal(err)
		}
		for _, participant := range session.Participants {
			if participant.Status == SeatRunning && participant.CallRef.AttemptID != "" {
				return participant.AgentID, participant.CallRef
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("没有席位开跑")
	return "", CallRef{}
}

// setRunMode 改一个席位的运行方式声明，模拟适配器被换掉只读机制。
func setRunMode(m *Manager, agentID, runMode string) {
	m.mu.Lock()
	adapter := m.adapters[agentID]
	adapter.runMode = runMode
	m.adapters[agentID] = adapter
	m.mu.Unlock()
}

// TestRetryRefusesUndeclaredRunMode：重试必须自己声明运行方式。声明不出来就
// 拒绝，并且不留下"正在重试"的记录——不能拿上一次的口径悄悄再跑一遍。
func TestRetryRefusesUndeclaredRunMode(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{Topic: "重试必须重声明运行方式", AgentIDs: []string{"alpha", "beta"}, Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	seat, _ := waitForRunningSeat(t, m, session.ID)
	setRunMode(m, seat, "")

	if err := m.RetryAgent(session.ID, seat); err == nil || !strings.Contains(err.Error(), "运行方式") {
		t.Fatalf("undeclared run mode retry error = %v, want a run-mode refusal", err)
	}
	got, err := m.Get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range got.Events {
		if event.Kind == EventAgentRetry {
			t.Fatalf("refused retry must not be recorded: %+v", event)
		}
	}
	for _, participant := range got.Participants {
		if participant.AgentID == seat && participant.Status != SeatRunning {
			t.Fatalf("refused retry changed seat status to %q", participant.Status)
		}
	}

	// 补上声明之后，同一次重试必须能走通。
	setRunMode(m, seat, "内存桩（只读）")
	if err := m.RetryAgent(session.ID, seat); err != nil {
		t.Fatalf("retry after declaring run mode: %v", err)
	}
}

// TestRetryDeclaresPlacementInEvent：重试事件里要写清这一次跑在什么方式、什么
// 工作空间下，记录本身能回答"它拿什么跑的第二遍"。
func TestRetryDeclaresPlacementInEvent(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	workspace := t.TempDir()
	session, err := m.Start(StartRequest{
		Topic: "重试写明运行方式", AgentIDs: []string{"alpha", "beta"}, Rounds: 1, Workspace: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	seat, _ := waitForRunningSeat(t, m, session.ID)
	if err := m.RetryAgent(session.ID, seat); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for _, event := range got.Events {
		if event.Kind == EventAgentRetry && event.AgentID == seat {
			text = event.Text
		}
	}
	if text == "" {
		t.Fatal("retry event missing")
	}
	if !strings.Contains(text, "运行方式：内存桩（只读）") {
		t.Fatalf("retry event does not declare run mode: %q", text)
	}
	if !strings.Contains(text, "工作空间：") {
		t.Fatalf("retry event does not declare workspace: %q", text)
	}
}

// TestCancelKeepsAttemptEvidence：取消只结束这一次调用，不收回已经发生的证据。
func TestCancelKeepsAttemptEvidence(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{Topic: "取消保留证据", AgentIDs: []string{"alpha", "beta"}, Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	seat, call := waitForRunningSeat(t, m, session.ID)
	if err := m.Cancel(session.ID); err != nil {
		t.Fatal(err)
	}
	got := waitForStatus(t, m, session.ID, StatusCancelled)

	for index, event := range got.Events {
		if event.Seq != int64(index+1) {
			t.Fatalf("event %d seq = %d, want %d (取消不应重排或删除事件)", index, event.Seq, index+1)
		}
	}
	var sawAttempt, sawStart bool
	for _, event := range got.Events {
		if event.AttemptID == call.AttemptID {
			sawAttempt = true
		}
		if event.Kind == EventAgentStart && event.AgentID == seat && event.AttemptID == call.AttemptID {
			sawStart = true
		}
	}
	if !sawAttempt || !sawStart {
		t.Fatalf("取消后丢了 attempt %s 的证据: attempt=%v start=%v", call.AttemptID, sawAttempt, sawStart)
	}
}

// TestRetryKeepsEarlierAttemptEvents：重试发新 attempt，但旧 attempt 的事件不被
// 覆盖、不被重排。
func TestRetryKeepsEarlierAttemptEvents(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{Topic: "重试保留旧 attempt", AgentIDs: []string{"alpha", "beta"}, Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	seat, first := waitForRunningSeat(t, m, session.ID)

	var startSeq int64
	if before, err := m.Get(session.ID); err == nil {
		for _, event := range before.Events {
			if event.Kind == EventAgentStart && event.AgentID == seat && event.AttemptID == first.AttemptID {
				startSeq = event.Seq
			}
		}
	}
	if startSeq == 0 {
		t.Fatal("first attempt start event missing")
	}
	if err := m.RetryAgent(session.ID, seat); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var second CallRef
	for time.Now().Before(deadline) {
		got, err := m.Get(session.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, participant := range got.Participants {
			if participant.AgentID == seat && participant.CallRef.Attempt > first.Attempt {
				second = participant.CallRef
			}
		}
		if second.AttemptID != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if second.AttemptID == "" {
		t.Fatal("retry never produced a second attempt")
	}
	if second.CallID != first.CallID || second.RetryOfAttempt != first.AttemptID {
		t.Fatalf("retry lineage wrong: first=%+v second=%+v", first, second)
	}

	got, err := m.Get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range got.Events {
		if event.AttemptID == first.AttemptID && event.Kind == EventAgentStart {
			if event.Seq != startSeq {
				t.Fatalf("earlier attempt start seq moved: %d -> %d", startSeq, event.Seq)
			}
			return
		}
	}
	t.Fatalf("older attempt events disappeared after retry: %+v", got.Events)
}
