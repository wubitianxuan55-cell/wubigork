// Package core — 内核：桌面 agent 文件读写、会话持久化、证据链共用类型。平迁自 internal/office/types.go（逐行等价，语义零变化）。
package core

// TaskPlan 任务计划
type TaskPlan struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Steps     []TaskStep `json:"steps"`
	CreatedAt string     `json:"createdAt"`
	Status    string     `json:"status"`
}

// TaskStep 任务步骤
type TaskStep struct {
	Index       int    `json:"index"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Path        string `json:"path,omitempty"`
	Target      string `json:"target,omitempty"`
	Status      string `json:"status"`
	Result      string `json:"result,omitempty"`
}

// AgentJobState 任务状态
type AgentJobState struct {
	SessionID string `json:"sessionId"`
	Phase     string `json:"phase"`
	Label     string `json:"label"`
	Active    bool   `json:"active"`
	Error     string `json:"error,omitempty"`
}

// AgentTaskResult agent 任务结果
type AgentTaskResult struct {
	TaskID     string `json:"taskId"`
	TurnID     string `json:"turnId"`
	Success    bool   `json:"success"`
	Summary    string `json:"summary"`
	Content    string `json:"content"`
	MemoryHint string `json:"memoryHint,omitempty"`
}

// DesktopAgentAction 桌面 agent 操作类型
type DesktopAgentAction string

const (
	ActionReadText   DesktopAgentAction = "read_text"
	ActionListFolder DesktopAgentAction = "list_folder"
	ActionSearchFile DesktopAgentAction = "search_file"
	ActionStatFile   DesktopAgentAction = "stat_file"
	ActionOpenFile   DesktopAgentAction = "open_file"
	ActionCopyFile   DesktopAgentAction = "copy_file"
	ActionMoveFile   DesktopAgentAction = "move_file"
	ActionDeleteFile DesktopAgentAction = "delete_file"
	ActionCreateDir  DesktopAgentAction = "create_dir"
	ActionWriteFile  DesktopAgentAction = "write_file"
	ActionWebSearch  DesktopAgentAction = "web_search"
	ActionWebFetch   DesktopAgentAction = "web_fetch"
)
