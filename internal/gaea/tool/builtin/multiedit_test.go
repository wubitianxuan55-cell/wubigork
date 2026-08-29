package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestMultiEditSerialApplication(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "aaa\n")

	// Each edit sees the previous one's result: a→b, then b→c.
	out := execTool(t, multiEdit{workDir: dir}, map[string]any{
		"path": "a.txt",
		"edits": []map[string]any{
			{"old_string": "a", "new_string": "b", "replace_all": true},
			{"old_string": "b", "new_string": "c", "replace_all": true},
		},
	})
	if !strings.Contains(out, "applied 2/2 edits") {
		t.Errorf("output should report 2/2: %q", out)
	}
	if !strings.Contains(out, "#0 ok") || !strings.Contains(out, "#1 ok") {
		t.Errorf("output should list per-edit ok lines: %q", out)
	}
	if content, _, _ := readFileEncoded(path); content != "ccc\n" {
		t.Errorf("content = %q, want serial application result ccc", content)
	}
}

func TestMultiEditAtomicOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "original\n")

	_, err := multiEdit{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": "a.txt",
		"edits": []map[string]any{
			{"old_string": "original", "new_string": "CHANGED"},
			{"old_string": "not-in-file", "new_string": "x"},
		},
	}))
	if err == nil {
		t.Fatal("a failing edit must fail the whole batch")
	}
	if !strings.Contains(err.Error(), "multi_edit[1]") || !strings.Contains(err.Error(), "no changes written") {
		t.Errorf("error should name the failing edit and the no-write guarantee: %v", err)
	}
	if content, _, _ := readFileEncoded(path); content != "original\n" {
		t.Errorf("file must be untouched after a failed batch, got %q", content)
	}
}

func TestMultiEditEmptyOldStringRejected(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "content\n")

	_, err := multiEdit{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path":  "a.txt",
		"edits": []map[string]any{{"old_string": "", "new_string": "inserted"}},
	}))
	if err == nil {
		t.Fatal("empty old_string must be rejected in Execute too")
	}
	if !strings.Contains(err.Error(), "old_string is required") {
		t.Errorf("error should explain the requirement: %v", err)
	}
	if content, _, _ := readFileEncoded(path); content != "content\n" {
		t.Errorf("file must be untouched, got %q", content)
	}
}

func TestMultiEditMultiMatchWithoutReplaceAllFails(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "dup dup\n")

	_, err := multiEdit{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path":  "a.txt",
		"edits": []map[string]any{{"old_string": "dup", "new_string": "x"}},
	}))
	if err == nil {
		t.Fatal("multi match without replace_all must fail")
	}
	if content, _, _ := readFileEncoded(path); content != "dup dup\n" {
		t.Errorf("file must be untouched, got %q", content)
	}
}

func TestMultiEditRequiresEdits(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "x\n")
	me := multiEdit{workDir: dir}

	if _, err := me.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": "a.txt", "edits": []map[string]any{},
	})); err == nil {
		t.Error("empty edits list must be rejected")
	}
	if _, err := me.Execute(context.Background(), mustArgs(t, map[string]any{
		"edits": []map[string]any{{"old_string": "x", "new_string": "y"}},
	})); err == nil {
		t.Error("missing path must be rejected")
	}
}

func TestMultiEditConfined(t *testing.T) {
	outside := t.TempDir()
	ws := t.TempDir()
	me := multiEdit{workDir: ws, roots: []string{ws}}
	path := writeTemp(t, outside, "secret.txt", "nope\n")

	_, err := me.Execute(context.Background(), mustArgs(t, map[string]any{
		"path":  path,
		"edits": []map[string]any{{"old_string": "nope", "new_string": "yes"}},
	}))
	if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("multi_edit must honour the write roots, got err=%v", err)
	}
}
