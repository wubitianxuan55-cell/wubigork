package cost

import (
	"fmt"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestLegacyCategoryTarget(t *testing.T) {
	cases := []struct {
		source, title, path string
		want                string
	}{
		// 房建成本测算手册：按章节归入 房建工程 专业组
		{"房建成本测算手册（wu整理）.xlsx / 给排水", "房建室内PPR热水管", "给排水", "综合单价/房建工程/给排水工程"},
		{"房建成本测算手册（wu整理）.xlsx / 电气", "房建吸顶灯", "电气", "综合单价/房建工程/电气工程"},
		{"房建成本测算手册（wu整理）.xlsx / 通风", "房建镀锌钢板风管", "通风空调", "综合单价/房建工程/通风空调工程"},
		{"房建成本测算手册（wu整理）.xlsx / 采暖", "房建地暖PE-RT管", "采暖", "综合单价/房建工程/采暖工程"},
		{"房建成本测算手册（wu整理）.xlsx / 弱电", "房建六类网线敷设", "弱电", "综合单价/房建工程/弱电工程"},
		{"房建成本测算手册（wu整理）.xlsx / 土建", "房建挖土方综合单价", "土方", "综合单价/房建工程/土建工程"},
		{"房建成本测算手册（wu整理）.xlsx / 单方指标", "房建劳务单方-内墙抹灰", "装饰装修", "综合单价/房建工程/单方指标"},
		// 市政成本测算手册：按章节 + 关键词
		{"市政成本测算手册（wu整理）.xlsx / 道路", "挖一般土方（市政道路）", "土方", "综合单价/道路工程/土方工程"},
		{"市政成本测算手册（wu整理）.xlsx / 道路", "沥青混凝土面层", "土方", "综合单价/道路工程/机动车道"},
		{"市政成本测算手册（wu整理）.xlsx / 交通", "热熔标线", "土方", "综合单价/交通工程/标线"},
		{"市政成本测算手册（wu整理）.xlsx / 交通", "单柱式标志牌", "土方", "综合单价/交通工程/标识标牌"},
		{"市政成本测算手册（wu整理）.xlsx / 交通", "车行道八边形灯杆信号灯", "土方", "综合单价/交通工程/信号灯"},
		{"市政成本测算手册（wu整理）.xlsx / 交通", "水平定向钻穿公路", "土方", "综合单价/交通工程"},
		{"市政成本测算手册（wu整理）.xlsx / 绿化", "栽植乔木", "土方", "综合单价/绿化工程/乔木"},
		{"市政成本测算手册（wu整理）.xlsx / 绿化", "栽植灌木球", "土方", "综合单价/绿化工程/灌木"},
		{"市政成本测算手册（wu整理）.xlsx / 绿化", "铺种混播草皮卷", "土方", "综合单价/绿化工程/地被"},
		{"市政成本测算手册（wu整理）.xlsx / 电力", "电力管沟", "土方", "综合单价/电力工程/管沟与井室"},
		{"市政成本测算手册（wu整理）.xlsx / 给水", "球墨铸铁给水管铺设", "土方", "综合单价/给水工程/管道铺设"},
		{"市政成本测算手册（wu整理）.xlsx / 暖气", "直埋预制保温热力管", "土方", "综合单价/暖气工程/管道铺设"},
		{"市政成本测算手册（wu整理）.xlsx / 雨污", "砖砌检查井", "土方", "综合单价/雨污工程/检查井及雨水口"},
		{"市政成本测算手册（wu整理）.xlsx / 照明", "中杆灯（含基础）", "土方", "综合单价/照明工程/灯杆灯具安装"},
		// 人工类别平铺名
		{"市政成本测算手册（wu整理）.xlsx / 人工系数", "市政人工折算系数（成都）", "普工", "人工/普工"},
		{"人工费数据库.xlsx / 市场参考2026-04", "技工（大工/师傅）日工资", "技工", "人工/技工"},
		{"人工费数据库.xlsx / 市场参考2026-04", "特殊工种操作工日工资", "特殊工种", "人工/特殊工种"},
		// 人工费数据库纯人工条目
		{"人工费数据库.xlsx / 2023年总包报价清单", "加气块砌筑人工费", "砌体", "人工/技工"},
		{"人工费数据库.xlsx / 行业参考2026-04", "HDPE膜铺设人工费（含焊接）", "土方", "人工/技工"},
		// 特殊旧分类
		{"四川省工程造价信息网2026年06月（成都市区，不含税）", "浮法玻璃（δ10）", "玻璃及玻璃制品", "材料/土建材料/玻璃及玻璃制品"},
		{"四川省工程造价信息网2026年06月（成都市区，不含税）", "锯材（一等）", "木竹材料", "材料/土建材料/木材及竹木制品"},
		{"中科一兵分项报价组价（2026）", "基坑降水（管井/明排）", "土方", "综合单价/土方"},
		// 叶名唯一映射
		{"重庆工程造价2026年第7期（2026年6月中心城区）", "普通商品混凝土", "水泥及水泥制品", "材料/土建材料/水泥及水泥制品"},
		{"重庆工程造价2026年第7期（2026年6月中心城区）", "螺纹钢", "钢材", "材料/土建材料/钢材"},
		{"重庆工程造价2026年第7期（2026年6月中心城区）", "页岩配砖", "砖瓦灰砂石", "材料/土建材料/砖瓦灰砂石"},
		{"机械费数据库.xlsx / 行业价2026-04", "挖掘机", "土方机械", "机械/土方机械"},
		{"材料单价数据库.xlsx / 阿里巴巴2026-04", "玻纤土工格栅", "土工合成材料", "材料/辅助材料/土工合成材料"},
		{"市场参考", "桩机", "桩基机械", "机械/桩基机械"},
		// 无法确定 → 空
		{"导入文件", "自定义条目", "自定义分类", ""},
	}
	for i, c := range cases {
		got := legacyCategoryTarget(c.source, c.title, c.path)
		if got != c.want {
			t.Errorf("case %d: legacyCategoryTarget(%q, %q, %q) = %q, want %q", i, c.source, c.title, c.path, got, c.want)
		}
	}
}

func TestPriceMetaFromSource(t *testing.T) {
	cases := []struct {
		source        string
		wantRegion    string
		wantPriceDate string
	}{
		{"重庆工程造价2026年第7期（2026年6月中心城区）", "重庆中心城区", "2026年6月"},
		{"四川省工程造价信息网2026年06月（成都市区，不含税）", "成都市区", "2026年6月"},
		{"乐山市建筑材料市场信息价2026年7月（乐山市，不含税）", "乐山市", "2026年7月"},
		{"乐山市建筑材料市场信息价2026年7月（市中区，不含税）", "乐山市中区", "2026年7月"},
		{"乐山市建筑材料市场信息价2026年7月（市中区，不含税；按官方价差折算）", "乐山市中区", "2026年7月"},
		{"乐山市建筑材料市场信息价2026年7月（乐山市，不含税出厂）", "乐山市", "2026年7月"},
		{"材料单价数据库.xlsx / 2026年3月重庆信息价", "重庆", "2026年3月"},
		{"材料单价数据库.xlsx / 2026年3月四川信息价", "四川", "2026年3月"},
		{"材料单价数据库.xlsx / 百年建筑2026-04-17周报", "", "2026年4月"},
		{"人工费数据库.xlsx / 市场参考2026-04", "", "2026年4月"},
		// 无日期/地区的来源 → 不回填（不臆造）
		{"房建成本测算手册（wu整理）.xlsx / 给排水", "", ""},
		{"达州渠县福达利项目成本测算（2026.8.9）", "", ""},
		{"2026.8.12市场复核：市场均价约750–900，原800微调780", "", ""},
		{"导入文件", "", ""},
	}
	for i, c := range cases {
		r, d := priceMetaFromSource(c.source)
		if r != c.wantRegion || d != c.wantPriceDate {
			t.Errorf("case %d: priceMetaFromSource(%q) = (%q, %q), want (%q, %q)", i, c.source, r, d, c.wantRegion, c.wantPriceDate)
		}
	}
}

// TestRepairCategoryPaths 修复平铺分类路径：目标节点缺失时自动创建。
func TestRepairCategoryPaths(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	// 播种默认分类树后插入旧平铺分类条目（模拟 SchemaV7 遗留数据）。
	seed := func(name, title, source, path string) {
		if _, err := gdb.Exec(`INSERT INTO cost_entries(name, title, category, category_path, unit, price, source, status, created_at, updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?)`,
			name, title, path, path, "m³", 100, source, "现行", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	seed("fanggai", "房建室内PPR热水管", "房建成本测算手册（wu整理）.xlsx / 给排水", "给排水")
	seed("shizheng", "挖一般土方（市政道路）", "市政成本测算手册（wu整理）.xlsx / 道路", "土方")
	seed("boli", "浮法玻璃（δ10）", "四川省工程造价信息网2026年06月（成都市区，不含税）", "玻璃及玻璃制品")
	seed("shuini", "普通商品混凝土", "重庆工程造价2026年第7期（2026年6月中心城区）", "水泥及水泥制品")
	seed("weizhi", "自定义条目", "导入文件", "自定义分类")

	fixed, left, err := s.RepairCategoryPaths()
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 4 || left != 1 {
		t.Fatalf("fixed=%d left=%d, want 4/1", fixed, left)
	}
	want := map[string]string{
		"fanggai":  "综合单价/房建工程/给排水工程",
		"shizheng": "综合单价/道路工程/土方工程",
		"boli":     "材料/土建材料/玻璃及玻璃制品",
		"shuini":   "材料/土建材料/水泥及水泥制品",
	}
	for name, path := range want {
		got, err := s.Get(name)
		if err != nil {
			t.Fatalf("Get %s: %v", name, err)
		}
		if got.CategoryPath != path {
			t.Errorf("%s path = %q, want %q", name, got.CategoryPath, path)
		}
		if got.Category != leafOfPath(path) {
			t.Errorf("%s category = %q, want %q", name, got.Category, leafOfPath(path))
		}
	}
	// 未知目标条目保持不动。
	if got, _ := s.Get("weizhi"); got.CategoryPath != "自定义分类" {
		t.Errorf("unresolved entry moved to %q", got.CategoryPath)
	}
	// 幂等：再次执行不再修复。
	fixed2, _, err := s.RepairCategoryPaths()
	if err != nil || fixed2 != 0 {
		t.Fatalf("second repair fixed=%d err=%v, want 0/nil", fixed2, err)
	}
	// 新增节点在树中可解析。
	for _, p := range []string{"综合单价/房建工程/给排水工程", "材料/土建材料/玻璃及玻璃制品"} {
		if !s.pathResolves(p) {
			t.Errorf("path %q should resolve after repair", p)
		}
	}
}

// TestBackfillPriceMeta 从来源回填地区/期数（幂等，不动已有值）。
func TestBackfillPriceMeta(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	seed := func(name, source, region, priceDate string) {
		if _, err := gdb.Exec(`INSERT INTO cost_entries(name, title, category, category_path, unit, price, source, region, price_date, status, created_at, updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			name, name, "材料", "材料/土建材料/钢材", "吨", 100, source, region, priceDate, "现行", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	seed("cq", "重庆工程造价2026年第7期（2026年6月中心城区）", "", "")
	seed("leshan", "乐山市建筑材料市场信息价2026年7月（市中区，不含税）", "", "")
	seed("keep", "材料单价数据库.xlsx / 2026年3月重庆信息价", "已填地区", "已填期数")
	seed("manual", "房建成本测算手册（wu整理）.xlsx / 给排水", "", "")

	rc, dc, err := s.BackfillPriceMeta()
	if err != nil {
		t.Fatal(err)
	}
	if rc != 2 || dc != 2 {
		t.Fatalf("region=%d date=%d, want 2/2", rc, dc)
	}
	if got, _ := s.Get("cq"); got.Region != "重庆中心城区" || got.PriceDate != "2026年6月" {
		t.Errorf("cq = %q/%q", got.Region, got.PriceDate)
	}
	if got, _ := s.Get("leshan"); got.Region != "乐山市中区" || got.PriceDate != "2026年7月" {
		t.Errorf("leshan = %q/%q", got.Region, got.PriceDate)
	}
	// 已有值不被覆盖；无来源信息不臆造。
	if got, _ := s.Get("keep"); got.Region != "已填地区" || got.PriceDate != "已填期数" {
		t.Errorf("keep overwritten: %q/%q", got.Region, got.PriceDate)
	}
	if got, _ := s.Get("manual"); got.Region != "" || got.PriceDate != "" {
		t.Errorf("manual fabricated: %q/%q", got.Region, got.PriceDate)
	}
	// 幂等。
	rc2, dc2, _ := s.BackfillPriceMeta()
	if rc2 != 0 || dc2 != 0 {
		t.Fatalf("second backfill region=%d date=%d, want 0/0", rc2, dc2)
	}
}

// TestOpenSelfHeal Open 时自动修复平铺分类路径。
func TestOpenSelfHeal(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	// 先插入旧数据（此时不经过 Open 的自愈），再重开。
	if _, err := gdb.Exec(`INSERT INTO cost_entries(name, title, category, category_path, unit, price, source, status, created_at, updated_at)
		VALUES('gzj', '加气块砌筑人工费', '砌体', '砌体', 'm³', 320, '人工费数据库.xlsx / 2023年总包报价清单', '现行', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	s := Open(gdb)
	got, err := s.Get("gzj")
	if err != nil {
		t.Fatal(err)
	}
	if got.CategoryPath != "人工/技工" {
		t.Fatalf("Open 自愈后 path = %q, want 人工/技工", got.CategoryPath)
	}
	db.CloseDatabase(dir)
}

// TestEnsurePathTx 幂等建节点。
func TestEnsurePathTx(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	tx, _ := gdb.Begin()
	if err := ensurePathTx(tx, "综合单价/房建工程/电气工程", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := ensurePathTx(tx, "综合单价/房建工程/电气工程", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := gdb.QueryRow("SELECT COUNT(*) FROM cost_categories").Scan(&n); err != nil {
		t.Fatal(err)
	}
	// 未播种默认树（未走 Open），ensurePathTx 只创建 综合单价/房建工程/电气工程 三节点。
	if n != 3 {
		t.Fatalf("category count = %d, want 3", n)
	}
	_ = fmt.Sprint()
}
