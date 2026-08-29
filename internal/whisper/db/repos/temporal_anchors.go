// Package repos — temporal_anchors 时间锚点仓库（v4.3a 会客厅·记忆持久化闭环）
// 100% 对齐 ackem src/main/db/repos/temporalAnchors.ts（表结构见 schema_v4.go）
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// temporalAnchorRow 表行（与 temporal_anchors 列一一对应）
type temporalAnchorRow struct {
	ID                 string
	AnchorDate         string
	AnchorType         string
	RecurrenceRule     sql.NullString
	LinkedFactIDs      sql.NullString
	EmotionalValence   float64
	EmotionalIntensity float64
	Domain             sql.NullString
	Summary            string
}

func (r *temporalAnchorRow) toAnchor() whisper.TemporalAnchor {
	a := whisper.TemporalAnchor{
		ID:                 r.ID,
		AnchorDate:         r.AnchorDate,
		AnchorType:         whisper.TemporalAnchorType(r.AnchorType),
		EmotionalValence:   r.EmotionalValence,
		EmotionalIntensity: r.EmotionalIntensity,
		Summary:            r.Summary,
	}
	if r.RecurrenceRule.Valid {
		a.RecurrenceRule = r.RecurrenceRule.String
	}
	if r.LinkedFactIDs.Valid && r.LinkedFactIDs.String != "" {
		_ = json.Unmarshal([]byte(r.LinkedFactIDs.String), &a.LinkedFactIDs)
	}
	if r.Domain.Valid {
		a.Domain = r.Domain.String
	}
	return a
}

// TemporalAnchorsRepo 时间锚点访问层（表 temporal_anchors，id TEXT 主键，无外键）
type TemporalAnchorsRepo struct {
	sqlDB *sql.DB
}

// OpenTemporalAnchorsRepo 以 dataRoot 打开锚点仓库（复用 hermes.db 单例连接）
func OpenTemporalAnchorsRepo(dataRoot string) (*TemporalAnchorsRepo, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}
	return &TemporalAnchorsRepo{sqlDB: sqlDB}, nil
}

// NewTemporalAnchorsRepo 以既有连接构造锚点仓库（供组合复用）
func NewTemporalAnchorsRepo(sqlDB *sql.DB) *TemporalAnchorsRepo {
	return &TemporalAnchorsRepo{sqlDB: sqlDB}
}

const anchorCols = "id, anchor_date, anchor_type, recurrence_rule, linked_fact_ids, emotional_valence, emotional_intensity, domain, summary"

// List 读取全部时间锚点
func (r *TemporalAnchorsRepo) List() ([]whisper.TemporalAnchor, error) {
	rows, err := r.sqlDB.Query("SELECT " + anchorCols + " FROM temporal_anchors")
	if err != nil {
		return nil, fmt.Errorf("查询 temporal_anchors 失败: %w", err)
	}
	defer rows.Close()

	var out []whisper.TemporalAnchor
	for rows.Next() {
		var row temporalAnchorRow
		if err := rows.Scan(&row.ID, &row.AnchorDate, &row.AnchorType, &row.RecurrenceRule,
			&row.LinkedFactIDs, &row.EmotionalValence, &row.EmotionalIntensity, &row.Domain, &row.Summary); err != nil {
			return nil, fmt.Errorf("扫描 temporal_anchors 失败: %w", err)
		}
		out = append(out, row.toAnchor())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 temporal_anchors 失败: %w", err)
	}
	return out, nil
}

// SaveAll 全量 upsert（id 主键，事务内完成；ID 或日期为空的行跳过——两列均 NOT NULL）
func (r *TemporalAnchorsRepo) SaveAll(items []whisper.TemporalAnchor) error {
	tx, err := r.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	for _, a := range items {
		if a.ID == "" || a.AnchorDate == "" {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO temporal_anchors(`+anchorCols+`)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   anchor_date = excluded.anchor_date,
			   anchor_type = excluded.anchor_type,
			   recurrence_rule = excluded.recurrence_rule,
			   linked_fact_ids = excluded.linked_fact_ids,
			   emotional_valence = excluded.emotional_valence,
			   emotional_intensity = excluded.emotional_intensity,
			   domain = excluded.domain,
			   summary = excluded.summary`,
			anchorArgs(a)...,
		); err != nil {
			return fmt.Errorf("upsert temporal_anchors 失败: %w", err)
		}
	}
	return tx.Commit()
}

// DeleteExpired 清理 anchor_date 早于 before 的锚点，返回删除条数。
// 表无到期时间列，以锚点日期（ISO "YYYY-MM-DD" 字符串比较）作为时间基准；
// 语义为「早于截止日期的历史锚点视为过期」，由调用方显式触发（如启动清理）。
func (r *TemporalAnchorsRepo) DeleteExpired(before string) (int, error) {
	res, err := r.sqlDB.Exec("DELETE FROM temporal_anchors WHERE anchor_date < ?", before)
	if err != nil {
		return 0, fmt.Errorf("清理过期时间锚点失败: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Clear 清空整表
func (r *TemporalAnchorsRepo) Clear() error {
	if _, err := r.sqlDB.Exec("DELETE FROM temporal_anchors"); err != nil {
		return fmt.Errorf("清空 temporal_anchors 失败: %w", err)
	}
	return nil
}

// anchorArgs 组装 temporal_anchors 行参数（可空列转 NULL 兼容）
func anchorArgs(a whisper.TemporalAnchor) []interface{} {
	var recurrenceRule, linkedFactIDs, domain interface{}
	if a.RecurrenceRule != "" {
		recurrenceRule = a.RecurrenceRule
	}
	if len(a.LinkedFactIDs) > 0 {
		b, _ := json.Marshal(a.LinkedFactIDs)
		linkedFactIDs = string(b)
	}
	if a.Domain != "" {
		domain = a.Domain
	}
	return []interface{}{
		a.ID, a.AnchorDate, string(a.AnchorType), recurrenceRule, linkedFactIDs,
		a.EmotionalValence, a.EmotionalIntensity, domain, a.Summary,
	}
}
