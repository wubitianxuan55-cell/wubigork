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

func TestEditLinesBasic(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "one\ntwo\nthree\nfour\n")

	out := execTool(t, editLines{workDir: dir}, map[string]any{
		"path": "a.txt", "start_line": 2, "end_line": 3, "new_content": "TWO\nTHREE\nTHREE-AND-A-HALF",
	})
	if !strings.Contains(out, "replaced lines 2-3 in "+path+" (2→3 lines)") {
		t.Errorf("output = %q", out)
	}
	if content, _, _ := readFileEncoded(path); content != "one\nTWO\nTHREE\nTHREE-AND-A-HALF\nfour\n" {
		t.Errorf("content = %q", content)
	}
}

func TestEditLinesDeleteRange(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "one\ntwo\nthree\n")

	execTool(t, editLines{workDir: dir}, map[string]any{
		"path": "a.txt", "start_line": 2, "end_line": 2, "new_content": "",
	})
	if content, _, _ := readFileEncoded(path); content != "one\nthree\n" {
		t.Errorf("content = %q, want line 2 deleted", content)
	}
}

func TestEditLinesValidation(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "one\ntwo\nthree\n")
	cases := []struct {
		name       string
		start, end int
		wantInErr  string
	}{
		{"start below 1", 0, 1, "start_line must be >= 1"},
		{"end before start", 3, 2, "end_line (2) must be >= start_line (3)"},
		{"start past EOF", 4, 4, "start_line 4 is past EOF"},
		{"end past EOF", 1, 4, "end_line 4 is past EOF"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := editLines{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
				"path": "a.txt", "start_line": tc.start, "end_line": tc.end, "new_content": "x",
			}))
			if err == nil || !strings.Contains(err.Error(), tc.wantInErr) {
				t.Errorf("err = %v, want containing %q", err, tc.wantInErr)
			}
		})
	}
}

func TestEditLinesNewContentRequired(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "one\n")

	// new_content key absent entirely → rejected (distinguishable from ""=delete).
	_, err := editLines{workDir: dir}.Execute(context.Background(), json.RawMessage(
		`{"path":"a.txt","start_line":1,"end_line":1}`))
	if err == nil || !strings.Contains(err.Error(), "new_content is required") {
		t.Errorf("err = %v, want new_content required", err)
	}
}

func TestEditLinesPreservesCRLFUntouchedLines(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "crlf.txt", "one\r\ntwo\r\nthree\r\n")

	execTool(t, editLines{workDir: dir}, map[string]any{
		"path": "crlf.txt", "start_line": 2, "end_line": 2, "new_content": "TWO",
	})
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Untouched lines keep their CRLF byte-for-byte; no silent normalization.
	if got, want := string(raw), "one\r\nTWO\nthree\r\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestEditLinesTrailingNewlineBehavior(t *testing.T) {
	dir := t.TempDir()
	// File without a trailing newline stays without one.
	path := writeTemp(t, dir, "noeol.txt", "one\ntwo")
	execTool(t, editLines{workDir: dir}, map[string]any{
		"path": "noeol.txt", "start_line": 2, "end_line": 2, "new_content": "TWO",
	})
	if content, _, _ := readFileEncoded(path); content != "one\nTWO" {
		t.Errorf("content = %q, want no trailing newline added", content)
	}
}

func TestEditLinesPreservesGB18030(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gb.txt")
	original := "第一行\n第二行\n第三行\n"
	if err := os.WriteFile(path, fileenc.Encode(original, fileenc.GB18030), 0o644); err != nil {
		t.Fatal(err)
	}

	execTool(t, editLines{workDir: dir}, map[string]any{
		"path": "gb.txt", "start_line": 2, "end_line": 2, "new_content": "REPLACED",
	})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := fileenc.Detect(raw)
	if enc != fileenc.GB18030 {
		t.Fatalf("encoding = %v, want GB18030", enc)
	}
	if got := string(fileenc.Decode(raw, enc)); got != "第一行\nREPLACED\n第三行\n" {
		t.Errorf("decoded content = %q", got)
	}
}

func TestEditLinesConfined(t *testing.T) {
	outside := t.TempDir()
	ws := t.TempDir()
	path := writeTemp(t, outside, "secret.txt", "one\n")

	_, err := editLines{workDir: ws, roots: []string{ws}}.Execute(context.Background(), mustArgs(t, map[string]any{
		"path": path, "start_line": 1, "end_line": 1, "new_content": "x",
	}))
	if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("edit_lines must honour the write roots, got err=%v", err)
	}
}

func TestSplitFileLines(t *testing.T) {
	lines, trailing := splitFileLines("")
	if len(lines) != 0 || trailing {
		t.Errorf("empty file: lines=%v trailing=%v", lines, trailing)
	}
	lines, trailing = splitFileLines("\n")
	if len(lines) != 1 || lines[0] != "" || !trailing {
		t.Errorf("lone newline: lines=%v trailing=%v", lines, trailing)
	}
	lines, trailing = splitFileLines("a\r\nb\r\n")
	if len(lines) != 2 || lines[0] != "a\r" || lines[1] != "b\r" || !trailing {
		t.Errorf("crlf: lines=%q trailing=%v", lines, trailing)
	}
	lines, trailing = splitFileLines("a\nb")
	if len(lines) != 2 || lines[1] != "b" || trailing {
		t.Errorf("no-eol: lines=%q trailing=%v", lines, trailing)
	}
	if got := newLinesFromContent(""); got != nil {
		t.Errorf("empty new_content = %v, want nil (delete)", got)
	}
	if got := newLinesFromContent("\n"); len(got) != 1 || got[0] != "" {
		t.Errorf("lone newline new_content = %q, want one empty line", got)
	}
}
