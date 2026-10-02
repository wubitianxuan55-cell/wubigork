package app

// ── GA6-09 app 面收尾用例：GaeaCostSearchPage 的错误通道 ──────────
//
// 生产代码已就位（memory_hub.go：CostSearchPage.Error + costSearchAll 回传 readErr），
// 本文件补 Error 字段的语义钉子：
//   1. 成功时 Error 恒为空串（omitempty ⇒ 旧前端零行为变化）；
//   2. 成本库完全取不到时 Error 非空（不再把「读失败」伪装成「无匹配条目」），
//      且 Items/Total 仍按已读到部分如实给出（此处为 0 条 + total 0）。
//
// 失败注入：直接关闭测试注入的成本库底层连接池（hubCostStore 走
// SetCostStoreForTest 的 store，见 gaea_cost_search_page_test.go 工装），
// 此后 Query 必失败——这是 app 面唯一能稳定注入 Query 级失败的通道。

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGaeaCostSearchPageErrorEmptyOnSuccess 成功路径 Error 必须为空串。
func TestGaeaCostSearchPageErrorEmptyOnSuccess(t *testing.T) {
	a := newCostSearchPageApp(t)
	p := a.GaeaCostSearchPage("", "", "", "", 1, 5, 0)
	if p.Error != "" {
		t.Fatalf("成功路径 Error = %q, want 空串", p.Error)
	}
	if p.Total != 12 || len(p.Items) != 5 {
		t.Fatalf("成功路径 Total/Items = %d/%d, want 12/5", p.Total, len(p.Items))
	}
	// 关键词零命中也是「成功」：Error 仍是空串（不得与读取失败混淆）。
	empty := a.GaeaCostSearchPage("不存在的关键词xyz", "", "", "", 1, 10, 0)
	if empty.Error != "" {
		t.Fatalf("零命中（读取成功）Error = %q, want 空串", empty.Error)
	}
	if empty.Total != 0 || len(empty.Items) != 0 {
		t.Fatalf("零命中应空页: total=%d items=%d", empty.Total, len(empty.Items))
	}
}

// TestGaeaCostSearchPageReportsReadError 读取失败（底层库已关闭）：
// Error 非空、Items/Total 如实（0 条 / total 0），不 panic、不静默当空结果。
func TestGaeaCostSearchPageReportsReadError(t *testing.T) {
	a := newCostSearchPageApp(t)
	if err := a.hubCostStore().DB().Close(); err != nil {
		t.Fatalf("关闭注入成本库: %v", err)
	}

	p := a.GaeaCostSearchPage("", "", "", "", 1, 5, 0)
	if p.Error == "" {
		t.Fatal("读取失败时 Error 必须非空（否则用户会误判为无匹配条目）")
	}
	if p.Items == nil {
		t.Fatal("Items 必须是已初始化切片（绑定面 JSON 恒为 []，不得为 null）")
	}
	if len(p.Items) != 0 || p.Total != 0 {
		t.Fatalf("完全取不到时应如实回 0 条/total 0，got items=%d total=%d", len(p.Items), p.Total)
	}
}

// TestGaeaCostSearchPagePartialDataKeepsItemsAndTotal 部分行解析失败：
// Error 说明「少了几行」，Items/Total 仍按已读到的部分给出（少的那条不计入）。
func TestGaeaCostSearchPagePartialDataKeepsItemsAndTotal(t *testing.T) {
	a := newCostSearchPageApp(t)
	// 直写坏行绕过 Save 的类型约束（Save 参数是 float64，写不进文本）。
	// 保留其余 12 条好行，只坏 e06 一条。
	if _, err := a.hubCostStore().DB().Exec(
		"UPDATE cost_entries SET price = 'not-a-number' WHERE name = 'e06'"); err != nil {
		t.Fatalf("注入坏行: %v", err)
	}

	p := a.GaeaCostSearchPage("", "", "", "", 1, 100, 0)
	if p.Error == "" {
		t.Fatal("有行解析失败时 Error 必须非空（结果少了几行）")
	}
	if p.Total != 11 || len(p.Items) != 11 {
		t.Fatalf("部分数据应回 11 条（12 条中 1 条坏），got total=%d items=%d", p.Total, len(p.Items))
	}
	for _, it := range p.Items {
		if it.Name == "e06" {
			t.Fatalf("坏行不应出现在结果里: %+v", p.Items)
		}
	}
}

// TestGaeaCostSearchPagePartialDataWithLimit 部分数据 + 分页：切片基于已读到的
// 部分（total 也是部分口径），错误照常上报——不因分页而吞掉错误通道。
func TestGaeaCostSearchPagePartialDataWithLimit(t *testing.T) {
	a := newCostSearchPageApp(t)
	if _, err := a.hubCostStore().DB().Exec(
		"UPDATE cost_entries SET price = 'not-a-number' WHERE name = 'e06'"); err != nil {
		t.Fatalf("注入坏行: %v", err)
	}

	p := a.GaeaCostSearchPage("", "", "", "", 1, 3, 0)
	if p.Error == "" {
		t.Fatal("分页路径同样必须上报读取失败")
	}
	if p.Total != 11 || len(p.Items) != 3 {
		t.Fatalf("部分数据分页: total=%d items=%d, want 11/3", p.Total, len(p.Items))
	}
}

// TestCostSearchPageErrorFieldIsStable 契约钉子：Error 是 omitempty 的附加字段，
// 空值时 JSON 里不出现（旧前端零行为变化），非空时如实下发。
func TestCostSearchPageErrorFieldIsStable(t *testing.T) {
	ok, err := json.Marshal(CostSearchPage{Items: []CostSummary{}, Total: 0})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(ok), "error") {
		t.Fatalf("空 Error 不得出现在 JSON（omitempty）: %s", ok)
	}
	bad, err := json.Marshal(CostSearchPage{Items: []CostSummary{}, Total: 0, Error: "读取失败"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(bad), `"error":"读取失败"`) {
		t.Fatalf("非空 Error 应如实下发: %s", bad)
	}
}
