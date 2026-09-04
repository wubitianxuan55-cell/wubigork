package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
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

// taskTitle 从子代理任务 prompt 提炼 UI 标题（首行、≤160 字符）。transcript
// 首条 user 消息是完整 prompt（含换行/上下文），标题只取首行避免侧栏刷屏。
func taskTitle(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if i := strings.IndexByte(prompt, '\n'); i >= 0 {
		prompt = strings.TrimSpace(prompt[:i])
	}
	r := []rune(prompt)
	if len(r) <= 160 {
		return prompt
	}
	return string(r[:159]) + "…"
}

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
	usageMu          sync.Mutex            // v4.63: 并行 task 调用时合并记账的互斥锁
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
	run, prepErr := t.prepareRun(ctx, p.ContinueFrom, p.RunInBackground)
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
			result, err := t.runSubSession(ctx, p.Prompt, subReg, subSink(ctx, run), run, maxSteps, p.OutputSchema)
			return t.finalizeRun(result, err, run)
		}
		parentID, parent, _, _ := CallContext(ctx)
		nested := subSinkFor(parentID, parent, func() string { return subagentRunRef(run) })
		label := p.Description
		if label == "" {
			label = "task"
		}
		// StartIn：嵌套派生自动挂父 job（终止级联；主回合派生无父=原行为）。
		job := jm.StartIn(ctx, "task", label, func(jobCtx context.Context, _ io.Writer) (string, error) {
			// S3 双空间：jobCtx 由 jobs.Manager 的 root（context.Background 派生）
			// 新建，不继承父调用 ctx 的 value——空间会在此丢失。显式补注父空间，
			// 后台子代理与前台一样继承（缺省 work）。
			result, runErr := t.runSubSession(WithSpace(jobCtx, SpaceFromContext(ctx)), p.Prompt, subReg, nested, run, maxSteps, p.OutputSchema)
			// 后台任务必须在此收尾 transcript（父 Execute 已返回，等不到回合末
			// finalizeRun 代跑；此前 store 模式下后台子代理从不落盘）。
			return t.finalizeRun(result, runErr, run)
		})
		return fmt.Sprintf("Started background task %q (%s). It runs across turns; collect its final answer with wait (or wait will return it once done), and you'll be notified when it finishes.", job.ID, label), nil
	}

	result, err := t.runSubSession(ctx, p.Prompt, subReg, subSink(ctx, run), run, maxSteps, p.OutputSchema)
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

// mergeSubUsage 并行安全地把一次子代理运行的用量并入累计值（v4.63）。
// 此前是整值覆写（最后一次运行赢）——串行派发时无感，多路并行会把其他
// 路的用量整段丢掉，且指针写本身是数据竞争。会话级累计字段（SessionCache*）
// 属 provider 侧逐会话口径，不在此合并。
func (t *TaskTool) mergeSubUsage(u *provider.Usage) {
	t.usageMu.Lock()
	defer t.usageMu.Unlock()
	if t.accumulatedUsage == nil {
		cp := *u
		t.accumulatedUsage = &cp
		return
	}
	a := t.accumulatedUsage
	a.PromptTokens += u.PromptTokens
	a.CompletionTokens += u.CompletionTokens
	a.TotalTokens += u.TotalTokens
	a.CacheHitTokens += u.CacheHitTokens
	a.CacheMissTokens += u.CacheMissTokens
	a.ReasoningTokens += u.ReasoningTokens
}

// SubagentFollowUpRunner 是用户侧追问执行器（v4.64 Side Chat 式追问）：
// 对一个已完结的 sa_ 运行追加一条用户消息并继续运行。boot 用 taskTool 的
// continue_from 管道组装，宿主保存后由 GaeaSubagentFollowUp 绑定调用。
type SubagentFollowUpRunner = func(ctx context.Context, ref, prompt string) error

// FollowUpSink 把追问运行的文本增量转成专用流式通道事件（SubagentText，
// wire-only），其余事件全部丢弃——追问的过程可见性由「tab 内流式打字 +
// transcript 快照轮询」承载，绝不进主对话账本（对齐 v4.62.2 的通道纪律）。
func FollowUpSink(ref string, onText func(text string)) event.Sink {
	return event.FuncSink(func(e event.Event) {
		if e.Kind == event.Text && onText != nil {
			onText(e.Text)
		}
	})
}

// RunFollowUp 对已完结的 sa_ 运行执行一次用户追问：追加 prompt 后继续运行
// （复用 continue_from 管道：PrepareContinue 拒绝 running/mt_/跨空间，
// MarkRunning + TrackProgress 维持 ~1s 快照，收尾 SaveCompleted/SaveFailed）。
// 结果不回投 SubagentMessage——追问的产出留在子代理会话 tab 内（Side Chat
// 语义：线程对主对话不可见），完成态由 meta 承载、前端轮询自校正。
// v4.66 失败可感知：后台失败原因摘要写回 meta（RecordFollowUpError），前端
// 凭轮询把乐观气泡转失败态，不再永久「等待中」；开跑即清旧值，失败态只属
// 于最近一次追问。
func (t *TaskTool) RunFollowUp(ctx context.Context, ref, prompt string, sink event.Sink) error {
	err := t.runFollowUp(ctx, ref, prompt, sink)
	if err != nil && t.transcripts != nil {
		// best-effort 写回失败摘要。此刻 runner 已返回（stop 先于终态写、
		// 绑定层同 ref 单飞），不再有并发 meta 写，本次写即该 meta 的最后一
		// 次——前端追问轮询必能读到，无需额外事件通道。
		_ = t.transcripts.RecordFollowUpError(ref, err.Error())
	}
	return err
}

func (t *TaskTool) runFollowUp(ctx context.Context, ref, prompt string, sink event.Sink) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	run, err := t.prepareRun(ctx, ref, false)
	if err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("subagent transcript store is not available")
	}
	defer run.Release()
	if err := t.transcripts.MarkRunning(run); err != nil {
		return fmt.Errorf("mark subagent running: %w", err)
	}
	// 开跑即清上一次追问的失败摘要：重试时前端不会把上一枪的失败误记到
	// 这一次头上（绑定层在受理时也同步清过一次，这里是管道自洽的兜底）。
	_ = t.transcripts.RecordFollowUpError(run.Ref, "")
	stop := t.transcripts.TrackProgress(run, 0)
	// stop 必须先于终态写（TrackProgress 契约）：其最终 flush 会把 Status
	// 写回 running，defer 到 SaveCompleted/SaveFailed 之后执行会把终态覆盖
	// 成 running（v4.64.0 回归）——追问后 meta 卡 running，再追问被
	// PrepareContinue 拒绝，tab 状态点也永远转不回完成态。
	stop()

	subReg := t.buildSubReg(nil)
	maxSteps := t.maxSteps / 2
	if maxSteps < 5 {
		maxSteps = 5
	}
	_, err = t.runSubSession(ctx, prompt, subReg, sink, run, maxSteps, nil)
	if err != nil {
		_ = t.transcripts.SaveFailed(run)
		return err
	}
	return t.transcripts.SaveCompleted(run)
}

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

	// v4.61：真机 transcript 实时化——每次子代理尝试开始时把 meta 置
	// running（Title 供 UI），并启动 1s 快照写盘；defer stop 先做最终 flush
	// 再退出，随后调用方的 SaveCompleted/SaveFailed 终态写不会被旧 running
	// 快照覆盖。无 store / 临时 run（Ref 空）时零开销 no-op。
	var stopProgress func()
	if run != nil && run.Ref != "" && t.transcripts != nil {
		if strings.TrimSpace(run.Title) == "" {
			run.Title = taskTitle(prompt)
		}
		_ = t.transcripts.MarkRunning(run)
		stopProgress = t.transcripts.TrackProgress(run, 0)
		defer stopProgress()
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
	if err == nil && strings.TrimSpace(result) != "" {
		// v4.26 对话流式重造（对标 Codex 2026-08 "Report completed sub-agent
		// activity on parent turns"）：子代理完成时把最终答复文本作为
		// SubagentMessage 事件回投父 sink。Why：子代理长跑期间主聊天只挂一张
		// task 卡、父级 Text/Reasoning 被有意隔离（上下文预算），用户在窗口里
		// 看不到子代理在产出什么；完成态回投让主聊天在子代理收尾时立即可见
		// 产出。取舍：只回投完成态 + 最终文本，中途进度（ reasoning/工具间
		// 文本）不回投——并行/重试子代理场景下逐段转发会刷屏，且父模型本来
		// 就只消费最终答复（结果结构不变，模型上下文零影响）。事件经
		// subSinkFor 透传时打点 ParentToolID（父 task 调用 ID），落盘走主会话
		// 日志 kind=subagent_message（旧日志无此 kind，读端跳过，兼容）。
		sink.Emit(event.Event{Kind: event.SubagentMessage, Text: result, SubagentRef: subagentRunRef(run)})
	}
	if err == nil && len(outputSchema) > 0 {
		// output_schema set: verify the result is parseable JSON.
		// We don't validate every field (full JSON Schema needs a lib),
		// but we confirm the sub-agent returned well-formed JSON.
		var parsed interface{}
		if json.Unmarshal([]byte(result), &parsed) != nil {
			result = "[output_schema: sub-agent returned non-JSON; parent should retry]" + "\n" + result
		}
		t.mergeSubUsage(&subUsage)
		return result, nil
	}
	if err == nil {
		t.mergeSubUsage(&subUsage)
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
		result, err := t.runSubSession(ctx, currentPrompt, subReg, subSink(ctx, run), run, maxSteps, outputSchema)
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
// (ephemeral mode). Rejects continue_from + run_in_background. The run
// context's space (SpaceFromContext) is handed to the store so a continue
// cannot cross spaces (S3 防穿越 C：请求空间与 ref 空间一致性校验).
func (t *TaskTool) prepareRun(ctx context.Context, continueFrom string, runInBackground bool) (*SubagentRun, error) {
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
		return t.transcripts.PrepareContinue(continueFrom, SpaceFromContext(ctx))
	}
	return t.transcripts.PrepareFresh(t.sysPrompt, SpaceFromContext(ctx))
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
	sess := NewSession(sysPrompt)
	// S3 双空间：新建子会话继承父运行上下文空间（缺省 work，空值已被
	// withSpace 归一）。空间一致性由 runSubAgentInternal 的 fail-closed 断言兜底。
	sess.SetSpace(SpaceFromContext(ctx))
	return runSubAgentInternal(ctx, prov, reg, sess, prompt, opts, sink, subUsage)
}

// RunSubAgentWithSession runs a sub-agent with an existing session (used for
// continue_from). Unlike RunSubAgent which creates a new session, this uses the
// provided session directly so the sub-agent continues from where it left off.
// S3 双空间：装载的子会话若无空间自描述（空值）则标记父运行上下文空间；
// 已带合法空间且与 ctx 不一致的会话不改写——由 runSubAgentInternal 的
// fail-closed 断言拒绝，防止跨空间续写。
func RunSubAgentWithSession(ctx context.Context, prov provider.LLMProvider, reg *tool.Registry, sess *session.Session, prompt string, opts Options, sink event.Sink, subUsage *provider.Usage) (string, error) {
	if sess != nil && !spaces.Valid(sess.Space()) {
		sess.SetSpace(SpaceFromContext(ctx))
	}
	return runSubAgentInternal(ctx, prov, reg, sess, prompt, opts, sink, subUsage)
}

// runSubAgentInternal is the shared sub-agent execution path: wire up an
// AgentRunner, run the prompt, and extract the final assistant message.
func runSubAgentInternal(ctx context.Context, prov provider.LLMProvider, reg *tool.Registry, sess *Session, prompt string, opts Options, sink event.Sink, subUsage *provider.Usage) (string, error) {
	// S3 防穿越 A（fail-closed）：子会话空间必须与运行上下文空间一致。
	// 两条继承路径（RunSubAgent 新建 / RunSubAgentWithSession 装载）都汇入
	// 这里，断言是防穿越的单点闸门：任何绕过继承注入、携带异空间会话到达
	// 此处的调用一律报错拒绝，绝不带着错误空间运行子代理。
	if sess != nil {
		if space := SpaceFromContext(ctx); sess.Space() != space {
			return "", fmt.Errorf("subagent space mismatch: session space %q != context space %q (fail-closed)", sess.Space(), space)
		}
	}
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
	return subSinkFor(parentID, parent, nil)
}

func subSink(ctx context.Context, run *SubagentRun) event.Sink {
	parentID, parent, _, ok := CallContext(ctx)
	if !ok || parent == nil {
		return event.Discard
	}
	return subSinkFor(parentID, parent, func() string { return subagentRunRef(run) })
}

// subSinkFor 把子代理事件选择性透传父 sink。refSrc 返回当前子代理 transcript
// 引用（"sa_..."）；非空时内层 Text 增量被转标为 SubagentText 透传（P1 逐
// token 流式，SubagentThread 会话 tab 实时渲染），为空（临时子代理/测试）
// 维持既有行为——Text 一律丢弃，避免过程噪音进入主聊天。
func subSinkFor(parentID string, parent event.Sink, refSrc func() string) event.Sink {
	if parent == nil {
		return event.Discard
	}
	return event.FuncSink(func(e event.Event) {
		switch e.Kind {
		case event.Text:
			ref := ""
			if refSrc != nil {
				ref = refSrc()
			}
			if ref == "" {
				return // 无消费方（无会话 tab），维持有意丢弃
			}
			parent.Emit(event.Event{Kind: event.SubagentText, Text: e.Text, SubagentRef: ref, ParentToolID: parentID})
		case event.SubagentText:
			// 技能子代理路径（RunPersistedSubAgent 在 sink 外层转标 ref）：
			// 这里补打父 task 调用 ID 后透传。
			e.ParentToolID = parentID
			parent.Emit(e)
		case event.ToolDispatch, event.ToolResult:
			e.Tool.ParentID = parentID
			e.Tool.ID = parentID + "/" + e.Tool.ID
			parent.Emit(e)
		case event.Usage:
			// Override source so StatsPanel can split main vs subagent.
			e.UsageSource = event.UsageSourceSubagent
			parent.Emit(e)
		case event.SubagentMessage:
			// v4.26：子代理完成回投透传父 sink，打点父 task 调用 ID——前端
			// 据此把答复文本挂到对应 task 卡片下（与子工具嵌套同键位）。
			// 其余 kind（Reasoning/Turn* 等）维持有意丢弃，避免子代理
			// 过程噪音进入主聊天。
			e.ParentToolID = parentID
			parent.Emit(e)
		}
	})
}

// subagentRunRef 返回子代理 transcript 引用（run 为 nil（临时子代理）或无
// ref 时返回空串——wire/log payload 中该字段省略）。
func subagentRunRef(run *SubagentRun) string {
	if run == nil {
		return ""
	}
	return run.Ref
}
