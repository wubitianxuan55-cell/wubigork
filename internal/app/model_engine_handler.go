package app

import (
	"log/slog"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// ── 模型引擎管理 API ─────────────────────────────────────────

// GetEngines 获取所有模型引擎配置（含连接状态）
func (c *core) GetEngines() []modelengine.EngineConfig {
	if c.engineMgr == nil {
		return nil
	}
	return c.engineMgr.GetEngines()
}

// SaveEngine 保存引擎配置（BaseURL、Enabled 等）
func (c *core) SaveEngine(cfg modelengine.EngineConfig) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	return c.engineMgr.SaveEngine(cfg)
}

// TestEngineConnection 测试引擎连接并拉取模型列表
func (c *core) TestEngineConnection(engineID string) (*modelengine.EngineStatus, error) {
	if c.engineMgr == nil {
		return nil, errNoEngineMgr
	}
	return c.engineMgr.TestConnection(c.ctx, engineID)
}

// RefreshEngineModels 刷新引擎的模型列表
func (c *core) RefreshEngineModels(engineID string) ([]modelengine.ModelInfo, error) {
	if c.engineMgr == nil {
		return nil, errNoEngineMgr
	}
	return c.engineMgr.RefreshModels(c.ctx, engineID)
}

func (c *core) SetEngineDefaultModel(engineID, modelName string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	if err := c.engineMgr.SetDefaultModel(engineID, modelName); err != nil {
		return err
	}

	// 同步更新 cfg.Model，确保 xAI 引擎和其他回退路径使用最新模型
	c.cfg.Model = modelName
	if err := config.Save("model", modelName); err != nil {
		slog.Warn("保存模型配置失败", "model", modelName, "error", err)
	}

	c.emit("model-changed", map[string]interface{}{"engine": engineID, "model": modelName})
	return nil
}

// SetActiveEngine 切换活跃的模型引擎并持久化
func (c *core) SetActiveEngine(engineID string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}

	// 验证引擎存在且已启用
	eng, ok := c.engineMgr.GetEngine(engineID)
	if !ok {
		return &appError{"引擎不存在: " + engineID}
	}
	if !eng.Enabled {
		return &appError{"引擎未启用: " + engineID}
	}

	c.client.SetActiveEngine(engineID)

	// 持久化到配置文件
	if err := config.Save(config.KeyActiveEngineID, engineID); err != nil {
		slog.Warn("保存活跃引擎配置失败", "engine", engineID, "error", err)
	}

	// 通知前端模型已切换
	c.emit("model-changed", map[string]interface{}{"engine": engineID})

	slog.Info("活跃引擎已切换", "engine", engineID)
	return nil
}

// GetActiveEngine 获取当前活跃的模型引擎 ID
func (c *core) GetActiveEngine() string {
	if c.client == nil {
		return "xai"
	}
	return c.client.ActiveEngineID()
}

// GetActiveModel 获取当前活跃的模型名称
func (c *core) GetActiveModel() string {
	if c.engineMgr == nil || c.client == nil {
		return ""
	}
	engineID := c.client.ActiveEngineID()
	model, _ := c.engineMgr.GetDefaultModel(engineID)
	if model == "" && engineID == "xai" {
		return c.cfg.Model
	}
	return model
}

// ── DeepSeek API ─────────────────────────────────────────────

// SetDeepseekKey 设置 DeepSeek API Key
func (c *core) SetDeepseekKey(apiKey string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	c.engineMgr.UpdateDeepseekKey(apiKey)
	c.cfg.DeepseekAPIKey = apiKey
	if err := config.Save(config.KeyDeepseekAPIKey, apiKey); err != nil {
		slog.Warn("保存 DeepSeek API Key 失败", "error", err)
		return err
	}
	slog.Info("DeepSeek API Key 已更新")
	return nil
}

// GetDeepseekKeyStatus 获取 DeepSeek API Key 配置状态
func (c *core) GetDeepseekKeyStatus() map[string]interface{} {
	hasKey := c.cfg.DeepseekAPIKey != ""
	masked := ""
	if hasKey {
		k := c.cfg.DeepseekAPIKey
		if len(k) > 8 {
			masked = k[:4] + "****" + k[len(k)-4:]
		} else {
			masked = "****"
		}
	}
	return map[string]interface{}{
		"configured": hasKey,
		"masked":     masked,
	}
}

// ── 错误 ──────────────────────────────────────────────────────

var errNoEngineMgr = &appError{"模型引擎管理器未初始化"}

type appError struct{ msg string }

func (e *appError) Error() string { return e.msg }
