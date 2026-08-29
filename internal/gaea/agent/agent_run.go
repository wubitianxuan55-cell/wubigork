package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

func (a *AgentRunner) Run(ctx context.Context, input string) (*TurnResult, error) {
	return a.runDirect(ctx, input)
}

// runDirect is the original single-model execution path.
func (a *AgentRunner) runDirect(ctx context.Context, input string) (*TurnResult, error) {
	// generate trace ID for this turn
	traceID := NewTraceID()
	ctx = WithTraceID(ctx, traceID)
	// S3 双空间：把会话空间注入运行上下文（组装运行上下文点，与 trace 同位）。
	// 工具（task / run_skill 等）经 SpaceFromContext 读取并把空间继承给子代理
	// 会话；空 space（space.mode=off 平铺形态）归一为 work，行为与改造前一致。
	// 子代理 runSubAgentInternal 会在自己的 runDirect 再盖一次同一空间（其会话
	// 已被断言与 ctx 一致），不产生漂移。
	ctx = WithSpace(ctx, a.session.Space())
	defer func() { a.steerMu.Lock(); a.steerQueue = nil; a.steerConsumed = false; a.steerMu.Unlock() }() // drain any remaining steer on turn exit

	if a.evidence != nil {
		a.evidence.Reset()
	}
	a.sink.Emit(event.Event{Kind: event.TurnStarted})
	// wrap user input with transient language preference blocks
	// (Design adopted from DeepSeek-Reasonix-V1.12)
	input = a.withTurnPreferences(input)
	a.session.Add(provider.Message{Role: provider.RoleUser, Content: input})

	// rebuild canonical todo state from session history
	// (Design adopted from DeepSeek-Reasonix-V1.12)
	a.rebuildTodoState(a.session.Messages)

	// P0-1: reset tool filter from previous turn (prefix must be immutable).
	a.activeSchemasMu.Lock()
	a.activeSchemas = nil
	a.activeSchemasMu.Unlock()

	// reset pre-execution cache and tool result cache for new turn
	a.preMu.Lock()
	a.preOutcomes = make(map[string]toolOutcome)
	a.dedupHashes = nil // P0-2: reset dedup hashes each turn
	a.steerCount = 0    // P0-3: reset steer counter each turn
	a.preMu.Unlock()
	// reset per-turn bg/stale/repeat state under turnMu — these fields are
	// written by executeOne's parallel goroutines (audit P0 race fix), so the
	// reset must serialize against them, not just against pre-execution.
	a.turnMu.Lock()
	a.bgJobStartedThisTurn = false // 每轮重置启停标志
	a.bgOutputReadThisTurn = false
	a.bgJobKilledThisTurn = false
	a.bgStartKillStreak = 0   // 新用户轮次重置循环计数
	a.staleWrittenFiles = nil // 每轮重置 stale anchor 追踪
	a.staleReadFiles = nil
	a.repeatSuccessCounts = nil // 每轮重置成功循环计数
	a.turnMu.Unlock()
	// per-turn TurnResult tracking — accumulated here and returned by Run().
	var turnFilesCreated []string
	var turnFilesModified []string
	var turnToolErrors []string
	var turnLastSummary string
	// 重置参数风暴断路器——每个 turn 独立统计
	if a.paramStorm != nil {
		a.paramStorm.Reset()
	}
	// the clear() method resets mtime caches auto-expired entries

	// recall-reminder lets the model know mid-turn context remains
	a.maybeRecallReminder()

	graceRound := false
	// stream recovery + empty final detection counters
	streamRecoveries := 0
	const maxStreamRecoveries = 3
	emptyFinalBlocks := 0
	const maxEmptyFinalBlocks = 3
	finalReadinessBlocks := 0
	const maxFinalReadinessBlocks = 3
	for step := 0; a.maxSteps <= 0 || step < a.maxSteps || graceRound; step++ {
		// consume a queued mid-turn steer as session guidance
		// (Design adopted from DeepSeek-Reasonix-V1.12)
		if text, ok := a.consumeSteer(); ok {
			a.session.Add(provider.Message{Role: provider.RoleUser, Content: midTurnSteerMessage(text)})
			a.sink.Emit(event.Event{Kind: event.Steer, Text: text})
		}
		// mid-turn compaction — before the next sampling round, check whether
		// the tool batch (or a mid-turn steer) pushed the estimated prompt past
		// the high-water mark. Compacting here prevents long tool chains from
		// growing the context monotonically to the window limit (distilled from
		// codex CompactionPhase::MidTurn). Skipped during the grace round so the
		// final summary never pays a message rewrite right before finishing.
		if !graceRound {
			a.midTurnMaybeCompact(ctx)
		}
		text, reasoning, signature, calls, usage, interrupted, err := a.stream(ctx, step+1)
		if err != nil {
			// stream recovery — save partial output and inject recovery prompt
			if interrupted && streamRecoveries < maxStreamRecoveries {
				streamRecoveries++
				if strings.TrimSpace(text) != "" {
					a.session.Add(provider.Message{
						Role:               provider.RoleAssistant,
						Content:            text,
						ReasoningContent:   reasoning,
						ReasoningSignature: signature,
					})
				}
				a.session.Add(provider.Message{
					Role:    provider.RoleUser,
					Content: streamRecoveryMessage(strings.TrimSpace(text) != ""),
				})
				a.sink.Emit(event.Event{Kind: event.Retrying, RetryAttempt: streamRecoveries, RetryMax: maxStreamRecoveries})
				// recovery retries do not consume the tool-round budget. step--
				// is clamped at 0: a negative step would keep the loop condition
				// `step < a.maxSteps` vacuously true and, combined with the
				// grace-round boundary below, could grant an extra model round.
				if step > 0 {
					step--
				}
				continue
			}
			a.preWG.Wait() // drain any in-flight pre-execution goroutines before returning
			// terminal stream error (unrecoverable, or recovery exhausted) — do
			// not drop already-received partial output. Persist it to the session
			// and to the result summary so the user and the next turn still see
			// the content the model produced before the failure.
			if strings.TrimSpace(text) != "" {
				a.session.Add(provider.Message{
					Role:               provider.RoleAssistant,
					Content:            text,
					ReasoningContent:   reasoning,
					ReasoningSignature: signature,
				})
				turnLastSummary = text
			}
			return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), err
		}
		streamRecoveries = 0

		// length-truncation — inject nudge when finish_reason="length" and no tool calls
		if a.maybeContinueOutputLength(usage, calls) {
			continue
		}
		// invalid output — handle empty reasoning/text after retry
		if a.maybeRetryInvalidOutput(text, reasoning, calls) {
			continue
		}

		if usage != nil && usage.TotalTokens > 0 {
			a.sink.Emit(event.Event{Kind: event.Usage, Usage: usage, Pricing: a.pricing,
				SessionHit: int(a.sessCacheHit.Load()), SessionMiss: int(a.sessCacheMiss.Load()),
				UsageSource: event.UsageSourceExecutor})
			// budget gate — cumulative fee check
			if a.budgetGate != nil {
				status := a.budgetGate.Check(a.pricing, usage)
				if status == BudgetWarn {
					a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
						Text: a.budgetGate.StatusMessage()})
				}
				if status == BudgetBlock {
					a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
						Text: a.budgetGate.StatusMessage()})
					a.preWG.Wait()
					return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), fmt.Errorf("budget exceeded: %s", a.budgetGate.StatusMessage())
				}
			}
		}
		// Phase 3: compute cache-shape fingerprint for TCCA diagnostics
		if a.prefixFingerprintSet {
			shape := a.CaptureShape()
			a.lastPrefixShape = shape
		}

		if msg, ok := finishReasonMessage(usage); ok {
			a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: msg})
		}

		// automatic compaction — truncates history when prompt
		// exceeds the high-water mark. legacyTruncate preserves
		// L1+L2+prefix+summary+tail for maximum cache continuity.
		// Skip during grace round: compaction rewrites the message
		// array (LogRewriteVersion++), zeroing the prefix cache
		// right before the turn ends — pure waste.
		if !graceRound {
			a.maybeCompact(ctx, usage)
		}

		// Keep reasoning_content on the assistant turn for display and session
		// archive. It is NOT re-uploaded to the API: the openai provider drops it
		// when building the request, since re-sent reasoning is billable prompt
		// input for no cache or coherence gain.
		a.session.Add(provider.Message{
			Role:               provider.RoleAssistant,
			Content:            text,
			ReasoningContent:   reasoning,
			ReasoningSignature: signature,
			ToolCalls:          calls,
		})
		// capture last assistant text for TurnResult
		turnLastSummary = text

		// archive the assistant turn for cross-session analysis
		if a.archive != nil {
			tcJSON := "[]"
			if len(calls) > 0 {
				if b, err := json.Marshal(calls); err == nil {
					tcJSON = string(b)
				}
			}
			a.archive.RecordMessage(a.sessionID, string(provider.RoleAssistant), text, tcJSON, step+1)
		}

		if len(calls) == 0 {
			// finish-gate — prevent premature model stop
			// Grace Round — model produced summary, done.
			if graceRound {
				// clean up grace-round nudge from session before exit
				if len(a.session.Messages) > 0 {
					a.session.Messages = a.session.Messages[:len(a.session.Messages)-1]
				}
				return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), nil
			}

			// empty final detection — model returned no tool calls
			// and no visible text. Inject retry prompt; fail after 3 blocks.
			if strings.TrimSpace(text) == "" {
				emptyFinalBlocks++
				if emptyFinalBlocks >= maxEmptyFinalBlocks {
					return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), fmt.Errorf("model finished without a visible final answer %d times", emptyFinalBlocks)
				}
				a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
					Text: fmt.Sprintf("empty final answer blocked: retrying (%d/%d)", emptyFinalBlocks, maxEmptyFinalBlocks)})
				a.session.Add(provider.Message{Role: provider.RoleUser, Content: emptyFinalRetryMessage()})
				a.maybeCompact(ctx, usage)
				continue
			}

			// Gate 1: task gate — check against task completion state

			if a.taskGate() {
				continue
			}
			// Gate 2: goal gate — judge model goal verification
			if a.goalGate(ctx) {
				continue
			}
			// verify gate merged into taskGate — no separate call needed
			// final-answer readiness gate — verify evidence before accepting completion
			if blocked, reason := a.finalReadinessCheck(); blocked {
				finalReadinessBlocks++
				if finalReadinessBlocks >= maxFinalReadinessBlocks {
					return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), fmt.Errorf("final-answer readiness failed %d times: %s", finalReadinessBlocks, reason)
				}
				a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
					Text: "final-answer readiness blocked: " + reason})
				a.session.Add(provider.Message{Role: provider.RoleUser, Content: finalReadinessRetryMessage(reason)})
				a.maybeCompact(ctx, usage)
				continue
			}
			if a.steerQueueLen() > 0 {
				continue // more steers pending — another pass
			}
			return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), nil // all gates passed
		}

		// wait for stream() pre-execution goroutines to finish before
		// dispatching the full batch — avoids races and double-execution.
		emptyFinalBlocks = 0 // reset empty-final counter when model calls tools successfully
		// Grace Round guard — if model still calls tools during grace round, exit.
		// Ported from Reasonix to prevent infinite loops under MaxSteps limit.
		if graceRound {
			a.preWG.Wait() // drain pre-exec goroutines started during grace streaming
			// clean up grace-round nudge to prevent leaking to next user turn
			if len(a.session.Messages) > 0 {
				a.session.Messages = a.session.Messages[:len(a.session.Messages)-1]
			}
			return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), fmt.Errorf("paused after %d tool-call rounds (agent.max_steps) — the model continued calling tools during the grace round; the work so far is saved. Send another message to continue, or increase max_steps", a.maxSteps)
		}
		a.preWG.Wait()
		results := a.executeBatch(ctx, calls)
		// P0-2: deterministic pruning — skip duplicate tool results.
		// only dedup ReadOnly tools — bash/git_commit etc. may produce
		// different results on repeated calls (state changed between calls).
		for i, call := range calls {
			// track writer tool paths for TurnResult
			switch call.Name {
			case "write_file":
				if p := extractFilePath(call.Name, call.Arguments); p != "" {
					turnFilesCreated = append(turnFilesCreated, p)
				}
			case "edit_file", "edit_lines", "move_file", "delete_range", "delete_symbol":
				if p := extractFilePath(call.Name, call.Arguments); p != "" {
					turnFilesModified = append(turnFilesModified, p)
				}
			}
			// collect tool errors, blocks, and suppressions for TurnResult (max 5):
			// a blocked or suppressed call still fails the turn — Success is true
			// only when the whole turn ran clean (no errors, no blocks, no
			// suppressions). Blocks arrive as plain "blocked:" / "precheck blocked:"
			// prefixed results, suppressions as "suppressed:", execution errors as
			// legacy "error:" / "tool panic:" prefixes.
			if len(turnToolErrors) < 5 && (strings.HasPrefix(results[i], "error:") ||
				strings.HasPrefix(results[i], "blocked:") ||
				strings.HasPrefix(results[i], "precheck blocked:") ||
				strings.HasPrefix(results[i], "suppressed:") ||
				strings.HasPrefix(results[i], "tool panic:")) {
				turnToolErrors = append(turnToolErrors, results[i])
			}
			// Skip suppressed calls (already have placeholder result).
			if strings.HasPrefix(results[i], "suppressed:") {
				a.session.Add(provider.Message{
					Role:       provider.RoleTool,
					Content:    results[i],
					ToolCallID: call.ID,
					Name:       call.Name,
				})
				continue
			}
			// Only dedup read-only tools — writers may legitimately change state
			dedupOK := false
			if t, ok := a.tools.Get(call.Name); ok {
				dedupOK = t.ReadOnly()
			}
			if dedupOK {
				dk := call.Name + "|" + truncateStr(call.Arguments, 64) + "|" + truncateStr(results[i], 64)
				if a.dedupHashes == nil {
					a.dedupHashes = make(map[string]bool)
				}
				if a.dedupHashes[dk] {
					results[i] = "[cached — same as previous " + call.Name + " call]"
				} else {
					a.dedupHashes[dk] = true
				}
			}
			a.session.Add(provider.Message{
				Role:       provider.RoleTool,
				Content:    results[i],
				ToolCallID: call.ID,
				Name:       call.Name,
			})
		}

		// advance canonical todo state for successful complete_step calls
		for i, call := range calls {
			if call.Name == "complete_step" && !strings.HasPrefix(results[i], "error:") && !strings.HasPrefix(results[i], "blocked:") {
				step := extractStepFromArgs(call.Arguments)
				if step != "" {
					a.advanceCanonicalTodo(step)
				}
			}
		}

		// P0-3: mid-turn steer — detect error patterns and inject corrective hints.
		if a.shouldMidTurnSteer(calls, results) {
			continue // steer injected, skip compaction and continue loop
		}

		// bg start-kill cycle — detect repeated background job start→kill
		// without reading output, inject corrective nudge after 3 cycles.
		if a.checkBgStartKillCycle() {
			continue
		}

		// repeat-detection — inject nudge after 3 same-tool calls
		if a.detectRepeatedSteps(calls) {
			continue // nudge injected, skip compaction and continue loop
		}

		// Grace Round — when maxSteps is reached, give one extra final turn.
		// step is always >= 0 here (the recovery path clamps step-- at 0), so
		// this boundary fires at most once per turn and never from a negative
		// step — no extra round beyond maxSteps + 1 (one grace).
		if a.maxSteps > 0 && step+1 >= a.maxSteps && !graceRound {
			graceRound = true
			nudge := "Do not call any more tools — your tool-call round limit (agent.max_steps) has been reached. Instead, synthesize a final answer from all the work already completed: summarize what was accomplished, what remains to be done, and any decisions the user should make."
			a.session.Add(provider.Message{
				Role:    provider.RoleUser,
				Content: nudge,
			})
			continue
		}

		// no mid-turn compaction — cache grows monotonically within each turn
	}
	// Only reached when a positive maxSteps guard is configured. The work so far
	// is already in the session, so the user can just send another message to pick
	// up where it left off.
	return buildTurnResult(turnFilesCreated, turnFilesModified, turnToolErrors, turnLastSummary), fmt.Errorf("paused after %d tool-call rounds (agent.max_steps)", a.maxSteps)
}

// buildTurnResult assembles a TurnResult from per-turn tracking variables.
// Used by runDirect at every return point so callers get partial results
// even when the turn ends with an error.
func buildTurnResult(created []string, modified []string, errors []string, summary string) *TurnResult {
	return &TurnResult{
		FilesCreated:  uniqFiles(created),
		FilesModified: uniqFiles(modified),
		Summary:       summary,
		Success:       len(errors) == 0,
		Errors:        errors,
	}
}

func uniqFiles(files []string) []string {
	seen := make(map[string]bool, len(files))
	uniq := make([]string, 0, len(files))
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			uniq = append(uniq, f)
		}
	}
	return uniq
}
