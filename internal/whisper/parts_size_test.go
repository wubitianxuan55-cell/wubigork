package whisper

import (
	"strings"
	"testing"
)

// 一次性分段测量（回答提示词构成）：打印稳定段/动态段/历史的字符占比
func TestPreLLMTurn_PartsSizeBreakdown(t *testing.T) {
	preset := GetPreset("tsundere")
	if preset == nil {
		t.Fatal("tsundere not found")
	}
	orch := NewOrchestrator("test_parts_size", *preset)
	orch.HistoryInMessages = true
	// 预灌 6 轮工作记忆（满窗）
	for i := 0; i < 6; i++ {
		orch.WM.Push("test_parts_size", Exchange{TurnIndex: i, UserText: "用户消息样例第八九十十一十二十三", AssistantText: "助手回复样例，大约一两句话的长度这样子"})
	}
	r := orch.PreLLMTurn("你好")
	t.Logf("system稳定段=%d 字符", len(r.SystemPrompt))
	t.Logf("动态段=%d 字符", len(r.DynamicContext))
	hist := 0
	for _, ex := range r.RecentExchanges {
		hist += len(ex.UserText) + len(ex.AssistantText)
	}
	t.Logf("历史messages=%d 字符（%d 轮）", hist, len(r.RecentExchanges))
	t.Logf("system前200字符: %.200s", r.SystemPrompt)
	t.Logf("动态段前600字符: %.600s", r.DynamicContext)
	if strings.Contains(r.SystemPrompt, "样例") {
		t.Error("对话样例不得进 system")
	}
}
