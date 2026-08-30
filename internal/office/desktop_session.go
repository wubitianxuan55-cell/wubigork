// Package office — desktop_session.go
package office

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

func modeFilePath(dataRoot string) string {
	return filepath.Join(dataRoot, "office", "session-modes.json")
}

func LoadModes(dataRoot string) map[string]bool {
	p := modeFilePath(dataRoot)
	data, err := os.ReadFile(p)
	if err != nil { return make(map[string]bool) }
	var modes map[string]bool
	if err := json.Unmarshal(data, &modes); err != nil { return make(map[string]bool) }
	return modes
}

func SaveModes(dataRoot string, modes map[string]bool) error {
	p := modeFilePath(dataRoot)
	data, err := json.MarshalIndent(modes, "", "  ")
	if err != nil { return err }
	// 原子写（临时文件 + rename）：会话模式是高频落盘的小配置，崩溃/断电
	// 不留半截 JSON——与持久化套件其余成员统一走 fileutil.AtomicWrite。
	return fileutil.AtomicWrite(p, data, 0o644)
}

func GetPersistedMode(dataRoot, sessionID string) bool { return LoadModes(dataRoot)[sessionID] }

func SetPersistedMode(dataRoot, sessionID string, enabled bool) error {
	all := LoadModes(dataRoot)
	if enabled { all[sessionID] = true } else { delete(all, sessionID) }
	return SaveModes(dataRoot, all)
}

func ClearAllPersistedModes(dataRoot string) error { return SaveModes(dataRoot, make(map[string]bool)) }
