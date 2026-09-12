package db

// SchemaV21 记忆双时间轴迁移测试（v4.240.0）：覆盖「旧库升级（V20 终点库 →
// V21 自动迁移 + 旧行 recorded_at=0）」与「新库全链（V1→V21 一次到位）」两条
// 路径。0=无记录时间真相（V21 前事件行没有独立事务时间），读取端
// （memory.LoadEvents）归一回落 at，本层只验列存在与缺省口径。

import (
	"database/sql"
	"testing"
)

// buildV20Database 在 dir 下手工搭一个 V20 终点库（跑 V1..V20 + user_version=20），
// 并写入一条 V21 之前的 memory_events 旧行（列清单不含 recorded_at）。
func buildV20Database(t *testing.T, dir string) {
	t.Helper()
	raw, err := sql.Open("sqlite", DatabasePath(dir))
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	defer raw.Close()
	for i, m := range migrations[:20] {
		if _, err := raw.Exec(m); err != nil {
			t.Fatalf("apply V%d: %v", i+1, err)
		}
	}
	if _, err := raw.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '20')"); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
	// 旧行：V18 时代写入方式（at 身兼事实/记录两轴，无 recorded_at 列值）
	if _, err := raw.Exec(`INSERT INTO memory_events(at, op, name) VALUES(1700000000000, 'save', 'legacy-fact')`); err != nil {
		t.Fatalf("seed event: %v", err)
	}
}

func TestSchemaV21UpgradeFromV20BackfillsZero(t *testing.T) {
	dir := t.TempDir()
	buildV20Database(t, dir)

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

	// 新列存在，旧行零成本回填 recorded_at=0（无记录时间真相，诚实缺省）
	probeColumn(t, gdb, "memory_events", "recorded_at")
	var recorded int64
	if err := gdb.QueryRow(`SELECT recorded_at FROM memory_events WHERE name='legacy-fact'`).Scan(&recorded); err != nil || recorded != 0 {
		t.Errorf("旧行 recorded_at = %d (err=%v), want 0", recorded, err)
	}

	// 迁移往返：显式写入记录时间后可原样读出（AppendEvent/LoadEvents 同列清单）
	if _, err := gdb.Exec(`UPDATE memory_events SET recorded_at=1700000001000 WHERE name='legacy-fact'`); err != nil {
		t.Fatalf("write recorded_at: %v", err)
	}
	if err := gdb.QueryRow(`SELECT recorded_at FROM memory_events WHERE name='legacy-fact'`).Scan(&recorded); err != nil || recorded != 1700000001000 {
		t.Errorf("recorded_at 往返 = %d (err=%v), want 1700000001000", recorded, err)
	}
}

func TestSchemaV21FreshDatabaseFullChain(t *testing.T) {
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
	probeColumn(t, gdb, "memory_events", "recorded_at")
}
