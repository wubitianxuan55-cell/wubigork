package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/jobs"
)

// InterruptTool — interrupt_agent（2026-10，dsh tool-subagent-control 蒸馏
// P0-A）：模型面中断。语义对齐上游：fire-and-return——取消信号发出即返回，
// 目标在其下一个取消点（LLM 采样间/工具边界）停止；返回的是「已受理中断」
// 而非目标的最终状态（task 结果/后台完成通知会如实报告被打断）。
//
// 两类目标：① sa_… 子代理引用（前台运行中或排队等待并发槽位，经 subRuns
// 句柄取消；排队中取消=等槽放弃+侧车 failed）；② 后台任务 id（转发
// jobs.Manager.Kill，含后代级联）。已收跑/未知 id 如实报错——完成态没有
// 「打断」可言，子代理续跑走 task 的 continue_from。
//
// 授权面：邻接即授权（模型只见自己回合产出的 sa_/job 引用；ref 含时间戳+
// 随机段不可枚举），与 SteerSubagent（v4.243）同一暴露面、同一威胁模型。
// 嵌套委派（2026-11-18 dsh 完整智能体形态）开放后子代理可见此工具，但按
// 「自己的子树」收窄：只能中断自己的后代（AncestryContains 守卫），后台
// job 面不开放（job 无子树所有权，兄弟任务不可越权杀）——原「中断权只属
// 父代理」边界收窄为「中断权只及自己的后代」。
type InterruptTool struct {
	store *SubagentStore
}

// NewInterruptTool 装配 interrupt_agent（boot 注册点调用）。store 用于子代理
// 调用面的子树守卫（血缘上溯），主会话面不消费。
func NewInterruptTool(store *SubagentStore) *InterruptTool { return &InterruptTool{store: store} }

func (t *InterruptTool) Name() string { return "interrupt_agent" }

func (t *InterruptTool) Description() string {
	return "Ask a running or queued sub-agent (sa_… reference) or a background job to stop. " +
		"Fire-and-return: the cancel signal is issued immediately; the target stops at its next cancellation point, " +
		"and its task result (or the job's completion notice) reports the interruption. " +
		"Finishing or unknown ids are reported as errors — a finished sub-agent cannot be interrupted; continue its conversation with task's continue_from instead."
}

func (t *InterruptTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "ref":{"type":"string","description":"The sa_… sub-agent reference (from the task result's 'Subagent reference' line) or a background job id."}
},
"required":["ref"]
}`)
}

// ReadOnly 为 false：中断改变子代理/任务的执行态（终止），不参与只读并行
// 判定与折叠降噪。
func (t *InterruptTool) ReadOnly() bool { return false }

func (t *InterruptTool) CompactDescription() string {
	// CompactDescriptor 无条件替换 provider 面描述（tool.go Schemas）——模型
	// 只看这里，fire-and-return 语义必须写进这一行，不能只写 Description。
	return "中断在跑或排队中的子代理(sa_引用)或后台任务:信号即发即返,目标在下一个取消点停止,其结果会如实报告被打断;已完成/未知id报错(已完成请用continue_from续跑)"
}

func (t *InterruptTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"ref":{"type":"string"}},"required":["ref"]}`)
}

func (t *InterruptTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	ref := strings.TrimSpace(p.Ref)
	if ref == "" {
		return "", fmt.Errorf("ref is required")
	}
	// 子树守卫（2026-11-18 嵌套委派）：子代理调用只可中断自己子树内的 sa_
	// 运行；job 面整个不开放（后台任务无子树所有权概念）。
	if callerRef := SubagentRefFromContext(ctx); callerRef != "" {
		if !IsSubagentRef(ref) {
			return "", fmt.Errorf("background jobs can only be interrupted from the main session")
		}
		if t.store == nil || !t.store.AncestryContains(callerRef, ref) {
			return "", fmt.Errorf("refusing to interrupt %s: not a descendant of your own sub-agent tree (see subagent_list for what is yours)", ref)
		}
	}
	// ① 子代理（前台运行中/排队中）。
	if err := InterruptSubagent(ref); err == nil {
		return fmt.Sprintf("Interrupt requested for sub-agent %s. It stops at its next cancellation point; its task result will report the interruption.", ref), nil
	}
	// ② 后台任务（job id，Kill 含后代级联）。
	if jm, ok := jobs.FromContext(ctx); ok {
		if jm.Kill(ref) {
			return fmt.Sprintf("Interrupt requested for background job %s. It stops at its next cancellation point; its completion notice will report the stop.", ref), nil
		}
	}
	return "", fmt.Errorf("%s is not running (unknown or already finished)", ref)
}
