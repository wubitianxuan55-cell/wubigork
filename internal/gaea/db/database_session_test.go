package db

// SchemaV20 任务会话维度迁移测试（v4.180 结构刀）：覆盖「旧库升级（V19 终点库
// → V20 自动迁移 + 旧行回填空串）」与「新库全链（V1→V20 一次到位）」两条路径。
// 空串语义=非会话入口（cron/系统周期任务诚实留空不造数）。

import (
	"database/sql"
	"testing"
)

// buildV19Database 在 dir 下手工搭一个 V19 终点库（跑 V1..V19 + user_version=19），
// 并写入迁移前的 tasks 旧行（无 session_id 列值）。
func buildV19Database(t *testing.T, dir string) {
	t.Helper()
	raw, err := sql.Open("sqlite", DatabasePath(dir))
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	defer raw.Close()
	for i, m := range migrations[:19] {
		if _, err := raw.Exec(m); err != nil {
			t.Fatalf("apply V%d: %v", i+1, err)
		}
	}
	if _, err := raw.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '19')"); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
	// 旧行：列清单不含 session_id（V19 时代写入方式）
	if _, err := raw.Exec(`INSERT INTO tasks(id, kind, label, status) VALUES('tsk-legacy', 'price_fetch', '旧任务', 'succeeded')`); err != nil {
		t.Fatalf("seed task: %v", err)
	}
}

func TestSchemaV20UpgradeFromV19BackfillsEmpty(t *testing.T) {
	dir := t.TempDir()
	buildV19Database(t, dir)

	// GetDatabase 打开同一目录 → 自动从 user_version=19 升到全链终点
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

	// 新列存在，旧行零成本回填 session_id=''（非会话入口诚实留空）
	probeColumn(t, gdb, "tasks", "session_id")
	var taskSession string
	if err := gdb.QueryRow(`SELECT session_id FROM tasks WHERE id='tsk-legacy'`).Scan(&taskSession); err != nil || taskSession != "" {
		t.Errorf("tasks 旧行 session_id = %q (err=%v), want 空串", taskSession, err)
	}

	// 迁移往返：显式写入会话标识后可原样读出（读取端 Get/List 同列清单）
	if _, err := gdb.Exec(`UPDATE tasks SET session_id=? WHERE id='tsk-legacy'`, "sa_roundtrip"); err != nil {
		t.Fatalf("write session_id: %v", err)
	}
	if err := gdb.QueryRow(`SELECT session_id FROM tasks WHERE id='tsk-legacy'`).Scan(&taskSession); err != nil || taskSession != "sa_roundtrip" {
		t.Errorf("tasks session_id 往返 = %q (err=%v), want sa_roundtrip", taskSession, err)
	}
}

func TestSchemaV20FreshDatabaseFullChain(t *testing.T) {
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
	probeColumn(t, gdb, "tasks", "session_id")
	var taskSession string
	if err := gdb.QueryRow(`SELECT session_id FROM tasks LIMIT 1`).Scan(&taskSession); err != nil && err != sql.ErrNoRows {
		t.Fatalf("session_id 探测失败: %v", err)
	}
}
