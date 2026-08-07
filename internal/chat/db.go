// Package chat — 聊天板块统一会话存储（话题 + 消息，SQLite）。
package chat

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

// GetDatabase 获取或创建指定 dataDir 的 chat.db 连接（单例，与 office.db 同模式）。
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
		log.Printf("[chat-db] 创建 dataDir 失败: %v", err)
		return nil
	}
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "chat.db")+
		"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=5000")
	if err != nil {
		log.Printf("[chat-db] 打开数据库失败: %v", err)
		return nil
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(SchemaV1); err != nil {
		log.Printf("[chat-db] 建表失败: %v", err)
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
		return fmt.Errorf("关闭 chat.db 失败: %w", err)
	}
	delete(pools, dataDir)
	dbPath := filepath.Join(dataDir, "chat.db")
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	return nil
}

// SchemaV1 话题 + 消息表（foreign_keys=ON 级联删除）。
const SchemaV1 = `
CREATE TABLE IF NOT EXISTS chat_topics (
	id         TEXT PRIMARY KEY,
	title      TEXT NOT NULL,
	mode       TEXT NOT NULL DEFAULT 'plain',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS chat_messages (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	topic_id   TEXT NOT NULL REFERENCES chat_topics(id) ON DELETE CASCADE,
	role       TEXT NOT NULL,
	content    TEXT NOT NULL,
	extra      TEXT NOT NULL DEFAULT '',
	seq        INTEGER NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_topic ON chat_messages(topic_id, seq);
`
