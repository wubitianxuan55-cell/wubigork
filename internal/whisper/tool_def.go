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



