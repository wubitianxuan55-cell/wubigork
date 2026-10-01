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
		"title":  pm.Meta.Title,
		"genre":  pm.Meta.Genre,
		"style":  pm.Meta.Style,
		"mature": pm.Meta.Mature,
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
		"title":  pm.Meta.Title,
		"genre":  pm.Meta.Genre,
		"style":  pm.Meta.Style,
		"mature": pm.Meta.Mature,
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
		"title":  pm.Meta.Title,
		"genre":  pm.Meta.Genre,
		"style":  pm.Meta.Style,
		"mature": pm.Meta.Mature,
		"path":   pm.Dir,
	}
}

// UpdateProjectMeta 更新当前项目可编辑元信息（标题/题材/文风/成人向档位）。
// mature 取值白名单归一（""/sensual/explicit，非法值拒绝而不是静默洗白——
// 档位挂错书是内容事故）。首个元信息更新绑定：此前 title/genre/style 均为
// 建档即定格，v4.439.0 起书架/创作间可改。
func (a *App) UpdateProjectMeta(title, genre, style, mature string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if normalizeMatureLevel(mature) != mature {
		return fmt.Errorf("成人向档位非法: %q（合法值：空/sensual/explicit）", mature)
	}
	pm.Meta.Title = strings.TrimSpace(title)
	if pm.Meta.Title == "" {
		return fmt.Errorf("书名不能为空")
	}
	pm.Meta.Genre = genre
	pm.Meta.Style = style
	pm.Meta.Mature = mature
	return pm.WriteMeta()
}
