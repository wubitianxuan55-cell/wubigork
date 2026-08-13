package modelengine

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/netclient"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ── 引擎类型 ───────────────────────────────────────────────

// EngineType 模型引擎类型
type EngineType string

const (
	EngineXAI      EngineType = "xai"
	EngineOllama   EngineType = "ollama"
	EngineHerdsman EngineType = "herdsman"
	EngineDeepseek EngineType = "deepseek"
	// EngineCosyVoice 本地 CosyVoice2 TTS 服务（OpenAI 兼容 /v1/audio/speech）
	EngineCosyVoice EngineType = "cosyvoice"
	// EngineOpencodeGo OpenCode Go 云端目录（OpenAI 兼容 /chat/completions）
	EngineOpencodeGo EngineType = "opencode-go"
	// EngineOpencodeZen OpenCode Zen 云端目录（OpenAI 兼容 /chat/completions 子集）
	EngineOpencodeZen EngineType = "opencode-zen"
)

// ── 数据结构 ───────────────────────────────────────────────

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
	Status  string `json:"status,omitempty"` // "running" / "stopped" / "unknown"
	Kind    string `json:"kind,omitempty"`   // "llm" / "tts" / "stt" / "image"，由后端按引擎/名称分类，前端不再猜测
}

// EngineConfig 引擎配置
type EngineConfig struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         EngineType   `json:"type"`
	Label        string       `json:"label,omitempty"`    // 展示名（如 "Ollama 本地"），前端优先使用
	Color        string       `json:"color,omitempty"`    // 主题色（十六进制），前端优先使用
	Icon         string       `json:"icon,omitempty"`     // 图标键（cloud/desktop/rocket/key/global），前端映射
	IsLocal      bool         `json:"is_local,omitempty"` // 本地引擎（免费/本机资源）
	BaseURL      string       `json:"base_url"`
	APIKey       string       `json:"api_key,omitempty"`
	Enabled      bool         `json:"enabled"`
	DefaultModel string       `json:"default_model"`
	Models       []ModelInfo  `json:"models,omitempty"`
	Status       EngineStatus `json:"status,omitempty"` // 最近连接状态缓存（刷新/测试后更新，随状态文件持久化）
}

// EngineStatus 引擎连接状态
type EngineStatus struct {
	ID          string `json:"id"`
	Connected   bool   `json:"connected"`
	ModelCount  int    `json:"model_count"`
	Error       string `json:"error,omitempty"`
	LastChecked string `json:"last_checked,omitempty"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
}

// ── OpenAI 兼容响应结构 ────────────────────────────────────

type modelsListResponse struct {
	Data []struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by"`
		Status  string `json:"status"`
	} `json:"data"`
}

// ── 引擎管理器 ─────────────────────────────────────────────

type Manager struct {
	mu             sync.RWMutex
	engines        map[string]*EngineConfig
	order          []string // 稳定展示顺序（GetEngines 按此返回，避免 map 随机序）
	statePath      string   // 状态文件路径（空=不落盘）
	xaiKey         string   // xAI API key（来自 OAuth token）
	deepseekKey    string   // DeepSeek API key（用户手动配置）
	opencodeKey    string   // OpenCode Go API key（用户手动配置，订阅后从 console 获取）
	opencodeZenKey string   // OpenCode Zen API key（按量付费，opencode.ai/auth 获取）
	httpClient     *http.Client
	statsMu        sync.Mutex     // 保护 statsRec 的懒初始化
	statsRec       *statsRecorder // 模型调用统计（可为 nil，首次记录时创建）
}

// NewManager 创建引擎管理器
func NewManager(xaiAPIKey, deepseekKey string) *Manager {
	m := &Manager{
		engines:     make(map[string]*EngineConfig),
		order:       []string{"xai", "ollama", "herdsman", "deepseek", "cosyvoice", "opencode-go", "opencode-zen"},
		xaiKey:      xaiAPIKey,
		deepseekKey: deepseekKey,
		httpClient:  netclient.NewSimpleClient(15 * time.Second),
	}

	// 预置四大引擎默认配置
	m.engines["xai"] = &EngineConfig{
		ID:           "xai",
		Name:         "xAI (Grok)",
		Type:         EngineXAI,
		Label:        "xAI 云端",
		Color:        "#60a5fa",
		Icon:         "cloud",
		BaseURL:      "https://api.x.ai/v1",
		Enabled:      true,
		DefaultModel: "grok-4.20",
	}
	m.engines["ollama"] = &EngineConfig{
		ID:           "ollama",
		Name:         "Ollama",
		Type:         EngineOllama,
		Label:        "Ollama 本地",
		Color:        "#f59e0b",
		Icon:         "desktop",
		IsLocal:      true,
		BaseURL:      "http://localhost:11434/v1",
		Enabled:      true,
		DefaultModel: "",
	}
	m.engines["herdsman"] = &EngineConfig{
		ID:           "herdsman",
		Name:         "Herdsman",
		Type:         EngineHerdsman,
		Label:        "Herdsman 本地",
		Color:        "#84cc16",
		Icon:         "rocket",
		IsLocal:      true,
		BaseURL:      "http://localhost:8080/v1",
		Enabled:      true,
		DefaultModel: "",
	}
	m.engines["deepseek"] = &EngineConfig{
		ID:           "deepseek",
		Name:         "DeepSeek",
		Type:         EngineDeepseek,
		Label:        "DeepSeek 云端",
		Color:        "#8b5cf6",
		Icon:         "key",
		BaseURL:      "https://api.deepseek.com",
		Enabled:      true,
		DefaultModel: "deepseek-v4-pro",
	}
	m.engines["cosyvoice"] = &EngineConfig{
		ID:           "cosyvoice",
		Name:         "CosyVoice2 (本地)",
		Type:         EngineCosyVoice,
		Label:        "CosyVoice2 本地",
		Color:        "#f472b6",
		Icon:         "rocket",
		IsLocal:      true,
		BaseURL:      "http://127.0.0.1:8010/v1",
		Enabled:      true,
		DefaultModel: "CosyVoice2-0.5B",
	}
	m.engines["opencode-go"] = &EngineConfig{
		ID:           "opencode-go",
		Name:         "OpenCode Go (云端)",
		Type:         EngineOpencodeGo,
		Label:        "OpenCode Go 云端",
		Color:        "#22d3ee",
		Icon:         "global",
		BaseURL:      "https://opencode.ai/zen/go/v1",
		Enabled:      true,
		DefaultModel: "deepseek-v4-pro",
	}
	m.engines["opencode-zen"] = &EngineConfig{
		ID:           "opencode-zen",
		Name:         "OpenCode Zen (云端)",
		Type:         EngineOpencodeZen,
		Label:        "OpenCode Zen 云端",
		Color:        "#a78bfa",
		Icon:         "global",
		BaseURL:      "https://opencode.ai/zen/v1",
		Enabled:      true,
		DefaultModel: "deepseek-v4-pro",
	}

	return m
}

// UpdateXAIKey 更新 xAI API key
func (m *Manager) UpdateXAIKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.xaiKey = key
}

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
	engine.Models = append(engine.Models, ModelInfo{ID: modelID, OwnedBy: engineID, Kind: classifyModelKind(engine.Type, modelID)})
	m.mu.Unlock()
	m.saveState()
}

// UpdateDeepseekKey 更新 DeepSeek API key
func (m *Manager) UpdateDeepseekKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deepseekKey = key
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
		existing.BaseURL = cfg.BaseURL
	}
	if cfg.DefaultModel != "" {
		existing.DefaultModel = cfg.DefaultModel
	}
	// Enabled 由前端控制
	existing.Enabled = cfg.Enabled
	m.mu.Unlock()

	m.saveState()
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

	start := time.Now()
	models, err := m.fetchModels(ctx, engine)
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

// fetchModels 从引擎获取模型列表
// fetchModels 从引擎获取模型列表
func (m *Manager) fetchModels(ctx context.Context, engine *EngineConfig) ([]ModelInfo, error) {
	baseURL := strings.TrimRight(engine.BaseURL, "/")
	url := baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// xAI / DeepSeek / OpenCode Go 需要认证
	if engine.Type == EngineXAI && m.xaiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.xaiKey)
	} else if engine.Type == EngineDeepseek && m.deepseekKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.deepseekKey)
	} else if engine.Type == EngineOpencodeGo && m.opencodeKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.opencodeKey)
	} else if engine.Type == EngineOpencodeZen && m.opencodeZenKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.opencodeZenKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 401 {
			if engine.Type == EngineXAI {
				return nil, fmt.Errorf("HTTP 401: 未登录 xAI，请先点击「登录 xAI」获取授权")
			} else if engine.Type == EngineDeepseek {
				return nil, fmt.Errorf("HTTP 401: DeepSeek API Key 无效或未配置，请在设置中配置")
			} else if engine.Type == EngineOpencodeGo {
				return nil, fmt.Errorf("HTTP 401: OpenCode Go API Key 无效或未配置，请先在模型中心配置（opencode.ai 订阅获取）")
			} else if engine.Type == EngineOpencodeZen {
				return nil, fmt.Errorf("HTTP 401: OpenCode Zen API Key 无效或未配置，请先在模型中心配置（opencode.ai/auth 获取）")
			}
		}
		return nil, fmt.Errorf("HTTP %d: 模型列表获取失败", resp.StatusCode)
	}

	var result modelsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析模型列表失败: %w", err)
	}

	models := make([]ModelInfo, len(result.Data))
	for i, d := range result.Data {
		// OpenCode 目录端点分布与 Go 不同（如 grok 在 Zen 走 /responses、
		// minimax 在 Zen 走 chat/completions），按引擎分别过滤，
		// 只保留当前聊天客户端支持的 OpenAI 兼容 /chat/completions 模型。
		if engine.Type == EngineOpencodeGo && !opencodeGoCompatible(d.ID) {
			continue
		}
		if engine.Type == EngineOpencodeZen && !opencodeZenCompatible(d.ID) {
			continue
		}
		models[i] = ModelInfo{
			ID:      d.ID,
			OwnedBy: d.OwnedBy,
			Status:  d.Status,
			Kind:    classifyModelKind(engine.Type, d.ID),
		}
	}

	// 过滤后可能有空洞（continue 跳过），压缩数组
	if engine.Type == EngineOpencodeGo || engine.Type == EngineOpencodeZen {
		filtered := make([]ModelInfo, 0, len(models))
		for _, m := range models {
			if m.ID != "" {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}

	// xAI 引擎补充内置语音模型 grok-tts（TTS API 不返回在 /v1/models 列表中）
	if engine.Type == EngineXAI {
		found := false
		for _, mdl := range models {
			if mdl.ID == "grok-tts" {
				found = true
				break
			}
		}
		if !found {
			models = append(models, ModelInfo{ID: "grok-tts", OwnedBy: "xai", Kind: "tts"})
		}
	}

	return models, nil
}

// classifyModelKind 按引擎类型与模型名分类（llm/tts/stt/image）。
// 分类下沉到后端后，前端不再需要根据名称猜测；逻辑与旧前端启发式保持一致，避免行为跳变。
func classifyModelKind(engineType EngineType, modelID string) string {
	l := strings.ToLower(modelID)
	if engineType == EngineCosyVoice ||
		strings.Contains(l, "tts") || strings.Contains(l, "voice") ||
		strings.Contains(l, "edge") || strings.Contains(l, "speech") ||
		strings.Contains(l, "voxcpm") {
		return "tts"
	}
	if strings.Contains(l, "sherpa") || strings.Contains(l, "whisper") ||
		strings.Contains(l, "zipformer") || strings.Contains(l, "asr") ||
		strings.Contains(l, "funasr") {
		return "stt"
	}
	if strings.Contains(l, "paddleocr") || strings.Contains(l, "ocr") ||
		strings.Contains(l, "mineru") {
		return "ocr"
	}
	if strings.Contains(l, "rerank") {
		return "rerank"
	}
	if strings.Contains(l, "embedding") || strings.Contains(l, "bge-m3") ||
		strings.Contains(l, "bge") {
		return "embedding"
	}
	if strings.Contains(l, "image") || strings.Contains(l, "zimage") ||
		strings.Contains(l, "flux") || strings.Contains(l, "turbo") ||
		strings.Contains(l, "sd") || strings.Contains(l, "dalle") ||
		strings.Contains(l, "krea") {
		return "image"
	}
	return "llm"
}

// opencodeGoCompatible 判断 OpenCode Go 模型是否走 OpenAI /chat/completions 端点。
// 参考 https://dev.opencode.ai/docs/go 端点分类表。
func opencodeGoCompatible(modelID string) bool {
	id := strings.ToLower(modelID)
	if strings.HasPrefix(id, "qwen3") || strings.HasPrefix(id, "minimax") || id == "gpt-5.6-luna" {
		return false
	}
	return true
}

// opencodeZenCompatible 判断 OpenCode Zen 模型是否走 OpenAI /chat/completions 端点。
// 参考 https://dev.opencode.ai/docs/zen 端点分类表：
//   - gpt-* / grok-* → /responses；claude-* / qwen3* → /messages；gemini-* → 专用端点
//   - deepseek-* / minimax-* / glm-* / kimi-* / mimo-* / 免费模型等 → /chat/completions
func opencodeZenCompatible(modelID string) bool {
	id := strings.ToLower(modelID)
	for _, prefix := range []string{"gpt-", "claude-", "gemini-", "qwen3", "grok-"} {
		if strings.HasPrefix(id, prefix) {
			return false
		}
	}
	return true
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

// ── 状态持久化（engines.json）───────────────────────────────

// stateFile 磁盘状态文件结构。
type stateFile struct {
	Engines map[string]EngineConfig `json:"engines"`
}

// LoadState 从 path 加载引擎状态并设置自动落盘路径。
// 首次启动文件不存在时静默降级（保留预置默认）。
func (m *Manager) LoadState(path string) error {
	m.mu.Lock()
	m.statePath = path
	m.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var f stateFile
	if err := json.Unmarshal(data, &f); err != nil {
		slog.Warn("引擎状态文件解析失败", "path", path, "error", err)
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for id, st := range f.Engines {
		eng, ok := m.engines[id]
		if !ok {
			// 未知引擎（新版本移除/手改文件）不创建
			continue
		}
		if st.BaseURL != "" {
			eng.BaseURL = st.BaseURL
		}
		eng.Enabled = st.Enabled
		if st.DefaultModel != "" {
			eng.DefaultModel = st.DefaultModel
		}
		if st.Models != nil {
			eng.Models = st.Models
		}
		if st.Status.LastChecked != "" {
			eng.Status = st.Status
		}
	}
	return nil
}

// saveState 将当前引擎状态快照写回状态文件（path 未设置时跳过）。
// 调用方不得持有写锁；内部自行加读锁快照。
func (m *Manager) saveState() {
	m.mu.RLock()
	path := m.statePath
	if path == "" {
		m.mu.RUnlock()
		return
	}
	f := stateFile{Engines: make(map[string]EngineConfig, len(m.engines))}
	for id, e := range m.engines {
		cfg := *e
		cfg.APIKey = ""
		f.Engines[id] = cfg
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		slog.Warn("序列化引擎状态失败", "error", err)
		return
	}
	if err := fileutil.AtomicWrite(path, data, 0644); err != nil {
		slog.Warn("保存引擎状态失败", "path", path, "error", err)
	}
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
	} else if engine.Type == EngineOpencodeGo {
		apiKey = m.opencodeKey
	} else if engine.Type == EngineOpencodeZen {
		apiKey = m.opencodeZenKey
	}

	return chatURL, apiKey, nil
}
