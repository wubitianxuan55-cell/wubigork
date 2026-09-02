package modelengine

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

// ── 引擎类型 ───────────────────────────────────────────────

// EngineType 模型引擎类型
type EngineType string

const (
	EngineXAI      EngineType = "xai"
	EngineOllama   EngineType = "ollama"
	EngineHerdsman EngineType = "herdsman"
	EngineDeepseek EngineType = "deepseek"
	// EngineGLM 智谱 GLM 云端（OpenAI 兼容 /api/paas/v4）
	EngineGLM EngineType = "glm"
	// EngineCosyVoice 本地 CosyVoice2 TTS 服务（OpenAI 兼容 /v1/audio/speech）
	EngineCosyVoice EngineType = "cosyvoice"
	// EngineOpencodeGo OpenCode Go 云端目录（OpenAI 兼容 /chat/completions）
	EngineOpencodeGo EngineType = "opencode-go"
	// EngineOpencodeZen OpenCode Zen 云端目录（OpenAI 兼容 /chat/completions 子集）
	EngineOpencodeZen EngineType = "opencode-zen"
	// EngineCustom 自定义引擎（A 刀）：用户自带的 OpenAI 兼容服务商
	// （自定义 BaseURL + /chat/completions + /models），Key 只存 Manager 内存
	// customKeys（落盘走 config 层 custom_engine_keys 密文）。IsLocal=false（云端语义，
	// 全局离线模式下与其他云端引擎一致被门控）。
	EngineCustom EngineType = "custom"
)

// GLM 官方双端点（docs.bigmodel.cn coding-plan/quick-start）：标准=按量付费，
// coding=编码套餐额度。填错端点：编码套餐 Key 会 404 或误走按量计费。
// 预置卡与 SetGlmEndpoint 共用单一来源。
const (
	GLMBaseURLStd    = "https://open.bigmodel.cn/api/paas/v4"
	GLMBaseURLCoding = "https://open.bigmodel.cn/api/coding/paas/v4"
)

// IsLocal 引擎是否本地服务（数据不出本机）——全局离线模式（v4.8）据此
// 门控路由：offline 开启时只允许本地引擎（ollama/herdsman/cosyvoice），
// 云端（xai/deepseek/opencode-*）一律跳过。
func (t EngineType) IsLocal() bool {
	switch t {
	case EngineOllama, EngineHerdsman, EngineCosyVoice:
		return true
	}
	return false
}

// ── 数据结构 ───────────────────────────────────────────────

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
	Status  string `json:"status,omitempty"` // "running" / "stopped" / "unknown"
	Kind    string `json:"kind,omitempty"`   // "llm" / "tts" / "stt" / "image"，由后端按引擎/名称分类，前端不再猜测
	// AliasOf coding 端点家族下服务端实际服务的模型（套餐旧名自动切换，
	// 见 glm_alias.go）；std 家族为空。诚实展示注记，请求模型名不改写。
	AliasOf string `json:"alias_of,omitempty"`

	// ── B 刀：能力/价格元数据（GLM 目录 v2 透传，其余引擎恒空）────────
	// 全部 omitempty：非 GLM 引擎 / 未注记模型零新增字节，旧 JSON 兼容。
	ContextLength int      `json:"context_length,omitempty"` // 上下文窗口（tokens 绝对值，如 1000000）
	MaxOutput     int      `json:"max_output,omitempty"`     // 最大输出 tokens
	PriceIn       float64  `json:"price_in,omitempty"`       // 官方目录价：每百万 tokens 输入；unit 非 tokens 时为单次价
	PriceOut      float64  `json:"price_out,omitempty"`      // 官方目录价：每百万 tokens 输出
	Currency      string   `json:"currency,omitempty"`       // "CNY" | "USD"
	Unit          string   `json:"unit,omitempty"`           // 空=tokens 每百万；"call"=每次；"minute"=每分钟
	Free          bool     `json:"free,omitempty"`           // 官方免费档（费用恒 0）
	Caps          []string `json:"caps,omitempty"`           // 能力标记：vision/tools/reasoning/search/json（宁缺勿滥，官方未列不填）
	PriceNote     string   `json:"price_note,omitempty"`     // 价格口径备注（如官方只给相对价）
	PointsIn      float64  `json:"points_in,omitempty"`      // coding 套餐积分系数：输入（积分=(输入×In+缓存×Cached+输出×Out)/10000）
	PointsCached  float64  `json:"points_cached,omitempty"`  // coding 套餐积分系数：缓存命中
	PointsOut     float64  `json:"points_out,omitempty"`     // coding 套餐积分系数：输出
	PointsPeak    float64  `json:"points_peak,omitempty"`    // coding 套餐高峰倍率（如 3、1.2）
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
	order          []string          // 稳定展示顺序（GetEngines 按此返回，避免 map 随机序）
	statePath      string            // 状态文件路径（空=不落盘）
	xaiKey         string            // xAI API key（来自 OAuth token）
	deepseekKey    string            // DeepSeek API key（用户手动配置）
	glmKey         string            // 智谱 GLM API key（用户手动配置）
	opencodeKey    string            // OpenCode Go API key（用户手动配置，订阅后从 console 获取）
	opencodeZenKey string            // OpenCode Zen API key（按量付费，opencode.ai/auth 获取）
	customKeys     map[string]string // 自定义引擎 Key（engineID → 明文，仅内存；落盘走 config 层密文）
	httpClient     *http.Client
	statsMu        sync.Mutex     // 保护 statsRec 的懒初始化
	statsRec       *statsRecorder // 模型调用统计（可为 nil，首次记录时创建）
	// catalogRemoteStop GLM 目录远程热更新拉取循环的停止通道（nil=未启动，
	// 见 catalog_remote.go；app Shutdown 关闭）。
	catalogRemoteStop chan struct{}
	// ── 健康巡检 + 故障转移（C 刀 v0，见 health_probe.go）─────────────
	healthStop   chan struct{}                   // 巡检循环停止通道（nil=未启动）
	healthNotify func(id string, connected bool) // 状态变化回调（app 接线 emit，Manager 不直接 emit）
	probeFails   map[string]int                  // 连续探测失败计数（仅内存，重启清零）
}

// NewManager 创建引擎管理器
func NewManager(xaiAPIKey, deepseekKey string) *Manager {
	m := &Manager{
		engines:     make(map[string]*EngineConfig),
		order:       []string{"xai", "ollama", "herdsman", "deepseek", "glm", "cosyvoice", "opencode-go", "opencode-zen"},
		xaiKey:      xaiAPIKey,
		deepseekKey: deepseekKey,
		customKeys:  make(map[string]string),
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
	m.engines["glm"] = &EngineConfig{
		ID:           "glm",
		Name:         "GLM (智谱)",
		Type:         EngineGLM,
		Label:        "GLM 云端",
		Color:        "#38bdf8",
		Icon:         "key",
		BaseURL:      GLMBaseURLStd,
		Enabled:      true,
		DefaultModel: "glm-5.3",
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
	engine.Models = append(engine.Models, ModelInfo{ID: modelID, OwnedBy: engineID, Kind: ClassifyModelKind(engine.Type, modelID)})
	m.mu.Unlock()
	m.saveState()
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

// ── 自定义引擎（A 刀：OpenAI 兼容自定义服务商）────────────────

// customEnginePrefix 自定义引擎 ID 前缀：仅该前缀且 Type=custom 的引擎允许
// 经 UpdateCustomEngine/RemoveCustomEngine 修改或删除（内置引擎受保护）。
const customEnginePrefix = "custom-"

// AddCustomEngine 创建自定义引擎并返回 engineID。
// ID 规则：custom- 前缀 + name 生成的安全小写 slug（去非法字符，空 slug 用
// "engine"），与现有 id 冲突时追加 -2、-3…。baseURL 校验 http(s) scheme +
// host 非空——云端引擎不露地址框防线（v4.9.1 Key 粘错框事故）的延伸，把
// API Key 当地址粘进来必须在此拒绝。Key 只存内存 customKeys，不进
// EngineConfig.APIKey（saveState/GetEngines 因此永不下发/落盘）。
func (m *Manager) AddCustomEngine(name, baseURL, apiKey string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("引擎名称不能为空")
	}
	u, err := validateCustomBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	id := m.allocCustomIDLocked(customSlug(name))
	m.engines[id] = &EngineConfig{
		ID:      id,
		Name:    "custom",
		Type:    EngineCustom,
		Label:   name,
		BaseURL: u,
		Enabled: true,
	}
	m.order = append(m.order, id)
	if m.customKeys == nil {
		m.customKeys = make(map[string]string)
	}
	m.customKeys[id] = apiKey
	m.mu.Unlock()
	m.saveState()
	slog.Info("自定义引擎已创建", "engine", id)
	return id, nil
}

// UpdateCustomEngine 更新自定义引擎（engineID 必须是 custom- 前缀的自定义引擎）。
// apiKey 传空串 = 保留原 Key 不变（前端不回显 Key，编辑时留空即「不改」）。
func (m *Manager) UpdateCustomEngine(engineID, name, baseURL, apiKey string) error {
	if !strings.HasPrefix(engineID, customEnginePrefix) {
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止修改", engineID)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("引擎名称不能为空")
	}
	u, err := validateCustomBaseURL(baseURL)
	if err != nil {
		return err
	}
	m.mu.Lock()
	eng, ok := m.engines[engineID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不存在", engineID)
	}
	if eng.Type != EngineCustom {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止修改", engineID)
	}
	eng.Label = name
	eng.BaseURL = u
	if apiKey != "" {
		if m.customKeys == nil {
			m.customKeys = make(map[string]string)
		}
		m.customKeys[engineID] = apiKey
	}
	m.mu.Unlock()
	m.saveState()
	slog.Info("自定义引擎已更新", "engine", engineID)
	return nil
}

// RemoveCustomEngine 删除自定义引擎（仅允许 custom- 前缀且 Type=custom），
// 一并清掉内存 Key 并从展示顺序中摘除。
func (m *Manager) RemoveCustomEngine(engineID string) error {
	if !strings.HasPrefix(engineID, customEnginePrefix) {
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止删除", engineID)
	}
	m.mu.Lock()
	eng, ok := m.engines[engineID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不存在", engineID)
	}
	if eng.Type != EngineCustom {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止删除", engineID)
	}
	delete(m.engines, engineID)
	for i, id := range m.order {
		if id == engineID {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	delete(m.customKeys, engineID)
	// 健康巡检连续失败计数一并清理（C 刀 v0）
	delete(m.probeFails, engineID)
	m.mu.Unlock()
	m.saveState()
	slog.Info("自定义引擎已删除", "engine", engineID)
	return nil
}

// SetCustomEngineKeys 批量注入自定义引擎 Key（【解密后的明文】，仅存内存；
// app 层启动时从 config custom_engine_keys 解密后调用）。传 nil/空 map = 清空。
func (m *Manager) SetCustomEngineKeys(keys map[string]string) {
	cp := make(map[string]string, len(keys))
	for id, k := range keys {
		cp[id] = k
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customKeys = cp
}

// CustomEngineKey 返回自定义引擎 Key（明文，仅内存）；未配置返回空串。
// 消费口径与 GLMKey 一致：Key 存 manager 而非 EngineConfig.APIKey。
func (m *Manager) CustomEngineKey(engineID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.customKeys[engineID]
}

// CustomEngineKeys 返回全部自定义引擎 Key 副本（明文；app 层加密后落
// config custom_engine_keys）。
func (m *Manager) CustomEngineKeys() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.customKeys))
	for id, k := range m.customKeys {
		out[id] = k
	}
	return out
}

// allocCustomIDLocked 生成 custom- 前缀 ID：与现有 id（内置或 custom）冲突时
// 追加 -2、-3…（调用方需持写锁）。
func (m *Manager) allocCustomIDLocked(base string) string {
	id := customEnginePrefix + base
	for n := 2; ; n++ {
		if _, exists := m.engines[id]; !exists {
			return id
		}
		id = fmt.Sprintf("%s%s-%d", customEnginePrefix, base, n)
	}
}

// customSlug 由 name 生成安全小写 slug：小写化后仅保留字母/数字/连字符，
// 空白折叠为 '-'（其余非法字符——中文/符号等——剔除），压缩连续 '-'、
// 修剪首尾，空结果回退 "engine"；超 40 字符截断（同样修剪尾部连字符）。
func customSlug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case unicode.IsSpace(r):
			b.WriteRune('-')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-':
			b.WriteRune(r)
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.Trim(out, "-")
	if len(out) > 40 {
		out = strings.Trim(out[:40], "-")
	}
	if out == "" {
		return "engine"
	}
	return out
}

// validateCustomBaseURL 自定义引擎地址校验：url.Parse 可解析 + scheme 为
// http/https + host 非空。v4.9.1「Key 粘错框」防线的延伸——把 API Key 当
// 地址粘进来（无 scheme/host）必须在此拒绝，不接受仅前缀判断的宽松口径。
func validateCustomBaseURL(u string) (string, error) {
	u = strings.TrimSpace(u)
	parsed, err := url.Parse(u)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("引擎地址无效：必须是 http:// 或 https:// 开头的完整地址（请勿把 API Key 粘进地址框）")
	}
	return u, nil
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

// fetchModels 从引擎获取模型列表
// fetchModels 从引擎获取模型列表
func (m *Manager) fetchModels(ctx context.Context, engine *EngineConfig) ([]ModelInfo, error) {
	if engine.Type == EngineGLM {
		// 智谱无 /models 端点：刷新直接返回官方静态目录（零 HTTP）
		return m.glmCatalogModels(), nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(engine.BaseURL), "/")
	if !validBaseURL(baseURL) {
		return nil, fmt.Errorf("引擎地址无效：需要 http:// 或 https:// 前缀，请在模型中心修正")
	}
	url := baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// xAI / DeepSeek / GLM / OpenCode Go 需要认证
	if engine.Type == EngineXAI && m.xaiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.xaiKey)
	} else if engine.Type == EngineDeepseek && m.deepseekKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.deepseekKey)
	} else if engine.Type == EngineGLM && m.glmKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.glmKey)
	} else if engine.Type == EngineOpencodeGo && m.opencodeKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.opencodeKey)
	} else if engine.Type == EngineOpencodeZen && m.opencodeZenKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.opencodeZenKey)
	} else if engine.Type == EngineCustom {
		// 自定义引擎：Key 在内存 customKeys（KeyStore 同源），空 Key 不带
		// Authorization 头（兼容无鉴权的本地 OpenAI 兼容服务）。
		if key := m.CustomEngineKey(engine.ID); key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
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
			} else if engine.Type == EngineGLM {
				return nil, fmt.Errorf("HTTP 401: GLM API Key 无效或未配置，请在模型中心配置（open.bigmodel.cn 获取）")
			} else if engine.Type == EngineOpencodeGo {
				return nil, fmt.Errorf("HTTP 401: OpenCode Go API Key 无效或未配置，请先在模型中心配置（opencode.ai 订阅获取）")
			} else if engine.Type == EngineOpencodeZen {
				return nil, fmt.Errorf("HTTP 401: OpenCode Zen API Key 无效或未配置，请先在模型中心配置（opencode.ai/auth 获取）")
			} else if engine.Type == EngineCustom {
				return nil, fmt.Errorf("HTTP 401: 自定义引擎 API Key 无效或未配置，请在模型中心自定义引擎卡片修正")
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
			Kind:    ClassifyModelKind(engine.Type, d.ID),
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

	// C 刀目录通用化：动态模型列表按通用目录补充徽标元数据（只填空字段，
	// 不覆盖引擎返回值，随 saveState 持久化）。GLM 分支已在函数头提前返回
	// 静态目录、不走此公共出口（无双重 enrich）；opencode-go/custom 不在
	// 目录内，enrich 原样返回（零行为变化）。
	models = enrichCatalogMeta(string(engine.Type), models)

	return models, nil
}

// ClassifyModelKind 按引擎类型与模型名分类（llm/tts/stt/ocr/rerank/embedding/image）。
// 3.0 Step 3d：模型能力关键词分类的单一来源——语音（voice_handler.go:isSTTModel）、
// OCR（gaea_ocr.go:pickHerdsmanModel）等消费点委托到本函数，不再各自维护关键词表。
// 分类下沉到后端后，前端不再需要按名称猜测；行为与旧前端启发式保持一致，避免行为跳变。
// EngineCustom（A 刀自定义 OpenAI 兼容服务商）无厂商特型规则，与多数类型一样
// 直接落通用关键词表、默认 llm——刻意不加类型分支（避免死代码），由测试锚定该行为。
func ClassifyModelKind(engineType EngineType, modelID string) string {
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
	// GLM 官方静态目录优先判型：glm-image 系为生图，其余 glm-* 均为对话/视觉
	// 理解模型（官方目录锚定）——通用 turbo 关键词曾把 glm-5-turbo 误判为生图
	// （v4.10.0 回归），故 GLM 引擎先走本块再落通用关键词表。
	if engineType == EngineGLM && strings.HasPrefix(l, "glm-") {
		if strings.HasPrefix(l, "glm-image") {
			return "image"
		}
		return "llm"
	}
	if strings.Contains(l, "image") || strings.Contains(l, "zimage") ||
		strings.Contains(l, "flux") || strings.Contains(l, "cogview") ||
		strings.Contains(l, "turbo") ||
		strings.Contains(l, "sd") || strings.Contains(l, "dalle") ||
		strings.Contains(l, "krea") {
		return "image"
	}
	return "llm"
}

// ClassifyModelByName 只按模型名关键词分类（不依赖引擎类型），供语音/OCR 侧
// 按模型 ID 单参数判断能力（如 isSTTModel / pickHerdsmanModel 委托）。
// 与 ClassifyModelKind 的引擎无关部分保持同一关键词表，避免双源漂移。
func ClassifyModelByName(modelID string) string {
	return ClassifyModelKind("", modelID)
}

// glmPing 用最小 chat 请求验证 Key 有效性——智谱没有模型列表端点可供鉴权
// 探测，官方鉴权口径 = Authorization: Bearer <API Key>（docs.bigmodel.cn
// 「HTTP API 调用」）。错误体官方形态 {"error":{"code","message"}}，原样透出。
func (m *Manager) glmPing(ctx context.Context, engine *EngineConfig) error {
	m.mu.RLock()
	key := m.glmKey
	m.mu.RUnlock()
	if key == "" {
		return fmt.Errorf("GLM API Key 未配置，请在模型中心 GLM 卡片保存 Key（open.bigmodel.cn 获取）")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(strings.TrimSpace(engine.BaseURL), "/")+"/chat/completions",
		strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"ping"}],"max_tokens":1}`, engine.DefaultModel)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if msg := zhipuErrorMessage(body); msg != "" {
		return fmt.Errorf("GLM Key 校验失败（HTTP %d）：%s", resp.StatusCode, msg)
	}
	return fmt.Errorf("GLM Key 校验失败：HTTP %d", resp.StatusCode)
}

// zhipuErrorMessage 解析智谱错误体 {"error":{"code","message"}}（官方形态，
// 真机实测 {"code":"500","message":"内部错误"} 等）。
func zhipuErrorMessage(body []byte) string {
	var e struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	return ""
}

// validBaseURL 引擎地址必须带 http(s) scheme——防御把 API Key 等非地址内容
// 粘进地址框（v4.9.1 真机实测：GLM 卡片曾对云端引擎露出地址框，Key 被存成
// base_url 后每个请求都报 unsupported protocol scheme ""）。
func validBaseURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
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
			// custom- 前缀条目：重启后重建（Manager 启动只种子内置引擎）。
			// 防线：Type 必须是 custom、BaseURL 必须合法（或空）——状态文件
			// 被手改成脏地址/伪 custom 条目时不采纳，沿用 v4.9.1 防线口径。
			// Key 不从此处恢复（状态文件从不存 Key），由 app 层从 config
			// custom_engine_keys 解密后经 SetCustomEngineKeys 注入。
			if strings.HasPrefix(id, customEnginePrefix) && st.Type == EngineCustom &&
				(st.BaseURL == "" || validBaseURL(st.BaseURL)) {
				eng = &EngineConfig{
					ID:      id,
					Name:    "custom",
					Type:    EngineCustom,
					Label:   st.Label,
					BaseURL: st.BaseURL,
					Enabled: st.Enabled,
				}
				m.engines[id] = eng
				m.order = append(m.order, id)
			} else {
				// 未知引擎（新版本移除/手改文件）不创建
				continue
			}
		}
		// 无 http(s) 前缀的存量脏地址不采纳（保留预置默认）——v4.9.1 真机
		// 实测：API Key 被粘进地址框保存后，引擎永久报 unsupported protocol scheme。
		if st.BaseURL != "" && validBaseURL(st.BaseURL) {
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
	} else if engine.Type == EngineGLM {
		apiKey = m.glmKey
	} else if engine.Type == EngineOpencodeGo {
		apiKey = m.opencodeKey
	} else if engine.Type == EngineOpencodeZen {
		apiKey = m.opencodeZenKey
	} else if engine.Type == EngineCustom {
		// 自定义引擎：Key 在内存 customKeys（已持读锁，直接读 map 防重入死锁）；
		// 空 Key 原样返回，调用方（ai.Client）为空串时省略 Authorization 头。
		apiKey = m.customKeys[engine.ID]
	}

	return chatURL, apiKey, nil
}
