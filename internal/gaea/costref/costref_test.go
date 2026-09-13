package costref

import (
	"encoding/json"
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

// TestWireShapeCamelCase 线上 JSON 形状回归锁（v4.276）：Note/Indicator 由
// GaeaCostNoteList/GaeaCostIndicators 直达前端，曾因缺 json 标签线上 PascalCase、
// 前端 camelCase 全读空（复盘笔记/造价参考视图断裂；保存方向 Unmarshal 大小写
// 不敏故录入正常未暴露，v4.269 同款模式）。
func TestWireShapeCamelCase(t *testing.T) {
	nb, err := json.Marshal(Note{ID: 1, Title: "t", Conclusion: "c", Boundary: "b", Risk: "r",
		Evidence: "e", Confidence: "高", ValidUntil: "2026-12-31", Status: "已确认",
		Category: "材料", ProjectType: "房建", Craft: "泵送", RefCount: 2})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var nm map[string]any
	if err := json.Unmarshal(nb, &nm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"id", "title", "conclusion", "boundary", "risk", "evidence",
		"confidence", "validUntil", "status", "category", "projectType", "craft", "refCount", "createdAt", "updatedAt"} {
		if _, ok := nm[k]; !ok {
			t.Errorf("Note 缺 camelCase 键 %q: %v", k, nm)
		}
	}
	for _, bad := range []string{"ValidUntil", "RefCount", "ProjectType"} {
		if _, ok := nm[bad]; ok {
			t.Errorf("Note 不应出现 PascalCase 键 %q", bad)
		}
	}
	// 存量兼容：旧 PascalCase 行 Unmarshal 仍可读（encoding/json 大小写不敏）。
	var old Note
	if err := json.Unmarshal([]byte(`{"ID":7,"ValidUntil":"2026-01-01","RefCount":3}`), &old); err != nil || old.ID != 7 || old.RefCount != 3 {
		t.Errorf("旧 PascalCase 数据应兼容读取, err=%v old=%+v", err, old)
	}

	ib, _ := json.Marshal(Indicator{Key: "钢筋", Unit: "t", Samples: 5, Min: 1, Max: 9, Mean: 4, Median: 3.5, P25: 2, P75: 5})
	var im map[string]any
	_ = json.Unmarshal(ib, &im)
	for _, k := range []string{"key", "unit", "samples", "min", "max", "mean", "median", "p25", "p75"} {
		if _, ok := im[k]; !ok {
			t.Errorf("Indicator 缺 camelCase 键 %q: %v", k, im)
		}
	}
	for _, bad := range []string{"P25", "P75"} {
		if _, ok := im[bad]; ok {
			t.Errorf("Indicator 不应出现 PascalCase 键 %q", bad)
		}
	}
}
