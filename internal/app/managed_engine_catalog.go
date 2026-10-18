package app

// managed_engine_catalog.go — 天璇 / Strata 的模型目录层（参照 Herdsman）。
//
// 外部引擎进程一次只服务一个模型，但模型中心需要完整目录：
// 每个档位都是一条模型记录，运行中的档位标 running，其余标 stopped。
// 这样「语言模型」页和「功能绑定」都能像 Herdsman 一样逐模型选择/启动，
// 而不是被 /v1/models 当前只返回一条所限制。

import (
	"strings"
	"time"

	"github.com/gaea/gaea/internal/modelengine"
)

func managedCatalogModels(profiles []engineProfileOption, served []string, activeProfile string) []modelengine.ModelInfo {
	activeID := ""
	if len(served) > 0 {
		activeID = served[0]
	}
	out := make([]modelengine.ModelInfo, 0, len(profiles))
	for _, p := range profiles {
		id := p.ModelID
		status := "stopped"
		if p.ID == activeProfile {
			status = "running"
			if activeID != "" {
				id = activeID
			}
		}
		out = append(out, modelengine.ModelInfo{
			ID:     id,
			Name:   p.Label,
			Kind:   "llm",
			Status: status,
		})
	}
	return out
}

// syncManagedEngineCatalog 把完整档位目录写入引擎模型列表；只改 Models，
// 不改 DefaultModel（启动引擎不等于替用户绑定模型）。
func (c *core) syncManagedEngineCatalog(engineID string, profiles []engineProfileOption, served []string, activeProfile string) {
	if c == nil || c.engineMgr == nil || len(profiles) == 0 {
		return
	}
	enabled := true
	if eng, ok := c.engineMgr.GetEngine(engineID); ok {
		enabled = eng.Enabled
	}
	_ = c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:      engineID,
		Enabled: enabled,
		Models:  managedCatalogModels(profiles, served, activeProfile),
	})
}

// managedEngineSpec 返回引擎的档位目录与当前 /v1/models 探活结果。
func managedEngineSpec(engineID string) ([]engineProfileOption, []string, bool) {
	switch engineID {
	case "tianxuan":
		served, ok := tianxuanProbe()
		return tianxuanProfiles, served, ok
	case "strata":
		served, ok := strataProbe()
		return strataProfiles, served, ok
	}
	return nil, nil, false
}

func managedEngineProfileFromModels(engineID string, served []string) string {
	switch engineID {
	case "tianxuan":
		return tianxuanProfileFromModels(served)
	case "strata":
		return strataProfileFromModels(served)
	}
	return ""
}

// managedProfileForModel 按模型 id（实际 served id 或档位 id）找档位。
func managedProfileForModel(profiles []engineProfileOption, model string) (engineProfileOption, bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return engineProfileOption{}, false
	}
	for _, p := range profiles {
		if p.ID == model || strings.EqualFold(p.ModelID, model) {
			return p, true
		}
	}
	return engineProfileOption{}, false
}

func managedEngineStatus(engineID string, connected bool, modelCount int, latencyMs int64, errMsg string) modelengine.EngineStatus {
	return modelengine.EngineStatus{
		ID:          engineID,
		Connected:   connected,
		ModelCount:  modelCount,
		Error:       errMsg,
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		LatencyMs:   latencyMs,
	}
}

func (c *core) markManagedEngineStopped(engineID string, profiles []engineProfileOption) {
	c.syncManagedEngineCatalog(engineID, profiles, nil, "")
	if c.engineMgr != nil {
		c.engineMgr.SetStatus(engineID, managedEngineStatus(engineID, false, 0, 0, "引擎未运行"))
	}
}

// App 级包装：单元测试里的 &App{} 可能没有 core，promoted method 直接调用
// 会在空指针上解引用；探活/状态方法必须能在这种零值 App 上安全运行。
func (a *App) syncManagedCatalog(engineID string, profiles []engineProfileOption, served []string, activeProfile string) {
	if a == nil || a.core == nil {
		return
	}
	a.core.syncManagedEngineCatalog(engineID, profiles, served, activeProfile)
}

func (a *App) markManagedCatalogStopped(engineID string, profiles []engineProfileOption) {
	if a == nil || a.core == nil {
		return
	}
	a.core.markManagedEngineStopped(engineID, profiles)
}
