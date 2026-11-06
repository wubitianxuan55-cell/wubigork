package whisper

import (
	"strings"
	"testing"
)

// B3 批回归（2026-11-06）：①回复面节奏标记归一扩剥裸 SPLIT 降级变体
// （复现探针 R4 泄漏实证）②PromptHistory 净行契约（存原句，动态块与搜索
// 增广不再跨轮重放——病灶①最后一面）。
func TestNormalizeReplySplitLines(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"协议标记分条归一", "好呀！[SPLIT]今天去哪玩[SPLIT][SPLIT]我准备好了", "好呀！\n今天去哪玩\n我准备好了"},
		{"裸降级变体行剥除", "好的。\n\n《秋午偶得》\n檐角风铃静\n窗间日影斜\n心随云去远\n无事即天涯\n嘿嘿，写好了。\nSPLIT", "好的。\n\n《秋午偶得》\n檐角风铃静\n窗间日影斜\n心随云去远\n无事即天涯\n嘿嘿，写好了。"},
		{"句中英文词 SPLIT 不误伤", "这个项目用大写 SPLIT 表示拆分", "这个项目用大写 SPLIT 表示拆分"},
		{"无标记原样返回", "普通回复\n第二行", "普通回复\n第二行"},
	}
	for _, c := range cases {
		if got := NormalizeReplySplitLines(c.in); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// TestPromptHistory_StoresRawUserText 净行契约：历史行存原句（无动态包裹/
// 搜索增广），同进程重放与重启回灌（warmPromptHistory 原句）/语音出口
// （voice_v2 原句）三口径一致。挂具语义测试（handler 侧调用点见
// whisper_handler RecordPromptTurn 注释）。
func TestPromptHistory_StoresRawUserText(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere preset not found")
	}
	orch := NewOrchestrator("test_prompt_history_raw", *preset)
	orch.HistoryInMessages = true
	orch.RecordPromptTurn(1, "第一句原句", "回复一")
	orch.RecordPromptTurn(2, "第二句原句", "回复二")
	snap := orch.PreLLMTurn("第三句")
	if len(snap.RecentExchanges) != 2 {
		t.Fatalf("历史应 2 行，got %d", len(snap.RecentExchanges))
	}
	for i, ex := range snap.RecentExchanges {
		if strings.Contains(ex.UserText, "【本轮上下文】") || strings.Contains(ex.UserText, "用户消息：") {
			t.Errorf("历史第 %d 行含动态包裹标记（净行契约破坏）: %.80s", i, ex.UserText)
		}
	}
	if snap.RecentExchanges[0].UserText != "第一句原句" {
		t.Errorf("历史行应为原句，got %.60s", snap.RecentExchanges[0].UserText)
	}
}
