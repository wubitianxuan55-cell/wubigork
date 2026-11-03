package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/pantheon"
)

func main() {
	rootFlag := flag.String("root", "", "众神殿数据根目录；默认 <UserConfigDir>/gaea/pantheon")
	jsonOutput := flag.Bool("json", false, "输出 JSON 而不是 Markdown")
	includeRuns := flag.Bool("details", false, "Markdown 中包含逐场指标")
	minCases := flag.Int("min-cases", 20, "R6 门禁要求的有效议题数")
	strict := flag.Bool("strict", false, "门禁未就绪时返回非零退出码")
	flag.Parse()

	root := *rootFlag
	if root == "" {
		root = filepath.Join(config.DataRoot(), "pantheon")
	}
	sessions, warnings, err := loadSessions(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runs := make([]pantheon.EvaluationRun, 0, len(sessions))
	for _, session := range sessions {
		runs = append(runs, pantheon.EvaluateSession(session))
	}
	report := pantheon.BuildEvaluationReport(runs, *minCases)
	report.Warnings = append(report.Warnings, warnings...)

	if *jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "序列化评测报告:", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		fmt.Print(pantheon.FormatEvaluationMarkdown(report, *includeRuns))
	}
	if *strict && !report.GateReady {
		os.Exit(1)
	}
}

func loadSessions(root string) ([]*pantheon.Session, []string, error) {
	sessionsDir := filepath.Join(root, "sessions")
	if filepath.Base(filepath.Clean(root)) == "sessions" {
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			sessionsDir = filepath.Clean(root)
		}
	}
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("读取众神殿会话目录 %s: %w", sessionsDir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	sessions := make([]*pantheon.Session, 0, len(entries))
	var warnings []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(sessionsDir, entry.Name(), "session.json")
		data, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过不可读会话 %s：%v", entry.Name(), err))
			continue
		}
		var session pantheon.Session
		if err := json.Unmarshal(data, &session); err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过损坏会话 %s：%v", entry.Name(), err))
			continue
		}
		sessions = append(sessions, &session)
	}
	return sessions, warnings, nil
}
