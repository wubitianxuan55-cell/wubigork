package whisper

import (
	"testing"
)

// 众神殿臂B刀①（docs/gaea-chat-probe-baseline-20261106.md）：Stage 晋级机制
// 修复——中性轮（闲聊/提问）计为正向延续，Stranger 可在 StageWarmupTurns 后
// 晋级 Familiar；冷淡/伤人照旧清零；慢热门槛不放宽。

func TestStageStrangerAdvancesOnNeutralStreak(t *testing.T) {
	l1 := L1State{Stage: StageStranger, Trust: 50}
	event := Event{Type: EvtCasualChat, Intensity: 0.8}
	// StageWarmupTurns 轮内不晋级
	for i := 0; i < StageWarmupTurns; i++ {
		l1 = UpdateRelationship(event, l1)
		if l1.Stage != StageStranger {
			t.Fatalf("第 %d 轮不应提前晋级，got %s", i+1, l1.Stage)
		}
	}
	// 第 StageWarmupTurns+1 轮晋级（条件：> StageWarmupTurns）
	l1 = UpdateRelationship(event, l1)
	if l1.Stage != StageFamiliar {
		t.Fatalf("连续 %d 轮中性正向后应晋级 Familiar，got %s", StageWarmupTurns+1, l1.Stage)
	}
	// 提问同样算正向延续
	l1q := L1State{Stage: StageStranger, Trust: 50}
	qe := Event{Type: EvtQuestion, Intensity: 0.8}
	for i := 0; i <= StageWarmupTurns; i++ {
		l1q = UpdateRelationship(qe, l1q)
	}
	if l1q.Stage != StageFamiliar {
		t.Fatalf("连续提问也应晋级，got %s", l1q.Stage)
	}
}

func TestStageStreakResetsOnNegative(t *testing.T) {
	l1 := L1State{Stage: StageStranger, Trust: 50}
	ce := Event{Type: EvtCasualChat, Intensity: 0.8}
	for i := 0; i < StageWarmupTurns; i++ {
		l1 = UpdateRelationship(ce, l1)
	}
	if l1.ConsecutivePositiveTurns != StageWarmupTurns {
		t.Fatalf("正向连续应累计到 %d，got %d", StageWarmupTurns, l1.ConsecutivePositiveTurns)
	}
	// 冷淡一轮清零
	l1 = UpdateRelationship(Event{Type: EvtCold, Intensity: 0.8}, l1)
	if l1.ConsecutivePositiveTurns != 0 {
		t.Fatalf("冷淡应清零正向连续，got %d", l1.ConsecutivePositiveTurns)
	}
	if l1.Stage != StageStranger {
		t.Fatalf("清零后不得晋级，got %s", l1.Stage)
	}
}
