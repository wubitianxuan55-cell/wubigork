// Package repos — openforu OpenForU 数据仓库
// 100% 对齐 ackem src/main/db/repos/openforu.ts
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper/db"
)

// ─── 工作区 ──────────────────────────────────────────────────────

// OpenForUWorkspaceIndex 工作区索引
type OpenForUWorkspaceIndex struct {
	Workspaces []OpenForUWorkspace `json:"workspaces"`
}

// OpenForUWorkspace 工作区
type OpenForUWorkspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SaveWorkspaceIndexToDB 保存工作区索引
func SaveWorkspaceIndexToDB(dataRoot string, index OpenForUWorkspaceIndex) error {
	dataJSON, err := json.Marshal(index)
	if err != nil {
		return err
	}

	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	updatedAt := time.Now().Format(time.RFC3339)
	_, err = sqlDB.Exec(
		`INSERT INTO openforu_workspaces(id, data_json, updated_at)
		 VALUES ('index', ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   data_json = excluded.data_json,
		   updated_at = excluded.updated_at`,
		string(dataJSON), updatedAt,
	)
	return err
}

// LoadWorkspaceIndexFromDB 加载工作区索引
func LoadWorkspaceIndexFromDB(dataRoot string) (*OpenForUWorkspaceIndex, error) {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
	}

	var dataJSON string
	err := sqlDB.QueryRow(
		"SELECT data_json FROM openforu_workspaces WHERE id = 'index'",
	).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return &OpenForUWorkspaceIndex{}, nil
	}
	if err != nil {
		return nil, err
	}

	var index OpenForUWorkspaceIndex
	if err := json.Unmarshal([]byte(dataJSON), &index); err != nil {
		return nil, fmt.Errorf("解析工作区索引失败: %w", err)
	}
	return &index, nil
}

// ─── 规划会话 ────────────────────────────────────────────────────

// PlanSession 规划会话（简化版，存储 JSON blob）
type PlanSession struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Title       string `json:"title"`
	DataJSON    string `json:"dataJson"`
}

// SavePlanSessionToDB 保存规划会话
func SavePlanSessionToDB(dataRoot string, session PlanSession) error {
	dataJSON, err := json.Marshal(session)
	if err != nil {
		return err
	}

	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	updatedAt := time.Now().Format(time.RFC3339)
	_, err = sqlDB.Exec(
		`INSERT INTO openforu_sessions(id, workspace_id, data_json, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   workspace_id = excluded.workspace_id,
		   data_json = excluded.data_json,
		   updated_at = excluded.updated_at`,
		session.ID, session.WorkspaceID, string(dataJSON), updatedAt,
	)
	return err
}

// LoadPlanSessionFromDB 加载规划会话
func LoadPlanSessionFromDB(dataRoot, sessionID string) (*PlanSession, error) {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
	}

	var dataJSON string
	err := sqlDB.QueryRow(
		"SELECT data_json FROM openforu_sessions WHERE id = ?",
		sessionID,
	).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var session PlanSession
	if err := json.Unmarshal([]byte(dataJSON), &session); err != nil {
		return nil, fmt.Errorf("解析规划会话失败: %w", err)
	}
	return &session, nil
}

// ─── Agent 运行 ──────────────────────────────────────────────────

// AgentRunMeta Agent 运行元数据
type AgentRunMeta struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	DataJSON  string `json:"dataJson"`
}

// SaveAgentRunToDB 保存 Agent 运行
func SaveAgentRunToDB(dataRoot string, run AgentRunMeta) error {
	dataJSON, err := json.Marshal(run)
	if err != nil {
		return err
	}

	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	updatedAt := time.Now().Format(time.RFC3339)
	_, err = sqlDB.Exec(
		`INSERT INTO openforu_runs(id, session_id, data_json, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   session_id = excluded.session_id,
		   data_json = excluded.data_json,
		   updated_at = excluded.updated_at`,
		run.ID, run.SessionID, string(dataJSON), updatedAt,
	)
	return err
}

// CountOpenForuSessionsInDB 返回 OpenForU 会话总数
func CountOpenForuSessionsInDB(dataRoot string) int {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return 0
	}
	var c int
	sqlDB.QueryRow("SELECT COUNT(*) FROM openforu_sessions").Scan(&c)
	return c
}
