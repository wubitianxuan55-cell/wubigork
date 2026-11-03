package whisper

import "testing"

func TestPreTurnSnapshotRestoresVisibleAndHiddenTurnState(t *testing.T) {
	o := NewOrchestrator("snapshot", PersonalityPresets[0])
	o.State.Counters.TotalTurns = 3
	o.State.Emotion.Aff = 42
	o.recentEventTypes = []string{"neutral", "praise"}
	o.consecutiveMeaningfulCount = 2
	o.consecutiveVulnerableCount = 1
	o.adultStateStr = "NORMAL"
	o.adultBudget = 0.75
	o.adultLockTurns = 2
	o.adultConsecutiveVulnerable = 1
	o.adultLastRejectedTurn = 2
	o.rhythm = RhythmCounters{Chatter: 4, Monologue: 1}
	o.memoryGraphRestored = true

	snap := o.SnapshotPreTurn()
	o.State.Counters.TotalTurns = 9
	o.State.Emotion.Aff = -10
	o.recentEventTypes = append(o.recentEventTypes, "hurtful")
	o.consecutiveMeaningfulCount = 0
	o.consecutiveVulnerableCount = 0
	o.adultStateStr = "ADULT"
	o.adultBudget = 0
	o.adultLockTurns = 9
	o.adultConsecutiveVulnerable = 3
	o.adultLastRejectedTurn = 9
	o.rhythm = RhythmCounters{Chatter: 0, Monologue: 8}
	o.memoryGraphRestored = false

	o.RestorePreTurn(snap)
	if o.State.Counters.TotalTurns != 3 || o.State.Emotion.Aff != 42 {
		t.Fatalf("可见状态未恢复: turns=%d aff=%v", o.State.Counters.TotalTurns, o.State.Emotion.Aff)
	}
	if len(o.recentEventTypes) != 2 || o.consecutiveMeaningfulCount != 2 || o.consecutiveVulnerableCount != 1 {
		t.Fatalf("涌现状态未恢复: %#v %d/%d", o.recentEventTypes, o.consecutiveMeaningfulCount, o.consecutiveVulnerableCount)
	}
	if o.adultStateStr != "NORMAL" || o.adultBudget != 0.75 || o.adultLockTurns != 2 ||
		o.adultConsecutiveVulnerable != 1 || o.adultLastRejectedTurn != 2 {
		t.Fatalf("成人模式状态未恢复")
	}
	if o.rhythm != (RhythmCounters{Chatter: 4, Monologue: 1}) || !o.memoryGraphRestored {
		t.Fatalf("节奏/图恢复标记未恢复: %+v %v", o.rhythm, o.memoryGraphRestored)
	}
}
