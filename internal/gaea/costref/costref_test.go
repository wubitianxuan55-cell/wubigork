package costref

import (
	"math"
	"testing"

	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/db"
)

func TestReviewNotesCRUD(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	id, err := s.Save(Note{Title: "C30 泵送价按到场价取", Conclusion: "成都市区到场价约 480-520", Boundary: "仅适用框架结构现浇", Risk: "冬季施工需加价", Evidence: "某厂房测算 V2", Confidence: "高", Status: "草稿", Category: "材料", ProjectType: "房建"})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatal("note id 无效")
	}
	if _, err := s.Save(Note{Title: ""}); err == nil {
		t.Fatal("空标题应拒绝")
	}
	// 更新 + 状态流转
	n := s.List("", "all")
	if len(n) != 1 || n[0].Status != "草稿" {
		t.Fatalf("list = %+v", n)
	}
	_, _ = s.Save(Note{ID: id, Title: n[0].Title, Conclusion: n[0].Conclusion, Status: "已确认", Confidence: "高", Category: n[0].Category})
	got := s.List("成都", "all")
	if len(got) != 1 || got[0].Status != "已确认" {
		t.Fatalf("keyword/status = %+v", got)
	}
	if got := s.List("", "已确认"); len(got) != 1 {
		t.Fatalf("status filter = %+v", got)
	}
	if got := s.List("不存在", "all"); len(got) != 0 {
		t.Fatalf("no-match filter = %+v", got)
	}
	// 引用计数
	if err := s.BumpRef(id); err != nil {
		t.Fatal(err)
	}
	if got := s.List("", "all"); len(got) != 1 || got[0].RefCount != 1 {
		t.Errorf("ref_count = %+v", got)
	}
	if err := s.Delete(id); err != nil {
		t.Fatal(err)
	}
	if got := s.List("", "all"); len(got) != 0 {
		t.Fatalf("delete 后 = %+v", got)
	}
}

func TestComputeIndicators(t *testing.T) {
	items := []costproject.Item{
		{Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Price: 460},
		{Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Price: 480},
		{Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Price: 500},
		{Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Price: 520},
		{Title: "HRB400 螺纹钢", CategoryPath: "材料/土建材料/钢材", Unit: "t", Price: 3900},
		{Title: "HRB400 螺纹钢", CategoryPath: "材料/土建材料/钢材", Unit: "t", Price: 0}, // 缺单价不参与
	}
	byTitle := ComputeIndicators(items, "title")
	if len(byTitle) != 2 {
		t.Fatalf("byTitle = %+v", byTitle)
	}
	c30 := byTitle[0]
	if c30.Key != "C30 商品混凝土" || c30.Samples != 4 || c30.Unit != "m³" {
		t.Errorf("c30 = %+v", c30)
	}
	if math.Abs(c30.Median-490) > 1e-9 || math.Abs(c30.Mean-490) > 1e-9 {
		t.Errorf("c30 median/mean = %v/%v", c30.Median, c30.Mean)
	}
	if math.Abs(c30.P25-475) > 1e-9 || math.Abs(c30.P75-505) > 1e-9 {
		t.Errorf("c30 quartiles = %v/%v", c30.P25, c30.P75)
	}
	rebar := byTitle[1]
	if rebar.Samples != 1 || rebar.Max != 3900 {
		t.Errorf("rebar（缺单价行排除）= %+v", rebar)
	}
	byCat := ComputeIndicators(items, "category")
	if len(byCat) != 1 || byCat[0].Key != "材料" || byCat[0].Samples != 5 {
		t.Errorf("byCategory = %+v", byCat)
	}
	if got := ComputeIndicators(nil, "title"); got != nil {
		t.Errorf("空输入应返回 nil，got %+v", got)
	}
}
