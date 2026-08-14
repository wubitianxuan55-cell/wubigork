// Package repos — kv 键值存储仓库
// 100% 对齐 ackem src/main/db/repos/kv.ts
package repos

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper/db"
)

// KVGet 读取键值
func KVGet(dataRoot, namespace, key string) (string, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return "", fmt.Errorf("数据库不可用: %w", openErr)
	}

	var value string
	err := sqlDB.QueryRow(
		"SELECT value FROM kv_store WHERE namespace = ? AND key = ?",
		namespace, key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// KVSet 设置键值（UPSERT）
func KVSet(dataRoot, namespace, key, value string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	updatedAt := time.Now().Format(time.RFC3339)
	_, err := sqlDB.Exec(
		`INSERT INTO kv_store(namespace, key, value, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(namespace, key) DO UPDATE SET
		   value = excluded.value,
		   updated_at = excluded.updated_at`,
		namespace, key, value, updatedAt,
	)
	return err
}

// KVDeleteNamespace 删除整个命名空间
func KVDeleteNamespace(dataRoot, namespace string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	_, err := sqlDB.Exec("DELETE FROM kv_store WHERE namespace = ?", namespace)
	return err
}
