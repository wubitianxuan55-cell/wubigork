package whisper

import (
	"strings"
	"testing"
)

// TestPreLLMTurn_SystemPromptCrossTurnStable KV 前缀稳定化核心契约（2026-11-01
// 提速刀）：system 提示跨轮逐字节不变（引擎前缀缓存据此跨轮命中），每轮变化段
// 走 DynamicContext。若本测试红=有动态段漏进了 system，前缀缓存将被每轮打掉，
// 长会话每轮全量冷 prefill 的 10~26s 慢病复发。
func TestPreLLMTurn_SystemPromptCrossTurnStable(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere preset not found")
	}
	orch := NewOrchestrator("test_prefix_stability", *preset)

	r1 := orch.PreLLMTurn("你好呀")
	r2 := orch.PreLLMTurn("今天过得怎么样？")
	r3 := orch.PreLLMTurn("那我们一起去散步吧")

	if r1.SystemPrompt == "" {
		t.Fatal("system prompt should not be empty")
	}
	if r1.SystemPrompt != r2.SystemPrompt || r2.SystemPrompt != r3.SystemPrompt {
		t.Errorf("system prompt 必须跨轮逐字节稳定（前缀缓存命中前提）\n轮1==轮2: %v 轮2==轮3: %v", r1.SystemPrompt == r2.SystemPrompt, r2.SystemPrompt == r3.SystemPrompt)
	}
	// 动态段非空且逐轮不同（状态机/情绪演进）
	if r1.DynamicContext == "" {
		t.Error("dynamic context should not be empty")
	}
	if r1.DynamicContext == r2.DynamicContext {
		t.Error("dynamic context should differ across turns（情绪/状态机每轮演进）")
	}
}

// TestPreLLMTurn_StableIdentityInSystem 动态/稳定分界：产品守卫与 Canon 身份块
// 必须在 system 稳定前缀里（人格身份不随轮漂移）；情绪/状态机数值在动态段。
func TestPreLLMTurn_StableIdentityInSystem(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere preset not found")
	}
	orch := NewOrchestrator("test_identity_split", *preset)
	r := orch.PreLLMTurn("你好")

	if !strings.Contains(r.SystemPrompt, "gaea") && !strings.Contains(r.SystemPrompt, "盖亚") {
		t.Errorf("system 稳定段应含产品/助手身份（守卫或 Canon 块），got: %.200s", r.SystemPrompt)
	}
	// PsycheBlock 完整语义保留：稳定身份段（canon）+动态心理状态块都在
	if !strings.Contains(r.PsycheBlock, "心理状态") {
		t.Error("PsycheBlock 应保留完整 psyche 语义（含动态心理状态块）")
	}
	// 动态心理状态块确实在动态路（system 不含）——分界成立
	if strings.Contains(r.SystemPrompt, "心理状态") {
		t.Error("心理状态块（每轮变化）不得进 system 稳定前缀")
	}
}

// TestPromptHistory_AppendOnlyAndHysteresis 四刀契约：提示历史 append-only +
// 滞回裁剪——RecentExchanges 的前 N-1 条在追加后保持不变（前缀命中前提），
// 超 HistoryMaxTurns 一次性裁到 HistoryKeepTurns。
func TestPromptHistory_AppendOnlyAndHysteresis(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere not found")
	}
	orch := NewOrchestrator("test_prompt_history", *preset)
	orch.HistoryInMessages = true

	r1 := orch.PreLLMTurn("第一句")
	if len(r1.RecentExchanges) != 0 {
		t.Fatalf("首轮历史应为空，got %d", len(r1.RecentExchanges))
	}
	// 模拟 handler：回复后记实际发送内容
	orch.RecordPromptTurn(1, "第一句(含动态前缀)", "回复一")
	r2 := orch.PreLLMTurn("第二句")
	if len(r2.RecentExchanges) != 1 || r2.RecentExchanges[0].UserText != "第一句(含动态前缀)" || r2.RecentExchanges[0].AssistantText != "回复一" {
		t.Fatalf("第二轮历史应含第一轮实际发送对，got %+v", r2.RecentExchanges)
	}
	// 追加多轮验证滞回裁剪：超 Max 一次性裁到 Keep
	for i := 2; i <= HistoryMaxTurns+2; i++ {
		orch.RecordPromptTurn(i, "轮次"+strings.Repeat("话", i), "回复")
	}
	snap := orch.PreLLMTurn("再问")
	if len(snap.RecentExchanges) > HistoryMaxTurns || len(snap.RecentExchanges) < HistoryKeepTurns {
		t.Errorf("滞回裁剪后轮数应在 [%d,%d] 区间，got %d", HistoryKeepTurns, HistoryMaxTurns, len(snap.RecentExchanges))
	}
	// 前缀不变性：连续两轮 PreLLMTurn 间，上一轮快照是新快照的前缀
	orch.RecordPromptTurn(99, "最新轮", "最新回复")
	next := orch.PreLLMTurn("下一问")
	for i, ex := range snap.RecentExchanges {
		if next.RecentExchanges[i] != ex {
			t.Errorf("提示历史第 %d 条被改写（append-only 破坏，前缀缓存将 miss）", i)
		}
	}
}
