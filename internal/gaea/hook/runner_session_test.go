package hook

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRunnerWithSessionIDSetsPayloadWithoutMutatingParent(t *testing.T) {
	var payloads []Payload
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		var p Payload
		if err := json.Unmarshal([]byte(in.Stdin), &p); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		payloads = append(payloads, p)
		return SpawnResult{ExitCode: 0}
	}
	base := NewRunner([]ResolvedHook{{
		HookConfig: HookConfig{Command: "probe"},
		Event:      PreToolUse,
	}}, "/tmp", spawner, nil)
	child := base.WithSessionID("sa_child_1")

	base.PreToolUse(context.Background(), "probe", nil)
	child.PreToolUse(context.Background(), "probe", nil)

	if len(payloads) != 2 {
		t.Fatalf("payload count = %d, want 2", len(payloads))
	}
	if payloads[0].SessionID != "" {
		t.Errorf("parent payload sessionId = %q, want empty", payloads[0].SessionID)
	}
	if payloads[1].SessionID != "sa_child_1" {
		t.Errorf("child payload sessionId = %q, want sa_child_1", payloads[1].SessionID)
	}
}
