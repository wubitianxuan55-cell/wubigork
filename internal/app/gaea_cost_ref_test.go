package app

// 造价参考与复盘笔记：案例（有版本留痕的项目）参与指标聚合，临时工作稿不参与；
// 复盘笔记 CRUD + 引用计数。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/costref"
	"github.com/gaea/gaea/internal/gaea/db"
)

func TestGaeaCostIndicatorsAndNotes(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	projStore := costproject.Open(gdb)
	refStore := costref.Open(gdb)
	SetCostProjectStoreForTest(projStore)
	SetCostRefStoreForTest(refStore)
	defer ResetCostProjectStoreForTest()
	defer ResetCostRefStoreForTest()

	a := &App{}
	// 案例项目（保存过版本）
	pid, err := a.GaeaCostProjectSave(costproject.Project{Name: "厂房 A", ProjectType: "房建"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = a.GaeaCostEstimateItemSave(costproject.Item{ProjectID: pid, Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Quantity: 1, Price: 480})
	_, _ = a.GaeaCostEstimateItemSave(costproject.Item{ProjectID: pid, Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Quantity: 1, Price: 500})
	_, _ = a.GaeaCostEstimateVersionSave(pid, "V1")
	// 临时工作稿（未保存版本）：不参与指标
	pid2, _ := a.GaeaCostProjectSave(costproject.Project{Name: "厂房 B（编制中）", ProjectType: "房建"})
	_, _ = a.GaeaCostEstimateItemSave(costproject.Item{ProjectID: pid2, Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Quantity: 1, Price: 9999})

	ind := a.GaeaCostIndicators("title")
	if len(ind) != 1 || ind[0].Samples != 2 || ind[0].Median != 490 {
		t.Fatalf("indicators = %+v", ind)
	}
	// 复盘笔记
	nid, err := a.GaeaCostNoteSave(costref.Note{Title: "C30 泵送价区间", Conclusion: "480-520", Status: "草稿", Confidence: "中"})
	if err != nil {
		t.Fatal(err)
	}
	if nid <= 0 {
		t.Fatal("note id 无效")
	}
	if _, err := a.GaeaCostNoteSave(costref.Note{Title: ""}); err == nil {
		t.Fatal("空标题应拒绝")
	}
	notes := a.GaeaCostNoteList("泵送", "all")
	if len(notes) != 1 {
		t.Fatalf("notes = %+v", notes)
	}
	if err := a.GaeaCostNoteBumpRef(nid); err != nil {
		t.Fatal(err)
	}
	if got := a.GaeaCostNoteList("", "all"); len(got) != 1 || got[0].RefCount != 1 {
		t.Errorf("ref_count = %+v", got)
	}
	if err := a.GaeaCostNoteDelete(nid); err != nil {
		t.Fatal(err)
	}
	if got := a.GaeaCostNoteList("", "all"); len(got) != 0 {
		t.Errorf("delete 后 = %+v", got)
	}
}

// v4.6.1 归因对标：参考池排除本项目自身（防自对标），无参考项目时仍可调用。
func TestGaeaCostAttributionExcludesSelf(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	projStore := costproject.Open(gdb)
	SetCostProjectStoreForTest(projStore)
	defer ResetCostProjectStoreForTest()

	a := &App{}
	// 参考项目（已保存版本）：C30 中位 500
	refID, err := a.GaeaCostProjectSave(costproject.Project{Name: "厂房 A", ProjectType: "房建"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = a.GaeaCostEstimateItemSave(costproject.Item{ProjectID: refID, Title: "C30 商品混凝土", Unit: "m³", Quantity: 1, Price: 480})
	_, _ = a.GaeaCostEstimateItemSave(costproject.Item{ProjectID: refID, Title: "C30 商品混凝土", Unit: "m³", Quantity: 1, Price: 520})
	_, _ = a.GaeaCostEstimateVersionSave(refID, "V1")

	// 目标项目：C30 单价 600（高于参考 P75），自身不参与参考池
	targetID, err := a.GaeaCostProjectSave(costproject.Project{Name: "新项目", ProjectType: "房建"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = a.GaeaCostEstimateItemSave(costproject.Item{ProjectID: targetID, Title: "C30 商品混凝土", Unit: "m³", Quantity: 10, Price: 600})

	at, err := a.GaeaCostAttribution(targetID)
	if err != nil {
		t.Fatalf("attribution: %v", err)
	}
	if len(at.Items) != 1 || at.Items[0].Level != "高" {
		t.Fatalf("attribution items = %+v, want C30 高", at.Items)
	}
	if at.Items[0].RefSamples != 2 {
		t.Fatalf("参考样本 = %d, want 2（参考池含参考项目 2 条明细）", at.Items[0].RefSamples)
	}
	if len(at.TopDrivers) != 1 || at.TopDrivers[0].Title != "C30 商品混凝土" {
		t.Fatalf("topDrivers = %+v", at.TopDrivers)
	}
	// 对参考项目自身对标：参考池应排除自身（若污染，样本>0）
	atSelf, err := a.GaeaCostAttribution(refID)
	if err != nil {
		t.Fatalf("attribution(self): %v", err)
	}
	for _, it := range atSelf.Items {
		if it.RefSamples != 0 {
			t.Fatalf("自对标参考池应排除自身, got samples=%d", it.RefSamples)
		}
	}
}
