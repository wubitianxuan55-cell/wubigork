// Package whisper — memory_binding.go
// 100% 对齐 ackem memory/memoryBinding.ts
// L4↔L1/L2 桥梁：情感上下文快照 + effectiveTrust + 记忆增强

package whisper

import "math"

// CaptureEmotionalContext 从 L1/L2 捕获情感上下文
func CaptureEmotionalContext(l1 L1State, l2 EmotionState) EmotionalContext {
	return EmotionalContext{
		Valence:    clamp(l2.Aff/100, -1, 1),
		Intensity:  math.Min(1, (math.Abs(l2.Aff)+math.Abs(l2.Sec))/200),
		RelStage:   l1.Stage,
		Trust:      l1.Trust,
		Atmosphere: l1.Atmosphere,
	}
}

// AugmentL1FromMemory 从 FactStore 增强 L1 状态
func AugmentL1FromMemory(l1 L1State, fs *FactStore) MemoryAugmentedL1 {
	n := 0
	if fs != nil {
		n = fs.CountSharedBondFacts()
	}
	sec := l1.SharedEventsCount
	if n > sec {
		sec = n
	}
	return MemoryAugmentedL1{SharedEventsCount: sec}
}

const effectiveTrustL1Weight = 0.6
const effectiveTrustMemWeight = 0.4

// EffectiveTrustForL0 计算用于 L0 的有效信任值
func EffectiveTrustForL0(l1 L1State, fs *FactStore) float64 {
	mem := 0.0
	if fs != nil {
		mem = fs.ComputeMemoirTrust()
	}
	if mem == 0 {
		mem = l1.Trust
	}
	return l1.Trust*effectiveTrustL1Weight + math.Min(l1.Trust, mem)*effectiveTrustMemWeight
}

// clamp 限制值在 [lo, hi] 范围内
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}