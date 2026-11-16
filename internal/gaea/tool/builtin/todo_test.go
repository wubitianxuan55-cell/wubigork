package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
)

func TestTodoWriteReportsRequiredSignoffAndCachePath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".gaea"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	ledger := evidence.NewLedger()
	ledger.Record(evidence.Receipt{
		ToolName: "todo_write",
		Success:  true,
		Todos:    []evidence.TodoItem{{Content: "写完第一章", Status: "in_progress"}},
	})
	ctx := evidence.WithLedger(context.Background(), ledger)

	out, err := (todoWrite{}).Execute(ctx, json.RawMessage(
		`{"todos":[{"content":"写完第一章","status":"completed"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"complete_step", "step=\"写完第一章\"", "Copyable shape"} {
		if !strings.Contains(out, want) {
			t.Fatalf("todo 回执缺 %q: %s", want, out)
		}
	}

	paths := (todoWrite{}).CachePaths(nil)
	wantPath := filepath.Join(root, ".gaea", "todos.md")
	if len(paths) != 1 || filepath.Clean(paths[0]) != wantPath {
		t.Fatalf("todo_write cache paths = %v, want %s", paths, wantPath)
	}
}
