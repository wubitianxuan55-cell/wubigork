package whisper

import (
	"strings"
	"testing"
)

// 众神殿臂B刀④（DSH 共识案，docs/gaea-chat-probe-baseline-20261106.md）：压制
// 文案拆分——长度节奏保留、话题禁令移除、回应用户句为义务。六条压制行全部
// 不得再含「不开启/不要开启新话题」；psyche silent 行保留极短倾向但加接话义务。
func TestSuppressionCopyTopicBanRemoved(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere preset not found")
	}
	// Stranger + 低 Aff：gate 与 strategy 的 silent/whisper 全命中面
	orch := NewOrchestrator("test_suppression_copy", *preset)
	res := orch.PreLLMTurn("嗨")
	block := res.PsycheBlock + "\n" + res.DynamicContext
	for _, banned := range []string{"不开启任何新话题", "不要开启新话题"} {
		if strings.Contains(block, banned) {
			t.Errorf("压制面不得再含话题禁令 %q", banned)
		}
	}
	if !strings.Contains(block, "要好好回应") || !strings.Contains(block, "别敷衍用户的话") {
		t.Errorf("silent/whisper 新文案应含接话义务: %s", abbrevBlock(block))
	}
	if strings.Contains(block, "或用极短句回应") {
		t.Errorf("psyche silent 行应已换新文案")
	}
}

func abbrevBlock(s string) string {
	r := []rune(s)
	if len(r) > 200 {
		return string(r[:200])
	}
	return s
}
