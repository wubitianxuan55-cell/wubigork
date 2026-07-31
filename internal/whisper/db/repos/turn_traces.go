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

// AppendTurnTraceToDB 追加一条轮次追踪
func AppendTurnTraceToDB(dataRoot string, trace whisper.TurnTrace) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	traceJSON, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("序列化 trace 失败: %w", err)
	}

	date := time.Now().Format("2006-01-02")
	createdAt := time.Now().Format(time.RFC3339)

	_, err = sqlDB.Exec(
		"INSERT INTO turn_traces(date, trace_json, created_at) VALUES (?, ?, ?)",
		date, string(traceJSON), createdAt,
	)
	return err
}

// LoadTurnTracesFromDB 按日期加载轮次追踪
func LoadTurnTracesFromDB(dataRoot, date string) ([]whisper.TurnTrace, error) {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
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

// CountTracesInDB 返回追踪总数
func CountTracesInDB(dataRoot string) int {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return 0
	}
	var c int
	sqlDB.QueryRow("SELECT COUNT(*) FROM turn_traces").Scan(&c)
	return c
}
