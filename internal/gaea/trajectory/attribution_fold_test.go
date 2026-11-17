package trajectory

// B5 工具归因（第四批 P3）：轨迹折叠把 dispatch 盖戳的 Model/Turn 浮出到
// ToolRec——「哪一轮、哪个模型、哪一次」三层可从折叠产物直接反查。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func TestFoldToolAttributionStamped(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "tool_dispatch", map[string]any{
			"id": "t1", "name": "bash", "args": "{}", "model": "test-model", "turn": 3,
		}),
		entry(3, "tool_result", map[string]any{
			"id": "t1", "name": "bash", "output": "ok", "model": "test-model", "turn": 3,
		}),
		entry(4, "turn_done", map[string]any{"err": ""}),
	}
	tl := FoldTrajectory(entries)
	if len(tl.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tl.Turns))
	}
	var rec *ToolRec
	for _, r := range tl.Turns[0].Records {
		if r.Kind == "tool" && r.Tool != nil {
			rec = r.Tool
		}
	}
	if rec == nil {
		t.Fatal("no tool record folded")
	}
	if rec.Model != "test-model" || rec.Turn != 3 {
		t.Errorf("ToolRec attribution = %q/%d, want test-model/3", rec.Model, rec.Turn)
	}
}

func TestFoldToolAttributionLegacyOmitted(t *testing.T) {
	// 旧日志（无 model/turn 字段）折叠 → 零值缺省，不误标。
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "tool_dispatch", map[string]any{"id": "t1", "name": "bash", "args": "{}"}),
		entry(3, "tool_result", map[string]any{"id": "t1", "name": "bash", "output": "ok"}),
		entry(4, "turn_done", map[string]any{"err": ""}),
	}
	tl := FoldTrajectory(entries)
	if len(tl.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tl.Turns))
	}
	for _, r := range tl.Turns[0].Records {
		if r.Kind == "tool" && r.Tool != nil {
			if r.Tool.Model != "" || r.Tool.Turn != 0 {
				t.Errorf("legacy ToolRec attribution = %q/%d, want 空/0", r.Tool.Model, r.Tool.Turn)
			}
			return
		}
	}
	t.Fatal("no tool record folded")
}
