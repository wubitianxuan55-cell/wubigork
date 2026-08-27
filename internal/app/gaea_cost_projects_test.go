package app

// T5-5b 测算项目 → 沉淀闭环：明细行（引用成本库单价）→ 保存版本 → 沉淀回成本库，
// 保留引用的规格/地区/口径，缺单价行不沉淀。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/db"
)

func TestGaeaCostEstimateSediment(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	costStore := cost.Open(gdb)
	projStore := costproject.Open(gdb)
	SetCostStoreForTest(costStore)
	SetCostProjectStoreForTest(projStore)
	defer ResetCostStoreForTest()
	defer ResetCostProjectStoreForTest()

	// 既有条目：沉淀时应保留其规格/地区/口径（引用的属性不丢）。
	if err := costStore.Save(cost.Entry{
		Name: "c30", Title: "C30 商品混凝土", Category: "水泥及水泥制品",
		CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Price: 480,
		Spec: "C30 泵送", Region: "成都市区", PriceType: "到场价", Status: "现行",
	}); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	pid, err := a.GaeaCostProjectSave(costproject.Project{Name: "某厂房土建测算", ProjectType: "房建", Scale: "2 万 m²"})
	if err != nil {
		t.Fatal(err)
	}
	itemID, err := a.GaeaCostEstimateItemSave(costproject.Item{
		ProjectID: pid, Name: "c30", Title: "C30 商品混凝土",
		CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Quantity: 100, Price: 500, EntryName: "c30",
	})
	if err != nil {
		t.Fatal(err)
	}
	// 缺单价行：不参与沉淀（对标「缺单价标记」）。
	_, _ = a.GaeaCostEstimateItemSave(costproject.Item{
		ProjectID: pid, Name: "glass", Title: "幕墙玻璃", Unit: "m²", Quantity: 10, Price: 0,
	})

	// 保存版本（版本留痕）
	v, err := a.GaeaCostEstimateVersionSave(pid, "初稿")
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != 1 || v.Total != 100*500 {
		t.Errorf("version = %+v", v)
	}
	if proj, _ := projStore.GetProject(pid); proj == nil || proj.Status != "已保存版本" {
		t.Fatalf("保存版本后状态 = %+v", proj)
	}

	// 沉淀：只沉淀选中的有价行
	n, err := a.GaeaCostEstimateSediment(pid, []int64{itemID})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("沉淀条数 = %d, want 1", n)
	}
	got, err := costStore.Get("c30")
	if err != nil {
		t.Fatal(err)
	}
	if got.Price != 500 || got.Region != "成都市区" || got.PriceType != "到场价" || got.Spec != "C30 泵送" {
		t.Errorf("沉淀后条目属性丢失: %+v", got)
	}
	if !strings.Contains(got.Source, "某厂房土建测算") {
		t.Errorf("来源未标注项目: %q", got.Source)
	}
	if proj, _ := projStore.GetProject(pid); proj == nil || proj.Status != "已沉淀" {
		t.Errorf("沉淀后项目状态 = %+v", proj)
	}
	// 缺单价行未被沉淀
	if _, err := costStore.Get("glass"); err == nil {
		t.Fatal("缺单价行不应被沉淀")
	}
}
