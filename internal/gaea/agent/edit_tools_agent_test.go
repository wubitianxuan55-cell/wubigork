package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/cache"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin" // 注册 builtin 工具（含 S0.6 编辑系）
)

// S0.6 integration tests: the edit-tools layer (edit_file / multi_edit /
// edit_lines / move_file / grep) wired through the agent runner — precheck,
// stale guard, loop guard, cache invalidation, conflict keys.

func lookupEditTool(t *testing.T, name string) tool.Tool {
	t.Helper()
	tl, ok := tool.LookupBuiltin(name)
	if !ok {
		t.Fatalf("builtin %q not registered — the tool layer must be implemented (S0.6)", name)
	}
	return tl
}

func editToolsRegistry(t *testing.T, names ...string) *tool.Registry {
	t.Helper()
	reg := tool.NewRegistry()
	for _, n := range names {
		reg.Add(lookupEditTool(t, n))
	}
	return reg
}

func TestEditToolsRegisteredWithReadOnlyClassification(t *testing.T) {
	for name, wantReadOnly := range map[string]bool{
		"edit_file": false, "multi_edit": false, "edit_lines": false,
		"move_file": false, "grep": true,
	} {
		tl := lookupEditTool(t, name)
		if got := tl.ReadOnly(); got != wantReadOnly {
			t.Errorf("%s ReadOnly() = %v, want %v", name, got, wantReadOnly)
		}
		// S0.6 risk 1: compact entries must exist or Schemas() exports empties.
		if cd, ok := tl.(tool.CompactDescriptor); ok {
			if cd.CompactDescription() == "" || len(cd.CompactSchema()) == 0 {
				t.Errorf("%s has empty compact desc/schema — missing compact.go entry", name)
			}
		}
	}
}

func TestPrecheckSilentForEditLinesMoveFileGrep(t *testing.T) {
	// edit_lines / move_file / grep carry no anchor semantics — precheck must
	// not interfere (design §3), letting Execute produce its own errors.
	a := &AgentRunner{}
	cases := []struct {
		name string
		args string
	}{
		{"edit_lines", `{"path":"x.txt","start_line":1,"end_line":2,"new_content":"y"}`},
		{"move_file", `{"source":"a.txt","destination":"b.txt"}`},
		{"grep", `{"pattern":"x"}`},
	}
	for _, tc := range cases {
		if msg := a.precheckTool(tc.name, []byte(tc.args)); msg != "" {
			t.Errorf("precheckTool(%s) = %q, want silent", tc.name, msg)
		}
	}
}

func TestGetConflictKeyEditTools(t *testing.T) {
	cases := []struct {
		name string
		args string
		want string
	}{
		{"move_file", `{"source":"a.txt","destination":"b.txt"}`, "!write"},
		{"edit_lines", `{"path":"x.txt","start_line":1,"end_line":1,"new_content":"y"}`, "file:x.txt"},
		{"edit_lines", `{"start_line":1,"end_line":1}`, "!write"},
		{"grep", `{"pattern":"x","path":"src"}`, "read:src"},
		{"grep", `{"pattern":"x"}`, ""},
	}
	for _, tc := range cases {
		if got := getConflictKey(provider.ToolCall{Name: tc.name, Arguments: tc.args}); got != tc.want {
			t.Errorf("getConflictKey(%s, %s) = %q, want %q", tc.name, tc.args, got, tc.want)
		}
	}
}

func TestRepeatSuccessSignatureCoversEditTools(t *testing.T) {
	for _, name := range []string{"edit_lines", "move_file"} {
		tl := lookupEditTool(t, name)
		sig, ok := repeatSuccessSignature(provider.ToolCall{
			Name: name, Arguments: `{"path":"a.txt","start_line":1,"end_line":1,"new_content":"x","source":"a.txt","destination":"b.txt"}`,
		}, tl)
		if !ok || !strings.HasPrefix(sig, name+"\x00") {
			t.Errorf("repeatSuccessSignature(%s) = %q ok=%v, want tool-prefixed signature", name, sig, ok)
		}
	}
	// grep is read-only → never signed (loop guard does not apply).
	grep := lookupEditTool(t, "grep")
	if _, ok := repeatSuccessSignature(provider.ToolCall{Name: "grep", Arguments: `{"pattern":"x"}`}, grep); ok {
		t.Error("grep must not participate in repeat-success signatures (ReadOnly)")
	}
}

func TestMoveFileLoopGuardAfterTwoSuccesses(t *testing.T) {
	tl := lookupEditTool(t, "move_file")
	a := &AgentRunner{}
	call := provider.ToolCall{ID: "c1", Name: "move_file",
		Arguments: `{"source":"a.txt","destination":"b.txt"}`}
	for i := 0; i < repeatSuccessAllowed; i++ {
		a.recordRepeatSuccess(call, tl)
	}
	if _, blocked := a.repeatedSuccessBlock(call, tl); !blocked {
		t.Fatal("move_file repeated with identical args must trip the loop guard")
	}
}

func TestMoveFileInvalidatesBothCachePaths(t *testing.T) {
	dir := t.TempDir()
	src := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	dst := filepath.ToSlash(filepath.Join(dir, "b.txt"))
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{
		tools: editToolsRegistry(t, "move_file"),
		tc:    cache.New(0),
	}
	a.tc.Set(src, 0, "cached-src")
	a.tc.Set(dst, 0, "cached-dst")

	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "move_file",
		Arguments: `{"source":"` + src + `","destination":"` + dst + `"}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("move_file should execute cleanly, got blocked=%v err=%q out=%q", out.blocked, out.errMsg, out.output)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be gone, stat err=%v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination should exist, stat err=%v", err)
	}
	// The S0.6 risk-2 assertion: BOTH endpoints' cache entries are gone —
	// move_file args carry no "path" key, so the generic branch would miss them.
	if _, hit := a.tc.Get(src, 0); hit {
		t.Error("cache entry for source must be invalidated after move_file")
	}
	if _, hit := a.tc.Get(dst, 0); hit {
		t.Error("cache entry for destination must be invalidated after move_file")
	}
}

func TestEditLinesCacheInvalidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{
		tools: editToolsRegistry(t, "edit_lines"),
		tc:    cache.New(0),
	}
	a.tc.Set(path, 0, "cached")

	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "edit_lines",
		Arguments: `{"path":"` + path + `","start_line":1,"end_line":1,"new_content":"ONE"}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("edit_lines should execute cleanly, got blocked=%v err=%q out=%q", out.blocked, out.errMsg, out.output)
	}
	if _, hit := a.tc.Get(path, 0); hit {
		t.Error("cache entry must be invalidated after edit_lines")
	}
	if content, _ := os.ReadFile(path); string(content) != "ONE\ntwo\n" {
		t.Errorf("file content = %q", content)
	}
}

func TestStaleGuardBlocksEditLinesWithoutRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{
		tools:             editToolsRegistry(t, "edit_lines"),
		staleWrittenFiles: map[string]bool{path: true}, // modified earlier this turn, not re-read
	}
	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "edit_lines",
		Arguments: `{"path":"` + path + `","start_line":1,"end_line":1,"new_content":"X"}`,
	})
	if !out.blocked || !strings.Contains(out.output, "[stale content]") {
		t.Fatalf("stale guard must block edit_lines on an unwritten-then-modified file, got %+v", out)
	}
	if content, _ := os.ReadFile(path); string(content) != "one\n" {
		t.Errorf("blocked edit must not touch the file, got %q", content)
	}
}

func TestStaleGuardBlocksMoveFileWithoutRead(t *testing.T) {
	dir := t.TempDir()
	src := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	if err := os.WriteFile(src, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{
		tools:             editToolsRegistry(t, "move_file"),
		staleWrittenFiles: map[string]bool{src: true},
	}
	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "move_file",
		Arguments: `{"source":"` + src + `","destination":"` + filepath.ToSlash(filepath.Join(dir, "b.txt")) + `"}`,
	})
	if !out.blocked || !strings.Contains(out.output, "[stale content]") {
		t.Fatalf("stale guard must block move_file on a stale source, got %+v", out)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("blocked move must not remove the source, stat err=%v", err)
	}
}

func TestEditLinesLoopGuardAfterTwoSuccesses(t *testing.T) {
	tl := lookupEditTool(t, "edit_lines")
	a := &AgentRunner{}
	call := provider.ToolCall{ID: "c1", Name: "edit_lines",
		Arguments: `{"path":"a.txt","start_line":1,"end_line":1,"new_content":"x"}`}
	for i := 0; i < repeatSuccessAllowed; i++ {
		a.recordRepeatSuccess(call, tl)
	}
	if _, blocked := a.repeatedSuccessBlock(call, tl); !blocked {
		t.Fatal("edit_lines repeated with identical args must trip the loop guard")
	}
}

func TestMoveFileEndToEndViaRunner(t *testing.T) {
	// Full executeOne pass for move_file with a name-list check: the result
	// is the moved contract output, not "unknown tool".
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	dst := filepath.Join(dir, "sub", "b.txt")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{tools: editToolsRegistry(t, "move_file")}
	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "move_file",
		Arguments: `{"source":"` + filepath.ToSlash(src) + `","destination":"` + filepath.ToSlash(dst) + `"}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("move_file failed: %+v", out)
	}
	if !strings.Contains(out.output, "moved") || !strings.Contains(out.output, "→") {
		t.Errorf("output = %q, want `moved <src> → <dst>`", out.output)
	}
	if content, err := os.ReadFile(dst); err != nil || string(content) != "payload" {
		t.Errorf("dst = %q err=%v", content, err)
	}
	// turnFilesModified bookkeeping: move_file is already on the list; the
	// extract helper must keep exposing the source for it.
	if p := extractFilePath("move_file", `{"source":"a.txt","destination":"b.txt"}`); p != "a.txt" {
		t.Errorf("extractFilePath(move_file) = %q, want source", p)
	}
}
