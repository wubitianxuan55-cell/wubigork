package context

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func TestMemoryStoreAppendRange(t *testing.T) {
	s := NewMemoryStore()
	if s.Len() != 0 {
		t.Fatalf("new MemoryStore Len = %d, want 0", s.Len())
	}
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "s1"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, Content: "a1"},
	}
	for _, m := range msgs {
		if err := s.Append(m); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if s.Len() != 3 {
		t.Fatalf("Len = %d, want 3", s.Len())
	}
	got, err := s.Range(0, 3)
	if err != nil {
		t.Fatalf("Range: %v", err)
	}
	if !reflect.DeepEqual(got, msgs) {
		t.Errorf("Range(0,3) = %v, want %v", got, msgs)
	}
	// out-of-bounds indices are clamped
	got, err = s.Range(-5, 99)
	if err != nil {
		t.Fatalf("Range(-5,99): %v", err)
	}
	if !reflect.DeepEqual(got, msgs) {
		t.Errorf("Range(-5,99) = %v, want %v", got, msgs)
	}
	// invalid window returns empty
	got, err = s.Range(2, 2)
	if err != nil {
		t.Fatalf("Range(2,2): %v", err)
	}
	if got != nil {
		t.Errorf("Range(2,2) = %v, want nil", got)
	}
}

func TestMemoryStoreTruncate(t *testing.T) {
	s := NewMemoryStore()
	for _, c := range []string{"a", "b", "c"} {
		_ = s.Append(provider.Message{Role: provider.RoleUser, Content: c})
	}
	if err := s.Truncate(1); err != nil {
		t.Fatalf("Truncate(1): %v", err)
	}
	if s.Len() != 1 {
		t.Fatalf("Len after Truncate(1) = %d, want 1", s.Len())
	}
	got, _ := s.Range(0, 10)
	if len(got) != 1 || got[0].Content != "a" {
		t.Errorf("after Truncate(1) = %v, want [a]", got)
	}
	// Truncate to a larger size is a no-op
	if err := s.Truncate(100); err != nil {
		t.Fatalf("Truncate(100): %v", err)
	}
	if s.Len() != 1 {
		t.Errorf("Len after Truncate(100) = %d, want 1", s.Len())
	}
}

func TestMemoryStoreEmptyRange(t *testing.T) {
	s := NewMemoryStore()
	got, err := s.Range(0, 0)
	if err != nil {
		t.Fatalf("Range(0,0): %v", err)
	}
	if got != nil {
		t.Errorf("empty Range = %v, want nil", got)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.jsonl")

	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "identity"},
		{Role: provider.RoleUser, Content: "hello", ReasoningContent: "think"},
		{Role: provider.RoleAssistant, Content: "hi", ToolCalls: []provider.ToolCall{{ID: "t1", Name: "read_file", Arguments: "{}"}}},
	}
	for _, m := range msgs {
		if err := fs.Append(m); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if fs.Len() != 3 {
		t.Fatalf("Len = %d, want 3", fs.Len())
	}
	if err := fs.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// reopen and verify messages were persisted
	fs2, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen NewFileStore: %v", err)
	}
	defer fs2.Close()
	if fs2.Len() != 3 {
		t.Fatalf("reopened Len = %d, want 3", fs2.Len())
	}
	got, err := fs2.Range(0, 10)
	if err != nil {
		t.Fatalf("Range: %v", err)
	}
	if !reflect.DeepEqual(got, msgs) {
		t.Errorf("round-trip = %v, want %v", got, msgs)
	}
}

func TestFileStoreTruncatePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.jsonl")

	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	for _, c := range []string{"a", "b", "c"} {
		_ = fs.Append(provider.Message{Role: provider.RoleUser, Content: c})
	}
	if err := fs.Truncate(1); err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	if fs.Len() != 1 {
		t.Fatalf("Len after Truncate(1) = %d, want 1", fs.Len())
	}
	if err := fs.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// reopen: stale tail must not come back (V4.0c fix)
	fs2, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer fs2.Close()
	if fs2.Len() != 1 {
		t.Fatalf("reopened Len = %d, want 1", fs2.Len())
	}
	got, _ := fs2.Range(0, 10)
	if len(got) != 1 || got[0].Content != "a" {
		t.Errorf("reopened = %v, want [a]", got)
	}
}

func TestFileStoreEmptyPath(t *testing.T) {
	fs, err := NewFileStore("")
	if err != nil {
		t.Fatalf("NewFileStore(\"\"): %v", err)
	}
	_ = fs.Append(provider.Message{Role: provider.RoleUser, Content: "mem"})
	if fs.Len() != 1 {
		t.Errorf("in-memory Len = %d, want 1", fs.Len())
	}
	if err := fs.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
