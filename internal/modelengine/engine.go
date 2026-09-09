// engine.go 引擎核心：类型定义（EngineType/ModelInfo/EngineConfig/EngineStatus）与
// Manager 结构、预置引擎默认配置。
//
// 纯结构拆分（P4 结构刀）：原单文件 55KB 超红线后按域重组到同包多文件，行为零
// 变化——engine_keys.go（各服务商 API Key）、engine_custom.go（自定义引擎 A 刀）、
// engine_crud.go（配置查询/保存）、engine_connect.go（连接测试/URL 装配）、
// engine_models.go（模型列表/分类）、engine_modelhub.go（Unsloth Studio）、
// engine_state.go（engines.json 持久化）。(本文件瘦身后保留核心类型与预置默认)
package modelengine

import (
	"net/http"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/netclient"
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
	// EngineModelHub Unsloth 本地 Model Hub（Desktop/Studio「Model hub」标签页
	// 选模型下载后，由 unsloth run/start 暴露 OpenAI 兼容 /v1 端点；本地引擎，
	// 请求带 sk-unsloth- 开头的 Bearer Key——见 unsloth.ai/docs/basics/api）。
	EngineModelHub EngineType = "modelhub"
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
// 门控路由：offline 开启时只允许本地引擎（ollama/herdsman/cosyvoice/modelhub），
// 云端（xai/deepseek/opencode-*）一律跳过。
func (t EngineType) IsLocal() bool {
	switch t {
	case EngineOllama, EngineHerdsman, EngineCosyVoice, EngineModelHub:
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
	// Name 展示名（可选）：服务商 /models 下发 display_name 或由 ID/别名解析
	// 出的友好名。请求路由仍一律使用 ID——展示名与请求名解耦，避免把
	// URL 编码的 ollama-manifest 别名直接当模型名展示给用户。
	Name string `json:"name,omitempty"`
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
	// ── 价目 v1：用户自填价目（UI 仅对自定义引擎暴露，字段本身引擎通用）──
	// 引擎级统一价：每百万 tokens 输入/输出单价，币种固定 CNY（TotalCost
	// 统一人民币口径）。指针三态：nil=未设置/不修改（SaveEngine 部分更新
	// 语义，地址框/启停等局部保存不会误清除；omitempty 零新增字节，旧
	// engines.json 兼容）；指向 <=0/NaN/Inf 视为清除。消费点：SyncUserPrices
	// 重建注册表 → stats.estimatePrice 最高优先层（用户价 > 目录/内置表）；
	// ollama/herdsman 本地引擎恒不计价，不消费用户价。见 user_price.go。
	UserPriceIn  *float64 `json:"user_price_in,omitempty"`  // 输入单价（¥/百万 tokens）
	UserPriceOut *float64 `json:"user_price_out,omitempty"` // 输出单价（¥/百万 tokens）
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
	modelhubKey    string            // Unsloth Model Hub API key（用户手动配置，sk-unsloth- 前缀）
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
		order:       []string{"xai", "ollama", "herdsman", "deepseek", "glm", "cosyvoice", "modelhub", "opencode-go", "opencode-zen"},
		xaiKey:      xaiAPIKey,
		deepseekKey: deepseekKey,
		customKeys:  make(map[string]string),
		httpClient:  netclient.NewSimpleClient(15 * time.Second),
	}

	// 预置引擎默认配置
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
	m.engines["modelhub"] = &EngineConfig{
		ID:      "modelhub",
		Name:    "Unsloth Model Hub",
		Type:    EngineModelHub,
		Label:   "Model Hub 本地",
		Color:   "#fb7185",
		Icon:    "rocket",
		IsLocal: true,
		// Unsloth Studio 后端固定 127.0.0.1:8888，并把 OpenAI 兼容 API 挂在
		// /v1（转发到当前已加载模型的 llama-server，llama 端口每次加载会变，
		// 8888/v1 是稳定入口）。鉴权：Settings → API 创建 sk-unsloth- Key，
		// 请求需带 Authorization: Bearer。地址框可改（8888 被占用时 Studio
		// 会漂移到其他端口）。
		BaseURL: "http://127.0.0.1:8888/v1",
		Enabled: true,
	}

	return m
}
