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

// RemoveModel 从引擎模型列表移除指定模型（幂等；目录退役清理用——模型从
// 启动 EnsureModel 名单退役后，持久化 engines.json 里的存量条目由此清掉）。
// 默认模型指向被删模型时一并清空（防悬空默认）；引擎不存在/模型不在列表
// 均为静默 no-op。有变更才落盘。
func (m *Manager) RemoveModel(engineID, modelID string) {
	m.mu.Lock()
	engine, ok := m.engines[engineID]
	if !ok {
		m.mu.Unlock()
		return
	}
	removed := false
	models := engine.Models[:0]
	for _, mdl := range engine.Models {
		if mdl.ID == modelID {
			removed = true
			continue
		}
		models = append(models, mdl)
	}
	if !removed {
		m.mu.Unlock()
		return
	}
	engine.Models = models
	if engine.DefaultModel == modelID {
		engine.DefaultModel = ""
	}
	m.mu.Unlock()
	m.saveState()
}

// modelsContainID 判断模型清单是否包含指定 id（空清单恒 false）。
func modelsContainID(models []ModelInfo, id string) bool {
	for _, mdl := range models {
		if mdl.ID == id {
			return true
		}
	}
	return false
}

// repairDanglingDefaultLocked 修复悬空默认模型：模型清单整体替换（刷新/
// 目录同步/保存覆盖）后，默认模型已不在新清单——模型退役、换库或服务端
// 改名——时接班取新清单首个 llm 档（无 llm 则清空，与 RemoveModel 防悬空
// 口径一致）。仅在悬空时动手：清单内的用户选择（含 stopped 档位，如天璇
// 未启动的另一档）绝不由此改写。调用方须持写锁并负责落盘。
//
// 2026-10-19 真机实锤：herdsman 换装 MTP-r2 权重后旧默认 ...Q4_K_P-2 悬空，
// 清单刷新只换 Models 不修默认——聊天故障转移（failoverTarget 取候选引擎
// default_model 直发）拿到悬空名 → HTTP 404 model_not_found，办公板块整板
// 不可用。app 侧 routeHerdsmanLocal 2026-09-27 已防（功能域），此处补引擎
// 域收口。
func repairDanglingDefaultLocked(engine *EngineConfig, models []ModelInfo) {
	if engine.DefaultModel == "" || modelsContainID(models, engine.DefaultModel) {
		return
	}
	old := engine.DefaultModel
	engine.DefaultModel = ""
	for _, mdl := range models {
		if mdl.Kind == "llm" {
			engine.DefaultModel = mdl.ID
			break
		}
	}
	slog.Warn("引擎默认模型已不在清单（退役/改名），自动接班首个 llm 档",
		"engine", engine.ID, "stale_model", old, "new_model", engine.DefaultModel)
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
		// 目录同步整体覆盖清单（如天璇/Strata 档位收窄）后同样修复悬空默认。
		repairDanglingDefaultLocked(existing, cfg.Models)
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

// SetStatus 只更新最近连接状态缓存（不碰模型列表/默认模型），供 app 层
// 外部单模型引擎（strata/tianxuan）把目录探活结果写回前端。
func (m *Manager) SetStatus(engineID string, status EngineStatus) {
	m.mu.Lock()
	if engine, ok := m.engines[engineID]; ok {
		engine.Status = status
	}
	m.mu.Unlock()
	m.saveState()
}
