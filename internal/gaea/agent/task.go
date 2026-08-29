package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// DefaultTaskSystemPrompt steers a sub-agent toward focused, terse delivery —
// it doesn't see the parent's conversation so it must self-contain.
const DefaultTaskSystemPrompt = `You are a sub-agent invoked by a parent engineering office assistant to carry out one focused task.
Use the provided tools to investigate or act. Return a single final answer that is concise
and self-contained — the parent will see only that answer, not your tool calls or reasoning.
If you need to ask for clarification, fail with a precise question instead of guessing.`

// taskResultTag wraps sub-agent output in structured XML so the parent agent can
// distinguish the result from other tool output. Borrowed from opencode.
const (
	taskResultTagOpen  = "<task-result>"
	taskResultTagClose = "</task-result>"
)

// wrapTaskResult wraps a sub-agent's final answer in structured XML tags so the
// parent model can reliably identify and parse it.
func wrapTaskResult(text string) string {
	return taskResultTagOpen + "\n" + strings.TrimSpace(text) + "\n" + taskResultTagClose
}

// RetryUntilConfig enables automatic retry loop for a task sub-agent.
// After the sub-agent returns, the check command is executed. If it fails
// (non-zero exit), the failure output is injected as context and the sub-agent
// is re-invoked. Repeats until the check passes or max_retries is exhausted.
type RetryUntilConfig struct {
	Check      string `json:"check"`       // Shell command to verify success, e.g. "go test ./..."
	MaxRetries int    `json:"max_retries"` // Maximum retry attempts (default 3, max 10)
}

var subagentMetaTools = []string{
	"task",
	"run_skill",
	"install_skill",
	"explore",
	"research",
	"review",
	"security_review",
}

// SubagentMetaTools returns the tool names that spawned agents should not inherit
// from the parent registry unless a future call site deliberately opts into a
// different boundary. They can spawn or author more agent work, so excluding them
// preserves one layer of delegation without adding a spawn-count cap.
func SubagentMetaTools() []string {
	out := make([]string, len(subagentMetaTools))
	copy(out, subagentMetaTools)
	return out
}

// IsSubagentMetaTool reports whether the tool name spawns a sub-agent that makes
// its own API calls. These calls can evict the parent's cache prefix on the
// provider side (especially on smaller cache pools like flash 128K), so the
// parent should re-warm after the sub-agent returns.
func IsSubagentMetaTool(name string) bool {
	for _, t := range subagentMetaTools {
		if t == name {
			return true
		}
	}
	return false
}

// TaskCompiler is the subset of cache.Compiler that TaskTool needs for
// fork-based cache sharing. Defined here so the agent package doesn't
// import the cache package. The Fork return is interface-typed because
// cache.Compiler.Fork() returns a concrete *Compiler, not this interface.
type TaskCompiler interface {
	Fork() interface{ SystemPrompt() string }
	SystemPrompt() string
}

// TaskTool spawns a sub-agent in its own session for a focused sub-task. The
// sub-agent runs with a filtered tool whitelist and the same step budget shape
// as the parent (see Execute); its tool calls are forwarded to the parent's
// event stream nested under this call, while only its final assistant message is
// returned to the parent model. Use cases: keep noisy tool sequences (multi-file
// exploration, repeated grep / read_file) out of the parent's context budget, or
// parallel research across independent areas (the parallel-dispatch path picks
// these up only when readOnly, which task is not).
type TaskTool struct {
	prov             provider.LLMProvider
	pricing          *provider.Pricing
	parentReg        *tool.Registry
	maxSteps         int
	contextWindow    int
	temperature      float64
	archiveDir       string
	sysPrompt        string
	gate             Gate
	hooks            ToolHooks             // optional: gates the retry_until check command like any normal tool call
	compiler         TaskCompiler          // optional, for cache sharing via Fork
	runtimePrompt    string                // V5.25: L2 runtime context for sub-agents
	templatePrefix   string                // V5.30: 子代理模板前缀，同类子代理共享缓存
	accumulatedUsage *provider.Usage       // V5.30: 子代理累计 token 用量
	activeSchemas    []provider.ToolSchema // V5.30: 父代理过滤工具集，子代理继承以共享缓存
	subagentProv     provider.LLMProvider  // V10.22: optional subagent model provider (nil → use prov)
	subagentPricing  *provider.Pricing
	subagentCtxWin   int

	transcripts *SubagentStore // V10.29: subagent transcript persistence (continue_from)
}

// NewTaskTool wires a task tool to the parent agent's environment so its
// sub-agents can use the same provider and tools. sysPrompt is the system
// prompt every sub-agent starts with; pass "" for DefaultTaskSystemPrompt. gate
// is the permission gate sub-agents inherit — pass the headless variant so
// deny rules still bite while autonomous sub-agents are never blocked on an
// interactive prompt (there is no UI to answer one).
func NewTaskTool(prov provider.LLMProvider, pricing *provider.Pricing, parentReg *tool.Registry,
	maxSteps, contextWindow int, temperature float64, archiveDir, sysPrompt string, gate Gate) *TaskTool {
	if sysPrompt == "" {
		sysPrompt = DefaultTaskSystemPrompt
	}
	return &TaskTool{
		prov:          prov,
		pricing:       pricing,
		parentReg:     parentReg,
		maxSteps:      maxSteps,
		contextWindow: contextWindow,
		temperature:   temperature,
		archiveDir:    archiveDir,
		sysPrompt:     sysPrompt,
		gate:          gate,
	}
}

func (t *TaskTool) Name() string { return "task" }

func (t *TaskTool) Description() string {
	return "Spawn a sub-agent for a focused sub-task. The sub-agent runs in its own session with the same provider and a filtered tool list (defaults to every parent tool except subagent/skill meta-tools, so delegation stays one layer deep). Only its final answer is returned. Set output_schema to get structured JSON back (e.g. {files_modified: [...], key_decisions: [...]}). Use this to (a) keep long exploration sequences out of the parent's context budget, or (b) delegate self-contained work like 'find every place that calls X and summarise the patterns'."
}

func (t *TaskTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "prompt":{"type":"string","description":"What the sub-agent should accomplish. Be specific about the deliverable — the sub-agent does not see this conversation."},
  "description":{"type":"string","description":"Short label for the sub-task (3-7 words). Surfaced in the dispatch line so the user sees what's running."},
  "tools":{"type":"array","items":{"type":"string"},"description":"Optional tool whitelist. Subagent/skill meta-tools are still excluded so delegation stays one layer deep."},
  "max_steps":{"type":"integer","description":"Optional cap on tool-call rounds. Defaults to half the parent's cap (min 5).","minimum":1},
  "run_in_background":{"type":"boolean","description":"Run the sub-agent asynchronously: returns a job id immediately and keeps working across turns. Collect its final answer with wait, and you'll be notified when it finishes. Use for long, independent sub-tasks you don't need to block on right now."},
  "output_schema":{"type":"object","description":"Optional JSON Schema the sub-agent MUST return its result in. If set, the parent will attempt to parse the final answer as JSON. If the result is valid JSON matching the expected shape, it is returned verbatim; otherwise a diagnostic note is prefixed. Use when the parent needs structured data from the sub-agent."},
  "retry_until":{"type":"object","properties":{"check":{"type":"string","description":"Shell command to verify success, e.g. 'go test ./...'. Non-zero exit = retry."},"max_retries":{"type":"integer","description":"Maximum retry attempts (default 3, max 10).","minimum":1,"maximum":10}},"required":["check"]},
  "continue_from":{"type":"string","description":"Continue a prior compatible subagent transcript in the current conversation context. Pass only the 'sa_...' value from the prior result's 'Subagent reference: ...' line."}
},
"required":["prompt"]
}`)
}

// ReadOnly is false: a sub-agent can invoke any whitelisted tool, including
// writers. Conservative classification keeps the parallel-dispatch path from
// running two sub-agents at once and letting their writes race.
func (t *TaskTool) ReadOnly() bool { return false }

// CompactDescriptor — V10.11: compact task description for prompt efficiency.
func (t *TaskTool) CompactDescription() string {
	return "派发隔离子代理执行子任务(可设置output_schema获取结构化JSON)"
}
func (t *TaskTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string"},"description":{"type":"string"},"tools":{"type":"array","items":{"type":"string"}},"max_steps":{"type":"integer"},"run_in_background":{"type":"boolean"},"output_schema":{"type":"object"},"retry_until":{"type":"object","properties":{"check":{"type":"string"},"max_retries":{"type":"integer"}},"required":["check"]},"continue_from":{"type":"string"}},"required":["prompt"]}`)
}

func (t *TaskTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Prompt          string            `json:"prompt"`
		Description     string            `json:"description"`
		Tools           []string          `json:"tools"`
		MaxSteps        int               `json:"max_steps"`
		RunInBackground bool              `json:"run_in_background"`
		OutputSchema    json.RawMessage   `json:"output_schema,omitempty"`
		RetryUntil      *RetryUntilConfig `json:"retry_until,omitempty"`
		ContinueFrom    string            `json:"continue_from,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}

	maxSteps := p.MaxSteps
	if maxSteps <= 0 {
		if t.maxSteps > 0 {
			maxSteps = t.maxSteps / 2
			if maxSteps < 5 {
				maxSteps = 5
			}
		}
	}

	subReg := t.buildSubReg(p.Tools)

	// V10.29: prepare transcript — continue_from loads existing, otherwise fresh.
	run, prepErr := t.prepareRun(p.ContinueFrom, p.RunInBackground)
	if prepErr != nil {
		return "", prepErr
	}
	if run != nil {
		defer run.Release()
	}

	// retry_until: foreground only (background retry doesn't make sense across turns).
	if p.RetryUntil != nil && !p.RunInBackground {
		result, err := t.runSubWithRetrySession(ctx, p.Prompt, p.RetryUntil, subReg, run, maxSteps, p.OutputSchema)
		return t.finalizeRun(result, err, run)
	}

	if p.RunInBackground {
		if p.ContinueFrom != "" {
			return "", fmt.Errorf("continue_from cannot be used with run_in_background")
		}
		jm, ok := jobs.FromContext(ctx)
		if !ok {
			// No jobs manager in this context (e.g. headless sub-agent).
			// Fall back to foreground execution — sub-agents are short-lived
			// and don't persist across turns.
			result, err := t.runSubSession(ctx, p.Prompt, subReg, subSink(ctx), run, maxSteps, p.OutputSchema)
			return t.finalizeRun(result, err, run)
		}
		parentID, parent, _, _ := CallContext(ctx)
		nested := subSinkFor(parentID, parent)
		label := p.Description
		if label == "" {
			label = "task"
		}
		job := jm.Start("task", label, func(jobCtx context.Context, _ io.Writer) (string, error) {
			return t.runSubSession(jobCtx, p.Prompt, subReg, nested, run, maxSteps, p.OutputSchema)
		})
		return fmt.Sprintf("Started background task %q (%s). It runs across turns; collect its final answer with wait (or wait will return it once done), and you'll be notified when it finishes.", job.ID, label), nil
	}

	result, err := t.runSubSession(ctx, p.Prompt, subReg, subSink(ctx), run, maxSteps, p.OutputSchema)
	return t.finalizeRun(result, err, run)
}

func (t *TaskTool) buildSubReg(names []string) *tool.Registry {
	return FilterRegistry(t.parentReg, names, SubagentMetaTools()...)
}

func FilterRegistry(parent *tool.Registry, names []string, exclude ...string) *tool.Registry {
	ex := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		ex[e] = true
	}
	sub := tool.NewRegistry()
	src := names
	if len(src) == 0 {
		src = parent.Names()
	}
	for _, name := range src {
		if ex[name] {
			continue
		}
		tl, ok := parent.Get(name)
		if !ok {
			continue
		}
		// 持久化写入工具（工具注册表 PersistWrite 标记，T6-2.5）一律不继承：
		// 子代理运行在 headless 审批通道（Approver 为空 → 自动放行），若继承
		// 这些工具可绕过主代理的逐条确认，静默写入成本库 / 记忆 / 知识库 /
		// 技能文件。这些写入必须由主代理执行并逐条经用户确认。新增持久化写
		// 工具只需在注册处加标记即自动纳入禁写集合。
		if tool.IsPersistWrite(tl) {
			continue
		}
		sub.Add(tl)
	}
	return sub
}

func (t *TaskTool) SetCompiler(c TaskCompiler)                     { t.compiler = c }
func (t *TaskTool) SetRuntimePrompt(p string)                      { t.runtimePrompt = p }
func (t *TaskTool) SetTemplatePrefix(prefix string)                { t.templatePrefix = prefix }
func (t *TaskTool) SetActiveSchemas(schemas []provider.ToolSchema) { t.activeSchemas = schemas }
func (t *TaskTool) SubUsage() *provider.Usage                      { return t.accumulatedUsage }

// SetSubagentProvider installs an optional provider for sub-agents. When nil the
// sub-agent falls back to the parent's execution provider (prov).
func (t *TaskTool) SetSubagentProvider(p provider.LLMProvider, pricing *provider.Pricing, ctxWin int) {
	t.subagentProv = p
	t.subagentPricing = pricing
	t.subagentCtxWin = ctxWin
}

// WithTranscripts wires the subagent transcript store for continue_from support.
// When nil, sub-agents are ephemeral and cannot be continued across turns.
func (t *TaskTool) WithTranscripts(store *SubagentStore) *TaskTool {
	t.transcripts = store
	return t
}

// WithHooks wires the parent's hook runner so the retry_until check command
// passes through the same PermissionRequest / PreToolUse gating as a normal
// tool call. Optional — nil hooks (the default) skip hook checks while the
// permission gate still applies to every check command.
func (t *TaskTool) WithHooks(h ToolHooks) *TaskTool {
	t.hooks = h
	return t
}

// runSubSession executes the sub-agent with the given session (from a SubagentRun if
// non-nil, otherwise creates an ephemeral session). When run is non-nil the session
// from the store is used directly (supporting continue_from).
func (t *TaskTool) runSubSession(ctx context.Context, prompt string, subReg *tool.Registry, sink event.Sink, run *SubagentRun, maxSteps int, outputSchema json.RawMessage) (string, error) {
	// V6.0: sub-agent does NOT inherit parent L1+L2 — uses DefaultTaskSystemPrompt independently.
	// This saves ~50K tokens per sub-agent call (97% reduction) and keeps cache stats separate.
	sysPrompt := t.sysPrompt

	// V5.30 / V10.36: ActiveSchemas sends parent's full tool set to the API so
	// tools JSON matches — DeepSeek prefix cache hits across parent→sub-agent.
	// Execution gated by subReg (buildSubReg filtering), meta-tools blocked.
	subProv, subPrice, subCtxWin := t.prov, t.pricing, t.contextWindow
	if t.subagentProv != nil {
		subProv = t.subagentProv
		if t.subagentPricing != nil {
			subPrice = t.subagentPricing
		}
		if t.subagentCtxWin > 0 {
			subCtxWin = t.subagentCtxWin
		}
	}

	var subUsage provider.Usage
	var result string
	var err error
	if run != nil && run.Session != nil {
		result, err = RunSubAgentWithSession(ctx, subProv, subReg, run.Session, prompt, Options{
			MaxSteps:      maxSteps,
			Temperature:   t.temperature,
			Pricing:       subPrice,
			Gate:          t.gate,
			ContextWindow: subCtxWin,
			Compaction:    CompactionConfig{ArchiveDir: t.archiveDir},
			ActiveSchemas: t.parentReg.Schemas(), // V10.36: align tools JSON with parent for cache
		}, sink, &subUsage)
	} else {
		result, err = RunSubAgent(ctx, subProv, subReg, sysPrompt, prompt, Options{
			MaxSteps:      maxSteps,
			Temperature:   t.temperature,
			Pricing:       subPrice,
			Gate:          t.gate,
			ContextWindow: subCtxWin,
			Compaction:    CompactionConfig{ArchiveDir: t.archiveDir},
			ActiveSchemas: t.parentReg.Schemas(), // V10.36: align tools JSON with parent for cache
		}, sink, &subUsage)
	}
	if err == nil && len(outputSchema) > 0 {
		// output_schema set: verify the result is parseable JSON.
		// We don't validate every field (full JSON Schema needs a lib),
		// but we confirm the sub-agent returned well-formed JSON.
		var parsed interface{}
		if json.Unmarshal([]byte(result), &parsed) != nil {
			result = "[output_schema: sub-agent returned non-JSON; parent should retry]" + "\n" + result
		}
		u := subUsage
		t.accumulatedUsage = &u
		return result, nil
	}
	if err == nil {
		u := subUsage
		t.accumulatedUsage = &u
		// V10.12: wrap successful sub-agent results in structured XML tags
		// so the parent can reliably identify the result. Borrowed from opencode.
		return wrapTaskResult(result), nil
	}
	return result, err
}

// runSubWithRetrySession executes the sub-agent in a retry loop with a check command.
// The run parameter provides the session for continue_from; if nil a fresh session
// is created per RunSubAgent default. After each retry the same session accumulates
// messages so the sub-agent sees the full failure history.
func (t *TaskTool) runSubWithRetrySession(ctx context.Context, prompt string, cfg *RetryUntilConfig, subReg *tool.Registry, run *SubagentRun, maxSteps int, outputSchema json.RawMessage) (string, error) {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if maxRetries > 10 {
		maxRetries = 10
	}

	currentPrompt := prompt
	var finalResult string
	var subSession *session.Session
	if run != nil {
		subSession = run.Session
	}
	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := t.runSubSession(ctx, currentPrompt, subReg, subSink(ctx), run, maxSteps, outputSchema)
		if err != nil {
			return result, err
		}
		finalResult = result
		// After first attempt with a persisted session, keep using it.
		if run == nil && subSession != nil {
			run = &SubagentRun{Session: subSession} // ephemeral wrapper for retries
		}

		checkOutput, checkErr := t.runCheckCommand(ctx, cfg.Check)
		if errors.Is(checkErr, errCheckBlocked) {
			// The permission gate or a PreToolUse hook refused the check
			// command and it was never executed. Re-running the sub-agent
			// cannot fix a permission denial — surface the block immediately
			// instead of burning retries on it.
			return result, fmt.Errorf("%w\n\nSub-agent's final result (unverified):\n%s", checkErr, result)
		}
		if checkErr == nil {
			// Check passed — return the sub-agent's result.
			return result, nil
		}

		if attempt == maxRetries {
			return result, fmt.Errorf("retry_until: check command %q failed after %d retries.\nLast check output:\n%s\n\nSub-agent's final result:\n%s",
				cfg.Check, maxRetries+1, checkOutput, result)
		}

		// Check failed — inject failure context and retry.
		currentPrompt = fmt.Sprintf(
			"Previous attempt failed the verification. The check command `%s` produced:\n\n%s\n\nFix the issues above and try again.\n\nOriginal task: %s",
			cfg.Check, checkOutput, prompt)
	}
	return finalResult, fmt.Errorf("retry_until: unreachable")
}

// errCheckBlocked marks that the retry_until check command was refused by the
// permission gate or a PreToolUse hook and was NOT executed. The retry loop
// treats it as terminal: a permission denial is not something re-running the
// sub-agent can fix, so it must never be retried.
var errCheckBlocked = errors.New("retry_until check command blocked")

// runCheckCommand executes the retry_until check command with the parent
// registry's bash tool. The command text is model-controlled input (it arrives
// in the task tool's retry_until arguments), so it must pass the same
// pre-execution gating as a normal tool call — dispatcher.Check:
// PermissionRequest hooks → permission gate → PreToolUse hooks. A refusal
// returns the block reason wrapped in errCheckBlocked and the command is never
// executed. Without this gate the check would be an approval bypass: arbitrary
// shell reachable through a tool call that never consults the permission gate.
func (t *TaskTool) runCheckCommand(ctx context.Context, command string) (string, error) {
	bashTool, ok := t.parentReg.Get("bash")
	if !ok {
		return "", fmt.Errorf("bash tool not available for retry check")
	}
	args, _ := json.Marshal(map[string]string{"command": command})
	// Same gate instance the parent's dispatcher uses for ordinary bash calls
	// (boot wires headlessGate here), so deny rules, headless auto-allow, and
	// hardAsk semantics are identical to a normal bash approval. readOnly
	// mirrors bash's own classification (false — its effects can't be inferred
	// from args).
	cr := NewToolDispatcher(t.gate, t.hooks).Check(ctx, "bash", args, bashTool.ReadOnly())
	if !cr.Allowed {
		return cr.Reason, fmt.Errorf("%w: %s", errCheckBlocked, cr.Reason)
	}
	return bashTool.Execute(ctx, args)
}

// ── Transcript lifecycle (V10.29) ────────────────────────────────────

// prepareRun returns a SubagentRun for the given continue_from ref or creates
// a fresh one. Returns (nil, nil) when no transcript store is available
// (ephemeral mode). Rejects continue_from + run_in_background.
func (t *TaskTool) prepareRun(continueFrom string, runInBackground bool) (*SubagentRun, error) {
	continueFrom = strings.TrimSpace(continueFrom)
	if t.transcripts == nil {
		if continueFrom != "" {
			return nil, fmt.Errorf("subagent transcript store is not available; continue_from requires a persisted session")
		}
		return nil, nil // ephemeral mode
	}

	if continueFrom != "" {
		if runInBackground {
			return nil, fmt.Errorf("continue_from cannot be used with run_in_background")
		}
		return t.transcripts.PrepareContinue(continueFrom)
	}
	return t.transcripts.PrepareFresh(t.sysPrompt)
}

// finalizeRun persists the run result and appends the reference to the output.
func (t *TaskTool) finalizeRun(result string, err error, run *SubagentRun) (string, error) {
	if run == nil || run.Ref == "" {
		return result, err
	}
	if err != nil {
		_ = t.transcripts.SaveFailed(run)
		return result, err
	}
	if saveErr := t.transcripts.SaveCompleted(run); saveErr != nil {
		return "", fmt.Errorf("save subagent transcript: %w", saveErr)
	}
	result += FormatSubagentReference(run)
	return result, nil
}

func RunSubAgent(ctx context.Context, prov provider.LLMProvider, reg *tool.Registry, sysPrompt, prompt string, opts Options, sink event.Sink, subUsage *provider.Usage) (string, error) {
	return runSubAgentInternal(ctx, prov, reg, NewSession(sysPrompt), prompt, opts, sink, subUsage)
}

// RunSubAgentWithSession runs a sub-agent with an existing session (used for
// continue_from). Unlike RunSubAgent which creates a new session, this uses the
// provided session directly so the sub-agent continues from where it left off.
func RunSubAgentWithSession(ctx context.Context, prov provider.LLMProvider, reg *tool.Registry, sess *session.Session, prompt string, opts Options, sink event.Sink, subUsage *provider.Usage) (string, error) {
	return runSubAgentInternal(ctx, prov, reg, sess, prompt, opts, sink, subUsage)
}

// runSubAgentInternal is the shared sub-agent execution path: wire up an
// AgentRunner, run the prompt, and extract the final assistant message.
func runSubAgentInternal(ctx context.Context, prov provider.LLMProvider, reg *tool.Registry, sess *Session, prompt string, opts Options, sink event.Sink, subUsage *provider.Usage) (string, error) {
	// sub-agents don't need orchestrate verify — they execute a single task
	opts.DisableVerify = true
	sub := New(prov, reg, sess, opts, sink)
	_, runErr := sub.Run(ctx, prompt)
	// Populate subUsage from the sub-agent's last usage so SubUsage() reflects
	// real token counts for cost tracking.
	if subUsage != nil {
		if lu := sub.LastUsage(); lu != nil {
			*subUsage = *lu
		}
	}
	// V10.5: even on error, extract partial result from last assistant message
	lastMsg := extractLastAssistantMessage(sess.Messages)
	if runErr != nil {
		if lastMsg != "" {
			return lastMsg, fmt.Errorf("sub-agent terminated with error (partial result returned): %w", runErr)
		}
		return "", fmt.Errorf("sub-agent: %w", runErr)
	}
	if lastMsg != "" {
		return lastMsg, nil
	}
	return "", fmt.Errorf("sub-agent finished without producing a final answer")
}

// extractLastAssistantMessage finds the last non-empty assistant message
// in the session, traversing from the end. Returns "" if none found.
func extractLastAssistantMessage(msgs []provider.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role == provider.RoleAssistant && strings.TrimSpace(m.Content) != "" {
			return m.Content
		}
	}
	return ""
}

func NestedSink(ctx context.Context, fallback event.Sink) event.Sink {
	parentID, parent, _, ok := CallContext(ctx)
	if !ok || parent == nil {
		return fallback
	}
	return subSinkFor(parentID, parent)
}

func subSink(ctx context.Context) event.Sink {
	parentID, parent, _, ok := CallContext(ctx)
	if !ok || parent == nil {
		return event.Discard
	}
	return subSinkFor(parentID, parent)
}

func subSinkFor(parentID string, parent event.Sink) event.Sink {
	if parent == nil {
		return event.Discard
	}
	return event.FuncSink(func(e event.Event) {
		switch e.Kind {
		case event.ToolDispatch, event.ToolResult:
			e.Tool.ParentID = parentID
			e.Tool.ID = parentID + "/" + e.Tool.ID
			parent.Emit(e)
		case event.Usage:
			// Override source so StatsPanel can split main vs subagent.
			e.UsageSource = event.UsageSourceSubagent
			parent.Emit(e)
		}
	})
}
