package cost

// 刀D（v4.247）BM25 排序缓存：同数据同过滤形态复用倒排（写路径推进版本
// 失效），保存/删除/分类变更后新数据必须立即可查。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestSearchRankerCacheInvalidation(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	mustSave := func(e Entry) {
		t.Helper()
		if err := s.Save(e); err != nil {
			t.Fatal(err)
		}
	}
	mustSave(Entry{Name: "vib-a", Title: "液压振动锤 A 型", Category: "机械", Unit: "台班", Price: 100})
	mustSave(Entry{Name: "vib-b", Title: "液压振动锤 B 型", Category: "机械", Unit: "台班", Price: 200})

	// 首查建立缓存；二查命中缓存（同数据同形态，结果必须一致）。
	first := s.Search("液压振动锤", "", "")
	second := s.Search("液压振动锤", "", "")
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("首查/二查 = %d/%d, want 2/2", len(first), len(second))
	}

	// 写路径推进版本：新条目必须立即可查（不得命中旧语料缓存）。
	mustSave(Entry{Name: "vib-c", Title: "液压振动锤 C 型 加强版", Category: "机械", Unit: "台班", Price: 300})
	third := s.Search("液压振动锤", "", "")
	if len(third) != 3 {
		t.Fatalf("写入后查询 = %d, want 3（缓存失效语义破坏）", len(third))
	}
	found := false
	for _, e := range third {
		if e.Name == "vib-c" {
			found = true
		}
	}
	if !found {
		t.Fatal("新条目未出现在查询结果（缓存失效语义破坏）")
	}

	// 删除同样失效。
	if err := s.Delete("vib-c"); err != nil {
		t.Fatal(err)
	}
	if got := s.Search("液压振动锤", "", ""); len(got) != 2 {
		t.Fatalf("删除后查询 = %d, want 2", len(got))
	}

	// 分类过滤形态各自缓存，互不污染。
	mustSave(Entry{Name: "mat-a", Title: "液压振动锤 配件 材料", Category: "材料", Unit: "件", Price: 10})
	mech := s.Search("液压振动锤", "机械", "")
	if len(mech) != 2 {
		t.Fatalf("机械过滤 = %d, want 2", len(mech))
	}
	mat := s.Search("液压振动锤", "材料", "")
	if len(mat) != 1 || mat[0].Name != "mat-a" {
		t.Fatalf("材料过滤 = %+v, want [mat-a]", mat)
	}
}
