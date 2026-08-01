// Package whisper — wave_messages.go
// 100% 对齐 ackem chat/buildWaveMessages.ts
// 波次消息组装：禁复读块 + 已发送感知 + 长度约束

package whisper

import "fmt"

// WaveSpec 波次规格
type WaveSpec struct {
	WaveIndex   int
	MaxChars    int
	SystemDelta string
}

// BuildAntiRepeatBlock 构建禁复读说明块
func BuildAntiRepeatBlock(waveIndex int) string {
	if waveIndex == 0 {
		return ""
	}
	return "【禁复读】\n" +
		"本轮是多气泡并行生成，其他气泡可能已应答用户。\n" +
		"禁止重复相同语义，禁止换说法再说一遍。\n" +
		"禁止再用：在、在呢、在的、我在、嗯我在 等在线确认。"
}

// BuildPriorAwareBlock 构建已发送感知块
func BuildPriorAwareBlock(priorParts []string, waveIndex int) string {
	if waveIndex == 0 || len(priorParts) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, "【已发送】")
	for i, p := range priorParts {
		if p != "" {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, p))
		}
	}
	lines = append(lines, "不得与上述矛盾；不得把已决定的事改口成疑问；只补充一个新细节或情绪。")

	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

// BuildWaveLengthHint 构建波次长度提示
func BuildWaveLengthHint(maxChars int) string {
	if maxChars <= 0 {
		return "\n【长度】只能有1句。"
	}
	return fmt.Sprintf("\n【长度】本条回复不超过 %d 字，且只能有1句。", maxChars)
}

// BuildWaveSystemDelta 构建波次系统增量
func BuildWaveSystemDelta(wave WaveSpec, enrichedTierB string) string {
	var parts []string

	if wave.SystemDelta != "" {
		parts = append(parts, wave.SystemDelta)
	}
	if wave.WaveIndex >= 1 {
		parts = append(parts, "【注意】这是第"+itoa(wave.WaveIndex+1)+"波，不要重复前一波的内容。")
	}
	if wave.WaveIndex >= 2 && enrichedTierB != "" {
		excerpt := enrichedTierB
		if len([]rune(excerpt)) > 400 {
			excerpt = string([]rune(excerpt)[:400]) + "\n…"
		}
		parts = append(parts, "【相关记忆摘录】\n"+excerpt)
	}

	if len(parts) == 0 {
		return ""
	}
	result := ""
	for _, p := range parts {
		result += p + "\n"
	}
	return result
}
