// Package event defines the typed event stream the agent emits as it runs a
// turn, and the Sink it emits to. It decouples "what happened" (the model
// produced reasoning, a tool was dispatched, a turn used N tokens) from "how to
// show it" (ANSI scrollback in a terminal, a card in a webview).
//
// The agent depends only on Sink; each frontend implements one. The chat TUI
// renders events to its scrollback; a headless run renders them to plain ANSI
// on stdout; a future GUI/serve transport forwards them to a webview or
// websocket. This replaces the old io.Writer contract, where the agent wrote
// pre-formatted ANSI and the consumer had to re-derive structure by matching
// line prefixes — fragile, and lossy for any frontend richer than a terminal.
package event

import "github.com/gaea/gaea/internal/gaea/provider"

// Kind tags an Event. Read the field(s) documented for that kind.
type Kind int

const (
	// TurnStarted marks the start of one top-level Run (one user turn). Sinks
	// reset any per-turn rendering state on it. Carries no payload.
	TurnStarted Kind = iota
	// Reasoning is a thinking-mode reasoning delta (Text). Streamed before the
	// visible answer; sinks typically render it muted under a "thinking" header.
	Reasoning
	// Text is an answer-text delta (Text).
	Text
	// Message marks the assistant turn's text as complete: Text holds the full
	// answer and Reasoning the full chain-of-thought (both already streamed via
	// the deltas above). A sink may use it to re-render the streamed raw text as
	// styled markdown; a plain sink can ignore it.
	Message
	// ToolDispatch announces a tool call is about to run (Tool: ID/Name/Args/ReadOnly).
	ToolDispatch
	// ToolResult reports a finished tool call (Tool: Output/Err/Truncated set).
	ToolResult
	// Usage carries per-turn token telemetry (Usage; Pricing optional, for cost).
	Usage
	// Notice is an out-of-band message — a warning, truncation, block, or
	// compaction notice (Level + Text).
	Notice
	// Phase marks a turn phase boundary (Text = label such as "deepseek · executing").
	Phase
	// ApprovalRequest asks the frontend to approve a pending tool call
	// (Approval: ID/Tool/Subject). The run blocks until the controller's
	// Approve(ID, …) resolves it; a frontend shows a prompt and answers.
	ApprovalRequest
	// AskRequest asks the frontend to put one or more structured multiple-choice
	// questions to the user (Ask: ID + Questions). The run blocks until the
	// controller's AnswerQuestion(ID, …) resolves it. Powers the `ask` tool.
	AskRequest
	// TurnDone marks the end of one top-level Run (Err non-nil on failure;
	// nil also for a user cancellation, which is not an error). Always the
	// last event of a turn.
	TurnDone
	// CompactionStarted marks the start of a context-compaction pass (Compaction
	// payload: Trigger). A frontend shows a "compacting…" placeholder while the
	// summarizer runs; CompactionDone replaces it. Mirrors ToolDispatch/ToolResult.
	CompactionStarted
	// CompactionDone reports a finished compaction pass (Compaction payload:
	// Trigger/Messages/Summary/Archive). An aborted pass emits this with an empty
	// Summary so the placeholder still resolves. Replaces the older plain Notice
	// so a sink can render a distinct, expandable card.
	CompactionDone
	// RequestHeader marks one model request's assembled header: the system
	// prompt and tool schemas that were actually sent. Emitted by the agent
	// right before the provider call, so the context-view fold can show the
	// system/tools composition of every request — and satisfies the
	// "model-visible inputs are logged" invariant.
	RequestHeader
	// Retrying marks a stream-recovery retry attempt.
	// Steer is a mid-turn user guidance message injected while the agent is running.
	Steer
	Retrying
	// SubagentMessage 是子代理完成回投事件（v4.26 对话流式重造，对标 Codex
	// 2026-08 "Report completed sub-agent activity on parent turns"）：子代理
	// （task/run_skill 等元工具派生）运行期间父回合只挂一张 task 卡，期间静默；
	// 本事件在子代理完成后把其最终答复文本回投到父 sink，让主聊天在子代理
	// 长跑结束时立即可见产出。只回投完成态（err==nil 且文本非空），中途进度
	// 不回投——避免并行/重试子代理刷屏。Text=最终答复全文；SubagentRef=子代理
	// transcript 引用（"sa_..."，临时子代理为空）；ParentToolID=父 task 调用 ID
	// （由 subSinkFor 打点），前端据此把答复挂到对应 task 卡片下。
	SubagentMessage
	// SubagentText 是持久化子代理运行中的助手文本增量（v4.62 P1 逐 token
	// 流式，对标 Codex live view）：subSinkFor 把内层 Text 增量转标为该 Kind
	// 透传父 sink，让 SubagentThread 会话 tab 在子代理运行中实时看到「正在
	// 打出的字」，而不是等 ~1s 快照/3s 轮询。Text=增量块；SubagentRef="sa_..."
	// （前端据此路由到对应会话 tab；为空=临时子代理无消费方，直接丢弃）；
	// ParentToolID=父 task 调用 ID。**wire-only**：EventLogSink 有意不把该
	// Kind 落主会话日志——增量全文已由 SubagentMessage 收尾 + 子代理自身
	// transcript 落盘承载，逐块落主日志只会膨胀体积、污染重放。
	SubagentText
)

// Level classifies a Notice so sinks can style or filter it.
type Level int

const (
	LevelInfo Level = iota
	LevelWarn
)

// UsageSource constants tag the origin of a Usage event so the frontend can
// split stats between the main agent and sub-agents.
const (
	UsageSourceMain     = "main"
	UsageSourceSubagent = "subagent"
	UsageSourceExecutor = "executor" // 单模型执行阶段
)

// Tool describes a tool call for ToolDispatch / ToolResult events. On dispatch
// only ID/Name/Args/ReadOnly are set; on result Output/Err/Truncated are filled
// in. Args is the raw JSON arguments — a sink compacts it for display.
type Tool struct {
	ID     string
	Name   string
	Args   string
	Output string // ToolResult: the result text fed to the model
	Err    string // ToolResult: non-empty when the call failed or was blocked
	// Recoverable is true when the error is one the agent can fix on the next
	// turn (bad arguments, wrong file path, command exit code) — not a genuine
	// system fault (unknown tool, permission block, panic). Frontends render
	// recoverable errors muted (strikethrough, no red) so the user isn't alarmed.
	Recoverable bool
	ReadOnly    bool
	Truncated   bool // ToolResult: Output was head+tailed before display/model
	// Partial marks an early ToolDispatch emitted when a call begins (ID/Name set,
	// Args still streaming) so a frontend can show the card immediately; a second,
	// full ToolDispatch (Partial false, Args set) follows when the call completes.
	Partial bool
	// ParentID, when set, is the ID of the tool call that spawned this one — a
	// sub-agent's calls carry the parent `task` call's ID so a frontend can nest
	// them under it. Empty for top-level calls.
	ParentID string
}

// Approval identifies a pending tool-call approval for an ApprovalRequest
// event. ID correlates the request with the controller's Approve(ID, …) reply.
type Approval struct {
	ID      string
	Tool    string
	Subject string
	// Reason 是模型随权限升级申请给出的申请理由（request_permission 工具，
	// Request=true 时携带）。普通工具闸门审批恒为空。前端必须原样展示，
	// 不得在未展示的情况下采信（审批纪律对 reason 同样适用）。
	Reason string
	// Request 标记这条审批是模型主动发起的「权限升级申请」（request_permission
	// 工具），而非真实工具调用前的常规审批。前端据此换标题/说明并展示 Reason；
	// 决策仍走同一 Approve(ID, decision) 通道与五决策语义。
	Request bool
}

// AskOption is one choice the user can pick for an AskQuestion.
type AskOption struct {
	Label       string
	Description string // optional one-line explanation shown under the label
}

// AskQuestion is one structured question the `ask` tool puts to the user.
type AskQuestion struct {
	ID      string // stable per-question id, so answers correlate back
	Header  string // short label (the tab title)
	Prompt  string // the question text
	Options []AskOption
	Multi   bool   // allow selecting more than one option
}

// Ask carries an AskRequest: a batch of questions and the ID that correlates the
// controller's AnswerQuestion(ID, …) reply.
type Ask struct {
	ID        string
	Questions []AskQuestion
}

// Compaction carries a context-compaction pass for the CompactionStarted /
// CompactionDone events. On CompactionStarted only Trigger is set. On
// CompactionDone, Messages/Summary/Archive are filled in (an aborted pass leaves
// Summary empty). Trigger is "auto" (the prompt reached the window threshold) or
// "manual" (the user ran /compact).
type Compaction struct {
	Trigger  string // "auto" | "manual"
	Messages int    // Done: how many messages were folded into the summary
	Summary  string // Done: the briefing the agent keeps relying on
	Archive  string // Done: path the dropped originals were archived to ("" if none)
	Quality  string // V3.2: post-hoc quality assessment (human-readable one-liner)
}

// AskAnswer is the user's reply to one AskQuestion: the chosen option label(s)
// (a free-typed answer is carried as a single Selected entry).
type AskAnswer struct {
	QuestionID string
	Selected   []string
}

// Event is one increment in a turn's event stream. Read the field(s) documented
// for Kind; the others are zero.
type Event struct {
	Kind      Kind
	Text      string            // Reasoning / Text / Message / Notice / Phase
	Reasoning string            // Message: the full reasoning chain
	Tool      Tool              // ToolDispatch / ToolResult
	Usage     *provider.Usage   // Usage
	Pricing   *provider.Pricing // Usage: for cost display (nil = omit cost)
	// UsageSource tags the origin of a Usage event so the frontend can
	// split stats between the main agent and sub-agents. "main" for the
	// parent agent's own API calls; "subagent" for spawned tasks.
	UsageSource string
	// SessionHit/SessionMiss carry cumulative cache tokens across the whole
	// session (Usage events only), so a frontend can show the aggregate hit-rate
	// — which doesn't crater on a short turn or after compaction — alongside
	// Usage's single-turn numbers.
	SessionHit  int        // Usage: cumulative cache-hit prompt tokens this session
	SessionMiss int        // Usage: cumulative cache-miss prompt tokens this session
	Turn        int        // Usage: 会话 API 调用轮次（从 1 开始，由 AgentRunner 维护）
	Level       Level      // Notice
	Approval    Approval   // ApprovalRequest
	Ask         Ask        // AskRequest
	Err         error      // TurnDone: non-nil on failure
	Compaction  Compaction // Compaction
	// Header carries the assembled request header (RequestHeader events):
	// the system prompt text and the tool schemas sent to the provider.
	Header RequestHeaderInfo
	// RetryAttempt / RetryMax track stream-recovery progress (Retrying events).
	RetryAttempt int // Retrying: current attempt number (1-based)
	RetryMax     int // Retrying: maximum allowed attempts
	// SubagentRef / ParentToolID carry sub-agent completion payload (v4.26
	// SubagentMessage events): Ref is the sub-agent transcript reference
	// ("sa_...", empty for ephemeral sub-agents); ParentToolID is the parent
	// task tool-call ID the sub-agent runs under (stamped by subSinkFor).
	SubagentRef  string // SubagentMessage: 子代理 transcript 引用
	ParentToolID string // SubagentMessage: 父 task 调用 ID
}

// RequestHeaderInfo is the assembled header of one model request.
// System is the concatenated system-role prompt (L1 + L2); Tools are the
// schemas actually sent (after per-agent filtering); Window is the model's
// context-window size in tokens (0 = unknown).
type RequestHeaderInfo struct {
	System string           `json:"system,omitempty"`
	Tools  []ToolSchemaInfo `json:"tools,omitempty"`
	Window int              `json:"window,omitempty"`
}

// ToolSchemaInfo is one tool schema as sent to the provider. Schema is the
// JSON-serialized schema (name + description + parameters), kept so the
// context browser can show the exact definition the model saw.
type ToolSchemaInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

// Sink consumes a turn's events. The agent calls Emit serially from its run
// loop (tool execution may fan out across goroutines, but emission does not),
// so an implementation need not be safe for concurrent Emit. Emit must not
// block indefinitely — a channel-backed sink should be buffered or drained by
// a live reader.
type Sink interface {
	Emit(Event)
}

// FuncSink adapts a plain function to a Sink.
type FuncSink func(Event)

// Emit calls the wrapped function.
func (f FuncSink) Emit(e Event) {
	if f != nil {
		f(e)
	}
}

// Discard is a Sink that drops every event. Useful in tests and for runs that
// only care about the final session state.
var Discard Sink = FuncSink(func(Event) {})
