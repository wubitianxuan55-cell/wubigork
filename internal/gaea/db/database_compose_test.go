package db

// SchemaV16 组价确认留痕表迁移测试(v4.158.0 AI 组价复核闭环):
// 覆盖「新库全链(V1→V16 一次到位)」与「旧库升级(V15 终点库 → V16)」,
// 以及 CREATE IF NOT EXISTS 幂等重跑。

import (
	"database/sql"
	"testing"
)

// probeComposeTable 断言 cost_compose_records 表与索引存在,并做一次写入读回。
func probeComposeTable(t *testing.T, gdb *sql.DB) {
	t.Helper()
	var n int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='cost_compose_records'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("cost_compose_records 表缺失 (n=%d, err=%v)", n, err)
	}
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_cost_compose_records_entry'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("idx_cost_compose_records_entry 索引缺失 (n=%d, err=%v)", n, err)
	}
	// 列形状:llm_used 默认 0(显式不传时)。
	if _, err := gdb.Exec(`INSERT INTO cost_compose_records(entry_name, created_at, snapshot) VALUES('c30', '2026-09-08T00:00:00Z', '{}')`); err != nil {
		t.Fatalf("写入留痕失败: %v", err)
	}
	var name, snapshot string
	var llm int64
	if err := gdb.QueryRow(`SELECT entry_name, llm_used, snapshot FROM cost_compose_records WHERE entry_name='c30'`).Scan(&name, &llm, &snapshot); err != nil || name != "c30" || llm != 0 {
		t.Fatalf("留痕读回异常: name=%q llm=%d err=%v", name, llm, err)
	}
}

// TestSchemaV16FreshDatabaseFullChain 新库全链:V1→V16 一次到位。
func TestSchemaV16FreshDatabaseFullChain(t *testing.T) {
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
	probeComposeTable(t, gdb)

	// 幂等重跑:重复执行 V16(CREATE IF NOT EXISTS)不报错、不重复建。
	if _, err := gdb.Exec(SchemaV16); err != nil {
		t.Fatalf("SchemaV16 幂等重跑失败: %v", err)
	}
}

// TestSchemaV16UpgradeFromV15 旧库升级:V15 终点库打开后自动补 V16。
func TestSchemaV16UpgradeFromV15(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", DatabasePath(dir))
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	for i, m := range migrations[:15] {
		if _, err := raw.Exec(m); err != nil {
			t.Fatalf("apply V%d: %v", i+1, err)
		}
	}
	if _, err := raw.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '15')"); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
	raw.Close()

	// GetDatabase 打开同一目录 → 自动从 user_version=15 升到全链终点。
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
	probeComposeTable(t, gdb)
}
