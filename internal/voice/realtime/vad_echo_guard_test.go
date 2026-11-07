// vad_echo_guard_test.go — 播放期回声门（2026-11-05 真麦根修）。
// 恒 0.75 把弱电平真麦全 discard（实弹：segment 820ms 里 active 仅 64~192ms，
// 过不了 384ms 门，四段全丢 STT 零触发）——监听期降 silero 默认 0.5，播放期
// 回声门抬 0.75（2026-11-01 实弹：0.6 被扬声器回声触发假插话）。
package realtime

import "testing"

func TestVADEchoGuard(t *testing.T) {
	h := NewVADHandler(DefaultVADConfig(), nil, nil, NewSpeculativeTurns())
	if h.iter.threshold != 0.5 || h.baseThreshold != 0.5 {
		t.Fatalf("监听默认阈值应为 0.5, got th=%v base=%v", h.iter.threshold, h.baseThreshold)
	}
	// session.update 改阈值 = 更新回落基准
	h.SetThreshold(0.62)
	if h.baseThreshold != 0.62 {
		t.Fatalf("base 应随 SetThreshold 更新, got %v", h.baseThreshold)
	}
	// 开门抬 0.75；关门回 session.update 设定值（非 cfg 默认）
	h.EchoGuardOn()
	if !h.echoGuard || h.iter.threshold != echoGuardThreshold {
		t.Fatalf("开门后应 %v, got guard=%v th=%v", echoGuardThreshold, h.echoGuard, h.iter.threshold)
	}
	h.EchoGuardOff()
	if h.echoGuard || h.iter.threshold != 0.62 {
		t.Fatalf("关门应回 session.update 值 0.62, got guard=%v th=%v", h.echoGuard, h.iter.threshold)
	}
	// 幂等：重复开/关不抖动
	h.EchoGuardOn()
	h.EchoGuardOn()
	if h.iter.threshold != echoGuardThreshold {
		t.Fatalf("重复开门仍应 %v, got %v", echoGuardThreshold, h.iter.threshold)
	}
	h.EchoGuardOff()
	h.EchoGuardOff()
	if h.iter.threshold != 0.62 {
		t.Fatalf("重复关门仍应 0.62, got %v", h.iter.threshold)
	}
}

// energyScorer 能量感知假打分器：有能量=0.9（模拟 TTS 回声/真语音，silero 高分），
// 全零=0.05（静音）。fixedScorer 不辨静音会让段永不闭合、active 一直涨（首版测试坑）。
type energyScorer struct{}

func (energyScorer) Prob(x []float32) (float32, error) {
	for _, v := range x {
		if v != 0 {
			return 0.9, nil
		}
	}
	return 0.05, nil
}
func (energyScorer) Reset() {}

// feedMs 按 512 样本/块（32ms@16k）喂 n 毫秒音频，返回全部事件；speech=false 喂静音。
func feedMs(t *testing.T, h *VADHandler, ms int, speech bool) []VADEvent {
	t.Helper()
	chunk := make([]float32, 512)
	if speech {
		for i := range chunk {
			chunk[i] = 0.1
		}
	}
	n := ms / 32
	var out []VADEvent
	for i := 0; i < n; i++ {
		out = append(out, h.Process(chunk)...)
	}
	return out
}

func hasStarted(events []VADEvent) bool {
	for _, ev := range events {
		if ev.Kind == VADEventStarted {
			return true
		}
	}
	return false
}

// 回声门 v2（2026-11-07 真麦实弹）：播放期回声猝发 192~384ms 必须被确认窗拦下
// ——threshold 0.75 拦不住 silero 对 TTS 回声的高分，600ms 确认窗才是确定性闸。
func TestVADEchoGuardMinSpeech(t *testing.T) {
	h := NewVADHandler(DefaultVADConfig(), energyScorer{}, nil, NewSpeculativeTurns())

	// 门开（播放期）：384ms 猝发不过门——无 started；静音收段后整段 discard 不给 STT
	h.EchoGuardOn()
	evs := feedMs(t, h, 384, true)
	if hasStarted(evs) {
		t.Fatalf("播放期 384ms 猝发不应确认 started: %+v", evs)
	}
	evs = append(evs, feedMs(t, h, 200, false)...) // 静音收段（minSilence 64ms 触发 utterance）
	if hasStarted(evs) {
		t.Fatalf("384ms 回声段应被 discard，不得 started/soft-end 接受: %+v", evs)
	}

	// 同一门内持续 640ms 真打断：过 600ms 确认窗
	h2 := NewVADHandler(DefaultVADConfig(), energyScorer{}, nil, NewSpeculativeTurns())
	h2.EchoGuardOn()
	evs = feedMs(t, h2, 640, true)
	if !hasStarted(evs) {
		t.Fatal("播放期持续 640ms 语音应过 600ms 确认窗")
	}

	// 门关（监听期）：384ms 回到 MinSpeechMs 门，行为不变（救弱麦）
	h3 := NewVADHandler(DefaultVADConfig(), energyScorer{}, nil, NewSpeculativeTurns())
	evs = feedMs(t, h3, 384, true)
	if !hasStarted(evs) {
		t.Fatal("监听期 384ms 应按原 MinSpeechMs 门确认（回声门只动播放期）")
	}
}
