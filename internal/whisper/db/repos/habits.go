// Package repos — user_habits 用户习惯仓库（v4.3a 会客厅·记忆持久化闭环）
// 100% 对齐 ackem src/main/db/repos/habits.ts（表结构见 schema_v6.go）
package repos

import (
	"database/sql"
	"fmt"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// HabitsRepo 用户习惯访问层（表 user_habits，id TEXT 主键）。
// 习惯表无外键、主键为业务 ID（genHexID），按主键 UPSERT 天然跨会话安全。
type HabitsRepo struct {
	sqlDB *sql.DB
}

// OpenHabitsRepo 以 dataRoot 打开习惯仓库（复用 hermes.db 单例连接）
func OpenHabitsRepo(dataRoot string) (*HabitsRepo, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}
	return &HabitsRepo{sqlDB: sqlDB}, nil
}

// NewHabitsRepo 以既有连接构造习惯仓库（供组合复用）
func NewHabitsRepo(sqlDB *sql.DB) *HabitsRepo {
	return &HabitsRepo{sqlDB: sqlDB}
}

const habitCols = "id, type, scope, weekday, hour_start, hour_end, confidence, occurrence_count, first_seen_at, last_confirmed_at, expires_at, source, suppress_target, note, created_at, updated_at"

// List 读取全部习惯
func (r *HabitsRepo) List() ([]whisper.UserHabit, error) {
	rows, err := r.sqlDB.Query("SELECT " + habitCols + " FROM user_habits")
	if err != nil {
		return nil, fmt.Errorf("查询 user_habits 失败: %w", err)
	}
	defer rows.Close()

	var out []whisper.UserHabit
	for rows.Next() {
		var h whisper.UserHabit
		var weekday, expiresAt sql.NullInt64
		var suppressTarget sql.NullString
		if err := rows.Scan(&h.ID, &h.Type, &h.Scope, &weekday, &h.HourStart, &h.HourEnd,
			&h.Confidence, &h.OccurrenceCount, &h.FirstSeenAt, &h.LastConfirmedAt, &expiresAt,
			&h.Source, &suppressTarget, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描 user_habits 失败: %w", err)
		}
		if weekday.Valid {
			v := int(weekday.Int64)
			h.Weekday = &v
		}
		if expiresAt.Valid {
			v := expiresAt.Int64
			h.ExpiresAt = &v
		}
		if suppressTarget.Valid {
			h.SuppressTarget = suppressTarget.String
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 user_habits 失败: %w", err)
	}
	return out, nil
}

// Upsert 按主键 upsert 单条习惯（ID 为空时跳过——表主键 NOT NULL）
func (r *HabitsRepo) Upsert(item whisper.UserHabit) error {
	if item.ID == "" {
		return nil
	}
	if _, err := r.sqlDB.Exec(
		`INSERT INTO user_habits(`+habitCols+`)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   type = excluded.type,
		   scope = excluded.scope,
		   weekday = excluded.weekday,
		   hour_start = excluded.hour_start,
		   hour_end = excluded.hour_end,
		   confidence = excluded.confidence,
		   occurrence_count = excluded.occurrence_count,
		   first_seen_at = excluded.first_seen_at,
		   last_confirmed_at = excluded.last_confirmed_at,
		   expires_at = excluded.expires_at,
		   source = excluded.source,
		   suppress_target = excluded.suppress_target,
		   note = excluded.note,
		   created_at = excluded.created_at,
		   updated_at = excluded.updated_at`,
		habitArgs(item)...,
	); err != nil {
		return fmt.Errorf("upsert user_habits 失败: %w", err)
	}
	return nil
}

// SaveAll 全量 upsert（主键去重，事务内完成；ID 为空的条目跳过）
func (r *HabitsRepo) SaveAll(items []whisper.UserHabit) error {
	tx, err := r.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	for _, h := range items {
		if h.ID == "" {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO user_habits(`+habitCols+`)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   type = excluded.type,
			   scope = excluded.scope,
			   weekday = excluded.weekday,
			   hour_start = excluded.hour_start,
			   hour_end = excluded.hour_end,
			   confidence = excluded.confidence,
			   occurrence_count = excluded.occurrence_count,
			   first_seen_at = excluded.first_seen_at,
			   last_confirmed_at = excluded.last_confirmed_at,
			   expires_at = excluded.expires_at,
			   source = excluded.source,
			   suppress_target = excluded.suppress_target,
			   note = excluded.note,
			   created_at = excluded.created_at,
			   updated_at = excluded.updated_at`,
			habitArgs(h)...,
		); err != nil {
			return fmt.Errorf("upsert user_habits 失败: %w", err)
		}
	}
	return tx.Commit()
}

// Clear 清空整表
func (r *HabitsRepo) Clear() error {
	if _, err := r.sqlDB.Exec("DELETE FROM user_habits"); err != nil {
		return fmt.Errorf("清空 user_habits 失败: %w", err)
	}
	return nil
}

// habitArgs 组装 user_habits 行参数（指针字段转 NULL 兼容）
func habitArgs(h whisper.UserHabit) []interface{} {
	var weekday, expiresAt interface{}
	if h.Weekday != nil {
		weekday = *h.Weekday
	}
	if h.ExpiresAt != nil {
		expiresAt = *h.ExpiresAt
	}
	var suppressTarget interface{}
	if h.SuppressTarget != "" {
		suppressTarget = h.SuppressTarget
	}
	return []interface{}{
		h.ID, h.Type, h.Scope, weekday, h.HourStart, h.HourEnd, h.Confidence,
		h.OccurrenceCount, h.FirstSeenAt, h.LastConfirmedAt, expiresAt,
		h.Source, suppressTarget, h.Note, h.CreatedAt, h.UpdatedAt,
	}
}
