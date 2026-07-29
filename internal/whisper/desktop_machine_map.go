// Package whisper — desktop_machine_map.go
// 100% 对齐 ackem desktop-agent/machine-map/
package whisper

import (
	"os"
	"path/filepath"
	"strings"
)

// MachineMapEntry 机器地图条目
type MachineMapEntry struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Type      string `json:"type"` // game/document/app
	SizeBytes int64  `json:"sizeBytes"`
}

// MachineMap 机器地图（内存索引）
type MachineMap struct {
	Entries   []MachineMapEntry `json:"entries"`
	IndexedAt string            `json:"indexedAt"`
}

// NewMachineMap 创建空机器地图
func NewMachineMap() *MachineMap {
	return &MachineMap{
		Entries: make([]MachineMapEntry, 0),
	}
}

// ScanCommonLocations 扫描常见位置建立索引
func (mm *MachineMap) ScanCommonLocations() {
	home, _ := os.UserHomeDir()
	dirs := []struct {
		path string
		kind string
	}{
		{"C:\\Program Files", "app"},
		{"C:\\Program Files (x86)", "app"},
		{home + "\\Desktop", "document"},
		{home + "\\Documents", "document"},
		{home + "\\Downloads", "document"},
	}
	for _, d := range dirs {
		entries, err := os.ReadDir(d.path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			mm.Entries = append(mm.Entries, MachineMapEntry{
				Path:      filepath.Join(d.path, e.Name()),
				Name:      e.Name(),
				Type:      d.kind,
				SizeBytes: info.Size(),
			})
		}
	}
}

// Search 搜索机器地图
func (mm *MachineMap) Search(query string) []MachineMapEntry {
	q := strings.ToLower(query)
	var results []MachineMapEntry
	for _, e := range mm.Entries {
		if strings.Contains(strings.ToLower(e.Name), q) {
			results = append(results, e)
		}
	}
	return results
}

// IsGameDirectory 判断是否为游戏目录
func IsGameDirectory(path string) bool {
	gameIndicators := []string{
		"Steam", "Epic Games", "GOG", "Battle.net",
		"Riot Games", "Ubisoft", "Origin", "EA Games",
	}
	base := strings.ToLower(filepath.Base(path))
	for _, g := range gameIndicators {
		if strings.Contains(base, strings.ToLower(g)) {
			return true
		}
	}
	return false
}
