package app

import (
	"log/slog"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
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
	// 本地 TTS 引擎：先确保服务在起（幂等），刚拉起时短暂等待，避免“测试连接”必失败
	if engineID == "cosyvoice" {
		c.ensureLocalTTSService(engineID)
		for i := 0; i < 4 && !ttsReady(engineID); i++ {
			time.Sleep(2 * time.Second)
		}
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

	// 仅 xAI 引擎的默认模型同步为全局回退模型（cfg.Model）；
	// 其他引擎（本地/第三方）不再污染全局配置，避免陈旧模型名发给活跃引擎（E03 同类问题）。
	if engineID == "xai" {
		c.cfg.Model = modelName
		if err := config.Save(config.KeyModel, modelName); err != nil {
			slog.Warn("保存模型配置失败", "model", modelName, "error", err)
		}
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

	// 切换活跃引擎时同步全局回退模型，确保旧路径（xAI 回退等）使用该引擎的默认模型
	if eng.DefaultModel != "" {
		c.cfg.Model = eng.DefaultModel
		if err := config.Save(config.KeyModel, eng.DefaultModel); err != nil {
			slog.Warn("保存全局模型配置失败", "model", eng.DefaultModel, "error", err)
		}
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
	enc, err := secure.EncryptString(apiKey)
	if err != nil {
		return &appError{"API Key 加密失败: " + err.Error()}
	}
	c.engineMgr.UpdateDeepseekKey(apiKey)
	c.cfg.DeepseekAPIKey = enc
	if err := config.Save(config.KeyDeepseekAPIKey, enc); err != nil {
		slog.Warn("保存 DeepSeek API Key 失败", "error", err)
		return err
	}
	slog.Info("DeepSeek API Key 已更新")
	return nil
}

// GetDeepseekKeyStatus 获取 DeepSeek API Key 配置状态
func (c *core) GetDeepseekKeyStatus() map[string]interface{} {
	hasKey, masked := maskKeyStatus(c.cfg.DeepseekAPIKey)
	return map[string]interface{}{
		"configured": hasKey,
		"masked":     masked,
	}
}

// ── OpenCode Go API ─────────────────────────────────────────

// SetOpencodeGoKey 设置 OpenCode Go API Key
func (c *core) SetOpencodeGoKey(apiKey string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	enc, err := secure.EncryptString(apiKey)
	if err != nil {
		return &appError{"API Key 加密失败: " + err.Error()}
	}
	c.engineMgr.UpdateOpencodeKey(apiKey)
	c.cfg.OpenCodeGoAPIKey = enc
	if err := config.Save(config.KeyOpencodeGoAPIKey, enc); err != nil {
		slog.Warn("保存 OpenCode Go API Key 失败", "error", err)
		return err
	}
	slog.Info("OpenCode Go API Key 已更新")
	return nil
}

// GetOpencodeGoKeyStatus 获取 OpenCode Go API Key 配置状态
func (c *core) GetOpencodeGoKeyStatus() map[string]interface{} {
	hasKey, masked := maskKeyStatus(c.cfg.OpenCodeGoAPIKey)
	return map[string]interface{}{
		"configured": hasKey,
		"masked":     masked,
	}
}

// SetOpencodeZenKey 设置 OpenCode Zen API Key
func (c *core) SetOpencodeZenKey(apiKey string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	enc, err := secure.EncryptString(apiKey)
	if err != nil {
		return &appError{"API Key 加密失败: " + err.Error()}
	}
	c.engineMgr.UpdateOpencodeZenKey(apiKey)
	c.cfg.OpenCodeZenAPIKey = enc
	if err := config.Save(config.KeyOpencodeZenAPIKey, enc); err != nil {
		slog.Warn("保存 OpenCode Zen API Key 失败", "error", err)
		return err
	}
	slog.Info("OpenCode Zen API Key 已更新")
	return nil
}

// GetOpencodeZenKeyStatus 获取 OpenCode Zen API Key 配置状态
func (c *core) GetOpencodeZenKeyStatus() map[string]interface{} {
	hasKey, masked := maskKeyStatus(c.cfg.OpenCodeZenAPIKey)
	return map[string]interface{}{
		"configured": hasKey,
		"masked":     masked,
	}
}

// maskKeyStatus 解密持久化的密钥并返回脱敏展示。
// 存储值可能为 DPAPI 密文或旧版明文；解密失败时保守显示 ****。
func maskKeyStatus(enc string) (hasKey bool, masked string) {
	if enc == "" {
		return false, ""
	}
	dec, err := secure.DecryptString(enc)
	if err != nil || dec == "" {
		return true, "****"
	}
	if len(dec) > 8 {
		return true, dec[:4] + "****" + dec[len(dec)-4:]
	}
	return true, "****"
}

// GetModelCallStats 获取模型调用统计汇总（按引擎/模型维度）。
func (c *core) GetModelCallStats() modelengine.ModelStatsSummary {
	if c.engineMgr == nil {
		return modelengine.ModelStatsSummary{}
	}
	return c.engineMgr.GetModelCallStats()
}

// ResetModelCallStats 清空全部模型调用统计。
func (c *core) ResetModelCallStats() {
	if c.engineMgr == nil {
		return
	}
	c.engineMgr.ResetModelCallStats()
	slog.Info("模型调用统计已重置")
}

// ── 错误 ──────────────────────────────────────────────────────

var errNoEngineMgr = &appError{"模型引擎管理器未初始化"}

type appError struct{ msg string }

func (e *appError) Error() string { return e.msg }
