package app

// v4.8 成本知识图谱绑定面：GaeaCostGraph 返回 JSON 串（CostGraphView），
// 取数复用 hub*Store（隔离注入验证 tree/entry 两种 scope）。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costinquiry"
	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/costref"
	"github.com/gaea/gaea/internal/gaea/db"
)

func TestGaeaCostGraphBinding(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	SetCostStoreForTest(cost.Open(gdb))
	SetCostProjectStoreForTest(costproject.Open(gdb))
	SetCostInquiryStoreForTest(costinquiry.Open(gdb))
	SetCostRefStoreForTest(costref.Open(gdb))
	defer ResetCostStoreForTest()
	defer ResetCostProjectStoreForTest()
	defer ResetCostInquiryStoreForTest()
	defer ResetCostRefStoreForTest()

	a := &App{}
	if err := a.hubCostStore().Save(cost.Entry{
		Name: "c30", Title: "C30 商品混凝土", Category: "土建", CategoryPath: "综合单价/土建",
		Unit: "m³", Price: 480, Status: "现行",
	}); err != nil {
		t.Fatal(err)
	}
	pid, err := a.GaeaCostProjectSave(costproject.Project{Name: "厂房 A", ProjectType: "房建"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaCostEstimateItemSave(costproject.Item{
		ProjectID: pid, Name: "c30-concrete", Title: "C30 商品混凝土", CategoryPath: "综合单价/土建",
		Unit: "m³", Quantity: 10, Price: 500, EntryName: "c30",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaCostInquirySave(costinquiry.Record{Title: "C30 商品混凝土", Unit: "m³", Price: 470, Source: "信息价"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaCostNoteSave(costref.Note{Title: "C30 泵送价区间", Category: "土建", Status: "草稿"}); err != nil {
		t.Fatal(err)
	}

	// tree：分类聚合 + 项目节点，JSON 串可解析。
	raw, err := a.GaeaCostGraph("tree", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	var tree costref.CostGraphView
	if err := json.Unmarshal([]byte(raw), &tree); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if tree.Stats.CountsByType["project"] != 1 || tree.Stats.CountsByType["category"] == 0 {
		t.Fatalf("tree 统计异常: %+v", tree.Stats)
	}

	// entry：项目展开（明细→条目 references + 询价 suggests）。
	rawEntry, err := a.GaeaCostGraph("entry", pid, 0)
	if err != nil {
		t.Fatal(err)
	}
	var g costref.CostGraphView
	if err := json.Unmarshal([]byte(rawEntry), &g); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	foundRef, foundSug := false, false
	for _, e := range g.Edges {
		if e.Type == "references" && e.Meta["matchedBy"] == "entry_name" {
			foundRef = true
		}
		if e.Type == "suggests" {
			foundSug = true
		}
	}
	if !foundRef || !foundSug {
		t.Fatalf("entry 展开缺边: %+v", g.Edges)
	}

	// limit 归一：<=0 → 600（不截断小数据）；非法 scope 回退 tree 聚合。
	rawDefault, err := a.GaeaCostGraph("entry", pid, -5)
	if err != nil || !strings.Contains(rawDefault, "\"nodes\"") {
		t.Fatalf("limit 归一失败: %v", err)
	}
}
