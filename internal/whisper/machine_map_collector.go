// Package whisper — machine_map_collector.go
// 100% 对齐 ackem desktop-agent/machine-map/collector.ts + gameClassifier.ts
// 机器地图采集器：扫描磁盘建立游戏/文档索引
package whisper

import (
	"os"
	"path/filepath"
	"strings"
)

// ─── 采集结果 ────────────────────────────────────────────────────

// MapCollectResult 采集结果
type MapCollectResult struct {
	Games     []MapEntry `json:"games"`
	Documents []MapEntry `json:"documents"`
	Apps      []MapEntry `json:"apps"`
}

// MapEntry 地图条目
type MapEntry struct {
	Path       string  `json:"path"`
	Name       string  `json:"name"`
	Source     string  `json:"source"`     // steam_common/epic_manifest/start_menu/desktop/program_files
	Confidence float64 `json:"confidence"` // high(0.9)/medium(0.6)/low(0.3)
}

// ─── 采集器 ──────────────────────────────────────────────────────

// CollectMachineMap 全量采集机器地图
func CollectMachineMap() *MapCollectResult {
	result := &MapCollectResult{}

	home, _ := os.UserHomeDir()

	scanDirs := []struct {
		path string
		kind string
	}{
		{"C:\\Program Files", "program_files"},
		{"C:\\Program Files (x86)", "program_files"},
		{filepath.Join(home, "AppData", "Local", "Programs"), "local_programs"},
		{filepath.Join(home, "Desktop"), "desktop"},
		{filepath.Join(home, "Documents"), "documents"},
		{filepath.Join(home, "Downloads"), "downloads"},
	}

	for _, dir := range scanDirs {
		entries, err := os.ReadDir(dir.path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			fullPath := filepath.Join(dir.path, e.Name())

			if dir.kind == "documents" || dir.kind == "downloads" {
				// 文档目录
				if isDocumentFile(e.Name()) {
					result.Documents = append(result.Documents, MapEntry{
						Path: fullPath, Name: e.Name(),
						Source: dir.kind, Confidence: 0.8,
					})
				}
			} else {
				// 程序目录
				if e.IsDir() && IsGameDirectory(fullPath) {
					result.Games = append(result.Games, MapEntry{
						Path: fullPath, Name: e.Name(),
						Source: dir.kind, Confidence: classifyGameConfidence(fullPath, dir.kind),
					})
				}
			}
		}
	}

	// Steam 游戏库
	steamLibs := ParseSteamLibraries()
	for _, lib := range steamLibs {
		result.Games = append(result.Games, MapEntry{
			Path: lib, Name: filepath.Base(lib),
			Source: "steam_common", Confidence: 0.9,
		})
	}

	// 桌面快捷方式
	shortcuts := ParseDesktopShortcuts()
	for _, sc := range shortcuts {
		result.Apps = append(result.Apps, MapEntry{
			Path: sc.Path, Name: sc.Name,
			Source: "desktop_shortcut", Confidence: 0.7,
		})
	}

	return result
}

// ─── 游戏分类器 ──────────────────────────────────────────────────

// gameLikePatterns 游戏特征关键词
var gameLikePatterns = []string{
	"steam", "epic", "origin", "ubisoft", "gog", "battle.net",
	"riot", "blizzard", "ea games", "2k", "bethesda", "activision",
	"square enix", "capcom", "sega", "bandai", "konami",
	"league of legends", "valorant", "genshin", "honkai",
	"minecraft", "terraria", "stardew", "hollow",
	"cyberpunk", "witcher", "skyrim", "fallout",
}

// nonGameDeny 非游戏黑名单
var nonGameDeny = []string{
	"douyin", "wechat", "chrome", "firefox", "edge",
	"vscode", "visual studio", "intellij", "pycharm",
	"nodejs", "python", "java", "golang",
	"office", "adobe", "autodesk",
	"nvidia", "amd", "intel",
	"discord", "slack", "teams", "zoom",
	"microsoft", "windows",
}

// classifyGameConfidence 判断游戏置信度
func classifyGameConfidence(path, source string) float64 {
	name := strings.ToLower(filepath.Base(path))

	// 确定游戏源 → high
	if source == "steam_common" || source == "epic_manifest" {
		return 0.9
	}

	// 路径包含 steamapps/common 或 Epic Games
	lower := strings.ToLower(path)
	if strings.Contains(lower, "steamapps\\common") || strings.Contains(lower, "epic games") {
		return 0.9
	}

	// 非游戏黑名单
	for _, deny := range nonGameDeny {
		if strings.Contains(name, deny) {
			return 0.1
		}
	}

	// 游戏特征匹配
	for _, pattern := range gameLikePatterns {
		if strings.Contains(name, pattern) {
			return 0.7
		}
	}

	return 0.4
}

// isDocumentFile 判断是否为文档文件
func isDocumentFile(name string) bool {
	docExts := []string{".txt", ".md", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".csv", ".json", ".log", ".rtf"}
	ext := strings.ToLower(filepath.Ext(name))
	for _, e := range docExts {
		if ext == e {
			return true
		}
	}
	return false
}
