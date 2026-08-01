package whisper

import (
	"testing"
)

func newTestOrch() *Orchestrator {
	return NewOrchestrator("test-session", PersonalityPresets[0])
}

// ─── applyReunionBoost ───────────────────────────────────────

func TestApplyReunionBoost_ShortGap(t *testing.T) {
	o := newTestOrch()
	e := EmotionState{Aff: 0, Sec: 0, Aro: 0}
	o.applyReunionBoost(&e, 10) // < 24h
	if e.Aff <= 0 || e.Sec <= 0 {
		t.Errorf("短间隔应提升 Aff/Sec: %+v", e)
	}
	if e.Aro != 0 {
		t.Errorf("短间隔不应提升 Aro: %+v", e)
	}
}

func TestApplyReunionBoost_MediumGap(t *testing.T) {
	o := newTestOrch()
	e := EmotionState{Aff: 0, Sec: 0, Aro: 0}
	o.applyReunionBoost(&e, 48) // 24-72h
	if e.Aff <= 0 {
		t.Errorf("中间隔应提升 Aff: %+v", e)
	}
	if e.Sec != 0 {
		t.Errorf("中间隔不应提升 Sec: %+v", e)
	}
}

func TestApplyReunionBoost_LongGap(t *testing.T) {
	o := newTestOrch()
	e := EmotionState{Aff: 0, Sec: 0, Aro: 0}
	o.applyReunionBoost(&e, 100) // > 72h
	if e.Aro <= 0 {
		t.Errorf("长间隔应提升 Aro: %+v", e)
	}
	if e.Aff != 0 || e.Sec != 0 {
		t.Errorf("长间隔不应提升 Aff/Sec: %+v", e)
	}
}

func TestApplyReunionBoost_Clamps(t *testing.T) {
	o := newTestOrch()
	e := EmotionState{Aff: 99, Sec: 99, Aro: -90}
	o.applyReunionBoost(&e, 10)
	if e.Aff > 100 || e.Sec > 100 {
		t.Errorf("Aff/Sec 应 clamp 到 100: %+v", e)
	}
}

// ─── applyPeriodicDrift ──────────────────────────────────────

func TestApplyPeriodicDrift_NoBaselineNoop(t *testing.T) {
	o := newTestOrch()
	st := FullState{}
	o.applyPeriodicDrift(&st, 20)
	// 无 baseline 不应崩溃，状态不变
	if st.Personality.T != 0 {
		t.Error("无 baseline 不应漂移")
	}
}

func TestApplyPeriodicDrift_Turn20Drifts(t *testing.T) {
	o := newTestOrch()
	st := FullState{
		PersonalityBaseline: &PersonalityDims{T: 50, I: 50, S: 50, O: 50, R: 50},
		Personality:         PersonalitySlice{T: 50, I: 50, S: 50, O: 50, R: 50},
	}
	o.applyPeriodicDrift(&st, 20)
	drifted := false
	for _, v := range []float64{st.Personality.T, st.Personality.I, st.Personality.S, st.Personality.O, st.Personality.R} {
		if v != 50 {
			drifted = true
		}
	}
	if !drifted {
		t.Error("turn=20 应触发漂移")
	}
}

func TestApplyPeriodicDrift_NonDriftTurn(t *testing.T) {
	o := newTestOrch()
	st := FullState{
		PersonalityBaseline: &PersonalityDims{T: 50, I: 50, S: 50, O: 50, R: 50},
		Personality:         PersonalitySlice{T: 50, I: 50, S: 50, O: 50, R: 50},
	}
	o.applyPeriodicDrift(&st, 5) // 非漂移轮次
	if st.Personality.T != 50 {
		t.Error("turn=5 不应漂移")
	}
}

// ─── computeIntensityMod ─────────────────────────────────────

func TestComputeIntensityMod_StageFactors(t *testing.T) {
	o := newTestOrch()
	// Intimate → 1.2 倍
	modIntimate := o.computeIntensityMod(EmotionState{Aro: 0}, L1State{Stage: StageIntimate})
	// Stranger → 0.8 倍
	modStranger := o.computeIntensityMod(EmotionState{Aro: 0}, L1State{Stage: StageStranger})
	if modIntimate <= modStranger {
		t.Errorf("Intimate 调制应高于 Stranger: %f vs %f", modIntimate, modStranger)
	}
	// 都在合法范围
	for _, m := range []float64{modIntimate, modStranger} {
		if m < 0.5 || m > 1.5 {
			t.Errorf("调制因子越界: %f", m)
		}
	}
}

func TestComputeIntensityMod_HighAroBoost(t *testing.T) {
	o := newTestOrch()
	low := o.computeIntensityMod(EmotionState{Aro: 0}, L1State{Stage: StageFamiliar})
	high := o.computeIntensityMod(EmotionState{Aro: 100}, L1State{Stage: StageFamiliar})
	if high <= low {
		t.Errorf("高 Aro 调制应更高: %f vs %f", high, low)
	}
}

// ─── runAdultModeFSM ─────────────────────────────────────────

func TestRunAdultModeFSM_HardStop(t *testing.T) {
	o := newTestOrch()
	o.AdultMode = true
	o.adultStateStr = "FLIRTING"
	e := EmotionState{Aff: 50, Sec: 50, Aro: 0}
	state, level := o.runAdultModeFSM(Event{Type: "hurtful"}, "hurtful", &e, 5)
	if state != "NORMAL" {
		t.Errorf("hardStop 后状态 = %q, want NORMAL", state)
	}
	if o.adultLockTurns <= 0 {
		t.Error("hardStop 应设置锁定轮次")
	}
	_ = level
}

func TestRunAdultModeFSM_NormalLowIntensity(t *testing.T) {
	o := newTestOrch()
	o.AdultMode = true
	e := EmotionState{Aff: -100, Sec: -100, Aro: 0}
	state, level := o.runAdultModeFSM(Event{Type: "casual"}, "casual", &e, 1)
	if level != "none" {
		t.Errorf("最低强度 level = %q, want none", level)
	}
	// 中性强度（Aff=0）应至少 light（产品语义：(0+100)/200*0.5=0.25+0.15=0.4）
	e2 := EmotionState{Aff: 0, Sec: 0, Aro: 0}
	_, level2 := o.runAdultModeFSM(Event{Type: "casual"}, "casual", &e2, 2)
	if level2 == "none" {
		t.Error("中性强度不应为 none（score=0.4 ≥ 0.3）")
	}
	if state != "NORMAL" {
		t.Errorf("初始状态 = %q, want NORMAL", state)
	}
}

func TestRunAdultModeFSM_HighIntensityFlirting(t *testing.T) {
	o := newTestOrch()
	o.AdultMode = true
	e := EmotionState{Aff: 100, Sec: 100, Aro: 50}
	state, level := o.runAdultModeFSM(Event{Type: "casual", IsAdultContent: true}, "casual", &e, 1)
	if level != "high" {
		t.Errorf("高强度 level = %q, want high", level)
	}
	if state != "INTIMATE" {
		t.Errorf("成人内容高强度状态 = %q, want INTIMATE", state)
	}
}

func TestRunAdultModeFSM_Aftercare(t *testing.T) {
	o := newTestOrch()
	o.AdultMode = true
	// 先进入 FLIRTING（高 intensity 成人内容）
	e := EmotionState{Aff: 100, Sec: 100, Aro: 50}
	o.runAdultModeFSM(Event{Type: "casual", IsAdultContent: true}, "casual", &e, 1)
	if o.adultStateStr != "INTIMATE" {
		t.Fatalf("前置状态 = %q, want INTIMATE", o.adultStateStr)
	}
	// 回归最低强度（score=0 → none）→ AFTERCARE
	e2 := EmotionState{Aff: -100, Sec: -100, Aro: 0}
	state, _ := o.runAdultModeFSM(Event{Type: "casual"}, "casual", &e2, 2)
	if state != "AFTERCARE" {
		t.Errorf("回归后状态 = %q, want AFTERCARE", state)
	}
	if e2.Aro >= 0 {
		t.Errorf("AFTERCARE 应降低 Aro: %f", e2.Aro)
	}
}

func TestRunAdultModeFSM_ConsecutiveVulnerableLock(t *testing.T) {
	o := newTestOrch()
	o.AdultMode = true
	e := EmotionState{Aff: 50, Sec: 50, Aro: 0}
	// 连续 3 次 vulnerable → 负锁定
	for i := 0; i < 3; i++ {
		o.runAdultModeFSM(Event{Type: "vulnerable"}, "vulnerable", &e, i)
	}
	if o.adultLockTurns <= 0 {
		t.Error("连续脆弱应触发负锁定")
	}
}

func TestRunAdultModeFSM_LockedTurnScoreZero(t *testing.T) {
	o := newTestOrch()
	o.AdultMode = true
	o.adultLockTurns = 2
	e := EmotionState{Aff: 100, Sec: 100, Aro: 50}
	// 锁定中：高情绪也应为 none（score 被清零）
	_, level := o.runAdultModeFSM(Event{Type: "casual", IsAdultContent: true}, "casual", &e, 1)
	if level != "none" {
		t.Errorf("锁定期 level = %q, want none", level)
	}
}
