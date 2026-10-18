package app

// managed_engine_models.go — 天璇 / Strata 的逐模型启停（Herdsman 式模型目录）。
//
// 外部引擎一次只驻留一个档位，所以「启动模型 B」= 若当前不是 B，先停掉当前
// 引擎，再用 B 的档位启动。模型目录里其余档位保持 stopped，供语言模型页和
// 功能绑定选择；默认模型由前端在显式启动模型后设置。

func (a *App) TianxuanModelStart(model string) map[string]interface{} {
	p, ok := managedProfileForModel(tianxuanProfiles, model)
	if !ok {
		if id := tianxuanProfileFromModels([]string{model}); id != "" {
			p, ok = engineProfileByID(tianxuanProfiles, id)
		}
	}
	if !ok {
		return map[string]interface{}{"started": false, "reason": "unknown-model", "model": model}
	}
	if served, running := tianxuanProbe(); running {
		active := tianxuanProfileFromModels(served)
		if active == p.ID {
			a.syncManagedCatalog("tianxuan", tianxuanProfiles, served, active)
			return map[string]interface{}{
				"started":       false,
				"reason":        "already-running",
				"profile":       active,
				"model":         model,
				"served_models": served,
			}
		}
		_ = a.TianxuanEngineStop()
	}
	res := a.tianxuanEngineStart(p.ID)
	res["model"] = model
	if _, ok := res["profile"]; !ok {
		res["profile"] = p.ID
	}
	return res
}

func (a *App) TianxuanModelStop(model string) map[string]interface{} {
	p, ok := managedProfileForModel(tianxuanProfiles, model)
	if !ok {
		if id := tianxuanProfileFromModels([]string{model}); id != "" {
			p, ok = engineProfileByID(tianxuanProfiles, id)
		}
	}
	if !ok {
		return map[string]interface{}{"stopped": false, "reason": "unknown-model", "model": model}
	}
	served, running := tianxuanProbe()
	if !running {
		return map[string]interface{}{"stopped": false, "reason": "not-running", "model": model}
	}
	if active := tianxuanProfileFromModels(served); active != p.ID {
		return map[string]interface{}{"stopped": false, "reason": "not-running", "model": model, "profile": active}
	}
	return a.TianxuanEngineStop()
}

func (a *App) StrataModelStart(model string) map[string]interface{} {
	p, ok := managedProfileForModel(strataProfiles, model)
	if !ok {
		if id := strataProfileFromModels([]string{model}); id != "" {
			p, ok = engineProfileByID(strataProfiles, id)
		}
	}
	if !ok {
		return map[string]interface{}{"started": false, "reason": "unknown-model", "model": model}
	}
	if served, running := strataProbe(); running {
		active := strataProfileFromModels(served)
		if active == p.ID {
			a.syncManagedCatalog("strata", strataProfiles, served, active)
			return map[string]interface{}{
				"started":       false,
				"reason":        "already-running",
				"profile":       active,
				"model":         model,
				"served_models": served,
			}
		}
		_ = a.StrataEngineStop()
	}
	res := a.StrataEngineStart(p.ID)
	res["model"] = model
	if _, ok := res["profile"]; !ok {
		res["profile"] = p.ID
	}
	return res
}

func (a *App) StrataModelStop(model string) map[string]interface{} {
	p, ok := managedProfileForModel(strataProfiles, model)
	if !ok {
		if id := strataProfileFromModels([]string{model}); id != "" {
			p, ok = engineProfileByID(strataProfiles, id)
		}
	}
	if !ok {
		return map[string]interface{}{"stopped": false, "reason": "unknown-model", "model": model}
	}
	served, running := strataProbe()
	if !running {
		return map[string]interface{}{"stopped": false, "reason": "not-running", "model": model}
	}
	if active := strataProfileFromModels(served); active != p.ID {
		return map[string]interface{}{"stopped": false, "reason": "not-running", "model": model, "profile": active}
	}
	return a.StrataEngineStop()
}
