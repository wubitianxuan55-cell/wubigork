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

// searchDirectory 搜索目录
func searchDirectory(root, query string, limit int) []InvestigationFinding {
	q := strings.ToLower(query)
	var findings []InvestigationFinding

	// 简化：仅扫描一级子目录
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if len(findings) >= limit {
			break
		}
		if strings.Contains(strings.ToLower(e.Name()), q) {
			path := filepath.Join(root, e.Name())
			findings = append(findings, InvestigationFinding{
				Source: "directory_scan",
				Path:   path,
				Match:  e.Name(),
			})
		}
	}
	return findings
}
