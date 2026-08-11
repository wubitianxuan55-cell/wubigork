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

// TestCostCategoriesTree 验证多级分类：默认树播种、路径过滤、子树计数。
func TestCostCategoriesTree(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	// 默认树已播种：一级 7 类 + 二级子类。
	tree := s.Categories()
	if len(tree) != 7 {
		t.Fatalf("root categories = %d, want 7", len(tree))
	}
	byName := map[string]*CategoryView{}
	for i := range tree {
		byName[tree[i].Name] = &tree[i]
	}
	if mat := byName["材料"]; mat == nil || len(mat.Children) == 0 {
		t.Fatalf("材料 should have children, got %+v", mat)
	}

	// 默认树已按信息价体系细分：材料 → 土建材料 → 钢材（三级）。
	var matID, tuID, steelID int
	for i := range tree {
		if tree[i].Name != "材料" {
			continue
		}
		matID = tree[i].ID
		for _, c := range tree[i].Children {
			if c.Name == "土建材料" {
				tuID = c.ID
				for _, g := range c.Children {
					if g.Name == "钢材" {
						steelID = g.ID
					}
				}
			}
		}
	}
	if matID == 0 || tuID == 0 || steelID == 0 {
		t.Fatalf("seeded tree missing 材料/土建材料/钢材: mat=%d tu=%d steel=%d", matID, tuID, steelID)
	}

	// 保存带完整路径（三级）的条目。
	if err := s.Save(Entry{Name: "steel-h", Title: "H 型钢", Category: "钢材", CategoryPath: "材料/土建材料/钢材", Unit: "吨", Price: 5200, Source: "市场询价"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "steel-r", Title: "螺纹钢", Category: "钢材", CategoryPath: "材料/土建材料/钢材", Unit: "吨", Price: 4600, Source: "定额"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "labor-p", Title: "普工", Category: "普工", CategoryPath: "人工/普工", Unit: "工日", Price: 320, Source: "定额"}); err != nil {
		t.Fatal(err)
	}

	// 子树过滤：选「材料」应命中钢材（含后代），不命中人工。
	res := s.Search("", "材料", "")
	if len(res) != 2 {
		t.Fatalf("path subtree search = %d, want 2: %+v", len(res), res)
	}
	// 三级路径精确过滤。
	res = s.Search("", "材料/土建材料/钢材", "")
	if len(res) != 2 {
		t.Fatalf("leaf path search = %d, want 2", len(res))
	}
	res = s.Search("", "材料/土建材料", "")
	if len(res) != 2 {
		t.Fatalf("二级 path search = %d, want 2", len(res))
	}
	res = s.Search("", "人工", "")
	if len(res) != 1 || res[0].Name != "labor-p" {
		t.Fatalf("labor path search = %+v", res)
	}

	// 树节点计数（直接归属）。
	tree = s.Categories()
	matID = 0
	for i := range tree {
		if tree[i].Name == "材料" {
			matID = tree[i].ID
			if tree[i].Count != 0 {
				t.Errorf("材料 direct count = %d, want 0（条目在子分类）", tree[i].Count)
			}
		}
		if tree[i].Name == "人工" && tree[i].Count != 0 {
			t.Errorf("人工 direct count = %d, want 0（条目在子分类）", tree[i].Count)
		}
		for _, ch := range tree[i].Children {
			if ch.Name == "土建材料" {
				for _, g := range ch.Children {
					if g.Name == "钢材" && g.Count != 2 {
						t.Errorf("材料/土建材料/钢材 count = %d, want 2", g.Count)
					}
					if g.Name == "钢材" {
						steelID = g.ID
					}
				}
			}
		}
	}

	// 改名重写条目路径：材料/土建材料/钢材 → 型钢。
	if _, err := s.SaveCategory(tuID, "型钢", 0, steelID); err != nil {
		t.Fatal(err)
	}
	res = s.Search("", "材料/土建材料/型钢", "")
	if len(res) != 2 {
		t.Fatalf("after rename path search = %d, want 2", len(res))
	}
	if got, _ := s.Get("steel-h"); got.CategoryPath != "材料/土建材料/型钢" {
		t.Errorf("steel-h category_path = %q, want 材料/土建材料/型钢", got.CategoryPath)
	}

	// 删除保护：有条目/子节点时拒绝。
	if err := s.DeleteCategory(steelID); err == nil {
		t.Error("delete category with entries should fail")
	}
	if err := s.DeleteCategory(matID); err == nil {
		t.Error("delete category with children should fail")
	}
	// 空叶子可删：人工/普工 下条目移走后再删。
	laborRoot := byName["人工"]
	var laborChildID int
	for _, ch := range laborRoot.Children {
		if ch.Name == "普工" {
			laborChildID = ch.ID
		}
	}
	if err := s.Delete("labor-p"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCategory(laborChildID); err != nil {
		t.Fatalf("delete empty leaf failed: %v", err)
	}
}

// TestCostSaveCategoryIdempotent 验证同父同名新建幂等返回既有 id。
func TestCostSaveCategoryIdempotent(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	id1, err := s.SaveCategory(0, "临时分类", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.SaveCategory(0, "临时分类", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 || id1 <= 0 {
		t.Fatalf("idempotent create ids = %d/%d", id1, id2)
	}
	if got := s.CategoryPath(id1); got != "临时分类" {
		t.Errorf("path = %q", got)
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
