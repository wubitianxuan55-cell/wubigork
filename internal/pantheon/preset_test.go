package pantheon

import (
	"strings"
	"testing"
)

func TestNormalizePreset(t *testing.T) {
	if got, err := NormalizePreset(""); err != nil || got != PresetRoundtable {
		t.Fatalf("blank preset = %q, %v", got, err)
	}
	if got, err := NormalizePreset("targeted_challenge"); err != nil || got != PresetTargetedChallenge {
		t.Fatalf("targeted preset = %q, %v", got, err)
	}
	if got, err := NormalizePreset("red_team"); err != nil || got != PresetRedTeam {
		t.Fatalf("red-team preset = %q, %v", got, err)
	}
	if _, err := NormalizePreset("unknown"); err == nil {
		t.Fatal("unknown preset must be rejected")
	}
}

func TestTargetedChallengeForcesIndependentOpening(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic:           "定向质询路径",
		Preset:          PresetTargetedChallenge,
		Protocol:        ProtocolCouncil,
		AgentIDs:        []string{"alpha", "beta"},
		ChairmanAgentID: "alpha",
		Rounds:          1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Request.Rounds < 2 {
		t.Fatalf("targeted challenge must reserve a challenge/response round: %+v", session.Request)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Request.Preset != PresetTargetedChallenge || done.Request.Protocol != ProtocolIndependent {
		t.Fatalf("targeted path normalized to preset=%q protocol=%q", done.Request.Preset, done.Request.Protocol)
	}
}

func TestTargetedChallengeRequiresTwoSeats(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	_, err := m.Start(StartRequest{
		Topic:           "单席不能定向质询",
		Preset:          PresetTargetedChallenge,
		AgentIDs:        []string{"alpha"},
		ChairmanAgentID: "alpha",
		Rounds:          1,
	})
	if err == nil || !strings.Contains(err.Error(), "至少需要选择两个") {
		t.Fatalf("targeted one-seat error = %v", err)
	}
}

func TestTargetedChallengePromptRequiresLocatedResponse(t *testing.T) {
	prompt := discussionPrompt(
		StartRequest{Topic: "定向质询", Preset: PresetTargetedChallenge},
		AgentStatus{ID: "beta", Name: "BETA"},
		nil, 2, 1, 2, nil, nil,
	)
	for _, want := range []string{"【定向质询模式】", "被挑战命题", "回应结论"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("targeted prompt missing %q:\n%s", want, prompt)
		}
	}
}
