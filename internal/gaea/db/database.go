package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动（与 whisper.db 同驱动）
)

// ─── 连接池（单例 per userDir）────────────────────────────────────

var (
	pools   = make(map[string]*sql.DB)
	poolsMu sync.Mutex
)

// DatabasePath 返回 Hephaestus.db 的绝对路径（userDir 下）。
func DatabasePath(userDir string) string {
	return filepath.Join(userDir, "Hephaestus.db")
}

// GetDatabase 获取或创建指定 userDir 的 Hephaestus.db 连接（单例）。
// userDir 为空或初始化失败返回 nil（调用方按禁用处理）。
func GetDatabase(userDir string) *sql.DB {
	if userDir == "" {
		return nil
	}
	poolsMu.Lock()
	defer poolsMu.Unlock()

	if db, ok := pools[userDir]; ok {
		return db
	}

	if err := os.MkdirAll(userDir, 0o755); err != nil {
		log.Printf("[hephaestus-db] 创建 userDir 失败: %v", err)
		return nil
	}

	dbPath := DatabasePath(userDir)
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=5000&_cache_size=-8000")
	if err != nil {
		log.Printf("[hephaestus-db] 打开数据库失败: %v", err)
		return nil
	}

	db.SetMaxOpenConns(1) // SQLite 串行写入最佳实践

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-8000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			log.Printf("[hephaestus-db] PRAGMA 失败 (%s): %v", p, err)
		}
	}

	if err := runMigrations(db); err != nil {
		log.Printf("[hephaestus-db] 迁移失败: %v", err)
		db.Close()
		return nil
	}

	pools[userDir] = db
	return db
}

// CloseDatabase 关闭指定 userDir 的数据库连接（WAL checkpoint + 清理）。
func CloseDatabase(userDir string) error {
	poolsMu.Lock()
	defer poolsMu.Unlock()

	db, ok := pools[userDir]
	if !ok {
		return nil
	}
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("[hephaestus-db] WAL checkpoint 失败: %v", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("关闭数据库失败: %w", err)
	}
	delete(pools, userDir)

	dbPath := DatabasePath(userDir)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	return nil
}

// CloseAllDatabases 关闭全部连接（应用退出时调用）。
func CloseAllDatabases() {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	for userDir, db := range pools {
		_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		_ = db.Close()
		dbPath := DatabasePath(userDir)
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-shm")
	}
	pools = make(map[string]*sql.DB)
}

// WithTransaction 在事务中执行 fn，自动 commit/rollback。
func WithTransaction(userDir string, fn func(tx *sql.Tx) error) error {
	db := GetDatabase(userDir)
	if db == nil {
		return fmt.Errorf("数据库不可用: %s", userDir)
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ─── 迁移框架（对齐 whisper db）────────────────────────────────────

var migrations = []string{SchemaV1, SchemaV2, SchemaV3, SchemaV4, SchemaV5, SchemaV6, SchemaV7, SchemaV8, SchemaV9, SchemaV10, SchemaV11, SchemaV12, SchemaV13, SchemaV14}

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
