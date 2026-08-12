// Package app — 功能级模型绑定
//
// 各功能板块（聊天/轻语/小说/办公）可独立指定 LLM 引擎 + 模型，
// 持久化到 ~/.gaea_config.json（重启不丢）；未绑定则用全局激活引擎。
// 满足本机单用户定位：一次设置好，各窗口直接使用，无需重启。
package app

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gaea/gaea/internal/config"
)

// featureModelKeys 功能 → 配置键映射
func featureModelKeys(feature string) (engineKey, modelKey string, ok bool) {
	switch feature {
	case "chat":
		return config.KeyFuncChatEngine, config.KeyFuncChatModel, true
	case "whisper":
		// 2.x 聊天/轻语合并：轻语绑定并入聊天，读写同一组键
		return config.KeyFuncChatEngine, config.KeyFuncChatModel, true
	case "novel":
		return config.KeyFuncNovelEngine, config.KeyFuncNovelModel, true
	case "office":
		return config.KeyFuncOfficeEngine, config.KeyFuncOfficeModel, true
	case "gaea":
		return config.KeyFuncGaeaEngine, config.KeyFuncGaeaModel, true
	case "characterlib":
		return config.KeyFuncCharLibEngine, config.KeyFuncCharLibModel, true
	case "routine":
		// 常规任务模型目标：routine_llm 工具默认引擎/模型（不做强制路由，
		// 是否调用由云端 agent 决定；此处只是"入口"的目标配置）
		return config.KeyFuncRoutineEngine, config.KeyFuncRoutineModel, true
	}
	return "", "", false
}

// featureModelEnabledKey 功能 → 启停配置键映射
func featureModelEnabledKey(feature string) (key string, ok bool) {
	switch feature {
	case "chat":
		return config.KeyFuncChatEnabled, true
	case "whisper":
		return config.KeyFuncChatEnabled, true
	case "novel":
		return config.KeyFuncNovelEnabled, true
	case "office":
		return config.KeyFuncOfficeEnabled, true
	case "gaea":
		return config.KeyFuncGaeaEnabled, true
	case "characterlib":
		return config.KeyFuncCharLibEnabled, true
	case "routine":
		return config.KeyFuncRoutineEnabled, true
	}
	return "", false
}

// featureModel 读取功能绑定的 (engine, model)，空 = 用全局激活
func (c *core) featureModel(feature string) (engine, model string) {
	return c.cfg.GetFeatureModel(feature)
}

// SetFeatureModel 设置功能绑定的引擎 + 模型（持久化，重启不丢）
func (c *core) SetFeatureModel(feature, engineID, modelName string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	engineKey, modelKey, ok := featureModelKeys(feature)
	if !ok {
		return &appError{"未知功能: " + feature}
	}

	eng, ok := c.engineMgr.GetEngine(engineID)
	if !ok {
		return &appError{"引擎不存在: " + engineID}
	}
	if !eng.Enabled {
		return &appError{"引擎未启用: " + engineID}
	}
	// 校验模型在引擎可用列表（有列表时）
	if len(eng.Models) > 0 {
		found := false
		for _, m := range eng.Models {
			if m.ID == modelName {
				found = true
				break
			}
		}
		if !found {
			return &appError{fmt.Sprintf("模型 %s 不在引擎 %s 的可用列表", modelName, engineID)}
		}
	}

	c.cfg.SetFeatureModel(feature, engineID, modelName)
	c.cfg.SetFeatureModelEnabled(feature, true) // 绑定即启用
	if err := config.Save(engineKey, engineID); err != nil {
		slog.Warn("保存功能引擎失败", "feature", feature, "error", err)
	}
	if err := config.Save(modelKey, modelName); err != nil {
		slog.Warn("保存功能模型失败", "feature", feature, "error", err)
	}
	enabledKey, _ := featureModelEnabledKey(feature)
	if err := config.Save(enabledKey, "1"); err != nil {
		slog.Warn("保存功能启用状态失败", "feature", feature, "error", err)
	}
	c.emit("feature-model-changed", map[string]interface{}{
		"feature": feature, "engine": engineID, "model": modelName, "enabled": true,
	})
	slog.Info("功能模型已绑定", "feature", feature, "engine", engineID, "model", modelName)
	return nil
}

// SetFeatureModelEnabled 功能级启停（FeatureModelBar 启停语义）：
// 只影响该功能的路由（停用后回退全局），不影响引擎整体启用状态。
func (c *core) SetFeatureModelEnabled(feature string, enabled bool) error {
	key, ok := featureModelEnabledKey(feature)
	if !ok {
		return &appError{"未知功能: " + feature}
	}
	c.cfg.SetFeatureModelEnabled(feature, enabled)
	if err := config.Save(key, strconv.FormatBool(enabled)); err != nil {
		slog.Warn("保存功能启用状态失败", "feature", feature, "error", err)
		return err
	}
	c.emit("feature-model-changed", map[string]interface{}{
		"feature": feature, "enabled": enabled,
	})
	slog.Info("功能模型启停", "feature", feature, "enabled", enabled)
	return nil
}

// GetFeatureModelEnabled 获取功能级启停状态（默认启用）。
func (c *core) GetFeatureModelEnabled(feature string) bool {
	return c.cfg.GetFeatureModelEnabled(feature)
}

// GetFeatureModel 获取功能绑定的模型（空 = 全局激活）
func (c *core) GetFeatureModel(feature string) map[string]string {
	engine, model := c.featureModel(feature)
	return map[string]string{"engine": engine, "model": model}
}

// GetModelMonitor 模型监控：返回已启用引擎的模型列表 + 系统资源。
// isLocal 标记本地模型（herdsman/ollama 占本机资源，预警统计用；云端 xai/deepseek 不算）；
// comfyRunning 标记 ComfyUI 是否运行中（本地大模型加载，预警需计入）。
func (a *App) GetModelMonitor() map[string]interface{} {
	engines := []map[string]interface{}{}
	if a.engineMgr != nil {
		for _, e := range a.engineMgr.GetEngines() {
			if !e.Enabled {
				continue
			}
			engines = append(engines, map[string]interface{}{
				"engine":  e.ID,
				"name":    e.Name,
				"model":   e.DefaultModel,
				"isLocal": isLocalEngine(e.ID),
			})
		}
	}
	comfyRunning := false
	var stats map[string]interface{}
	if a.mediaState != nil {
		comfyRunning = a.mediaState.isComfyUIRunning()
		stats = a.mediaState.GetSystemStats()
	}
	return map[string]interface{}{
		"engines":      engines,
		"stats":        stats,
		"comfyRunning": comfyRunning,
	}
}

// isLocalEngine 判断引擎是否为本地模型（占本机资源，预警统计用）。
// 云端引擎（xai/deepseek 走 API）不加载到本机，不计入模型加载预警。
func isLocalEngine(id string) bool {
	switch id {
	case "herdsman", "ollama", "cosyvoice":
		return true
	}
	return false
}
