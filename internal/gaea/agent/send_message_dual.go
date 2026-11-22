package agent

// send_message_dual.go — 嵌套中间层的双向 send_message（pan_1795306273871
// action-4：中间层子代理既是子又是父——同名工具此前被子→父版覆盖，丢失
// 「向下 steer 孙代」能力，运行时测试只锁名字未锁方向）。
//
// 合并语义（单一 "send_message"，按参数分向）：
//   - 带 agent_id → 向下（父向版 SendMessageTool；子树守卫在其 Execute 内）
//   - 缺 agent_id → 向上（子→父中期回报 sendMessageTool）
//
// 叶子层（到底）只注入向上版（现状）；主会话只有向下版（现状）；只有嵌套
// 中间层用本合并件。

import (
	"context"
	"encoding/json"
)

// dualSendMessageTool 双向 send_message：向上（本运行的宿主）+ 向下（自己
// 子树里的 sa_ 引用）。
type dualSendMessageTool struct {
	up   *sendMessageTool
	down *SendMessageTool
}

func (d *dualSendMessageTool) Name() string { return "send_message" }

func (d *dualSendMessageTool) Description() string {
	return "Send a message, direction by arguments: omit agent_id to send an interim message to YOUR parent agent (report a finding that changes what the parent should do next; the parent reads it at its next step boundary; returns delivery confirmation, not an answer; does not end your task). Pass agent_id (an sa_… reference from your own sub-agent tree) to steer a child: running = injected as steering; queued = held until it starts; finished = starts a follow-up turn."
}

func (d *dualSendMessageTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"The self-contained message to deliver."},"agent_id":{"type":"string","description":"可选：向下发话时=子代理 sa_ 引用（自己的子树内）；缺省=向父代理发中期回报。"}},"required":["message"]}`)
}

func (d *dualSendMessageTool) ReadOnly() bool { return false }

func (d *dualSendMessageTool) CompactDescription() string {
	return "发消息(按参数定方向):缺 agent_id=向父代理中期回报(下一步界读到,回执非答案);带 agent_id=向自己子树的子代理发话(运行中=steer/排队=起跑送达/终态=续聊)"
}

func (d *dualSendMessageTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"},"agent_id":{"type":"string"}},"required":["message"]}`)
}

func (d *dualSendMessageTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var probe struct {
		AgentID string `json:"agent_id"`
	}
	_ = json.Unmarshal(args, &probe)
	if probe.AgentID != "" && d.down != nil {
		return d.down.Execute(ctx, args)
	}
	return d.up.Execute(ctx, args)
}
