// Package whisper — desktop_synthesize.go
// 100% 对齐 ackem desktop-agent/investigation/synthesize.ts
// LLM 合成调查回复 + 幻觉守卫验证 + 模板回退

package whisper

import (
	"fmt"
	"strings"
)

// ─── 辅助转换 ──────────────────────────────────────────────────

// findingsToStrings 将 InvestigationFinding 切片转为字符串切片
func findingsToStrings(findings []InvestigationFinding) []string {
	result := make([]string, 0, len(findings))
	for _, f := range findings {
		name := f.Name
		if name == "" {
			name = f.Match
		}
		if name != "" {
			if f.Source != "" {
				result = append(result, name+" ("+f.Source+")")
			} else {
				result = append(result, name)
			}
		}
	}
	return result
}

// findingsToNames 提取名称列表
func findingsToNames(findings []InvestigationFinding) []string {
	names := make([]string, 0, len(findings))
	for _, f := range findings {
		name := f.Name
		if name == "" {
			name = f.Match
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// ─── LLM 合成 ──────────────────────────────────────────────────

// SynthesizeInvestigationReply 用 LLM 合成调查结果的自然语言回复
// 100% 对齐 ackem synthesize.ts synthesizeInvestigationReply
func SynthesizeInvestigationReply(
	userQuery string,
	findings []InvestigationFinding,
	template string,
	llmCall func(systemPrompt, userPrompt string) (string, error),
) (string, error) {
	reportText := formatFindingsForLLM(findings, template)
	systemPrompt := buildSynthesizeSystemPrompt(template)

	reply, err := llmCall(systemPrompt, reportText+"\n\n用户问题："+userQuery)
	if err != nil {
		return FormatFindingsFallbackReply(findingsToStrings(findings), nil), nil
	}

	reply = strings.TrimSpace(reply)
	if reply == "" {
		return FormatFindingsFallbackReply(findingsToStrings(findings), nil), nil
	}

	// 幻觉检测
	ok, _ := DetectHallucination(reply, findingsToNames(findings))
	if !ok {
		return FormatFindingsFallbackReply(findingsToStrings(findings), nil), nil
	}

	return reply, nil
}

// ─── 提示词构建 ────────────────────────────────────────────────

// buildSynthesizeSystemPrompt 构建合成的 system prompt
func buildSynthesizeSystemPrompt(template string) string {
	if template == "games" {
		return strings.Join([]string{
			"你是用户的 AI gaea，正在帮用户查看电脑上的游戏清单。",
			"请用自然中文组织回复，包括：",
			"1. 简单一句话引起兴趣（如「找到几款」「来看看」）",
			"2. 列出游戏名称（带平台来源，如 Steam/Epic）",
			"3. 可以简单评价或推荐其中的1-2款",
			"",
			"约束：",
			"- 仅引用 findings 中的游戏名称，不得新增或编造",
			"- 语气贴合人设，不要百科客服腔",
			"- 不要逐条罗列 action 名或路径",
			"- 保持简洁，一般不超过 10 行",
		}, "\n")
	}

	return strings.Join([]string{
		"你是用户的 AI gaea，正在帮用户查看电脑上的文档清单。",
		"请用自然中文组织回复，包括：",
		"1. 简单一句话总结（如「找到几个文件」「在某某位置」）",
		"2. 列出文件名和大致位置",
		"3. 提示用户可以让你读取或打开哪个",
		"",
		"约束：",
		"- 仅引用 findings 中的文件名，不得新增或编造",
		"- 语气贴合人设，不要百科客服腔",
		"- 不要逐条罗列 action 名或完整路径",
		"- 保持简洁，一般不超过 10 行",
	}, "\n")
}

// buildSimpleReport builds a simple fallback report when LLM call fails
func buildSimpleReport(findings []InvestigationFinding, template string) string {
	return FormatFindingsFallbackReply(findingsToStrings(findings), nil)
}

// formatFindingsForLLM 格式化 findings 供 LLM 使用
func formatFindingsForLLM(findings []InvestigationFinding, template string) string {
	if len(findings) == 0 {
		return "未找到任何结果。"
	}

	var lines []string
	if template == "games" {
		lines = append(lines, "在本机上找到以下游戏：")
	} else {
		lines = append(lines, "在本机上找到以下文件：")
	}

	for i, f := range findings {
		name := f.Name
		if name == "" {
			name = f.Match
		}
		source := ""
		if f.Source != "" {
			source = fmt.Sprintf("（来源：%s）", f.Source)
		}
		lines = append(lines, fmt.Sprintf("%d. %s %s", i+1, name, source))
		if f.Path != "" {
			lines = append(lines, fmt.Sprintf("   路径：%s", f.Path))
		}
	}

	lines = append(lines, fmt.Sprintf("\n总计：%d 个条目", len(findings)))
	return strings.Join(lines, "\n")
}
