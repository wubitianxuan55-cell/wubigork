// engine_crud.go 引擎配置的查询与保存（确保内置伪模型、列表/单查、
// 部分更新保存、默认模型读写）。GetEngines/GetEngine 统一清空 APIKey 不下发前端。
package modelengine

import (
	"fmt"
	"log/slog"
	"strings"
)

// EnsureModel 确保引擎模型列表中包含指定模型（用于内置伪模型，如 xAI grok-tts），缺失则追加并持久化
func (m *Manager) EnsureModel(engineID, modelID string) {
	m.mu.Lock()
	engine, ok := m.engines[engineID]
	if !ok {
		m.mu.Unlock()
		return
	}
	for _, mdl := range engine.Models {
		if mdl.ID == modelID {
			m.mu.Unlock()
			return
		}
	}
	engine.Models = append(engine.Models, ModelInfo{ID: modelID, OwnedBy: engineID, Kind: ClassifyModelKind(engine.Type, modelID)})
	m.mu.Unlock()
	m.saveState()
}

// GetEngines 获取所有引擎配置
func (m *Manager) GetEngines() []EngineConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]EngineConfig, 0, len(m.engines))
	for _, id := range m.order {
		e, ok := m.engines[id]
		if !ok {
			continue
		}
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
	existing, ok := m.engines[cfg.ID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不存在", cfg.ID)
	}

	// 保留已有的 models 列表（除非被明确覆盖）
	if len(cfg.Models) > 0 {
		existing.Models = cfg.Models
	}
	if cfg.BaseURL != "" {
		u := strings.TrimSpace(cfg.BaseURL)
		if !validBaseURL(u) {
			// 不回显原值——用户曾把 API Key 粘进地址框，回显等于二次泄漏。
			m.mu.Unlock()
			return fmt.Errorf("引擎地址无效：必须以 http:// 或 https:// 开头")
		}
		existing.BaseURL = u
	}
	if cfg.DefaultModel != "" {
		existing.DefaultModel = cfg.DefaultModel
	}
	// Enabled 由前端控制
	existing.Enabled = cfg.Enabled
	// 价目 v1：指针三态合并（nil=不修改；数字=设置，<=0/NaN/Inf=清除），
	// 见 user_price.go mergeUserPrice。
	existing.UserPriceIn = mergeUserPrice(existing.UserPriceIn, cfg.UserPriceIn)
	existing.UserPriceOut = mergeUserPrice(existing.UserPriceOut, cfg.UserPriceOut)
	m.mu.Unlock()

	// 价目可能变化：重建用户价目注册表（费用折算 estimatePrice 消费）
	m.SyncUserPrices()
	m.saveState()
	return nil
}

// SetDefaultModel 设置引擎的默认模型
func (m *Manager) SetDefaultModel(engineID, modelName string) error {
	m.mu.Lock()
	engine, ok := m.engines[engineID]
	if !ok {
		m.mu.Unlock()
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
		m.mu.Unlock()
		return fmt.Errorf("模型 %s 不在引擎 %s 的可用列表中", modelName, engineID)
	}

	engine.DefaultModel = modelName
	m.mu.Unlock()
	m.saveState()
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
