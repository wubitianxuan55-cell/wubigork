package db

// SchemaV17 条目匹配键迁移测试(v4.178.0 造价·条目匹配键刀):
// 覆盖「新库全链(V1→V17 一次到位)」与「旧库升级(V16 终点库 → V17)」,
// 并断言 code 列与索引存在、旧行默认空串。

import (
	"database/sql"
	"testing"
)

// probeCodeColumns 断言 cost_entries/cost_estimate_items 的 code 列与
// idx_cost_code 索引存在,并验证旧行 code 默认空串。
func probeCodeColumns(t *testing.T, gdb *sql.DB) {
	t.Helper()
	for _, tbl := range []string{"cost_entries", "cost_estimate_items"} {
		rows, err := gdb.Query("SELECT name FROM pragma_table_info('" + tbl + "')")
		if err != nil {
			t.Fatalf("pragma_table_info(%s): %v", tbl, err)
		}
		hasCode := false
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}
			if name == "code" {
				hasCode = true
			}
		}
		rows.Close()
		if !hasCode {
			t.Fatalf("%s 缺少 code 列", tbl)
		}
	}
	var n int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_cost_code'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("idx_cost_code 索引缺失 (n=%d, err=%v)", n, err)
	}
	// 旧行语义:未录入编码 = 空串默认值。
	if _, err := gdb.Exec(`INSERT INTO cost_entries(name, title) VALUES('code-probe', '探针条目')`); err != nil {
		t.Fatalf("写入探针失败: %v", err)
	}
	var code string
	if err := gdb.QueryRow(`SELECT code FROM cost_entries WHERE name='code-probe'`).Scan(&code); err != nil || code != "" {
		t.Fatalf("code 默认值异常: code=%q err=%v", code, err)
	}
}

// TestSchemaV17FreshDatabaseFullChain 新库全链:V1→V17 一次到位。
func TestSchemaV17FreshDatabaseFullChain(t *testing.T) {
	dir := t.TempDir()
	gdb := GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase returned nil")
	}
	defer CloseDatabase(dir)

	var ver int
	if err := gdb.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Fatalf("user_version = %d, want %d", ver, len(migrations))
	}
	probeCodeColumns(t, gdb)
}

// TestSchemaV17UpgradeFromV16 旧库升级:V16 终点库打开后自动补 V17。
func TestSchemaV17UpgradeFromV16(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", DatabasePath(dir))
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	for i, m := range migrations[:16] {
		if _, err := raw.Exec(m); err != nil {
			t.Fatalf("apply V%d: %v", i+1, err)
		}
	}
	if _, err := raw.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '16')"); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
	// V16 终点库的历史数据:一条既有成本条目,升级后 code 必须为空串。
	if _, err := raw.Exec(`INSERT INTO cost_entries(name, title) VALUES('legacy', '旧条目')`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	raw.Close()

	gdb := GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase returned nil")
	}
	defer CloseDatabase(dir)

	var ver int
	if err := gdb.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Fatalf("升级后 user_version = %d, want %d", ver, len(migrations))
	}
	var code string
	if err := gdb.QueryRow(`SELECT code FROM cost_entries WHERE name='legacy'`).Scan(&code); err != nil || code != "" {
		t.Fatalf("历史行 code 异常: %q err=%v", code, err)
	}
	probeCodeColumns(t, gdb)
}
