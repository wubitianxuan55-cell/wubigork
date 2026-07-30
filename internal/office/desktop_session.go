// Package office — desktop_session.go
package office

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil { return err }
	data, err := json.MarshalIndent(modes, "", "  ")
	if err != nil { return err }
	return os.WriteFile(p, data, 0644)
}

func GetPersistedMode(dataRoot, sessionID string) bool { return LoadModes(dataRoot)[sessionID] }

func SetPersistedMode(dataRoot, sessionID string, enabled bool) error {
	all := LoadModes(dataRoot)
	if enabled { all[sessionID] = true } else { delete(all, sessionID) }
	return SaveModes(dataRoot, all)
}

func ClearAllPersistedModes(dataRoot string) error { return SaveModes(dataRoot, make(map[string]bool)) }
