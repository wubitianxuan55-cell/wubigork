// engine_connect.go 引擎连接域：连接测试（TestConnection）、模型刷新
// （RefreshModels）与 chat completions URL/Key 装配（BuildChatURL）。
// GLM 无 /models 端点，连接测试与巡检同源走 glmPing 最小 chat 探测（engine_models.go）。
package modelengine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

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

	start := time.Now()
	var models []ModelInfo
	var err error
	if engine.Type == EngineGLM {
		// 智谱官方无 /models 端点（docs.bigmodel.cn 仅有 chat/completions 等）：
		// Key 校验走最小 chat ping，模型目录用官方文档锚定的静态清单。
		err = m.glmPing(ctx, engine)
		models = m.glmCatalogModels()
	} else {
		models, err = m.fetchModels(ctx, engine)
	}
	status.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		status.Connected = false
		status.Error = err.Error()
		// 缓存失败状态（前端可看到上次连接错误）
		m.mu.Lock()
		engine.Status = *status
		m.mu.Unlock()
		m.saveState()
		slog.Warn("模型引擎连接失败", "engine", engineID, "error", err)
		return status, nil // 不返回 error，让前端展示状态
	}

	status.Connected = true
	status.ModelCount = len(models)

	// 更新引擎的模型列表
	m.mu.Lock()
	engine.Models = models
	engine.Status = *status
	if engine.DefaultModel == "" && len(models) > 0 {
		engine.DefaultModel = models[0].ID
	}
	m.mu.Unlock()
	m.saveState()

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

	start := time.Now()
	models, err := m.fetchModels(ctx, engine)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		// 本地引擎停掉后 /models 会直接拒连；刷新失败必须同步状态，否则模型中心
		// 仍显示上一次的「连接正常」，模型卡也继续可点。
		m.mu.Lock()
		engine.Status = EngineStatus{
			ID:          engineID,
			Connected:   false,
			Error:       err.Error(),
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
			LatencyMs:   latencyMs,
		}
		m.mu.Unlock()
		m.saveState()
		return nil, err
	}

	m.mu.Lock()
	engine.Models = models
	engine.Status = EngineStatus{
		ID:          engineID,
		Connected:   true,
		ModelCount:  len(models),
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		LatencyMs:   latencyMs,
	}
	m.mu.Unlock()
	m.saveState()

	return models, nil
}

// buildChatURL 根据引擎类型构建 chat completions URL。
// 每次成功解析端点即记一次「真实使用」（activity 记账，独立小锁）——空闲自动
// 退出机制据此判定外部引擎是否在忙；探活/拉模型列表不走这里，不会续命。
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

	chatURL := strings.TrimRight(engine.BaseURL, "/") + "/chat/completions"
	m.TouchEngine(engineID)
	return chatURL, m.apiKeyForLocked(engine), nil
}

// EngineAPIKey 返回引擎的 API Key（明文，仅内存；本地引擎恒空串）。
// Key 统一存 Manager（xai/deepseek/glm/opencode/custom），消费方禁止改读
// EngineConfig.APIKey（下发前端前已清空）。视觉绑定等非 chat 链路按引擎
// 取 Key 的单一入口，与 BuildChatURL 同源判定。
func (m *Manager) EngineAPIKey(engineID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	engine, ok := m.engines[engineID]
	if !ok {
		return ""
	}
	return m.apiKeyForLocked(engine)
}

// apiKeyForLocked 按引擎类型取 Key（调用方须已持 Manager 读/写锁）；
// 自定义引擎 Key 在内存 customKeys——空 Key 原样返回，调用方为空串时
// 省略 Authorization 头。
func (m *Manager) apiKeyForLocked(engine *EngineConfig) string {
	switch engine.Type {
	case EngineXAI:
		return m.xaiKey
	case EngineDeepseek:
		return m.deepseekKey
	case EngineGLM:
		return m.glmKey
	case EngineOpencodeGo:
		return m.opencodeKey
	case EngineOpencodeZen:
		return m.opencodeZenKey
	case EngineCustom:
		return m.customKeys[engine.ID]
	}
	return ""
}
