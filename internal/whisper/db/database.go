// Package db — whisper 嵌入式 SQLite 网关（单例 per dataRoot）
// 100% 对齐 ackem src/main/db/database.ts
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动
)

// ─── 连接池 ──────────────────────────────────────────────────────

var (
	pools   = make(map[string]*sql.DB)
	poolsMu sync.Mutex
)

// ─── 公共 API ────────────────────────────────────────────────────

// GetDatabase 获取或创建指定 dataRoot 的数据库连接（单例）
// 返回 nil 表示初始化失败
func GetDatabase(dataRoot string) *sql.DB {
	poolsMu.Lock()
	defer poolsMu.Unlock()

	if db, ok := pools[dataRoot]; ok {
		return db
	}

	if err := EnsureDataRoot(dataRoot); err != nil {
		log.Printf("[whisper-db] 创建 dataRoot 失败: %v", err)
		return nil
	}

	// 旧库迁移（whisper.db -> hermes.db，首次运行新文件名时执行一次）
	migrateLegacyDB(dataRoot)

	dbPath := DatabasePath(dataRoot)
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=5000&_cache_size=-8000")
	if err != nil {
		log.Printf("[whisper-db] 打开数据库失败: %v", err)
		return nil
	}

	// 配置连接池
	db.SetMaxOpenConns(1) // SQLite 串行写入最佳实践

	// 应用 PRAGMA
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-8000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			log.Printf("[whisper-db] PRAGMA 失败 (%s): %v", p, err)
		}
	}

	// 执行迁移
	if err := runMigrations(db); err != nil {
		log.Printf("[whisper-db] 迁移失败: %v", err)
		db.Close()
		return nil
	}

	pools[dataRoot] = db
	return db
}

// CloseDatabase 关闭指定 dataRoot 的数据库连接
func CloseDatabase(dataRoot string) error {
	poolsMu.Lock()
	defer poolsMu.Unlock()

	db, ok := pools[dataRoot]
	if !ok {
		return nil
	}

	// WAL checkpoint
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("[whisper-db] WAL checkpoint 失败: %v", err)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("关闭数据库失败: %w", err)
	}

	delete(pools, dataRoot)

	// 清理 WAL/SHM 文件
	dbPath := DatabasePath(dataRoot)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	return nil
}

// CloseAllDatabases 关闭所有数据库连接
func CloseAllDatabases() {
	poolsMu.Lock()
	defer poolsMu.Unlock()

	for dataRoot, db := range pools {
		db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		db.Close()

		dbPath := DatabasePath(dataRoot)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}

	pools = make(map[string]*sql.DB)
}

// ─── 事务 ────────────────────────────────────────────────────────

// WithTransaction 在事务中执行函数，自动 commit/rollback
func WithTransaction(dataRoot string, fn func(tx *sql.Tx) error) error {
	db := GetDatabase(dataRoot)
	if db == nil {
		return fmt.Errorf("数据库不可用: %s", dataRoot)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// ─── 迁移框架 ────────────────────────────────────────────────────

// migration SQL 列表，user_version N 对应 migrations[N-1]
var migrations = []string{
	SchemaV1,
	SchemaV2,
	SchemaV3,
	SchemaV4,
	SchemaV5,
	SchemaV6,
	SchemaV7,
	SchemaV8,
	SchemaV9,
	SchemaV10,
	SchemaV11,
	SchemaV12,
}

// runMigrations 执行递增迁移链 V1 → V11
func runMigrations(db *sql.DB) error {
	// 确保 schema_meta 表存在（首次运行）
	if _, err := db.Exec(SchemaV1); err != nil {
		return fmt.Errorf("首次建表失败: %w", err)
	}

	// 读取当前 user_version
	var currentVersion int
	err := db.QueryRow("SELECT COALESCE(CAST(value AS INTEGER), 1) FROM schema_meta WHERE key = 'user_version'").Scan(&currentVersion)
	if err != nil {
		// schema_meta 表刚创建，插入初始版本
		_, err = db.Exec("INSERT INTO schema_meta(key, value) VALUES('user_version', '1')")
		if err != nil {
			return fmt.Errorf("初始化 schema_meta 失败: %w", err)
		}
		currentVersion = 1
	}

	// 依次执行后续迁移
	for v := currentVersion + 1; v <= len(migrations); v++ {
		sql := migrations[v-1] // 0-indexed
		if sql == "" {
			// 空迁移（如 V3），仅升级版本号
		} else {
			if _, err := db.Exec(sql); err != nil {
				return fmt.Errorf("迁移 V%d 失败: %w", v, err)
			}
		}

		// 更新版本号
		_, err := db.Exec("UPDATE schema_meta SET value = ? WHERE key = 'user_version'", fmt.Sprintf("%d", v))
		if err != nil {
			return fmt.Errorf("更新 user_version 到 %d 失败: %w", v, err)
		}
	}

	return nil
}

// ─── 清理 ────────────────────────────────────────────────────────

// clearStructuredData 清空所有结构化数据（保留 schema 结构）
func ClearStructuredData(dataRoot string) error {
	db := GetDatabase(dataRoot)
	if db == nil {
		return fmt.Errorf("数据库不可用")
	}

	tables := []string{
		"companion_state", "chat_history", "memory_facts",
		"episodes", "procedural_habits", "kv_store",
		"knowledge_triples", "turn_traces", "diary",
		"openforu_workspaces", "openforu_sessions", "openforu_runs",
		"shared_events", "memory_associations", "temporal_anchors",
		"user_habits", "foreground_history", "decision_log",
		"fact_embeddings",
		"weixin_account", "weixin_sync", "weixin_context", "weixin_seen",
	}

	return WithTransaction(dataRoot, func(tx *sql.Tx) error {
		for _, table := range tables {
			if _, err := tx.Exec("DELETE FROM " + table); err != nil {
				// 表可能不存在（旧版本），忽略
				continue
			}
		}
		return nil
	})
}


// migrateLegacyDB 将旧库 whisper.db（含 WAL/SHM）复制为 hermes.db。
// 仅当新库不存在且旧库存在时执行一次；保留旧文件作备份，不删除。
func migrateLegacyDB(dataRoot string) {
	newPath := DatabasePath(dataRoot)
	oldPath := filepath.Join(dataRoot, LegacyDBFilename)
	if _, err := os.Stat(newPath); err == nil {
		return // 新库已存在
	}
	if _, err := os.Stat(oldPath); err != nil {
		return // 无旧库，全新安装
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		src := oldPath + suffix
		dst := newPath + suffix
		if _, err := os.Stat(src); err != nil {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			log.Printf("[whisper-db] 旧库迁移失败 (%s): %v", src, err)
		}
	}
	log.Printf("[whisper-db] 已迁移旧库 %s -> %s", LegacyDBFilename, HermesDBFilename)
}
