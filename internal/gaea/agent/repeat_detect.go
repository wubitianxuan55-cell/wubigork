package agent

import (
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// repeatedStepNudge is the turn-tail synthetic user message injected when the
// model has taken the same action 3 times in a row (escalation level 1).
// Compile-time constant for DeepSeek prefix-cache stability.
const repeatedStepNudge = "[system] Your last 3 steps have been identical — " +
	"you appear to be repeating the same action without making progress. " +
	"Stop and reconsider: the current approach is not working. " +
	"Try a different strategy, use a different tool, or if you are blocked, " +
	"explain the situation instead of repeating the same step."

// repeatedStepFinalNudge is the level-3 escalation at 8 repeats: stop calling
// tools and hand back to the user.
const repeatedStepFinalNudge = "[system] The same tool call has now repeated 8 times. " +
	"Stop calling tools. Summarize what you were trying to accomplish, what you " +
	"observed each time, and hand back a clear statement of the blocker so the " +
	"user can decide the next step."

// repeatPreviewLimit caps the argument preview inside the level-2 nudge.
// The detection key always uses the full arguments; the preview is display-only.
const repeatPreviewLimit = 500

// repeatNudgeFor picks the escalation level for the given repeat count —
// thresholds [3,5,8] with escalating detail (Distilled from Reasonix/dsh
// repeat-tool-reminder). Between tiers (4, 6, 7, 9+) it stays silent: nudging
// every repeat fattens the context without adding information. Denied calls
// count too — a model dead-looping on a rejected call is exactly the loop
// worth breaking (detection sees the calls regardless of their results).
func repeatNudgeFor(count int, calls []provider.ToolCall) string {
	switch count {
	case 3:
		return repeatedStepNudge
	case 5:
		return detailedRepeatNudge(calls)
	case 8:
		return repeatedStepFinalNudge
	default:
		return ""
	}
}

// detailedRepeatNudge names the offending call with a truncated argument
// preview so the model can see what it keeps re-emitting.
func detailedRepeatNudge(calls []provider.ToolCall) string {
	var b strings.Builder
	b.WriteString("[system] You have now repeated the exact same tool call 5 times in a row: ")
	for i, tc := range calls {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(tc.Name)
		b.WriteByte('(')
		b.WriteString(truncateStr(tc.Arguments, repeatPreviewLimit))
		b.WriteByte(')')
	}
	b.WriteString(". The call is not making progress — change the approach or the " +
		"arguments substantially, or explain what is blocking you instead of repeating it.")
	return b.String()
}

// stepSignature computes a stable signature for an assistant step's action —
// the tool calls it made (name + key-order-independent canonical args).
// Returns "" when the step has no tool calls (pure text turn), since there
// is no repeated *action* to detect.
//
// Mirrors MiMo-Code's stepSignature() in session/prompt.ts:140-148.
func stepSignature(calls []provider.ToolCall) string {
	if len(calls) == 0 {
		return ""
	}
	// Sort by (name, id) for deterministic ordering — models may emit
	// the same set of calls in different order.
	sorted := make([]provider.ToolCall, len(calls))
	copy(sorted, calls)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].ID < sorted[j].ID
	})

	var sb strings.Builder
	for i, tc := range sorted {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(tc.Name)
		sb.WriteByte(':')
		sb.WriteString(canonicalizeArgs(tc.Arguments))
	}
	return sb.String()
}

// detectRepeatedSteps tracks consecutive identical assistant steps and
// injects an escalating nudge at the [3,5,8] tiers. Returns true when it
// injected a nudge (caller should continue the loop).
func (a *AgentRunner) detectRepeatedSteps(calls []provider.ToolCall) bool {
	sig := stepSignature(calls)
	if sig == "" {
		a.repeatSig, a.repeatCount = "", 0
		return false
	}

	if sig != a.repeatSig {
		a.repeatSig, a.repeatCount = sig, 1
		return false
	}

	a.repeatCount++
	nudge := repeatNudgeFor(a.repeatCount, calls)
	if nudge == "" {
		return false
	}

	// Inject deterministic nudge as turn-tail synthetic user message.
	a.session.Add(provider.Message{
		Role:    provider.RoleUser,
		Content: nudge,
	})
	return true
}
