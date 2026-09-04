package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
)

// GaeaSubagentContextView 绑定测试（2.5e 后半）：合法 sa_ ref → 子代理
// transcript 折叠为上下文快照；非法 ref / transcript 缺失诚实报错。
func TestGaeaSubagentContextView(t *testing.T) {
	dir := t.TempDir()
	// sessionDirForPath 要求 <root>/sessions/work 结构
	dir = filepath.Join(dir, ".gaea", "sessions", "work")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ref := "sa_20260817_100000_0000000001_a1a1a1a1"
	if !gaeaAgent.ValidRunRef(ref) {
		t.Fatalf("测试 ref 应通过 ValidRunRef：%q", ref)
	}
	subDir := filepath.Join(dir, "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "{\"kind\":\"request_header\",\"ts\":1,\"payload\":{\"system\":\"sys\",\"tools\":[],\"window\":1000000}}\n" +
		"{\"kind\":\"user_message\",\"ts\":2,\"payload\":{\"content\":\"子代理任务内容\"}}\n"
	if err := os.WriteFile(filepath.Join(subDir, ref+".jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	tl, err := a.GaeaSubagentContextView(filepath.Join(dir, "s1.jsonl"), ref)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if tl.Window != 1_000_000 {
		t.Fatalf("window = %d, want 1000000（transcript header）", tl.Window)
	}
	if !strings.Contains(tl.Nodes[0].Text, "sys") || len(tl.Nodes) == 0 {
		t.Fatalf("nodes 未折叠: %+v", tl.Nodes)
	}

	// transcript 缺失 → 空快照（前端空态）
	empty, err := a.GaeaSubagentContextView(filepath.Join(dir, "s1.jsonl"), "sa_20260817_100000_0000000002_b2b2b2b2")
	if err != nil {
		t.Fatalf("缺失 transcript 应不报错: %v", err)
	}
	if len(empty.Nodes) != 0 {
		t.Fatalf("缺失 transcript 应为空快照: %+v", empty)
	}

	// 非法 ref → 诚实报错
	if _, err := a.GaeaSubagentContextView(filepath.Join(dir, "s1.jsonl"), "../escape"); err == nil {
		t.Fatal("非法 ref 应报错")
	}
}
