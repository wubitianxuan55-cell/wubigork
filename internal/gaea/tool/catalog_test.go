package tool

import (
	"encoding/json"
	"reflect"
	"testing"
)

// T8（codex 蒸馏）验收：三轮不变零输出 / 新增只报一个 / 命名空间吞并 /
// 检索确定序 / 搜索工具不进索引。

func fakeDefs(names ...string) map[string]string {
	out := map[string]string{}
	for _, n := range names {
		out[n] = CatalogHash(json.RawMessage(`{"name":"` + n + `"}`))
	}
	return out
}

// ① 连续三轮工具集不变：后两轮 added/removed 全空（工具部分上下文逐字节相同）。
func TestCatalogDiffUnchanged(t *testing.T) {
	c := NewCatalog()
	first := fakeDefs("alpha", "beta", "gamma")
	a, r, changed := c.Diff(first)
	if !changed || len(a) != 3 || len(r) != 0 {
		t.Fatalf("首轮应全量 added: %v %v %v", a, r, changed)
	}
	for round := 0; round < 2; round++ {
		a, r, changed = c.Diff(fakeDefs("alpha", "beta", "gamma"))
		if changed || len(a) != 0 || len(r) != 0 {
			t.Fatalf("第 %d 轮应零输出: %v %v %v", round, a, r, changed)
		}
	}
}

// ② 新增一个工具 → 只有它出现在 added；③ 定义变了也算 added（哈希不同即重发）。
func TestCatalogDiffAddedAndChanged(t *testing.T) {
	c := NewCatalog()
	c.Diff(fakeDefs("alpha", "beta"))
	a, r, changed := c.Diff(fakeDefs("alpha", "beta", "delta"))
	if !changed || len(a) != 1 || a[0] != "delta" || len(r) != 0 {
		t.Fatalf("新增应只报 delta: %v %v %v", a, r, changed)
	}
	// 定义变化（同名单独哈希不同）→ 重发该工具。
	a, _, _ = c.Diff(fakeDefs("alpha-v2", "beta", "delta"))
	if len(a) != 1 || a[0] != "alpha-v2" {
		t.Fatalf("定义变化应重发: %v", a)
	}
	// 删除。
	a, r, _ = c.Diff(fakeDefs("alpha-v2", "delta"))
	if len(a) != 0 || len(r) != 1 || r[0] != "beta" {
		t.Fatalf("删除应报 beta: %v %v", a, r)
	}
}

// ③ 整命名空间移除吞并成员（ns/* 一条，不出现 N 条）。
func TestCoalesceRemoved(t *testing.T) {
	removed := []string{"mcp.office.word", "mcp.office.excel", "bash"}
	got := CoalesceRemoved(removed, func(name string) string {
		if name == "bash" {
			return ""
		}
		return "mcp.office"
	})
	want := []string{"bash", "mcp.office/*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// 单成员命名空间不吞并。
	got = CoalesceRemoved([]string{"mcp.solo.x"}, func(string) string { return "mcp.solo" })
	if !reflect.DeepEqual(got, []string{"mcp.solo.x"}) {
		t.Fatalf("单成员不应吞并: %v", got)
	}
}

// ④ 检索：40 条目中查询命中目标进 top-8；同查询两次顺序一致；得分非零才返回。
func TestSearchIndexBM25(t *testing.T) {
	var entries []SearchEntry
	for i := 0; i < 40; i++ {
		entries = append(entries, SearchEntry{
			Name:       "tool-" + string(rune('a'+i)),
			SearchText: "普通工具 描述占位 内容 placeholder number",
		})
	}
	entries = append(entries, SearchEntry{
		Name:       "xlsx-export",
		SearchText: "xlsx-export 导出 xlsx 表格 excel 电子表格 export spreadsheet",
	})
	ix := NewSearchIndex(entries)
	got := ix.Search("导出 xlsx 表格", 8)
	if len(got) == 0 || got[0].Name != "xlsx-export" {
		t.Fatalf("目标应排第一: %+v", got)
	}
	again := ix.Search("导出 xlsx 表格", 8)
	if len(again) != len(got) {
		t.Fatalf("两次结果长度漂移")
	}
	for i := range got {
		if got[i].Name != again[i].Name {
			t.Fatalf("确定序破坏 @%d", i)
		}
	}
	// 完全无关查询不硬凑结果。
	if nohit := ix.Search("zzzqqq", 8); len(nohit) != 0 {
		t.Fatalf("无命中应返回空: %+v", nohit)
	}
}

// ⑤ BuildSearchIndex 排除 tool_search/websearch；隐藏工具进索引（藏而可搜）。
func TestBuildSearchIndexExclusions(t *testing.T) {
	reg := NewRegistry()
	reg.Add(stubTool{name: "tool_search"})
	reg.Add(stubTool{name: "websearch"})
	reg.Add(stubTool{name: "real-tool"})
	reg.Hide("real-tool") // 藏起来但模型应能搜到
	ix := BuildSearchIndex(reg)
	got := ix.Search("real tool", 8)
	if len(got) != 1 || got[0].Name != "real-tool" {
		t.Fatalf("隐藏工具应可搜: %+v", got)
	}
	// 排除纪律：搜索工具自身不进索引——搜 "search" 零命中。
	if s := ix.Search("search", 8); len(s) != 0 {
		t.Fatalf("tool_search/websearch 不应进索引: %+v", s)
	}
}
