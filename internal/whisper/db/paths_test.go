package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyDB(t *testing.T) {
	dir := t.TempDir()
	// 构造旧库 whisper.db（含 WAL）
	old := filepath.Join(dir, LegacyDBFilename)
	if err := os.WriteFile(old, []byte("legacy-sqlite-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old+"-wal", []byte("wal"), 0o644); err != nil {
		t.Fatal(err)
	}

	migrateLegacyDB(dir)

	newPath := filepath.Join(dir, HermesDBFilename)
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("hermes.db not created: %v", err)
	}
	if string(data) != "legacy-sqlite-content" {
		t.Errorf("hermes.db content mismatch: %q", data)
	}
	// WAL 也迁移
	if _, err := os.Stat(newPath + "-wal"); err != nil {
		t.Error("hermes.db-wal not migrated")
	}
	// 旧库保留（备份）
	if _, err := os.Stat(old); err != nil {
		t.Error("legacy whisper.db should be kept as backup")
	}

	// 幂等：再次调用（新库已存在）不覆盖
	migrateLegacyDB(dir)
	data2, _ := os.ReadFile(newPath)
	if string(data2) != "legacy-sqlite-content" {
		t.Error("second migrate should be a no-op")
	}
}

func TestMigrateLegacyDBSkipsWhenNoLegacy(t *testing.T) {
	dir := t.TempDir()
	migrateLegacyDB(dir)
	if _, err := os.Stat(filepath.Join(dir, HermesDBFilename)); err == nil {
		t.Error("hermes.db should not be created when no legacy db exists")
	}
}
