package pantheon

import (
	"strings"
	"testing"
)

func TestDIDSeenIsReconstructedFromEvents(t *testing.T) {
	seen := didSeenFromEvents([]Event{
		{Kind: EventDIDUsed, DIDKeys: []string{"issue-1|verify-a", "issue-2|verify-b"}},
		{Kind: EventDIDStop},
	})
	if !seen["issue-1|verify-a"] || !seen["issue-2|verify-b"] {
		t.Fatalf("DID event reconstruction = %+v", seen)
	}
	if actionableVerification("继续讨论") || actionableVerification("") {
		t.Fatal("non-actionable verification must not count as DID")
	}
}

func TestDIDStopsUntilConsensusWhenNoVerificationExists(t *testing.T) {
	runner := defaultFakeRunner()
	for id, original := range runner.outputs {
		output := original
		runner.outputs[id] = func(spec commandSpec) (string, error) {
			text, err := output(spec)
			if err == nil && strings.Contains(spec.Stdin, "全员一致检查") {
				text += "\n共识状态：仍有分歧\n剩余分歧：成本边界未收口"
			}
			return text, err
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic:           "没有核验动作",
		AgentIDs:        []string{"alpha", "beta"},
		ChairmanAgentID: "alpha",
		UntilConsensus:  true,
		MaxRounds:       6,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.CurrentRound != 2 {
		t.Fatalf("without DID the council should stop after round 2, got %d", done.CurrentRound)
	}
	stopped := false
	for _, event := range done.Events {
		if event.Kind == EventDIDStop {
			stopped = true
		}
	}
	if !stopped {
		t.Fatal("did_stop event missing")
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(export, "没有新的可执行核验动作") {
		t.Fatalf("export must carry DID stop reason:\n%s", export)
	}
}

func TestDIDAllowsOneNewVerificationThenStopsOnRepeat(t *testing.T) {
	runner := defaultFakeRunner()
	for id, original := range runner.outputs {
		seat := id
		output := original
		runner.outputs[seat] = func(spec commandSpec) (string, error) {
			text, err := output(spec)
			if err != nil || !strings.Contains(spec.Stdin, "全员一致检查") {
				return text, err
			}
			return text + `
共识状态：仍有分歧
剩余分歧：成本边界未收口
分歧依据：当前只有单席估算【未核验】
翻转条件：完成同任务对照
下一步核验：由 alpha 在独立工作区记录 8 组同任务成本`, nil
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic:           "DID 重复",
		AgentIDs:        []string{"alpha", "beta"},
		ChairmanAgentID: "alpha",
		UntilConsensus:  true,
		MaxRounds:       6,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.CurrentRound != 3 {
		t.Fatalf("one new DID should allow one more round, then stop at 3; got %d", done.CurrentRound)
	}
	var used, stopped int
	for _, event := range done.Events {
		switch event.Kind {
		case EventDIDUsed:
			used++
		case EventDIDStop:
			stopped++
		}
	}
	if used != 1 || stopped != 1 {
		t.Fatalf("DID events = used %d / stop %d, want 1/1", used, stopped)
	}
}
