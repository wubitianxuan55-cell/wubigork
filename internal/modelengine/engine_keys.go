// engine_keys.go 各服务商 API Key 的更新/读取方法与 GLM 端点家族切换
// （Key 统一存 Manager 内存，不进 EngineConfig.APIKey，消费方经本文件方法取用）。
package modelengine

import (
	"fmt"
	"log/slog"
)

// UpdateXAIKey 更新 xAI API key
func (m *Manager) UpdateXAIKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.xaiKey = key
}

// UpdateDeepseekKey 更新 DeepSeek API key
func (m *Manager) UpdateDeepseekKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deepseekKey = key
}

// UpdateGLMKey 更新智谱 GLM API key
func (m *Manager) UpdateGLMKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.glmKey = key
}

// GLMKey 返回当前 GLM API Key（chat/glmPing 与生图后端装配同源取用；
// Key 存 manager 而非 EngineConfig.APIKey，消费方禁止改读 EngineConfig）。
func (m *Manager) GLMKey() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.glmKey
}

// SetGlmEndpoint 切换 GLM 端点家族：std=标准按量付费 / coding=编码套餐额度
// （官方双端点，见 GLMBaseURL* 常量）。防呆：只接受两个官方常量，不透传
// 自由地址——云端引擎不露地址框防线（v4.9.1）的延伸，杜绝 Key 粘错框类事故。
func (m *Manager) SetGlmEndpoint(family string) error {
	baseURL, ok := map[string]string{"std": GLMBaseURLStd, "coding": GLMBaseURLCoding}[family]
	if !ok {
		return fmt.Errorf("未知 GLM 端点 %q（支持 std=标准 / coding=编码套餐）", family)
	}
	m.mu.Lock()
	eng, exists := m.engines["glm"]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("引擎 glm 不存在")
	}
	eng.BaseURL = baseURL
	m.mu.Unlock()
	m.saveState()
	slog.Info("GLM 端点已切换", "family", family)
	return nil
}

// GlmEndpointFamily 返回当前端点家族（"std"/"coding"；非 coding 地址一律按 std
// 兜底——LoadState 已保证 GLM 地址只会是两个官方常量之一）。
func (m *Manager) GlmEndpointFamily() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if eng, ok := m.engines["glm"]; ok && eng.BaseURL == GLMBaseURLCoding {
		return "coding"
	}
	return "std"
}

// UpdateOpencodeKey 更新 OpenCode Go API key
func (m *Manager) UpdateOpencodeKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.opencodeKey = key
}

// UpdateOpencodeZenKey 更新 OpenCode Zen API key
func (m *Manager) UpdateOpencodeZenKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.opencodeZenKey = key
}

// UpdateModelHubKey 更新 Unsloth Model Hub API key（sk-unsloth- 前缀，
// Unsloth 设置 → API 创建；本地端点每次请求都必须带 Bearer）。
func (m *Manager) UpdateModelHubKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modelhubKey = key
}

// ModelHubKeyConfigured 报告 Model Hub API Key 是否已配置（MH4 预热前置判定用，
// 只查布尔不回传明文）。未配置时预热静默跳过，不用等 HTTP 401 才发现。
func (m *Manager) ModelHubKeyConfigured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.modelhubKey != ""
}
