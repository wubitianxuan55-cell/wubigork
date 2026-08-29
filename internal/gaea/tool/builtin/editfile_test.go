package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

// toolRunner is the slice of tool.Tool the white-box tests exercise.
type toolRunner interface {
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// mustArgs marshals a map into json.RawMessage for tool calls.
func mustArgs(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return b
}

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func execTool(t *testing.T, tool toolRunner, args map[string]any) string {
	t.Helper()
	out, err := tool.Execute(context.Background(), mustArgs(t, args))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out
}

func TestEditFileBasic(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "hello world\nfarewell\n")

	out := execTool(t, editFile{workDir: dir}, map[string]any{
		"path": "a.txt", "old_string": "hello", "new_string": "bye",
	})
	if got, want := out, "edited "+path+": 1 occurrence(s) replaced (+3/-5 bytes)"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
	if content, _, _ := readFileEncoded(path); content != "bye world\nfarewell\n" {
		t.Errorf("content = %q", content)
	}
}

func TestEditFileReplaceAll(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "x x x\n")

	out := execTool(t, editFile{workDir: dir}, map[string]any{
		"path": "a.txt", "old_string": "x", "new_string": "yy", "replace_all": true,
	})
	if !strings.Contains(out, "3 occurrence(s) replaced") {
		t.Errorf("output should report 3 replacements: %q", out)
	}
	if content, _, _ := readFileEncoded(path); content != "yy yy yy\n" {
		t.Errorf("content = %q", content)
	}
}

func TestEditFileNotFound(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "alpha\nbeta\n")

	_, err := editFile{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": "a.txt", "old_string": "missing", "new_string": "x",
	}))
	if err == nil {
		t.Fatal("want error for missing old_string")
	}
	if !strings.Contains(err.Error(), "old_string not found") {
		t.Errorf("error should say old_string not found: %v", err)
	}
	if !strings.Contains(err.Error(), "Re-read") {
		t.Errorf("error should suggest re-reading: %v", err)
	}
}

func TestEditFileMultipleNoReplaceAll(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "dup dup dup\n")

	_, err := editFile{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": "a.txt", "old_string": "dup", "new_string": "one",
	}))
	if err == nil {
		t.Fatal("want error when old_string matches multiple times without replace_all")
	}
	if !strings.Contains(err.Error(), "3 locations") || !strings.Contains(err.Error(), "replace_all") {
		t.Errorf("error should name the count and the escape hatch: %v", err)
	}
}

func TestEditFileEmptyOldStringRejected(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "content\n")

	_, err := editFile{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": "a.txt", "old_string": "", "new_string": "inserted",
	}))
	if err == nil {
		t.Fatal("empty old_string must be rejected")
	}
	if !strings.Contains(err.Error(), "old_string is required") {
		t.Errorf("error should explain the requirement: %v", err)
	}
	if content, _, _ := readFileEncoded(filepath.Join(dir, "a.txt")); content != "content\n" {
		t.Errorf("file must be untouched, got %q", content)
	}
}

func TestEditFileDeleteViaEmptyNewString(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "keep [DELETE ME] keep\n")

	execTool(t, editFile{workDir: dir}, map[string]any{
		"path": "a.txt", "old_string": " [DELETE ME]", "new_string": "",
	})
	if content, _, _ := readFileEncoded(filepath.Join(dir, "a.txt")); content != "keep keep\n" {
		t.Errorf("content = %q, want deletion", content)
	}
}

func TestEditFileCRLFHint(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "crlf.txt", "line one\r\nline two\r\n")

	_, err := editFile{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": "crlf.txt", "old_string": "line one\nline two", "new_string": "x",
	}))
	if err == nil {
		t.Fatal("bare-\\n old_string must not match a CRLF file")
	}
	if !strings.Contains(err.Error(), "CRLF") {
		t.Errorf("error should carry the CRLF diagnostic: %v", err)
	}

	// With the CRLF copied exactly the edit succeeds.
	execTool(t, editFile{workDir: dir}, map[string]any{
		"path": "crlf.txt", "old_string": "line one\r\nline two", "new_string": "done",
	})
	if content, _, _ := readFileEncoded(filepath.Join(dir, "crlf.txt")); content != "done\r\n" {
		t.Errorf("content = %q", content)
	}
}

func TestEditFilePreservesGB18030(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gb.txt")
	original := "中文内容第二行\nplain line\n"
	if err := os.WriteFile(path, fileenc.Encode(original, fileenc.GB18030), 0o644); err != nil {
		t.Fatal(err)
	}

	execTool(t, editFile{workDir: dir}, map[string]any{
		"path": "gb.txt", "old_string": "第二行", "new_string": "EDITED",
	})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := fileenc.Detect(raw)
	if enc != fileenc.GB18030 {
		t.Fatalf("encoding = %v, want GB18030 (file must not be re-coded to UTF-8)", enc)
	}
	if got := string(fileenc.Decode(raw, enc)); got != "中文内容EDITED\nplain line\n" {
		t.Errorf("decoded content = %q", got)
	}
}

func TestEditFilePreservesPerm(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "p.txt", "one\n")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	execTool(t, editFile{workDir: dir}, map[string]any{
		"path": "p.txt", "old_string": "one", "new_string": "two",
	})

	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("perm changed: before %v, after %v", before.Mode().Perm(), after.Mode().Perm())
	}
}

func TestEditFileConfined(t *testing.T) {
	outside := t.TempDir()
	ws := t.TempDir()
	path := writeTemp(t, outside, "secret.txt", "nope\n")

	_, err := editFile{workDir: ws, roots: []string{ws}}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": path, "old_string": "nope", "new_string": "yes",
	}))
	if err == nil {
		t.Fatal("edit outside the write roots must be confined")
	}
	if !strings.Contains(err.Error(), "outside the workspace") {
		t.Errorf("error should name the boundary: %v", err)
	}
}

func TestEditFileSymlinkEscapeBlocked(t *testing.T) {
	outside := t.TempDir()
	ws := t.TempDir()
	target := writeTemp(t, outside, "target.txt", "outside content\n")
	link := filepath.Join(ws, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}

	_, err := editFile{workDir: ws, roots: []string{ws}}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": "link.txt", "old_string": "outside", "new_string": "pwned",
	}))
	if err == nil {
		t.Fatal("editing through a symlink must be confined")
	}
	if !strings.Contains(err.Error(), "outside the workspace") {
		t.Errorf("error should name the boundary: %v", err)
	}
}

func TestReplaceInContentKernel(t *testing.T) {
	// Shared kernel: zero match / multi match / replaceAll semantics.
	got, n, err := replaceInContent("a b c", "a", "z", false)
	if err != nil || n != 1 || got != "z b c" {
		t.Errorf("single replace = %q n=%d err=%v", got, n, err)
	}
	if _, _, err := replaceInContent("a b c", "zz", "c", false); err == nil {
		t.Error("zero match must error")
	}
	if _, _, err := replaceInContent("a b a", "a", "c", false); err == nil {
		t.Error("multi match without replace_all must error")
	}
	got, n, err = replaceInContent("a b a", "a", "c", true)
	if err != nil || n != 2 || got != "c b c" {
		t.Errorf("replace_all = %q n=%d err=%v", got, n, err)
	}
}
