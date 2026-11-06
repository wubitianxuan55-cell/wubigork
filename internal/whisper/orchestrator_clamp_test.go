package whisper

import (
	"strings"
	"testing"
)

// 众神殿臂B刀③（docs/gaea-chat-probe-baseline-20261106.md）：≤15 字钳制豁免
// 祈使/任务句——「讲一个短故事」6 字吃 terse 谓词但 15 字装不下故事；
// 纯寒暄短句照旧限长（DSH 共识：可以限长，不能禁止接话）。
func TestClampExemptsImperative(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere preset not found")
	}
	cases := []struct {
		msg       string
		wantClamp bool
	}{
		{"讲一个短故事", false}, // 6 字 terse + 祈使 → 豁免
		{"写一首小诗", false},
		{"嗯", true}, // 纯寒暄短句照旧限长
		{"哈喽", true},
		{"你是谁", false}, // 短问句既有豁免
	}
	for _, c := range cases {
		orch := NewOrchestrator("test_clamp_"+c.msg, *preset)
		res := orch.PreLLMTurn(c.msg)
		got := strings.Contains(res.PsycheBlock, "不超过15字")
		if got != c.wantClamp {
			t.Errorf("PreLLMTurn(%q) 钳制注入 = %v, want %v", c.msg, got, c.wantClamp)
		}
	}
}
