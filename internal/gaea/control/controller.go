// Package control is the transport-agnostic session driver. A Controller owns
// the agent run loop and session lifecycle, takes commands (Send/Cancel/Approve/
// Compact/NewSession/…), and emits everything that happens —
// reasoning, tool calls, approvals, turn completion — as a typed event stream to
// a single event.Sink.
//
// The point is one orchestration layer behind every frontend: a terminal TUI, a
// desktop webview, or an HTTP/SSE server each drive the Controller identically
// (issue commands, render events) and none of them re-implement turn lifecycle,
// cancellation, or approval. The Controller depends on no frontend.
package control

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/billing"
	"github.com/gaea/gaea/internal/gaea/command"
	tiancontext "github.com/gaea/gaea/internal/gaea/context"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/hook"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/nilutil"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/plugin"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// Controller drives one chat session. Construct with New; drive with the command
// methods; observe through the Sink passed in Options.
type Controller struct {
	runner   agent.Runner
	executor *agent.Agent
	sink     event.Sink
	policy   permission.Policy

	label        string
	systemPrompt string
	sessionDir   string
	// logFormat 是会话持久化格式（3.0 Step 1 回退开关）："legacy"/""=旧行为，
	// "event"=事件日志模式（Snapshot 双写、回合前落用户消息 + flush 检查点、
	// Resume 走 Restore）。由 Options.LogFormat / SetLogFormat 注入。
	logFormat     string
	host          *plugin.Host
	commands      []command.Command
	skills        []skill.Skill
	hooks         *hook.Runner // session hook runner; nil-safe (no hooks configured)
	mem           *memory.Set
	memoryEnabled bool // false 时跳过逐轮记忆上下文注入（记忆开关）
	cleanup       func()
	startedOnce   bool // guards the one-shot SessionStart hook on first turn

	// balanceURL/balanceKey target the active provider's optional wallet-balance
	// endpoint (empty when the provider declares none). balanceKind selects the
	// balance-backend registry kind ("" = historical default "deepseek"). Captured
	// at build so a model/key switch — which rebuilds the controller — refreshes them.
	balanceURL  string
	balanceKey  string
	balanceKind string

	// jobs is the session-scoped background-job manager. The agent's background
	// tools spawn into it; Compose drains its completion notes into the next turn;
	// Close cancels its still-running jobs.
	jobs *jobs.Manager

	// reg is the live tool registry the executor reads each turn; pluginCtx is the
	// session-scoped context a hot-added stdio server binds its subprocess to.
	// Together they let AddMCPServer connect a server mid-session and have its tools
	// available on the next turn (see AddMCPServer / RemoveMCPServer).
	reg       *tool.Registry
	pluginCtx context.Context

	ctxMgr *tiancontext.ContextManager // V3.0 Phase 5

	// promptMu serialises approval prompts so at most one is outstanding at a
	// time (parallel read-only tool calls don't normally gate, writers run
	// serially — but this keeps the contract explicit). Held across the blocking
	// wait, so it must never be taken by the Approve command path.
	promptMu sync.Mutex

	// mu guards the run state and approval bookkeeping; every critical section
	// under it is short and non-blocking.
	mu          sync.Mutex
	cancel      context.CancelFunc
	running     bool
	sessionPath string
	approvals   map[string]chan approvalReply
	asks        map[string]chan []event.AskAnswer
	granted     map[string]bool
	nextID      int
	turn        int
	autoApprove bool
	// approvalTimeout 是审批等待超时（C4 TimedOut）：>0 时审批请求等待超过该
	// 时长按拒绝处理并发通知（回合继续，不静默放行）；0 = 不超时（默认等待）。
	approvalTimeout time.Duration

	// permLevel controls permission strictness: "ask" (prompt before writes, default),
	// "auto" (allow writes without asking), or "yolo" (skip all prompts).
	permLevel string

	// pendingMemory holds memory notes added mid-session (via "#" quick-add or a
	// memory edit) that haven't yet been folded into a turn. Compose drains it
	// onto the next outgoing turn — never into the cache-stable system prefix — so
	// a fresh memory takes effect this session without busting the prompt cache;
	// it joins the prefix naturally on the next session.
	pendingMemory []string
	// sessionFacts holds temporary memories the model saved with session=true.
	// They persist across turns and can be promoted via PromoteSessionFacts().
	sessionFacts []memory.Memory
	// goal is set via /goal — the stopping condition for the session.
	goal string

	// pendingSends 是回合进行中排队等待的用户消息（T6-2.5 Send 排队），
	// turnLoop 在每个回合结束后按序排空。由 mu 保护。
	pendingSends []sendRequest
}

type approvalReply struct {
	allow   bool
	session bool
	// abort 拒绝并终止本轮（蒸馏 codex ReviewDecision::Abort）：本次调用按
	// 拒绝处理，同时取消当前回合——与「拒绝但继续」（allow=false）区分。
	abort bool
}

// sendRequest 是一条排队等待执行的消息：回合进行中收到 Send 时入队，
// 当前回合结束后按序执行（T6-2.5 Send 排队）。
type sendRequest struct {
	input string
	raw   string
}

// sendQueueLimit 是 Send 等待队列上限（条）：回合进行中最多排队
// sendQueueLimit 条用户消息，队满时新消息被拒绝并发出明确错误 notice。
const sendQueueLimit = 8

// Options carries the already-built pieces setup assembles. Lifecycle metadata
// lets the controller mint and rotate session files; Host/Commands are surfaced
// to frontends that resolve MCP prompts and slash commands.
type Options struct {
	// ApprovalTimeout 是审批等待超时（C4 TimedOut，蒸馏 codex ReviewDecision::TimedOut）：
	// >0 时无人响应的审批请求在超时后按拒绝处理（回合继续）并发 Notice；0 = 永久等待。
	ApprovalTimeout time.Duration
	Runner          agent.Runner
	Executor        *agent.Agent
	Sink            event.Sink
	Policy          permission.Policy
	Label           string
	SystemPrompt    string
	SessionDir      string
	SessionPath     string
	// LogFormat 是会话持久化格式（"legacy"/""=旧行为，"event"=事件日志）。
	// 事件日志模式下：Snapshot 双写（legacy 镜像+日志）、回合开始前落用户
	// 消息并 flush 检查点（fail-closed）、Resume 走 Restore（checkpoint+tail）。
	LogFormat string
	Host      *plugin.Host
	Commands  []command.Command
	Skills    []skill.Skill
	Hooks     *hook.Runner
	Memory    *memory.Set
	Cleanup   func()
	// BalanceURL/BalanceKey wire the active provider's optional wallet-balance
	// endpoint and bearer key; empty when the provider declares no balance_url.
	BalanceURL string
	BalanceKey string
	// BalanceKind selects the balance-backend registry kind (billing package).
	// Empty = historical default "deepseek" shape; unknown kinds fail closed.
	// 3.0 Wave 4：从 ProviderEntry 贯通，切换余额后端只改配置。
	BalanceKind string
	// Jobs is the session-scoped background-job manager (nil disables background jobs).
	Jobs *jobs.Manager
	// Registry is the executor's live tool set, and PluginCtx the session-scoped
	// context; both are needed for hot-adding MCP servers via AddMCPServer.
	Registry  *tool.Registry
	PluginCtx context.Context
	CtxMgr    *tiancontext.ContextManager // V3.0 Phase 5
	// MemoryDisabled 关闭自动记忆注入（系统提示词画像 + 逐轮记忆上下文）。
	// 零值（false）= 记忆开启；文档记忆文件不受影响，重新开启即恢复。
	MemoryDisabled bool
	// no confinement). Frontends pass the cwd they launched the session in.
	WorkspaceRoot string
}

// New builds a Controller. A nil Sink is replaced with event.Discard.
func New(opts Options) *Controller {
	sink := opts.Sink
	if nilutil.IsNil(sink) {
		sink = event.Discard
	}
	pluginCtx := opts.PluginCtx
	if pluginCtx == nil {
		pluginCtx = context.Background()
	}
	c := &Controller{
		runner:          opts.Runner,
		executor:        opts.Executor,
		sink:            sink,
		policy:          opts.Policy,
		label:           opts.Label,
		systemPrompt:    opts.SystemPrompt,
		sessionDir:      opts.SessionDir,
		sessionPath:     opts.SessionPath,
		logFormat:       opts.LogFormat,
		host:            opts.Host,
		commands:        opts.Commands,
		skills:          opts.Skills,
		hooks:           opts.Hooks,
		mem:             opts.Memory,
		memoryEnabled:   !opts.MemoryDisabled,
		cleanup:         opts.Cleanup,
		balanceURL:      opts.BalanceURL,
		balanceKey:      opts.BalanceKey,
		balanceKind:     opts.BalanceKind,
		jobs:            opts.Jobs,
		reg:             opts.Registry,
		pluginCtx:       pluginCtx,
		ctxMgr:          opts.CtxMgr,
		permLevel:       "ask",
		approvalTimeout: opts.ApprovalTimeout,
		approvals:       map[string]chan approvalReply{},
		asks:            map[string]chan []event.AskAnswer{},
		granted:         map[string]bool{},
	}
	// Checkpoints: bind a store to the session and route writer pre-edits into it.
	if c.executor != nil {
		c.executor.SetSessionSaver(c)
		c.executor.SetPromoter(c)
	}
	// 3.0 Step 1 事件日志模式接线：把持久化格式注入当前会话并挂上检查点
	// flush 钩子（fail-closed：模型调用前由 executor 触发，失败中止回合）。
	c.applyLogFormat(c.logFormat)
	return c
}

// runGuarded starts a turn unless one is already in flight, guarding against
// concurrent turns and emitting a TurnDone event when the turn finishes (Err set
// on failure; nil also for a user Cancel). While a turn is running the request
// is ignored — the Send path uses its own bounded queue instead (see
// SendWithRaw).
func (c *Controller) runGuarded(body func(ctx context.Context) error) {
	if !c.tryStartTurn(body) {
		c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
			Text: "a turn is already running — this request was ignored"})
	}
}

// tryStartTurn starts a turn goroutine when no turn is in flight and reports
// whether it did. The running check and the state flip happen in the same
// critical section, so a concurrent caller either starts its own turn or
// observes the in-flight one.
func (c *Controller) tryStartTurn(body func(ctx context.Context) error) bool {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return false
	}
	c.launchTurnLocked(body)
	return true
}

// launchTurnLocked marks the session running and starts the turn goroutine
// (which also drains the Send queue, see turnLoop). The caller must already
// hold c.mu and must have checked c.running under that lock; this method
// releases the lock before returning.
func (c *Controller) launchTurnLocked(body func(ctx context.Context) error) {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.running = true
	c.mu.Unlock()
	go c.turnLoop(ctx, body)
}

// turnLoop executes body and then every message queued via Send (T6-2.5) in
// FIFO order, emitting a TurnDone after each turn. running stays true while the
// queue is non-empty so new Sends keep queueing; it is cleared only when the
// queue drains. A panicking turn is recovered into a failed TurnDone and the
// queue still continues.
func (c *Controller) turnLoop(ctx context.Context, body func(ctx context.Context) error) {
	for {
		err := c.runOneTurn(ctx, body)

		c.mu.Lock()
		if len(c.pendingSends) > 0 {
			next := c.pendingSends[0]
			c.pendingSends = c.pendingSends[1:]
			newCtx, cancel := context.WithCancel(context.Background())
			c.cancel = cancel
			c.mu.Unlock()
			c.sink.Emit(event.Event{Kind: event.TurnDone, Err: err})
			ctx = newCtx
			body = func(ctx context.Context) error {
				return c.runSendTurn(ctx, next.input, next.raw)
			}
			continue
		}
		c.running = false
		c.cancel = nil
		c.mu.Unlock()
		c.sink.Emit(event.Event{Kind: event.TurnDone, Err: err})
		return
	}
}

// runOneTurn runs a single turn body, converting a panic into an error so the
// queue drain above keeps going (behaviour matches the pre-queue runGuarded).
func (c *Controller) runOneTurn(ctx context.Context, body func(ctx context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("controller: turn panic recovered", "panic", r)
			err = fmt.Errorf("turn panic: %v", r)
		}
	}()
	return body(ctx)
}

// Send starts a turn with an uncomposed message. The controller applies
// auto-plan, plan-mode, memory, and background-job framing inside the async turn
// path so frontends do not block on classifier I/O.
func (c *Controller) Send(input string) {
	c.SendWithRaw(input, input)
}

// SendWithRaw starts a turn with separate model input and raw prompt text. The
// raw prompt is used only for auto-plan scoring; it deliberately excludes
// resolved @-reference payloads so referenced file contents cannot inflate the
// complexity score.
//
// While a turn is running the message is queued (bounded, sendQueueLimit) and
// executed in FIFO order after the current turn ends; a full queue rejects the
// message with an explicit error Notice. With no turn in flight the message
// starts a turn immediately (unchanged behaviour).
func (c *Controller) SendWithRaw(input, raw string) {
	c.mu.Lock()
	if c.running {
		if len(c.pendingSends) >= sendQueueLimit {
			c.mu.Unlock()
			c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
				Text: fmt.Sprintf("发送队列已满（最多 %d 条），这条消息未发送 — 请等待当前回合结束", sendQueueLimit)})
			return
		}
		c.pendingSends = append(c.pendingSends, sendRequest{input: input, raw: raw})
		queued := len(c.pendingSends)
		c.mu.Unlock()
		c.notice(fmt.Sprintf("当前回合进行中，消息已加入发送队列（%d/%d），回合结束后自动发送", queued, sendQueueLimit))
		return
	}
	c.launchTurnLocked(func(ctx context.Context) error {
		return c.runSendTurn(ctx, input, raw)
	})
}

// runSendTurn runs one Send turn: resolve @-references first (consistent with
// the Submit path), then run the composed turn.
func (c *Controller) runSendTurn(ctx context.Context, input, raw string) error {
	// 与 Submit 路径保持一致：先解析 @ 引用（文件内容 / MCP 资源 /
	// 图片识图结果），再进入回合，保证桌面端和 HTTP 端行为一致。
	block, errs := c.ResolveRefs(ctx, input)
	for _, e := range errs {
		c.notice(e)
	}
	sent := input
	if block != "" {
		sent = "Referenced context:\n\n" + block + "\n\n" + input
	}
	return c.runTurnWithRaw(ctx, sent, raw)
}

func (c *Controller) runTurnWithRaw(ctx context.Context, input, raw string) error {
	c.maybeSessionStart(ctx)

	// 会话中断标记（T5-4）：回合开始时写 running=true，回合正常结束
	// （无论成功/失败/取消）由 defer 写回 running=false 与最后助手摘要；
	// 若进程在回合中被杀/崩溃，running=true 残留即为「上次未完成」信号。
	// 仅交互回合标记（runTurnWithRaw）；headless Run 路径不写状态。
	if p := c.statePath(); p != "" {
		if err := session.SaveState(p, session.SessionState{Running: true, UpdatedAt: time.Now().UnixMilli()}); err != nil {
			slog.Warn("controller: mark session running", "path", p, "err", err)
		}
	}
	defer func() {
		if p := c.statePath(); p != "" {
			if err := session.SaveState(p, session.SessionState{
				Running:   false,
				Summary:   truncateSummary(lastAssistantText(c.History())),
				UpdatedAt: time.Now().UnixMilli(),
			}); err != nil {
				slog.Warn("controller: clear session running mark", "path", p, "err", err)
			}
		}
	}()

	// V3.0 Phase 5: ContextManager handles first-turn orchestration.

	// V3.0 Phase 5: ContextManager handles first-turn orchestration.
	// ProcessFirstTurn locks the runtime (idempotent). On the first turn,
	// also push the L2 system prompt into the agent so the model gets
	// project/task context. Subsequent turns reuse the cached L2 bytes.
	if c.ctxMgr != nil {
		wasLocked := c.ctxMgr.Runtime().IsLocked()
		c.ctxMgr.ProcessFirstTurn(input)
		if !wasLocked {
			// V7.5: 将运行时上下文合并到 L1 系统提示词末尾，
			// 取代原 L2 注入 + WarmupCache 方案，前缀永不改变。
			c.executor.MergeRuntimePrompt(c.ctxMgr.Runtime().SystemPrompt())
		}
	}

	input = c.Compose(input)
	// Open a checkpoint for this turn before the user message is appended, so the
	// recorded message boundary precedes it and pre-edit snapshots land here.
	// UserPromptSubmit / Stop hooks bracket the whole turn (incl. the plan
	// research + approved-execution sub-turns below): a gating UserPromptSubmit
	// aborts before any model call; Stop fires once when the turn returns.
	if c.hooks.Enabled() {
		c.mu.Lock()
		c.turn++
		turn := c.turn
		c.mu.Unlock()
		if block, _ := c.hooks.PromptSubmit(ctx, input, turn); block {
			return nil // the hook's notify callback already surfaced the reason
		}
		defer func() { c.hooks.Stop(ctx, lastAssistantText(c.History()), turn) }()
	}
	// 3.0 Step 1 事件日志模式（session.log_format="event"）：模型调用前
	// flush 检查点（fail-closed——检查点写失败即中止回合，绝不带着未持久化
	// 的状态调用模型），随后把用户消息写入事件日志（「模型可见必入日志」）。
	if c.EventMode() {
		if c.executor != nil {
			if err := c.executor.FlushCheckpointFailClosed(); err != nil {
				return fmt.Errorf("flush checkpoint before model call: %w", err)
			}
		}
		if err := c.logUserMessage(input); err != nil {
			return err
		}
	}
	if _, err := c.runner.Run(ctx, input); err != nil {
		return err
	}
	c.touchMemoryCitations()
	// 每轮对话后自动快照保存，确保崩溃/重启不丢上下文
	if err := c.Snapshot(); err != nil {
		slog.Warn("controller: snapshot after turn", "err", err)
	}
	return nil
}

// touchMemoryCitations 解析最终回复中的 [MEM:name] 引用并触达对应记忆（更新
// last_used_at，高频排序与前端引用徽标同源）——记忆引用可追溯的回传侧。
// 静默：记忆关闭/未命中/未知键都不报错，不影响回合结果。
func (c *Controller) touchMemoryCitations() {
	if c.mem == nil || !c.memoryEnabled {
		return
	}
	c.mem.ResolveCitations(lastAssistantText(c.History()))
}

// lastAssistantText returns the content of the most recent assistant message with
// non-empty text.
func lastAssistantText(msgs []provider.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == provider.RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
			return msgs[i].Content
		}
	}
	return ""
}

// statePath 返回当前会话的中断状态文件路径。持久化不可用（未接会话目录）
// 或执行器未构建时返回空串，调用方据此跳过状态读写。
func (c *Controller) statePath() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.executor == nil || c.sessionPath == "" {
		return ""
	}
	return session.StatePath(c.sessionPath)
}

// truncateSummary 将摘要截断到 240 字符（按 rune，兼容中文），
// 避免把超长助手消息整体写进状态文件。
func truncateSummary(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 240 {
		return string(r[:240])
	}
	return s
}

// Submit is the one-call entry for a simple frontend: it takes raw user input
// and does everything — slash-command dispatch, @-reference expansion —
// emitting all output as events. The HTTP/SSE server uses this so
// a browser client only POSTs the typed line.
//
// Slash commands route to the matching primitive: /compact and /new run their
// session op and emit a Notice; /mcp__server__prompt and custom /commands
// resolve to a turn; an unknown slash emits a Notice. Anything else is a normal
// turn with its @-references resolved first.
func (c *Controller) notice(text string) {
	c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: text})
}

// Run executes a turn synchronously, returning the agent's error. Used by the
// headless `gaea run` path, where the Sink renders to stdout and the caller
// just needs the exit status — no TurnDone event, no cancel bookkeeping.
func (c *Controller) Run(ctx context.Context, input string) error {
	c.maybeSessionStart(ctx)

	// V3.0 Phase 5: ContextManager takes over first-turn orchestration.
	if c.ctxMgr != nil {
		wasLocked := c.ctxMgr.Runtime().IsLocked()
		c.ctxMgr.ProcessFirstTurn(input)
		if !wasLocked {
			// V7.5: 将运行时上下文合并到 L1
			c.executor.MergeRuntimePrompt(c.ctxMgr.Runtime().SystemPrompt())
		}
	}

	if c.hooks.Enabled() {
		c.mu.Lock()
		c.turn++
		turn := c.turn
		c.mu.Unlock()
		if block, _ := c.hooks.PromptSubmit(ctx, input, turn); block {
			return nil
		}
		defer func() { c.hooks.Stop(ctx, lastAssistantText(c.History()), turn) }()
	}
	_, err := c.runner.Run(ctx, input)
	if err == nil {
		c.touchMemoryCitations()
	}
	return err
}

// Cancel aborts the in-flight turn. A goroutine blocked awaiting approval
// unblocks via the cancelled context.
func (c *Controller) Cancel() {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Steer 在任务运行中插话调整（2026-08-28，对齐豆包工作「边跑边改」/
// Codex mid-turn steer）：把消息注入当前回合的 steer 队列，agent 在下一轮
// 采样前作为 guidance 消费——模型看到的是「对当前任务的补充指引」而不是
// 新任务。不打断当前工具执行；回合结束后若仍有未消费消息则自然转为下一轮。
// 未运行（无回合可插话）时走普通 Send 排队，保证消息不丢。
func (c *Controller) Steer(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	c.mu.Lock()
	executor := c.executor
	running := c.running
	c.mu.Unlock()
	if executor != nil && running {
		executor.Steer(text)
		c.sink.Emit(event.Event{Kind: event.Steer, Text: text})
		return
	}
	c.Send(text)
}

// Running reports whether a turn is currently in flight.
func (c *Controller) Running() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// Approve answers a pending ApprovalRequest by ID: allow runs the call, session
// also remembers a grant for the rest of the session so the same tool+subject
// isn't re-prompted; abort rejects the call AND cancels the in-flight turn
// (拒绝并停止本轮). Unknown/expired IDs are ignored.
func (c *Controller) Approve(id string, allow, session, abort bool) {
	c.mu.Lock()
	reply := c.approvals[id]
	delete(c.approvals, id)
	c.mu.Unlock()
	if reply != nil {
		reply <- approvalReply{allow: allow, session: session, abort: abort} // buffered, never blocks
	}
}

// EnableInteractiveApproval swaps the executor's gate for one that routes "ask"
// decisions to the frontend via ApprovalRequest events, and wires the controller
// in as the executor's Asker so the `ask` tool can question the user. Interactive
// frontends (chat, desktop) call this; the headless run keeps the silent gate and
// a nil asker from setup.
func (c *Controller) EnableInteractiveApproval() {
	if c.executor != nil {
		g := permission.NewGate(c.policy, gateApprover{c})
		g.AlwaysAsk = hardAskTools
		c.executor.SetGate(g)
		c.executor.SetAsker(c)
	}
}

// Ask implements agent.Asker: it emits an AskRequest and blocks until
// AnswerQuestion(ID, …) answers or ctx is cancelled. promptMu serialises it
// against tool-approval prompts so at most one user prompt is outstanding.
func (c *Controller) Ask(ctx context.Context, questions []event.AskQuestion) ([]event.AskAnswer, error) {
	c.promptMu.Lock()
	defer c.promptMu.Unlock()

	c.mu.Lock()
	c.nextID++
	id := strconv.Itoa(c.nextID)
	reply := make(chan []event.AskAnswer, 1)
	c.asks[id] = reply
	c.mu.Unlock()

	c.sink.Emit(event.Event{Kind: event.AskRequest, Ask: event.Ask{ID: id, Questions: questions}})

	select {
	case ans := <-reply:
		return ans, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.asks, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

// AnswerQuestion resolves a pending AskRequest by ID with the user's selections.
// Unknown/expired IDs are ignored.
func (c *Controller) AnswerQuestion(id string, answers []event.AskAnswer) {
	c.mu.Lock()
	reply := c.asks[id]
	delete(c.asks, id)
	c.mu.Unlock()
	if reply != nil {
		reply <- answers // buffered, never blocks
	}
}

// SetPermLevel sets the permission strictness and immediately updates the gate:
//
//	"ask"  — prompt before writes (default), interactive gate active
//	"auto" — allow writes without asking, deny rules still block
//	"yolo" — skip all gating (nil gate = every tool auto-approved)
func (c *Controller) SetPermLevel(level string) {
	c.mu.Lock()
	c.permLevel = level
	// 持久化写入（成本库/记忆/知识库）必须逐条确认：
	// 任何权限级别都保留 AlwaysAsk 硬门。
	switch level {
	case "auto":
		c.policy.Mode = permission.Allow
		if c.executor != nil {
			g := permission.NewGate(c.policy, gateApprover{c})
			g.AlwaysAsk = hardAskTools
			c.executor.SetGate(g)
		}
	case "yolo":
		if c.executor != nil {
			// yolo 保持"跳过一切门禁"，但持久化写入例外仍必须确认：
			// 用空策略 + AlwaysAsk 的 gate 替代 nil gate。
			g := permission.NewGate(permission.New("allow", nil, nil, nil), gateApprover{c})
			g.AlwaysAsk = hardAskTools
			c.executor.SetGate(g)
		}
	default: // "ask"
		c.policy.Mode = permission.Ask
		if c.executor != nil {
			g := permission.NewGate(c.policy, gateApprover{c})
			g.AlwaysAsk = hardAskTools
			c.executor.SetGate(g)
		}
	}
	c.mu.Unlock()
}

// PermLevel returns the current permission level.
func (c *Controller) PermLevel() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.permLevel
}

// SetGoal sets the session goal (set via /goal) and propagates it to the
// SetGoal sets the session goal (set via /goal) and propagates it to the
// executor so the stop gate can enforce it.
func (c *Controller) SetGoal(g string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.goal = g
	if c.executor != nil {
		c.executor.SetGoal(g)
	}
}

// Goal returns the current session goal.
func (c *Controller) Goal() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.goal
}

// Compact runs one compaction pass on the executor's session on demand.
// instructions is optional `/compact <focus>` guidance steering what to keep.
// Since V5.0, explicit compaction has been replaced by automatic truncation in
// the run loop (≥500K tokens → three-tier compression). This method exists for
// API compatibility — the built-in truncation handles context pressure.
func (c *Controller) Compact(ctx context.Context, instructions string) error {
	if c.executor == nil {
		return nil
	}
	c.notice("compact: automatic truncation handles context compression — no manual /compact needed")
	return nil
}

// Dream extracts knowledge from the current session into project memory.
// Uses deterministic session summary (no LLM call). V6.0 Feature.
// maybeSessionStart fires the SessionStart hook exactly once per session, lazily
// on the first turn — by then the sink/notify is wired, and a resumed session
// fires it too (its first post-resume turn).
func (c *Controller) maybeSessionStart(ctx context.Context) {
	c.mu.Lock()
	if c.startedOnce {
		c.mu.Unlock()
		return
	}
	// 首次对话自动创建会话文件：不点「新会话」直接开聊也应可持久化，
	// 否则 Snapshot() 因 sessionPath 为空而跳过，重启后历史全部丢失。
	if c.sessionPath == "" && c.sessionDir != "" {
		c.sessionPath = agent.NewSessionPath(c.sessionDir, c.label)
	}
	c.startedOnce = true
	c.mu.Unlock()
	c.hooks.SessionStart(ctx)
}

// TCCAStats returns a formatted cache metrics report (V3.0).
// Returns empty string when ctxMgr is not wired.
func (c *Controller) TCCAStats() string {
	if c.ctxMgr == nil {
		return "TCCA not available (ContextManager not wired)"
	}
	r := c.ctxMgr.Metrics()
	return fmt.Sprintf(
		"TCCA Session Cache Report\n"+
			"========================\n"+
			"Layers:\n"+
			"  L1 Identity:  %d bytes\n"+
			"  L2 Runtime:   %d bytes\n"+
			"  L3 Skill:     v%d\n"+
			"  L4 Flow:      %d messages\n"+
			"\n"+
			"Savings (session):\n"+
			"  Compaction:   %d tokens saved (%d passes)\n"+
			"  Fork reuse:   %d tokens saved (%d forks)\n"+
			"  节省:         ¥%.4f\n"+
			"  Latency:      %d ms\n",
		r.L1Size, r.L2Size, r.L3Version, r.L4Messages,
		r.SavedByCompact, r.CompactionCount,
		r.SavedByFork, r.ForkCount,
		r.SavedUSD*7.25, r.SavedLatencyMs,
	)
}

// TCCAReport returns the structured cache metrics report (V3.0).
// Returns zero-value CacheReport when ctxMgr is not wired.
func (c *Controller) TCCAReport() tiancontext.CacheReport {
	if c.ctxMgr == nil {
		return tiancontext.CacheReport{}
	}
	return c.ctxMgr.Metrics()
}

// SystemPrompt returns the L1 system prompt.
func (c *Controller) SystemPrompt() string {
	if c.ctxMgr != nil {
		return c.ctxMgr.Identity().SystemPrompt()
	}
	return c.systemPrompt
}

// NewSession snapshots the current conversation, rotates to a fresh file, and
// resets the executor to a clean session carrying the same system prompt. It
// ends the old session and starts the new one for lifecycle hooks.
func (c *Controller) NewSession() error {
	if c.executor == nil {
		return nil
	}
	if err := c.Snapshot(); err != nil {
		return err
	}
	c.hooks.SessionEnd(context.Background())
	if c.sessionDir != "" {
		c.mu.Lock()
		c.sessionPath = agent.NewSessionPath(c.sessionDir, c.label)
		c.mu.Unlock()
	}
	ns := agent.NewSession(c.systemPrompt)
	c.mu.Lock()
	f := c.logFormat
	c.mu.Unlock()
	ns.SetLogFormat(f)
	c.executor.SetSession(ns)
	c.applyLogFormat(f)
	// Reset V3.0 TCCA state so the new session starts clean.
	if c.ctxMgr != nil {
		c.ctxMgr.Flow().ReplaceMessages(nil)
	}
	c.mu.Lock()
	c.startedOnce = true // NewSession fires SessionStart itself; don't re-fire on the next turn
	c.mu.Unlock()
	c.hooks.SessionStart(context.Background())
	return nil
}

// RewindScope selects what a Rewind restores.
// Resume seeds the session from a loaded transcript and pins the active file to
// its path so auto-save keeps appending there.
func (c *Controller) Resume(s *agent.Session, path string) {
	if c.executor != nil {
		c.executor.SetSession(s)
	}
	c.mu.Lock()
	c.sessionPath = path
	f := c.logFormat
	c.mu.Unlock()
	// 3.0 Step 1：恢复的会话继承当前持久化格式（事件日志模式下 Save 双写、
	// 后续回合落用户消息 + flush 检查点都依赖该标记）。
	s.SetLogFormat(f)
	if c.executor != nil {
		c.executor.SetCheckpointFlusher(c.flushCheckpoint)
	}
}

// ResumeFromDisk 从磁盘恢复会话并接管为当前会话。事件日志模式下：
// DetectLegacy → 旧格式先迁移 → Restore（checkpoint 消息 + log tail 重放），
// 无日志时回退 legacy Load；legacy 模式保持原 LoadSession + Resume 行为。
// 返回恢复后的会话（已注入 c.Resume）。
func (c *Controller) ResumeFromDisk(path string) (*agent.Session, error) {
	var (
		s   *agent.Session
		err error
	)
	if !c.EventMode() {
		s, err = agent.LoadSession(path)
		if err != nil {
			return nil, err
		}
	} else {
		// 旧格式会话（无事件日志而有 <id>.jsonl）先迁移，旧文件保留。
		if legacy, legacyPath, derr := session.DetectLegacy(path); derr != nil {
			return nil, derr
		} else if legacy {
			if _, merr := session.MigrateLegacyToLog(session.LogPathFor(path), legacyPath); merr != nil {
				return nil, fmt.Errorf("resume: migrate legacy session: %w", merr)
			}
		}
		s, err = session.LoadWithFormat(path, "event")
		if err != nil {
			return nil, err
		}
	}
	c.Resume(s, path)
	return s, nil
}

// EventMode 报告控制器是否处于事件日志模式（session.log_format="event"）。
func (c *Controller) EventMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.EqualFold(c.logFormat, "event")
}

// SetLogFormat 设置会话持久化格式（"legacy"/""=旧行为，"event"=事件日志），
// 并同步注入当前会话与检查点 flush 钩子。boot.Build 后由宿主（app）按配置
// 注入；缺省（空）时所有事件日志接线均为 no-op，行为与改造前一致。
func (c *Controller) SetLogFormat(f string) { c.applyLogFormat(f) }

// applyLogFormat 把持久化格式写入控制器并传播到当前会话与 executor 的
// 检查点 flush 钩子（fail-closed：模型调用前由 executor.FlushCheckpointFailClosed
// 触发，失败中止回合）。
func (c *Controller) applyLogFormat(f string) {
	c.mu.Lock()
	c.logFormat = f
	exec := c.executor
	c.mu.Unlock()
	if exec != nil {
		exec.Session().SetLogFormat(f)
		exec.SetCheckpointFlusher(c.flushCheckpoint)
	}
}

// flushCheckpoint 把当前会话消息投影 + 已消费的 log seq 写入检查点
// （事件日志模式）。无会话路径 / 无内容 / 尚无日志条目时不写（幂等）。
// 返回错误供 fail-closed 调用方（模型调用前）中止回合。
func (c *Controller) flushCheckpoint() error {
	c.mu.Lock()
	path := c.sessionPath
	exec := c.executor
	c.mu.Unlock()
	if path == "" || exec == nil {
		return nil
	}
	s := exec.Session()
	if !s.IsEventMode() || !s.HasContent() {
		return nil
	}
	logPath := session.LogPathFor(path)
	if logPath == "" {
		return nil
	}
	seq := session.LastLogSeq(logPath)
	if seq == 0 {
		return nil // 尚无日志条目（本会话还没有任何事件），无可固化内容
	}
	return session.WriteCheckpoint(session.CheckpointPathFor(path), seq, s.Snapshot())
}

// logUserMessage 把本回合用户消息追加到事件日志（「模型可见必入日志」：
// 运行期 user_message 由控制器在模型调用前落盘）。日志缺失时自动创建
// （旧格式会话先迁移）。失败返回错误（fail-closed）。
func (c *Controller) logUserMessage(input string) error {
	path := c.SessionPath()
	if path == "" {
		return nil
	}
	logPath := session.LogPathFor(path)
	if logPath == "" {
		return nil
	}
	if _, err := session.AppendUserMessage(logPath, path, input); err != nil {
		return fmt.Errorf("append user message to event log: %w", err)
	}
	return nil
}

// Snapshot writes the executor's conversation to the active session file. No-op
// when persistence is unavailable or the session has never been used (no user
// interaction). Called after every turn so a crash loses at most one in-flight
// prompt.
func (c *Controller) Snapshot() error {
	c.mu.Lock()
	path := c.sessionPath
	c.mu.Unlock()
	if c.executor == nil || path == "" {
		return nil
	}
	s := c.executor.Session()
	if !s.HasContent() {
		return nil
	}
	if err := s.Save(path); err != nil {
		return err
	}
	if err := agent.TouchBranchMeta(path); err != nil {
		return err
	}
	// 3.0 Step 1 事件日志模式：回合结束后 flush 检查点（含压缩后的消息
	// 投影 + 已消费 log seq），断电/崩溃后可由 checkpoint + log tail 恢复。
	if s.IsEventMode() {
		if err := c.flushCheckpoint(); err != nil {
			return err
		}
	}
	return nil
}

// SetSessionPath pins where auto-save lands (a fresh session file minted by the
// caller when no resume path applies).
func (c *Controller) SetSessionPath(p string) {
	c.mu.Lock()
	c.sessionPath = p
	f := c.logFormat
	exec := c.executor
	c.mu.Unlock()
	if exec != nil {
		exec.Session().SetLogFormat(f)
		exec.SetCheckpointFlusher(c.flushCheckpoint)
	}
}

// SessionDir reports the directory new session files land in ("" disables
// persistence), so the caller can decide whether to mint a path.
func (c *Controller) SessionDir() string { return c.sessionDir }

// SessionPath reports the file the current conversation auto-saves to ("" when
// persistence is disabled), so a history view can mark the active session.
func (c *Controller) SessionPath() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionPath
}

// History returns the executor's current message log (for repopulating a
// resumed frontend's view).
func (c *Controller) History() []provider.Message {
	if c.executor == nil {
		return nil
	}
	return c.executor.Session().Snapshot() // copy — a turn may be appending concurrently
}

// ContextSnapshot returns (promptTokens, contextWindow) from the most recent
// turn. Both zero means no data yet — a gauge hides itself.
func (c *Controller) ContextSnapshot() (int, int) {
	if c.executor == nil {
		return 0, 0
	}
	u := c.executor.LastUsage()
	if u == nil {
		return 0, c.executor.ContextWindow()
	}
	return u.PromptTokens, c.executor.ContextWindow()
}

// SeedContextUsage 用当前会话消息估算的 token 数作为初始上下文用量，
// 使恢复会话后顶栏"上下文"状态立即显示真实读数而非 0。
func (c *Controller) SeedContextUsage() {
	if c.executor == nil {
		return
	}
	c.executor.SeedUsage(provider.Usage{PromptTokens: c.executor.EstimateContextTokens()})
}

// CompactRatio returns the auto-compaction threshold as a fraction of the window
// (0 when the executor is unset). The status line shows headroom against it.
func (c *Controller) CompactRatio() float64 {
	if c.executor == nil {
		return 0
	}
	return c.executor.CompactRatio()
}

// LastUsage returns the most recent turn's token telemetry (nil before the first
// turn), so frontends can derive the prompt cache-hit rate for the status line.
func (c *Controller) LastUsage() *provider.Usage {
	if c.executor == nil {
		return nil
	}
	return c.executor.LastUsage()
}

// SessionCache returns cumulative cache hit/miss prompt tokens for the session,
// so a frontend can render the aggregate (session-wide) cache-hit rate — steadier
// than the single-turn rate and unaffected by compaction.
func (c *Controller) SessionCache() (hit, miss int) {
	if c.executor == nil {
		return 0, 0
	}
	return c.executor.SessionCache()
}

// Balance queries the active provider's wallet balance, or (nil, nil) when the
// provider declares no balance_url — so a caller treats "not configured" and
// "fetched" the same and just omits the readout when nil. The balance-backend
// kind comes from Options.BalanceKind ("" = historical default "deepseek");
// an unknown kind fails closed with an explicit error.
func (c *Controller) Balance(ctx context.Context) (*billing.Balance, error) {
	if strings.TrimSpace(c.balanceURL) == "" {
		return nil, nil
	}
	kind := strings.TrimSpace(c.balanceKind)
	if kind == "" {
		kind = billing.BalanceKindDeepSeek // 历史默认：DeepSeek GET /user/balance 形状
	}
	return billing.FetchByKind(ctx, kind, c.balanceURL, c.balanceKey)
}

// Host returns the running MCP host (nil when no plugins), for frontends that
// list servers / resolve MCP prompts.
func (c *Controller) Host() *plugin.Host { return c.host }

// Commands returns the loaded custom slash commands.
func (c *Controller) Commands() []command.Command { return c.commands }

// Skills returns the discoverable skills (for the slash menu and `/skill`).
func (c *Controller) Skills() []skill.Skill { return c.skills }

// Tools returns the executor's live tool set (enabled built-ins + plugins +
// dynamic tools) in registration order. nil registry (never built) → nil.
func (c *Controller) Tools() []tool.Tool {
	if c.reg == nil {
		return nil
	}
	out := make([]tool.Tool, 0, c.reg.Len())
	for _, name := range c.reg.Names() {
		if t, ok := c.reg.Get(name); ok {
			out = append(out, t)
		}
	}
	return out
}

// HookRunner returns the session's hook runner (nil-safe; may hold zero hooks),
// so a frontend can list the active hooks via `/hooks`.
func (c *Controller) HookRunner() *hook.Runner { return c.hooks }

// AddMCPServer connects an MCP server live and persists it to the config file. Its
// tools are registered immediately and become available on the next turn (the
// agent reads the registry per turn). The raw entry — ${VARS} intact — is what's
// written to disk; the live connection uses the expanded form. Returns the number
// of tools the server exposed. A save failure after a successful connect is
// reported but non-fatal: the server still works this session.

// Label returns the human-readable model label, e.g. "deepseek-flash".
func (c *Controller) Label() string { return c.label }

// Close stops plugin subprocesses and releases resources. A session that ever
// started fires SessionEnd so a teardown hook runs.
func (c *Controller) Close() {
	c.mu.Lock()
	started := c.startedOnce
	c.mu.Unlock()
	if started {
		c.hooks.SessionEnd(context.Background())
	}
	if c.jobs != nil {
		c.jobs.Close() // cancel any still-running background jobs
	}
	if c.cleanup != nil {
		c.cleanup()
	}
}

// Jobs returns the still-running background jobs for the status bar (nil when
// background jobs are disabled).
func (c *Controller) Jobs() []jobs.View {
	if c.jobs == nil {
		return nil
	}
	return c.jobs.Running()
}
