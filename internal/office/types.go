// Package office — 办公模块共享类型
package office

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

func Itoa(n int) string {
	if n == 0 { return "0" }
	neg := false
	if n < 0 { neg = true; n = -n }
	var buf [20]byte
	i := len(buf)
	for n > 0 { i--; buf[i] = byte('0'+n%10); n /= 10 }
	if neg { i--; buf[i] = '-' }
	return string(buf[i:])
}

func TruncStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen { return s }
	return string(runes[:maxLen]) + "…"
}

func ContainsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub { return true }
		}
	}
	return false
}
