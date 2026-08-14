// Package repos — turn_traces 对话轮次追踪仓库
// 100% 对齐 ackem src/main/db/repos/turnTraces.ts
package repos

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// AppendTurnTraceToDB 追加一条轮次追踪（带会话归属，角色追踪隔离）
func AppendTurnTraceToDB(dataRoot, sessionID string, trace whisper.TurnTrace) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	traceJSON, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("序列化 trace 失败: %w", err)
	}

	date := time.Now().Format("2006-01-02")
	createdAt := time.Now().Format(time.RFC3339)

	_, err = sqlDB.Exec(
		"INSERT INTO turn_traces(date, trace_json, created_at, session_id) VALUES (?, ?, ?, ?)",
		date, string(traceJSON), createdAt, sessionID,
	)
	return err
}

// LoadTurnTracesFromDB 按日期加载轮次追踪
func LoadTurnTracesFromDB(dataRoot, date string) ([]whisper.TurnTrace, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	rows, err := sqlDB.Query(
		"SELECT trace_json FROM turn_traces WHERE date = ? ORDER BY created_at ASC",
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []whisper.TurnTrace
	for rows.Next() {
		var traceJSON string
		if err := rows.Scan(&traceJSON); err != nil {
			continue
		}
		var t whisper.TurnTrace
		if json.Unmarshal([]byte(traceJSON), &t) == nil {
			traces = append(traces, t)
		}
	}
	return traces, nil
}

// LoadTurnTracesFromDBSession 按会话加载最近 N 条轮次追踪（升序返回）
func LoadTurnTracesFromDBSession(dataRoot, sessionID string, limit int) ([]whisper.TurnTrace, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}
	if limit <= 0 || limit > 500 {
		limit = 80
	}
	rows, err := sqlDB.Query(
		"SELECT trace_json FROM turn_traces WHERE session_id = ? ORDER BY created_at DESC LIMIT ?",
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []whisper.TurnTrace
	for rows.Next() {
		var traceJSON string
		if err := rows.Scan(&traceJSON); err != nil {
			continue
		}
		var t whisper.TurnTrace
		if json.Unmarshal([]byte(traceJSON), &t) == nil {
			traces = append(traces, t)
		}
	}
	// 倒序取回后反转，按时间升序返回
	for i, j := 0, len(traces)-1; i < j; i, j = i+1, j-1 {
		traces[i], traces[j] = traces[j], traces[i]
	}
	return traces, rows.Err()
}

// CountTracesInDB 返回追踪总数
func CountTracesInDB(dataRoot string) int {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return 0
	}
	var c int
	sqlDB.QueryRow("SELECT COUNT(*) FROM turn_traces").Scan(&c)
	return c
}
