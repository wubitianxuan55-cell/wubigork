package app

import (
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/gaea/gaea/internal/config"
)

// routeModel 解析功能域生效的 (engine, model, source)。
// 降级链：功能绑定 → 全局活跃 → 首个可用引擎。
// source: feature | global | fallback（供前端与诊断展示）。
// 全局离线模式（v4.8，offline_mode）开启时只允许本地引擎
// （EngineType.IsLocal：ollama/herdsman/cosyvoice）——云端引擎在各步一律
// 跳过；无本地可用则返回空（调用方按「模型不可用」既有降级路径处理）。
func (c *core) routeModel(feature string) (engine, model, source string) {
	offline := c.cfg != nil && c.cfg.GetOfflineMode()
	// 1. 功能绑定（功能启用 + 引擎必须存在且启用；FeatureModelBar 停用后回退全局）
	if eng, m := c.cfg.GetFeatureModel(feature); c.cfg.GetFeatureModelEnabled(feature) && eng != "" && m != "" {
		if e, ok := c.engineMgr.GetEngine(eng); ok && e.Enabled && !(offline && !e.Type.IsLocal()) {
			c.emitModelRoute(feature, eng, m, "feature")
			return eng, m, "feature"
		}
	}
	// 2. 全局活跃引擎（存在且启用）
	if eng := c.GetActiveEngine(); eng != "" {
		if e, ok := c.engineMgr.GetEngine(eng); ok && e.Enabled && !(offline && !e.Type.IsLocal()) {
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
		if offline && !e.Type.IsLocal() {
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

// routeSensitiveLocal 敏感域（成本/报价）AI 路由（S2-4/D8）：
// 敏感域本地化开关开启时，成本/报价类 AI 操作强制路由本地 Herdsman
// （商务数据不出本机）；herdsman 引擎不可用/停用时回退常规路由
// （功能绑定 → 全局 → 兜底），保证功能可用性。source 标记
// "sensitive-local" 供前端与诊断展示。
func (c *core) routeSensitiveLocal(feature string) (engine, model, source string) {
	if !c.cfg.GetSensitiveLocal() {
		return c.routeModel(feature)
	}
	return c.routeHerdsmanLocal(feature, "sensitive-local")
}

// routeOfficeLocal 办公板块功能级 AI 路由（2026-08-28，本地优先强化）：
// 办公本地优先开关开启时，Word/Excel AI 编辑、资料摘要、知识导入、记忆整理
// 等功能级调用优先路由本地 Herdsman（数据不出本机、省 token）；
// herdsman 不可用/停用时回退常规路由，保证功能可用性。source 标记
// "office-local" 供前端与诊断展示。聊天主 agent（统筹规划）仍按用户绑定的
// 模型走，不受本策略影响——本地小模型不足以承担复杂工具编排。
func (c *core) routeOfficeLocal(feature string) (engine, model, source string) {
	if !c.cfg.GetOfficeLocal() {
		return c.routeModel(feature)
	}
	return c.routeHerdsmanLocal(feature, "office-local")
}

// routeHerdsmanLocal 公共实现：开关已开时，若本地 Herdsman 引擎可用则强制
// 路由本地，否则回退常规路由。source 由调用方指定（sensitive-local /
// office-local），前端与诊断据此区分策略来源。
func (c *core) routeHerdsmanLocal(feature, source string) (engine, model, sourceOut string) {
	if c.engineMgr != nil {
		if eng, ok := c.engineMgr.GetEngine("herdsman"); ok && eng.Enabled && eng.BaseURL != "" {
			m := eng.DefaultModel
			if m == "" {
				if dm, err := c.engineMgr.GetDefaultModel("herdsman"); err == nil {
					m = dm
				}
			}
			if m == "" && len(eng.Models) > 0 {
				m = eng.Models[0].ID
			}
			c.emitModelRoute(feature, "herdsman", m, source)
			return "herdsman", m, source
		}
	}
	return c.routeModel(feature)
}

// GetOfficeLocal 读取办公本地优先开关（Wails 绑定，默认开启）。
func (a *App) GetOfficeLocal() bool {
	return a.cfg.GetOfficeLocal()
}

// SetOfficeLocal 设置办公本地优先开关并持久化（true=办公功能级 AI 调用优先
// 本地 Herdsman；false=按常规路由可回云端）。
func (a *App) SetOfficeLocal(enabled bool) error {
	a.cfg.SetOfficeLocal(enabled)
	if err := config.Save(config.KeyOfficeLocal, strconv.FormatBool(enabled)); err != nil {
		slog.Warn("保存办公本地优先开关失败", "error", err)
		return err
	}
	slog.Info("办公本地优先开关已更新", "enabled", enabled)
	return nil
}

// GetSensitiveLocal 读取敏感域本地化开关（Wails 绑定，默认开启）。
func (a *App) GetSensitiveLocal() bool {
	return a.cfg.GetSensitiveLocal()
}

// GetOfflineMode 读取全局离线模式开关（Wails 绑定，默认关闭；true=所有 AI
// 路由只允许本地引擎，云端一律跳过）。
func (a *App) GetOfflineMode() bool {
	return a.cfg.GetOfflineMode()
}

// SetOfflineMode 设置全局离线模式开关并持久化（true=数据不出本机总闸：
// 功能绑定/全局活跃/兜底三步全滤云端引擎，无本地可用走「模型不可用」降级）。
func (a *App) SetOfflineMode(enabled bool) error {
	a.cfg.SetOfflineMode(enabled)
	if err := config.Save(config.KeyOfflineMode, strconv.FormatBool(enabled)); err != nil {
		slog.Warn("保存全局离线模式开关失败", "error", err)
		return err
	}
	slog.Info("全局离线模式开关已更新", "enabled", enabled)
	return nil
}

// SetSensitiveLocal 设置敏感域本地化开关并持久化（true=成本/报价 AI 走本地
// Herdsman；false=按常规路由可回云端）。
func (a *App) SetSensitiveLocal(enabled bool) error {
	a.cfg.SetSensitiveLocal(enabled)
	if err := config.Save(config.KeySensitiveLocal, strconv.FormatBool(enabled)); err != nil {
		slog.Warn("保存敏感域本地化开关失败", "error", err)
		return err
	}
	slog.Info("敏感域本地化开关已更新", "enabled", enabled)
	return nil
}
