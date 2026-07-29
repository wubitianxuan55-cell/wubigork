// Package whisper — desktop_session_mode.go
// 100% 对齐 ackem desktop-agent/sessionMode.ts
// 会话模式持久化：JSON 文件存储每个 session 的桌面助手开关
package whisper

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// modeFilePath 会话模式文件路径
func modeFilePath(dataRoot string) string {
	return filepath.Join(dataRoot, "desktop-agent", "session-modes.json")
}

// readAllSessionModes 读取所有会话模式
func readAllSessionModes(dataRoot string) map[string]bool {
	p := modeFilePath(dataRoot)
	data, err := os.ReadFile(p)
	if err != nil {
		return make(map[string]bool)
	}
	var modes map[string]bool
	if err := json.Unmarshal(data, &modes); err != nil {
		return make(map[string]bool)
	}
	return modes
}

// writeAllSessionModes 写入所有会话模式
func writeAllSessionModes(dataRoot string, modes map[string]bool) error {
	p := modeFilePath(dataRoot)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(modes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// GetDesktopAgentChatMode 获取指定 session 的桌面助手模式
func GetDesktopAgentChatMode(dataRoot, sessionID string) bool {
	return readAllSessionModes(dataRoot)[sessionID]
}

// SetDesktopAgentChatMode 设置指定 session 的桌面助手模式
func SetDesktopAgentChatMode(dataRoot, sessionID string, enabled bool) error {
	all := readAllSessionModes(dataRoot)
	if enabled {
		all[sessionID] = true
	} else {
		delete(all, sessionID)
	}
	return writeAllSessionModes(dataRoot, all)
}

// ClearDesktopAgentChatModeForAllSessions 清除所有会话的桌面助手模式
func ClearDesktopAgentChatModeForAllSessions(dataRoot string) error {
	return writeAllSessionModes(dataRoot, make(map[string]bool))
}
