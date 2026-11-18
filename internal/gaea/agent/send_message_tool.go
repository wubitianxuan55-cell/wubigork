// send_message_tool.go — send_message（2026-11-18 DSH 式通用智能体工具集刀2，
// 对照上游 tool-subagent-control 的 send_message，backend.md §7.1）：父/子→
// 子级**统一传话通道**，按目标状态自动路由（上游语义「delivery 由目标态决
// 定」的 gaea 折算）：
//
//	running → SteerSubagent（插最近一步，用户插话同队列）
//	queued  → pending_messages 排队落盘（MarkRunning 起跑 flush 逐条 steer 送达）
//	终态    → RunFollowUp（全量 transcript 续聊——上游 coldResume 的 gaea 形）
//
// read_subagent_result 的模型面同批退役：子代理结果自动回投（前台=工具结果；
// 后台=SubagentMessage+job 完成注记唤醒），「读结果」不再是模型动作——主动
// 追问/补充指引一律走本工具。子代理调用面按「自己的子树」收窄
// （AncestryContains，与 interrupt_agent/subagent_list 同款）。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// SendMessageTool — send_message（followUp 用既有 SubagentFollowUpRunner
// 形状——boot.go OnFollowUpReady 同款闭包，走 FollowUpSink 专用流式通道）。
type SendMessageTool struct {
	store    *SubagentStore
	followUp SubagentFollowUpRunner
}

// NewSendMessageTool 装配 send_message（boot 注册点调用）：store 做子树守卫
// 与状态分诊；followUp 是终态续聊执行器（boot 复用用户追问闭包；nil = 不支
// 持续聊分支，终态目标报错可执行文案）。
func NewSendMessageTool(store *SubagentStore, followUp SubagentFollowUpRunner) *SendMessageTool {
	return &SendMessageTool{store: store, followUp: followUp}
}

func (t *SendMessageTool) Name() string { return "send_message" }

func (t *SendMessageTool) Description() string {
	return "Send a message to a sub-agent (sa_… reference). Delivery is automatic by target state: running = injected as steering at its next step; queued = held and delivered when it starts; finished = starts a follow-up turn continuing that sub-agent's full transcript. Use it to steer running work, add context, or ask follow-ups — results are delivered to you automatically when a sub-agent finishes."
}

func (t *SendMessageTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "agent_id":{"type":"string","description":"The sa_… sub-agent reference (from the task result's 'Subagent reference' line or subagent_list)."},
  "message":{"type":"string","description":"What to deliver: steering guidance, added context, or a follow-up request."}
},
"required":["agent_id","message"]
}`)
}

// ReadOnly 为 false：投递改变目标的执行态/上下文，不参与只读并行判定。
func (t *SendMessageTool) ReadOnly() bool { return false }

func (t *SendMessageTool) CompactDescription() string {
	return "给子智能体传话(按状态自动路由:运行中=插话;排队中=起跑送达;已完成=续聊新回合)——结果会自动回投,无需主动读取"
}

func (t *SendMessageTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"agent_id":{"type":"string"},"message":{"type":"string"}},"required":["agent_id","message"]}`)
}

func (t *SendMessageTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.ExecuteWithContext(ctx, tool.ToolContext{}, args)
}

// ExecuteWithContext 满足 tool.ContextualTool（与 interrupt_agent 同面）。
func (t *SendMessageTool) ExecuteWithContext(ctx context.Context, tc tool.ToolContext, args json.RawMessage) (string, error) {
	var p struct {
		AgentID string `json:"agent_id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	ref := strings.TrimSpace(p.AgentID)
	msg := strings.TrimSpace(p.Message)
	if ref == "" || msg == "" {
		return "", fmt.Errorf("agent_id and message are required")
	}
	// 子树守卫：子代理调用面只能给自己子树内的后代传话（主会话调用方
	// callerSessionID 为空，不做血缘限制——邻接即授权，与 interrupt_agent
	// 同一暴露面/威胁模型）。
	if caller := SubagentRefFromContext(ctx); caller != "" && !t.store.AncestryContains(caller, ref) {
		return "", fmt.Errorf("subagent %s is outside your own sub-agent tree", ref)
	}
	status, err := t.store.PeekStatus(ref)
	if err != nil {
		return "", fmt.Errorf("subagent %s not found in this session's store", ref)
	}
	switch status {
	case SubagentRunning:
		if err := SteerSubagent(ref, msg); err != nil {
			// Peek/直穿竞态窗（刚起跑句柄未登记等）：如实上抛，调用方可重试。
			return "", err
		}
		return fmt.Sprintf("delivered to running subagent %s as steering", ref), nil
	case SubagentQueued:
		if err := t.store.AppendPending(ref, msg); err != nil {
			return "", err
		}
		return fmt.Sprintf("queued for subagent %s; it will be delivered when the sub-agent starts", ref), nil
	default:
		// 终态（completed/failed/aborted/interrupted）：续聊新回合。
		if t.followUp == nil {
			return "", fmt.Errorf("subagent %s is finished and follow-up delivery is unavailable in this session", ref)
		}
		if err := t.followUp(ctx, ref, msg); err != nil {
			return "", err
		}
		return fmt.Sprintf("follow-up started for subagent %s; its answer will be delivered when it finishes", ref), nil
	}
}
