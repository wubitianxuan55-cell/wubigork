package db

import (
	"os"
	"path/filepath"
	"testing"
)

// ── GetDatabase 签名（(*sql.DB, error)）相关测试（T6-5.5）──────────

// TestGetDatabase_ErrorPathReturnsError dataRoot 不可用（被普通文件占位）时
// GetDatabase 必须返回 error 而非 nil 连接——调用方据此处理，不再判空吞错。
func TestGetDatabase_ErrorPathReturnsError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("写占位文件: %v", err)
	}

	got, err := GetDatabase(blocker)
	if err == nil {
		t.Fatal("dataRoot 被文件占位时应返回 error")
	}
	if got != nil {
		t.Fatalf("失败时不应返回连接, got %v", got)
	}
}

// TestGetDatabase_SuccessPathAndDSN dataRoot 正常时返回可用连接与 nil error；
// 同时验证 DSN 是 PRAGMA 唯一来源——foreign_keys 未重复执行 PRAGMA 也生效。
func TestGetDatabase_SuccessPathAndDSN(t *testing.T) {
	dir := t.TempDir()

	db, err := GetDatabase(dir)
	if err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	if db == nil {
		t.Fatal("GetDatabase 返回 nil 连接")
	}

	// 单例：同一 dataRoot 返回同一连接
	db2, err2 := GetDatabase(dir)
	if err2 != nil {
		t.Fatalf("GetDatabase 第二次调用: %v", err2)
	}
	if db != db2 {
		t.Fatal("同一 dataRoot 应返回同一连接（单例）")
	}

	// DSN 单源断言：foreign_keys 由 DSN 参数启用（无 PRAGMA 循环兜底）
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("查询 foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys 应由 DSN 参数启用, got %d", fk)
	}

	// 迁移生效：schema_meta 存在
	var v string
	if err := db.QueryRow("SELECT value FROM schema_meta LIMIT 1").Scan(&v); err != nil {
		t.Fatalf("查询 schema_meta: %v", err)
	}

	if err := CloseDatabase(dir); err != nil {
		t.Fatalf("CloseDatabase: %v", err)
	}
}
