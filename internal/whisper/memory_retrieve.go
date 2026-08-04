// Package whisper — memory_retrieve.go
// 记忆回声聚合：从检索结果计算记忆情绪回声（对齐 ackem computeMemoryEcho）
// 多路检索主流程在 orchestrator.buildTierBBlock 单轨实现

package whisper

import (
	"math"
	"time"
)

// sfPair 事实+得分对
type sfPair struct {
	f *Fact
	s float64
}

// ─── ComputeMemoryEchoFacts ───────────────────────────────────

// ComputeMemoryEchoFacts 从检索结果计算记忆回声（对齐 ackem computeMemoryEcho）
func ComputeMemoryEchoFacts(ranked []sfPair, aff float64) MemoryEcho {
	if len(ranked) == 0 {
		return MemoryEcho{}
	}
	var sw, affSum, secSum, aroSum, domSum float64
	top := ranked
	if len(top) > 5 {
		top = top[:5]
	}
	for _, sf := range top {
		w := sf.f.Weight * sf.f.SelfRelevance
		if sf.f.EmotionalContext != nil {
			// 有情感上下文时使用真实值
			ec := sf.f.EmotionalContext
			intensity := ec.Intensity
			trust := ec.Trust
			decay := math.Exp(-0.003 * time.Since(sf.f.CreatedAt).Hours() / 24)
			w *= intensity * decay
			if w <= 0 {
				continue
			}
			sw += w
			affSum += ec.Valence * w * MemoryEchoAffWeight
			if trust > 50 {
				secSum += MemoryEchoSecPositive * w
			} else {
				secSum += MemoryEchoSecNegative * w
			}
			aroSum += intensity * w * MemoryEchoAroIntensityWeight
			// dom 基于 trust 信号
			domSignal := (trust - 50) / 50 * MemoryEchoDomTrustWeight
			domSum += domSignal * w
		} else {
			// 回退：无情感上下文时使用简单估算
			w2 := w * 0.3
			if w2 <= 0 {
				continue
			}
			sw += w2
			affSum += 1.0 * w2 * MemoryEchoAffWeight
			if sf.f.Confidence > 0.5 {
				secSum += MemoryEchoSecPositive * w2
			} else {
				secSum += MemoryEchoSecNegative * w2
			}
			aroSum += 1.0 * w2 * MemoryEchoAroIntensityWeight
			domSum += 0.1 * w2 * MemoryEchoDomTrustWeight
		}
	}
	if sw <= 0 {
		return MemoryEcho{}
	}
	return MemoryEcho{
		Aff: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, affSum/sw)),
		Sec: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, secSum/sw)),
		Aro: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, aroSum/sw)),
		Dom: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, domSum/sw)),
	}
}
