package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AppVersion 应用版本（与 wails.json productVersion / versioninfo.rc 三处对齐；
// 由 scripts/sync-version.ps1 统一维护，勿手工修改）
const AppVersion = "4.59.0"

// GetAppInfo 返回应用信息与最近更新日志（供设置中心「更新信息」展示）
func (a *core) GetAppInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":     "gaea",
		"version":  AppVersion,
		"tagline":  "多功能 AI 助手 · 本机单用户",
		"releases": parseChangelog(8),
	}
}

// parseChangelog 解析 CHANGELOG.md 最近 N 个版本块。
// 块格式：## v1.6.4「设置瘦身」(2026-08-02) + > 简介 + - 要点
func parseChangelog(limit int) []map[string]string {
	data, err := os.ReadFile(findChangelogPath())
	if err != nil {
		return []map[string]string{}
	}
	content := string(data)
	re := regexp.MustCompile(`(?m)^## (v\d+\.\d+\.\d+)「([^」]+)」\((\d{4}-\d{2}-\d{2})\)`)
	matches := re.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return []map[string]string{}
	}

	releases := make([]map[string]string, 0, limit)
	for i, m := range matches {
		if i >= limit {
			break
		}
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		ver := content[m[2]:m[3]]
		title := content[m[4]:m[5]]
		date := content[m[6]:m[7]]
		body := content[m[1]:end]

		var intro []string
		var points []string
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, ">"):
				intro = append(intro, strings.TrimSpace(strings.TrimPrefix(line, ">")))
			case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "-"):
				points = append(points, strings.TrimSpace(strings.TrimPrefix(line, "-")))
			}
		}
		releases = append(releases, map[string]string{
			"version": ver,
			"title":   title,
			"date":    date,
			"intro":   strings.Join(intro, " "),
			"points":  strings.Join(points, "\n"),
		})
	}
	return releases
}

// findChangelogPath 定位 CHANGELOG.md：cwd → 可执行文件目录向上逐级
func findChangelogPath() string {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 4; i++ {
			candidates = append(candidates, filepath.Join(dir, "CHANGELOG.md"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	// cwd 优先（与函数注释一致）：开发/测试时 cwd 下若有 CHANGELOG.md 应最先命中，
	// 否则 ci 把 TMP 指到仓库内时，测试二进制会沿上级目录误命中仓库根的 CHANGELOG.md（E21 测试隔离）。
	candidates = append([]string{"CHANGELOG.md"}, candidates...)
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "CHANGELOG.md"
}
