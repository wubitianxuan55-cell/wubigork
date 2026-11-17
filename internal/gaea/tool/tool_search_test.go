package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// T8 接线验收：tool_search 端到端——隐藏工具搜得到、schema 随结果回喂
// （物化即文本）、搜索工具自身不进结果。

func TestToolSearchToolEndToEnd(t *testing.T) {
	reg := NewRegistry()
	reg.Add(stubTool{name: "visible-tool"})
	reg.Add(stubTool{name: "xlsx-export"})
	reg.Add(stubTool{name: "tool_search"})
	reg.Hide("xlsx-export") // 藏而可搜（boot 的 deferred 名单形态）

	ts := NewToolSearchTool(reg, 8)
	if ts.Name() != "tool_search" || !ts.ReadOnly() {
		t.Fatalf("工具元信息不符")
	}
	out, err := ts.Execute(context.Background(), json.RawMessage(`{"query":"xlsx 表格导出"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "xlsx-export") || !strings.Contains(out, "schema:") {
		t.Fatalf("结果应含目标工具与 schema 文本: %q", out)
	}
	// 搜索工具自身不出现在结果里。
	if strings.Contains(out, "tool_search") {
		t.Fatalf("tool_search 不应进索引结果")
	}
	// 空查询拒绝。
	if _, err := ts.Execute(context.Background(), json.RawMessage(`{"query":"  "}`)); err == nil {
		t.Fatalf("空查询应报错")
	}
}

func TestToolSearchNoHitMessage(t *testing.T) {
	reg := NewRegistry()
	reg.Add(stubTool{name: "only-one"})
	ts := NewToolSearchTool(reg, 8)
	out, err := ts.Execute(context.Background(), json.RawMessage(`{"query":"zzzqqq"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "no matching tools") {
		t.Fatalf("无命中应有可操作文案: %q", out)
	}
}
