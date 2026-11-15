package app

import (
	"path/filepath"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/skill"
)

// ── Skill ────────────────────────────────────────────────────

// ListSkills 列出所有 Skill
func (a *App) ListSkills() []map[string]interface{} {
	// 审计 P1 AP8-05：懒构造走 sync.Once——此前并发首调会双重构造并互相覆盖
	// skillLoader 指针（绑定方法体不得裸写共享字段）。
	a.skillLoaderOnce.Do(func() {
		dir := ""
		if a.cfg != nil {
			dir = a.cfg.ResourceDir
		}
		a.skillLoader = skill.NewLoader(filepath.Join(dir, "skills"))
	})
	skills := a.skillLoader.List()
	result := make([]map[string]interface{}, len(skills))
	for i, s := range skills {
		result[i] = map[string]interface{}{
			"name":        s.Name,
			"description": s.Description,
			"appliesTo":   s.AppliesTo,
			"version":     s.Version,
		}
	}
	return result
}

// ── 配置 ──────────────────────────────────────────────────────

// GetConfig 返回当前配置
func (a *App) GetConfig() map[string]string {
	return map[string]string{
		"model":     config.GetModelMem(a.cfg),
		"baseURL":   a.cfg.XaiAPIBaseURL,
		"tokenPath": a.cfg.TokenStorePath,
	}
}
