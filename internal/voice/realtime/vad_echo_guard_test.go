// vad_echo_guard_test.go — 播放期回声门（2026-11-05 真麦根修）。
// 恒 0.75 把弱电平真麦全 discard（实弹：segment 820ms 里 active 仅 64~192ms，
// 过不了 384ms 门，四段全丢 STT 零触发）——监听期降 silero 默认 0.5，播放期
// 回声门抬 0.75（2026-11-01 实弹：0.6 被扬声器回声触发假插话）。
package realtime

import (
	"testing"
	"time"
)

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

// 回声门 v3（2026-11-07 真麦二轮实弹，上游 block_mic_during_playback 同位）：
// 播放期捕获侧整帧丢弃——v2 的 600ms 确认窗输给连续回声（active=608ms 过门），
// 长句 TTS 回声可过任意有限确认窗；半双工=VAD 零摄入，语音打断让位 UI/协议
// interrupt 通道；门关后尾窗 600ms 继续丢弃压回声余响（上游无此层，Go 实弹加设）。
func TestVADEchoGuardHalfDuplex(t *testing.T) {
	h := NewVADHandler(DefaultVADConfig(), energyScorer{}, nil, NewSpeculativeTurns())

	// 播放期：持续 640ms 回声整帧丢弃——零事件、时间轴冻结（无触发态/段缓冲可泄漏）
	h.EchoGuardOn()
	evs := feedMs(t, h, 640, true)
	if len(evs) != 0 {
		t.Fatalf("播放期应整帧丢弃（零事件）: %+v", evs)
	}
	if h.AudioMs() != 0 {
		t.Fatalf("丢弃帧不计时间轴，AudioMs=%d", h.AudioMs())
	}

	// 门关：尾窗 600ms 内仍压制（回声余响+AGC 余振）
	h.EchoGuardOff()
	evs = feedMs(t, h, 300, true)
	if hasStarted(evs) {
		t.Fatalf("尾窗内新轮应被压制: %+v", evs)
	}

	// 尾窗过后：监听期 384ms 按 MinSpeechMs 门正常确认（救弱麦行为不变）
	time.Sleep(700 * time.Millisecond) // 尾窗过期
	evs = feedMs(t, h, 384, true)
	if !hasStarted(evs) {
		t.Fatal("尾窗后监听期 384ms 应正常确认")
	}
}

// 回声门真实锚点（2026-11-07 末轮实弹：合成毕 +0.4s 即确认出 4 个幽灵轮
// =「几段语音叠加」真身）：合成终态不再关门，锚=客户端播放排空回执。
func TestEchoGateDrainAnchored(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "好的"})
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	deadline := time.Now().Add(3 * time.Second)
	finalized := false
	for time.Now().Before(deadline) {
		p.muHist.Lock()
		for _, m := range p.chat {
			if m.Role == "assistant" && !m.provisional {
				finalized = true
			}
		}
		p.muHist.Unlock()
		if finalized {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !finalized {
		t.Fatal("前置：响应未在 3s 内完成")
	}
	if !p.vad.echoGuard {
		t.Fatal("合成终态后回声门应保持开启（等播放排空回执）——提前关门=尾段回声进 VAD")
	}
	p.OnPlaybackDrained(p.session)
	if p.vad.echoGuard {
		t.Fatal("播放排空回执应关闭回声门")
	}
}

// TestReopenSupersedeSilentCancel — 2026-11-07「说散文根本不输出」回归钉：
// 用户连续短语 → rev 级联，中间 rev 的响应**从未安装槽位**（delta 停靠/被
// DROP）→ 旧 barge-in 条件（inResponse&&rs!=nil）恒假 → 取消丢失 → 帧尾
// 哨兵以 audioEmitted=0 判 empty_output+O2 回滚=散文链式蒸发。修复后：
// 打断性 speech_started 无条件全量收口，排队 rev 落墓碑静默消失。
func TestReopenSupersedeSilentCancel(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "好的我来说", slow: true})
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	time.Sleep(30 * time.Millisecond) // llm 起流（delta 尚未必达发送循环）

	// rev1 打断性 speech_started（同轮重开+interrupt）
	p.dispatchText(&textEvent{kind: evKindSpeechStarted, started: &VADStarted{
		TurnID: "turn_1", Rev: 1, Reopened: true, InterruptResponse: true}})
	time.Sleep(80 * time.Millisecond)

	p.muTerm.Lock()
	tombstoned := len(p.closedKeys) > 0
	queuedLeft := len(p.queuedResponses)
	p.muTerm.Unlock()
	if queuedLeft != 0 {
		t.Fatalf("打断后排队响应应清空, 剩 %d", queuedLeft)
	}
	if !tombstoned {
		t.Fatal("被打断的排队响应应落墓碑（哨兵静默跳过, 不得 empty_output）")
	}
	// 帧尾哨兵到达 → 墓碑拦截 → 无线缆终态帧（前端不见幽灵 response.done）
	if n := countRoles(p, "assistant"); n != 0 {
		t.Fatalf("被收口的排队响应不得写历史, 残留 %d 条", n)
	}
}
