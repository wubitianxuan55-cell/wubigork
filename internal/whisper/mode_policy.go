// Package whisper — mode_policy.go
// 100% 对齐 ackem desktop-agent/modePolicy.ts
// 桌面助手模式策略：抑制联网搜索、技能工具门控

package whisper

// ShouldSuppressWebSearch 桌面助手模式下抑制联网搜索
func ShouldSuppressWebSearch(desktopAgentActive bool) bool {
	return desktopAgentActive
}

// ShouldOfferSkillTools 桌面助手模式下是否提供技能工具
func ShouldOfferSkillTools(desktopAgentActive bool) bool {
	return !desktopAgentActive
}

// WorkIntentResult 工作意图结果
type WorkIntentResult struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Proactive  bool    `json:"proactive"`
}

// ApplyDesktopAgentModeToWorkIntent 桌面助手模式下修正工作意图
func ApplyDesktopAgentModeToWorkIntent(workIntent WorkIntentResult, sessionActive bool) WorkIntentResult {
	if !sessionActive {
		return workIntent
	}
	if workIntent.Intent == "search_web" {
		return WorkIntentResult{Intent: "none", Confidence: 0, Proactive: false}
	}
	return workIntent
}
