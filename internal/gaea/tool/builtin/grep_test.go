package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGrepTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeTemp(t, dir, "a.go", "package main\nfunc hello() {}\nfunc bye() {}\n")
	writeTemp(t, dir, "b.txt", "hello from text\nnothing here\nhello again\n")
	writeTemp(t, dir, "c.md", "# title\nno match body\n")
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemp(t, dir, filepath.Join("node_modules", "pkg", "dep.go"), "func hello() {}\n")
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemp(t, dir, filepath.Join(".git", "config.go"), "func hello() {}\n")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemp(t, dir, filepath.Join("sub", "d.go"), "hello in subdir\n")
	return dir
}

func TestGrepBasicContract(t *testing.T) {
	dir := writeGrepTree(t)

	out := execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "hello"})
	// Contract: `path:line: content` lines (compressGrep parses this format).
	for _, want := range []string{
		"a.go:2: func hello() {}",
		"b.txt:1: hello from text",
		"b.txt:3: hello again",
		"sub/d.go:1: hello in subdir",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "node_modules") || strings.Contains(out, ".git") {
		t.Errorf("noise directories must be skipped:\n%s", out)
	}
	if !strings.Contains(out, ":2: ") {
		t.Errorf("line numbers must be 1-based with `:N: ` separator:\n%s", out)
	}
}

func TestGrepZeroMatches(t *testing.T) {
	dir := writeGrepTree(t)
	out := execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "zzz-not-there"})
	if !strings.HasPrefix(out, "(no matches") {
		t.Errorf("zero hits should say (no matches ...), got %q", out)
	}
}

func TestGrepInvalidRegex(t *testing.T) {
	dir := writeGrepTree(t)
	_, err := grepTool{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"pattern": "[unclosed",
	}))
	if err == nil {
		t.Fatal("invalid regex must be rejected")
	}
	if !strings.Contains(err.Error(), "invalid pattern") {
		t.Errorf("error should be validation-style: %v", err)
	}
}

func TestGrepSkipsBinary(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "bin.go", "hello\x00binary\nhello again\n")
	writeTemp(t, dir, "text.go", "hello text\n")

	out := execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "hello"})
	if strings.Contains(out, "bin.go") {
		t.Errorf("NUL-carrying file must be skipped as binary:\n%s", out)
	}
	if !strings.Contains(out, "text.go:1: hello text") {
		t.Errorf("text file must still match:\n%s", out)
	}
}

func TestGrepIncludeFilter(t *testing.T) {
	dir := writeGrepTree(t)
	out := execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "hello", "include": "*.go"})
	if strings.Contains(out, "b.txt") || strings.Contains(out, "c.md") {
		t.Errorf("include=*.go must exclude non-go files:\n%s", out)
	}
	for _, want := range []string{"a.go:2: ", "sub/d.go:1: "} {
		if !strings.Contains(out, want) {
			t.Errorf("include=*.go must keep %q:\n%s", want, out)
		}
	}
}

func TestGrepMaxResultsTruncates(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("needle here\n")
	}
	writeTemp(t, dir, "many.txt", sb.String())

	out := execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "needle", "max_results": 3})
	got := strings.Count(out, "needle here")
	if got != 3 {
		t.Errorf("want exactly 3 results, got %d:\n%s", got, out)
	}
	if !strings.Contains(out, "[truncated at 3 results") {
		t.Errorf("truncation should be announced:\n%s", out)
	}
}

func TestGrepSingleFileAndRelativePath(t *testing.T) {
	dir := writeGrepTree(t)
	// path pointing at a single file searches just that file.
	out := execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "hello", "path": "b.txt"})
	if !strings.Contains(out, "b.txt:1: hello from text") || strings.Contains(out, "a.go") {
		t.Errorf("single-file search failed:\n%s", out)
	}
	// Relative directory path.
	out = execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "hello", "path": "sub"})
	if !strings.Contains(out, "sub/d.go:1: hello in subdir") || strings.Contains(out, "a.go") {
		t.Errorf("relative path search failed:\n%s", out)
	}
}

func TestGrepSkipsUnreadableGracefully(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "ok.txt", "needle\n")
	// A directory named like a file can't happen, but a missing subdir must not abort.
	out := execTool(t, grepTool{workDir: dir}, map[string]any{"pattern": "needle", "path": "missing-dir"})
	if !strings.HasPrefix(out, "(no matches") {
		t.Errorf("missing search root should degrade to zero-hit text, got %q (err path)", out)
	}
}
