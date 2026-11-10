package builtin

// 刀1（D5，dsh-ops 蒸馏 2026-11-10）测试：三态跳过记账（excluded vs gap）+
// scanner 超长行误差账 + 取消诚实 + 缺口不发续读指引。

import (
	"context"
	"strings"
	"testing"
)

func TestGrepLongLineGapAccounted(t *testing.T) {
	ws, g := d0Workspace(t)
	// 长行前的命中可发射；>1MB 行让 scanner 截断，行后的命中旧实现静默丢失。
	var b strings.Builder
	b.WriteString("needle before long line\n")
	b.WriteString(strings.Repeat("x", 2*1024*1024) + "\n")
	b.WriteString("needle after long line\n")
	if err := writeFileForTest(ws, "long.txt", b.String()); err != nil {
		t.Fatal(err)
	}
	out, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "needle before long line") {
		t.Fatalf("长行前命中应保留: %q", out)
	}
	if !strings.Contains(out, "1 path(s) unreadable") {
		t.Fatalf("超长行截断应记缺口: %q", out)
	}
	if !strings.Contains(out, "scan=partial") {
		t.Fatalf("有缺口应 scan=partial: %q", out)
	}
	if strings.Contains(out, "needle after long line") {
		t.Fatalf("截断后的命中本就不该出现（gap 记账是重点）: %q", out)
	}
}

func TestGrepCancelledScanHonest(t *testing.T) {
	ws, g := d0Workspace(t)
	if err := writeFileForTest(ws, "a.txt", "needle\n"); err != nil {
		t.Fatal(err)
	}
	// 预取消 ctx：旧实现伪装完整「no matches」（假完备），新实现如实报中断。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, err := g.Execute(ctx, d0Args(t, map[string]any{"pattern": "needle"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "scan interrupted") {
		t.Fatalf("取消应如实报中断: %q", out)
	}
	if strings.Contains(out, "(no matches for") {
		t.Fatalf("被取消不得伪装完整无命中: %q", out)
	}
}

func TestGrepGapSuppressesContinuation(t *testing.T) {
	ws, g := d0Workspace(t)
	// 预算满页 + 有缺口：不发 offset 续读指引（覆盖是下界）。缺口文件须在
	// 预算耗尽**前**被访问——命名 0-long 前置（字典序先于 a/b/c）。
	var long strings.Builder
	long.WriteString(strings.Repeat("x", 2*1024*1024) + "\n")
	if err := writeFileForTest(ws, "0-long.txt", long.String()+"needle tail\n"); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		if err := writeFileForTest(ws, string(rune('a'+i-1))+".txt", "needle one\nneedle two\n"); err != nil {
			t.Fatal(err)
		}
	}
	out, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "scan=partial") || !strings.Contains(out, "unreadable") {
		t.Fatalf("缺口应记 partial: %q", out)
	}
	// 无缺口满页的续读指引格式是 "re-run with offset="——有缺口时不得出现。
	if strings.Contains(out, "re-run with offset=") {
		t.Fatalf("有缺口不得发续读指引: %q", out)
	}
	if !strings.Contains(out, "[truncated at 4 results") {
		t.Fatalf("预算截断尾行应在: %q", out)
	}
}
