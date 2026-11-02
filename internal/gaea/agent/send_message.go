package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// sendMessageTool 子代理→父代理中期回报（send_message 蒸馏 2026-11，上游
// tool-subagent-control 邻接双向的子→父半边；父→子已有 SteerSubagent）。
// 送达语义对齐上游模型面措辞：运行中的父在下一步界读到；空闲的父在下回合
// 读到（B3 收尾闸外入队的会被 drain 收进会话记录，不丢）。返回送达回执而
// 非答案——「Returns delivery confirmation, not the agent's answer」。
// 归因：经 SteerFromSubagent 署名信封进父上下文（agent-message 与 user 强制
// 分离，Z3/Z4 同款防洗白）。发消息不结束子代理回合；最终答复仍走任务结果。
//
// 挂载：buildSubReg 后按运行注入（prepareRun 拿到 sa_ ref 之后）——父注册表
// 永不持有本工具（它只属于子代理），兄弟运行各自实例互不可见。
type sendMessageTool struct {
	parent *AgentRunner // 宿主执行器（派发时 ctx 盖章取到；这里持引用投递）
	ref    string       // 本子代理运行引用（sa_…），署名必填
}

func (t *sendMessageTool) Name() string { return "send_message" }

func (t *sendMessageTool) Description() string {
	return "Send an interim message to your parent agent while your task is still running: report an interim finding that changes what the parent should do next, before your final result. The parent reads it at its next step boundary (or its next turn if idle). Returns delivery confirmation, not the parent's answer. Sending a message does not end your task; your final answer must still be your task result. Your parent does not automatically see your transcript, tool output, or reasoning — include everything the message needs on its own."
}

func (t *sendMessageTool) CompactDescription() string {
	return "向父代理发中期消息:运行中的父在下一步界读到(空闲则下回合),返回送达回执非答案;发消息不结束任务,最终答复仍走任务结果;父看不到你的过程,消息须自包含"
}

func (t *sendMessageTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"The self-contained message to deliver to the parent agent."}},"required":["message"]}`)
}

func (t *sendMessageTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}`)
}

func (t *sendMessageTool) ReadOnly() bool { return false }

func (t *sendMessageTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(p.Message) == "" {
		return "", fmt.Errorf("message is required")
	}
	parent := t.parent
	if parent == nil {
		if c, ok := RunnerFromContext(ctx); ok {
			parent = c
		}
	}
	if parent == nil {
		// 上游同款诚实拒绝：无常驻宿主概念的面（技能/测试直调）如实报错，
		// 不静默吞消息。
		return "", fmt.Errorf("send_message requires a live parent agent; none is available in this context")
	}
	ref := t.ref
	if ref == "" {
		ref = SubagentRefFromContext(ctx)
	}
	if ref == "" {
		return "", fmt.Errorf("send_message requires the subagent run reference for attribution; none is available")
	}
	parent.SteerFromSubagent(ref, p.Message)
	return "message delivered to your parent agent (it reads mid-turn messages at its next step boundary, or its next turn if idle). This does not end your task; your final answer must still be your task result.", nil
}
