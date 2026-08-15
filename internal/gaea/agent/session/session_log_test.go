package session

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// Session 薄封装：ReplayFromLog（日志 + 投影）与 LogSeq 游标
func TestSessionReplayFromLog(t *testing.T) {
	s := New("sys")
	if s.LogSeq() != 0 {
		t.Fatalf("initial log seq = %d, want 0", s.LogSeq())
	}
	entries := []LogEntry{
		{Seq: 1, Kind: KindUserMessage, Payload: mustMarshal(userLogPayload{Content: "u1"})},
		{Seq: 2, Kind: KindAssistantMessage, Payload: mustMarshal(assistantLogPayload{Text: "a1"})},
		{Seq: 3, Kind: "usage", Payload: mustMarshal(usageLogPayload{PromptTokens: 5})},
	}
	s.ReplayFromLog(entries)
	msgs := s.Snapshot()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2 (usage 不投影)", len(msgs))
	}
	if msgs[0].Role != provider.RoleUser || msgs[1].Role != provider.RoleAssistant {
		t.Errorf("roles = %q,%q", msgs[0].Role, msgs[1].Role)
	}
	if s.LogSeq() != 3 {
		t.Fatalf("log seq = %d, want 3", s.LogSeq())
	}
	// 既有 API 不受影响
	s.Add(provider.Message{Role: provider.RoleUser, Content: "u2"})
	if len(s.Snapshot()) != 3 {
		t.Fatalf("after Add = %d, want 3", len(s.Snapshot()))
	}
	if !s.HasContent() {
		t.Fatal("HasContent should be true")
	}
}
