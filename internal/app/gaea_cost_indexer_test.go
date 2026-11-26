package app

// gaea_cost_indexer_test.go — 造价库向量索引器的应用层接线：随启动的后台索引器、状态与
// 补齐绑定、清单检索绑定。模型依赖走测试注入缝（假向量化：同义词落同一维）。

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/vecindex"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

func waitCostIndex(t *testing.T, cond func() bool) {
	t.Helper()
	// 写入后的增量对账要等 3 秒合并窗口，上限放到 6 秒
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待 6 秒内索引条件未满足")
}

func indexKindView(st CostIndexStatusView, kind string) CostIndexKindView {
	for _, k := range st.Kinds {
		if k.Kind == kind {
			return k
		}
	}
	return CostIndexKindView{}
}

// 索引器随启动先对一轮；之后库的写入打脏 → 增量对账；状态面读得到条数；清单检索走通。
func TestCostIndexer_StartsAndTracksState(t *testing.T) {
	a, gdb, closeDB := workcostTestAppDB(t)
	a.officeState = &officeState{core: &core{}}
	s := a.hubWorkcostStore()
	if _, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMaterial, Title: "商品混凝土", Unit: "m3"}); err != nil {
		t.Fatal(err)
	}
	vs := vecindex.Open(gdb)
	restore := SetCostIndexForTest(vs, func() workcost.SearchEmbedder { return costBagEmbedder{} }, nil)
	a.startCostIndexer()
	t.Cleanup(func() {
		if a.officeState.costIndexStop != nil {
			a.officeState.costIndexStop()
		}
		vecindex.SetDefault(nil)
		restore()
		vecindex.Forget(gdb)
		closeDB()
	})

	waitCostIndex(t, func() bool { _, ok := a.costIndexer().Last(workcost.IndexKindResource); return ok })
	st := a.GaeaCostIndexStatus()
	if !st.Running || st.Model != "bge-m3" || len(st.Kinds) != 3 {
		t.Fatalf("状态面: %+v", st)
	}
	if k := indexKindView(st, workcost.IndexKindResource); k.Label != "工料机" || k.Count != 1 || k.Total != 1 || k.LastError != "" {
		t.Fatalf("工料机类目状态: %+v", k)
	}
	if a.startCostIndexer(); a.costIndexer().Model() != "bge-m3" {
		t.Fatal("重复启动应幂等")
	}

	// 写入 → 句柄打脏 → 后台增量对账
	if _, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMachine, Title: "单斗挖掘机 挖土", Unit: "m3"}); err != nil {
		t.Fatal(err)
	}
	waitCostIndex(t, func() bool { return indexKindView(a.GaeaCostIndexStatus(), workcost.IndexKindResource).Count == 2 })

	// 显式补齐：幂等，返回最新状态
	if st, err := a.GaeaCostIndexBackfill(""); err != nil || len(st.Kinds) != 3 {
		t.Fatalf("补齐: %+v err=%v", st, err)
	}
	if st, err := a.GaeaCostIndexBackfill(workcost.IndexKindResource); err != nil || indexKindView(st, workcost.IndexKindResource).Count != 2 {
		t.Fatalf("补齐单个类目: %+v err=%v", st, err)
	}

	// 清单检索绑定：索引就绪后同义说法可召回；project_id 过滤照常
	pid, err := s.UpsertBillProject("测算P", "", "", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "WP01", Title: "混凝土垫层", Unit: "m3", Quantity: 10}); err != nil {
		t.Fatal(err)
	}
	waitCostIndex(t, func() bool { return indexKindView(a.GaeaCostIndexStatus(), workcost.IndexKindBill).Count == 1 })
	if got := a.GaeaWorkcostBillSearch("砼", "", 0); len(got) != 1 || got[0].Title != "混凝土垫层" {
		t.Fatalf("清单检索应召回同义说法: %+v", got)
	}
	if got := a.GaeaWorkcostBillSearch("砼", "", 999); len(got) != 0 {
		t.Fatalf("project_id 过滤: %+v", got)
	}
	if got := a.GaeaWorkcostBillSearch("", "", 0); len(got) != 1 || got[0].QuotaCodes == nil {
		t.Fatalf("不带关键词按列表返回: %+v", got)
	}
}

// 模型不可用：补齐如实报错；状态面把最近一轮的错误带出来；类目名不认识也报错；索引器没启动报错。
func TestCostIndexBackfill_ReportsUnavailable(t *testing.T) {
	a, gdb, closeDB := workcostTestAppDB(t)
	a.officeState = &officeState{core: &core{}}
	restore := SetCostIndexForTest(vecindex.Open(gdb), func() workcost.SearchEmbedder { return nil }, nil)
	a.startCostIndexer()
	t.Cleanup(func() {
		if a.officeState.costIndexStop != nil {
			a.officeState.costIndexStop()
		}
		vecindex.SetDefault(nil)
		restore()
		vecindex.Forget(gdb)
		closeDB()
	})

	if _, err := a.GaeaCostIndexBackfill(""); err == nil || !strings.Contains(err.Error(), "不可用") {
		t.Fatalf("模型不可用时补齐应如实报错: %v", err)
	}
	// 后台那一轮也失败了：状态面带出错误，条数为 0
	waitCostIndex(t, func() bool {
		_, ok := a.costIndexer().LastError(workcost.IndexKindQuota)
		return ok
	})
	st := a.GaeaCostIndexStatus()
	if !st.Running || indexKindView(st, workcost.IndexKindQuota).LastError == "" || indexKindView(st, workcost.IndexKindBill).Count != 0 {
		t.Fatalf("状态面应带出最近一轮的错误: %+v", st)
	}
	if _, err := a.GaeaCostIndexBackfill("wc_nope"); err == nil || !strings.Contains(err.Error(), "不支持的类目") {
		t.Fatalf("类目名不认识应报错: %v", err)
	}
	bare := &App{officeState: &officeState{core: &core{}}}
	if _, err := bare.GaeaCostIndexBackfill(""); err == nil || !strings.Contains(err.Error(), "没有启动") {
		t.Fatalf("索引器没启动应报错: %v", err)
	}
	if st := bare.GaeaCostIndexStatus(); st.Running || len(st.Kinds) != 3 {
		t.Fatalf("没启动时状态面应如实报未运行、类目仍列全: %+v", st)
	}
}
