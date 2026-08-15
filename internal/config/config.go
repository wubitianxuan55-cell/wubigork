package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// ── 配置键常量 ──────────────────────────────────────────

// ── 配置键常量 ──────────────────────────────────────────
const (
	KeyNovelsDir           = "novels_dir"
	KeyXaiClientID         = "xai_client_id"
	KeyHTTPTimeoutSeconds  = "http_timeout_seconds"
	KeyDefaultTemperature  = "default_temperature"
	KeyAnalysisTemperature = "analysis_temperature"
	KeyReasoningEffort     = "reasoning_effort"
	KeyQualityThreshold    = "quality_threshold"
	KeyQualityMaxRetries   = "quality_max_retries"
	KeyTTSBinaryPath       = "tts_binary_path"
	KeyTTSModelPath        = "tts_model_path"
	KeyTTSPort             = "tts_port"
	KeyTTSBackend          = "tts_backend"
	KeyTTSSpeed            = "tts_speed"
	KeyImageBackend        = "image_backend"
	KeyComfyUIURL          = "comfyui_url"
	KeyImageSaveDir        = "image_save_dir"
	KeyImageModel          = "image_model"
	KeyPortraitBackend     = "portrait_backend" // 角色库剧照独立后端（空=跟随绘梦）
	KeyPortraitModel       = "portrait_model"   // 角色库剧照独立模型（空=跟随绘梦）
	KeyComfyUIPath         = "comfyui_path"
	KeyComfyUIPythonPath   = "comfyui_python_path"
	KeyActiveEngineID      = "active_engine_id"
	KeyModel               = "model"
	KeyActiveASREngine     = "active_asr_engine"      // 语音识别激活引擎
	KeyActiveASRModel      = "active_asr_model"       // 语音识别激活模型
	KeyActiveTTSEngine     = "active_tts_engine"      // 语音合成激活引擎
	KeyActiveTTSModel      = "active_tts_model"       // 语音合成激活模型
	KeyTTSVoice            = "tts_voice"              // 语音合成音色（Herdsman / Edge）
	KeyActiveOCREngine     = "active_ocr_engine"      // OCR 激活引擎
	KeyActiveOCRModel      = "active_ocr_model"       // OCR 激活模型
	KeyVoicePersonality    = "voice_personality"      // 语音对话角色（首页语音固定 gaea；聊天板块内语音跟随所选人格）
	KeyFuncChatVoiceEngine = "func_chat_voice_engine" // 聊天语音合成引擎（功能绑定，空=全局 TTS）
	KeyFuncChatVoiceModel  = "func_chat_voice_model"  // 聊天语音合成模型
	// 功能级模型绑定（聊天/轻语/小说/办公 各自独立 LLM，持久化重启不丢）
	KeyFuncChatEngine    = "func_chat_engine"
	KeyFuncChatModel     = "func_chat_model"
	KeyFuncNovelEngine   = "func_novel_engine"
	KeyFuncNovelModel    = "func_novel_model"
	KeyFuncOfficeEngine  = "func_office_engine"
	KeyFuncOfficeModel   = "func_office_model"
	KeyFuncGaeaEngine    = "func_gaea_engine"
	KeyFuncGaeaModel     = "func_gaea_model"
	KeyFuncCharLibEngine = "func_characterlib_engine"
	KeyFuncCharLibModel  = "func_characterlib_model"
	KeyFuncRoutineEngine = "func_routine_engine"
	KeyFuncRoutineModel  = "func_routine_model"
	// 功能级启停（FeatureModelBar 启停语义：只影响该功能的路由，不影响整个引擎）
	KeyFuncChatEnabled    = "func_chat_enabled"
	KeyFuncNovelEnabled   = "func_novel_enabled"
	KeyFuncOfficeEnabled  = "func_office_enabled"
	KeyFuncGaeaEnabled    = "func_gaea_enabled"
	KeyFuncCharLibEnabled = "func_characterlib_enabled"
	KeyFuncRoutineEnabled = "func_routine_enabled"
	// 敏感域本地化（S2-4/D8）：成本/报价类 AI 操作默认路由本地 Herdsman，
	// 可配置回云端。默认开启。
	KeySensitiveLocal = "sensitive_local"
	// 本地模型调度（T5-3a/b）：保活 + 启动自动预载，默认开启。
	KeyKeepWarm          = "keep_warm_enabled" // 保活：周期性探活已运行的本地模型，防卸载/降温
	KeyAutoPreload       = "auto_preload"      // 启动自动预载：按功能绑定预载 herdsman 模型
	KeyDeepseekAPIKey    = "deepseek_api_key"
	KeyOpencodeGoAPIKey  = "opencode_go_api_key"
	KeyOpencodeZenAPIKey = "opencode_zen_api_key"
	// 美元→人民币汇率（费用估算折算用，默认 7.2，可在模型中心配置）
	KeyUsdCnyRate = "usd_cny_rate"
	// CosyVoice 本地 TTS 服务（T6-9.5）：路径/端口可配置，默认与历史硬编码一致。
	KeyCosyVoiceDir  = "cosyvoice_dir"
	KeyCosyVoicePort = "cosyvoice_port"
)

// DefaultUsdCnyRate 美元→人民币汇率默认值（费用估算折算口径）。
const DefaultUsdCnyRate = 7.2

// CosyVoice 本地服务默认值（T6-9.5：路径/端口可配置，未配置时与历史硬编码一致）。
const (
	DefaultCosyVoiceDir  = `C:\AI\cosyvoice`
	DefaultCosyVoicePort = 8010
)

// configFile 表示 ~/.gaea_config.json 的结构
type configFile struct {
	XaiClientID         string  `json:"xai_client_id"`
	NovelsDir           string  `json:"novels_dir"`
	HTTPTimeoutSeconds  int     `json:"http_timeout_seconds"`
	DefaultTemperature  float64 `json:"default_temperature"`
	AnalysisTemperature float64 `json:"analysis_temperature"`
	ReasoningEffort     string  `json:"reasoning_effort"`    // Grok 推理深度: "low" / "high"
	QualityThreshold    int     `json:"quality_threshold"`   // 章节质量阈值 1-10，低于此触发自动重试
	QualityMaxRetries   int     `json:"quality_max_retries"` // 最大自动重试次数
	TTSBinaryPath       string  `json:"tts_binary_path,omitempty"`
	TTSModelPath        string  `json:"tts_model_path,omitempty"`
	ImageBackend        string  `json:"image_backend,omitempty"` // "xai" (默认) | "comfyui" | "herdsman" | "ollama"
	ComfyUIURL          string  `json:"comfyui_url,omitempty"`
	ImageSaveDir        string  `json:"image_save_dir,omitempty"`         // 图片生成存放目录
	ImageModel          string  `json:"image_model,omitempty"`            // 图片模型
	PortraitBackend     string  `json:"portrait_backend,omitempty"`       // 角色库剧照后端（空=跟随绘梦）
	PortraitModel       string  `json:"portrait_model,omitempty"`         // 角色库剧照模型（空=跟随绘梦）
	ComfyUIPath         string  `json:"comfyui_path,omitempty"`           // ComfyUI 安装目录
	ComfyUIPythonPath   string  `json:"comfyui_python_path,omitempty"`    // Python 解释器路径
	TTSPort             int     `json:"tts_port,omitempty"`               // TTS 服务端口
	TTSBackend          string  `json:"tts_backend,omitempty"`            // TTS 后端: "cpu" | "cuda"
	TTSSpeed            float64 `json:"tts_speed,omitempty"`              // TTS 语速
	ActiveEngineID      string  `json:"active_engine_id,omitempty"`       // 活跃模型引擎 ID
	Model               string  `json:"model,omitempty"`                  // 默认 LLM 模型名
	DeepseekAPIKey      string  `json:"deepseek_api_key,omitempty"`       // DeepSeek API Key
	OpenCodeGoAPIKey    string  `json:"opencode_go_api_key,omitempty"`    // OpenCode Go API Key
	OpenCodeZenAPIKey   string  `json:"opencode_zen_api_key,omitempty"`   // OpenCode Zen API Key
	ActiveASREngine     string  `json:"active_asr_engine,omitempty"`      // 语音识别激活引擎
	ActiveASRModel      string  `json:"active_asr_model,omitempty"`       // 语音识别激活模型
	ActiveTTSEngine     string  `json:"active_tts_engine,omitempty"`      // 语音合成激活引擎
	ActiveTTSModel      string  `json:"active_tts_model,omitempty"`       // 语音合成激活模型
	TTSVoice            string  `json:"tts_voice,omitempty"`              // 语音合成音色
	ActiveOCREngine     string  `json:"active_ocr_engine,omitempty"`      // OCR 激活引擎
	ActiveOCRModel      string  `json:"active_ocr_model,omitempty"`       // OCR 激活模型
	VoicePersonality    string  `json:"voice_personality,omitempty"`      // 语音对话角色
	FuncChatVoiceEngine string  `json:"func_chat_voice_engine,omitempty"` // 聊天语音合成引擎
	FuncChatVoiceModel  string  `json:"func_chat_voice_model,omitempty"`  // 聊天语音合成模型
	FuncChatEngine      string  `json:"func_chat_engine,omitempty"`
	FuncChatModel       string  `json:"func_chat_model,omitempty"`
	// ── 旧品牌遗留（聊天/轻语合并前）：仅用于读取迁移，不再写入 ──
	FuncWhisperEngine  string `json:"func_whisper_engine,omitempty"`
	FuncWhisperModel   string `json:"func_whisper_model,omitempty"`
	FuncWhisperEnabled *bool  `json:"func_whisper_enabled,omitempty"`
	FuncNovelEngine    string `json:"func_novel_engine,omitempty"`
	FuncNovelModel     string `json:"func_novel_model,omitempty"`
	FuncOfficeEngine   string `json:"func_office_engine,omitempty"`
	FuncOfficeModel    string `json:"func_office_model,omitempty"`
	FuncGaeaEngine     string `json:"func_gaea_engine,omitempty"`
	FuncGaeaModel      string `json:"func_gaea_model,omitempty"`
	FuncCharLibEngine  string `json:"func_characterlib_engine,omitempty"`
	FuncCharLibModel   string `json:"func_characterlib_model,omitempty"`
	FuncChatEnabled    *bool  `json:"func_chat_enabled,omitempty"` // nil=默认启用
	FuncNovelEnabled   *bool  `json:"func_novel_enabled,omitempty"`
	FuncOfficeEnabled  *bool  `json:"func_office_enabled,omitempty"`
	FuncGaeaEnabled    *bool  `json:"func_gaea_enabled,omitempty"`
	FuncCharLibEnabled *bool  `json:"func_characterlib_enabled,omitempty"`
	FuncRoutineEngine  string `json:"func_routine_engine,omitempty"` // 常规任务模型目标（routine_llm 工具）
	FuncRoutineModel   string `json:"func_routine_model,omitempty"`
	FuncRoutineEnabled *bool  `json:"func_routine_enabled,omitempty"`
	// 敏感域本地化开关（nil=默认开启，true=成本/报价 AI 走本地 Herdsman）
	SensitiveLocal *bool `json:"sensitive_local,omitempty"`
	// 本地模型调度开关（T5-3a/b，nil=默认开启）
	KeepWarmEnabled *bool `json:"keep_warm_enabled,omitempty"` // 保活探针
	AutoPreload     *bool `json:"auto_preload,omitempty"`      // 启动自动预载
	// 美元→人民币汇率（费用估算折算用；0=未配置，加载时回退默认 7.2）
	UsdCnyRate float64 `json:"usd_cny_rate,omitempty"`
	// CosyVoice 本地 TTS 服务（T6-9.5）：路径/端口可配置，空值回退默认。
	CosyVoiceDir  string `json:"cosyvoice_dir,omitempty"`
	CosyVoicePort int    `json:"cosyvoice_port,omitempty"`
}
type Config struct {
	// XAI OAuth 配置
	XaiClientID   string
	XaiAPIBaseURL string
	RedirectHost  string
	RedirectPort  string

	// OIDC Discovery（自动获取，优先于硬编码）
	OIDCDiscoveryURL string

	// 默认模型
	Model string

	// Token 存储路径
	TokenStorePath string

	// 小说书架目录
	NovelsDir string

	// HTTP 超时（秒）
	HTTPTimeoutSeconds int

	// 默认 temperature（写作温度）
	DefaultTemperature float64

	// 分析/审稿专用参数（Grok 推理优化）
	AnalysisTemperature float64 // 分析类任务温度（建议 0.1-0.3，需精确）
	ReasoningEffort     string  // Grok 推理深度: "low" / "high"（空字符串=不开启）

	// 章节质量自动重试（蒸馏自 MM-StoryAgent 的 success_check_fn + retry loop）
	QualityThreshold  int // 质量阈值 1-10，低于此分自动重试（默认 6）
	QualityMaxRetries int // 最大自动重试次数（默认 2）

	// 资源目录（prompts/ skills/ 等所在的绝对路径）
	ResourceDir string

	// TTS 语音朗读配置
	TTSServerPath string  // 旧版 TTS server 可执行文件路径（保留兼容）
	TTSBinaryPath string  // 旧版 TTS CLI 可执行文件路径
	TTSModelPath  string  // GGUF 模型文件路径
	TTSPort       int     // TTS 服务端口（默认 8765）
	TTSBackend    string  // 推理后端: cpu / cuda / vulkan（默认 cuda）
	TTSSpeed      float64 // 默认朗读语速（0.25-4.0，默认 1.0）
	// 图片生成后端
	ImageBackend      string // "xai" (默认) | "comfyui" | "herdsman" | "ollama"
	ComfyUIURL        string // ComfyUI 服务地址，默认 http://127.0.0.1:8188
	ImageSaveDir      string // 生成图片存放目录，空字符串=不存盘
	ImageModel        string // 图片模型: "grok-imagine-image-quality" (xAI默认) | "flux" | "z-image-turbo"
	PortraitBackend   string // 角色库剧照后端（空=跟随绘梦）
	PortraitModel     string // 角色库剧照模型（空=跟随绘梦）
	ComfyUIPath       string // ComfyUI 安装目录（main.py 所在路径），空=需手动启动
	ComfyUIPythonPath string // Python 解释器路径（留空则自动查找）

	// 活跃模型引擎 ID（"xai" | "ollama" | "herdsman" | "deepseek"）
	ActiveEngineID string

	// DeepSeek API Key
	DeepseekAPIKey string

	// OpenCode Go API Key（opencode.ai 订阅，模型中心配置）
	OpenCodeGoAPIKey string

	// OpenCode Zen API Key（按量付费，opencode.ai/auth 获取）
	OpenCodeZenAPIKey string

	// 语音识别激活引擎 + 模型（来自模型中心选择，空=自动）
	ActiveASREngine string
	ActiveASRModel  string

	// 语音合成激活引擎 + 模型（来自模型中心选择，空=自动）
	ActiveTTSEngine     string
	ActiveTTSModel      string
	TTSVoice            string // 语音合成音色（来自设置面板，空=按模型默认）
	ActiveOCREngine     string // OCR 激活引擎（空=自动）
	ActiveOCRModel      string // OCR 激活模型（空=自动）
	VoicePersonality    string // 语音对话角色（与聊天板块一致，空=gaea）
	FuncChatVoiceEngine string // 聊天语音合成引擎（功能绑定，空=全局 TTS）
	FuncChatVoiceModel  string // 聊天语音合成模型

	// 功能级模型绑定（各功能独立 LLM，空=用全局激活引擎+模型）
	FuncChatEngine    string
	FuncChatModel     string
	FuncNovelEngine   string
	FuncNovelModel    string
	FuncOfficeEngine  string
	FuncOfficeModel   string
	FuncGaeaEngine    string
	FuncGaeaModel     string
	FuncCharLibEngine string
	FuncCharLibModel  string
	// 常规任务模型目标（routine）：routine_llm 工具默认调用的引擎/模型，
	// 供云端 agent 按需把摘要/归一化/抽取等简单活卸给本地或免费云端模型。
	// 不参与强制路由——是否调用由云端 agent 自行决定。
	FuncRoutineEngine string
	FuncRoutineModel  string
	// 功能级启停（默认启用；停用后该功能路由回退全局）
	FuncChatEnabled    bool
	FuncNovelEnabled   bool
	FuncOfficeEnabled  bool
	FuncGaeaEnabled    bool
	FuncCharLibEnabled bool
	FuncRoutineEnabled bool

	// 敏感域本地化（S2-4/D8）：成本/报价类 AI 操作默认路由本地 Herdsman。
	// true=本地优先（默认）；false=按常规路由（可回云端）。
	SensitiveLocal bool

	// 本地模型调度（T5-3a/b，默认开启）：
	//   KeepWarmEnabled：保活——周期性对已运行的本地模型发轻量探针，防止被
	//                     herdsman 空闲卸载/降温，保持「说用就能用」；
	//   AutoPreload：启动自动预载——按功能绑定（gaea→office→chat）预载一个
	//                 herdsman 模型，降低首次对话的冷启动等待。
	KeepWarmEnabled bool
	AutoPreload     bool

	// 美元→人民币汇率（费用估算折算用，默认 7.2；模型中心可配置）
	UsdCnyRate float64

	// CosyVoice 本地 TTS 服务（T6-9.5）：路径/端口可配置，默认 C:\AI\cosyvoice / 8010。
	CosyVoiceDir  string
	CosyVoicePort int
}

// funcMu 保护功能级模型绑定字段（GetFeatureModel/SetFeatureModel 并发读写）
var funcMu sync.RWMutex

// GetFeatureModel 读取功能绑定的 (engine, model)，空 = 用全局激活
func (c *Config) GetFeatureModel(feature string) (engine, model string) {
	funcMu.RLock()
	defer funcMu.RUnlock()
	switch feature {
	case "chat":
		return c.FuncChatEngine, c.FuncChatModel
	case "whisper":
		// 2.x 聊天/轻语合并：轻语绑定并入聊天，查询走 chat 别名
		return c.FuncChatEngine, c.FuncChatModel
	case "novel":
		return c.FuncNovelEngine, c.FuncNovelModel
	case "office":
		return c.FuncOfficeEngine, c.FuncOfficeModel
	case "gaea":
		return c.FuncGaeaEngine, c.FuncGaeaModel
	case "characterlib":
		return c.FuncCharLibEngine, c.FuncCharLibModel
	case "routine":
		return c.FuncRoutineEngine, c.FuncRoutineModel
	}
	return "", ""
}

// SetFeatureModel 写入功能绑定的 (engine, model)
func (c *Config) SetFeatureModel(feature, engine, model string) {
	funcMu.Lock()
	defer funcMu.Unlock()
	switch feature {
	case "chat":
		c.FuncChatEngine, c.FuncChatModel = engine, model
	case "whisper":
		// 2.x 聊天/轻语合并：写入 chat 绑定
		c.FuncChatEngine, c.FuncChatModel = engine, model
	case "novel":
		c.FuncNovelEngine, c.FuncNovelModel = engine, model
	case "office":
		c.FuncOfficeEngine, c.FuncOfficeModel = engine, model
	case "gaea":
		c.FuncGaeaEngine, c.FuncGaeaModel = engine, model
	case "characterlib":
		c.FuncCharLibEngine, c.FuncCharLibModel = engine, model
	case "routine":
		c.FuncRoutineEngine, c.FuncRoutineModel = engine, model
	}
}

// GetFeatureModelEnabled 读取功能级启停状态（未显式配置时默认启用）。
func (c *Config) GetFeatureModelEnabled(feature string) bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	switch feature {
	case "chat":
		return c.FuncChatEnabled
	case "whisper":
		return c.FuncChatEnabled
	case "novel":
		return c.FuncNovelEnabled
	case "office":
		return c.FuncOfficeEnabled
	case "gaea":
		return c.FuncGaeaEnabled
	case "characterlib":
		return c.FuncCharLibEnabled
	case "routine":
		return c.FuncRoutineEnabled
	}
	return true
}

// SetFeatureModelEnabled 写入功能级启停状态。
func (c *Config) SetFeatureModelEnabled(feature string, enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	switch feature {
	case "chat":
		c.FuncChatEnabled = enabled
	case "whisper":
		c.FuncChatEnabled = enabled
	case "novel":
		c.FuncNovelEnabled = enabled
	case "office":
		c.FuncOfficeEnabled = enabled
	case "gaea":
		c.FuncGaeaEnabled = enabled
	case "characterlib":
		c.FuncCharLibEnabled = enabled
	case "routine":
		c.FuncRoutineEnabled = enabled
	}
}

// GetSensitiveLocal 读取敏感域本地化开关（未显式配置时默认开启）。
func (c *Config) GetSensitiveLocal() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.SensitiveLocal
}

// SetSensitiveLocal 写入敏感域本地化开关（true=成本/报价 AI 走本地 Herdsman）。
func (c *Config) SetSensitiveLocal(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.SensitiveLocal = enabled
}

// GetKeepWarm 读取本地模型保活开关（T5-3a，未显式配置时默认开启）。
func (c *Config) GetKeepWarm() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.KeepWarmEnabled
}

// SetKeepWarm 写入本地模型保活开关（true=周期性探活已运行模型）。
func (c *Config) SetKeepWarm(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.KeepWarmEnabled = enabled
}

// GetAutoPreload 读取启动自动预载开关（T5-3b，未显式配置时默认开启）。
func (c *Config) GetAutoPreload() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.AutoPreload
}

// SetAutoPreload 写入启动自动预载开关。
func (c *Config) SetAutoPreload(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.AutoPreload = enabled
}

// Load 加载配置（只应调用一次）。
// 优先级：config 文件 > 环境变量 > 默认值。
func Load() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("获取用户主目录失败", "error", err)
	}
	tokenPath := filepath.Join(home, ".gaea_token.json")

	// 1. 硬编码默认值（最低优先级）
	cfg := &Config{
		XaiClientID:      "b1a00492-073a-47ea-816f-4c329264a828",
		XaiAPIBaseURL:    "https://api.x.ai/v1",
		RedirectHost:     "127.0.0.1",
		RedirectPort:     "56121",
		OIDCDiscoveryURL: "https://auth.x.ai/.well-known/openid-configuration",
		Model:            "grok-4.20",
		TokenStorePath:   tokenPath,
		// 本机单用户定位：小说目录固定为 C:\AI\xiaoshuo（记忆：novels-directory）
		NovelsDir:           `C:\AI\xiaoshuo`,
		HTTPTimeoutSeconds:  180,
		DefaultTemperature:  0.7,
		AnalysisTemperature: 0.15,   // 分析任务低温度以确保精确
		ReasoningEffort:     "high", // 分析任务默认开启深度推理
		QualityThreshold:    6,      // 章节质量低于 6 分自动重试
		QualityMaxRetries:   2,      // 最多重试 2 次
		// 功能级模型默认启用（未显式停用时，绑定立即生效）
		FuncChatEnabled:    true,
		FuncNovelEnabled:   true,
		FuncOfficeEnabled:  true,
		FuncGaeaEnabled:    true,
		FuncCharLibEnabled: true,
		FuncRoutineEnabled: true, // 常规办公默认启用：routine_llm 工具按绑定目标执行
		// S2-4/D8：敏感域（成本/报价）AI 默认本地优先。
		SensitiveLocal: true,
		// T5-3a/b：本地模型保活 + 启动自动预载默认开启。
		KeepWarmEnabled: true,
		AutoPreload:     true,
		// 汇率默认 7.2（费用估算折算口径）。
		UsdCnyRate: DefaultUsdCnyRate,
		// CosyVoice 本地 TTS 服务（T6-9.5，默认与历史硬编码一致）。
		CosyVoiceDir:  DefaultCosyVoiceDir,
		CosyVoicePort: DefaultCosyVoicePort,

		// TTS 默认值
		TTSBinaryPath: filepath.Join(home, "legacy-tts", "legacy_tts.exe"),
		TTSServerPath: "", // 默认不设置，优先用 TTSBinaryPath
		TTSModelPath:  filepath.Join(home, "legacy-tts", "models", "legacy-tts-model.gguf"),
		TTSPort:       8765,
		TTSBackend:    "cpu",
		TTSSpeed:      1.0,

		// 图片生成默认值
		ImageBackend: "xai",
		ComfyUIURL:   "http://127.0.0.1:8188",
		ImageSaveDir: "", // 默认不存盘
		ImageModel:   "grok-imagine-image-quality",
		// 本机单用户定位：ComfyUI 启动位置直接写死（gaea 仅此电脑使用）
		ComfyUIPath:       `C:\AI\ComfyUI\ComfyUI`,
		ComfyUIPythonPath: `C:\AI\ComfyUI\standalone-env\python.exe`,
	}

	// 2. 环境变量覆盖（中优先级）
	if v := os.Getenv("WUBI_XAI_CLIENT_ID"); v != "" {
		cfg.XaiClientID = v
	}
	if v := os.Getenv("XAI_API_BASE_URL"); v != "" {
		cfg.XaiAPIBaseURL = v
	}
	if v := os.Getenv("XAI_REDIRECT_HOST"); v != "" {
		cfg.RedirectHost = v
	}
	if v := os.Getenv("XAI_REDIRECT_PORT"); v != "" {
		cfg.RedirectPort = v
	}
	if v := os.Getenv("XAI_OIDC_DISCOVERY_URL"); v != "" {
		cfg.OIDCDiscoveryURL = v
	}
	if v := os.Getenv("WUBI_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("WUBI_TOKEN_PATH"); v != "" {
		cfg.TokenStorePath = v
	}
	if v := os.Getenv("WUBI_NOVELS_DIR"); v != "" {
		cfg.NovelsDir = v
	}
	if v := os.Getenv("WUBI_HTTP_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HTTPTimeoutSeconds = n
		}
	}
	if v := os.Getenv("WUBI_DEFAULT_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.DefaultTemperature = f
		}
	}
	if v := os.Getenv("WUBI_ANALYSIS_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.AnalysisTemperature = f
		}
	}
	if v := os.Getenv("WUBI_REASONING_EFFORT"); v != "" {
		cfg.ReasoningEffort = v
	}
	if v := os.Getenv("WUBI_QUALITY_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.QualityThreshold = n
		}
	}
	if v := os.Getenv("WUBI_QUALITY_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.QualityMaxRetries = n
		}
	}

	// TTS 环境变量覆盖
	if v := os.Getenv("WUBI_TTS_SERVER_PATH"); v != "" {
		cfg.TTSServerPath = v
	}
	if v := os.Getenv("WUBI_TTS_BINARY_PATH"); v != "" {
		cfg.TTSBinaryPath = v
	}
	if v := os.Getenv("WUBI_TTS_MODEL_PATH"); v != "" {
		cfg.TTSModelPath = v
	}
	if v := os.Getenv("WUBI_TTS_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.TTSPort = n
		}
	}
	if v := os.Getenv("WUBI_TTS_BACKEND"); v != "" {
		cfg.TTSBackend = v
	}
	if v := os.Getenv("WUBI_TTS_SPEED"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.TTSSpeed = f
		}
	}

	// 3. Config 文件覆盖（最高优先级）
	// 兼容旧品牌：优先读 .gaea_config.json，不存在时回退 .wubigork_config.json（老用户配置迁移）
	configPath := filepath.Join(home, ".gaea_config.json")
	var data []byte
	if d, err := os.ReadFile(configPath); err == nil {
		data = d
	} else if legacy, lerr := os.ReadFile(filepath.Join(home, ".wubigork_config.json")); lerr == nil {
		data = legacy
		slog.Warn("主配置文件读取失败，回退旧品牌配置", "path", configPath, "error", err)
	} else if !os.IsNotExist(err) {
		// 两个配置文件都不存在是首次启动的正常情况；其余读取错误（权限/损坏路径）需可见
		slog.Warn("配置文件读取失败（主/旧品牌均不可用）", "path", configPath, "error", err)
	}
	if data != nil {
		var cf configFile
		if err := json.Unmarshal(data, &cf); err == nil {
			if cf.XaiClientID != "" {
				cfg.XaiClientID = cf.XaiClientID
			}
			if cf.NovelsDir != "" {
				cfg.NovelsDir = cf.NovelsDir
			}
			if cf.HTTPTimeoutSeconds != 0 {
				cfg.HTTPTimeoutSeconds = cf.HTTPTimeoutSeconds
			}
			if cf.DefaultTemperature != 0 {
				cfg.DefaultTemperature = cf.DefaultTemperature
			}
			if cf.AnalysisTemperature != 0 {
				cfg.AnalysisTemperature = cf.AnalysisTemperature
			}
			if cf.ReasoningEffort != "" {
				cfg.ReasoningEffort = cf.ReasoningEffort
			}
			if cf.QualityThreshold != 0 {
				cfg.QualityThreshold = cf.QualityThreshold
			}
			if cf.QualityMaxRetries != 0 {
				cfg.QualityMaxRetries = cf.QualityMaxRetries
			}
			if cf.TTSBinaryPath != "" {
				cfg.TTSBinaryPath = cf.TTSBinaryPath
			}
			if cf.TTSModelPath != "" {
				cfg.TTSModelPath = cf.TTSModelPath
			}
			if cf.ImageBackend != "" {
				cfg.ImageBackend = cf.ImageBackend
			}
			if cf.ComfyUIURL != "" {
				cfg.ComfyUIURL = cf.ComfyUIURL
			}
			if cf.ImageSaveDir != "" {
				cfg.ImageSaveDir = cf.ImageSaveDir
			}
			if cf.ImageModel != "" {
				cfg.ImageModel = cf.ImageModel
			}
			if cf.PortraitBackend != "" {
				cfg.PortraitBackend = cf.PortraitBackend
			}
			if cf.PortraitModel != "" {
				cfg.PortraitModel = cf.PortraitModel
			}
			if cf.ComfyUIPath != "" {
				cfg.ComfyUIPath = cf.ComfyUIPath
			}
			if cf.ComfyUIPythonPath != "" {
				cfg.ComfyUIPythonPath = cf.ComfyUIPythonPath
			}
			if cf.TTSPort != 0 {
				cfg.TTSPort = cf.TTSPort
			}
			if cf.TTSSpeed != 0 {
				cfg.TTSSpeed = cf.TTSSpeed
			}
			if cf.Model != "" {
				cfg.Model = cf.Model
			}
			if cf.ActiveEngineID != "" {
				cfg.ActiveEngineID = cf.ActiveEngineID
			}
			if cf.DeepseekAPIKey != "" {
				cfg.DeepseekAPIKey = cf.DeepseekAPIKey
			}
			if cf.OpenCodeGoAPIKey != "" {
				cfg.OpenCodeGoAPIKey = cf.OpenCodeGoAPIKey
			}
			if cf.OpenCodeZenAPIKey != "" {
				cfg.OpenCodeZenAPIKey = cf.OpenCodeZenAPIKey
			}
			if cf.ActiveASREngine != "" {
				cfg.ActiveASREngine = cf.ActiveASREngine
			}
			if cf.ActiveASRModel != "" {
				cfg.ActiveASRModel = cf.ActiveASRModel
			}
			if cf.ActiveTTSEngine != "" {
				cfg.ActiveTTSEngine = cf.ActiveTTSEngine
			}
			if cf.ActiveTTSModel != "" {
				cfg.ActiveTTSModel = cf.ActiveTTSModel
			}
			if cf.TTSVoice != "" {
				cfg.TTSVoice = cf.TTSVoice
			}
			if cf.ActiveOCREngine != "" {
				cfg.ActiveOCREngine = cf.ActiveOCREngine
			}
			if cf.ActiveOCRModel != "" {
				cfg.ActiveOCRModel = cf.ActiveOCRModel
			}
			if cf.VoicePersonality != "" {
				cfg.VoicePersonality = cf.VoicePersonality
			}
			if cf.FuncChatVoiceEngine != "" {
				cfg.FuncChatVoiceEngine = cf.FuncChatVoiceEngine
			}
			if cf.FuncChatVoiceModel != "" {
				cfg.FuncChatVoiceModel = cf.FuncChatVoiceModel
			}
			if cf.FuncChatEngine != "" {
				cfg.FuncChatEngine = cf.FuncChatEngine
			}
			if cf.FuncChatModel != "" {
				cfg.FuncChatModel = cf.FuncChatModel
			}
			if cf.FuncNovelEngine != "" {
				cfg.FuncNovelEngine = cf.FuncNovelEngine
			}
			if cf.FuncNovelModel != "" {
				cfg.FuncNovelModel = cf.FuncNovelModel
			}
			if cf.FuncOfficeEngine != "" {
				cfg.FuncOfficeEngine = cf.FuncOfficeEngine
			}
			if cf.FuncOfficeModel != "" {
				cfg.FuncOfficeModel = cf.FuncOfficeModel
			}
			if cf.FuncGaeaEngine != "" {
				cfg.FuncGaeaEngine = cf.FuncGaeaEngine
			}
			if cf.FuncGaeaModel != "" {
				cfg.FuncGaeaModel = cf.FuncGaeaModel
			}
			if cf.FuncCharLibEngine != "" {
				cfg.FuncCharLibEngine = cf.FuncCharLibEngine
			}
			if cf.FuncCharLibModel != "" {
				cfg.FuncCharLibModel = cf.FuncCharLibModel
			}
			if cf.FuncChatEnabled != nil {
				cfg.FuncChatEnabled = *cf.FuncChatEnabled
			}
			if cf.FuncNovelEnabled != nil {
				cfg.FuncNovelEnabled = *cf.FuncNovelEnabled
			}
			if cf.FuncOfficeEnabled != nil {
				cfg.FuncOfficeEnabled = *cf.FuncOfficeEnabled
			}
			if cf.FuncGaeaEnabled != nil {
				cfg.FuncGaeaEnabled = *cf.FuncGaeaEnabled
			}
			if cf.FuncCharLibEnabled != nil {
				cfg.FuncCharLibEnabled = *cf.FuncCharLibEnabled
			}
			if cf.FuncRoutineEngine != "" {
				cfg.FuncRoutineEngine = cf.FuncRoutineEngine
			}
			if cf.FuncRoutineModel != "" {
				cfg.FuncRoutineModel = cf.FuncRoutineModel
			}
			if cf.FuncRoutineEnabled != nil {
				cfg.FuncRoutineEnabled = *cf.FuncRoutineEnabled
			}
			if cf.SensitiveLocal != nil {
				cfg.SensitiveLocal = *cf.SensitiveLocal
			}
			if cf.KeepWarmEnabled != nil {
				cfg.KeepWarmEnabled = *cf.KeepWarmEnabled
			}
			if cf.AutoPreload != nil {
				cfg.AutoPreload = *cf.AutoPreload
			}
			if cf.UsdCnyRate != 0 {
				cfg.UsdCnyRate = cf.UsdCnyRate
			}
			if cf.CosyVoiceDir != "" {
				cfg.CosyVoiceDir = cf.CosyVoiceDir
			}
			if cf.CosyVoicePort != 0 {
				cfg.CosyVoicePort = cf.CosyVoicePort
			}
			// 2.x 聊天/轻语合并：旧配置只写 func_whisper_* 时迁移到 func_chat；
			// chat 显式配置优先，不覆盖；func_whisper_enabled=false 同步为 chat 停用。
			if cfg.FuncChatEngine == "" && cf.FuncWhisperEngine != "" {
				cfg.FuncChatEngine = cf.FuncWhisperEngine
				cfg.FuncChatModel = cf.FuncWhisperModel
				if cf.FuncWhisperEnabled != nil {
					cfg.FuncChatEnabled = *cf.FuncWhisperEnabled
				}
			}
		} else {
			// 损坏恢复（T6-9.4）：把损坏文件备份为 .gaea_config.json.corrupt-<时间戳>
			// （不丢用户数据），再用默认值继续——应用可正常启动，设置重置但文件可追溯。
			backup := filepath.Join(home, fmt.Sprintf(".gaea_config.json.corrupt-%d", time.Now().UnixNano()))
			if berr := os.WriteFile(backup, data, 0644); berr != nil {
				slog.Error("配置文件损坏且备份失败", "path", configPath, "error", err, "backup_error", berr)
			} else {
				slog.Warn("配置文件解析失败，已备份并重置为默认值", "path", configPath, "backup", backup, "error", err)
			}
		}
	}

	// 4. 解析资源目录（prompts/ skills/ 等）
	cfg.ResourceDir = resolveResourceDir()

	return cfg
}

// resolveResourceDir 找到 prompts/ 和 skills/ 所在的资源根目录。
// 优先基于 os.Executable() 向上查找，回退到 CWD。
func resolveResourceDir() string {
	// 环境变量优先：部署/开发可显式指定资源根，避免桌面副本找不到 prompts
	// 导致数据目录分裂（统计/引擎状态落在 exe 所在目录）。
	if v := os.Getenv("GAEA_RESOURCE_DIR"); v != "" {
		if dirExists(filepath.Join(v, "prompts")) {
			return v
		}
	}
	// 尝试从可执行文件路径向上查找
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for range 4 {
			if dirExists(filepath.Join(dir, "prompts")) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	// 用户级数据根（%APPDATA%/gaea）：exe 放在任意位置（如桌面）也能找到资源与数据。
	if ud, err := os.UserConfigDir(); err == nil {
		userRoot := filepath.Join(ud, "gaea")
		if dirExists(filepath.Join(userRoot, "prompts")) {
			return userRoot
		}
	}
	// 回退：当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		if dirExists(filepath.Join(cwd, "prompts")) {
			return cwd
		}
	}
	return "."
}

// ResolveResourceDirForTest 暴露资源目录解析结果（仅测试/诊断用）。
func ResolveResourceDirForTest() string {
	return resolveResourceDir()
}

// DataRoot 返回用户级数据根目录（引擎状态/模型统计/轻语/聊天/角色库等）。
// 与 exe 位置无关：桌面副本或任意路径启动都读写同一份数据，避免统计/状态
// 因 ResourceDir 解析差异而分裂。优先级：GAEA_DATA_ROOT > 用户配置目录/gaea
// > 回退 ResourceDir（历史行为，防止取不到用户目录时数据丢失）。
func DataRoot() string {
	if v := os.Getenv("GAEA_DATA_ROOT"); v != "" {
		return v
	}
	if ud, err := os.UserConfigDir(); err == nil && ud != "" {
		return filepath.Join(ud, "gaea")
	}
	return resolveResourceDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// saveMu 串行化 Save 的 read-modify-write（并发写不同 key 不互相覆盖）
var saveMu sync.Mutex

// Save 将单个配置项写回 ~/.gaea_config.json。
// 使用 config 包的 Key* 常量指定 key。
func Save(key, value string) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".gaea_config.json")

	var cf configFile
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &cf)
	}

	setter, ok := saveSetters[key]
	if !ok {
		return fmt.Errorf("不支持的配置项: %s", key)
	}
	if err := setter(&cf, value); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	return saveConfigFile(configPath, data)
}

// renameFile 覆盖写目标文件；抽为变量便于测试注入失败路径。
var renameFile = os.Rename

// saveConfigFile 原子写配置文件（T6-9.4）：同目录临时文件 → 写入 → fsync → rename 覆盖。
// 任一步失败都会清理临时文件并保留原文件不破坏（中断不会截断/半写配置文件）。
func saveConfigFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := renameFile(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// saveSetters 配置项 → setter 函数注册表
var saveSetters = map[string]func(cf *configFile, value string) error{
	KeyNovelsDir:   func(cf *configFile, v string) error { cf.NovelsDir = v; return nil },
	KeyXaiClientID: func(cf *configFile, v string) error { cf.XaiClientID = v; return nil },
	KeyHTTPTimeoutSeconds: func(cf *configFile, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		cf.HTTPTimeoutSeconds = n
		return nil
	},
	KeyDefaultTemperature: func(cf *configFile, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		cf.DefaultTemperature = f
		return nil
	},
	KeyAnalysisTemperature: func(cf *configFile, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		cf.AnalysisTemperature = f
		return nil
	},
	KeyReasoningEffort: func(cf *configFile, v string) error { cf.ReasoningEffort = v; return nil },
	KeyQualityThreshold: func(cf *configFile, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		cf.QualityThreshold = n
		return nil
	},
	KeyQualityMaxRetries: func(cf *configFile, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		cf.QualityMaxRetries = n
		return nil
	},
	KeyTTSBinaryPath:     func(cf *configFile, v string) error { cf.TTSBinaryPath = v; return nil },
	KeyTTSModelPath:      func(cf *configFile, v string) error { cf.TTSModelPath = v; return nil },
	KeyImageBackend:      func(cf *configFile, v string) error { cf.ImageBackend = v; return nil },
	KeyComfyUIURL:        func(cf *configFile, v string) error { cf.ComfyUIURL = v; return nil },
	KeyImageSaveDir:      func(cf *configFile, v string) error { cf.ImageSaveDir = v; return nil },
	KeyImageModel:        func(cf *configFile, v string) error { cf.ImageModel = v; return nil },
	KeyPortraitBackend:   func(cf *configFile, v string) error { cf.PortraitBackend = v; return nil },
	KeyPortraitModel:     func(cf *configFile, v string) error { cf.PortraitModel = v; return nil },
	KeyComfyUIPath:       func(cf *configFile, v string) error { cf.ComfyUIPath = v; return nil },
	KeyComfyUIPythonPath: func(cf *configFile, v string) error { cf.ComfyUIPythonPath = v; return nil },
	KeyTTSPort: func(cf *configFile, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		cf.TTSPort = n
		return nil
	},
	KeyCosyVoiceDir: func(cf *configFile, v string) error { cf.CosyVoiceDir = v; return nil },
	KeyCosyVoicePort: func(cf *configFile, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		if n < 1 || n > 65535 {
			return fmt.Errorf("CosyVoice 端口必须在 1-65535 之间（当前值: %s）", v)
		}
		cf.CosyVoicePort = n
		return nil
	},
	KeyTTSSpeed: func(cf *configFile, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		if f < 0.25 || f > 4.0 {
			return fmt.Errorf("语速必须在 0.25-4.0 之间（当前值: %s）", v)
		}
		cf.TTSSpeed = f
		return nil
	},
	KeyTTSBackend:          func(cf *configFile, v string) error { cf.TTSBackend = v; return nil },
	KeyActiveEngineID:      func(cf *configFile, v string) error { cf.ActiveEngineID = v; return nil },
	KeyModel:               func(cf *configFile, v string) error { cf.Model = v; return nil },
	KeyDeepseekAPIKey:      func(cf *configFile, v string) error { cf.DeepseekAPIKey = v; return nil },
	KeyOpencodeGoAPIKey:    func(cf *configFile, v string) error { cf.OpenCodeGoAPIKey = v; return nil },
	KeyOpencodeZenAPIKey:   func(cf *configFile, v string) error { cf.OpenCodeZenAPIKey = v; return nil },
	KeyActiveASREngine:     func(cf *configFile, v string) error { cf.ActiveASREngine = v; return nil },
	KeyActiveASRModel:      func(cf *configFile, v string) error { cf.ActiveASRModel = v; return nil },
	KeyActiveTTSEngine:     func(cf *configFile, v string) error { cf.ActiveTTSEngine = v; return nil },
	KeyActiveTTSModel:      func(cf *configFile, v string) error { cf.ActiveTTSModel = v; return nil },
	KeyTTSVoice:            func(cf *configFile, v string) error { cf.TTSVoice = v; return nil },
	KeyActiveOCREngine:     func(cf *configFile, v string) error { cf.ActiveOCREngine = v; return nil },
	KeyActiveOCRModel:      func(cf *configFile, v string) error { cf.ActiveOCRModel = v; return nil },
	KeyVoicePersonality:    func(cf *configFile, v string) error { cf.VoicePersonality = v; return nil },
	KeyFuncChatVoiceEngine: func(cf *configFile, v string) error { cf.FuncChatVoiceEngine = v; return nil },
	KeyFuncChatVoiceModel:  func(cf *configFile, v string) error { cf.FuncChatVoiceModel = v; return nil },
	KeyFuncChatEngine:      func(cf *configFile, v string) error { cf.FuncChatEngine = v; return nil },
	KeyFuncChatModel:       func(cf *configFile, v string) error { cf.FuncChatModel = v; return nil },
	KeyFuncNovelEngine:     func(cf *configFile, v string) error { cf.FuncNovelEngine = v; return nil },
	KeyFuncNovelModel:      func(cf *configFile, v string) error { cf.FuncNovelModel = v; return nil },
	KeyFuncOfficeEngine:    func(cf *configFile, v string) error { cf.FuncOfficeEngine = v; return nil },
	KeyFuncOfficeModel:     func(cf *configFile, v string) error { cf.FuncOfficeModel = v; return nil },
	KeyFuncGaeaEngine:      func(cf *configFile, v string) error { cf.FuncGaeaEngine = v; return nil },
	KeyFuncGaeaModel:       func(cf *configFile, v string) error { cf.FuncGaeaModel = v; return nil },
	KeyFuncCharLibEngine:   func(cf *configFile, v string) error { cf.FuncCharLibEngine = v; return nil },
	KeyFuncCharLibModel:    func(cf *configFile, v string) error { cf.FuncCharLibModel = v; return nil },
	KeyFuncChatEnabled: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.FuncChatEnabled = b
		return nil
	},
	KeyFuncNovelEnabled: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.FuncNovelEnabled = b
		return nil
	},
	KeyFuncOfficeEnabled: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.FuncOfficeEnabled = b
		return nil
	},
	KeyFuncGaeaEnabled: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.FuncGaeaEnabled = b
		return nil
	},
	KeyFuncCharLibEnabled: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.FuncCharLibEnabled = b
		return nil
	},
	KeyFuncRoutineEngine: func(cf *configFile, v string) error { cf.FuncRoutineEngine = v; return nil },
	KeyFuncRoutineModel:  func(cf *configFile, v string) error { cf.FuncRoutineModel = v; return nil },
	KeyFuncRoutineEnabled: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.FuncRoutineEnabled = b
		return nil
	},
	KeySensitiveLocal: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.SensitiveLocal = b
		return nil
	},
	KeyKeepWarm: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.KeepWarmEnabled = b
		return nil
	},
	KeyAutoPreload: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.AutoPreload = b
		return nil
	},
	KeyUsdCnyRate: func(cf *configFile, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		if f <= 0 || math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("汇率必须为正数（当前值: %s）", v)
		}
		cf.UsdCnyRate = f
		return nil
	},
}

// parseBoolPtr 解析 "true"/"1"/"0" 等布尔值并返回指针（用于 *bool 配置项）。
func parseBoolPtr(v string) (*bool, error) {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
