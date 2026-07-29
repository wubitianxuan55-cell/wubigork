// Package whisper — memory_assoc.go
// 100% 对齐 ackem memory/associationIndex.ts
// 记忆关联索引：O(1) 查询、激活追踪、衰减

package whisper

import "time"

// ─── AssociationIndex ─────────────────────────────────────────

// AssociationIndex 记忆关联索引
type AssociationIndex struct {
	assocs   []Association
	byFactID map[string][]int // factID → assoc indices
}

// NewAssociationIndex 创建空关联索引
func NewAssociationIndex() *AssociationIndex {
	return &AssociationIndex{byFactID: make(map[string][]int)}
}

// Add 添加关联边（自动去重）
func (ai *AssociationIndex) Add(a Association) {
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

// GetAssociations O(1) 查询某事实的所有关联
func (ai *AssociationIndex) GetAssociations(factID string) []Association {
	indices := ai.byFactID[factID]
	result := make([]Association, 0, len(indices))
	for _, idx := range indices {
		result = append(result, ai.assocs[idx])
	}
	return result
}

// StrengthenOrCreate 加强或创建关联
func (ai *AssociationIndex) StrengthenOrCreate(factIDA, factIDB, assocType string, strength float64) {
	// 查找已有关联
	for _, idx := range ai.byFactID[factIDA] {
		existing := &ai.assocs[idx]
		if existing.FactIDB == factIDB || existing.FactIDA == factIDB {
			existing.Strength = clampF(existing.Strength+strength, 0, 1)
			existing.LastActivatedAt = time.Now().UnixMilli()
			return
		}
	}
	// 新建
	ai.Add(Association{
		FactIDA:         factIDA,
		FactIDB:         factIDB,
		AssociationType: assocType,
		Strength:        strength,
	})
}

// RecordActivation 标记关联被激活
func (ai *AssociationIndex) RecordActivation(assocID string) {
	nowMs := time.Now().UnixMilli()
	for i := range ai.assocs {
		if ai.assocs[i].ID == assocID {
			ai.assocs[i].LastActivatedAt = nowMs
			return
		}
	}
}

// DecayEdges 衰减低于阈值的关联边
func (ai *AssociationIndex) DecayEdges(minStrength float64) {
	nowMs := time.Now().UnixMilli()
	// 30 天未激活的边衰减 50%
	threshold := int64(30 * 24 * 3600 * 1000)
	var kept []Association
	newByFactID := make(map[string][]int)

	for _, a := range ai.assocs {
		if nowMs-a.LastActivatedAt > threshold {
			a.Strength *= 0.5
		}
		if a.Strength >= minStrength {
			newIdx := len(kept)
			kept = append(kept, a)
			newByFactID[a.FactIDA] = append(newByFactID[a.FactIDA], newIdx)
			newByFactID[a.FactIDB] = append(newByFactID[a.FactIDB], newIdx)
		}
	}
	ai.assocs = kept
	ai.byFactID = newByFactID
}

// Remove 移除关联
func (ai *AssociationIndex) Remove(assocID string) {
	for i, a := range ai.assocs {
		if a.ID == assocID {
			ai.assocs = append(ai.assocs[:i], ai.assocs[i+1:]...)
			// 重建索引
			ai.rebuildIndex()
			return
		}
	}
}

func (ai *AssociationIndex) rebuildIndex() {
	ai.byFactID = make(map[string][]int)
	for i, a := range ai.assocs {
		ai.byFactID[a.FactIDA] = append(ai.byFactID[a.FactIDA], i)
		ai.byFactID[a.FactIDB] = append(ai.byFactID[a.FactIDB], i)
	}
}

// Size 关联数量
func (ai *AssociationIndex) Size() int {
	return len(ai.assocs)
}
