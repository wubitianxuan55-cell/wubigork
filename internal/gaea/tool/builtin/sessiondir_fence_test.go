package builtin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestSessionDirFenceRule 围栏规则表：.gaea/sessions 段序列命中（大小写/
// 斜杠双形态/嵌套深度无关）；非会话形态不误伤。
func TestSessionDirFenceRule(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{`C:\ws\.gaea\sessions\a.jsonl`, true},
		{`/ws/.gaea/sessions/sub/b.gaea-log.jsonl`, true},
		{`/WS/.GAEA/Sessions/c.jsonl`, true},  // 大小写不敏感
		{`.gaea/sessions/d.jsonl`, true},      // 相对形态
		{`/ws/.gaea/other/x.jsonl`, false},    // .gaea 其他子树不拦
		{`/ws/sessions/a.jsonl`, false},       // 无 .gaea 前缀不拦
		{`/ws/.gaea-sessions/a.jsonl`, false}, // 段相等才命中
		{`/ws/notes.md`, false},
	}
	for _, c := range cases {
		if got := SessionDirFenceRule(c.path); got != c.want {
			t.Errorf("SessionDirFenceRule(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// TestReadFileSessionFence 集成：read_file 读 .gaea/sessions 件结构化拒绝
// （SESSION_DENIED+自愈指引指 session_search），相邻 .gaea 件不受影响。
func TestReadFileSessionFence(t *testing.T) {
	dir := t.TempDir()
	sessions := filepath.Join(dir, ".gaea", "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	big := filepath.Join(sessions, "s1.gaea.jsonl")
	if err := os.WriteFile(big, []byte("{\"kind\":\"message\",\"text\":\"会话原文行\"}\n"), 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}
	plain := filepath.Join(dir, ".gaea", "notes.md")
	if err := os.WriteFile(plain, []byte("普通工作区件"), 0o644); err != nil {
		t.Fatalf("write plain file: %v", err)
	}

	tf := readFile{}
	_, err := tf.Execute(context.Background(), mustArgs(t, map[string]any{"path": big}))
	if err == nil {
		t.Fatal("读 .gaea/sessions 件应被围栏拒绝")
	}
	var f *tool.Failure
	if !errors.As(err, &f) || f.Code != tool.CodeDenied {
		t.Fatalf("拒绝应为 CodeDenied Failure，got %v", err)
	}
	if !strings.Contains(f.Err.Error(), "SESSION_DENIED") || !strings.Contains(f.Err.Error(), "session_search") {
		t.Fatalf("拒绝文案缺自愈指引：%v", f.Err)
	}

	if _, err := tf.Execute(context.Background(), mustArgs(t, map[string]any{"path": plain})); err != nil {
		t.Fatalf(".gaea 非 sessions 子树不应被拦：%v", err)
	}
}

// TestGrepSessionFenceSkips grep 显式 root 指进 sessions 时逐件跳过并记账。
func TestGrepSessionFenceSkips(t *testing.T) {
	dir := t.TempDir()
	sessions := filepath.Join(dir, ".gaea", "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessions, "s1.gaea.jsonl"), []byte("早啊，包含 关键词 的行\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	tf := grepTool{workDir: dir}
	out, err := tf.Execute(context.Background(), mustArgs(t, map[string]any{"pattern": "关键词", "path": sessions}))
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if !strings.Contains(out, "session-fence skipped 1 file(s)") {
		t.Fatalf("grep 收据应记会话围栏跳过：%s", out)
	}
	if strings.Contains(out, "早啊") {
		t.Fatal("被围栏跳过的件不得产生命中")
	}
}
