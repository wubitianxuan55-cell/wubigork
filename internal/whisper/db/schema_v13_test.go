package db

import (
	"testing"
)

// TestSchemaV13_DropsWeixinTables 迁移链到 V13 后 weixin_* 4 表被 DROP（死表治理），
// 既有表（memory_facts，V1 建表）不受影响。
func TestSchemaV13_DropsWeixinTables(t *testing.T) {
	dir := t.TempDir()
	db, err := GetDatabase(dir)
	if err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	t.Cleanup(func() { _ = CloseDatabase(dir) })

	// 迁移应已推进到 V13
	var v string
	if err := db.QueryRow("SELECT value FROM schema_meta WHERE key='user_version'").Scan(&v); err != nil {
		t.Fatalf("查询 user_version: %v", err)
	}
	if v != "13" {
		t.Fatalf("user_version = %s, want 13", v)
	}

	// 4 张 weixin_* 表应不存在
	for _, tbl := range []string{"weixin_account", "weixin_sync", "weixin_context", "weixin_seen"} {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
		if err == nil {
			t.Errorf("表 %s 应已被 SchemaV13 DROP", tbl)
		}
	}

	// 既有表不受影响
	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='memory_facts'").Scan(&cnt); err != nil {
		t.Fatalf("查询 memory_facts: %v", err)
	}
	if cnt != 1 {
		t.Errorf("memory_facts 应存在, got %d", cnt)
	}
}
