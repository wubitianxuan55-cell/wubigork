package app

import (
	"github.com/gaea/gaea/internal/project"
)

// ── 项目管理 ─────────────────────────────────────────────────

// CreateProject 新建小说项目
func (a *App) CreateProject(dir, title, genre, style string) (map[string]interface{}, error) {
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
