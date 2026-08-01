// Package app — 功能级模型绑定
//
// 各功能板块（聊天/轻语/小说/办公）可独立指定 LLM 引擎 + 模型，
// 持久化到 ~/.gaea_config.json（重启不丢）；未绑定则用全局激活引擎。
// 满足本机单用户定位：一次设置好，各窗口直接使用，无需重启。
package app

import (
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/config"
)

// featureModelKeys 功能 → 配置键映射
func featureModelKeys(feature string) (engineKey, modelKey string, ok bool) {
	switch feature {
	case "chat":
		return config.KeyFuncChatEngine, config.KeyFuncChatModel, true
	case "whisper":
		return config.KeyFuncWhisperEngine, config.KeyFuncWhisperModel, true
	case "novel":
		return config.KeyFuncNovelEngine, config.KeyFuncNovelModel, true
	case "office":
		return config.KeyFuncOfficeEngine, config.KeyFuncOfficeModel, true
	}
	return "", "", false
}

// featureModel 读取功能绑定的 (engine, model)，空 = 用全局激活
func (c *core) featureModel(feature string) (engine, model string) {
	switch feature {
	case "chat":
		return c.cfg.FuncChatEngine, c.cfg.FuncChatModel
	case "whisper":
		return c.cfg.FuncWhisperEngine, c.cfg.FuncWhisperModel
	case "novel":
		return c.cfg.FuncNovelEngine, c.cfg.FuncNovelModel
	case "office":
		return c.cfg.FuncOfficeEngine, c.cfg.FuncOfficeModel
	}
	return "", ""
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

	switch feature {
	case "chat":
		c.cfg.FuncChatEngine, c.cfg.FuncChatModel = engineID, modelName
	case "whisper":
		c.cfg.FuncWhisperEngine, c.cfg.FuncWhisperModel = engineID, modelName
	case "novel":
		c.cfg.FuncNovelEngine, c.cfg.FuncNovelModel = engineID, modelName
	case "office":
		c.cfg.FuncOfficeEngine, c.cfg.FuncOfficeModel = engineID, modelName
	}
	if err := config.Save(engineKey, engineID); err != nil {
		slog.Warn("保存功能引擎失败", "feature", feature, "error", err)
	}
	if err := config.Save(modelKey, modelName); err != nil {
		slog.Warn("保存功能模型失败", "feature", feature, "error", err)
	}
	c.emit("feature-model-changed", map[string]interface{}{"feature": feature, "engine": engineID, "model": modelName})
	slog.Info("功能模型已绑定", "feature", feature, "engine", engineID, "model", modelName)
	return nil
}

// GetFeatureModel 获取功能绑定的模型（空 = 全局激活）
func (c *core) GetFeatureModel(feature string) map[string]string {
	engine, model := c.featureModel(feature)
	return map[string]string{"engine": engine, "model": model}
}
