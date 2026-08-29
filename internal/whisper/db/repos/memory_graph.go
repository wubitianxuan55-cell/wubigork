// Package repos — memory_graph 关系记忆持久化装配（v4.3a 会客厅·记忆持久化闭环）
//
// 把内存态关联/习惯/时间锚点接入 memory_associations / user_habits / temporal_anchors
// 三表，供 companion_state 的保存/恢复路径调用。
//
// 为什么放在本包而不是 whisper 主包：whisper 主包不直接依赖 db/repos
// （repos 反向依赖 whisper 类型，直接引用会形成循环）。内存态 ↔ FullState 的
// 双向同步（syncMemoryGraphToState / restoreMemoryGraphFromState）在 whisper 侧
// （memory_graph_persist.go），落库/读库装配在这里，由 app 层既有的
// SaveCompanionStateToDB / LoadCompanionStateFromDB 调用链自动触发——
// 无需改动 app 层即可闭环。
package repos

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// saveMemoryGraphFromState 把状态快照中的关联/习惯/锚点写入三表。
// 任一表失败即记录并累计错误（不影响其他表与 companion_state 本身）。
// 空（nil）切片表示"该域无数据"：跳过对应表，避免清空其他会话的数据。
func saveMemoryGraphFromState(sqlDB *sql.DB, state *whisper.FullState) error {
	if sqlDB == nil || state == nil {
		return nil
	}
	var errs []error
	if state.Associations != nil {
		if err := NewAssociationsRepo(sqlDB).SaveAll(state.Associations); err != nil {
			slog.Error("whisper-repos: memory_associations 落库失败", "error", err)
			errs = append(errs, err)
		}
	}
	if state.Habits != nil {
		if err := NewHabitsRepo(sqlDB).SaveAll(state.Habits); err != nil {
			slog.Error("whisper-repos: user_habits 落库失败", "error", err)
			errs = append(errs, err)
		}
	}
	if state.TemporalAnchors != nil {
		if err := NewTemporalAnchorsRepo(sqlDB).SaveAll(state.TemporalAnchors); err != nil {
			slog.Error("whisper-repos: temporal_anchors 落库失败", "error", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("关系记忆三表落库失败: %w", errors.Join(errs...))
	}
	return nil
}

// loadMemoryGraphIntoState 从三表读回关系记忆并回填状态。
// fail-open：表读取失败仅记日志（保留 state_json 内嵌字段作回退），
// 不阻断状态恢复；表非空时以表为准（表是规范存储，JSON 是兼容回退）。
func loadMemoryGraphIntoState(sqlDB *sql.DB, state *whisper.FullState) {
	if sqlDB == nil || state == nil {
		return
	}
	if assocs, err := NewAssociationsRepo(sqlDB).List(); err == nil {
		if len(assocs) > 0 {
			state.Associations = assocs
		}
	} else {
		slog.Warn("whisper-repos: 读取 memory_associations 失败，回退 state_json", "error", err)
	}
	if habits, err := NewHabitsRepo(sqlDB).List(); err == nil {
		if len(habits) > 0 {
			state.Habits = habits
		}
	} else {
		slog.Warn("whisper-repos: 读取 user_habits 失败，回退 state_json", "error", err)
	}
	if anchors, err := NewTemporalAnchorsRepo(sqlDB).List(); err == nil {
		if len(anchors) > 0 {
			state.TemporalAnchors = anchors
		}
	} else {
		slog.Warn("whisper-repos: 读取 temporal_anchors 失败，回退 state_json", "error", err)
	}
}

// PersistMemoryGraphToDB 独立装配入口：以 dataRoot 直接落库三表
// （供测试或后续步骤显式调用；常规路径由 SaveCompanionStateToDB 触发）。
func PersistMemoryGraphToDB(dataRoot string, assocs []whisper.Association, habits []whisper.UserHabit, anchors []whisper.TemporalAnchor) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	return saveMemoryGraphFromState(sqlDB, &whisper.FullState{
		Associations:    assocs,
		Habits:          habits,
		TemporalAnchors: anchors,
	})
}

// RestoreMemoryGraphFromDB 独立装配入口：以 dataRoot 直接读回三表
// （常规路径由 LoadCompanionStateFromDB 触发，二者语义一致：fail-open）。
func RestoreMemoryGraphFromDB(dataRoot string) ([]whisper.Association, []whisper.UserHabit, []whisper.TemporalAnchor, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, nil, nil, fmt.Errorf("数据库不可用: %w", openErr)
	}
	var state whisper.FullState
	loadMemoryGraphIntoState(sqlDB, &state)
	return state.Associations, state.Habits, state.TemporalAnchors, nil
}
