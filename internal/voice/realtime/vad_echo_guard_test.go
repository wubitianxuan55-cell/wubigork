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
