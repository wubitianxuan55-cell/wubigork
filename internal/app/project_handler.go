package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/project"
)

// ── 项目管理 ─────────────────────────────────────────────────

// CreateProject 新建小说项目。创建目录必须在书架（NovelsDir）内——对齐
// DeleteProject 的 containment 护栏（2026-09-19 审计 P2：此前可在任意位置
// MkdirAll 并写脚手架文件）。
func (a *App) CreateProject(dir, title, genre, style string) (map[string]interface{}, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("路径解析失败: %w", err)
	}
	absNovels, err := filepath.Abs(a.cfg.NovelsDir)
	if err != nil {
		return nil, fmt.Errorf("解析书架目录路径失败: %w", err)
	}
	if !strings.HasPrefix(absDir, absNovels+string(filepath.Separator)) {
		return nil, fmt.Errorf("只能在书架目录下新建项目")
	}
	if _, err := os.Stat(absDir); err == nil {
		return nil, fmt.Errorf("目标目录已存在: %s", dir)
	}
	pm, err := project.Create(dir, title, genre, style, "")
	if err != nil {
		return nil, err
	}
	a.setPM(pm)
	a.initAgents()
	return map[string]interface{}{
		"title": pm.Meta.Title,
		"genre": pm.Meta.Genre,
		"style": pm.Meta.Style,
	}, nil
}

// ═══════════════════════════════════════════════════════════════

// OpenProject 打开已有项目
func (a *App) OpenProject(dir string) (map[string]interface{}, error) {
	pm, err := project.Open(dir)
	if err != nil {
		return nil, err
	}
	a.setPM(pm)
	a.initAgents()
	return map[string]interface{}{
		"title": pm.Meta.Title,
		"genre": pm.Meta.Genre,
		"style": pm.Meta.Style,
	}, nil
}

// CloseProject 关闭当前项目
func (a *App) CloseProject() error {
	// closePM 内部已处理 nil 检查和写锁
	return a.closePM()
}

// GetProjectInfo 获取当前项目信息
func (a *App) GetProjectInfo() map[string]interface{} {
	pm := a.getPM()
	if pm == nil {
		return nil
	}
	return map[string]interface{}{
		"title": pm.Meta.Title,
		"genre": pm.Meta.Genre,
		"style": pm.Meta.Style,
		"path":  pm.Dir,
	}
}
