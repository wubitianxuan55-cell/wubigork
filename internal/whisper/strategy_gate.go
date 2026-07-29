// Package whisper — strategy_gate.go
// 100% 对齐 ackem extensions/policy/proactiveGate.ts
// 主动发言门控：aff 历史追踪、波动性计算、基础门控决策

package whisper

import "math"

const (
	affVolatilityWindow   = 10
	affVolatileThreshold  = 20
	affPositiveThreshold  = 60
	affLowThreshold       = 30
	riftsSilentThreshold  = 2
)

// ─── Aff 历史 ─────────────────────────────────────────────────

var affHistoryWindow []float64

// PushAffToHistory 推送最新 aff 值到滑动窗口
func PushAffToHistory(aff float64) {
	affHistoryWindow = append(affHistoryWindow, aff)
	if len(affHistoryWindow) > affVolatilityWindow {
		affHistoryWindow = affHistoryWindow[len(affHistoryWindow)-affVolatilityWindow:]
	}
}

// GetAffHistory 获取 aff 历史快照
func GetAffHistory() []float64 {
	result := make([]float64, len(affHistoryWindow))
	copy(result, affHistoryWindow)
	return result
}

// ResetAffHistory 重置 aff 历史
func ResetAffHistory() {
	affHistoryWindow = nil
}

// ComputeAffVolatility 计算 aff 波动性（标准差）
func ComputeAffVolatility() float64 {
	if len(affHistoryWindow) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range affHistoryWindow {
		mean += v
	}
	mean /= float64(len(affHistoryWindow))
	variance := 0.0
	for _, v := range affHistoryWindow {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(affHistoryWindow))
	return math.Sqrt(variance)
}

// ─── 门控决策 ────────────────────────────────────────────────

// ProactiveGateResult 门控决策结果
type ProactiveGateResult struct {
	Level        string  // silent/whisper/normal/proactive
	Confidence   float64 // 0-1
	ReasonCode   string  // 决策原因代码
}

// EvaluateProactiveGate 评估主动发言门控
func EvaluateProactiveGate(
	aff float64,
	aro float64,
	sec float64,
	trust float64,
	rifts int,
	stage RelationshipStage,
	timeOfDay string,
	adultMode bool,
) ProactiveGateResult {
	result := ProactiveGateResult{Level: "normal", Confidence: 0.5}

	// 裂痕过多 → 沉默
	if rifts >= riftsSilentThreshold {
		result.Level = "silent"
		result.Confidence = 0.9
		result.ReasonCode = "rifts"
		return result
	}

	// 情绪剧烈波动 → 倾听
	vol := ComputeAffVolatility()
	if vol > affVolatileThreshold {
		result.Level = "whisper"
		result.Confidence = 0.7
		result.ReasonCode = "volatile"
		return result
	}

	// 高亲和 + 高安全 → 主动
	if aff > affPositiveThreshold && sec > 50 && stage == StageIntimate {
		result.Level = "proactive"
		result.Confidence = 0.8
		result.ReasonCode = "high_aff_sec"
		return result
	}

	// 低亲和 → 沉默
	if aff < affLowThreshold {
		result.Level = "silent"
		result.Confidence = 0.6
		result.ReasonCode = "low_aff"
		return result
	}

	// 深夜 → 轻声
	if timeOfDay == "late_night" && !adultMode {
		result.Level = "whisper"
		result.Confidence = 0.7
		result.ReasonCode = "late_night"
		return result
	}

	// 高唤醒 + 信任高 → 偏主动
	if math.Abs(aro) > 50 && trust > 60 {
		result.Level = "proactive"
		result.Confidence = 0.65
		result.ReasonCode = "high_aro_trust"
	}

	return result
}
