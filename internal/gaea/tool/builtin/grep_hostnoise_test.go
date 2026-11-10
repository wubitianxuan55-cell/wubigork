package builtin

// 刀0 测试（dsh-ops 第二波 2026-11-10）：.gaea 宿主自管子树跳过 + 头行真前置。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func d0Workspace(t *testing.T) (string, grepTool) {
	t.Helper()
	ws := t.TempDir()
	return ws, grepTool{workDir: ws}
}

func d0Args(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGrepSkipsGaeaHostSubtrees(t *testing.T) {
	ws, g := d0Workspace(t)
	// 宿主自管：会话日志/台账/隔离——不得入结果。
	for _, host := range []string{
		".gaea/sessions/work/s1.gaea-log.jsonl",
		".gaea/work/journal/key.jsonl",
		".gaea/work/rollback/f.before",
		".gaea/publish-quarantine/20260101/report.docx",
		".gaea/renders/r.png",
		".gaea/archive/old.jsonl",
	} {
		full := filepath.Join(ws, filepath.FromSlash(host))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("needle host artifact"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 保留面：exports 交付产物与普通正文必须可搜。
	for _, keep := range []string{".gaea/exports/deliver.md", "src/main.go"} {
		full := filepath.Join(ws, filepath.FromSlash(keep))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("needle keep"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "sessions") || strings.Contains(out, "journal") ||
		strings.Contains(out, "rollback") || strings.Contains(out, "quarantine") ||
		strings.Contains(out, "renders") || strings.Contains(out, "archive") {
		t.Fatalf("宿主自管子树不得入结果: %q", out)
	}
	if !strings.Contains(out, "exports/deliver.md") || !strings.Contains(out, "src/main.go") {
		t.Fatalf("exports 与普通正文必须可搜: %q", out)
	}
}

func TestGrepHeaderLineFirst(t *testing.T) {
	ws, g := d0Workspace(t)
	// 3 文件各 2 命中 → offset=2 翻页，头行必须是输出第一行。
	for i := 1; i <= 3; i++ {
		name := filepath.Join(ws, "h"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(name, []byte("needle one\nmiddle\nneedle two\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "offset": 2}))
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(first, "[grep page] offset=2 shown=4") {
		t.Fatalf("头行应前置且为本页起始 offset: first=%q full=%q", first, out)
	}
}

func TestGrepNegativeOffsetClampedInEcho(t *testing.T) {
	ws, g := d0Workspace(t)
	if err := os.WriteFile(filepath.Join(ws, "n.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 凭据文件触发头行（围栏记账），负 offset 借此进入回显路径。
	if err := os.WriteFile(filepath.Join(ws, ".env"), []byte("needle=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "offset": -5}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "offset=-") {
		t.Fatalf("负 offset 不得回显负游标: %q", out)
	}
	if !strings.Contains(out, "offset=0 ") {
		t.Fatalf("应回显钳位后 offset=0: %q", out)
	}
}

// writeFileForTest 造工作区文件（D 系测试共用小助手）。
func writeFileForTest(ws, name, content string) error {
	return os.WriteFile(filepath.Join(ws, name), []byte(content), 0o644)
}
