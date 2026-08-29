package db

// SchemaV14 双空间列落库迁移测试（docs/gaea-space-dimension-design.md §1-2/§7 S1）：
// 覆盖「旧库升级（V13 终点库 → V14 自动迁移 + 旧行回填 work）」与
// 「新库全链（V1→V14 一次到位）」两条路径。

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildV13Database 在 dir 下手工搭一个 V13 终点库（跑 V1..V13 + user_version=13），
// 并写入迁移前的 facts/tasks 旧行（无 space_id 列值）。
func buildV13Database(t *testing.T, dir string) {
	t.Helper()
	raw, err := sql.Open("sqlite", DatabasePath(dir))
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	defer raw.Close()
	for i, m := range migrations[:13] {
		if _, err := raw.Exec(m); err != nil {
			t.Fatalf("apply V%d: %v", i+1, err)
		}
	}
	if _, err := raw.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '13')"); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
	// 旧行：列清单不含 space_id（V13 时代写入方式）
	if _, err := raw.Exec(`INSERT INTO facts(project, name, title) VALUES('proj-a', 'legacy-fact', '旧事实')`); err != nil {
		t.Fatalf("seed fact: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO tasks(id, kind, label, status) VALUES('tsk-legacy', 'price_fetch', '旧任务', 'succeeded')`); err != nil {
		t.Fatalf("seed task: %v", err)
	}
}

// probeColumn 断言列存在：列缺失时报 "no such column"（而非 ErrNoRows）。
func probeColumn(t *testing.T, gdb *sql.DB, table, col string) {
	t.Helper()
	var v string
	if err := gdb.QueryRow("SELECT " + col + " FROM " + table + " LIMIT 1").Scan(&v); err != nil && err != sql.ErrNoRows {
		t.Fatalf("列 %s.%s 探测失败: %v", table, col, err)
	}
}

func TestSchemaV14UpgradeFromV13BackfillsWork(t *testing.T) {
	dir := t.TempDir()
	buildV13Database(t, dir)

	// GetDatabase 打开同一目录 → 自动从 user_version=13 升到全链终点
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

	// 新列存在，旧行零成本回填 space_id='work'
	probeColumn(t, gdb, "facts", "space_id")
	probeColumn(t, gdb, "tasks", "space_id")
	var factSpace, taskSpace string
	if err := gdb.QueryRow(`SELECT space_id FROM facts WHERE project='proj-a' AND name='legacy-fact'`).Scan(&factSpace); err != nil || factSpace != "work" {
		t.Errorf("facts 旧行 space_id = %q (err=%v), want 'work'", factSpace, err)
	}
	if err := gdb.QueryRow(`SELECT space_id FROM tasks WHERE id='tsk-legacy'`).Scan(&taskSpace); err != nil || taskSpace != "work" {
		t.Errorf("tasks 旧行 space_id = %q (err=%v), want 'work'", taskSpace, err)
	}

	// 空间过滤索引已建
	var idx int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name IN ('idx_facts_space','idx_tasks_space')`).Scan(&idx); err != nil {
		t.Fatal(err)
	}
	if idx != 2 {
		t.Errorf("空间索引数量 = %d, want 2", idx)
	}

	// 旧唯一键未动：同 project 同名仍冲突，跨空间同名受同一约束（S1.2 再决策）
	if _, err := gdb.Exec(`INSERT INTO facts(project, name, title) VALUES('proj-a', 'legacy-fact', '重复')`); err == nil {
		t.Error("facts UNIQUE(project,name) 应保持不变（期望唯一约束冲突）")
	}
}

func TestSchemaV14FreshDatabaseFullChain(t *testing.T) {
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
	probeColumn(t, gdb, "facts", "space_id")
	probeColumn(t, gdb, "tasks", "space_id")
	var idx int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name IN ('idx_facts_space','idx_tasks_space')`).Scan(&idx); err != nil {
		t.Fatal(err)
	}
	if idx != 2 {
		t.Errorf("空间索引数量 = %d, want 2", idx)
	}

	// 数据库文件确实落盘（对齐既有测试口径）
	if _, err := filepath.Glob(filepath.Join(dir, "Hephaestus.db")); err != nil {
		t.Fatalf("Hephaestus.db not on disk: %v", err)
	}
}
