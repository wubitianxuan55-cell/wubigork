package workcost

// gcode_test.go — 造价自有编码的规矩（2026-11-25）：补授、稳定、不复用、双码检索。
//
// 断言口径：字典专业码按名称序补数字号（「土壤修复」<「市政」按字节序），
// 段位 = 族前缀 + 专业2 + 作用域2 + 流水；同一章节/分部/分类的段号 get-or-create
// 恒不变；删行腾出的流水号永不复用（水位表）。

import (
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// TestGCodeBackfill 存量补授：三表空码行在 EnsureGCodes 后各得各族自有码；
// 字典专业码按名称序补齐；同类目行共享段号、流水递增；幂等重跑不产生重复码。
func TestGCodeBackfill(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复", "市政")

	// 直接 SQL 摆存量行（绕过写路径，模拟 V39 升级前的旧数据）。
	must := func(q string, args ...any) {
		t.Helper()
		if _, err := s.db.Exec(q, args...); err != nil {
			t.Fatalf("摆行失败: %v", err)
		}
	}
	must(`INSERT INTO gf_quotas(code, title, specialty, chapter, unit, source, created_at, updated_at)
VALUES('WP01','测量放线','市政','桩基工程','项','导入','t','t')`)
	must(`INSERT INTO gf_quotas(code, title, specialty, chapter, unit, source, created_at, updated_at)
VALUES('WP02','钻孔桩','市政','桩基工程','m','导入','t','t')`)
	must(`INSERT INTO gf_quotas(code, title, specialty, unit, source, created_at, updated_at)
VALUES('WP03','场地清表','土壤修复','m²','导入','t','t')`)
	must(`INSERT INTO gf_projects(name, file_name, created_at, updated_at)
VALUES('旺平','旺平.xlsx','t','t')`)
	must(`INSERT INTO gf_bill_items(project_id, code, title, unit, division, quantity, specialty, sort)
VALUES(1,'WP01','土方开挖','m³','A土方',100,'','1')`)
	must(`INSERT INTO gf_resources(code, kind, title, unit, base_price, category_path, region, tags, status, created_at, updated_at)
VALUES('L01','人工','普工','工日',180,'','重庆中心城区','[]','现行','t','t')`)
	must(`INSERT INTO gf_resources(code, kind, title, spec, unit, base_price, category_path, region, tags, status, created_at, updated_at)
VALUES('C20','材料','商品混凝土','C20','m³',430,'材料/土建材','重庆中心城区','[]','现行','t','t')`)
	must(`INSERT INTO gf_resources(code, kind, title, unit, base_price, category_path, region, tags, status, created_at, updated_at)
VALUES('MIX','调和沥青','神秘物质','t',1,'','成都市区','[]','现行','t','t')`)

	if err := s.EnsureGCodes(); err != nil {
		t.Fatalf("补授失败: %v", err)
	}

	gcode := func(table, code string) string {
		t.Helper()
		var g string
		if err := s.db.QueryRow(`SELECT gcode FROM `+table+` WHERE code=?`, code).Scan(&g); err != nil {
			t.Fatalf("读 %s.%s 的 gcode 失败: %v", table, code, err)
		}
		return g
	}

	// 字典专业码：名称字节序「土壤修复」<「市政」→ 01 / 02。
	dictCode := func(name string) string {
		t.Helper()
		var c string
		if err := s.db.QueryRow(`SELECT COALESCE(code,'') FROM gf_dicts WHERE kind='specialty' AND name=?`, name).Scan(&c); err != nil {
			t.Fatalf("读字典 %s 失败: %v", name, err)
		}
		return c
	}
	if got := dictCode("土壤修复"); got != "TR" {
		t.Fatalf("土壤修复是规划专业，应带预设代号 TR，得到 %q", got)
	}
	if got := dictCode("市政"); got != "01" {
		t.Fatalf("市政是本用例临时加的，补授时应发数字代号 01，得到 %q", got)
	}

	// 定额：D + 专业2 + 章节2 + 流水3；同章节两行共享段号、流水递增。
	if got := gcode("gf_quotas", "WP01"); got != "D0101001" {
		t.Fatalf("WP01 应授 D0101001（D+市政01+桩基工程01+001），得到 %q", got)
	}
	if got := gcode("gf_quotas", "WP02"); got != "D0101002" {
		t.Fatalf("WP02 应授 D0101002（同章节段号不变），得到 %q", got)
	}
	if got := gcode("gf_quotas", "WP03"); got != "DTR00001" {
		t.Fatalf("WP03 没归章节，应授 DTR00001（土壤修复TR+章节00+001），得到 %q", got)
	}
	// 清单：Q + 专业2（行专业空→项目默认；项目也没标→00 待整理段）+ 分部2 + 流水3。
	if got := gcode("gf_bill_items", "WP01"); got != "Q0001001" {
		t.Fatalf("清单 WP01 应授 Q0001001（Q+待整理00+A土方01+001），得到 %q", got)
	}
	// 工料机：R + 类别1 + 分类2 + 流水4；类别不在四类之内 → X 段。
	if got := gcode("gf_resources", "L01"); got != "RR000001" {
		t.Fatalf("普工没细分，应授 RR000001（人工R+分类00+0001），得到 %q", got)
	}
	if got := gcode("gf_resources", "C20"); got != "RC010001" {
		t.Fatalf("商品混凝土 应授 RC010001（材料C+材料/土建材01），得到 %q", got)
	}
	if got := gcode("gf_resources", "MIX"); got[:2] != "RX" {
		t.Fatalf("四类之外的「调和沥青」应授 RX…（X=类别待整理），得到 %q", got)
	}

	// 幂等：重跑不改码。
	again := gcode("gf_quotas", "WP01")
	if err := s.EnsureGCodes(); err != nil {
		t.Fatalf("幂等重跑失败: %v", err)
	}
	if gcode("gf_quotas", "WP01") != again {
		t.Fatal("幂等重跑不应改码")
	}

	// 双码检索：自有码当关键词能命中。
	qs, err := s.ListQuotas("", "D0101001")
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 1 || qs[0].Code != "WP01" {
		t.Fatalf("自有码检索应命中 WP01，得到 %v", qs)
	}
	rs, err := s.ListResources("", "RR000001")
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 || rs[0].Code != "L01" {
		t.Fatalf("自有码检索应命中 L01，得到 %v", rs)
	}
}

// TestGCodeNoReuseAfterDelete 删行不复用：水位只前进——删掉最大流水号的定额，
// 新定额的流水号接在已发最大号之后，不回落复用。
func TestGCodeNoReuseAfterDelete(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "市政")

	q1, err := s.SaveQuota(Quota{Code: "SZ-A", Title: "桩一", Specialty: "市政", Chapter: "桩基", Unit: "m"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	q2, err := s.SaveQuota(Quota{Code: "SZ-B", Title: "桩二", Specialty: "市政", Chapter: "桩基", Unit: "m"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if q1.GCode == "" || q2.GCode == "" {
		t.Fatalf("新定额应自带自有码（%q / %q）", q1.GCode, q2.GCode)
	}
	if q1.GCode[5:] != "001" || q2.GCode[5:] != "002" {
		t.Fatalf("流水应 001/002 递增，得到 %q / %q", q1.GCode, q2.GCode)
	}
	if q1.GCode[:5] != q2.GCode[:5] {
		t.Fatalf("同章节段号应一致，得到 %q / %q", q1.GCode, q2.GCode)
	}

	// 删掉 002，再建：应发 003，不回落 002。
	if err := s.DeleteQuota(q2.Code); err != nil {
		t.Fatal(err)
	}
	q3, err := s.SaveQuota(Quota{Code: "SZ-C", Title: "桩三", Specialty: "市政", Chapter: "桩基", Unit: "m"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if q3.GCode[5:] != "003" {
		t.Fatalf("删行后流水应接 003 不复用 002，得到 %q", q3.GCode)
	}
}

// TestGCodeStableSlotAndDictPriority 字典码优先与段号稳定：字典里给过缩写的专业
// 用缩写做专业段；同分部的新清单行拿到同一分部段号。
func TestGCodeStableSlotAndDictPriority(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "市政", "土壤修复")
	// 用户在字典里给「市政」手填缩写。
	if _, err := s.DictUpdate(DictSpecialty, "市政", "", "", "SZ", 0, ""); err != nil {
		t.Fatal(err)
	}

	pid, err := s.UpsertBillProject("旺平", "旺平.xlsx", "导入", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	b1, err := s.SaveBillItem(BillItem{ProjectID: pid, Title: "土方开挖", Unit: "m³", Division: "A土方", Quantity: 100, Specialty: "市政"})
	if err != nil {
		t.Fatal(err)
	}
	b2, err := s.SaveBillItem(BillItem{ProjectID: pid, Title: "土方回填", Unit: "m³", Division: "A土方", Quantity: 50, Specialty: "市政"})
	if err != nil {
		t.Fatal(err)
	}
	// 专业段=手填缩写 SZ；分部段同 scope 恒不变；流水递增。
	if b1.GCode != "QSZ01001" || b2.GCode != "QSZ01002" {
		t.Fatalf("清单应授 QSZ01001/QSZ01002，得到 %q / %q", b1.GCode, b2.GCode)
	}
}

// TestGCodeLegacyFixtureWithoutColumn 旧夹具静默跳过：库没有 gcode 列时
// EnsureGCodes 不报错也不碰表（旧测试树与迁移前读面的兼容保障）。
func TestGCodeLegacyFixtureWithoutColumn(t *testing.T) {
	dir := t.TempDir()
	raw := db.GetDatabase(dir)
	if raw == nil {
		t.Fatal("GetDatabase 返回 nil")
	}
	// 造一张没有 gcode 列的同名表（模拟旧夹具）。
	if _, err := raw.Exec(`DROP TABLE gf_quotas`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE gf_quotas (id INTEGER PRIMARY KEY, code TEXT)`); err != nil {
		t.Fatal(err)
	}
	s := Open(raw)
	if s == nil {
		t.Fatal("Open 返回 nil")
	}
	if err := s.EnsureGCodes(); err != nil {
		t.Fatalf("无 gcode 列的库应静默跳过，得到错误: %v", err)
	}
	db.CloseDatabase(dir)
	_ = filepath.Join(dir, "x") // 保温 dir 引用，兼容清理函数形状
}

// TestGCodeSlotDistinctPerScope 第三段在作用域内按名称编号：同一专业里不同分部段号
// 不同、同一分部段号不变；换专业从自己的 01 起算；没归分部 / 章节固定 00；专业
// 改名后新行仍得同一段（段号跟着字典 code 走）。
//
// 2026-11-25 真实库副本取证抓到的缺陷：每个名称各开一个计数器，所有分部都拿到 01
// （329 条清单全挤在 Q0001 一个前缀下）。旧用例每个作用域只出现一个分部，没钉住。
func TestGCodeSlotDistinctPerScope(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "市政", "土壤修复")
	pid, err := s.UpsertBillProject("甲", "a.xlsx", "", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	bill := func(title, division, specialty string) string {
		t.Helper()
		b, err := s.SaveBillItem(BillItem{ProjectID: pid, Title: title, Unit: "m", Division: division, Specialty: specialty})
		if err != nil {
			t.Fatalf("建清单行 %s 失败: %v", title, err)
		}
		return b.GCode
	}
	quota := func(code, chapter string) string {
		t.Helper()
		q, err := s.SaveQuota(Quota{Code: code, Title: code, Specialty: "市政", Chapter: chapter, Unit: "m"}, nil)
		if err != nil {
			t.Fatalf("建定额 %s 失败: %v", code, err)
		}
		return q.GCode
	}
	// 清单：首见的专业（市政）得 01；分部按首见顺序 01/02/03；没归分部 00。
	checkBill := func(got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("清单自有码 = %s, want %s", got, want)
		}
	}
	checkBill(bill("便道", "A临建", "市政"), "Q0101001")
	checkBill(bill("围挡", "A临建", "市政"), "Q0101002")    // 同分部：段号不变、流水递增
	checkBill(bill("降水井", "B支护降水", "市政"), "Q0102001") // 另一个分部：另一个段号
	checkBill(bill("场平", "C场地", "市政"), "Q0103001")
	checkBill(bill("没归分部", "", "市政"), "Q0100001")
	checkBill(bill("修复区降水", "B支护降水", "土壤修复"), "QTR01001") // 换专业：土壤修复是规划专业，段号是预设的 TR
	// 定额的章节同理。
	if got, want := quota("Q1", "道路工程"), "D0101001"; got != want {
		t.Errorf("定额自有码 = %s, want %s", got, want)
	}
	if got, want := quota("Q2", "桥梁工程"), "D0102001"; got != want {
		t.Errorf("定额自有码 = %s, want %s", got, want)
	}
	if got, want := quota("Q3", ""), "D0100001"; got != want {
		t.Errorf("定额自有码 = %s, want %s", got, want)
	}
	// 发出去的号写回了字典。
	dictCode := func(kind, name string) string {
		t.Helper()
		var c string
		if err := s.db.QueryRow(`SELECT COALESCE(code,'') FROM gf_dicts WHERE kind=? AND name=?`, kind, name).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	if dictCode(DictSpecialty, "市政") != "01" || dictCode(DictSpecialty, "土壤修复") != "TR" {
		t.Errorf("专业段号：临时加的市政发 01 并写回字典，土壤修复用预设的 TR，得到 %q / %q", dictCode(DictSpecialty, "市政"), dictCode(DictSpecialty, "土壤修复"))
	}
	// 专业改名：数据跟着改，段号不漂——新行仍是 01 段，流水接着排。
	if _, err := s.DictUpdate(DictSpecialty, "市政", "市政道路", "", "", 0, ""); err != nil {
		t.Fatal(err)
	}
	if got, want := bill("新便道", "A临建", "市政道路"), "Q0101003"; got != want {
		t.Errorf("专业改名后新清单应沿用 01 段与 A临建 段号，得到 %s, want %s", got, want)
	}
}

// TestGCodeCategorySlotFromDict 工料机的分类段落在字典 code 列：用户给过码的用用户的
// 码；没给的自动发号并写回字典（看得见、改得动）；自动发号跳过已被手填码占用的
// 前缀；没细分固定 00；分类改名后段号不漂；不在字典里的存量分类只记映射、不写字典。
func TestGCodeCategorySlotFromDict(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictCategory, "材料/钢材", "材料/水泥", "材料/木材", "材料/砂石")
	if _, err := s.DictUpdate(DictCategory, "材料/钢材", "", "", "GZ", 0, ""); err != nil {
		t.Fatal(err) // 用户手填缩写
	}
	if _, err := s.DictUpdate(DictCategory, "材料/木材", "", "", "03", 0, ""); err != nil {
		t.Fatal(err) // 用户手填 03：自动编号到 3 时必须跳过
	}
	save := func(title, category string) string {
		t.Helper()
		r, err := s.SaveResource(Resource{Kind: KindMaterial, Title: title, Unit: "t", CategoryPath: category})
		if err != nil {
			t.Fatalf("建工料机 %s 失败: %v", title, err)
		}
		return r.GCode
	}
	check := func(got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("工料机自有码 = %s, want %s", got, want)
		}
	}
	check(save("螺纹钢", "材料/钢材"), "RCGZ0001")  // 用户给的码
	check(save("散装水泥", "材料/水泥"), "RC010001") // 自动 01
	check(save("中砂", "材料/砂石"), "RC020001")   // 自动 02
	check(save("原木", "材料/木材"), "RC030001")   // 用户给的 03
	check(save("没细分", ""), "RC000001")       // 没细分 00
	check(save("袋装水泥", "材料/水泥"), "RC010002") // 同分类：段号不变、流水递增
	dictCode := func(name string) string {
		t.Helper()
		var c string
		if err := s.db.QueryRow(`SELECT COALESCE(code,'') FROM gf_dicts WHERE kind='category' AND name=?`, name).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	if dictCode("材料/水泥") != "01" || dictCode("材料/砂石") != "02" || dictCode("材料/钢材") != "GZ" || dictCode("材料/木材") != "03" {
		t.Fatalf("分类段号应写回字典且不覆盖用户给的码，得到 水泥 %q / 砂石 %q / 钢材 %q / 木材 %q",
			dictCode("材料/水泥"), dictCode("材料/砂石"), dictCode("材料/钢材"), dictCode("材料/木材"))
	}
	// 分类改名：数据跟着改，段号不漂。
	if _, err := s.DictUpdate(DictCategory, "材料/水泥", "材料/水泥及制品", "", "", 0, ""); err != nil {
		t.Fatal(err)
	}
	if got, want := save("砌筑水泥", "材料/水泥及制品"), "RC010003"; got != want {
		t.Errorf("分类改名后新工料机应沿用 01 段，得到 %s, want %s", got, want)
	}
	// 存量里不在字典的分类（写入面已拒收，直插模拟）：补授时只记映射，不往字典里加条目；
	// 自动号跳过已被占用的 03。
	for _, code := range []string{"OLD-1", "OLD-2"} {
		if _, err := s.db.Exec(`INSERT INTO gf_resources(code, kind, title, unit, category_path, tags) VALUES(?, ?, ?, ?, ?, '[]')`,
			code, "材料", code, "t", "材料/土建材"); err != nil {
			t.Fatalf("直插存量工料机失败: %v", err)
		}
	}
	if err := s.EnsureGCodes(); err != nil {
		t.Fatalf("补授失败: %v", err)
	}
	gcodeOf := func(code string) string {
		t.Helper()
		var g string
		if err := s.db.QueryRow(`SELECT gcode FROM gf_resources WHERE code=?`, code).Scan(&g); err != nil {
			t.Fatal(err)
		}
		return g
	}
	if gcodeOf("OLD-1") != "RC040001" || gcodeOf("OLD-2") != "RC040002" {
		t.Errorf("不在字典的存量分类应得没人用的 04 段，得到 %s / %s", gcodeOf("OLD-1"), gcodeOf("OLD-2"))
	}
	if n, _ := s.DictHas(DictCategory, "材料/土建材"); n {
		t.Error("补授不该往分类字典里加条目")
	}
}

// TestGCodeCategorySlotIsGlobal 分类段是全局编号：不同类别共用一个计数器，字典 code
// 一列唯一；同一个分类挂在别的类别下仍得同一段（只有类别字母不同）。
// 按类别各编一套会让字典里出现多个「01」（真实库副本取证抓到）。
func TestGCodeCategorySlotIsGlobal(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictCategory, "材料/水泥", "机械/土方机械", "人工/技工")
	save := func(kind, title, category string) string {
		t.Helper()
		r, err := s.SaveResource(Resource{Kind: kind, Title: title, Unit: "t", CategoryPath: category})
		if err != nil {
			t.Fatalf("build resource %s: %v", title, err)
		}
		return r.GCode
	}
	check := func(got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("gcode = %s, want %s", got, want)
		}
	}
	check(save(KindMaterial, "散装水泥", "材料/水泥"), "RC010001")
	check(save(KindMachine, "挖掘机", "机械/土方机械"), "RJ020001")
	check(save(KindLabor, "钢筋工", "人工/技工"), "RR030001")
	check(save(KindOutsourced, "外购水泥", "材料/水泥"), "RW010001")
	var dup int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM (SELECT code FROM gf_dicts WHERE kind='category' GROUP BY code HAVING COUNT(*)>1)`).Scan(&dup); err != nil {
		t.Fatal(err)
	}
	if dup != 0 {
		t.Errorf("分类字典 code 列应一列唯一，有 %d 个重码", dup)
	}
}
