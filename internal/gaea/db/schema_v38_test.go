package db

import (
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// buildAtVersion 在临时目录建一个停在 v 版的库（按迁移链逐版跑到 v），返回其
// userDir。供「从旧版升级」的用例摆出旧库形态。
func buildAtVersion(t *testing.T, v int) string {
	t.Helper()
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", DatabasePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	if _, err := raw.Exec(SchemaV1); err != nil {
		t.Fatalf("建 V1 失败: %v", err)
	}
	if _, err := raw.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '1')"); err != nil {
		t.Fatalf("写 user_version 失败: %v", err)
	}
	for i := 1; i < v; i++ { // migrations[i] = V(i+1)
		if migrations[i] == "" {
			continue
		}
		if _, err := raw.Exec(migrations[i]); err != nil {
			t.Fatalf("迁移 V%d 失败: %v", i+1, err)
		}
		if _, err := raw.Exec("UPDATE schema_meta SET value=? WHERE key='user_version'", strconv.Itoa(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestSchemaV38UpgradeFromV37 库定规矩的迁移（2026-11-25）：从 V37 的库升上来——
//   - 只加列，行数一行不多不少；
//   - 工料机身份唯一索引换成含地区，同身份不同地区可并存、同地区仍互斥；
//   - 字典从库里已有的取值机械种子（专业/地区/分类），不预设任何清单；
//   - 期数「2026年6月」统一成「2026-06」；认不出的写法原样保留（进待整理）。
func TestSchemaV38UpgradeFromV37(t *testing.T) {
	dir := buildAtVersion(t, 37)
	{
		raw, err := sql.Open("sqlite", DatabasePath(dir))
		if err != nil {
			t.Fatal(err)
		}
		quota := `INSERT INTO gf_quotas(code, title, specialty, unit, source, price_date, note, status, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`
		for _, q := range [][]string{
			{"WP01", "测量放线", "土壤修复", "项", "项目导入：A.xlsx", "2026年6月", "", "现行", "t", "t"},
			{"WP02", "施工便道", "土壤修复", "m", "项目导入：A.xlsx", "2026年第7期", "", "现行", "t", "t"},
			{"WP03", "水泥搅拌桩", "市政", "m", "手工录入", "", "", "现行", "t", "t"},
		} {
			if _, err := raw.Exec(quota, q[0], q[1], q[2], q[3], q[4], q[5], q[6], q[7], q[8], q[9]); err != nil {
				t.Fatalf("摆定额 %s 失败: %v", q[0], err)
			}
		}
		res := `INSERT INTO gf_resources(code, kind, title, spec, unit, base_price, current_price, category_path, region, price_date, tags, status)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`
		for _, r := range [][]string{
			{"L01", "人工", "普工", "", "工日", "180", "180", "人工/普工", "", "2026年6月"},
			{"C20", "材料", "商品混凝土", "C20", "m³", "430", "430", "材料/土建材料/钢材", "重庆中心城区", "2026年11月"},
			{"CEM", "材料", "水泥", "P.O42.5", "t", "380", "380", "", "成都市区", "2026.4"},
		} {
			if _, err := raw.Exec(res, r[0], r[1], r[2], r[3], r[4], r[5], r[6], r[7], r[8], r[9], "[]", "现行"); err != nil {
				t.Fatalf("摆工料机 %s 失败: %v", r[0], err)
			}
		}
		_ = raw.Close()
	}

	db := GetDatabase(dir)
	if db == nil {
		t.Fatal("升级后库打不开（迁移失败，见日志）")
	}
	defer CloseDatabase(dir)

	var ver int
	if err := db.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Fatalf("user_version = %d, want %d", ver, len(migrations))
	}

	// 行数不变。
	for _, c := range []struct {
		table string
		want  int
	}{{"gf_quotas", 3}, {"gf_resources", 3}} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + c.table).Scan(&n); err != nil || n != c.want {
			t.Fatalf("%s 行数 = %d（err=%v），want %d", c.table, n, err, c.want)
		}
	}
	// 新列到位。
	for _, c := range []struct{ table, col string }{
		{"gf_projects", "specialty"}, {"gf_projects", "region"}, {"gf_projects", "price_period"},
		{"gf_bill_items", "specialty"}, {"gf_quotas", "derived_from"},
	} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?", c.table, c.col).Scan(&n); err != nil || n != 1 {
			t.Errorf("%s.%s 应已加上（n=%d err=%v）", c.table, c.col, n, err)
		}
	}
	// 身份索引含地区：同身份不同地区并存，同地区互斥。
	var idxSQL string
	if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='index' AND name='idx_gf_resources_ident'").Scan(&idxSQL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(idxSQL, "region") {
		t.Fatalf("身份唯一索引应含地区，得到 %s", idxSQL)
	}
	if _, err := db.Exec(`INSERT INTO gf_resources(code, kind, title, spec, unit, region, tags) VALUES('C20-CD','材料','商品混凝土','C20','m³','成都市区','[]')`); err != nil {
		t.Fatalf("同身份不同地区应可并存: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO gf_resources(code, kind, title, spec, unit, region, tags) VALUES('C20-DUP','材料','商品混凝土','C20','m³','重庆中心城区','[]')`); err == nil {
		t.Fatal("同身份同地区应被唯一索引拦下")
	}
	// 字典：机械种子，取自库里已有的取值。
	dict := func(kind string) []string {
		rows, err := db.Query(`SELECT name FROM gf_dicts WHERE kind=? ORDER BY name`, kind)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rows.Close() }()
		var out []string
		for rows.Next() {
			var n string
			if rows.Scan(&n) == nil {
				out = append(out, n)
			}
		}
		return out
	}
	// 专业：数据里已有的取值（土壤修复 / 市政）都在；另加 SchemaV40 预设的规划专业
	// （土壤修复既在数据里也在规划表里，只占一条）。
	if got := dict("specialty"); len(got) != len(PlannedSpecialties)+1 {
		t.Errorf("专业字典应 = 库里已有取值 + 规划专业共 %d 条，得到 %v", len(PlannedSpecialties)+1, got)
	}
	for _, want := range []string{"土壤修复", "市政", "临建", "检测"} {
		found := false
		for _, n := range dict("specialty") {
			found = found || n == want
		}
		if !found {
			t.Errorf("专业字典缺 %q", want)
		}
	}
	if got := dict("region"); strings.Join(got, "|") != "成都市区|重庆中心城区" {
		t.Errorf("地区字典应取自已有取值，得到 %v", got)
	}
	if got := dict("category"); strings.Join(got, "|") != "人工/普工|材料/土建材料/钢材" {
		t.Errorf("分类字典应取自已有取值（整条路径），得到 %v", got)
	}
	// 期数：年月写法统一成 YYYY-MM；认不出的原样保留。
	periodOf := func(table, code string) string {
		var p string
		if err := db.QueryRow(`SELECT price_date FROM `+table+` WHERE code=?`, code).Scan(&p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if got := periodOf("gf_quotas", "WP01"); got != "2026-06" {
		t.Errorf("WP01 期数应转成 2026-06，得到 %q", got)
	}
	if got := periodOf("gf_quotas", "WP02"); got != "2026年第7期" {
		t.Errorf("认不出的期数应原样保留，得到 %q", got)
	}
	if got := periodOf("gf_resources", "C20"); got != "2026-11" {
		t.Errorf("C20 期数应转成 2026-11，得到 %q", got)
	}
	if got := periodOf("gf_resources", "CEM"); got != "2026.4" {
		t.Errorf("不在转换范围内的写法应原样保留，得到 %q", got)
	}
}

// TestSchemaV38FreshChain 新库跑全链：地区与工料机分类不预设（空表），规矩留给用户
// 按需加。专业自 SchemaV40（2026-11-25 用户令重新规划）起预设规划专业表——那一条
// 「不预设任何清单」的口径只对专业翻了案。
func TestSchemaV38FreshChain(t *testing.T) {
	dir := t.TempDir()
	db := GetDatabase(dir)
	if db == nil {
		t.Fatal("GetDatabase returned nil")
	}
	defer CloseDatabase(dir)
	for _, kind := range []string{"region", "category"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM gf_dicts WHERE kind=?`, kind).Scan(&n); err != nil || n != 0 {
			t.Fatalf("新库的%s字典应为空（n=%d err=%v）", map[string]string{"region": "地区", "category": "工料机分类"}[kind], n, err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM gf_dicts WHERE kind='specialty'`).Scan(&n); err != nil || n != len(PlannedSpecialties) {
		t.Fatalf("新库的专业字典应为规划专业 %d 个（n=%d err=%v）", len(PlannedSpecialties), n, err)
	}
	if _, err := filepath.Glob(filepath.Join(dir, "Hephaestus.db")); err != nil {
		t.Fatal(err)
	}
}
