// Package whisper — desktop_agent_loop_messages.go
// 100% 对齐 ackem desktop-agent/agentLoopMessages.ts
// 工具循环消息拼接：将工具结果注入对话历史
package whisper

import "encoding/json"

// DesktopAgentMaxToolRounds 最大工具循环轮数
const DesktopAgentMaxToolRounds = 16

var toolLabel = map[string]string{
	"use_computer": "电脑助手",
	"web_search":   "网页搜索",
	"read_file":    "文件读取",
}

// ToolResultForFollowUp 工具结果（供 FollowUp 使用）
type ToolResultForFollowUp struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// AppendDesktopAgentRoundMessages 将一轮工具结果追加到对话
func AppendDesktopAgentRoundMessages(
	base []map[string]interface{},
	assistantPartial string,
	toolResults []ToolResultForFollowUp,
	taskPlanActive bool,
	taskPlanNudge string,
) []map[string]interface{} {
	next := make([]map[string]interface{}, 0, len(base)+3)

	for _, m := range base {
		role, _ := m["role"].(string)
		content := m["content"]
		if s, ok := content.(string); ok {
			next = append(next, map[string]interface{}{"role": role, "content": s})
		} else {
			b, _ := json.Marshal(content)
			next = append(next, map[string]interface{}{"role": role, "content": string(b)})
		}
	}

	if assistantPartial != "" {
		next = append(next, map[string]interface{}{"role": "assistant", "content": assistantPartial})
	}

	var blocks []string
	for _, tr := range toolResults {
		if tr.Name == "append_memory" {
			continue
		}
		label := tr.Name
		if l, ok := toolLabel[tr.Name]; ok {
			label = l
		}
		blocks = append(blocks, "【"+label+"结果】\n"+tr.Content)
	}

	if len(blocks) > 0 {
		continueHint := ""
		if taskPlanActive {
			continueHint = "【继续任务】以上是工具返回的真实结果。多步骤任务计划尚未全部验收通过时，必须继续调用 use_computer 完成下一步，禁止仅用文字声称已完成。"
		} else {
			continueHint = "【继续任务】以上是工具返回的真实结果。若已足够回答用户最初的问题，请直接给出完整结论；若还需查看其他目录/文件，请继续调用 use_computer 自行探索，不要重复询问用户。"
		}

		blockText := ""
		for i, b := range blocks {
			if i > 0 {
				blockText += "\n\n"
			}
			blockText += b
		}
		next = append(next, map[string]interface{}{
			"role": "user", "content": blockText + "\n\n" + continueHint,
		})
	}

	if taskPlanNudge != "" {
		next = append(next, map[string]interface{}{"role": "user", "content": taskPlanNudge})
	}

	return next
}
