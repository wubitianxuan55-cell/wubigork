// Package whisper — desktop_investigation.go
// 100% 对齐 ackem desktop-agent/investigation/
package whisper

import (
	"os"
	"path/filepath"
	"strings"
)

// InvestigationResult 调查结果
type InvestigationResult struct {
	Findings []InvestigationFinding `json:"findings"`
	Summary  string                 `json:"summary"`
}

// InvestigationFinding 单个发现
type InvestigationFinding struct {
	Step   int    `json:"step,omitempty"`
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"` // game/document/app/generic
	Match  string `json:"match,omitempty"`
}

// RunDesktopInvestigation 执行桌面调查
// 扫描常见位置收集信息
func RunDesktopInvestigation(query string, cwd string) *InvestigationResult {
	result := &InvestigationResult{}

	// 检查常见游戏目录
	gameDirs := []string{
		"C:\\Program Files",
		"C:\\Program Files (x86)",
		"C:\\Users\\Public\\Desktop",
	}

	for _, dir := range gameDirs {
		findings := searchDirectory(dir, query, 20)
		result.Findings = append(result.Findings, findings...)
	}

	if len(result.Findings) > 0 {
		var summaries []string
		for _, f := range result.Findings {
			summaries = append(summaries, f.Match)
		}
		result.Summary = strings.Join(summaries, "；")
	} else {
		result.Summary = "未找到相关信息"
	}

	return result
}

// searchDirectory 递归搜索目录（深度上限6，结果上限200）
// 100% 对齐 ackem desktop-agent/adapters/win/executor.ts searchFiles
func searchDirectory(root, query string, limit int) []InvestigationFinding {
	q := strings.ToLower(query)
	var findings []InvestigationFinding
	searchRecursiveDepth(root, q, limit, 0, &findings)
	return findings
}

// searchRecursiveDepth 递归搜索（跳过隐藏目录和系统目录）
func searchRecursiveDepth(dir, query string, limit, depth int, findings *[]InvestigationFinding) {
	if depth > 6 || len(*findings) >= limit {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if len(*findings) >= limit {
			return
		}
		name := e.Name()
		// 跳过隐藏文件/目录
		if strings.HasPrefix(name, ".") {
			continue
		}
		fullPath := filepath.Join(dir, name)
		if e.IsDir() {
			// 跳过系统目录
			lower := strings.ToLower(name)
			if lower == "windows" || lower == "system32" || lower == "node_modules" ||
				lower == ".git" || lower == "__pycache__" || lower == "vendor" {
				continue
			}
			searchRecursiveDepth(fullPath, query, limit, depth+1, findings)
			continue
		}
		if strings.Contains(strings.ToLower(name), query) {
			*findings = append(*findings, InvestigationFinding{
				Source: "directory_scan",
				Path:   fullPath,
				Name:   name,
				Match:  name,
			})
		}
	}
}
