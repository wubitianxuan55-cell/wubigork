// Package whisper — tool_def.go
// 100% 对齐 ackem desktop-agent/toolDef.ts
// use_computer 工具定义

package whisper

// UseComputerToolName 工具名
const UseComputerToolName = "use_computer"

// UseComputerArgs 工具参数
type UseComputerArgs struct {
	Action  DesktopAgentAction  `json:"action"`
	Path    string              `json:"path,omitempty"`
	PathTo  string              `json:"path_to,omitempty"`
	Target  string              `json:"target,omitempty"`
	Query   string              `json:"query,omitempty"`
	URL     string              `json:"url,omitempty"`
	Options *UseComputerOptions `json:"options,omitempty"`
}

// UseComputerOptions 额外选项
type UseComputerOptions struct {
	Content string `json:"content,omitempty"`
}

// UseComputerAction 所有支持的动作
var UseComputerActions = []string{
	"list_folder", "search_files", "stat_file", "grep_text",
	"read_text", "read_document", "read_image",
	"open_folder", "open_file", "open_app",
	"close_file", "close_app",
	"copy_path", "move_path", "mkdir", "write_text", "delete_path",
	"download_file", "download_and_install", "run_installer",
	"import_to_ackem", "focus_app",
}

// UseComputerToolDef 返回 use_computer 工具定义（OpenAI function calling 格式）
func UseComputerToolDef() map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        UseComputerToolName,
			"description": "对本机文件或应用程序执行操作（浏览/搜索/读取/打开/整理/下载/导入等）。每次执行前需用户在弹窗中确认。仅在电脑助手模式开启时使用。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"description": "动作：" + joinActions(),
					},
					"path":    map[string]interface{}{"type": "string", "description": "本机路径（绝对或相对用户目录）"},
					"path_to": map[string]interface{}{"type": "string", "description": "目标路径（复制/移动）"},
					"target":  map[string]interface{}{"type": "string", "description": "应用名、窗口名或关闭对象"},
					"query":   map[string]interface{}{"type": "string", "description": "搜索关键词"},
					"url":     map[string]interface{}{"type": "string", "description": "HTTPS 下载地址"},
					"options": map[string]interface{}{
						"type":        "object",
						"description": "额外选项，如 write_text 的 content",
						"properties": map[string]interface{}{
							"content": map[string]interface{}{"type": "string"},
						},
					},
				},
				"required": []string{"action"},
			},
		},
	}
}

func joinActions() string {
	result := ""
	for i, a := range UseComputerActions {
		if i > 0 {
			result += ", "
		}
		result += a
	}
	return result
}

// ParseUseComputerArgs 解析工具调用参数
func ParseUseComputerArgs(raw map[string]interface{}) *UseComputerArgs {
	action, ok := raw["action"].(string)
	if !ok || action == "" {
		return nil
	}
	args := &UseComputerArgs{Action: DesktopAgentAction(action)}
	if v, ok := raw["path"].(string); ok {
		args.Path = v
	}
	if v, ok := raw["path_to"].(string); ok {
		args.PathTo = v
	}
	if v, ok := raw["target"].(string); ok {
		args.Target = v
	}
	if v, ok := raw["query"].(string); ok {
		args.Query = v
	}
	if v, ok := raw["url"].(string); ok {
		args.URL = v
	}
	if v, ok := raw["options"].(map[string]interface{}); ok {
		opts := &UseComputerOptions{}
		if content, ok := v["content"].(string); ok {
			opts.Content = content
		}
		args.Options = opts
	}
	return args
}
