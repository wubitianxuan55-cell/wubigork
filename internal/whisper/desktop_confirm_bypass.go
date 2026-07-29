// Package whisper — desktop_confirm_bypass.go
// 100% 对齐 ackem desktop-agent/confirmBypass.ts
// 确认绕过管理：session 级 + TaskPlan 级自动批准
package whisper

import "sync"

// AlwaysConfirmActions 即使开启自动批准也必须单独确认的操作
var AlwaysConfirmActions = map[DesktopAgentAction]bool{
	ActionCloseFile:         true,
	ActionCloseApp:          true,
	ActionDeletePath:        true,
	ActionRunInstaller:      true,
	ActionDownloadAndInstall: true,
}

// ─── Session 级自动批准 ─────────────────────────────────────────

var (
	sessionAutoApprove   = make(map[string]bool)
	sessionAutoApproveMu sync.RWMutex
)

func sessionKey(dataRoot, sessionID string) string {
	return dataRoot + "::" + sessionID
}

// SetDesktopAgentSessionAutoApprove 开启 session 级自动批准
func SetDesktopAgentSessionAutoApprove(dataRoot, sessionID string) {
	sessionAutoApproveMu.Lock()
	defer sessionAutoApproveMu.Unlock()
	sessionAutoApprove[sessionKey(dataRoot, sessionID)] = true
}

// HasDesktopAgentSessionAutoApprove 检查 session 是否开启了自动批准
func hasDesktopAgentSessionAutoApprove(dataRoot, sessionID string) bool {
	sessionAutoApproveMu.RLock()
	defer sessionAutoApproveMu.RUnlock()
	return sessionAutoApprove[sessionKey(dataRoot, sessionID)]
}

// ClearDesktopAgentSessionAutoApprove 清除 session 级自动批准
func ClearDesktopAgentSessionAutoApprove(dataRoot, sessionID string) {
	sessionAutoApproveMu.Lock()
	defer sessionAutoApproveMu.Unlock()
	if sessionID == "" {
		for k := range sessionAutoApprove {
			if len(k) > len(dataRoot)+2 && k[:len(dataRoot)+2] == dataRoot+"::" {
				delete(sessionAutoApprove, k)
			}
		}
		return
	}
	delete(sessionAutoApprove, sessionKey(dataRoot, sessionID))
}

// ─── TaskPlan 级 delete 自动批准 ─────────────────────────────────

var (
	taskPlanDeleteAutoApprove   = make(map[string]bool)
	taskPlanDeleteAutoApproveMu sync.RWMutex
)

// SetTaskPlanDeleteAutoApprove 开启 TaskPlan 级 delete 自动批准
func SetTaskPlanDeleteAutoApprove(taskPlanID string) {
	if taskPlanID == "" {
		return
	}
	taskPlanDeleteAutoApproveMu.Lock()
	defer taskPlanDeleteAutoApproveMu.Unlock()
	taskPlanDeleteAutoApprove[taskPlanID] = true
}

// ClearTaskPlanDeleteAutoApprove 清除 TaskPlan 级 delete 自动批准
func ClearTaskPlanDeleteAutoApprove(taskPlanID string) {
	taskPlanDeleteAutoApproveMu.Lock()
	defer taskPlanDeleteAutoApproveMu.Unlock()
	if taskPlanID == "" {
		taskPlanDeleteAutoApprove = make(map[string]bool)
		return
	}
	delete(taskPlanDeleteAutoApprove, taskPlanID)
}

// ShouldSkipDesktopAgentConfirm 判断是否可以跳过确认
func ShouldSkipDesktopAgentConfirm(dataRoot, sessionID string, action DesktopAgentAction, taskPlanID string) bool {
	// TaskPlan 级 delete 豁免
	if action == ActionDeletePath && taskPlanID != "" {
		taskPlanDeleteAutoApproveMu.RLock()
		ok := taskPlanDeleteAutoApprove[taskPlanID]
		taskPlanDeleteAutoApproveMu.RUnlock()
		if ok {
			return true
		}
	}

	if !hasDesktopAgentSessionAutoApprove(dataRoot, sessionID) {
		return false
	}
	return !AlwaysConfirmActions[action]
}
