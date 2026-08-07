package characterlib

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

// GetDatabase 获取或创建指定 dataDir 的 characterlib.db 连接（单例，与 chat/office 同模式）。
func GetDatabase(dataDir string) *sql.DB {
	if dataDir == "" {
		return nil
	}
	poolsMu.Lock()
	defer poolsMu.Unlock()
	if db, ok := pools[dataDir]; ok {
		return db
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Printf("[characterlib-db] 创建 dataDir 失败: %v", err)
		return nil
	}
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "characterlib.db")+
		"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=5000")
	if err != nil {
		log.Printf("[characterlib-db] 打开数据库失败: %v", err)
		return nil
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(SchemaV1); err != nil {
		log.Printf("[characterlib-db] 建表失败: %v", err)
		_ = db.Close()
		return nil
	}
	pools[dataDir] = db
	return db
}

// CloseDatabase 关闭连接并清理 WAL。
func CloseDatabase(dataDir string) error {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	db, ok := pools[dataDir]
	if !ok {
		return nil
	}
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err := db.Close(); err != nil {
		return fmt.Errorf("关闭 characterlib.db 失败: %w", err)
	}
	delete(pools, dataDir)
	dbPath := filepath.Join(dataDir, "characterlib.db")
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	return nil
}

// SchemaV1 统一角色表 + 项目关联表。
const SchemaV1 = `
CREATE TABLE IF NOT EXISTS characters (
	id               TEXT PRIMARY KEY,
	name             TEXT NOT NULL,
	kind             TEXT NOT NULL DEFAULT 'custom',
	gender           TEXT NOT NULL DEFAULT '',
	age              TEXT NOT NULL DEFAULT '',
	tags             TEXT NOT NULL DEFAULT '[]',
	portrait_url     TEXT NOT NULL DEFAULT '',
	role_type        TEXT NOT NULL DEFAULT '',
	personality      TEXT NOT NULL DEFAULT '',
	background       TEXT NOT NULL DEFAULT '',
	appearance       TEXT NOT NULL DEFAULT '',
	figure           TEXT NOT NULL DEFAULT '',
	motivation       TEXT NOT NULL DEFAULT '',
	arc              TEXT NOT NULL DEFAULT '',
	status           TEXT NOT NULL DEFAULT '',
	notes            TEXT NOT NULL DEFAULT '',
	dialogue_samples TEXT NOT NULL DEFAULT '[]',
	chat_enabled     INTEGER NOT NULL DEFAULT 0,
	dims             TEXT NOT NULL DEFAULT '{"T":50,"I":50,"S":50,"O":50,"R":50}',
	voice_guide      TEXT NOT NULL DEFAULT '',
	behavior_rules   TEXT NOT NULL DEFAULT '',
	emotion_logic    TEXT NOT NULL DEFAULT '',
	hidden_persona   TEXT NOT NULL DEFAULT '',
	assistant_id     TEXT NOT NULL DEFAULT '',
	hidden           INTEGER NOT NULL DEFAULT 0,
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS project_characters (
	project_id      TEXT NOT NULL,
	character_id    TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
	role_in_project TEXT NOT NULL DEFAULT '',
	arc_state       TEXT NOT NULL DEFAULT '',
	status          TEXT NOT NULL DEFAULT '',
	joined_at       TEXT NOT NULL,
	PRIMARY KEY (project_id, character_id)
);
CREATE INDEX IF NOT EXISTS idx_chars_name ON characters(name);
CREATE INDEX IF NOT EXISTS idx_chars_kind ON characters(kind);
CREATE INDEX IF NOT EXISTS idx_chars_chat ON characters(chat_enabled);
`
