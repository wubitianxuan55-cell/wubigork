// Package whisper — profiling_dimension.go
// 100% 对齐 ackem engine/user-dimension-inferrer.ts
// 用户六维推断：从对话行为推断 T/I/S/O/R

package whisper

// ─── SixDimensions ────────────────────────────────────────────

// SixDimensions 用户六维画像
type SixDimensions struct {
	T float64 // Tenderness 温柔度
	I float64 // Initiative 主动性
	S float64 // Submission 顺从度
	O float64 // Originality 独特度
	R float64 // Reserve 矜持度
}

// SixDimensionsResult 推断结果
type SixDimensionsResult struct {
	Dims       SixDimensions
	Confidence float64
}

// ─── Dimension Inferrer ───────────────────────────────────────

// InferSixDimensions 从事件类型推断六维
func InferSixDimensions(
	eventTypes []string,
	totalTurns int,
	profile *UserProfile,
) SixDimensionsResult {
	if len(eventTypes) == 0 || totalTurns < 5 {
		return SixDimensionsResult{
			Dims:       SixDimensions{T: 50, I: 50, S: 50, O: 50, R: 50},
			Confidence: 0.1,
		}
	}

	dims := SixDimensions{T: 50, I: 50, S: 50, O: 50, R: 50}

	for _, et := range eventTypes {
		switch et {
		case "praise":
			dims.T += 0.5
			dims.I += 0.3
		case "vulnerable":
			dims.T += 0.3
			dims.S += 0.2
			dims.R -= 0.3
		case "tease":
			dims.O += 0.4
			dims.I += 0.2
		case "question":
			dims.I += 0.3
		case "cold":
			dims.R += 0.5
			dims.T -= 0.2
		case "hurtful":
			dims.R += 0.8
			dims.T -= 0.5
		case "apology":
			dims.S += 0.3
			dims.T += 0.2
		}
	}

	// 从用户画像修正
	if profile != nil {
		dims.T += profile.EmotionalNeediness * 10
		dims.S += profile.DominancePreference * (-5)
	}

	// 钳制到 0-100
	dims.T = clampF(dims.T, 0, 100)
	dims.I = clampF(dims.I, 0, 100)
	dims.S = clampF(dims.S, 0, 100)
	dims.O = clampF(dims.O, 0, 100)
	dims.R = clampF(dims.R, 0, 100)

	confidence := clampF(float64(totalTurns)/50.0, 0.1, 0.9)

	return SixDimensionsResult{Dims: dims, Confidence: confidence}
}

// SixDimensionsToHint 六维→回复提示
func SixDimensionsToHint(dims SixDimensions) string {
	var hints []string
	if dims.T > 70 {
		hints = append(hints, "ta很温柔，你可以放心地表达感情。")
	}
	if dims.I > 70 {
		hints = append(hints, "ta很主动，跟上ta的节奏。")
	} else if dims.I < 30 {
		hints = append(hints, "ta比较被动，你可以多主动一些。")
	}
	if dims.R > 70 {
		hints = append(hints, "ta比较矜持，不要逼ta说太多。")
	}
	if dims.O > 70 {
		hints = append(hints, "ta很有个性，欣赏ta的独特之处。")
	}
	if len(hints) == 0 {
		return ""
	}
	result := "【关于ta】"
	for _, h := range hints {
		result += "\n" + h
	}
	return result
}
