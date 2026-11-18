package agent

import (
	"strings"
	"testing"
)

// TestDefaultTaskSystemPromptPantheonPins 钉死 2026-11-19 众神殿
// pan_1795007052552 原则迁移的关键段：中文输出锚（子代理 V6.0 不拼
// LanguagePolicy，输出语言只在此锚定）、事实纪律（工具真实返回/不伪装完成）、
// 不猜工具名+替代路径兜底、禁报告文件、评审轮次预算。改提示词不得静默丢段。
func TestDefaultTaskSystemPromptPantheonPins(t *testing.T) {
	pins := []string{
		"用中文输出",
		"必须出自工具真实返回",
		"不把失败伪装成完成",
		"不猜未显示的工具名",
		"改走 schema 内可达的替代路径",
		"不写报告/总结/发现/分析类文件",
		"至多修复尝试一次",
	}
	for _, pin := range pins {
		if !strings.Contains(DefaultTaskSystemPrompt, pin) {
			t.Fatalf("DefaultTaskSystemPrompt 缺少众神殿迁移段 %q", pin)
		}
	}
	if strings.Contains(DefaultTaskSystemPrompt, "general-purpose agent delegated") {
		t.Fatalf("DefaultTaskSystemPrompt 仍是英文旧稿（11-18 稿），应已中文重写")
	}
}
