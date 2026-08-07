package app

import "encoding/json"

// routeModel 解析功能域生效的 (engine, model, source)。
// 降级链：功能绑定 → 全局活跃 → 首个可用引擎。
// source: feature | global | fallback（供前端与诊断展示）。
func (c *core) routeModel(feature string) (engine, model, source string) {
	// 1. 功能绑定（引擎必须存在且启用）
	if eng, m := c.cfg.GetFeatureModel(feature); eng != "" && m != "" {
		if e, ok := c.engineMgr.GetEngine(eng); ok && e.Enabled {
			c.emitModelRoute(feature, eng, m, "feature")
			return eng, m, "feature"
		}
	}
	// 2. 全局活跃引擎（存在且启用）
	if eng := c.GetActiveEngine(); eng != "" {
		if e, ok := c.engineMgr.GetEngine(eng); ok && e.Enabled {
			model = c.GetActiveModel()
			if model == "" {
				if dm, err := c.engineMgr.GetDefaultModel(eng); err == nil {
					model = dm
				}
			}
			if model == "" && eng == "xai" {
				model = c.cfg.Model
			}
			c.emitModelRoute(feature, eng, model, "global")
			return eng, model, "global"
		}
	}
	// 3. 首个启用引擎兜底
	for _, e := range c.engineMgr.GetEngines() {
		if !e.Enabled {
			continue
		}
		m, _ := c.engineMgr.GetDefaultModel(e.ID)
		c.emitModelRoute(feature, e.ID, m, "fallback")
		return e.ID, m, "fallback"
	}
	c.emitModelRoute(feature, "", "", "")
	return "", "", ""
}

// emitModelRoute 发布路由决策（ctx 未启动时跳过，测试安全）。
func (c *core) emitModelRoute(feature, engine, model, source string) {
	if c.ctx == nil {
		return
	}
	c.emit("model.route", map[string]interface{}{
		"feature": feature, "engine": engine, "model": model, "source": source,
	})
}

// GetModelRoute 返回功能域当前生效路由（Wails 绑定，前端模型中心展示用）。
func (a *App) GetModelRoute(feature string) (string, error) {
	eng, model, source := a.routeModel(feature)
	b, _ := json.Marshal(map[string]string{
		"feature": feature, "engine": eng, "model": model, "source": source,
	})
	return string(b), nil
}
