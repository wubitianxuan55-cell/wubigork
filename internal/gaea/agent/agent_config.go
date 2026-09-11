package agent

import (
	"github.com/gaea/gaea/internal/gaea/context"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// Options configures an AgentRunner.
type Options struct {
	MaxSteps    int
	Temperature float64
	Pricing     *provider.Pricing // optional, for per-turn cost display

	// Gate is the per-call permission gate. nil disables gating.
	Gate Gate

	// Hooks fires PreToolUse / PostToolUse shell hooks around tool calls. nil
	// disables hook firing.
	Hooks ToolHooks

	// Jobs is the session's background-job manager (nil disables background tools).
	Jobs *jobs.Manager

	// Context management. ContextWindow <= 0 disables compaction.
	ContextWindow int
	// Compaction groups compaction settings (V3.0).
	Compaction CompactionConfig
	// Dispatcher is the centralized pre-execution check pipeline (V2.4).
	// nil means the agent uses inline checks (backward compatible).
	Dispatcher *ToolDispatcher
	// CtxMgr is the TCCA context kernel (V3.0). When set, the agent uses it
	// for prompt assembly and tool filtering instead of inline logic.
	CtxMgr *context.ContextManager
	// AuditFunc, when non-nil, is called after every tool execution with a
	// summary of the call. V3.2: foundational audit trail.
	AuditFunc func(tool string, taskKind string, readOnly bool, outcome string, errMsg string, outputLen int, durationMs int64)

	// ParamStorm enables parameter-level duplicate tool call detection (V5.13).
	// nil disables; non-nil provides WindowSize/Threshold/ExemptTools.
	ParamStorm *ParamStormOptions
	// BudgetLimit is the per-session cost budget in yuan (V5.15).
	// <=0 means unlimited. When set, the agent tracks cumulative cost
	// and warns at 80% / blocks at 100%.
	BudgetLimit float64

	// ModelProfile overrides compaction thresholds for specific models (V5.17).
	// nil means use defaults from CompactionConfig.
	ModelProfile *ModelProfile

	// TemplatePrefix is the sub-agent template prefix injected before the
	// user message in spawned agents. Same-class sub-agents share the same
	// template bytes — DeepSeek prefix cache hits across sub-agent invocations.
	TemplatePrefix string
	// ActiveSchemas are the filtered tool schemas for sub-agents (V5.30).
	// When set, RunSubAgent uses these as the tools JSON field so the
	// prefix cache includes the same tools the parent sends.
	ActiveSchemas []provider.ToolSchema
	RuntimePrompt string
	// Goal is the session-level stopping condition (V6.0 P7). When non-empty,
	// the stop gate checks whether the model's final answer satisfies the goal.
	Goal string
	// DisableVerify suppresses the orchestrate verify nudge (V10.22).
	// Sub-agents set this to true so the verify gate doesn't inject
	// "[system] All tasks complete" into their fresh session.
	DisableVerify bool

	// JournalDir 是 v4.1 证据链 Journal 目录（<cwd>/.gaea/work/journal）。
	// 空 = 关闭证据链（旧形态/测试缺省）。回合结束把变更证据卡写入
	// JSONL + markdown 投影（play 回合不落盘，红线）。
	JournalDir string

	// SessionID 是本执行器在证据卡上的会话标识（v4.221 子代理证据落账）：
	// flushJournal 把它盖进每张卡的 SessionID（Journal 按会话分文件）。
	// 主执行器经 SetArchive 注入；子代理由 runSubSession 传 run.Ref（sa_…）——
	// DAG 节点产物据此按会话精确归因（设计 docs/gaea-office-dag-63-design-2026-09.md §3）。
	// 空 = 证据卡 SessionID 留空（旧行为）。
	SessionID string

	// FinalizeText 是最终文本定稿闸（v4.211 记忆 5.1 出口判据「[MEM:] 引用
	// 在回复发出前全部解析到节点」）：每次模型轮次收尾（Message 全文事件发出
	// 前 + 文本进 session/摘要前）对答案文本做一次改写。装配点注入（boot 按
	// 记忆开关注入悬空 [MEM:] 键剥离闭包）；nil = 不改写（子代理/测试缺省，
	// 行为与历史逐字节一致）。改写只作用于收尾全文——流式增量原样透传，前端
	// 收到 Message 事件整泡替换，不新增事件形态。
	FinalizeText func(text string) string
}

// StormBreaker tracks repeated failures to detect death spirals (V3.0 Phase 4).
type StormBreaker struct {
	Sig   string // per-turn fixation signature
	Count int    // consecutive identical failures
}

// Agent is a backward-compatible alias for AgentRunner.
type Agent = AgentRunner

// fallbackTokPerChar is ~4 chars per token — the middle-of-the-road estimate
// used before any provider usage data is available to calibrate.
const fallbackTokPerChar = 0.25

// MidTurnSteerPrefix marks user messages that were injected mid-turn as
// guidance (via Steer). The model sees them as instructions; frontends
// display them as a notice, not a regular user bubble.
// (Design adopted from DeepSeek-Reasonix-V1.12)
const MidTurnSteerPrefix = "[Mid-turn steer queued by the user. Do not treat this as a new task; use it only as additional guidance for the current task after completing the current step.]"
