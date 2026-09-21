package agent

import (

	"github.com/gaea/gaea/internal/gaea/provider"
)

// outputLenNudgeCap prevents infinite continuation when the model keeps
// hitting the output length limit without producing tool calls.
const outputLenNudgeCap = 5

// outputLengthNudge is injected when finish_reason="length" and there are
// no tool calls — the model hit the output token limit mid-response.
const outputLengthNudge = "[system] " +
	"The previous response was cut off because it hit the output token limit. " +
	"Continue from where you left off — do NOT restart, recap, or repeat prior reasoning. " +
	"Keep reasoning concise, prefer concrete tool calls or final output."

// maybeContinueOutputLength checks whether the model's output was truncated
// and injects a continuation nudge. Returns true when it did (caller should
// continue the loop).
func (a *AgentRunner) maybeContinueOutputLength(u *provider.Usage, calls []provider.ToolCall) bool {
	if u == nil || u.FinishReason != "length" {
		a.lenContCount = 0
		return false
	}
	// Only continue for pure-text truncation. If the model had tool calls,
	// tool results are the natural continuation — let the loop handle it.
	if len(calls) > 0 {
		return false
	}
	a.lenContCount++
	if a.lenContCount > outputLenNudgeCap {
		return false // safety valve
	}
	a.session.Add(provider.Message{
		Role:    provider.RoleUser,
		Content: outputLengthNudge,
	})
	return true
}

