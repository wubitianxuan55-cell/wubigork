// Package whisper — desktop_parsers.go
// 100% 对齐 ackem desktop-agent/parsers/win/
package whisper

import (
	"os"
	"path/filepath"
	"strings"
)

// ShortcutInfo 快捷方式信息
type ShortcutInfo struct {
	Name     string `json:"name"`
	Target   string `json:"target"`
	Path     string `json:"path"`
}

// ParseDesktopShortcuts 解析桌面快捷方式
func ParseDesktopShortcuts() []ShortcutInfo {
	home, _ := os.UserHomeDir()
	desktopDir := filepath.Join(home, "Desktop")
	entries, err := os.ReadDir(desktopDir)
	if err != nil {
		return nil
	}

	var shortcuts []ShortcutInfo
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) != ".lnk" {
			continue
		}
		baseName := strings.TrimSuffix(name, ".lnk")
		shortcuts = append(shortcuts, ShortcutInfo{
			Name:   baseName,
			Target: baseName + ".exe",
			Path:   filepath.Join(desktopDir, name),
		})
	}
	return shortcuts
}

// ParseSteamLibraries 解析 Steam 库目录
func ParseSteamLibraries() []string {
	steamPaths := []string{
		"C:\\Program Files (x86)\\Steam\\steamapps\\common",
		"C:\\Program Files\\Steam\\steamapps\\common",
	}

	var libraries []string
	for _, p := range steamPaths {
		if entries, err := os.ReadDir(p); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					libraries = append(libraries, filepath.Join(p, e.Name()))
				}
			}
		}
	}
	return libraries
}

// ParseEpicManifests 解析 Epic 清单
func ParseEpicManifests() []string {
	epicPath := "C:\\ProgramData\\Epic\\EpicGamesLauncher\\Data\\Manifests"
	entries, err := os.ReadDir(epicPath)
	if err != nil {
		return nil
	}

	var manifests []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".item") {
			manifests = append(manifests, filepath.Join(epicPath, e.Name()))
		}
	}
	return manifests
}
