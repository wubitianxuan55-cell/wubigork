package config

// 办公身份稿 v1.0 冻结锚（2026-11-18 众神殿 pan_1795007052552，DeepSeek r1
// 附录「办公 A」原样）：身份层不点名具体工具（能力说明单源=工具 schema，
// 发现兜底=执行常量两行路由），53 行工具手册已淘汰；敏感文档本地优先句与
// 确认制记忆在场（既有约束保留）。

import (
	"strings"
	"testing"
)

func TestOfficeSystemPromptV1FrozenBody(t *testing.T) {
	p := DefaultSystemPrompt
	for _, sub := range []string{
		"通用办公 AI 助手", // 开场自称（boot 装配断言同锚）
		"【先判断任务】",
		"【事实与原稿】",
		"优先使用本地模型与本地工具链", // 敏感文档数据边界句保留
		"主动调用 remember", // 记忆确认制
	} {
		if !strings.Contains(p, sub) {
			t.Errorf("办公身份稿缺 v1.0 锚 %q", sub)
		}
	}
	// 身份层不点名工具（手册句/工具名回流即双源）：能力发现路由在执行常量。
	for _, banned := range []string{
		"semantic_search：", "routine_llm", "chart_gen", "format_convert",
		"fact_add", "office_", "doc-assemble", "chart-builder",
	} {
		if strings.Contains(p, banned) {
			t.Errorf("办公身份稿含工具手册/工具名 %q（身份层不点名工具）", banned)
		}
	}
}
