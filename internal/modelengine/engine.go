package modelengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ── 引擎类型 ───────────────────────────────────────────────

// EngineType 模型引擎类型
type EngineType string

const (
	EngineXAI     EngineType = "xai"
	EngineOllama  EngineType = "ollama"
	EngineHerdsman EngineType = "herdsman"
)

// ── 数据结构 ───────────────────────────────────────────────

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
	Status  string `json:"status,omitempty"` // "running" / "stopped" / "unknown"
}

// EngineConfig 引擎配置
type EngineConfig struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Type         EngineType  `json:"type"`
	BaseURL      string      `json:"base_url"`
	APIKey       string      `json:"api_key,omitempty"`
	Enabled      bool        `json:"enabled"`
	DefaultModel string      `json:"default_model"`
	Models       []ModelInfo `json:"models,omitempty"`
}

// EngineStatus 引擎连接状态
type EngineStatus struct {
	ID          string `json:"id"`
	Connected   bool   `json:"connected"`
	ModelCount  int    `json:"model_count"`
	Error       string `json:"error,omitempty"`
	LastChecked string `json:"last_checked,omitempty"`
}

// ── OpenAI 兼容响应结构 ────────────────────────────────────

type modelsListResponse struct {
	Data []struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// ── 引擎管理器 ─────────────────────────────────────────────

// Manager 模型引擎管理器
type Manager struct {
	mu      sync.RWMutex
	engines map[string]*EngineConfig
	xaiKey  string // xAI API key（来自 OAuth token）

	httpClient *http.Client
}

// NewManager 创建引擎管理器
func NewManager(xaiAPIKey string) *Manager {
	m := &Manager{
		engines:    make(map[string]*EngineConfig),
		xaiKey:     xaiAPIKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}

	// 预置三大引擎默认配置
	m.engines["xai"] = &EngineConfig{
		ID:           "xai",
		Name:         "xAI (Grok)",
		Type:         EngineXAI,
		BaseURL:      "https://api.x.ai/v1",
		Enabled:      true,
		DefaultModel: "grok-4.20",
	}
	m.engines["ollama"] = &EngineConfig{
		ID:           "ollama",
		Name:         "Ollama",
		Type:         EngineOllama,
		BaseURL:      "http://localhost:11434/v1",
		Enabled:      false,
		DefaultModel: "",
	}
	m.engines["herdsman"] = &EngineConfig{
		ID:           "herdsman",
		Name:         "Herdsman",
		Type:         EngineHerdsman,
		BaseURL:      "http://localhost:8080/v1",
		Enabled:      false,
		DefaultModel: "",
	}

	return m
}

// UpdateXAIKey 更新 xAI API key
func (m *Manager) UpdateXAIKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.xaiKey = key
}

// GetEngines 获取所有引擎配置
func (m *Manager) GetEngines() []EngineConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]EngineConfig, 0, len(m.engines))
	for _, e := range m.engines {
		cfg := *e
		// 不暴露 API key 到前端
		cfg.APIKey = ""
		if cfg.Models == nil {
			cfg.Models = []ModelInfo{}
		}
		result = append(result, cfg)
	}
	return result
}

// GetEngine 获取单个引擎配置
func (m *Manager) GetEngine(id string) (*EngineConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.engines[id]
	if !ok {
		return nil, false
	}
	cfg := *e
	cfg.APIKey = ""
	if cfg.Models == nil {
		cfg.Models = []ModelInfo{}
	}
	return &cfg, true
}

// SaveEngine 保存引擎配置
func (m *Manager) SaveEngine(cfg EngineConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.engines[cfg.ID]
	if !ok {
		return fmt.Errorf("引擎 %s 不存在", cfg.ID)
	}

	// 保留已有的 models 列表（除非被明确覆盖）
	if len(cfg.Models) > 0 {
		existing.Models = cfg.Models
	}
	if cfg.BaseURL != "" {
		existing.BaseURL = cfg.BaseURL
	}
	if cfg.DefaultModel != "" {
		existing.DefaultModel = cfg.DefaultModel
	}
	// Enabled 由前端控制
	existing.Enabled = cfg.Enabled

	return nil
}

// TestConnection 测试引擎连接并返回模型列表
func (m *Manager) TestConnection(ctx context.Context, engineID string) (*EngineStatus, error) {
	m.mu.RLock()
	engine, ok := m.engines[engineID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("引擎 %s 不存在", engineID)
	}

	status := &EngineStatus{
		ID:          engineID,
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
	}

	models, err := m.fetchModels(ctx, engine)
	if err != nil {
		status.Connected = false
		status.Error = err.Error()
		slog.Warn("模型引擎连接失败", "engine", engineID, "error", err)
		return status, nil // 不返回 error，让前端展示状态
	}

	status.Connected = true
	status.ModelCount = len(models)

	// 更新引擎的模型列表
	m.mu.Lock()
	engine.Models = models
	if engine.DefaultModel == "" && len(models) > 0 {
		engine.DefaultModel = models[0].ID
	}
	m.mu.Unlock()

	return status, nil
}

// RefreshModels 刷新引擎模型列表
func (m *Manager) RefreshModels(ctx context.Context, engineID string) ([]ModelInfo, error) {
	m.mu.RLock()
	engine, ok := m.engines[engineID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("引擎 %s 不存在", engineID)
	}

	models, err := m.fetchModels(ctx, engine)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	engine.Models = models
	m.mu.Unlock()

	return models, nil
}

// fetchModels 从引擎获取模型列表
func (m *Manager) fetchModels(ctx context.Context, engine *EngineConfig) ([]ModelInfo, error) {
	baseURL := strings.TrimRight(engine.BaseURL, "/")
	url := baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// xAI 需要认证
	if engine.Type == EngineXAI && m.xaiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.xaiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 401 && engine.Type == EngineXAI {
			return nil, fmt.Errorf("HTTP 401: 未登录 xAI，请先点击「登录 xAI」获取授权")
		}
		return nil, fmt.Errorf("HTTP %d: 模型列表获取失败", resp.StatusCode)
	}

	var result modelsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	models := make([]ModelInfo, len(result.Data))
	for i, d := range result.Data {
		models[i] = ModelInfo{
			ID:      d.ID,
			OwnedBy: d.OwnedBy,
			Status:  "running",
		}
	}

	return models, nil
}

// SetDefaultModel 设置引擎的默认模型
func (m *Manager) SetDefaultModel(engineID, modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	engine, ok := m.engines[engineID]
	if !ok {
		return fmt.Errorf("引擎 %s 不存在", engineID)
	}

	// 验证模型是否在列表中
	found := false
	for _, mdl := range engine.Models {
		if mdl.ID == modelName {
			found = true
			break
		}
	}
	if !found && len(engine.Models) > 0 {
		return fmt.Errorf("模型 %s 不在引擎 %s 的可用列表中", modelName, engineID)
	}

	engine.DefaultModel = modelName
	slog.Info("设置默认模型", "engine", engineID, "model", modelName)
	return nil
}

// GetDefaultModel 获取指定引擎的默认模型
func (m *Manager) GetDefaultModel(engineID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	engine, ok := m.engines[engineID]
	if !ok {
		return "", fmt.Errorf("引擎 %s 不存在", engineID)
	}
	return engine.DefaultModel, nil
}

// buildChatURL 根据引擎类型构建 chat completions URL
func (m *Manager) BuildChatURL(engineID string) (string, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	engine, ok := m.engines[engineID]
	if !ok {
		return "", "", fmt.Errorf("引擎 %s 不存在", engineID)
	}

	if !engine.Enabled {
		return "", "", fmt.Errorf("引擎 %s 未启用", engineID)
	}

	baseURL := strings.TrimRight(engine.BaseURL, "/")
	chatURL := baseURL + "/chat/completions"

	apiKey := ""
	if engine.Type == EngineXAI {
		apiKey = m.xaiKey
	}

	return chatURL, apiKey, nil
}
