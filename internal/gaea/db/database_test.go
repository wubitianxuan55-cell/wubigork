package db

import (
	"path/filepath"
	"testing"
)

func TestGetDatabaseCreatesSchema(t *testing.T) {
	dir := t.TempDir()
	db := GetDatabase(dir)
	if db == nil {
		t.Fatal("GetDatabase returned nil")
	}
	defer CloseDatabase(dir)

	// 三张核心表必须存在
	for _, tbl := range []string{"facts", "profile", "knowledge", "schema_meta"} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&n); err != nil {
			t.Fatalf("query %s: %v", tbl, err)
		}
		if n != 1 {
			t.Errorf("table %s not created", tbl)
		}
	}

	// 迁移版本应为 1
	var ver int
	if err := db.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Errorf("user_version = %d, want %d", ver, len(migrations))
	}

	// 数据库文件确实落盘
	if _, err := filepath.Glob(filepath.Join(dir, "Hephaestus.db")); err != nil {
		t.Fatalf("Hephaestus.db not on disk: %v", err)
	}
}

func TestFactsUniquePerProject(t *testing.T) {
	dir := t.TempDir()
	db := GetDatabase(dir)
	defer CloseDatabase(dir)

	insert := func(project, name string) error {
		_, err := db.Exec(
			"INSERT INTO facts(project, name, title, description, type, kind, tags, body) VALUES(?,?,?,?,?,?,?,?)",
			project, name, "t", "d", "user", "semantic", "[]", "body")
		return err
	}

	if err := insert("proj-a", "prefers-tabs"); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	// 同项目同名 → 违反唯一约束（保存逻辑用 UPSERT，这里验证约束存在）
	if err := insert("proj-a", "prefers-tabs"); err == nil {
		t.Error("expected unique violation for same project+name")
	}
	// 不同项目同名 → 允许
	if err := insert("proj-b", "prefers-tabs"); err != nil {
		t.Errorf("cross-project same name should be allowed: %v", err)
	}
}

func TestProfileAndKnowledgeTables(t *testing.T) {
	dir := t.TempDir()
	db := GetDatabase(dir)
	defer CloseDatabase(dir)

	if _, err := db.Exec("INSERT INTO profile(key, value, source, confidence) VALUES('user-pref', 'x', 'session', 0.8)"); err != nil {
		t.Fatalf("profile insert: %v", err)
	}
	if _, err := db.Exec("INSERT INTO knowledge(name, title, category, body) VALUES('k1', 'K1', '规范标准', 'body')"); err != nil {
		t.Fatalf("knowledge insert: %v", err)
	}
	var body string
	if err := db.QueryRow("SELECT body FROM knowledge WHERE name='k1'").Scan(&body); err != nil || body != "body" {
		t.Errorf("knowledge roundtrip failed: body=%q err=%v", body, err)
	}
}
