// Package whisper — investigation_runner.go
// 100% 对齐 ackem desktop-agent/investigation/runInvestigation.ts + findingsMerge.ts
// 调查完整管线：意图路由→清单生成→逐步收集→综合→交付
package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ─── 调查运行器 ──────────────────────────────────────────────────

// InvestigationRun 一次调查运行
type InvestigationRun struct {
	ID             string                `json:"id"`
	Intent         InvestigationIntent   `json:"intent"`
	Checklist      InvestigationChecklist `json:"checklist"`
	Findings       []InvestigationFinding `json:"findings"`
	CompletedSteps int                   `json:"completedSteps"`
	Status         string                `json:"status"` // running/synthesizing/done
}

// NewInvestigationRun 创建调查运行

// CollectStep 执行一个调查步骤
func (run *InvestigationRun) CollectStep() []InvestigationFinding {
	if run.CompletedSteps >= len(run.Checklist.Steps) {
		run.Status = "synthesizing"
		return nil
	}

	step := run.Checklist.Steps[run.CompletedSteps]
	findings := run.executeStep(step, run.CompletedSteps)
	run.Findings = append(run.Findings, findings...)
	run.CompletedSteps++

	if run.CompletedSteps >= len(run.Checklist.Steps) {
		run.Status = "synthesizing"
	}

	return findings
}

// executeStep 执行单个调查步骤
func (run *InvestigationRun) executeStep(step string, stepIdx int) []InvestigationFinding {
	var findings []InvestigationFinding

	switch {
	case strings.Contains(step, "Steam"):
		libs := ParseSteamLibraries()
		for _, lib := range libs {
			findings = append(findings, InvestigationFinding{
				Step: stepIdx + 1, Path: lib,
				Name: filepath.Base(lib), Type: "game", Source: "steam",
			})
		}
	case strings.Contains(step, "Epic"):
		manifests := ParseEpicManifests()
		for _, m := range manifests {
			findings = append(findings, InvestigationFinding{
				Step: stepIdx + 1, Path: m,
				Name: filepath.Base(m), Type: "game", Source: "epic",
			})
		}
	case strings.Contains(step, "桌面"):
		home, _ := os.UserHomeDir()
		desktopDir := filepath.Join(home, "Desktop")
		entries, err := os.ReadDir(desktopDir)
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				path := filepath.Join(desktopDir, name)
				if e.IsDir() && IsGameDirectory(path) {
					findings = append(findings, InvestigationFinding{
						Step: stepIdx + 1, Path: path,
						Name: name, Type: "game", Source: "desktop",
					})
				} else if !e.IsDir() && isDocumentFile(name) {
					findings = append(findings, InvestigationFinding{
						Step: stepIdx + 1, Path: path,
						Name: name, Type: "document", Source: "desktop",
					})
				}
			}
		}
	case strings.Contains(step, "文档文件夹"):
		home, _ := os.UserHomeDir()
		docsDir := filepath.Join(home, "Documents")
		entries, err := os.ReadDir(docsDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && isDocumentFile(e.Name()) {
					findings = append(findings, InvestigationFinding{
						Step: stepIdx + 1, Path: filepath.Join(docsDir, e.Name()),
						Name: e.Name(), Type: "document", Source: "documents",
					})
				}
			}
		}
	case strings.Contains(step, "下载文件夹"):
		home, _ := os.UserHomeDir()
		dlDir := filepath.Join(home, "Downloads")
		entries, err := os.ReadDir(dlDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && isDocumentFile(e.Name()) {
					findings = append(findings, InvestigationFinding{
						Step: stepIdx + 1, Path: filepath.Join(dlDir, e.Name()),
						Name: e.Name(), Type: "document", Source: "downloads",
					})
				}
			}
		}
	case strings.Contains(step, "Program Files"):
		paths := []string{"C:\\Program Files", "C:\\Program Files (x86)"}
		home, _ := os.UserHomeDir()
		paths = append(paths, filepath.Join(home, "AppData", "Local", "Programs"))

		for _, pf := range paths {
			entries, err := os.ReadDir(pf)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					path := filepath.Join(pf, e.Name())
					if IsGameDirectory(path) {
						findings = append(findings, InvestigationFinding{
							Step: stepIdx + 1, Path: path,
							Name: e.Name(), Type: "game", Source: "program_files",
						})
					}
				}
			}
		}
	default:
		// 通用目录扫描
		home, _ := os.UserHomeDir()
		scanDirs := []string{
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Documents"),
		}
		for _, dir := range scanDirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				findings = append(findings, InvestigationFinding{
					Step: stepIdx + 1, Path: filepath.Join(dir, e.Name()),
					Name: e.Name(), Type: "generic", Source: dir,
				})
			}
		}
	}

	return findings
}

// ─── 发现合并 ────────────────────────────────────────────────────

// MergeFindings 去重合并调查发现
func MergeFindings(findings []InvestigationFinding) []InvestigationFinding {
	seen := make(map[string]bool)
	var merged []InvestigationFinding

	for _, f := range findings {
		key := strings.ToLower(f.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, f)
	}
	return merged
}

// SynthesizeFindings 综合调查发现为自然语言报告
func SynthesizeFindings(findings []InvestigationFinding, question string) string {
	merged := MergeFindings(findings)
	if len(merged) == 0 {
		return "未找到相关信息。"
	}

	var sb strings.Builder
	sb.WriteString("根据扫描结果：\n\n")

	games := filterByType(merged, "game")
	docs := filterByType(merged, "document")

	if len(games) > 0 {
		sb.WriteString("**已安装的游戏：**\n")
		for _, g := range games {
			sb.WriteString("· " + g.Name + "\n")
		}
		sb.WriteString("\n")
	}

	if len(docs) > 0 {
		sb.WriteString("**找到的文档：**\n")
		for _, d := range docs {
			sb.WriteString("· " + filepath.Base(d.Name) + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("共扫描到 %d 个游戏、%d 个文档。", len(games), len(docs)))

	return sb.String()
}

func filterByType(findings []InvestigationFinding, t string) []InvestigationFinding {
	var result []InvestigationFinding
	for _, f := range findings {
		if f.Type == t {
			result = append(result, f)
		}
	}
	return result
}
