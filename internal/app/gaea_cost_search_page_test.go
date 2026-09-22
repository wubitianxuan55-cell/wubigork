package app

// v4.386 造价库分页绑定用例：GaeaCostSearchPage 在全量检索管线之上
// 排序切片。钉死四件事——分页切片与 total、排序键全序（name tie-break，
// 跨页稳定的前提）、limit/offset 钳制、空库诚实空页。语义召回/精排路径
// 依赖本地模型，不在本刀测试面（管线复用 costSearchAll，另有既有覆盖）。

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/modelengine"
)

func seedCostPageEntries(t *testing.T, a *App) {
	t.Helper()
	// 12 条：价格递增与 name 序刻意不同（钉排序确实发生）；
	// b/b2 同价（钉 price 键的 name tie-break 全序）。UpdatedAt 不种——
	// Save 恒盖 time.Now，需要确定时间序的用例用 SQL 直铺（见下）。
	entries := []cost.Entry{
		{Name: "e01", Title: "挖掘机租赁", Price: 900},
		{Name: "e02", Title: "C30 商品混凝土", Price: 480},
		{Name: "e03", Title: "钢筋 HRB400", Price: 4200},
		{Name: "b", Title: "普工", Price: 120},
		{Name: "b2", Title: "技工", Price: 120},
		{Name: "e06", Title: "水泥 P.O42.5", Price: 390},
		{Name: "e07", Title: "标准砖", Price: 0.4},
		{Name: "e08", Title: "模板覆膜板", Price: 45},
		{Name: "e09", Title: "脚手架钢管", Price: 65},
		{Name: "e10", Title: "防水卷材", Price: 28},
		{Name: "e11", Title: "沥青混合料", Price: 520},
		{Name: "e12", Title: "级配碎石", Price: 85},
	}
	for _, e := range entries {
		e.Status = "现行"
		if err := a.hubCostStore().Save(e); err != nil {
			t.Fatalf("seed %s: %v", e.Name, err)
		}
	}
}

func newCostSearchPageApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	SetCostStoreForTest(cost.Open(gdb))
	t.Cleanup(ResetCostStoreForTest)
	// 双重隔离（坑：App 嵌 *core，nil core 下 a.engineMgr 字段访问即
	// panic；而 NewManager 种子目录 herdsman 恒 Enabled+localhost:8080——
	// 本机 embedding 服务在跑时，零命中查询会走真实语义管线：Stale 清
	// 真实库 cost 向量行 + Ensure 写入假向量。必须显式禁用引擎把通道
	// 断死，测试只剩 SQL+Go 过滤路径）。
	mgr := modelengine.NewManager("", "")
	if eng, ok := mgr.GetEngine("herdsman"); ok {
		eng.Enabled = false
		if err := mgr.SaveEngine(*eng); err != nil {
			t.Fatalf("disable herdsman: %v", err)
		}
	}
	a := &App{core: &core{engineMgr: mgr}}
	seedCostPageEntries(t, a)
	return a
}

// TestGaeaCostSearchPageSlicesAndTotal：默认序（name）分页切片连续无缝、
// total 恒为过滤后总数；翻页拼回全量。
func TestGaeaCostSearchPageSlicesAndTotal(t *testing.T) {
	a := newCostSearchPageApp(t)
	p1 := a.GaeaCostSearchPage("", "", "", "", 1, 5, 0)
	if p1.Total != 12 {
		t.Fatalf("Total = %d, want 12", p1.Total)
	}
	// name 序：b < b2 < e01 < e02 < e03（第 5 条=e03）。
	if len(p1.Items) != 5 || p1.Items[0].Name != "b" || p1.Items[4].Name != "e03" {
		t.Fatalf("page1 first/last = %s/%s, len=%d", p1.Items[0].Name, p1.Items[4].Name, len(p1.Items))
	}
	p2 := a.GaeaCostSearchPage("", "", "", "", 1, 5, 5)
	p3 := a.GaeaCostSearchPage("", "", "", "", 1, 5, 10)
	if len(p2.Items) != 5 || len(p3.Items) != 2 {
		t.Fatalf("page2/page3 len = %d/%d, want 5/2", len(p2.Items), len(p3.Items))
	}
	all := append(append([]CostSummary{}, p1.Items...), append(p2.Items, p3.Items...)...)
	for i := 1; i < len(all); i++ {
		if all[i-1].Name >= all[i].Name {
			t.Fatalf("pages not continuous name order at %d: %s >= %s", i, all[i-1].Name, all[i].Name)
		}
	}
}

// TestGaeaCostSearchPageSortTotalOrder：排序键生效 + 同键 name tie-break
// （price 同价的 b/b2 顺序不随 sortDir 翻转）；降序整体翻转。
func TestGaeaCostSearchPageSortTotalOrder(t *testing.T) {
	a := newCostSearchPageApp(t)
	asc := a.GaeaCostSearchPage("", "", "", "price", 1, 100, 0)
	if asc.Total != 12 || len(asc.Items) != 12 {
		t.Fatalf("asc total/len = %d/%d", asc.Total, len(asc.Items))
	}
	if asc.Items[0].Name != "e07" || asc.Items[0].Price != 0.4 {
		t.Fatalf("asc[0] = %s(%.2f), want e07(0.40) 最便宜", asc.Items[0].Name, asc.Items[0].Price)
	}
	// 同价 120 的 b/b2：升序 b 前 b2 后。
	bi, b2i := -1, -1
	for i, e := range asc.Items {
		if e.Name == "b" {
			bi = i
		}
		if e.Name == "b2" {
			b2i = i
		}
	}
	if bi == -1 || b2i == -1 || bi > b2i {
		t.Fatalf("tie-break asc: b@%d b2@%d, want b before b2", bi, b2i)
	}
	desc := a.GaeaCostSearchPage("", "", "", "price", -1, 100, 0)
	if desc.Items[0].Name != "e03" || desc.Items[0].Price != 4200 {
		t.Fatalf("desc[0] = %s(%.2f), want e03(4200) 最贵", desc.Items[0].Name, desc.Items[0].Price)
	}
	for i, e := range desc.Items {
		if e.Name != asc.Items[len(asc.Items)-1-i].Name {
			t.Fatalf("desc not exact reverse of asc at %d", i)
		}
	}
}

// TestGaeaCostSearchPageClamps：limit<=0 → 100、limit>200 → 200、
// offset<0 → 0、offset 越界 → 空页但 total 仍在。
func TestGaeaCostSearchPageClamps(t *testing.T) {
	a := newCostSearchPageApp(t)
	if p := a.GaeaCostSearchPage("", "", "", "", 1, 0, 0); len(p.Items) != 12 {
		t.Fatalf("limit<=0 → default 100 (12 条全回), got %d", len(p.Items))
	}
	if p := a.GaeaCostSearchPage("", "", "", "", 1, 999, 0); len(p.Items) != 12 {
		t.Fatalf("limit>200 → clamp 200 (12 条全回), got %d", len(p.Items))
	}
	if p := a.GaeaCostSearchPage("", "", "", "", 1, 5, -3); len(p.Items) != 5 || p.Items[0].Name != "b" {
		t.Fatalf("offset<0 → 0, got len=%d first=%s", len(p.Items), p.Items[0].Name)
	}
	if p := a.GaeaCostSearchPage("", "", "", "", 1, 5, 50); len(p.Items) != 0 || p.Total != 12 {
		t.Fatalf("offset 越界 → 空页但 total=12, got len=%d total=%d", len(p.Items), p.Total)
	}
	if p := a.GaeaCostSearchPage("不存在的关键词xyz", "", "", "", 1, 10, 0); p.Total != 0 || len(p.Items) != 0 {
		t.Fatalf("零命中 → 空页 total=0, got len=%d total=%d", len(p.Items), p.Total)
	}
}

// TestGaeaCostSearchPageUnknownSortKeyKeepsPipelineOrder：不认识的
// sortKey 不排序（防御前端传错键），保持管线序（name 序前缀）。
func TestGaeaCostSearchPageUnknownSortKeyKeepsPipelineOrder(t *testing.T) {
	a := newCostSearchPageApp(t)
	p := a.GaeaCostSearchPage("", "", "", "bogus", 1, 3, 0)
	if p.Items[0].Name != "b" {
		t.Fatalf("unknown sortKey should keep name order, first=%s", p.Items[0].Name)
	}
}

// TestGaeaCostSearchPageFilterAndSortCombined：status 过滤收窄 total +
// updatedAt 排序组合（与前端表格「更新」列口径一致）。updated_at 以
// RFC3339 秒级落盘且 Save 恒盖 time.Now——同测试内的保存回读后全同秒，
// 故此处用 SQL 直铺已知时间戳（name 倒序铺，钉排序与 name 序不同）。
func TestGaeaCostSearchPageFilterAndSortCombined(t *testing.T) {
	a := newCostSearchPageApp(t)
	if err := a.hubCostStore().Save(cost.Entry{Name: "d01", Title: "草稿条目", Price: 10, Status: "草稿"}); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	names := []string{"b", "b2", "d01", "e01", "e02", "e03", "e06", "e07", "e08", "e09", "e10", "e11", "e12"}
	gdb := a.hubCostStore().DB()
	for i, n := range names {
		if _, err := gdb.Exec("UPDATE cost_entries SET updated_at = ? WHERE name = ?",
			base.Add(time.Duration(i)*time.Hour).Format(time.RFC3339), n); err != nil {
			t.Fatalf("stamp %s: %v", n, err)
		}
	}
	p := a.GaeaCostSearchPage("", "", "草稿", "updatedAt", -1, 10, 0)
	if p.Total != 1 || len(p.Items) != 1 || p.Items[0].Name != "d01" {
		t.Fatalf("status filter + sort: total=%d items=%d", p.Total, len(p.Items))
	}
	all := a.GaeaCostSearchPage("", "", "", "updatedAt", 1, 100, 0)
	if all.Total != 13 || all.Items[0].Name != "b" || all.Items[12].Name != "e12" {
		t.Fatalf("updatedAt asc: total=%d first=%s last=%s, want 13/b/e12",
			all.Total, all.Items[0].Name, all.Items[len(all.Items)-1].Name)
	}
}
