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

	if err := s.Delete("hp300"); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 1 {
		t.Error("entry not deleted")
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
