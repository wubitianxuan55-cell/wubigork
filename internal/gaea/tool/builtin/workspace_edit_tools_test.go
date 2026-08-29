package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestWorkspaceToolsIncludeEditTools guards the S0.6 wiring trap: the new
// edit tools must be in Workspace.Tools()'s `all` list (bound to the
// workspace dir + write roots), or a desktop front-end would see
// "unknown tool" forever while the CLI works.
func TestWorkspaceToolsIncludeEditTools(t *testing.T) {
	dir := t.TempDir()

	names := map[string]bool{}
	for _, tl := range (Workspace{Dir: dir}).Tools() {
		names[tl.Name()] = true
	}
	for _, want := range []string{"edit_file", "multi_edit", "edit_lines", "move_file", "grep"} {
		if !names[want] {
			t.Errorf("Workspace.Tools() missing %q (workspace-bound instance)", want)
		}
	}
}

// TestWorkspaceEditToolsBoundToDir verifies the binding is real: a relative
// path edit through Workspace.Tools() lands in the workspace directory, not
// the process cwd (mirrors TestAddBuiltins_WorkspaceDir in package boot).
func TestWorkspaceEditToolsBoundToDir(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "w.txt", "alpha beta\n")

	var editTool tool.Tool
	for _, tl := range (Workspace{Dir: dir}).Tools("edit_file") {
		editTool = tl
	}
	if editTool == nil {
		t.Fatal("edit_file not returned by Tools(\"edit_file\")")
	}
	execTool(t, editTool, map[string]any{
		"path": "w.txt", "old_string": "alpha", "new_string": "ALPHA",
	})
	if content, _, _ := readFileEncoded(filepath.Join(dir, "w.txt")); content != "ALPHA beta\n" {
		t.Errorf("edit must resolve relative to the workspace dir, got %q", content)
	}
	if _, err := os.Stat("w.txt"); err == nil {
		t.Error("w.txt must not appear in the process cwd — tool must bind the workspace dir")
	}
}

// TestWorkspaceToolsFilter verifies enabled-list filtering still works with
// the new tools present in `all`.
func TestWorkspaceToolsFilter(t *testing.T) {
	dir := t.TempDir()
	got := (Workspace{Dir: dir}).Tools("grep", "move_file")
	if len(got) != 2 {
		t.Fatalf("want 2 tools, got %d", len(got))
	}
	// Filtered output follows `all`-list order, not the enabled list's order.
	if got[0].Name() != "move_file" || got[1].Name() != "grep" {
		t.Errorf("names = %q %q", got[0].Name(), got[1].Name())
	}
}

// TestConfineWritersCoversNewTools verifies ConfineWriters binds every new
// writer and that confinement actually fires through the bound instance.
func TestConfineWritersCoversNewTools(t *testing.T) {
	ws := t.TempDir()
	outside := t.TempDir()
	confined := ConfineWriters([]string{ws})

	byName := map[string]tool.Tool{}
	for _, tl := range confined {
		byName[tl.Name()] = tl
	}
	for _, want := range []string{"write_file", "edit_file", "multi_edit", "edit_lines", "move_file"} {
		if _, ok := byName[want]; !ok {
			t.Errorf("ConfineWriters missing %q", want)
		}
	}

	path := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"edit_file", map[string]any{"path": path, "old_string": "x", "new_string": "y"}},
		{"multi_edit", map[string]any{"path": path, "edits": []map[string]any{{"old_string": "x", "new_string": "y"}}}},
		{"edit_lines", map[string]any{"path": path, "start_line": 1, "end_line": 1, "new_content": "y"}},
		{"move_file", map[string]any{"source": path, "destination": filepath.Join(ws, "stolen.txt")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := byName[tc.name].Execute(context.Background(), mustArgs(t, tc.args))
			if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
				t.Errorf("%s through ConfineWriters must confine, err=%v", tc.name, err)
			}
		})
	}
}

// TestCompactEntriesPresent guards S0.6 risk 1: every new tool implementing
// CompactDescriptor must have both compact entries, or Schemas() exports an
// empty description + empty schema (tool.go has no fallback).
func TestCompactEntriesPresent(t *testing.T) {
	for name, tl := range map[string]tool.Tool{
		"edit_file": editFile{}, "multi_edit": multiEdit{}, "edit_lines": editLines{},
		"move_file": moveFile{}, "grep": grepTool{},
	} {
		cd, ok := tl.(tool.CompactDescriptor)
		if !ok {
			t.Fatalf("%s must implement tool.CompactDescriptor", name)
		}
		if cd.CompactDescription() == "" {
			t.Errorf("%s: CompactDescription empty — compactDesc entry missing from compact.go", name)
		}
		if got := cd.CompactDescription(); got != compactDesc[name] {
			t.Errorf("%s: CompactDescription %q != compactDesc entry %q", name, got, compactDesc[name])
		}
		if len(cd.CompactSchema()) == 0 || string(cd.CompactSchema()) == "null" {
			t.Errorf("%s: CompactSchema empty — compactSchema entry missing from compact.go", name)
		}
	}
}
