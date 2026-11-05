// voice_v2_tools_test.go — M5 刀二：模型工具通道测试。
// 分派（T0 回执/T1 台账/频控/清单外拒绝）+ Reply 集成（tool_calls 聚合→ack
// 经 onDelta 下发）。执行层桩化（同 voice_v2_action_test 纪律）。
package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/intent"
)

func newVoiceV2ToolTestApp(t *testing.T) *App {
	t.Helper()
	a := newVoiceV2ActionTestApp(t) // 台账清空+执行桩
	resetVoiceV2ToolRate()
	t.Cleanup(resetVoiceV2ToolRate)
	a.mediaState = &mediaState{core: a.core, app: a}
	return a
}

func toolCall(name, args string) ai.ChatToolCall {
	return ai.ChatToolCall{ID: "call_1", Type: "function",
		Function: ai.ChatToolFunction{Name: name, Arguments: args}}
}

func TestVoiceV2ToolDispatchStatus(t *testing.T) {
	a := newVoiceV2ToolTestApp(t)
	ack, preview := a.mediaState.dispatchVoiceV2Tool(toolCall("get_status", `{}`))
	if preview != nil {
		t.Fatalf("get_status=T0 不得出预览: %v", preview)
	}
	if !strings.Contains(ack, "状态") {
		t.Fatalf("status 回执异常: %q", ack)
	}
}

func TestVoiceV2ToolDispatchSaveTaskLedger(t *testing.T) {
	a := newVoiceV2ToolTestApp(t)
	ack, preview := a.mediaState.dispatchVoiceV2Tool(toolCall("save_task", `{"title":"  写周报  "}`))
	if preview == nil {
		t.Fatalf("save_task=T1 必须登记台账出预览")
	}
	id, _ := preview["previewId"].(string)
	hash, _ := preview["intentHash"].(string)
	if id == "" || hash == "" || preview["action"] != string(intent.ActionSaveTask) {
		t.Fatalf("预览凭据缺失: %v", preview)
	}
	if !strings.Contains(ack, "写周报") || !strings.Contains(ack, "确认") {
		t.Fatalf("回执应含任务名与确认引导: %q", ack)
	}
	voiceV2PreviewMu.Lock()
	p := voiceV2PreviewLedger[id]
	voiceV2PreviewMu.Unlock()
	if p == nil || p.target != "写周报" || p.hash != hash {
		t.Fatalf("台账未登记/内容不符: %+v", p)
	}
	// 台账凭据可经 ExecuteAction 幂等执行（P2 同一面）
	res := a.VoiceV2ExecuteAction(id, hash)
	if res["executed"] != true {
		t.Fatalf("台账凭据应可执行: %v", res)
	}
}

func TestVoiceV2ToolRateLimit(t *testing.T) {
	a := newVoiceV2ToolTestApp(t)
	for i := 0; i < 3; i++ {
		ack, _ := a.mediaState.dispatchVoiceV2Tool(toolCall("get_status", `{}`))
		if strings.Contains(ack, "频繁") {
			t.Fatalf("前 3 次不应频控: %q", ack)
		}
	}
	ack, preview := a.mediaState.dispatchVoiceV2Tool(toolCall("get_status", `{}`))
	if preview != nil || !strings.Contains(ack, "频繁") {
		t.Fatalf("第 4 次应频控拒绝: ack=%q preview=%v", ack, preview)
	}
	// 另一 action 独立计数
	ack2, _ := a.mediaState.dispatchVoiceV2Tool(toolCall("save_task", `{"title":"x"}`))
	if strings.Contains(ack2, "频繁") {
		t.Fatalf("save_task 应独立计数: %q", ack2)
	}
	// 窗口滑出即恢复
	voiceV2ToolRateNow = func() time.Time { return time.Now().Add(2 * voiceV2ToolRateWin) }
	t.Cleanup(func() { voiceV2ToolRateNow = time.Now })
	if ack3, _ := a.mediaState.dispatchVoiceV2Tool(toolCall("get_status", `{}`)); strings.Contains(ack3, "频繁") {
		t.Fatalf("窗口滑出应恢复: %q", ack3)
	}
}

func TestVoiceV2ToolUnknownAndBadArgs(t *testing.T) {
	a := newVoiceV2ToolTestApp(t)
	ack, preview := a.mediaState.dispatchVoiceV2Tool(toolCall("send_weixin", `{"to":"x"}`))
	if preview != nil || !strings.Contains(ack, "没有开通") {
		t.Fatalf("清单外工具应诚实拒绝: %q %v", ack, preview)
	}
	ack2, preview2 := a.mediaState.dispatchVoiceV2Tool(toolCall("save_task", `{"title":"   "}`))
	if preview2 != nil || !strings.Contains(ack2, "没听清") {
		t.Fatalf("空标题应诚实回问: %q %v", ack2, preview2)
	}
}

func TestVoiceNativeLLMLegToolCallsToAck(t *testing.T) {
	a := newVoiceV2ToolTestApp(t)
	oldStream := voiceV2ChatLLMStream
	t.Cleanup(func() { voiceV2ChatLLMStream = oldStream })
	var gotTools []ai.ChatToolSchema
	voiceV2ChatLLMStream = func(_ context.Context, _ *mediaState, _ []ai.ChatMessage, tools []ai.ChatToolSchema) (<-chan ai.SSEChunk, error) {
		gotTools = tools
		ch := make(chan ai.SSEChunk, 4)
		ch <- ai.SSEChunk{Content: "让我"}
		ch <- ai.SSEChunk{Content: "查一下"}
		ch <- ai.SSEChunk{Done: true, ToolCalls: []ai.ChatToolCall{toolCall("get_status", `{}`)}}
		close(ch)
		return ch, nil
	}

	var deltas []string
	full, _, _, err := voiceNativeLLMLeg{a: a.mediaState}.Reply(
		context.Background(), "现在用什么模型", nil, "turn_t1", 0, "item_t1",
		func(d string) { deltas = append(deltas, d) },
	)
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if len(gotTools) != len(voiceV2ToolSchemas) {
		t.Fatalf("tools 未注入请求: %d", len(gotTools))
	}
	// 工具轮回复=模型前缀（已播半句）+ 执行回执 ack 追加——transcript 与已播
	// 音频同序一致（前缀在 tool_calls 确定前已流经管线，不可撤回）。
	if full != "让我查一下当前状态：测试引擎（本地）。" {
		t.Fatalf("工具轮 full 应为前缀+ack: %q", full)
	}
	if len(deltas) != 3 || deltas[2] != "当前状态：测试引擎（本地）。" {
		t.Fatalf("ack 应经 onDelta 追加下发: %v", deltas)
	}
}
