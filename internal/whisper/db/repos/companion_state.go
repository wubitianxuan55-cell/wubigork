// Package repos — companion_state 同伴状态仓库
// 100% 对齐 ackem src/main/db/repos/companionState.ts
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// LoadCompanionStateFromDB 从数据库加载同伴状态
func LoadCompanionStateFromDB(dataRoot, sessionID string) (*whisper.FullState, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	var stateJSON string
	var emergenceJSON sql.NullString
	err := sqlDB.QueryRow(
		"SELECT state_json, emergence_json FROM companion_state WHERE session_id = ?",
		sessionID,
	).Scan(&stateJSON, &emergenceJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 companion_state 失败: %w", err)
	}

	var state whisper.FullState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("解析 state_json 失败: %w", err)
	}

	// 基本校验：情绪标签必须存在
	if state.Emotion.PrimaryLabel == "" {
		return nil, fmt.Errorf("invalid companion state: missing emotion label")
	}

	// 加载 emergence
	if emergenceJSON.Valid && emergenceJSON.String != "" {
		var ep whisper.EmergencePersistence
		if err := json.Unmarshal([]byte(emergenceJSON.String), &ep); err == nil {
			state.EmergencePersistence = &ep
		}
	}

	// v4.3a: 关系记忆三表回填（关联/习惯/锚点），fail-open（表失败保留 JSON 字段）
	loadMemoryGraphIntoState(sqlDB, &state)

	return &state, nil
}

// SaveCompanionStateToDB 保存同伴状态到数据库
func SaveCompanionStateToDB(dataRoot, sessionID string, state whisper.FullState) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化 state 失败: %w", err)
	}

	var emergenceJSON sql.NullString
	if state.EmergencePersistence != nil {
		b, _ := json.Marshal(state.EmergencePersistence)
		emergenceJSON = nullStr(string(b))
	}

	updatedAt := time.Now().Format(time.RFC3339)
	version := state.Version
	if version == "" {
		version = "1"
	}

	_, err = sqlDB.Exec(
		`INSERT INTO companion_state(session_id, version, state_json, emergence_json, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(session_id) DO UPDATE SET
		   version = excluded.version,
		   state_json = excluded.state_json,
		   emergence_json = excluded.emergence_json,
		   updated_at = excluded.updated_at`,
		sessionID, version, string(stateJSON), emergenceJSON, updatedAt,
	)
	if err != nil {
		return err
	}

	// v4.3a: 关系记忆三表落库（关联/习惯/锚点随状态持久化）
	if err := saveMemoryGraphFromState(sqlDB, &state); err != nil {
		return err
	}
	return nil
}

// DeleteCompanionStateFromDB 删除同伴状态
func DeleteCompanionStateFromDB(dataRoot string, sessionID string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	if sessionID != "" {
		_, err := sqlDB.Exec("DELETE FROM companion_state WHERE session_id = ?", sessionID)
		return err
	}
	_, err := sqlDB.Exec("DELETE FROM companion_state")
	return err
}
