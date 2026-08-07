package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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
	KeyComfyUIPath         = "comfyui_path"
	KeyComfyUIPythonPath   = "comfyui_python_path"
	KeyActiveEngineID      = "active_engine_id"
	KeyModel               = "model"
	KeyActiveASREngine     = "active_asr_engine" // 语音识别激活引擎
	KeyActiveASRModel      = "active_asr_model"  // 语音识别激活模型
	KeyActiveTTSEngine     = "active_tts_engine" // 语音合成激活引擎
	KeyActiveTTSModel      = "active_tts_model"  // 语音合成激活模型
	// 功能级模型绑定（聊天/轻语/小说/办公 各自独立 LLM，持久化重启不丢）
	KeyFuncChatEngine      = "func_chat_engine"
	KeyFuncChatModel       = "func_chat_model"
	KeyFuncNovelEngine     = "func_novel_engine"
	KeyFuncNovelModel      = "func_novel_model"
	KeyFuncOfficeEngine    = "func_office_engine"
	KeyFuncOfficeModel     = "func_office_model"
	KeyFuncGaeaEngine      = "func_gaea_engine"
	KeyFuncGaeaModel       = "func_gaea_model"
	// 功能级启停（FeatureModelBar 启停语义：只影响该功能的路由，不影响整个引擎）
	KeyFuncChatEnabled      = "func_chat_enabled"
	KeyFuncNovelEnabled     = "func_novel_enabled"
	KeyFuncOfficeEnabled    = "func_office_enabled"
	KeyFuncGaeaEnabled      = "func_gaea_enabled"
	KeyDeepseekAPIKey      = "deepseek_api_key"
)

// configFile 表示 ~/.gaea_config.json 的结构
type configFile struct {
	XaiClientID         string  `json:"xai_client_id"`
	NovelsDir           string  `json:"novels_dir"`
	HTTPTimeoutSeconds  int     `json:"http_timeout_seconds"`
	DefaultTemperature  float64 `json:"default_temperature"`
	AnalysisTemperature float64 `json:"analysis_temperature"`
	ReasoningEffort     string  `json:"reasoning_effort"` // Grok 推理深度: "low" / "high"
	QualityThreshold    int     `json:"quality_threshold"` // 章节质量阈值 1-10，低于此触发自动重试
	QualityMaxRetries   int     `json:"quality_max_retries"` // 最大自动重试次数
	TTSBinaryPath       string  `json:"tts_binary_path,omitempty"`
	TTSModelPath        string  `json:"tts_model_path,omitempty"`
	ImageBackend        string  `json:"image_backend,omitempty"`  // "xai" (默认) | "comfyui" | "herdsman" | "ollama"
	ComfyUIURL          string  `json:"comfyui_url,omitempty"`
	ImageSaveDir        string  `json:"image_save_dir,omitempty"`   // 图片生成存放目录
	ImageModel          string  `json:"image_model,omitempty"`    // 图片模型
	ComfyUIPath         string  `json:"comfyui_path,omitempty"`  // ComfyUI 安装目录
	ComfyUIPythonPath   string  `json:"comfyui_python_path,omitempty"` // Python 解释器路径
	TTSPort             int     `json:"tts_port,omitempty"`       // TTS 服务端口
	TTSBackend          string  `json:"tts_backend,omitempty"`    // TTS 后端: "cpu" | "cuda"
	TTSSpeed            float64 `json:"tts_speed,omitempty"`      // TTS 语速
	ActiveEngineID      string  `json:"active_engine_id,omitempty"` // 活跃模型引擎 ID
	Model               string  `json:"model,omitempty"`             // 默认 LLM 模型名
	DeepseekAPIKey      string  `json:"deepseek_api_key,omitempty"`  // DeepSeek API Key
	ActiveASREngine     string  `json:"active_asr_engine,omitempty"` // 语音识别激活引擎
	ActiveASRModel      string  `json:"active_asr_model,omitempty"`  // 语音识别激活模型
	ActiveTTSEngine     string  `json:"active_tts_engine,omitempty"` // 语音合成激活引擎
	ActiveTTSModel      string  `json:"active_tts_model,omitempty"`  // 语音合成激活模型
	FuncChatEngine      string  `json:"func_chat_engine,omitempty"`
	FuncChatModel       string  `json:"func_chat_model,omitempty"`
	// ── 旧品牌遗留（聊天/轻语合并前）：仅用于读取迁移，不再写入 ──
	FuncWhisperEngine   string  `json:"func_whisper_engine,omitempty"`
	FuncWhisperModel    string  `json:"func_whisper_model,omitempty"`
	FuncWhisperEnabled  *bool   `json:"func_whisper_enabled,omitempty"`
	FuncNovelEngine     string  `json:"func_novel_engine,omitempty"`
	FuncNovelModel      string  `json:"func_novel_model,omitempty"`
	FuncOfficeEngine    string  `json:"func_office_engine,omitempty"`
	FuncOfficeModel     string  `json:"func_office_model,omitempty"`
	FuncGaeaEngine      string  `json:"func_gaea_engine,omitempty"`
	FuncGaeaModel       string  `json:"func_gaea_model,omitempty"`
	FuncChatEnabled     *bool   `json:"func_chat_enabled,omitempty"`    // nil=默认启用
	FuncNovelEnabled    *bool   `json:"func_novel_enabled,omitempty"`
	FuncOfficeEnabled   *bool   `json:"func_office_enabled,omitempty"`
	FuncGaeaEnabled     *bool   `json:"func_gaea_enabled,omitempty"`
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
	TTSServerPath string  // voxcpm-server 可执行文件路径（保留兼容）
	TTSBinaryPath string  // voxcpm_tts CLI 可执行文件路径
	TTSModelPath  string  // GGUF 模型文件路径
	TTSPort       int     // TTS 服务端口（默认 8765）
	TTSBackend    string  // 推理后端: cpu / cuda / vulkan（默认 cuda）
	TTSSpeed      float64 // 默认朗读语速（0.25-4.0，默认 1.0）
	// 图片生成后端
	ImageBackend string // "xai" (默认) | "comfyui" | "herdsman" | "ollama"
	ComfyUIURL   string // ComfyUI 服务地址，默认 http://127.0.0.1:8188
	ImageSaveDir string // 生成图片存放目录，空字符串=不存盘
	ImageModel   string // 图片模型: "grok-imagine-image-quality" (xAI默认) | "flux" | "z-image-turbo"
	ComfyUIPath string // ComfyUI 安装目录（main.py 所在路径），空=需手动启动
	ComfyUIPythonPath string // Python 解释器路径（留空则自动查找）

	// 活跃模型引擎 ID（"xai" | "ollama" | "herdsman" | "deepseek"）
	ActiveEngineID string

	// DeepSeek API Key
	DeepseekAPIKey string

	// 语音识别激活引擎 + 模型（来自模型中心选择，空=自动）
	ActiveASREngine string
	ActiveASRModel  string

	// 语音合成激活引擎 + 模型（来自模型中心选择，空=自动）
	ActiveTTSEngine string
	ActiveTTSModel  string

	// 功能级模型绑定（各功能独立 LLM，空=用全局激活引擎+模型）
	FuncChatEngine    string
	FuncChatModel     string
	FuncNovelEngine   string
	FuncNovelModel    string
	FuncOfficeEngine  string
	FuncOfficeModel   string
	FuncGaeaEngine    string
	FuncGaeaModel     string
	// 功能级启停（默认启用；停用后该功能路由回退全局）
	FuncChatEnabled   bool
	FuncNovelEnabled  bool
	FuncOfficeEnabled bool
	FuncGaeaEnabled   bool
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
	}
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
		XaiClientID:        "b1a00492-073a-47ea-816f-4c329264a828",
		XaiAPIBaseURL:      "https://api.x.ai/v1",
		RedirectHost:       "127.0.0.1",
		RedirectPort:       "56121",
		OIDCDiscoveryURL:   "https://auth.x.ai/.well-known/openid-configuration",
		Model:              "grok-4.20",
		TokenStorePath:     tokenPath,
		// 本机单用户定位：小说目录固定为 C:\AI\xiaoshuo（记忆：novels-directory）
		NovelsDir:          `C:\AI\xiaoshuo`,
		HTTPTimeoutSeconds:  180,
		DefaultTemperature:  0.7,
		AnalysisTemperature: 0.15,   // 分析任务低温度以确保精确
		ReasoningEffort:     "high", // 分析任务默认开启深度推理
		QualityThreshold:    6,      // 章节质量低于 6 分自动重试
		QualityMaxRetries:   2,      // 最多重试 2 次
		// 功能级模型默认启用（未显式停用时，绑定立即生效）
		FuncChatEnabled:   true,
		FuncNovelEnabled:  true,
		FuncOfficeEnabled: true,
		FuncGaeaEnabled:   true,

		// TTS 默认值
		TTSBinaryPath: filepath.Join(home, "voxcpm-tts", "voxcpm_tts.exe"),
		TTSServerPath: "", // 默认不设置，优先用 TTSBinaryPath
		TTSModelPath:  filepath.Join(home, "voxcpm-tts", "models", "voxcpm1.5-q8_0-audiovae-f16.gguf"),
		TTSPort:       8765,
		TTSBackend:    "cpu",
		TTSSpeed:      1.0,

		// 图片生成默认值
		ImageBackend: "xai",
		ComfyUIURL:   "http://127.0.0.1:8188",
		ImageSaveDir: "", // 默认不存盘
		ImageModel:   "grok-imagine-image-quality",
		// 本机单用户定位：ComfyUI 启动位置直接写死（gaea 仅此电脑使用）
		ComfyUIPath:      `C:\AI\ComfyUI\ComfyUI`,
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
	}
	if data != nil {
		var cf configFile
		if json.Unmarshal(data, &cf) == nil {
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
			// 2.x 聊天/轻语合并：旧配置只写 func_whisper_* 时迁移到 func_chat；
			// chat 显式配置优先，不覆盖；func_whisper_enabled=false 同步为 chat 停用。
			if cfg.FuncChatEngine == "" && cf.FuncWhisperEngine != "" {
				cfg.FuncChatEngine = cf.FuncWhisperEngine
				cfg.FuncChatModel = cf.FuncWhisperModel
				if cf.FuncWhisperEnabled != nil {
					cfg.FuncChatEnabled = *cf.FuncWhisperEnabled
				}
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
	// 回退：当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		if dirExists(filepath.Join(cwd, "prompts")) {
			return cwd
		}
	}
	return "."
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
	return os.WriteFile(configPath, data, 0644)
}

// saveSetters 配置项 → setter 函数注册表
var saveSetters = map[string]func(cf *configFile, value string) error{
	KeyNovelsDir:          func(cf *configFile, v string) error { cf.NovelsDir = v; return nil },
	KeyXaiClientID:        func(cf *configFile, v string) error { cf.XaiClientID = v; return nil },
	KeyHTTPTimeoutSeconds: func(cf *configFile, v string) error { n, err := strconv.Atoi(v); if err != nil { return err }; cf.HTTPTimeoutSeconds = n; return nil },
	KeyDefaultTemperature: func(cf *configFile, v string) error { f, err := strconv.ParseFloat(v, 64); if err != nil { return err }; cf.DefaultTemperature = f; return nil },
	KeyAnalysisTemperature: func(cf *configFile, v string) error { f, err := strconv.ParseFloat(v, 64); if err != nil { return err }; cf.AnalysisTemperature = f; return nil },
	KeyReasoningEffort:    func(cf *configFile, v string) error { cf.ReasoningEffort = v; return nil },
	KeyQualityThreshold:   func(cf *configFile, v string) error { n, err := strconv.Atoi(v); if err != nil { return err }; cf.QualityThreshold = n; return nil },
	KeyQualityMaxRetries:  func(cf *configFile, v string) error { n, err := strconv.Atoi(v); if err != nil { return err }; cf.QualityMaxRetries = n; return nil },
	KeyTTSBinaryPath:      func(cf *configFile, v string) error { cf.TTSBinaryPath = v; return nil },
	KeyTTSModelPath:       func(cf *configFile, v string) error { cf.TTSModelPath = v; return nil },
	KeyImageBackend:       func(cf *configFile, v string) error { cf.ImageBackend = v; return nil },
	KeyComfyUIURL:         func(cf *configFile, v string) error { cf.ComfyUIURL = v; return nil },
	KeyImageSaveDir:       func(cf *configFile, v string) error { cf.ImageSaveDir = v; return nil },
	KeyImageModel:         func(cf *configFile, v string) error { cf.ImageModel = v; return nil },
	KeyComfyUIPath:        func(cf *configFile, v string) error { cf.ComfyUIPath = v; return nil },
	KeyComfyUIPythonPath:  func(cf *configFile, v string) error { cf.ComfyUIPythonPath = v; return nil },
	KeyTTSPort:           func(cf *configFile, v string) error { n, err := strconv.Atoi(v); if err != nil { return err }; cf.TTSPort = n; return nil },
	KeyTTSBackend:         func(cf *configFile, v string) error { cf.TTSBackend = v; return nil },
	KeyActiveEngineID:    func(cf *configFile, v string) error { cf.ActiveEngineID = v; return nil },
	KeyModel:             func(cf *configFile, v string) error { cf.Model = v; return nil },
	KeyDeepseekAPIKey:    func(cf *configFile, v string) error { cf.DeepseekAPIKey = v; return nil },
	KeyActiveASREngine:   func(cf *configFile, v string) error { cf.ActiveASREngine = v; return nil },
	KeyActiveASRModel:    func(cf *configFile, v string) error { cf.ActiveASRModel = v; return nil },
	KeyActiveTTSEngine:   func(cf *configFile, v string) error { cf.ActiveTTSEngine = v; return nil },
	KeyActiveTTSModel:    func(cf *configFile, v string) error { cf.ActiveTTSModel = v; return nil },
	KeyFuncChatEngine:    func(cf *configFile, v string) error { cf.FuncChatEngine = v; return nil },
	KeyFuncChatModel:     func(cf *configFile, v string) error { cf.FuncChatModel = v; return nil },
	KeyFuncNovelEngine:   func(cf *configFile, v string) error { cf.FuncNovelEngine = v; return nil },
	KeyFuncNovelModel:    func(cf *configFile, v string) error { cf.FuncNovelModel = v; return nil },
	KeyFuncOfficeEngine:  func(cf *configFile, v string) error { cf.FuncOfficeEngine = v; return nil },
	KeyFuncOfficeModel:   func(cf *configFile, v string) error { cf.FuncOfficeModel = v; return nil },
	KeyFuncGaeaEngine:    func(cf *configFile, v string) error { cf.FuncGaeaEngine = v; return nil },
	KeyFuncGaeaModel:     func(cf *configFile, v string) error { cf.FuncGaeaModel = v; return nil },
	KeyFuncChatEnabled:   func(cf *configFile, v string) error { b, err := parseBoolPtr(v); if err != nil { return err }; cf.FuncChatEnabled = b; return nil },
	KeyFuncNovelEnabled:   func(cf *configFile, v string) error { b, err := parseBoolPtr(v); if err != nil { return err }; cf.FuncNovelEnabled = b; return nil },
	KeyFuncOfficeEnabled:  func(cf *configFile, v string) error { b, err := parseBoolPtr(v); if err != nil { return err }; cf.FuncOfficeEnabled = b; return nil },
	KeyFuncGaeaEnabled:    func(cf *configFile, v string) error { b, err := parseBoolPtr(v); if err != nil { return err }; cf.FuncGaeaEnabled = b; return nil },
}

// parseBoolPtr 解析 "true"/"1"/"0" 等布尔值并返回指针（用于 *bool 配置项）。
func parseBoolPtr(v string) (*bool, error) {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
