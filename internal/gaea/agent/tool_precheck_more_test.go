package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrecheckMultiEdit_AllFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "test.go"))
	content := "line one\nline two\nline three\n"
	os.WriteFile(path, []byte(content), 0644)

	a := &AgentRunner{}
	msg := a.precheckMultiEdit([]byte(`{
		"path": "` + path + `",
		"edits": [
			{"old_string": "line one", "new_string": "LINE ONE"},
			{"old_string": "line three", "new_string": "LINE THREE"}
		]
	}`))
	if msg != "" {
		t.Fatalf("all old_strings present, should pass: %s", msg)
	}
}

func TestPrecheckMultiEdit_OneNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "test.go"))
	content := "line one\nline two\nline three\n"
	os.WriteFile(path, []byte(content), 0644)

	a := &AgentRunner{}
	msg := a.precheckMultiEdit([]byte(`{
		"path": "` + path + `",
		"edits": [
			{"old_string": "line one", "new_string": "x"},
			{"old_string": "LINE FOUR", "new_string": "y"},
			{"old_string": "line three", "new_string": "z"}
		]
	}`))
	if msg == "" {
		t.Fatal("LINE FOUR not in file, should block")
	}
	if !strings.Contains(msg, "precheck blocked") {
		t.Fatalf("msg should say precheck blocked: %s", msg)
	}
	if !strings.Contains(msg, "multi_edit[1]") {
		t.Fatalf("msg should mention which edit failed (index 1): %s", msg)
	}
}

func TestPrecheckMultiEdit_EmptyEdits(t *testing.T) {
	a := &AgentRunner{}
	msg := a.precheckMultiEdit([]byte(`{"path":"/tmp/x.go","edits":[]}`))
	if msg != "" {
		t.Fatal("empty edits should be silent")
	}
}

func TestPrecheckMultiEdit_MissingFile(t *testing.T) {
	a := &AgentRunner{}
	msg := a.precheckMultiEdit([]byte(`{
		"path": "/nonexistent/file.go",
		"edits": [{"old_string": "x", "new_string": "y"}]
	}`))
	if msg != "" {
		t.Fatalf("missing file should be silent: %s", msg)
	}
}

func TestPrecheckMultiEdit_EmptyOldString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "test.go"))
	os.WriteFile(path, []byte("content"), 0644)

	a := &AgentRunner{}
	// S0.6: empty old_string is rejected uniformly — precheck must block what
	// multi_edit's Execute would reject, instead of silently passing it through
	// (the old "insert at position" reading no longer exists).
	msg := a.precheckMultiEdit([]byte(`{
		"path": "` + path + `",
		"edits": [{"old_string": "", "new_string": "inserted"}]
	}`))
	if msg == "" {
		t.Fatal("empty old_string should be blocked by precheck")
	}
	if !strings.Contains(msg, "precheck blocked") {
		t.Fatalf("msg should say precheck blocked: %s", msg)
	}
	if !strings.Contains(msg, "multi_edit[0]") {
		t.Fatalf("msg should mention which edit failed (index 0): %s", msg)
	}
}

func TestPrecheckToolDispatch_MultiEdit(t *testing.T) {
	a := &AgentRunner{}
	// These don't touch real files — they should just route to the right checker.
	if msg := a.precheckTool("multi_edit", []byte(`{"path":"/tmp/x","edits":[]}`)); msg != "" {
		t.Fatal("multi_edit with empty edits should be silent")
	}
}
