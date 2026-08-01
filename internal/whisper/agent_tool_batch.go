// Package whisper — agent_tool_batch.go
// 100% 对齐 ackem desktop-agent/openAiToolRound.ts
// 批量工具调用执行器：路由分发 + 结果汇总
package whisper

import "fmt"

// ExecuteOpenAiToolBatch 执行 OpenAI 格式的工具调用批次
// 分发到：use_computer / web_search / append_memory / extract_facts / read_file
func ExecuteOpenAiToolBatch(
	actions []AgentAction,
	router *RouterContext,
	prefetchedFacts []string,
) []ToolResultForFollowUp {
	var results []ToolResultForFollowUp

	for _, act := range actions {
		switch act.Name {
		case "use_computer":
			result := dispatchUseComputer(act, router)
			results = append(results, result)

		case "web_search":
			result := dispatchWebSearch(act)
			results = append(results, result)

		case "append_memory":
			result := dispatchAppendMemory(act)
			results = append(results, result)

		case "read_file":
			result := dispatchReadFile(act)
			results = append(results, result)

		case "extract_facts":
			if len(prefetchedFacts) > 0 {
				results = append(results, ToolResultForFollowUp{
					Name:    "extract_facts",
					Content: fmt.Sprintf("已提取 %d 条事实", len(prefetchedFacts)),
				})
			}

		default:
			results = append(results, ToolResultForFollowUp{
				Name:    act.Name,
				Content: fmt.Sprintf("未知工具：%s。可用工具：use_computer, web_search, read_file, append_memory", act.Name),
			})
		}
	}

	return results
}

// dispatchUseComputer 分发 use_computer 调用
func dispatchUseComputer(act AgentAction, router *RouterContext) ToolResultForFollowUp {
	if router == nil {
		return ToolResultForFollowUp{
			Name:    "use_computer",
			Content: "错误：路由器未初始化",
		}
	}

	args := UseComputerArgs{
		Action: DesktopAgentAction(act.Args["action"]),
		Path:   act.Args["path"],
		PathTo: act.Args["path_to"],
		Target: act.Args["target"],
		Query:  act.Args["query"],
		URL:    act.Args["url"],
	}

	result := ExecuteUseComputer(args, *router)
	return ToolResultForFollowUp{
		Name:    "use_computer",
		Content: result.Content,
	}
}

// dispatchWebSearch 分发网页搜索
func dispatchWebSearch(act AgentAction) ToolResultForFollowUp {
	query := act.Args["query"]
	if query == "" {
		query = act.Args["search_term"]
	}
	if query == "" {
		return ToolResultForFollowUp{
			Name:    "web_search",
			Content: "错误：缺少搜索关键词",
		}
	}
	// 调用 Web 搜索
	result, err := WebSearch(query)
	if err != nil {
		return ToolResultForFollowUp{
			Name:    "web_search",
			Content: fmt.Sprintf("搜索「%s」失败：%v", query, err),
		}
	}
	return ToolResultForFollowUp{
		Name:    "web_search",
		Content: result,
	}
}

// dispatchAppendMemory 分发记忆追加
func dispatchAppendMemory(act AgentAction) ToolResultForFollowUp {
	fact := act.Args["fact"]
	if fact == "" {
		fact = act.Args["content"]
	}
	if fact == "" {
		return ToolResultForFollowUp{
			Name:    "append_memory",
			Content: "错误：缺少记忆内容",
		}
	}

	return ToolResultForFollowUp{
		Name:    "append_memory",
		Content: fmt.Sprintf("记忆已记录：%s", fact),
	}
}

// dispatchReadFile 分发文件读取
func dispatchReadFile(act AgentAction) ToolResultForFollowUp {
	path := act.Args["path"]
	if path == "" {
		path = act.Args["file_path"]
	}
	if path == "" {
		return ToolResultForFollowUp{
			Name:    "read_file",
			Content: "错误：缺少文件路径",
		}
	}

	result := ExecuteDesktopAgentAction(
		ActionReadText,
		path, "", "", "", "", "",
		DesktopExecContext{},
	)

	return ToolResultForFollowUp{
		Name:    "read_file",
		Content: result.Content,
	}
}

// ─── 循环判断 ────────────────────────────────────────────────────

// ShouldContinueDesktopAgentLoop 判断是否应继续 Agent 循环
func ShouldContinueDesktopAgentLoop(results []ToolResultForFollowUp, round, maxRounds int) bool {
	if round >= maxRounds {
		return false
	}

	// 有 use_computer 成功 → 继续
	for _, tr := range results {
		if tr.Name == "use_computer" {
			return true
		}
	}

	return false
}
