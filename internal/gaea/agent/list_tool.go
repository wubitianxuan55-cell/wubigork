package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SubagentListTool — subagent_list（2026-10，dsh list_agents 蒸馏 P1-D）：
// 模型面枚举当前会话的子代理运行（ref/状态/标题，新优先）。多轮之后父模型
// 要 steer / interrupt / continue_from 一个子代理时，ref 的唯一入口此前是
// task 结果尾行的 "Subagent reference"——本工具给持久枚举入口，三者闭环。
// 数据源 = SubagentStore meta 侧车（会话作用域由 lazy 目录闭包跟随），
// 损坏 meta 单条跳过（对齐 dsh corrupt 降级，不炸整表）。
//
// 授权/可见性与 interrupt_agent 同面：已入 subagentMetaTools——子代理不见
// 此工具，兄弟运行的清单不进子代理上下文。
type SubagentListTool struct {
	store *SubagentStore
}

// NewSubagentListTool 装配 subagent_list（boot 注册点调用，共享 lazy store）。
func NewSubagentListTool(store *SubagentStore) *SubagentListTool {
	return &SubagentListTool{store: store}
}

func (t *SubagentListTool) Name() string { return "subagent_list" }

func (t *SubagentListTool) Description() string {
	return "List this session's sub-agent runs (ref, status, title; newest first). Use it to find the sa_… reference you need for interrupt_agent (running/queued) or task continue_from (completed/failed)."
}

func (t *SubagentListTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
}

// ReadOnly：纯读 meta 侧车，无任何状态变更——折叠降噪（quiet 行）适用。
func (t *SubagentListTool) ReadOnly() bool { return true }

func (t *SubagentListTool) CompactDescription() string {
	// CompactDescriptor 无条件替换 provider 面描述（模型只看这里）。
	return "列出本会话的子代理运行(ref/状态/标题,新优先):为 interrupt_agent 或 task continue_from 找 sa_ 引用"
}

func (t *SubagentListTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
}

func (t *SubagentListTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	runs, err := t.store.ListRuns()
	if err != nil || len(runs) == 0 {
		// 目录未解析（会话路径未就绪）/为空/读失败：对模型都是同一句诚实
		// 空态——列不出东西时没有可纠错的动作，报错只会诱导无意义重试。
		return "No sub-agent runs in this session yet.", nil
	}
	var sb strings.Builder
	sb.WriteString("Sub-agent runs in this session (newest first):\n")
	for _, r := range runs {
		title := strings.TrimSpace(r.Title)
		if title == "" {
			title = "(untitled)"
		}
		line := fmt.Sprintf("- %s [%s] %s", r.Ref, r.Status, title)
		if !r.UpdatedAt.IsZero() {
			line += fmt.Sprintf(" (updated %s)", time.Since(r.UpdatedAt).Truncate(time.Second))
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	sb.WriteString("Interrupt a running/queued entry with interrupt_agent; resume a completed/failed one with task's continue_from.")
	return strings.TrimRight(sb.String(), "\n"), nil
}
