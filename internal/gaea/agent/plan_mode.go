package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// plan_mode.go — 计划模式审批门（dsh plan-mode 蒸馏，取道不取器；规格
// 进度计划/gaea-plan-mode-20260926.md）。计划模式下模型只研究并产出计划，
// exit_plan_mode 把计划提交用户审批：批准=退出计划模式开始执行，继续计划=
// 带着反馈修订后再次提交。与上游的两处关键偏离：
//  1. 系统前缀冻结（V10.36）——政策/叙事走 user 消息注入，不动 system 段；
//  2. 状态 v1 内存态（runner 字段），事件日志持久化+恢复折叠留二刀。
// 沙箱与权限政策独立：计划模式是提示+审批流，不做硬工具门（与上游同判）。

// PlanPolicyNotice 计划模式开启时注入的 user 消息（政策随历史走，前缀不动）。
const PlanPolicyNotice = "[plan mode] 用户已将本会话切换到计划模式：从现在起你只做研究和方案设计——" +
	"读文件、检索、提问、分析，不执行任何写入或变更类操作（编辑、写文件、执行带副作用的命令等）。" +
	"研究完成后，把完整方案整理成 markdown 计划（以 # 标题开头命名），调用 exit_plan_mode 提交用户审批；" +
	"用户批准前不要开始执行。用户的消息仍会照常到达，但都按「完善计划」对待。"

// PlanExitByUserNotice 用户手动 /plan off 时注入的叙事。
const PlanExitByUserNotice = "[plan mode] 用户已将本会话切回默认模式：计划模式结束，按正常方式执行任务。"

// planApprovedNotice 批准后注入的叙事（工具结果同文告知，历史里留痕）。
const planApprovedNotice = "[plan mode] 用户已批准计划：退出计划模式，从你的下一步开始执行该计划。"

// PlanGate 计划模式状态闸：ExitPlanModeTool 经 callContext 读取（与 asker 同
// 盖章纪律），由 AgentRunner 实现。窄接口避免把整个 runner 盖进每个工具调用。
type PlanGate interface {
	PlanMode() bool
	ApprovePlan() // 翻转 off + 注入批准叙事（用户已批准，落历史）
}

type planGateKey struct{}

// WithPlanGate 把计划模式闸盖章进工具 ctx（executeOne 逐调用盖章；nil 不盖章，
// headless/子代理保持「闸不可用」语义）。
func WithPlanGate(ctx context.Context, g PlanGate) context.Context {
	if g == nil {
		return ctx
	}
	return context.WithValue(ctx, planGateKey{}, g)
}

// PlanGateFromContext 读回计划模式闸。
func PlanGateFromContext(ctx context.Context) (PlanGate, bool) {
	g, ok := ctx.Value(planGateKey{}).(PlanGate)
	return g, ok
}

// ExitPlanModeTool 计划审批门工具。恒注册（目录跨模式切换稳定，与上游同判）；
// 非计划模式调用如实报错。审批走 ask 通道（前端泛型选项卡渲染，零新绑定）。
type ExitPlanModeTool struct{}

func NewExitPlanModeTool() *ExitPlanModeTool { return &ExitPlanModeTool{} }

func (*ExitPlanModeTool) Name() string { return "exit_plan_mode" }

// ReadOnly 恒真：工具自身只做审批问询不做变更（执行发生在批准后的后续回合，
// 走各工具自身的权限闸）。
func (*ExitPlanModeTool) ReadOnly() bool { return true }

func (*ExitPlanModeTool) Description() string {
	return "Use only in plan mode. Present your complete plan for the user's review and, on approval, leave plan mode. " +
		"The plan must be markdown starting with a # heading that names it. " +
		"The user may approve (the plan is carried out from your next step) or keep planning — " +
		"revise and present again."
}

func (*ExitPlanModeTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "plan":{
    "type":"string",
    "description":"完整计划，markdown，以 # 标题开头命名计划。"
  }
},
"required":["plan"]
}`)
}

func (*ExitPlanModeTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Plan string `json:"plan"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("exit_plan_mode: 解析参数失败: %w", err)
	}
	plan := strings.TrimSpace(p.Plan)
	if !strings.HasPrefix(plan, "# ") && !headedByAnyLevel(plan) {
		return "", fmt.Errorf("exit_plan_mode: 计划必须是非空 markdown 且以 # 标题开头（命名该计划）")
	}
	gate, ok := PlanGateFromContext(ctx)
	if !ok || gate == nil {
		return "", fmt.Errorf("exit_plan_mode: 计划模式闸不可用（headless/子代理上下文）")
	}
	if !gate.PlanMode() {
		return "", fmt.Errorf("exit_plan_mode 仅在计划模式可用（用户可用 /plan on 开启）")
	}
	_, _, asker, ok := CallContext(ctx)
	if !ok || asker == nil {
		// 与 ask 的 Never-Ask 语义刻意不同：执行闸必须人来开，headless 不自批。
		return "", fmt.Errorf("exit_plan_mode: 无交互用户可审批（headless）；保持计划模式并停止，等待用户输入")
	}
	const reviewID = "plan-review"
	answers, err := asker.Ask(ctx, []event.AskQuestion{{
		ID:     reviewID,
		Header: "计划审批",
		Prompt: "批准该计划并退出计划模式？",
		Options: []event.AskOption{
			{Label: "批准", Description: "退出计划模式，从下一步开始执行该计划"},
			{Label: "继续计划", Description: "留在计划模式；把你的反馈作为下一条消息发给模型"},
		},
	}})
	if err != nil {
		return "", fmt.Errorf("exit_plan_mode: 审批通道失败: %w", err)
	}
	approved := false
	for _, a := range answers {
		if a.QuestionID == reviewID && len(a.Selected) == 1 && a.Selected[0] == "批准" {
			approved = true
		}
	}
	if !approved {
		// 无自由文本槽（gaea ask 通道 v1）：反馈由用户下一条消息携带。
		return "", fmt.Errorf("用户选择继续计划：请按既有讨论修订计划后再次提交；用户的反馈会作为下一条消息到达")
	}
	gate.ApprovePlan()
	return "计划已批准——已退出计划模式；从你的下一步开始执行该计划。", nil
}

// headedByAnyLevel 首个非空行是否 markdown 标题（#~###### 任意级；上游
// firstHeading 同口径的入口校验面，主路径仍鼓励 # 一级标题）。
func headedByAnyLevel(plan string) bool {
	for _, line := range strings.Split(plan, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		level := 0
		for level < len(trimmed) && trimmed[level] == '#' {
			level++
		}
		return level >= 1 && level <= 6 && level < len(trimmed) && trimmed[level] == ' '
	}
	return false
}

// SetPlanMode 翻转计划模式（v1 内存态；事件日志持久化留二刀）。on=true 时注入
// 政策 user 消息；off 由调用方区分「用户手动」与「批准退出」（两条叙事文本），
// 故这里不做注入。
func (a *AgentRunner) SetPlanMode(on bool) {
	a.planMu.Lock()
	a.planMode = on
	a.planMu.Unlock()
}

// PlanMode 读当前计划模式状态。
func (a *AgentRunner) PlanMode() bool {
	a.planMu.Lock()
	defer a.planMu.Unlock()
	return a.planMode
}

// ApprovePlan 批准出口：翻转 off + 注入批准叙事（PlanGate 实现）。
func (a *AgentRunner) ApprovePlan() {
	a.SetPlanMode(false)
	a.AppendUserMessage(planApprovedNotice)
}

// AppendUserMessage 在会话末尾追加一条 user 消息（计划模式政策/叙事注入用）。
// turn 间执行；不触碰系统前缀（缓存纪律），随下一 checkpoint/正常落盘通道持久化。
func (a *AgentRunner) AppendUserMessage(content string) {
	a.sessMu.Lock()
	s := a.session
	a.sessMu.Unlock()
	if s == nil {
		return
	}
	s.Add(provider.Message{Role: provider.RoleUser, Content: content})
}
