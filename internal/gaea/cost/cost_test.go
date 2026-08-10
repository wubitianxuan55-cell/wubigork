package cost

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestCostStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	e := Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Spec: "300kW", Source: "市场询价", Tags: []string{"振动锤", "桩基"}, Status: "现行", Body: "含燃油与操作手。"}
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("hp300")
	if err != nil || got.Price != 3200 || got.Unit != "台班" || got.Category != "机械" {
		t.Fatalf("Get = %+v err=%v", got, err)
	}
	// UPSERT 更新
	e.Price = 3400
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get("hp300"); got.Price != 3400 {
		t.Errorf("upsert price = %v, want 3400", got.Price)
	}

	// 搜索：关键词命中规格；分类过滤
	_ = s.Save(Entry{Name: "cement", Title: "P.O 42.5 水泥", Category: "材料", Unit: "吨", Price: 480, Source: "定额"})
	res := s.Search("振动锤", "", "")
	if len(res) != 1 || res[0].Name != "hp300" {
		t.Errorf("keyword search = %+v", res)
	}
	res = s.Search("", "材料", "")
	if len(res) != 1 || res[0].Name != "cement" {
		t.Errorf("category filter = %+v", res)
	}
	// 关键词命中 name / tags（别名信号）。
	res = s.Search("hp300", "", "")
	if len(res) != 1 || res[0].Name != "hp300" {
		t.Errorf("name keyword search = %+v", res)
	}
	res = s.Search("桩基", "", "")
	if len(res) != 1 || res[0].Name != "hp300" {
		t.Errorf("tags keyword search = %+v", res)
	}
	res = s.Search("台班", "", "")
	if len(res) != 1 || res[0].Name != "hp300" {
		t.Errorf("unit keyword search = %+v", res)
	}
	// 多词查询：词间 AND，跨字段命中（标题含振动锤 + 单位=台班）。
	res = s.Search("液压振动锤 台班", "", "")
	if len(res) != 1 || res[0].Name != "hp300" {
		t.Errorf("multi-term search = %+v", res)
	}

	if err := s.Delete("hp300"); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 1 {
		t.Error("entry not deleted")
	}
}

// TestCostSearchBM25Order 验证本地 BM25 排序：同样命中子串时，
// 查询词出现密度更高（TF 更高）的条目排最前。
func TestCostSearchBM25Order(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)
	_ = s.Save(Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Status: "现行"})
	_ = s.Save(Entry{Name: "vib2", Title: "液压振动锤 液压振动锤配件", Category: "机械", Unit: "件", Price: 100, Status: "现行"})

	res := s.Search("液压振动锤", "", "")
	if len(res) != 2 {
		t.Fatalf("expected 2 matches, got %d: %+v", len(res), res)
	}
	if res[0].Name != "vib2" || res[1].Name != "hp300" {
		t.Errorf("bm25 order = %+v, want vib2 在前（标题命中两次）", res)
	}
}

func TestCostStoreUnavailable(t *testing.T) {
	s := Open(nil)
	if s.Available() {
		t.Error("nil db should be unavailable")
	}
	if err := s.Save(Entry{Name: "x"}); err == nil {
		t.Error("save on nil db should error")
	}
	if got := s.Search("", "", ""); got != nil {
		t.Errorf("search on nil db should be nil, got %+v", got)
	}
}
