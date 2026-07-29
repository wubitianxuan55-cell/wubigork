// Package whisper — 记忆绑定桥（100% 对齐 ackem memory/memoryBinding.ts）
package whisper

import "math"

// ─── 情感上下文快照 ────────────────────────────────────────────

// CaptureEmotionalContext 捕获当前情感上下文
func CaptureEmotionalContext(l1 L1State, l2 EmotionState) EmotionalContext {
	return EmotionalContext{
		Valence:    clampF(l2.Aff/100, -1, 1),
		Intensity:  math.Min(1, (math.Abs(l2.Aff)+math.Abs(l2.Sec))/200),
		RelStage:   l1.Stage,
		Trust:      l1.Trust,
		Atmosphere: l1.Atmosphere,
	}
}

// ─── L1 记忆增强 ──────────────────────────────────────────────

// AugmentL1FromMemory 用记忆事实增强 L1 sharedEventsCount
func AugmentL1FromMemory(l1 L1State, fs *FactStore) MemoryAugmentedL1 {
	n := fs.CountSharedBondFacts()
	return MemoryAugmentedL1{
		SharedEventsCount: maxInt(l1.SharedEventsCount, n),
	}
}

// ─── 记忆调整后的有效信任 ─────────────────────────────────────

// EffectiveTrustForL0 计算记忆调整后的有效信任值（用于 L0 分类）
func EffectiveTrustForL0(l1 L1State, fs *FactStore) float64 {
	memoir := fs.ComputeMemoirTrust()
	m := l1.Trust
	if memoir != nil {
		m = *memoir
	}
	return l1.Trust*EffectiveTrustL1Weight + math.Min(l1.Trust, m)*EffectiveTrustMemWeight
}

// ─── FactStore 补充方法 ───────────────────────────────────────

// CountSharedBondFacts 统计 shared_bond 领域的事实数
func (fs *FactStore) CountSharedBondFacts() int {
	count := 0
	for _, f := range fs.facts {
		if f.Status == "active" && f.Domain == "shared_bond" {
			count++
		}
	}
	return count
}

// ComputeMemoirTrust 基于记忆事实计算信任基线
func (fs *FactStore) ComputeMemoirTrust() *float64 {
	var totalWeight float64
	var weightedTrust float64
	for _, f := range fs.facts {
		if f.Status != "active" || f.Confidence < MinConfidenceForInjection {
			continue
		}
		// 高置信度 + 高自相关的事实贡献信任
		w := f.Weight * f.SelfRelevance * f.Confidence
		totalWeight += w
		// 正向事实贡献信任，负向事实降低信任
		if f.Confidence > 0.6 {
			weightedTrust += w * 60 // 基线偏高
		} else if f.Confidence > 0.4 {
			weightedTrust += w * 45
		} else {
			weightedTrust += w * 30
		}
	}
	if totalWeight <= 0 {
		return nil
	}
	trust := weightedTrust / totalWeight
	return &trust
}
