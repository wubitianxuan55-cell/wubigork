package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileNegativeOffsetReadsEndWithAbsoluteLineNumbers(t *testing.T) {
	dir := t.TempDir()
	var body strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&body, "line%d\n", i)
	}
	path := filepath.Join(dir, "chapter.md")
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := (readFile{workDir: dir}).Execute(context.Background(), json.RawMessage(`{"path":"chapter.md","offset":-5,"limit":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "96→line96") || !strings.Contains(out, "97→line97") {
		t.Fatalf("负 offset 应返回末尾窗口且标绝对行号:\n%s", out)
	}
	if strings.Contains(out, "1→line1") {
		t.Fatalf("负 offset 不应回落到文件头:\n%s", out)
	}
	if !strings.Contains(out, "pass offset=97 to continue") {
		t.Fatalf("截断尾窗应给出正确续读行号:\n%s", out)
	}
}

func TestReadFileNegativeOffsetBeyondFileStartClamps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (readFile{workDir: dir}).Execute(context.Background(), json.RawMessage(`{"path":"short.txt","offset":-20}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1→a") || !strings.Contains(out, "3→c") {
		t.Fatalf("尾窗超过文件长度时应从第 1 行开始:\n%s", out)
	}
}
