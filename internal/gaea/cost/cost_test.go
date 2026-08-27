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

	// 蒸馏字段（SchemaV9）：价格三要素（规格+地区+期数）+ 口径 + 有效期 + 导入行号。
	e := Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Spec: "300kW", Source: "市场询价", Region: "成都市区", PriceDate: "2026-08", PriceType: "到场价", ValidUntil: "2026-12-31", SourceRow: 12, Tags: []string{"振动锤", "桩基"}, Status: "现行", Body: "含燃油与操作手。"}
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("hp300")
	if err != nil || got.Price != 3200 || got.Unit != "台班" || got.Category != "机械" ||
		got.Region != "成都市区" || got.PriceDate != "2026-08" || got.PriceType != "到场价" ||
		got.ValidUntil != "2026-12-31" || got.SourceRow != 12 {
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

	// 综合单价架构：人材机二级组成 + 费率（SchemaV12/13 全链路读写）。
	e.LaborFee = 0
	e.MaterialFee = 15.76
	e.MachineFee = 5
	e.ManagementFee = 0.62
	e.ProfitFee = 2.08
	e.AdvanceFee = 0.62
	e.TaxRate = 9
	e.Components = []Component{
		{Kind: "材料", Title: "石灰稳定土", Unit: "m²", Quantity: 0.5, Price: 30, Amount: 15, Note: "30*0.5*1.03"},
		{Kind: "机械", Title: "机械摊铺+人工配合", Unit: "m²", Quantity: 1, Price: 5, Amount: 5},
	}
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}
	got2, err := s.Get("hp300")
	if err != nil {
		t.Fatal(err)
	}
	if got2.LaborFee != 0 || got2.MaterialFee != 15.76 || got2.MachineFee != 5 ||
		got2.ManagementFee != 0.62 || got2.ProfitFee != 2.08 || got2.AdvanceFee != 0.62 || got2.TaxRate != 9 {
		t.Fatalf("fees/rates not persisted: %+v", got2)
	}
	if len(got2.Components) != 2 || got2.Components[0].Title != "石灰稳定土" ||
		got2.Components[1].Kind != "机械" || got2.Components[1].Amount != 5 {
		t.Fatalf("components not persisted: %+v", got2.Components)
	}
	// 覆盖保存后组成整组替换。
	e.Components = e.Components[:1]
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}
	if got3, _ := s.Get("hp300"); len(got3.Components) != 1 {
		t.Fatalf("components replace = %d, want 1", len(got3.Components))
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
	// 蒸馏检索：按地区/期数/口径关键词召回（价格三要素可检索）。
	res = s.Search("成都", "", "")
	if len(res) != 1 || res[0].Name != "hp300" || res[0].Region != "成都市区" {
		t.Errorf("region keyword search = %+v", res)
	}
	res = s.Search("2026-08", "", "")
	if len(res) != 1 || res[0].PriceDate != "2026-08" {
		t.Errorf("price period keyword search = %+v", res)
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

// TestCostCategoriesTree 验证多级分类：默认树播种（综合单价→专业→分部）、
// 路径过滤、子树计数、改名重写、删除保护。
func TestCostCategoriesTree(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	// 默认树已播种：资源库层（人工/材料/机械/运输/检测/综合单价/其他 7 根），
	// 综合单价下按专业细分到分部。
	tree := s.Categories()
	if len(tree) != 7 {
		t.Fatalf("root categories = %d, want 7（人工/材料/机械/运输/检测/综合单价/其他）", len(tree))
	}
	byName := map[string]*CategoryView{}
	for i := range tree {
		byName[tree[i].Name] = &tree[i]
	}
	if zj := byName["综合单价"]; zj == nil || len(zj.Children) == 0 {
		t.Fatalf("综合单价 should have children, got %+v", zj)
	}

	// 默认树已按市政手册细分：综合单价 → 道路工程 → 土方工程（三级）。
	var zjID, roadID, earthID int
	for i := range tree {
		if tree[i].Name != "综合单价" {
			continue
		}
		zjID = tree[i].ID
		for _, c := range tree[i].Children {
			if c.Name == "道路工程" {
				roadID = c.ID
				for _, g := range c.Children {
					if g.Name == "土方工程" {
						earthID = g.ID
					}
				}
			}
		}
	}
	if zjID == 0 || roadID == 0 || earthID == 0 {
		t.Fatalf("seeded tree missing 综合单价/道路工程/土方工程: zj=%d road=%d earth=%d", zjID, roadID, earthID)
	}

	// 保存带完整路径（三级）的条目。
	if err := s.Save(Entry{Name: "dig-shallow", Title: "挖一般土方（深2m）", Category: "土方工程", CategoryPath: "综合单价/道路工程/土方工程", Unit: "m³", Price: 3.79, Source: "市政成本测算手册", LaborFee: 0, MaterialFee: 0, MachineFee: 3}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "backfill", Title: "土方回填", Category: "土方工程", CategoryPath: "综合单价/道路工程/土方工程", Unit: "m³", Price: 8.85, Source: "市政成本测算手册", MachineFee: 7}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "road-pipe", Title: "排水管铺设", Category: "管道铺设", CategoryPath: "综合单价/雨污工程/管道铺设", Unit: "m", Price: 638.43, Source: "市政成本测算手册"}); err != nil {
		t.Fatal(err)
	}

	// 子树过滤：选「道路工程」应命中土方条目（含后代），不命中雨污。
	res := s.Search("", "综合单价/道路工程", "")
	if len(res) != 2 {
		t.Fatalf("path subtree search = %d, want 2: %+v", len(res), res)
	}
	// 叶子路径精确过滤（三级）。
	res = s.Search("", "综合单价/道路工程/土方工程", "")
	if len(res) != 2 {
		t.Fatalf("leaf path search = %d, want 2", len(res))
	}
	res = s.Search("", "综合单价/道路工程", "")
	if len(res) != 2 {
		t.Fatalf("二级 path search = %d, want 2", len(res))
	}
	res = s.Search("", "综合单价/雨污工程", "")
	if len(res) != 1 || res[0].Name != "road-pipe" {
		t.Fatalf("雨污 path search = %+v", res)
	}

	// 树节点计数（直接归属）。
	tree = s.Categories()
	zjID = 0
	for i := range tree {
		if tree[i].Name == "综合单价" {
			zjID = tree[i].ID
			if tree[i].Count != 0 {
				t.Errorf("综合单价 direct count = %d, want 0（条目在子分类）", tree[i].Count)
			}
		}
		for _, ch := range tree[i].Children {
			if ch.Name == "道路工程" {
				for _, g := range ch.Children {
					if g.Name == "土方工程" && g.Count != 2 {
						t.Errorf("综合单价/道路工程/土方工程 count = %d, want 2", g.Count)
					}
					if g.Name == "土方工程" {
						earthID = g.ID
					}
				}
			}
		}
	}

	// 改名重写条目路径：综合单价/道路工程/土方工程 → 场地平整。
	if _, err := s.SaveCategory(roadID, "场地平整", 0, earthID); err != nil {
		t.Fatal(err)
	}
	res = s.Search("", "综合单价/道路工程/场地平整", "")
	if len(res) != 2 {
		t.Fatalf("after rename path search = %d, want 2", len(res))
	}
	if got, _ := s.Get("dig-shallow"); got.CategoryPath != "综合单价/道路工程/场地平整" {
		t.Errorf("dig-shallow category_path = %q, want 综合单价/道路工程/场地平整", got.CategoryPath)
	}

	// 删除保护：有条目/子节点时拒绝。
	if err := s.DeleteCategory(earthID); err == nil {
		t.Error("delete category with entries should fail")
	}
	if err := s.DeleteCategory(zjID); err == nil {
		t.Error("delete category with children should fail")
	}
	// 空叶子可删：雨污工程/管道铺设 下条目移走后再删。
	pipeLeaf := 0
	for _, ch := range byName["综合单价"].Children {
		if ch.Name == "雨污工程" {
			for _, g := range ch.Children {
				if g.Name == "管道铺设" {
					pipeLeaf = g.ID
				}
			}
		}
	}
	if err := s.Delete("road-pipe"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCategory(pipeLeaf); err != nil {
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
