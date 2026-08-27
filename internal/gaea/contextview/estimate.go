package contextview

// fallbackTokPerChar 与 agent 包的口径一致：约 4 字符/token。
const fallbackTokPerChar = 0.25

// estimateTokens 按仓库既有口径估算文本 token 数（字节 × 0.25）。
func estimateTokens(s string) int64 {
	return int64(float64(len(s)) * fallbackTokPerChar)
}

// scaleCategory 把估算分类按实际 promptTokens 等比缩放（锚定真实用量，
// 与顶栏 ContextBar 口径同源）。est<=0 时原样返回。
func scaleCategory(c Category, actual, est int64) Category {
	if est <= 0 || actual <= 0 {
		return c
	}
	f := float64(actual) / float64(est)
	if f < 0.25 {
		f = 0.25
	}
	if f > 4 {
		f = 4
	}
	return Category{
		System:    int64(float64(c.System) * f),
		Tools:     int64(float64(c.Tools) * f),
		User:      int64(float64(c.User) * f),
		Inject:    int64(float64(c.Inject) * f),
		Assistant: int64(float64(c.Assistant) * f),
		Tool:      int64(float64(c.Tool) * f),
	}
}

// briefOf 截断为单行预览（≤ maxBriefLen runes）。
func briefOf(s string, max int) string {
	if s == "" {
		return ""
	}
	one := s
	if i := indexByte(one, '\n'); i >= 0 {
		one = one[:i]
	}
	r := []rune(one)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return one
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
