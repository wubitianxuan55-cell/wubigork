package builtin

// 刀2（D2，dsh-ops 蒸馏 2026-11-10 第二波）测试：自校验续读游标——
// 同树续页过 / 树变化冲突 / 查询身份变化拒绝 / 缺口不发 cursor。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// cursorFromHeader 从头行提取 cursor token。
func cursorFromHeader(t *testing.T, out string) string {
	t.Helper()
	rest, ok := strings.CutPrefix(out, "[grep page] ")
	if !ok {
		t.Fatalf("头行缺失: %q", out)
	}
	seg, _, _ := strings.Cut(rest, "\n")
	if tok, ok := strings.CutPrefix(seg, "offset="); ok {
		for _, part := range strings.Fields(tok) {
			if v, ok := strings.CutPrefix(part, "cursor=g1."); ok {
				return "g1." + v
			}
			if strings.HasPrefix(part, "cursor=") {
				return strings.TrimPrefix(part, "cursor=")
			}
		}
	}
	t.Fatalf("头行未带 cursor: %q", seg)
	return ""
}

func cursorWorkspace(t *testing.T) (string, grepTool) {
	t.Helper()
	ws, g := d0Workspace(t)
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		if err := writeFileForTest(ws, n+".txt", "needle one\nmiddle\nneedle two\n"); err != nil {
			t.Fatal(err)
		}
	}
	return ws, g
}

func TestGrepCursorRoundtrip(t *testing.T) {
	_, g := cursorWorkspace(t)
	// 第 1 页（满页 4/10）应发自校验 cursor。
	p1, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4}))
	if err != nil {
		t.Fatal(err)
	}
	tok := cursorFromHeader(t, p1)
	if !strings.HasPrefix(tok, "g1.") {
		t.Fatalf("cursor 应为 g1 域 token: %q", tok)
	}
	// 第 2 页带 cursor：同树应通过校验，与裸 offset=4 同结果。
	p2, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4, "cursor": tok}))
	if err != nil {
		t.Fatalf("同树续页应通过: %v", err)
	}
	p2ref, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4, "offset": 4}))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != p2ref {
		t.Fatalf("cursor 页应与 offset 页同结果:\ncursor: %q\noffset: %q", p2, p2ref)
	}
	if !strings.Contains(p2, "c.txt:1:") || !strings.Contains(p2, "d.txt:3:") {
		t.Fatalf("第 2 页应是 c/d 的命中: %q", p2)
	}
	// 第 2 页自己也发 cursor（仍有剩余）。
	if got := cursorFromHeader(t, p2); !strings.HasPrefix(got, "g1.") {
		t.Fatalf("第 2 页应继续发 cursor: %q", p2)
	}
}

func TestGrepCursorTreeChangeConflict(t *testing.T) {
	ws, g := cursorWorkspace(t)
	p1, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4}))
	if err != nil {
		t.Fatal(err)
	}
	tok := cursorFromHeader(t, p1)
	// 修改前缀文件 a.txt（第 2 页的前缀快照覆盖 a+b）——身份漂移必须冲突。
	if err := writeFileForTest(ws, "a.txt", "needle one\nmiddle\nneedle two\nextra line changes identity\n"); err != nil {
		t.Fatal(err)
	}
	_, err = g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4, "cursor": tok}))
	if err == nil {
		t.Fatal("树已变化，cursor 应冲突")
	}
	f, ok := tool.AsFailure(err)
	if !ok || f.Code != tool.CodeConflict {
		t.Fatalf("应为 conflict 结构化失败: %v", err)
	}
	if f.EffectState != tool.EffectNone || !f.Retryable {
		t.Fatalf("只读工具冲突应为 none+可重试(重跑): %+v", f)
	}
}

func TestGrepCursorBindMismatch(t *testing.T) {
	_, g := cursorWorkspace(t)
	p1, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4}))
	if err != nil {
		t.Fatal(err)
	}
	tok := cursorFromHeader(t, p1)
	// 换 pattern 续页：bind 不匹配=「游标属于另一个查询」。
	_, err = g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "middle", "max_results": 4, "cursor": tok}))
	if err == nil {
		t.Fatal("参数变化，cursor 应被拒")
	}
	f, ok := tool.AsFailure(err)
	if !ok || f.Code != tool.CodeValidationError {
		t.Fatalf("应为 validation 结构化失败: %v", err)
	}
	// 伪造 token（mac 必不过）。
	_, err = g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "cursor": "g1.dead.dead.4.beef"}))
	if err == nil {
		t.Fatal("伪造 cursor 必须被拒")
	}
}

func TestGrepGapSuppressesCursorEmission(t *testing.T) {
	ws, g := cursorWorkspace(t)
	var long strings.Builder
	long.WriteString(strings.Repeat("x", 2*1024*1024) + "\n")
	// 0-long 字典序最先被访问且产生缺口；满页后不发 cursor。
	if err := writeFileForTest(ws, "0-long.txt", long.String()+"needle tail\n"); err != nil {
		t.Fatal(err)
	}
	out, err := g.Execute(context.Background(), d0Args(t, map[string]any{"pattern": "needle", "max_results": 4}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "cursor=g1.") || strings.Contains(out, "cursor=") {
		t.Fatalf("有缺口不得发 cursor: %q", out)
	}
}
