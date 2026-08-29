// Package whisper — 长期心境(Mood)测试（v4.3d）
//
// 覆盖：EWMA α 权重手算、冷启动播种、连续低落收敛、未被识别事件 Mood 不变、
// JSON 持久化 round-trip、记忆回声保留 Mood。
package whisper

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

// approx 浮点近似断言（1e-9 容差）
func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// TestStepMood_AlphaWeight 手算 2-3 轮 EWMA：α=MoodAlpha=0.01。
// 第 1 轮: 100*0.99+0*0.01=99；第 2 轮: 99*0.99+0*0.01=98.01；
// 第 3 轮: 98.01*0.99+0*0.01=97.0299。
func TestStepMood_AlphaWeight(t *testing.T) {
	emotion := Emotion4D{} // 目标全 0：每轮 Mood 各维收缩 1%

	// 第 1 轮
	m1 := stepMood([4]float64{100, 50, 0, -50}, emotion)
	if !approx(m1[0], 99) || !approx(m1[1], 49.5) || !approx(m1[2], 0) || !approx(m1[3], -49.5) {
		t.Fatalf("第1轮 EWMA 错误: %v", m1)
	}

	// 第 2 轮
	m2 := stepMood(m1, emotion)
	if !approx(m2[0], 98.01) || !approx(m2[1], 49.005) || !approx(m2[2], 0) || !approx(m2[3], -49.005) {
		t.Fatalf("第2轮 EWMA 错误: %v", m2)
	}

	// 第 3 轮
	m3 := stepMood(m2, emotion)
	if !approx(m3[0], 97.0299) || !approx(m3[1], 48.51495) || !approx(m3[2], 0) || !approx(m3[3], -48.51495) {
		t.Fatalf("第3轮 EWMA 错误: %v", m3)
	}
}

// TestStepMood_ConvergeTowardEmotion EWMA 向目标情绪靠拢：目标非 0 时
// 每轮向目标推进 1%，方向与幅度正确。
func TestStepMood_ConvergeTowardEmotion(t *testing.T) {
	emotion := Emotion4D{Aff: -100, Sec: 0, Aro: 0, Dom: 0}
	m := [4]float64{50, 50, 50, 50} // 非全 0 初始（播种语义由 TestStepMood_Seeding 单独覆盖）
	for i := 0; i < 100; i++ {
		m = stepMood(m, emotion)
	}
	// 100 轮后收敛 1-(0.99)^100 ≈ 63.4%：
	// Aff: 50 → 约 -45.1（向 -100 靠拢）；目标 0 的各维: 50 → 约 18.3
	if !(m[0] < -40 && m[0] > -50) {
		t.Fatalf("100轮后 Aff 应收敛到约 -45.1, got %v", m[0])
	}
	if !(math.Abs(m[1]) < 19 && math.Abs(m[2]) < 19 && math.Abs(m[3]) < 19) {
		t.Fatalf("目标 0 维度应收敛到约 18.3: %v", m)
	}
}

// TestStepMood_Seeding 新会话/无历史（prevMood 全 0）时以即时情绪首值播种。
func TestStepMood_Seeding(t *testing.T) {
	emotion := Emotion4D{Aff: 23.5, Sec: -8.25, Aro: 14, Dom: 3.75}
	got := stepMood([4]float64{}, emotion)
	want := [4]float64{23.5, -8.25, 14, 3.75}
	if got != want {
		t.Fatalf("全 0 应直接播种为即时情绪: got %v, want %v", got, want)
	}
}

// neutralMod 中性调制（测试用）
func neutralMod() Modulation {
	return Modulation{TrustMod: 1.0, RiftMod: 1.0, StageWeight: 1.0, Atmosphere: AtmoNeutral}
}

// TestEmotionStep_SeedsMoodOnFirstEvent 全新会话（EmotionState 零值）首个被识别
// 事件：Mood 直接播种为即时情绪首值。
func TestEmotionStep_SeedsMoodOnFirstEvent(t *testing.T) {
	prev := EmotionState{}
	ev := Event{Type: EvtPraise, Intensity: 0.5, Sincerity: 0.6}
	next := EmotionStep(ev, neutralMod(), prev, nil)
	want := [4]float64{next.Aff, next.Sec, next.Aro, next.Dom}
	if next.Mood != want {
		t.Fatalf("首个事件应播种 Mood: got %v, want %v", next.Mood, want)
	}
}

// TestEmotionStep_MoodConvergesLow 连续低落事件后 Mood 向低落收敛：
// 每轮严格递减（滞后于即时情绪，尚未跟上）、低于首值播种点、始终在 [-100,100]。
func TestEmotionStep_MoodConvergesLow(t *testing.T) {
	state := EmotionState{}
	prevAff := math.MaxFloat64
	for i := 0; i < 6; i++ {
		ev := Event{Type: EvtHurtful, Intensity: 1.0, Sincerity: 1.0}
		state = EmotionStep(ev, neutralMod(), state, nil)

		if state.Mood[0] >= prevAff {
			t.Fatalf("第%d轮 Mood.Aff 应严格递减: %v >= %v", i+1, state.Mood[0], prevAff)
		}
		prevAff = state.Mood[0]

		for _, v := range state.Mood {
			if v < -100 || v > 100 {
				t.Fatalf("第%d轮 Mood 越界: %v", i+1, state.Mood)
			}
		}
	}
	// 收敛：Mood.Aff 低于首值播种点（-10 附近）且仍为负（低落方向）
	if state.Mood[0] >= -10 || state.Mood[0] >= 0 {
		t.Fatalf("连续低落后 Mood.Aff 应向低落收敛, got %v", state.Mood[0])
	}
	// 滞后性：即时情绪跌得更深，Mood 在其上方（EWMA 慢速跟随）
	if state.Mood[0] <= state.Aff {
		t.Fatalf("Mood 应滞后于即时情绪（Mood=%v 应 > Aff=%v）", state.Mood[0], state.Aff)
	}
}

// TestEmotionStep_UnrecognizedEventKeepsMood 未被识别的事件轮：Mood 不变
// （与 Emotion 行为一致，EmotionStep 入口提前返回 prev）。
func TestEmotionStep_UnrecognizedEventKeepsMood(t *testing.T) {
	prev := EmotionState{
		PrimaryLabel: "CALM_RATIONAL",
		Mood:         [4]float64{5, -3, 7, 1},
	}
	// 未知事件类型（不在 baseStimulus 表）
	ev := Event{Type: EventType("mystery_event"), Intensity: 1.0, Sincerity: 1.0}
	next := EmotionStep(ev, neutralMod(), prev, nil)
	if next != prev {
		t.Fatalf("未知事件应原样返回 prev, got %+v", next)
	}

	// 极端红线事件（入口显式提前返回）
	prev2 := EmotionState{PrimaryLabel: "CALM_RATIONAL", Mood: [4]float64{9, 9, 9, 9}}
	ev2 := Event{Type: EvtExtremeRedline, IsExtremeRedline: true}
	next2 := EmotionStep(ev2, neutralMod(), prev2, nil)
	if next2 != prev2 {
		t.Fatalf("红线事件应原样返回 prev, got %+v", next2)
	}
}

// TestMood_JSONRoundTrip EmotionState JSON 序列化含 mood 字段，round-trip 一致。
func TestMood_JSONRoundTrip(t *testing.T) {
	es := EmotionState{
		Aff: 10, Sec: -5, Aro: 30, Dom: 0,
		PrimaryLabel: "SWEET_ATTACHMENT",
		IsLocked:     false,
		Mood:         [4]float64{8.5, -6.25, 4, 2},
	}
	b, err := json.Marshal(es)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"mood"`)) {
		t.Fatalf("序列化应含 mood 字段: %s", b)
	}
	var back EmotionState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Mood != es.Mood || back.Aff != es.Aff || back.PrimaryLabel != es.PrimaryLabel {
		t.Fatalf("round-trip 不一致: got %+v", back)
	}
}

// TestMood_FullStatePersistence Mood 随 FullState 整体 JSON 持久化：
// 显式设置 Mood 后序列化包含 mood，反序列化恢复；零值 Mood 也全量写出（无 omitempty）。
func TestMood_FullStatePersistence(t *testing.T) {
	fs := DefaultFullState(PersonalitySlice{PresetID: "gaea"})
	fs.Emotion.Mood = [4]float64{-40, -30, -25, -10}
	fs.Emotion.PrimaryLabel = "HURT_GRIEVANCE"

	b, err := json.Marshal(fs)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"mood":[-40,-30,-25,-10]`)) {
		t.Fatalf("FullState 序列化应含 mood 全量字段: %s", b)
	}

	var back FullState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Emotion.Mood != fs.Emotion.Mood {
		t.Fatalf("FullState round-trip Mood 不一致: got %v", back.Emotion.Mood)
	}

	// 零值 Mood（默认状态）也应写出，保证显式可观测
	b0, err := json.Marshal(DefaultFullState(PersonalitySlice{PresetID: "gaea"}))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b0, []byte(`"mood":[0,0,0,0]`)) {
		t.Fatalf("默认状态也应写出 mood 全零字段: %s", b0)
	}
}

// TestApplyMemoryEcho_PreservesMood 记忆回声按值拷贝 EmotionState，Mood 应保留。
func TestApplyMemoryEcho_PreservesMood(t *testing.T) {
	es := EmotionState{
		Aff: 10, Sec: 10, Aro: 10, Dom: 10,
		PrimaryLabel: "CALM_RATIONAL",
		Mood:         [4]float64{3, -4, 5, -6},
	}
	got := ApplyMemoryEcho(es, MemoryEcho{Aff: 20, Sec: -20})
	if got.Mood != es.Mood {
		t.Fatalf("记忆回声应保留 Mood: got %v, want %v", got.Mood, es.Mood)
	}
	if got.Aff != 30 || got.Sec != -10 {
		t.Fatalf("记忆回声应叠加四维: got %+v", got)
	}
}

// TestMoodLowAroAffMean_Threshold 低落判定阈值常量语义：均值 = (Aro+Aff)/2。
func TestMoodLowAroAffMean_Threshold(t *testing.T) {
	low := [4]float64{-30, 0, -25, 0}   // (Aro+Aff)/2 = -27.5 < -20 → 低落
	calm := [4]float64{-10, 0, -10, 0}  // (Aro+Aff)/2 = -10 ≥ -20 → 正常
	mean := func(m [4]float64) float64 { return (m[2] + m[0]) / 2 }
	if !(mean(low) < MoodLowAroAffMean) {
		t.Fatalf("低落样本均值 %v 应低于阈值 %v", mean(low), MoodLowAroAffMean)
	}
	if mean(calm) < MoodLowAroAffMean {
		t.Fatalf("正常样本均值 %v 不应低于阈值 %v", mean(calm), MoodLowAroAffMean)
	}
}
