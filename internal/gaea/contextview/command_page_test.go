package contextview

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func TestCommandPageAtWalksOlderPages(t *testing.T) {
	entries := make([]session.LogEntry, 0, 10)
	for i := 0; i < 5; i++ {
		dispatchSeq := int64(i*2 + 1)
		resultSeq := dispatchSeq + 1
		id := "b" + string(rune('1'+i))
		entries = append(entries,
			entry(dispatchSeq, "tool_dispatch", map[string]any{
				"id": id, "name": "bash", "args": `{"command":"echo ` + id + `"}`,
			}),
			entry(resultSeq, "tool_result", map[string]any{
				"id": id, "name": "bash", "output": "ok",
			}),
		)
	}
	page := CommandPageAt(entries, 8, 2)
	if page.Total != 5 || page.AtStart || len(page.Commands) != 2 {
		t.Fatalf("page = %+v", page)
	}
	if page.Commands[0].Seq != 5 || page.Commands[1].Seq != 7 {
		t.Fatalf("page seqs = %d,%d", page.Commands[0].Seq, page.Commands[1].Seq)
	}
	first := CommandPageAt(entries, page.Commands[0].Seq, 2)
	if !first.AtStart || len(first.Commands) != 2 || first.Commands[0].Seq != 1 || first.Commands[1].Seq != 3 {
		t.Fatalf("first page = %+v", first)
	}
}
