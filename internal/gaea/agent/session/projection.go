package session

// 3.0 Step 1: 投影层。ProjectMessages 是纯函数：只有 user/assistant/tool
// 三类事件（外加迁移用的 system_message）投影为 provider.Message，其余
// 事件（usage/notice/phase/审批/提问/压缩/重试/steer/turn 边界等）不投影。
// 接口与现有 Session.Messages 完全兼容：projected 消息可直接喂给模型
// （含 assistant 的 ToolCalls 与 tool 消息的 ToolCallID 配对）。

import (
	"encoding/json"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// ProjectMessages 把日志条目流投影为消息列表（保持事件顺序）。
// 语义：
//   - user_message / system_message          → user / system 消息
//   - assistant_started                      → 打开一条待累积的 assistant 消息
//   - assistant_delta / text                 → 累积到当前 assistant 消息的 Content
//   - reasoning                              → 累积到当前 assistant 消息的 ReasoningContent
//   - assistant_message / message            → 终结并落一条完整 assistant 消息
//   - tool_call / tool_dispatch              → 给当前 assistant 消息附加 ToolCall
//   - tool_result                            → 终结当前 assistant 消息并落一条 tool 消息
//
// 其余 kind 全部忽略（它们不是消息）。
func ProjectMessages(entries []LogEntry) []provider.Message {
	var out []provider.Message
	pending := -1 // 当前正在累积的 assistant 消息下标；-1 = 无

	closePending := func() { pending = -1 }
	ensurePending := func() *provider.Message {
		if pending < 0 {
			out = append(out, provider.Message{Role: provider.RoleAssistant})
			pending = len(out) - 1
		}
		return &out[pending]
	}

	for _, e := range entries {
		switch e.Kind {
		case KindSystemMessage:
			closePending()
			var p userLogPayload
			_ = json.Unmarshal(e.Payload, &p)
			out = append(out, provider.Message{Role: provider.RoleSystem, Content: p.Content, Name: p.Name})

		case KindUserMessage:
			closePending()
			var p userLogPayload
			_ = json.Unmarshal(e.Payload, &p)
			out = append(out, provider.Message{Role: provider.RoleUser, Content: p.Content, Name: p.Name})

		case KindAssistantStarted:
			closePending()
			out = append(out, provider.Message{Role: provider.RoleAssistant})
			pending = len(out) - 1

		case KindAssistantDelta, "text":
			m := ensurePending()
			var p struct {
				Text string `json:"text"`
			}
			_ = json.Unmarshal(e.Payload, &p)
			m.Content += p.Text

		case "reasoning":
			m := ensurePending()
			var p struct {
				Text string `json:"text"`
			}
			_ = json.Unmarshal(e.Payload, &p)
			m.ReasoningContent += p.Text

		case KindAssistantMessage, "message":
			// 完整消息到达：若已累积流式 delta 则原地替换（全文是 delta 的超集），
			// 否则追加。注意此处不关闭 pending：运行期 executor 中文本与工具调用
			// 属于同一条 assistant 消息，后续 tool_dispatch 仍要附加到它；
			// pending 由 tool_result / user / system / 下一条 assistant_started 终结。
			var p assistantLogPayload
			_ = json.Unmarshal(e.Payload, &p)
			if pending >= 0 {
				out[pending].Content = p.Text
				out[pending].ReasoningContent = p.Reasoning
				out[pending].ReasoningSignature = p.ReasoningSignature
				out[pending].ToolCallID = p.ID
				out[pending].ToolCalls = out[pending].ToolCalls[:0]
				for _, tc := range p.ToolCalls {
					out[pending].ToolCalls = append(out[pending].ToolCalls, provider.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Args})
				}
			} else {
				msg := provider.Message{
					Role:               provider.RoleAssistant,
					Content:            p.Text,
					ReasoningContent:   p.Reasoning,
					ReasoningSignature: p.ReasoningSignature,
					ToolCallID:         p.ID,
				}
				for _, tc := range p.ToolCalls {
					msg.ToolCalls = append(msg.ToolCalls, provider.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Args})
				}
				out = append(out, msg)
				pending = len(out) - 1
			}

		case KindToolCall, "tool_dispatch":
			m := ensurePending()
			var p toolCallLogPayload
			_ = json.Unmarshal(e.Payload, &p)
			m.ToolCalls = append(m.ToolCalls, provider.ToolCall{ID: p.ID, Name: p.Name, Arguments: p.Args})

		case KindToolResult:
			closePending()
			var p toolResultLogPayload
			_ = json.Unmarshal(e.Payload, &p)
			content := p.Output
			if content == "" {
				content = p.Err
			}
			out = append(out, provider.Message{
				Role:       provider.RoleTool,
				Content:    content,
				ToolCallID: p.ID,
				Name:       p.Name,
			})
		}
	}
	return out
}

// lastLogSeq 返回条目流中最后的 seq（无条目返回 0）。
func lastLogSeq(entries []LogEntry) int64 {
	if len(entries) == 0 {
		return 0
	}
	return entries[len(entries)-1].Seq
}
