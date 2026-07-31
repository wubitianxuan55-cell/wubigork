package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/event"
	"github.com/wubigork/wubigork/internal/gaea/provider"
	"github.com/wubigork/wubigork/internal/gaea/tool"

	"github.com/wubigork/wubigork/internal/gaea/cache"
)

// HermesPrompt steers the planner toward research-backed plans.
// V10.33: planWithTools is now the sole plan path — planStream is the
// backward-compatible fallback when readonlyTools is nil (e.g. test harness).
const HermesPrompt = `You are Hermes — the planner in a two-model engineering office assistant.
Given a task, produce a concise, ordered plan for the Hephaestus executor to carry out.
Investigate project files with read-only tools. For text/code files use read_file.
For structured data use the dedicated parser: csv_parse for CSV, docx_read for Word,
pdf_extract for PDF, xlsx_read for Excel, format_convert for doc-to-markdown.
Use web_search/web_fetch for external research. Use ls to explore directories.
Keep research targeted — stop once you have enough evidence.
If a file type has no dedicated parser, use format_convert as fallback, then read_file.

Structured data handling (engineering office context):
- CSV/XLSX: use csv_parse or xlsx_read to inspect the first few rows — confirm encoding (UTF-8/GB2312/GBK), delimiter, column structure — before any analysis.
- DOCX/PDF: use docx_read or pdf_extract to extract text; use format_convert for doc-to-markdown when clean Markdown is needed.
- Encoding fallback: Chinese engineering files commonly use GB2312/GBK. If text appears garbled, try explicit GBK/GB18030 decode.
- Data validation: before processing test/measurement data, check for plausibility (range, units, missing values). Use ask to confirm suspicious values.
- No dedicated parser: use format_convert as a general fallback, then read_file.

Design quality — every plan must respect these principles:
1. Evidence over assumptions — engineering conclusions must cite actual data, spec queries, or code evidence; never rely on user's verbal claim alone.
2. Push back when needed — if a request violates engineering standards (wrong GB standard, out-of-range parameters, unit mismatch), flag the issue and propose a correct alternative.
3. Clarify, don't guess — when requirements are vague (e.g. "generate a report" without report type or phase), use the ask tool for a targeted question; never fill in assumptions.
4. Simpler is better — if an existing skill or template solves the problem, use it; do not design a new workflow.
5. Never agree to a bad plan — if no compliant solution exists, say "this won't work because X" — that is more valuable than a flawed plan.
6. Design quality — each step has single responsibility; don't over-design (YAGNI); keep it simple (KISS).
7. Every step needs a verifiable success criterion — a test, build check, generated artifact, or observable result. (This criterion becomes the evidence Hephaestus submits to complete_step — make it concrete and verifiable, not abstract.)

Engineering standards (mandatory pre-requisites):
- Before any exceedance judgment: call spec_judge to compare against GB 36600-2018 (soil) or GB 15618-2018 (agricultural land).
- Before writing any engineering proposal: call spec_query to consult the corresponding HJ technical guideline (HJ 25.1~25.6).
- All engineering conclusions must cite the specific standard number — never rely on experience alone.
- When the user's referenced standard does not match the data (e.g. agricultural standard used for建设用地 data), flag the mismatch and suggest the correct standard.
- SI units and 3-decimal precision (from AGENTS.md configuration) must be followed in all plans involving engineering computation.

About Hephaestus (your executor):
Your plan's target audience is Hephaestus, an executor agent with the following workflow:
- **Execute → Verify → Sign-off**: Hephaestus executes each step, actively verifies results (tests, build checks, spec validation), then calls complete_step with evidence from actual tool output.
- **Evidence-driven sign-off**: complete_step requires concrete evidence — command results, file diffs, spec query outputs. Pure "manual" claims are rejected. Your success criterion for each step should describe what verifiable evidence proves completion.
- **Error recovery**: Hephaestus retries failed tools with corrections, falls back to alternatives, and uses ask for user decisions after 3 consecutive failures.
- **No re-planning**: Hephaestus will not re-plan or wait for approval — the plan is already confirmed by the user. It will adapt details but not change scope.

Your plan describes WHAT to do: task breakdown, target files, key decisions, approach,
and constraints.
NEVER write code blocks, function bodies, class definitions, or file
contents. If a design decision requires a signature or pseudo-code, keep it to a
one-line signature at most.
contents. If a design decision requires a signature or pseudo-code, keep it to a
one-line signature at most.

If the task is a read-only query, answer directly — do not produce a plan.

If the task is a purely operational task — building, starting, testing, formatting,
committing, installing dependencies — your FIRST AND ONLY action must be to output
<!--plan--> followed by the command(s). Do NOT use any tools — no read_file, no ls,
no grep, no web_search. The task is already fully specified. Examples:
- "build the project" → <!--plan-->\nRun: wails build
- "run tests" → <!--plan-->\nRun: go test ./...
- "commit changes" → <!--plan-->\nRun: git add -A && git commit -m "..."


If you need to clarify scope or ask the user a question, you MUST use the ask tool.
Never output a question as plain text — that ends your turn immediately and forces
a full restart of the planning cycle on the next turn. Put <!--plan--> in your
output only when you have a concrete executable plan ready.
When you have a concrete executable plan ready, start it with <!--plan--> on its own line.
Never include <!--plan--> in a question, clarification, or direct answer.
When you receive a message prefixed with [上一轮执行结果], it is a reliable summary of Hephaestus'
execution from the previous turn. Use it to understand what happened — trust its file-modification
list, error messages, and summary. Do not re-read files unless the summary contradicts itself
or indicates errors that require deeper investigation.`

// HephaestusPrompt steers the executor toward reliable, verifiable execution.
const HephaestusPrompt = `## 角色与原则
你是 Hephaestus — 负责将 Hermes 规划转化为实际成果的执行者。你的职责是按计划执行并验证，而非重新规划。

About Hermes (your planner):
- Hermes is a read-only researcher — it can read files, query specs, search the web, but cannot write, edit, or execute anything.
- Hermes produces structural plans (WHAT to do), not implementations. Any code-like fragments in the plan are conceptual pseudo-code — implement them yourself.
- Hermes' design principles: evidence over assumptions, may push back on unsound requests, uses ask for clarification on vague requirements.
- Feedback loop: after you complete execution, your results (files created/modified, errors, summary) are fed back to Hermes as [上一轮执行结果]. Your complete_step evidence and error messages become the starting point for Hermes' next planning turn — be thorough and accurate.

## 执行-验证-签退 三阶段工作流
2. **验证**：每步完成后主动自检。代码任务跑测试或编译检查，文档任务核对规范引用和格式，数据任务核对单位、精度和完整性。（The verification should directly address the success criterion stated in Hermes' plan step.）
3. **签退**：验证通过后才调用 complete_step 标记完成。证据必须来自实际工具输出（命令结果、文件内容、规范查询结果），不接受纯 manual 声明。

## 错误恢复模式
- 工具报错时：先读错误信息 → 诊断原因 → 修正参数重试 → 换替代工具尝试 → 仍失败用 ask 呈现问题给用户决策
- 静默跳过失败步骤是不可接受的
- 连续 3 次同类失败时，使用 ask 让用户决策，不要死循环

## 工程办公领域质量检查点
- **规范引用**：spec_judge/spec_query 结果必须与方案结论一致，注明标准编号和条款号
- **单位与精度**：统一 SI 单位制，保留 3 位小数，标注单位；不混用 SI/Imperial
- **数据完整性**：CSV/XLSX 先用对应解析器确认编码和结构再处理，检测数据标注检出限
- **文档结构**：报告/方案必须使用对应工具（survey_report/imple_plan/bid_proposal）生成框架后填充

## 禁止事项
- 不要用纯文本提问 — 使用 ask 工具
- 不要重新规划或等待批准 — 计划已由用户确认
- 不要批量签退 — 一任务一 complete_step
- 不要在无验证的情况下签退`

const hephaestusHandoffMarker = "gaea hephaestus handoff"

// Hermes runs two models in separate sessions to keep each one's prompt
// prefix cache-stable: a low-frequency planner proposes an approach, then the
// executor (a full tool-using AgentRunner) carries it out. The sessions never
// mix, so neither model's prefix is disturbed by the other's turns.
//
// V10.32: when readonlyTools is set, the planner uses AgentRunner with
// read-only tools (read_file/grep/glob/web_search/...) so it can investigate
// the codebase before proposing a plan. planMaxSteps caps planner turns.
type Hermes struct {
	hermesProvider provider.Provider
	hermesSess     *Session
	hermesSystem   string
	hermesPricing  *provider.Pricing
	hephaestus     *AgentRunner
	temperature    float64
	sink           event.Sink

	readonlyTools *tool.Registry // V10.32: if set, planner runs as AgentRunner
	planMaxSteps  int            // max planner tool-call turns (<=0 = unlimited)
	asker         Asker          // V10.34: interactive plan confirmation (nil = auto-confirm)

	// V10.36: persistent planner Agent with compaction — replaces per-turn temp AgentRunner.
	// The planner accumulates planning history + execution results across turns, with
	// compaction keeping the context bounded. This gives the planner a proper TCCA-like
	// architecture (L1 stable prefix + L4 growing flow + compaction).
	plannerAgent *AgentRunner
}

// NewHermes creates a Hermes orchestrator. hermesProvider is the planning model,
// hephaestus is the execution AgentRunner. sink receives events from both.
//
// V10.32: pass readonlyTools (nil for stream-only) and planMaxSteps to let
// Hermes use read-only tools for code investigation before proposing a plan.
// V10.36: contextWindow + archiveDir enable compaction on the planner's persistent session.
func NewHermes(hermesProvider provider.Provider, hermesSession *Session, hermesPricing *provider.Pricing, hephaestus *AgentRunner, temperature float64, sink event.Sink, readonlyTools *tool.Registry, planMaxSteps int, contextWindow int, archiveDir string) *Hermes {
	if hermesSession == nil {
		hermesSession = NewSession("")
	}
	hermesSystem := sessionSystemPrompt(hermesSession)
	h := &Hermes{
		hermesProvider: hermesProvider,
		hermesSess:     hermesSession,
		hermesSystem:   hermesSystem,
		hermesPricing:  hermesPricing,
		hephaestus:     hephaestus,
		temperature:    temperature,
		sink:           sink,
		readonlyTools:  readonlyTools,
		planMaxSteps:   planMaxSteps,
	}
	// V10.36: create persistent planner Agent with compaction so the planner
	// accumulates history across turns without unbounded growth.
	if readonlyTools != nil {
		plannerSink := event.FuncSink(func(e event.Event) {
			// Suppress TurnStarted from the planner agent — Hermes
			// already started the turn (line 150). A redundant
			// TurnStarted would reset perTurnPlannerUsage in the
			// frontend, zeroing out the planner's cost stats.
			if e.Kind == event.TurnStarted {
				return
			}
			if e.Kind == event.Usage {
				if e.UsageSource == "" || e.UsageSource == event.UsageSourceExecutor {
					e.UsageSource = event.UsageSourcePlanner
				}
			}
			sink.Emit(e)
		})
		h.plannerAgent = New(hermesProvider, readonlyTools, hermesSession, Options{
			MaxSteps:      planMaxSteps,
			Temperature:   temperature,
			Pricing:       hermesPricing,
			Gate:          &autoGate{},
			DisableVerify: true,
			PlannerMode:   true,
			ContextWindow: contextWindow,
			Compaction:    CompactionConfig{ArchiveDir: archiveDir, Window: contextWindow},
		}, plannerSink)
	}
	return h
}

func sessionSystemPrompt(s *Session) string {
	if s == nil {
		return ""
	}
	for _, m := range s.Snapshot() {
		if m.Role == provider.RoleSystem {
			return m.Content
		}
	}
	return ""
}

// ResetSession discards turn-local planner history when switching
// executor sessions. Carrying the old Hermes transcript across sessions
// can make the next plan reuse unrelated tasks.
func (h *Hermes) ResetSession() {
	if h == nil {
		return
	}
	system := h.hermesSystem
	if system == "" {
		system = sessionSystemPrompt(h.hermesSess)
	}
	h.hermesSess = NewSession(system)
	// V10.??: sync plannerAgent's session pointer so planWithTools uses the
	// new session instead of the stale one from NewHermes.
	if h.plannerAgent != nil {
		h.plannerAgent.SetSession(h.hermesSess)
	}
}

// PlannerContext returns the planner agent's last usage and context window,
// for the status bar's per-model context gauge.
func (h *Hermes) PlannerContext() (used int, window int) {
	if h == nil || h.plannerAgent == nil {
		return 0, 0
	}
	u := h.plannerAgent.LastUsage()
	if u == nil {
		return 0, h.plannerAgent.ContextWindow()
	}
	return u.PromptTokens, h.plannerAgent.ContextWindow()
}

// SetAsker installs the interactive asker for plan confirmation (V10.34).
// nil means headless mode — plans auto-confirm without user approval.
// Also wires the asker into the plannerAgent so it can ask clarifying questions
// during planning (scope negotiation, detail gathering).
func (h *Hermes) SetAsker(a Asker) {
	h.asker = a
	if h.plannerAgent != nil {
		h.plannerAgent.SetAsker(a)
	}
}

// Run plans with the planner model, then hands the plan to the executor.
// Returns a merged TurnResult combining the planner's and executor's outcomes.
func (h *Hermes) Run(ctx context.Context, input string) (*TurnResult, error) {
	h.sink.Emit(event.Event{Kind: event.TurnStarted})

	// V10.31: fast path — "!" prefix skips planner and executes directly.
	if task, ok := shouldSkipPlanner(input); ok {
		h.sink.Emit(event.Event{Kind: event.Phase, Text: h.hephaestus.ProvName() + " · 快速执行"})
		return h.hephaestus.Run(ctx, task)
	}

	// V10.XX: simple query detection — short read-only inputs trigger an ask
	// confirmation so the user can choose whether to skip the planner.
	if cache.IsSimpleQuery(strings.ToLower(input), input) {
		if skip, err := h.confirmSimpleTask(ctx, input); err != nil {
			return nil, err
		} else if skip {
			h.sink.Emit(event.Event{Kind: event.Phase, Text: h.hephaestus.ProvName() + " · 简单查询直接执行"})
			return h.hephaestus.Run(ctx, input)
		}
		// User chose normal planning — fall through.
	}

	var userNote, plan string
	var planErr error
	prePlanLen := len(h.hermesSess.Messages)
	originalTask := input // V10.??: preserve original user input for clean handoff across replan cycles
	// V10.??: replan loop — user clicks "按用户意见修改计划" to revise the plan
	// with feedback, then the new plan goes through confirmation again.
	for {
		// V10.??: capture session length before each plan call; on error rollback partial tool-call messages
		planPreLen := len(h.hermesSess.Messages)
		plan, planErr = h.plan(ctx, input)
		if planErr != nil {
			h.hermesSess.Truncate(planPreLen)
			return nil, fmt.Errorf("hermes: %w", planErr)
		}
		if isAnswerNotAction(plan) {
			// Hermes answered directly — no Hephaestus needed.
			// Text has already been streamed by planWithTools/planStream; emitting
			// the full plan again here would duplicate the output.
			h.persistAnswer(input, plan)
			return &TurnResult{Summary: plan, Success: true}, nil
		}

		var chatOnly, revise bool
		userNote, chatOnly, revise, planErr = h.confirmPlan(ctx, input, plan)
		if planErr != nil {
			// User cancelled — roll back planner session to pre-plan state.
			h.hermesSess.Truncate(prePlanLen)
			return nil, planErr
		}
		if chatOnly {
			// User chose "仅聊天" — treat as direct answer, don't dispatch executor.
			h.persistAnswer(input, plan)
			return &TurnResult{Summary: plan, Success: true}, nil
		}
		if revise {
			// User chose "按用户意见修改计划" — append feedback and re-plan.
			if userNote != "" {
				input = originalTask + "\n\n—— User feedback ——\n" + userNote
			}
			prePlanLen = len(h.hermesSess.Messages) // new baseline for next round
			continue
		}
		break // execute with Hephaestus
	}
	h.sink.Emit(event.Event{Kind: event.Phase, Text: h.hephaestus.ProvName() + " · Hephaestus"})
	// Suppress the executor's TurnStarted — Hermes already started the turn.
	// Without this, the redundant TurnStarted resets perTurnPlannerUsage in the
	// frontend, zeroing out the planner's cost stats.
	execSink := h.hephaestus.Sink()
	h.hephaestus.SetSink(event.FuncSink(func(e event.Event) {
		if e.Kind == event.TurnStarted {
			return
		}
		execSink.Emit(e)
	}))
	execResult, execErr := h.hephaestus.Run(ctx, formatHandoff(originalTask, plan, userNote))

	// V10.37: executor returns structured TurnResult — no more post-hoc extraction.
	// Flow the structured result back into the planner's session so it has context
	// for the next turn.
	if execResult != nil && execResult.Summary != "" {
		h.hermesSess.Add(provider.Message{
			Role:    provider.RoleUser,
			Content: formatExecutionFeedback(execResult),
		})
	} else if execErr != nil {
		h.hermesSess.Add(provider.Message{
			Role:    provider.RoleUser,
			Content: "[上一轮执行结果] errors\nErrors: " + execErr.Error(),
		})
	}
	return execResult, execErr
}

// formatExecutionFeedback converts a TurnResult into a structured summary
// for injection into the planner's session so the planner knows what happened.
func formatExecutionFeedback(r *TurnResult) string {
	var b strings.Builder
	b.WriteString("[上一轮执行结果]")
	if r.Success {
		b.WriteString(" success")
	} else {
		b.WriteString(" errors")
	}
	b.WriteString("\n")
	if len(r.FilesCreated) > 0 {
		b.WriteString("Created: ")
		b.WriteString(strings.Join(r.FilesCreated, ", "))
		b.WriteString("\n")
	}
	if len(r.FilesModified) > 0 {
		b.WriteString("Modified: ")
		b.WriteString(strings.Join(r.FilesModified, ", "))
		b.WriteString("\n")
	}
	if len(r.Errors) > 0 {
		b.WriteString("Errors: ")
		b.WriteString(strings.Join(r.Errors, "; "))
		b.WriteString("\n")
	}
	if r.Summary != "" {
		b.WriteString("Summary: ")
		b.WriteString(r.Summary)
	}
	return b.String()
}

// confirmPlan asks the user to approve the planner's output before handing off to
// the executor. Returns the user's free-typed note ("" when none), a chatOnly
// flag, and a revise flag (= user clicked "按用户意见修改计划"), and an error on
// cancellation. In headless mode (asker == nil) it auto-confirms.
//
// The confirmation dialog shows:
//
//	○ 提交执行          — 同意计划，直接交由 Hephaestus 执行
//	○ 仅聊天            — 计划误触发，仅作为普通对话回复，不派送执行者
//	○ 按用户意见修改计划   — 将修改意见送回 Hermes 重新规划
//	○ 取消              — 放弃本次任务
//	📝 文本框 — 输入修改意见
//
// For "按用户意见修改计划", the note text is extracted from Selected[1] (when
// available) and returned as the first string so the caller can feed it back
// to Hermes for re-planning.
func (h *Hermes) confirmPlan(ctx context.Context, task, plan string) (note string, chatOnly bool, revise bool, err error) {
	if h.asker == nil {
		return "", false, false, nil // headless: auto-confirm
	}
	answers, err := h.asker.Ask(ctx, []event.AskQuestion{{
		ID:     "confirm",
		Header: "计划确认",
		Prompt: fmt.Sprintf("任务：%s", truncateStr(task, 200)),
		Plan:   plan, // full plan rendered by PlanCard with Markdown
		Options: []event.AskOption{
			{Label: "提交执行", Description: "按计划交由 Hephaestus 立即执行"},
			{Label: "仅聊天", Description: "计划误触发，仅作为普通对话回复，不派送执行者"},
			{Label: "按用户意见修改计划", Description: "将修改意见送回 Hermes 重新规划"},
			{Label: "取消", Description: "放弃本次任务，不做任何更改"},
		},
	}})
	if err != nil {
		return "", false, false, fmt.Errorf("plan confirmation cancelled: %w", err)
	}
	if len(answers) == 0 || len(answers[0].Selected) == 0 {
		return "", false, false, fmt.Errorf("计划被取消（无回复）")
	}
	selected := answers[0].Selected[0]
	switch selected {
	case "提交执行":
		return "", false, false, nil // agree without notes
	case "仅聊天":
		return "", true, false, nil // chat-only: don't dispatch to executor
	case "按用户意见修改计划":
		feedback := ""
		if len(answers[0].Selected) > 1 {
			feedback = answers[0].Selected[1]
		}
		return feedback, false, true, nil // revise: re-plan with feedback
	case "取消":
		return "", false, false, fmt.Errorf("计划被用户取消")
	default:
		// Free-typed text in the input box: agree with user notes
		return selected, false, false, nil
	}
}

// confirmSimpleTask asks the user whether to skip the planner for a short,
// read-only query detected by classifyIntent. Returns true when the user
// chooses to bypass Hermes entirely. In headless mode (asker == nil) it
// defaults to false — without a user to confirm, we never skip.
func (h *Hermes) confirmSimpleTask(ctx context.Context, task string) (bool, error) {
	if h.asker == nil {
		return false, nil // headless: never skip without user confirmation
	}
	answers, err := h.asker.Ask(ctx, []event.AskQuestion{{
		ID:     "simple-task",
		Header: "简单查询",
		Prompt: fmt.Sprintf("检测到简短查询，是否跳过规划者直接执行？\n\n任务：%s", truncateStr(task, 200)),
		Options: []event.AskOption{
			{Label: "跳过规划，直接执行", Description: "不经过 Hermes 规划，直接交由 Hephaestus 执行"},
			{Label: "正常规划", Description: "按正常流程先规划再执行"},
			{Label: "取消", Description: "放弃本次任务"},
		},
	}})
	if err != nil {
		return false, fmt.Errorf("simple-task confirmation cancelled: %w", err)
	}
	if len(answers) == 0 || len(answers[0].Selected) == 0 {
		return false, fmt.Errorf("简单查询被取消（无回复）")
	}
	switch answers[0].Selected[0] {
	case "跳过规划，直接执行":
		return true, nil
	case "正常规划":
		return false, nil
	case "取消":
		return false, fmt.Errorf("任务被用户取消")
	default:
		return false, nil
	}
}

// ── Plan implementation ──────────────────────────────────────────────────

// plan runs Hermes as an AgentRunner with read-only tools so it can investigate
// the codebase before proposing a plan. Falls back to planStream (zero-tool stream)
// when readonlyTools is nil — e.g. in tests or when no read-only registry is wired.
func (h *Hermes) plan(ctx context.Context, input string) (string, error) {
	// V10.32+: AgentRunner mode — planner can call read-only tools.
	// planMaxSteps <= 0 means unlimited (rely on model to stop itself).
	if h.readonlyTools != nil && h.planMaxSteps >= 0 {
		return h.planWithTools(ctx, input)
	}
	return h.planStream(ctx, input)
}

// planStream is the backward-compatible zero-tool stream fallback, used when
// Hermes is constructed without a read-only tool registry (e.g. in tests).
func (h *Hermes) planStream(ctx context.Context, input string) (string, error) {
	msgs := make([]provider.Message, len(h.hermesSess.Messages)+1)
	copy(msgs, h.hermesSess.Messages)
	msgs[len(msgs)-1] = provider.Message{Role: provider.RoleUser, Content: input}

	ch, err := h.hermesProvider.Stream(ctx, provider.Request{
		Messages:    msgs,
		Temperature: h.temperature,
	})
	if err != nil {
		return "", err
	}

	var text strings.Builder
	var usage *provider.Usage
	for chunk := range ch {
		switch chunk.Type {
		case provider.ChunkText:
			text.WriteString(chunk.Text)
			h.sink.Emit(event.Event{Kind: event.Text, Text: chunk.Text})
		case provider.ChunkUsage:
			usage = chunk.Usage
		case provider.ChunkError:
			return "", chunk.Err
		}
	}
	h.sink.Emit(event.Event{Kind: event.Usage, Usage: usage, Pricing: h.hermesPricing, UsageSource: event.UsageSourcePlanner})

	plan := text.String()
	// Persist the conversation in hermesSess so planStream behaves consistently
	// with planWithTools. The replan loop (Run) uses Truncate(prePlanLen) to
	// roll back these messages when the user cancels or revises.
	h.hermesSess.Add(provider.Message{Role: provider.RoleUser, Content: input})
	h.hermesSess.Add(provider.Message{Role: provider.RoleAssistant, Content: plan})
	return plan, nil
}

// planWithTools runs the persistent planner Agent with read-only tools.
// V10.36: uses the persistent plannerAgent (created in NewHermes) instead of
// building a temporary AgentRunner each turn. The planner's session accumulates
// planning history + execution results across turns; compaction keeps it bounded.
func (h *Hermes) planWithTools(ctx context.Context, input string) (string, error) {
	if h.plannerAgent == nil {
		return "", fmt.Errorf("hermes: planner agent not initialized (no read-only tools)")
	}
	if _, err := h.plannerAgent.Run(ctx, input); err != nil {
		return "", fmt.Errorf("hermes: %w", err)
	}

	// Extract the plan from the planner's persistent session (last assistant message).
	var plan string
	msgs := h.hermesSess.Messages
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == provider.RoleAssistant {
			plan = msgs[i].Content
			break
		}
	}
	if plan == "" {
		plan = "(hermes produced no output)"
	}
	// NOTE: <!--plan--> marker is not stripped — it's an HTML comment, invisible
	// in rendered Markdown (PlanCard) and harmless in executor prompts.

	return plan, nil
}

// autoGate approves every tool call — safe for read-only planners.
type autoGate struct{}

func (g *autoGate) Check(_ context.Context, _ string, _ json.RawMessage, _ bool) (bool, string, error) {
	return true, "", nil
}

func (h *Hermes) persistAnswer(input, plan string) {
	if h == nil || h.hephaestus == nil || h.hephaestus.session == nil {
		return
	}
	h.hephaestus.session.Add(provider.Message{Role: provider.RoleUser, Content: input})
	h.hephaestus.session.Add(provider.Message{Role: provider.RoleAssistant, Content: plan})
}

// ── Plan helpers ─────────────────────────────────────────────────────

// shouldSkipPlanner detects the "!" fast-path marker in user input.
// When present, the stripped task bypasses Hermes and goes straight to Hephaestus.
// V10.34: only the explicit "!" marker skips the planner — simple and read-only
// tasks go through Hermes for direct answers instead of bypassing it.
// Heuristic keyword matching removed: Hermes is better at classifying tasks
// than a fixed keyword list, and the direct-answer path costs one planner call.
func shouldSkipPlanner(input string) (string, bool) {
	if stripped, ok := strings.CutPrefix(input, "!"); ok {
		return strings.TrimSpace(stripped), true
	}
	return "", false
}

// isAnswerNotAction checks whether the planner's output is a direct answer
// that needs no executor. The planner self-marks executable plans with
// <!--plan--> — if absent, Hermes answered directly. No length short-circuit:
// even short plans with the <!--plan--> marker trigger confirmation.
func isAnswerNotAction(plan string) bool {
	trimmed := strings.TrimSpace(plan)
	// <!--plan--> marks executable plans; absent means direct answer.
	return !strings.Contains(trimmed, "<!--plan-->")
}

func formatHandoff(task, plan, userNote string) string {
	note := ""
	if userNote != "" {
		note = fmt.Sprintf("\n\n📌 User note (written during plan confirmation):\n%s\n", userNote)
	}
	return fmt.Sprintf(`# %s

You are Hephaestus now. Use your available tools to execute the task.

Original task:
%s

Hermes output:
%s%s

Carry out the task, adapting the plan as needed.`, hephaestusHandoffMarker, task, plan, note)
}

// HandoffTask returns the original user task embedded in an executor handoff
// message, or s unchanged when it is not one. Session previews and auto-titles
// use it so dual-model sessions surface the user's words, not the handoff
// boilerplate.
func HandoffTask(s string) string {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "# "+hephaestusHandoffMarker) {
		return s
	}
	const header = "Original task:\n"
	i := strings.Index(trimmed, header)
	if i < 0 {
		return s
	}
	rest := trimmed[i+len(header):]
	if j := strings.Index(rest, "\n\nHermes output:"); j >= 0 {
		rest = rest[:j]
	}
	if task := strings.TrimSpace(rest); task != "" {
		return task
	}
	return s
}
