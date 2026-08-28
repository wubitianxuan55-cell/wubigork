package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestMemoryGetTool(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(Memory{Name: "prefers-tabs", Title: "Prefers tabs", Description: "desc", Type: TypeUser, Kind: KindEpisodic, Tags: []string{"tabs"}, Body: "Use tabs."}); err != nil {
		t.Fatal(err)
	}

	tool := NewMemoryGetTool(s)
	args, _ := json.Marshal(map[string]string{"name": "prefers-tabs"})
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"prefers-tabs", "Use tabs.", "episodic"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}

	// 不存在 → 报错
	bad, _ := json.Marshal(map[string]string{"name": "nope"})
	if _, err := tool.Execute(context.Background(), bad); err == nil {
		t.Error("expected error for missing memory")
	}
	// 缺参数 → 报错
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("expected error for missing name")
	}
}
