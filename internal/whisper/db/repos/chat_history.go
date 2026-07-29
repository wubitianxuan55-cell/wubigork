// Package repos — chat_history 聊天历史仓库
// 100% 对齐 ackem src/main/db/repos/chatHistory.ts
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wubigork/wubigork/internal/whisper/db"
)

const chatHistoryMaxRows = 2000

// LoadChatHistoryFromDB 加载聊天历史
func LoadChatHistoryFromDB(dataRoot, sessionID string) ([]map[string]interface{}, error) {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
	}

	var rowsJSON string
	err := sqlDB.QueryRow(
		"SELECT rows_json FROM chat_history WHERE session_id = ?", sessionID,
	).Scan(&rowsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 chat_history 失败: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(rowsJSON), &rows); err != nil {
		return nil, fmt.Errorf("解析 rows_json 失败: %w", err)
	}
	return rows, nil
}

// SaveChatHistoryToDB 保存聊天历史（自动裁剪至 2000 条）
func SaveChatHistoryToDB(dataRoot, sessionID string, rows []map[string]interface{}) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	// 裁剪至最近 2000 条
	if len(rows) > chatHistoryMaxRows {
		rows = rows[len(rows)-chatHistoryMaxRows:]
	}

	rowsJSON, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("序列化 rows 失败: %w", err)
	}

	updatedAt := time.Now().Format(time.RFC3339)
	_, err = sqlDB.Exec(
		`INSERT INTO chat_history(session_id, rows_json, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(session_id) DO UPDATE SET
		   rows_json = excluded.rows_json,
		   updated_at = excluded.updated_at`,
		sessionID, string(rowsJSON), updatedAt,
	)
	return err
}

// DeleteChatHistoryFromDB 删除聊天历史
func DeleteChatHistoryFromDB(dataRoot string, sessionID string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	if sessionID != "" {
		_, err := sqlDB.Exec("DELETE FROM chat_history WHERE session_id = ?", sessionID)
		return err
	}
	_, err := sqlDB.Exec("DELETE FROM chat_history")
	return err
}
