package app

import (
	"log/slog"

	"github.com/wubigork/wubigork/internal/config"
	"github.com/wubigork/wubigork/internal/modelengine"
)

// ── 模型引擎管理 API ─────────────────────────────────────────

// GetEngines 获取所有模型引擎配置（含连接状态）
func (a *App) GetEngines() []modelengine.EngineConfig {
	if a.engineMgr == nil {
		return nil
	}
	return a.engineMgr.GetEngines()
}

// SaveEngine 保存引擎配置（BaseURL、Enabled 等）
func (a *App) SaveEngine(cfg modelengine.EngineConfig) error {
	if a.engineMgr == nil {
		return errNoEngineMgr
	}
	return a.engineMgr.SaveEngine(cfg)
}

// TestEngineConnection 测试引擎连接并拉取模型列表
func (a *App) TestEngineConnection(engineID string) (*modelengine.EngineStatus, error) {
	if a.engineMgr == nil {
		return nil, errNoEngineMgr
	}
	return a.engineMgr.TestConnection(a.ctx, engineID)
}

// RefreshEngineModels 刷新引擎的模型列表
func (a *App) RefreshEngineModels(engineID string) ([]modelengine.ModelInfo, error) {
	if a.engineMgr == nil {
		return nil, errNoEngineMgr
	}
	return a.engineMgr.RefreshModels(a.ctx, engineID)
}

func (a *App) SetEngineDefaultModel(engineID, modelName string) error {
	if a.engineMgr == nil {
		return errNoEngineMgr
	}
	if err := a.engineMgr.SetDefaultModel(engineID, modelName); err != nil {
		return err
	}

	// 同步更新 cfg.Model，确保 xAI 引擎和其他回退路径使用最新模型
	a.cfg.Model = modelName
	if err := config.Save("model", modelName); err != nil {
		slog.Warn("保存模型配置失败", "model", modelName, "error", err)
	}

	a.emit("model-changed", map[string]interface{}{"engine": engineID, "model": modelName})
	return nil
}
// SetActiveEngine 切换活跃的模型引擎并持久化
func (a *App) SetActiveEngine(engineID string) error {
	if a.engineMgr == nil {
		return errNoEngineMgr
	}

	// 验证引擎存在且已启用
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok {
		return &appError{"引擎不存在: " + engineID}
	}
	if !eng.Enabled {
		return &appError{"引擎未启用: " + engineID}
	}

	a.client.SetActiveEngine(engineID)

	// 持久化到配置文件
	if err := config.Save(config.KeyActiveEngineID, engineID); err != nil {
		slog.Warn("保存活跃引擎配置失败", "engine", engineID, "error", err)
	}

	// 通知前端模型已切换
	a.emit("model-changed", map[string]interface{}{"engine": engineID})

	slog.Info("活跃引擎已切换", "engine", engineID)
	return nil
}

// GetActiveEngine 获取当前活跃的模型引擎 ID
func (a *App) GetActiveEngine() string {
	if a.client == nil {
		return "xai"
	}
	return a.client.ActiveEngineID()
}

// GetActiveModel 获取当前活跃的模型名称
func (a *App) GetActiveModel() string {
	if a.engineMgr == nil || a.client == nil {
		return ""
	}
	engineID := a.client.ActiveEngineID()
	model, _ := a.engineMgr.GetDefaultModel(engineID)
	if model == "" && engineID == "xai" {
		return a.cfg.Model
	}
	return model
}

// ── DeepSeek API ─────────────────────────────────────────────

// SetDeepseekKey 设置 DeepSeek API Key
func (a *App) SetDeepseekKey(apiKey string) error {
	if a.engineMgr == nil {
		return errNoEngineMgr
	}
	a.engineMgr.UpdateDeepseekKey(apiKey)
	a.cfg.DeepseekAPIKey = apiKey
	if err := config.Save(config.KeyDeepseekAPIKey, apiKey); err != nil {
		slog.Warn("保存 DeepSeek API Key 失败", "error", err)
		return err
	}
	slog.Info("DeepSeek API Key 已更新")
	return nil
}

// GetDeepseekKeyStatus 获取 DeepSeek API Key 配置状态
func (a *App) GetDeepseekKeyStatus() map[string]interface{} {
	hasKey := a.cfg.DeepseekAPIKey != ""
	masked := ""
	if hasKey {
		k := a.cfg.DeepseekAPIKey
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
