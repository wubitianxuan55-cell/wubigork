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
