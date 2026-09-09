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

	chatURL := strings.TrimRight(engine.BaseURL, "/") + "/chat/completions"
	var apiKey string
	if engine.Type == EngineXAI {
		apiKey = m.xaiKey
	} else if engine.Type == EngineDeepseek {
		apiKey = m.deepseekKey
	} else if engine.Type == EngineGLM {
		apiKey = m.glmKey
	} else if engine.Type == EngineOpencodeGo {
		apiKey = m.opencodeKey
	} else if engine.Type == EngineOpencodeZen {
		apiKey = m.opencodeZenKey
	} else if engine.Type == EngineModelHub {
		// 本地 Model Hub：Key 由 Unsloth 生成（sk-unsloth-），存 Manager 内存
		// （与 GLMKey/自定义 Key 同口径，不进 EngineConfig.APIKey）。
		apiKey = m.modelhubKey
	} else if engine.Type == EngineCustom {
		// 自定义引擎：Key 在内存 customKeys（已持读锁，直接读 map 防重入死锁）；
		// 空 Key 原样返回，调用方（ai.Client）为空串时省略 Authorization 头。
		apiKey = m.customKeys[engine.ID]
	}

	return chatURL, apiKey, nil
}
