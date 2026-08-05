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

	for _, tbl := range []string{"projects", "proposals", "sections", "files", "versions", "templates", "schema_meta"} {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&n); err != nil {
			t.Fatalf("query %s: %v", tbl, err)
		}
		if n != 1 {
			t.Errorf("table %s not created", tbl)
		}
	}
	var ver int
	if err := db.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Errorf("user_version = %d, want %d", ver, len(migrations))
	}
	if _, err := filepath.Glob(filepath.Join(dir, "office.db")); err != nil {
		t.Fatalf("office.db not on disk: %v", err)
	}
}
