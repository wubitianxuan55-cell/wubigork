package app

import (
	"encoding/json"
	"log/slog"
	"math"
	"strconv"
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
		for i := 0; i < 4 && !c.ttsReady(engineID); i++ {
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

// ── 自定义引擎（A 刀：OpenAI 兼容自定义服务商）────────────────

// AddCustomEngine 创建自定义引擎（OpenAI 兼容服务商），返回 engineID。
// 校验与 slug/ID 规则在 modelengine.Manager；Key 经本层加密落 config
// custom_engine_keys（密文），Manager 内存只持明文。
func (c *core) AddCustomEngine(name string, baseURL string, apiKey string) (string, error) {
	if c.engineMgr == nil {
		return "", errNoEngineMgr
	}
	id, err := c.engineMgr.AddCustomEngine(name, baseURL, apiKey)
	if err != nil {
		return "", err
	}
	if err := c.saveCustomEngineKeys(); err != nil {
		return "", err
	}
	return id, nil
}

// UpdateCustomEngine 更新自定义引擎；apiKey 传空串 = 保留原 Key 不变
// （前端不回显 Key，编辑留空即「不改」，防 Key 无意清空/回显泄漏）。
func (c *core) UpdateCustomEngine(engineID string, name string, baseURL string, apiKey string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	if err := c.engineMgr.UpdateCustomEngine(engineID, name, baseURL, apiKey); err != nil {
		return err
	}
	return c.saveCustomEngineKeys()
}

// RemoveCustomEngine 删除自定义引擎（仅允许 custom- 前缀；内置引擎拒绝）。
func (c *core) RemoveCustomEngine(engineID string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	if err := c.engineMgr.RemoveCustomEngine(engineID); err != nil {
		return err
	}
	return c.saveCustomEngineKeys()
}

// saveCustomEngineKeys 把 Manager 内存中的自定义引擎明文 Key 全量加密后落
// config custom_engine_keys（JSON map：engineID → 密文），并同步 c.cfg 内存副本。
// 全量覆盖写：Add/Update/Remove 三入口共用，config 与 Manager 状态不漂移。
func (c *core) saveCustomEngineKeys() error {
	keys := c.engineMgr.CustomEngineKeys()
	enc := make(map[string]string, len(keys))
	for id, k := range keys {
		if k == "" {
			continue // 空 Key（无鉴权本地服务）不落盘
		}
		e, err := secure.EncryptString(k)
		if err != nil {
			return &appError{"API Key 加密失败: " + err.Error()}
		}
		enc[id] = e
	}
	raw, err := json.Marshal(enc)
	if err != nil {
		return &appError{"自定义引擎 Key 序列化失败: " + err.Error()}
	}
	if err := config.Save(config.KeyCustomEngineKeys, string(raw)); err != nil {
		slog.Warn("保存自定义引擎 Key 失败", "error", err)
		return err
	}
	if c.cfg != nil {
		c.cfg.CustomEngineKeys = enc
	}
	return nil
}

// decryptCustomEngineKeys 把 config 里的密文 Key 库解为明文 map（解密失败的
// 条目丢弃——保守不注入半截 Key），供 Startup 注入 Manager。
func decryptCustomEngineKeys(enc map[string]string) map[string]string {
	out := make(map[string]string, len(enc))
	for id, e := range enc {
		if plain, err := secure.DecryptString(e); err == nil && plain != "" {
			out[id] = plain
		}
	}
	return out
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

// ── GLM (智谱) API ───────────────────────────────────────────

// SetGlmKey 设置 GLM (智谱) API Key
func (c *core) SetGlmKey(apiKey string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	enc, err := secure.EncryptString(apiKey)
	if err != nil {
		return &appError{"API Key 加密失败: " + err.Error()}
	}
	c.engineMgr.UpdateGLMKey(apiKey)
	c.cfg.GLMAPIKey = enc
	if err := config.Save(config.KeyGLMAPIKey, enc); err != nil {
		slog.Warn("保存 GLM API Key 失败", "error", err)
		return err
	}
	slog.Info("GLM API Key 已更新")
	return nil
}

// GetGlmKeyStatus 获取 GLM API Key 配置状态
func (c *core) GetGlmKeyStatus() map[string]interface{} {
	hasKey, masked := maskKeyStatus(c.cfg.GLMAPIKey)
	return map[string]interface{}{
		"configured": hasKey,
		"masked":     masked,
	}
}

// SetGlmEndpoint 切换 GLM 端点家族（std=标准按量付费 / coding=编码套餐额度）。
// 只接受官方双端点常量（modelengine.GLMBaseURL*），不透传自由地址——
// 云端引擎不露地址框防线（v4.9.1）的延伸。
func (c *core) SetGlmEndpoint(family string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	return c.engineMgr.SetGlmEndpoint(family)
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

// ── Model Hub (Unsloth) API ────────────────────────────────

// SetModelHubKey 设置 Unsloth Model Hub API Key（本地引擎；Unsloth 设置 →
// API 创建，sk-unsloth- 开头。与云端引擎 Key 同一安全口径：DPAPI 密文落盘、
// Manager 内存只持明文）。
func (c *core) SetModelHubKey(apiKey string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	enc, err := secure.EncryptString(apiKey)
	if err != nil {
		return &appError{"API Key 加密失败: " + err.Error()}
	}
	c.engineMgr.UpdateModelHubKey(apiKey)
	c.cfg.ModelHubAPIKey = enc
	if err := config.Save(config.KeyModelHubAPIKey, enc); err != nil {
		slog.Warn("保存 Model Hub API Key 失败", "error", err)
		return err
	}
	slog.Info("Model Hub API Key 已更新")
	return nil
}

// GetModelHubKeyStatus 获取 Model Hub API Key 配置状态（脱敏展示）
func (c *core) GetModelHubKeyStatus() map[string]interface{} {
	hasKey, masked := maskKeyStatus(c.cfg.ModelHubAPIKey)
	return map[string]interface{}{
		"configured": hasKey,
		"masked":     masked,
	}
}

// StartModelHubModel 让 Unsloth Studio 加载指定模型（modelID 为
// ollama-manifest:… 引用，来自 Model Hub 引擎刷新出的模型列表）。
// 调用后 Studio 切换当前加载模型，前端刷新即可看到状态变化。
func (c *core) StartModelHubModel(modelID string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}
	return c.engineMgr.StartModelHubModel(c.ctx, modelID)
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

// ── 汇率配置（T6-6.2） ───────────────────────────────────────

// GaeaGetUsdCnyRate 获取美元→人民币汇率（费用估算折算用，默认 7.2）。
// 单一来源：模型中心 stats 的折算口径（engineMgr 持有的内存副本），
// 与 GetModelCallStats().UsdToCny 严格一致。
func (c *core) GaeaGetUsdCnyRate() float64 {
	if c.engineMgr != nil {
		return c.engineMgr.UsdCnyRate()
	}
	if c.cfg != nil && c.cfg.UsdCnyRate > 0 {
		return c.cfg.UsdCnyRate
	}
	return config.DefaultUsdCnyRate
}

// GaeaSetUsdCnyRate 设置美元→人民币汇率：持久化到 ~/.gaea_config.json
// （usd_cny_rate）并即时注入统计折算（summary 与后续 record 立即生效）。
func (c *core) GaeaSetUsdCnyRate(rate float64) error {
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return &appError{"汇率必须为正数"}
	}
	if err := config.Save(config.KeyUsdCnyRate, strconv.FormatFloat(rate, 'f', -1, 64)); err != nil {
		slog.Warn("保存汇率配置失败", "rate", rate, "error", err)
		return err
	}
	if c.cfg != nil {
		c.cfg.UsdCnyRate = rate
	}
	if c.engineMgr != nil {
		c.engineMgr.SetUsdCnyRate(rate)
	}
	slog.Info("美元→人民币汇率已更新", "rate", rate)
	return nil
}

// ── 错误 ──────────────────────────────────────────────────────

var errNoEngineMgr = &appError{"模型引擎管理器未初始化"}

type appError struct{ msg string }

func (e *appError) Error() string { return e.msg }
