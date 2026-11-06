// vad_prefix_test.go — 新轮开清前缀守卫（2026-11-06 实弹翻车主修）。
// 上游 _ensure_turn_for_speech_start 同位守卫在 Go 刀3 移植时漏抄：audioPrefix/
// rawPrefix 跨轮只增不清，每轮 final 音频=全会话历史语音，STT 逐轮重转全文
// （实弹转写字符数 2→6→13→…→54 滚雪球，角色答非所问+输入复读循环）。
package realtime

import "testing"

func TestVADPrefixClearedOnNewTurn(t *testing.T) {
	h := NewVADHandler(DefaultVADConfig(), nil, nil, NewSpeculativeTurns())
	seed := make([]float32, 1600) // 模拟上一轮遗留前缀（0.1s@16k）

	// 首轮开：新轮（rev=0 非重开）→ 前缀清零
	h.audioPrefix = seed
	h.rawPrefix = seed
	turnID, rev, reopened := h.ensureTurnForSpeechStart(100)
	if reopened || rev != 0 || turnID == "" {
		t.Fatalf("首轮应开新轮, got turn=%s rev=%d reopened=%v", turnID, rev, reopened)
	}
	if h.audioPrefix != nil || h.rawPrefix != nil {
		t.Fatal("新轮开未清前缀——跨轮滚雪球会复发")
	}

	// 同轮重开窗内续说：reopen（rev+1）→ 前缀保留（重开转写依赖前段音频拼接）
	h.audioPrefix = seed
	h.rawPrefix = seed
	h.turns.SegmentFinalized(1000, 800, 0)
	turnID, rev, reopened = h.ensureTurnForSpeechStart(1500)
	if !reopened || rev != 1 {
		t.Fatalf("重开窗内应 rev+1, got turn=%s rev=%d reopened=%v", turnID, rev, reopened)
	}
	if len(h.audioPrefix) != len(seed) || len(h.rawPrefix) != len(seed) {
		t.Fatal("同轮 reopen 不得清前缀")
	}

	// 重开窗已过（音频时钟超 unansweredMs=7000）：开新轮 → 前缀清零
	h.turns.SegmentFinalized(2000, 800, 0)
	turnID, rev, reopened = h.ensureTurnForSpeechStart(2000 + defaultUnansweredMs + 1000)
	if reopened || rev != 0 {
		t.Fatalf("超窗应开新轮, got turn=%s rev=%d reopened=%v", turnID, rev, reopened)
	}
	if h.audioPrefix != nil || h.rawPrefix != nil {
		t.Fatal("超窗新轮未清前缀——上一轮音频会滚进下一轮转写")
	}
}
