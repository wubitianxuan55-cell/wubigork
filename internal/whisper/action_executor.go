// Package whisper — action_executor.go
// 100% 对齐 ackem engine/actionExecutor.ts
// L5 工具执行引擎：信任门控 + 语气包装 + 执行调度

package whisper

import "strings"

// ─── ActionContext ────────────────────────────────────────────

// ActionContext 工具执行上下文
type ActionContext struct {
	DataRoot string
	L1       L1State
	L2       EmotionState
}

// ─── TrustGate ────────────────────────────────────────────────

// TrustGate 信任门控：检查是否有足够信任执行某工具
func TrustGate(toolName string, trust float64) bool {
	switch toolName {
	case "web_search":
		return trust >= 10
	case "read_file":
		return trust >= 20
	case "append_memory":
		return trust >= 40
	case "write_file":
		return trust >= 40
	case "run_command":
		return trust >= 60
	default:
		return true
	}
}

// ─── ToneWrap ─────────────────────────────────────────────────

// ToneWrap 根据情绪包裹语气
func ToneWrap(l2 Emotion4D, neutralText string) string {
	if l2.Aff >= 60 {
		return "✨ " + strings.ReplaceAll(neutralText, "\n", "\n✨ ")
	}
	if l2.Aff <= -30 {
		return "… " + strings.ReplaceAll(neutralText, "\n", "\n… ")
	}
	if l2.Aro >= 70 {
		return "⚡ " + strings.ReplaceAll(neutralText, "\n", "\n⚡ ")
	}
	return neutralText
}

// ─── ExecuteAction ────────────────────────────────────────────

// ExecuteAction 执行单个工具调用
func ExecuteAction(toolName string, args map[string]string, ctx *ActionContext) AgentToolResult {
	// 信任门控
	if ctx != nil && !TrustGate(toolName, ctx.L1.Trust) {
		return AgentToolResult{
			ToolName: toolName,
			Success:  false,
			Content:  "信任度不足，无法执行 " + toolName,
			Summary:  "需要更高的信任度才能执行此操作",
		}
	}

	switch toolName {
	case "web_search":
		return executeSearch(args)
	case "read_file":
		return executeReadFile(args)
	case "write_file":
		return executeWriteFile(args)
	case "list_directory":
		return executeListDir(args)
	default:
		return AgentToolResult{
			ToolName: toolName,
			Success:  false,
			Content:  "未知工具：" + toolName,
			Summary:  "工具 " + toolName + " 不在支持列表中",
		}
	}
}

func executeSearch(args map[string]string) AgentToolResult {
	query := strings.TrimSpace(args["query"])
	if query == "" {
		return AgentToolResult{
			ToolName: "web_search",
			Success:  false,
			Content:  "搜索词为空",
			Summary:  "搜索失败：未提供搜索词",
		}
	}
	return AgentToolResult{
		ToolName: "web_search",
		Success:  true,
		Content:  "搜索请求已提交：" + query,
		Summary:  "搜索「" + truncStr(query, 50) + "」",
	}
}

func executeReadFile(args map[string]string) AgentToolResult {
	path := strings.TrimSpace(args["path"])
	if path == "" {
		return AgentToolResult{
			ToolName: "read_file",
			Success:  false,
			Content:  "文件路径为空",
			Summary:  "读取失败：未提供路径",
		}
	}
	return AgentToolResult{
		ToolName: "read_file",
		Success:  true,
		Content:  "文件读取请求已提交：" + path,
		Summary:  "读取文件「" + path + "」",
	}
}

func executeWriteFile(args map[string]string) AgentToolResult {
	path := strings.TrimSpace(args["path"])
	if path == "" {
		return AgentToolResult{
			ToolName: "write_file",
			Success:  false,
			Content:  "文件路径为空",
			Summary:  "写入失败：未提供路径",
		}
	}
	return AgentToolResult{
		ToolName: "write_file",
		Success:  true,
		Content:  "文件写入请求已提交：" + path,
		Summary:  "写入文件「" + path + "」",
	}
}

func executeListDir(args map[string]string) AgentToolResult {
	path := strings.TrimSpace(args["path"])
	if path == "" {
		path = "."
	}
	return AgentToolResult{
		ToolName: "list_directory",
		Success:  true,
		Content:  "目录列表请求已提交：" + path,
		Summary:  "列出目录「" + path + "」",
	}
}
