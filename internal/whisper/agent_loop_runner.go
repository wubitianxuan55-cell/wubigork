// Package whisper — agent_loop_runner.go
// 100% 对齐 ackem desktop-agent/openAiAgentJobRunner.ts
// Agent 多轮循环运行器：LLM 驱动 + 工具执行 + 结果反馈 + 交付
package whisper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// AgentLoopRunner Agent 循环运行器
type AgentLoopRunner struct {
	Llm          LlmClient
	Router       *RouterContext
	JobManager   *AgentJobManager
	MaxRounds    int
	SessionID    string
	SystemPrompt string
}

// AgentLoopResult 循环最终结果
type AgentLoopResult struct {
	FinalReply    string            `json:"finalReply"`
	ToolRounds    int               `json:"toolRounds"`
	TotalResults  int               `json:"totalResults"`
	AllPassed     bool              `json:"allPassed"`
	TaskPlan      *TaskPlan         `json:"taskPlan,omitempty"`
	AuditEntries  []string          `json:"auditEntries,omitempty"`
}

// DefaultAgentLoopRunner 创建默认运行器
func DefaultAgentLoopRunner(llm LlmClient) *AgentLoopRunner {
	return &AgentLoopRunner{
		Llm:       llm,
		MaxRounds: DesktopAgentMaxToolRounds,
	}
}

// RunAgentLoop 执行 Agent 多轮循环
func (r *AgentLoopRunner) RunAgentLoop(ctx context.Context, userMsg string, plan *TaskPlan) *AgentLoopResult {
	result := &AgentLoopResult{}

	// 构建初始消息
	messages := r.buildInitialMessages(userMsg, plan)

	for round := 1; round <= r.MaxRounds; round++ {
		select {
		case <-ctx.Done():
			result.FinalReply = "任务已取消"
			return result
		default:
		}

		// 更新状态
		if r.JobManager != nil {
			r.JobManager.UpdateJobPhase(r.SessionID, "executing",
				fmt.Sprintf("第 %d/%d 轮执行中…", round, r.MaxRounds))
		}

		// 调用 LLM
		assistantText, toolCalls, err := r.callLLM(messages)
		if err != nil {
			result.FinalReply = fmt.Sprintf("LLM 调用失败：%v", err)
			return result
		}

		// 无工具调用 → LLM 认为任务完成
		if len(toolCalls) == 0 {
			result.FinalReply = assistantText
			result.ToolRounds = round - 1
			result.AllPassed = true
			return result
		}

		// 执行工具调用
		toolResults := r.executeToolBatch(toolCalls)
		result.TotalResults += len(toolResults)
		result.ToolRounds = round

		// 将本轮结果追加到消息
		messages = AppendDesktopAgentRoundMessages(
			messages,
			assistantText,
			toolResults,
			plan != nil,
			"",
		)

		// 检查是否应继续
		if !r.shouldContinue(toolResults) {
			// 生成最终回复
			finalReply, _, _ := r.callLLM(append(messages, map[string]interface{}{
				"role": "user",
				"content": "以上是全部工具执行结果。请用自然中文给用户一个完整总结。",
			}))
			result.FinalReply = finalReply
			result.AllPassed = true
			return result
		}
	}

	// 超过最大轮数
	result.FinalReply = "任务已达到最大执行轮数，部分步骤可能未完成。"
	return result
}

// buildInitialMessages 构建初始消息
func (r *AgentLoopRunner) buildInitialMessages(userMsg string, plan *TaskPlan) []map[string]interface{} {
	messages := []map[string]interface{}{
		{"role": "system", "content": r.SystemPrompt},
		{"role": "user", "content": userMsg},
	}

	// 注入任务计划
	if plan != nil {
		planBlock := InjectTaskPlanToSystemPrompt("", plan)
		messages = append(messages, map[string]interface{}{
			"role": "system", "content": planBlock,
		})
	}

	return messages
}

// ─── LLM 调用 ────────────────────────────────────────────────────

// toolCallRaw LLM 返回的工具调用
type toolCallRaw struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// callLLM 调用 LLM 并解析工具调用
func (r *AgentLoopRunner) callLLM(messages []map[string]interface{}) (string, []AgentAction, error) {
	// 构建 system + user prompt
	sysPrompt := r.SystemPrompt
	var userPrompt string

	for _, m := range messages {
		role, _ := m["role"].(string)
		content, _ := m["content"].(string)
		switch role {
		case "system":
			if sysPrompt != "" {
				sysPrompt += "\n\n"
			}
			sysPrompt += content
		case "user", "assistant":
			roleLabel := "用户"
			if role == "assistant" {
				roleLabel = "助手"
			}
			userPrompt += fmt.Sprintf("[%s] %s\n", roleLabel, content)
		}
	}

	// 追加工具调用格式提示
	userPrompt += "\n如需使用工具，按以下格式输出：\n"
	userPrompt += "```tool_call\n{\"name\":\"use_computer\",\"arguments\":\"{\\\"action\\\":\\\"list_folder\\\",\\\"path\\\":\\\"C:\\\\\"}\"}\n```\n"
	userPrompt += "如无需工具，直接回复用户。"

	raw, err := r.Llm.Chat(sysPrompt, userPrompt)
	if err != nil {
		return "", nil, err
	}

	// 解析工具调用
	assistantText, toolCalls := parseToolCallsFromReply(raw)
	return assistantText, toolCalls, nil
}

// parseToolCallsFromReply 从 LLM 回复中解析工具调用
func parseToolCallsFromReply(raw string) (string, []AgentAction) {
	// 查找 ```tool_call 块
	var toolCalls []AgentAction
	var cleanedText strings.Builder

	lines := strings.Split(raw, "\n")
	inToolBlock := false
	var toolBlock strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```tool_call") {
			inToolBlock = true
			continue
		}
		if inToolBlock && strings.HasPrefix(strings.TrimSpace(line), "```") {
			inToolBlock = false
			// 解析工具调用
			if call := parseSingleToolCall(toolBlock.String()); call != nil {
				toolCalls = append(toolCalls, *call)
			}
			toolBlock.Reset()
			continue
		}
		if inToolBlock {
			toolBlock.WriteString(line)
			toolBlock.WriteString("\n")
		} else {
			cleanedText.WriteString(line)
			cleanedText.WriteString("\n")
		}
	}

	return strings.TrimSpace(cleanedText.String()), toolCalls
}

// parseSingleToolCall 解析单个工具调用 JSON
func parseSingleToolCall(raw string) *AgentAction {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var tc toolCallRaw
	if err := json.Unmarshal([]byte(raw), &tc); err != nil {
		return nil
	}

	// 解析 arguments 子 JSON
	var args map[string]string
	if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
		// 尝试直接用
		args = map[string]string{"raw": tc.Arguments}
	}

	return &AgentAction{
		Name:   tc.Name,
		Args:   args,
		Reason: "",
	}
}

// ─── 工具执行 ────────────────────────────────────────────────────

// executeToolBatch 批量执行工具调用
func (r *AgentLoopRunner) executeToolBatch(actions []AgentAction) []ToolResultForFollowUp {
	var results []ToolResultForFollowUp

	for _, act := range actions {
		switch act.Name {
		case "use_computer":
			if r.Router != nil {
				args := UseComputerArgs{
					Action: DesktopAgentAction(act.Args["action"]),
					Path:   act.Args["path"],
					PathTo: act.Args["path_to"],
					Target: act.Args["target"],
					Query:  act.Args["query"],
					URL:    act.Args["url"],
				}
				result := ExecuteUseComputer(args, *r.Router)
				results = append(results, ToolResultForFollowUp{
					Name:    "use_computer",
					Content: result.Content,
				})
			}
		case "web_search":
			query := ""
			if act.Args != nil {
				query = act.Args["query"]
			}
			result, err := WebSearch(query)
			if err != nil {
				result = fmt.Sprintf("搜索「%s」失败", query)
			}
			results = append(results, ToolResultForFollowUp{
				Name:    "web_search",
				Content: result,
			})
		case "append_memory":
			// 记忆写入由 ingest 管线处理
			results = append(results, ToolResultForFollowUp{
				Name:    "append_memory",
				Content: "记忆已记录",
			})
		default:
			results = append(results, ToolResultForFollowUp{
				Name:    act.Name,
				Content: fmt.Sprintf("未知工具：%s", act.Name),
			})
		}
	}

	return results
}

// shouldContinue 判断是否应继续循环
func (r *AgentLoopRunner) shouldContinue(results []ToolResultForFollowUp) bool {
	// 有 use_computer 调用 → 继续
	for _, tr := range results {
		if tr.Name == "use_computer" {
			return true
		}
	}
	return false
}
