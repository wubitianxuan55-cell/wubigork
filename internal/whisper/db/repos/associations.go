// Package repos — memory_associations 记忆关联仓库（v4.3a 会客厅·记忆持久化闭环）
// 100% 对齐 ackem src/main/db/repos/associations.ts（表结构见 schema_v4.go）
package repos

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// AssociationsRepo 记忆关联访问层（表 memory_associations）。
//
// 表无业务唯一键（id 为 INTEGER 自增行号，不承载业务 ID），因此 SaveAll 采用
// 「二元组 UPSERT」：先删与 items 同 (fact_id_a, fact_id_b)（含反向）的旧行，
// 再插入 items —— 其他会话的边不受影响（不做整表清空）。
//
// 外键护栏：memory_associations 有 FOREIGN KEY 指向 memory_facts(id)，且
// hermes.db 以 _foreign_keys=ON 打开。同一持久化周期内"本轮新事实"尚未落库
// （app 层先存 companion_state 再存 facts），直接插入会触发外键违例；这里在
// 写入前预检两端事实是否已在库中，缺失的边本轮跳过、下轮持久化自然补上
// （事实已落库），重启后亦由 ReseedAssociationGraph 从事实重建兜底。
type AssociationsRepo struct {
	sqlDB *sql.DB
}

// OpenAssociationsRepo 以 dataRoot 打开关联仓库（复用 hermes.db 单例连接）
func OpenAssociationsRepo(dataRoot string) (*AssociationsRepo, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}
	return &AssociationsRepo{sqlDB: sqlDB}, nil
}

// NewAssociationsRepo 以既有连接构造关联仓库（供组合复用）
func NewAssociationsRepo(sqlDB *sql.DB) *AssociationsRepo {
	return &AssociationsRepo{sqlDB: sqlDB}
}

// List 读取全部关联边。
// 注意：表的 id 为自增行号，不还原为业务 ID（内存 Association.ID 是会话内
// 不透明标识，由 AssociationIndex.Add 在恢复时重新生成）。
func (r *AssociationsRepo) List() ([]whisper.Association, error) {
	rows, err := r.sqlDB.Query(
		"SELECT fact_id_a, fact_id_b, association_type, strength, last_activated_at FROM memory_associations",
	)
	if err != nil {
		return nil, fmt.Errorf("查询 memory_associations 失败: %w", err)
	}
	defer rows.Close()

	var out []whisper.Association
	for rows.Next() {
		var a whisper.Association
		var lastActivated sql.NullString
		if err := rows.Scan(&a.FactIDA, &a.FactIDB, &a.AssociationType, &a.Strength, &lastActivated); err != nil {
			return nil, fmt.Errorf("扫描 memory_associations 失败: %w", err)
		}
		if lastActivated.Valid && lastActivated.String != "" {
			if t, perr := time.Parse(time.RFC3339, lastActivated.String); perr == nil {
				a.LastActivatedAt = t.UnixMilli()
			}
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 memory_associations 失败: %w", err)
	}
	return out, nil
}

// SaveAll 按二元组 UPSERT 全量写入（外键预检 + 先删后插，事务内完成）。
func (r *AssociationsRepo) SaveAll(items []whisper.Association) error {
	tx, err := r.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 外键预检：收集当前库中全部事实 ID（同一周期内 app 先存状态后存事实，
	// 本轮新事实尚未落库，其关联边需跳过，下轮补写）。
	factIDs := make(map[string]bool)
	rows, err := tx.Query("SELECT id FROM memory_facts")
	if err != nil {
		return fmt.Errorf("查询 memory_facts 失败: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil && id != "" {
			factIDs[id] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历 memory_facts 失败: %w", err)
	}

	for _, a := range items {
		if !factIDs[a.FactIDA] || !factIDs[a.FactIDB] {
			continue // 两端事实未全部落库，本轮跳过（外键护栏）
		}
		// 双向删除旧行（内存索引按无序二元组去重，库中可能残留反向旧行）
		if _, err := tx.Exec(
			"DELETE FROM memory_associations WHERE (fact_id_a = ? AND fact_id_b = ?) OR (fact_id_a = ? AND fact_id_b = ?)",
			a.FactIDA, a.FactIDB, a.FactIDB, a.FactIDA,
		); err != nil {
			return fmt.Errorf("删除旧关联失败: %w", err)
		}
	}

	for _, a := range items {
		if !factIDs[a.FactIDA] || !factIDs[a.FactIDB] {
			continue
		}
		var lastActivated interface{}
		if a.LastActivatedAt > 0 {
			lastActivated = time.UnixMilli(a.LastActivatedAt).UTC().Format(time.RFC3339)
		}
		if _, err := tx.Exec(
			"INSERT INTO memory_associations(fact_id_a, fact_id_b, association_type, strength, last_activated_at) VALUES (?, ?, ?, ?, ?)",
			a.FactIDA, a.FactIDB, a.AssociationType, a.Strength, lastActivated,
		); err != nil {
			return fmt.Errorf("插入关联失败: %w", err)
		}
	}
	return tx.Commit()
}

// Clear 清空整表
func (r *AssociationsRepo) Clear() error {
	if _, err := r.sqlDB.Exec("DELETE FROM memory_associations"); err != nil {
		return fmt.Errorf("清空 memory_associations 失败: %w", err)
	}
	return nil
}
