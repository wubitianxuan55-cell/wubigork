// Package db — 方案编写板块 SQLite 网关（纯 Go 驱动，与主脑库同模式）
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	pools   = make(map[string]*sql.DB)
	poolsMu sync.Mutex
)

// DatabasePath 返回 office.db 的绝对路径（officeDir 下）。
func DatabasePath(officeDir string) string {
	return filepath.Join(officeDir, "office.db")
}

// GetDatabase 获取或创建指定 officeDir 的 office.db 连接（单例）。
func GetDatabase(officeDir string) *sql.DB {
	if officeDir == "" {
		return nil
	}
	poolsMu.Lock()
	defer poolsMu.Unlock()
	if db, ok := pools[officeDir]; ok {
		return db
	}
	if err := os.MkdirAll(officeDir, 0o755); err != nil {
		log.Printf("[office-db] 创建 officeDir 失败: %v", err)
		return nil
	}
	dbPath := DatabasePath(officeDir)
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=5000&_cache_size=-8000")
	if err != nil {
		log.Printf("[office-db] 打开数据库失败: %v", err)
		return nil
	}
	db.SetMaxOpenConns(1)
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-8000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			log.Printf("[office-db] PRAGMA 失败 (%s): %v", p, err)
		}
	}
	if err := runMigrations(db); err != nil {
		log.Printf("[office-db] 迁移失败: %v", err)
		_ = db.Close()
		return nil
	}
	pools[officeDir] = db
	return db
}

// CloseDatabase 关闭指定 officeDir 的连接（WAL checkpoint + 清理）。
func CloseDatabase(officeDir string) error {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	db, ok := pools[officeDir]
	if !ok {
		return nil
	}
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err := db.Close(); err != nil {
		return fmt.Errorf("关闭数据库失败: %w", err)
	}
	delete(pools, officeDir)
	dbPath := DatabasePath(officeDir)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	return nil
}

var migrations = []string{SchemaV1, SchemaV2, SchemaV3, SchemaV4, SchemaV5, SchemaV6, SchemaV7}

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(SchemaV1); err != nil {
		return fmt.Errorf("首次建表失败: %w", err)
	}
	var currentVersion int
	err := db.QueryRow("SELECT COALESCE(CAST(value AS INTEGER), 1) FROM schema_meta WHERE key = 'user_version'").Scan(&currentVersion)
	if err != nil {
		_, err = db.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '1')")
		if err != nil {
			return fmt.Errorf("初始化 schema_meta 失败: %w", err)
		}
		currentVersion = 1
	}
	for v := currentVersion + 1; v <= len(migrations); v++ {
		sql := migrations[v-1]
		if sql != "" {
			if _, err := db.Exec(sql); err != nil {
				return fmt.Errorf("迁移 V%d 失败: %w", v, err)
			}
		}
		if _, err := db.Exec("UPDATE schema_meta SET value = ? WHERE key = 'user_version'", fmt.Sprintf("%d", v)); err != nil {
			return fmt.Errorf("更新 user_version 到 %d 失败: %w", v, err)
		}
	}
	return nil
}
