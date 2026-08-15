// Package whisper — memory_assoc.go
// 100% 对齐 ackem memory/associationIndex.ts
// 记忆关联索引：O(1) 查询、激活追踪、衰减

package whisper

import (
	"sync"
	"time"
)

// ─── AssociationIndex ─────────────────────────────────────────

// AssociationIndex 记忆关联索引。T7-1.1：异步记忆写入协程与主流程
// 并发读写，必须加锁（并发 map 读写直接 panic）。
type AssociationIndex struct {
	mu               sync.RWMutex
	assocs           []Association
	byFactID         map[string][]int // factID → assoc indices
	lastActivatedIDs []string         // 本轮激活的关联 ID
}

// NewAssociationIndex 创建空关联索引
func NewAssociationIndex() *AssociationIndex {
	return &AssociationIndex{byFactID: make(map[string][]int)}
}

// Add 添加关联边（自动去重）
func (ai *AssociationIndex) Add(a Association) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.addLocked(a)
}

func (ai *AssociationIndex) addLocked(a Association) {
	if a.ID == "" {
		a.ID = genHexID()
	}
	nowMs := time.Now().UnixMilli()
	if a.CreatedAt == 0 {
		a.CreatedAt = nowMs
	}
	if a.LastActivatedAt == 0 {
		a.LastActivatedAt = nowMs
	}

	// 去重：同 factIdA+factIdB
	for _, idx := range ai.byFactID[a.FactIDA] {
		existing := &ai.assocs[idx]
		if existing.FactIDB == a.FactIDB || existing.FactIDA == a.FactIDB {
			// 加强已有关联
			if a.Strength > existing.Strength {
				existing.Strength = a.Strength
			}
			existing.LastActivatedAt = nowMs
			return
		}
	}

	idx := len(ai.assocs)
	ai.assocs = append(ai.assocs, a)
	ai.byFactID[a.FactIDA] = append(ai.byFactID[a.FactIDA], idx)
	ai.byFactID[a.FactIDB] = append(ai.byFactID[a.FactIDB], idx)
}

// GetAssociations O(1) 查询某事实的所有关联（拷贝）。
func (ai *AssociationIndex) GetAssociations(factID string) []Association {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	indices := ai.byFactID[factID]
	result := make([]Association, 0, len(indices))
	for _, idx := range indices {
		result = append(result, ai.assocs[idx])
	}
	return result
}

// StrengthenOrCreate 加强或创建关联
func (ai *AssociationIndex) StrengthenOrCreate(factIDA, factIDB, assocType string, strength float64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	// 查找已有关联
	for _, idx := range ai.byFactID[factIDA] {
		existing := &ai.assocs[idx]
		if existing.FactIDB == factIDB || existing.FactIDA == factIDB {
			existing.Strength = clampF(1-(1-existing.Strength)*0.95, 0, 1)
			existing.LastActivatedAt = time.Now().UnixMilli()
			return
		}
	}
	// 新建
	ai.addLocked(Association{
		FactIDA:         factIDA,
		FactIDB:         factIDB,
		AssociationType: assocType,
		Strength:        strength,
	})
}

// RecordActivation 标记关联被激活（保留本轮 ID 供 post-turn 纠正）
func (ai *AssociationIndex) RecordActivation(assocID string) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	nowMs := time.Now().UnixMilli()
	for i := range ai.assocs {
		if ai.assocs[i].ID == assocID {
			ai.assocs[i].LastActivatedAt = nowMs
			ai.lastActivatedIDs = append(ai.lastActivatedIDs, assocID)
			return
		}
	}
}

// GetLastActivated 返回本轮被激活的关联 ID 列表（供 post-turn 纠正，拷贝）
func (ai *AssociationIndex) GetLastActivated() []string {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	out := make([]string, len(ai.lastActivatedIDs))
	copy(out, ai.lastActivatedIDs)
	return out
}

// ClearActivated 清空本轮激活记录（每轮对话开始时调用）
func (ai *AssociationIndex) ClearActivated() {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.lastActivatedIDs = nil
}

// Weaken 削弱关联强度（隐式纠正 cold/hurtful 或显式纠正用户指出错误）
func (ai *AssociationIndex) Weaken(assocID string, factor float64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	for i := range ai.assocs {
		if ai.assocs[i].ID == assocID {
			ai.assocs[i].Strength = clampF(ai.assocs[i].Strength*factor, 0, 1)
			return
		}
	}
}

// WeakenByFactID 按事实ID削弱所有关联（对齐 ackem weakenByFactId）
func (ai *AssociationIndex) WeakenByFactID(factID string, factor float64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	for _, idx := range ai.byFactID[factID] {
		ai.assocs[idx].Strength = clampF(ai.assocs[idx].Strength*factor, 0, 1)
	}
}

// ListAll 返回所有关联（对齐 ackem listAll，拷贝）
func (ai *AssociationIndex) ListAll() []Association {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	result := make([]Association, len(ai.assocs))
	copy(result, ai.assocs)
	return result
}

// GetByID 按ID获取关联（对齐 ackem getById，返回拷贝）
func (ai *AssociationIndex) GetByID(id string) *Association {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	for i := range ai.assocs {
		if ai.assocs[i].ID == id {
			v := ai.assocs[i]
			return &v
		}
	}
	return nil
}

// Count 返回关联总数
func (ai *AssociationIndex) Count() int {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return len(ai.assocs)
}
