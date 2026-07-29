// Package whisper — tool_followup.go
// 100% 对齐 ackem prompt/tool-followup.ts
// 工具调用跟进：人格化 fallback + 结果块组装

package whisper

import (
	"fmt"
	"strings"
)

// ToolLabel 工具中文标签
var ToolLabel = map[string]string{
	"web_search": "网页搜索",
	"read_file":  "文件读取",
}

// BuildToolResultsFallback 工具跟进的人格化 fallback
func BuildToolResultsFallback(personalityID string) string {
	fallbacks := map[string]string{
		"tsundere": "哼……查是查到了，但我一时不知道怎么说。你自己看上面吧。",
		"kuudere":  "……查到了。看上面。",
		"deredere": "我帮你查了，但一时组织不好语言。你先看看上面的内容，有疑问再问我。",
		"yandere":  "我查到了……但我现在不想说。你自己看。",
		"genki":    "诶~查到了但我说不太好！你先看看上面的！",
	}
	if f, ok := fallbacks[personalityID]; ok {
		return f
	}
	return "我帮你查了，详情在上面。"
}

// BuildEmptyResultFallback 空结果人格化 fallback
func BuildEmptyResultFallback(personalityID string) string {
	fallbacks := map[string]string{
		"tsundere": "这破网站什么都没写，别问我了。",
		"kuudere":  "没找到。换个说法试试。",
		"deredere": "我帮你查了，但没找到有用的。要不换个关键词？",
		"yandere":  "查不到……是不是有人把信息藏起来了？",
		"genki":    "诶~没找到！换个说法试试吧！",
	}
	if f, ok := fallbacks[personalityID]; ok {
		return f
	}
	return "没找到相关信息。"
}

// ToolResult 工具调用结果
type ToolResult struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// BuildToolFollowUpBlock 构建工具跟进 user prompt block
func BuildToolFollowUpBlock(toolResults []ToolResult) string {
	var blocks []string
	for _, tr := range toolResults {
		if tr.Name == "append_memory" {
			continue
		}
		label := ToolLabel[tr.Name]
		if label == "" {
			label = tr.Name
		}
		blocks = append(blocks, fmt.Sprintf("【%s结果】\n%s", label, tr.Content))
	}
	if len(blocks) == 0 {
		return ""
	}
	return strings.Join(blocks, "\n\n") + "\n\n" +
		"【任务】请直接回答用户上一句的问题。\n" +
		"- 以搜索结果为主，若摘要偏泛可结合常识简要补充；\n" +
		"- 禁止说「要不要再搜」「换个关键词」等推脱话。"
}
