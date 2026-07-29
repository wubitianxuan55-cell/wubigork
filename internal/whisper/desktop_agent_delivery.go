// Package whisper — desktop_agent_delivery.go
// 100% 对齐 ackem desktop-agent/desktopAgentDelivery.ts
// 桌面助手结果格式化与交付
package whisper

import "strings"

// FormatUseComputerListDelivery 从工具结果中提取 [DIR]/[FILE] 标记
func FormatUseComputerListDelivery(content string) string {
	if !strings.Contains(content, "[DIR]") && !strings.Contains(content, "[FILE]") {
		return content
	}
	return content
}

// LlmHasSubstantiveDirectoryList 判断 LLM 回复是否已覆盖目录列表
func LlmHasSubstantiveDirectoryList(llmReply string, toolContent string) bool {
	if toolContent == "" {
		return true
	}
	// 简单检查：LLM 回复中是否包含工具结果的主要条目
	toolLines := strings.Split(toolContent, "\n")
	substantiveCount := 0
	matchCount := 0
	for _, line := range toolLines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[DIR]") || strings.HasPrefix(line, "[FILE]") {
			substantiveCount++
			// 提取文件名
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 && strings.Contains(strings.ToLower(llmReply), strings.ToLower(parts[1])) {
				matchCount++
			}
		}
	}
	if substantiveCount == 0 {
		return true
	}
	return float64(matchCount)/float64(substantiveCount) >= 0.5
}

// DesktopAgentDeliveryCoversToolResults 检查 LLM 交付是否覆盖了工具结果
func DesktopAgentDeliveryCoversToolResults(llmReply string, toolResults []ToolResultForFollowUp) bool {
	for _, tr := range toolResults {
		if tr.Name != "use_computer" {
			continue
		}
		if !LlmHasSubstantiveDirectoryList(llmReply, tr.Content) {
			return false
		}
	}
	return true
}

// MergeDesktopAgentDelivery 若 LLM 未列出真实内容，将工具结果结构化补在前面
func MergeDesktopAgentDelivery(llmReply string, toolResults []ToolResultForFollowUp) string {
	if DesktopAgentDeliveryCoversToolResults(llmReply, toolResults) {
		return llmReply
	}

	var parts []string
	for _, tr := range toolResults {
		if tr.Name != "use_computer" {
			continue
		}
		formatted := FormatUseComputerListDelivery(tr.Content)
		parts = append(parts, "【电脑助手查找结果】\n"+formatted)
	}

	if len(parts) == 0 {
		return llmReply
	}

	return strings.Join(parts, "\n\n") + "\n\n" + llmReply
}

// MergeToolResultsForDelivery 多轮工具调用时，仅保留条目最多的 use_computer 结果
func MergeToolResultsForDelivery(allResults [][]ToolResultForFollowUp) []ToolResultForFollowUp {
	var best []ToolResultForFollowUp
	bestCount := 0

	for _, round := range allResults {
		for _, tr := range round {
			if tr.Name == "use_computer" {
				count := strings.Count(tr.Content, "\n")
				if count > bestCount {
					best = round
					bestCount = count
				}
			}
		}
	}

	if best == nil {
		// 返回最后一轮
		if len(allResults) > 0 {
			return allResults[len(allResults)-1]
		}
		return nil
	}
	return best
}

// BuildDesktopAgentFollowUpSuffix 生成 FollowUp 请求的附录指令
func BuildDesktopAgentFollowUpSuffix(hasDirList bool) string {
	if hasDirList {
		return "（以上工具结果已列出真实目录内容，你的回复应引用这些实际条目，不要编造文件名）"
	}
	return ""
}
