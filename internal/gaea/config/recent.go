package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// maxRecentWorkspaces 限制侧边栏「项目」分组里保留的工作区数量，
// 避免注册表无限膨胀；按最近使用排序，超出的旧项目只留在磁盘上。
const maxRecentWorkspaces = 10

// RecentWorkspacesPath 返回最近工作区注册表路径（用户配置目录 gaea/ 下）。
// 注册表只记录「用户打开过的工作区路径」，供办公侧边栏按项目聚合会话；
// 会话正文仍留在各工作区 .gaea/sessions/ 下，不搬动任何数据。
func RecentWorkspacesPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea", "recent-workspaces.json")
}

// LoadRecentWorkspaces 读取最近工作区列表（旧→新无需保证；调用方负责排序）。
// 文件缺失或损坏都返回 nil，绝不阻塞启动。
func LoadRecentWorkspaces() []string {
	p := RecentWorkspacesPath()
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var out []string
	if json.Unmarshal(b, &out) == nil && len(out) > 0 {
		return out
	}
	return nil
}

// SaveRecentWorkspaces 持久化最近工作区列表（尽力而为）。
func SaveRecentWorkspaces(paths []string) error {
	p := RecentWorkspacesPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(paths)
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

// TouchRecentWorkspace 把工作区置顶到最近列表（去重、截断）并持久化，
// 返回更新后的列表。返回列表与磁盘可能不一致（保存失败时仍返回内存结果）。
func TouchRecentWorkspace(abs string) []string {
	if abs == "" {
		return LoadRecentWorkspaces()
	}
	paths := LoadRecentWorkspaces()
	out := make([]string, 0, maxRecentWorkspaces)
	out = append(out, abs)
	for _, p := range paths {
		if p == abs || p == "" {
			continue
		}
		out = append(out, p)
		if len(out) >= maxRecentWorkspaces {
			break
		}
	}
	_ = SaveRecentWorkspaces(out)
	return out
}
