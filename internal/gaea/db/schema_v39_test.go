package db

import (
	"database/sql"
	"testing"
)

// TestSchemaV39UpgradeFromV38 造价自有编码迁移（2026-11-25）：从 V38 的库升上来——
//   - 三表各加 gcode 列（存量值全空，补授走 workcost.EnsureGCodes，不在迁移里做）；
//   - 唯一索引是部分索引（WHERE gcode<>”），放行待授码的空值；
//   - gf_gcode_seq 水位表就位（授码不复用的机制保障）。
func TestSchemaV39UpgradeFromV38(t *testing.T) {
	dir := buildAtVersion(t, 38)
	{
		raw, err := sql.Open("sqlite", DatabasePath(dir))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(`INSERT INTO gf_quotas(code, title, specialty, unit, source, created_at, updated_at)
VALUES('WP01','测量放线','土壤修复','项','项目导入：A.xlsx','t','t')`); err != nil {
			t.Fatalf("摆定额失败: %v", err)
		}
		_ = raw.Close()
	}

	g := GetDatabase(dir)
	if g == nil {
		t.Fatal("升级后库打不开（迁移失败，见日志）")
	}
	defer CloseDatabase(dir)

	var ver int
	if err := g.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Fatalf("user_version = %d, want %d", ver, len(migrations))
	}

	// 三表 gcode 列到位；存量行 gcode 为空（补授不在迁移里做）。
	for _, c := range []string{"gf_quotas", "gf_bill_items", "gf_resources"} {
		var n int
		if err := g.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name='gcode'`, c).Scan(&n); err != nil || n != 1 {
			t.Errorf("%s.gcode 应已加上（n=%d err=%v）", c, n, err)
		}
	}
	var gc string
	if err := g.QueryRow(`SELECT gcode FROM gf_quotas WHERE code='WP01'`).Scan(&gc); err != nil || gc != "" {
		t.Errorf("存量行 gcode 应为空（待 workcost.EnsureGCodes 补授），得到 %q err=%v", gc, err)
	}
	// 部分唯一索引：多个空码可并存，同码非空互斥。
	for i := 0; i < 2; i++ {
		if _, err := g.Exec(`INSERT INTO gf_quotas(code, title, source, created_at, updated_at)
VALUES(?,?,?,?,?)`, "N"+string(rune('A'+i)), "空码行", "t", "t", "t"); err != nil {
			t.Fatalf("空 gcode 行应可并存: %v", err)
		}
	}
	if _, err := g.Exec(`INSERT INTO gf_quotas(code, title, gcode, source, created_at, updated_at)
VALUES('G1','x','DSZ01001','t','t','t')`); err != nil {
		t.Fatalf("授码行应可插入: %v", err)
	}
	if _, err := g.Exec(`INSERT INTO gf_quotas(code, title, gcode, source, created_at, updated_at)
VALUES('G2','x','DSZ01001','t','t','t')`); err == nil {
		t.Error("同 gcode 第二行应被部分唯一索引拦下")
	}
	// 水位表就位。
	var n int
	if err := g.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('gf_gcode_seq') WHERE name='last'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("gf_gcode_seq 水位表应已建（n=%d err=%v）", n, err)
	}
}
