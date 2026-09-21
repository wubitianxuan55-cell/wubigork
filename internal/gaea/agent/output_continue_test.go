package agent

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// TestOutputLengthContinueInjectsNudge when finish_reason="length" and no tool calls.
func TestOutputLengthContinueInjectsNudge(t *testing.T) {
	s := NewSession("")
	a := &AgentRunner{session: s}
	u := &provider.Usage{FinishReason: "length"}

	fired := a.maybeContinueOutputLength(u, nil)
	if !fired {
		t.Fatal("expected true for truncated output")
	}
	last := s.Messages[len(s.Messages)-1]
	if last.Content != outputLengthNudge {
		t.Fatalf("nudge text mismatch: got %q", last.Content)
	}
}

// TestOutputLengthContinueSkipsWithToolCalls when the model had tool calls.
func TestOutputLengthContinueSkipsWithToolCalls(t *testing.T) {
	s := NewSession("")
	a := &AgentRunner{session: s}
	u := &provider.Usage{FinishReason: "length"}
	calls := []provider.ToolCall{{ID: "1", Name: "read_file"}}

	if a.maybeContinueOutputLength(u, calls) {
		t.Fatal("expected false when tool calls present")
	}
}

// TestOutputLengthContinueSkipsNormalFinish when finish_reason is "stop".
func TestOutputLengthContinueSkipsNormalFinish(t *testing.T) {
	s := NewSession("")
	a := &AgentRunner{session: s}
	u := &provider.Usage{FinishReason: "stop"}

	if a.maybeContinueOutputLength(u, nil) {
		t.Fatal("expected false for normal finish")
	}
}

// TestOutputLengthContinueSafetyValve caps retries.
func TestOutputLengthContinueSafetyValve(t *testing.T) {
	a := &AgentRunner{session: NewSession("")}
	u := &provider.Usage{FinishReason: "length"}

	for i := 0; i < outputLenNudgeCap; i++ {
		if !a.maybeContinueOutputLength(u, nil) {
			t.Fatalf("retry %d: expected true", i+1)
		}
	}
	if a.maybeContinueOutputLength(u, nil) {
		t.Fatal("after cap: expected false")
	}
}
