// config_types.go — 类型定义域（P4 纯结构拆分）。
// configFile / Config 结构体原样保留（字段名 / JSON tag 逐字节未动）。

package config

// configFile 表示 ~/.gaea_config.json 的结构
type configFile struct {
	XaiClientID         string            `json:"xai_client_id"`
	NovelsDir           string            `json:"novels_dir"`
	HTTPTimeoutSeconds  int               `json:"http_timeout_seconds"`
	DefaultTemperature  float64           `json:"default_temperature"`
	AnalysisTemperature float64           `json:"analysis_temperature"`
	ReasoningEffort     string            `json:"reasoning_effort"`    // Grok 推理深度: "low" / "high"
	QualityThreshold    int               `json:"quality_threshold"`   // 章节质量阈值 1-10，低于此触发自动重试
	QualityMaxRetries   int               `json:"quality_max_retries"` // 最大自动重试次数
	TTSBinaryPath       string            `json:"tts_binary_path,omitempty"`
	TTSModelPath        string            `json:"tts_model_path,omitempty"`
	ImageBackend        string            `json:"image_backend,omitempty"` // "xai" (默认) | "comfyui" | "herdsman" | "ollama"
	ComfyUIURL          string            `json:"comfyui_url,omitempty"`
	ImageSaveDir        string            `json:"image_save_dir,omitempty"`         // 图片生成存放目录
	ImageModel          string            `json:"image_model,omitempty"`            // 图片模型
	PortraitBackend     string            `json:"portrait_backend,omitempty"`       // 角色库剧照后端（空=跟随绘梦）
	PortraitModel       string            `json:"portrait_model,omitempty"`         // 角色库剧照模型（空=跟随绘梦）
	SinImageBackend     string            `json:"sin_image_backend,omitempty"`      // 原罪插图生图后端（空=跟随全局生图设置）
	SinImageModel       string            `json:"sin_image_model,omitempty"`        // 原罪插图生图模型（空=跟随全局生图设置）
	ComfyUIPath         string            `json:"comfyui_path,omitempty"`           // ComfyUI 安装目录
	ComfyUIPythonPath   string            `json:"comfyui_python_path,omitempty"`    // Python 解释器路径
	TTSPort             int               `json:"tts_port,omitempty"`               // TTS 服务端口
	TTSBackend          string            `json:"tts_backend,omitempty"`            // TTS 后端: "cpu" | "cuda"
	TTSSpeed            float64           `json:"tts_speed,omitempty"`              // TTS 语速
	ActiveEngineID      string            `json:"active_engine_id,omitempty"`       // 活跃模型引擎 ID
	Model               string            `json:"model,omitempty"`                  // 默认 LLM 模型名
	DeepseekAPIKey      string            `json:"deepseek_api_key,omitempty"`       // DeepSeek API Key
	GLMAPIKey           string            `json:"glm_api_key,omitempty"`            // GLM (智谱) API Key
	OpenCodeGoAPIKey    string            `json:"opencode_go_api_key,omitempty"`    // OpenCode Go API Key
	OpenCodeZenAPIKey   string            `json:"opencode_zen_api_key,omitempty"`   // OpenCode Zen API Key
	ModelHubAPIKey      string            `json:"modelhub_api_key,omitempty"`       // Unsloth Model Hub API Key
	CustomEngineKeys    map[string]string `json:"custom_engine_keys,omitempty"`     // 自定义引擎 Key 库（engineID → 密文）
	ActiveASREngine     string            `json:"active_asr_engine,omitempty"`      // 语音识别激活引擎
	ActiveASRModel      string            `json:"active_asr_model,omitempty"`       // 语音识别激活模型
	ActiveTTSEngine     string            `json:"active_tts_engine,omitempty"`      // 语音合成激活引擎
	ActiveTTSModel      string            `json:"active_tts_model,omitempty"`       // 语音合成激活模型
	TTSVoice            string            `json:"tts_voice,omitempty"`              // 语音合成音色
	ActiveOCREngine     string            `json:"active_ocr_engine,omitempty"`      // OCR 激活引擎
	ActiveOCRModel      string            `json:"active_ocr_model,omitempty"`       // OCR 激活模型
	VoicePersonality    string            `json:"voice_personality,omitempty"`      // 语音对话角色
	FuncChatVoiceEngine string            `json:"func_chat_voice_engine,omitempty"` // 聊天语音合成引擎
	FuncChatVoiceModel  string            `json:"func_chat_voice_model,omitempty"`  // 聊天语音合成模型
	FuncChatEngine      string            `json:"func_chat_engine,omitempty"`
	FuncChatModel       string            `json:"func_chat_model,omitempty"`
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
	// 原罪板块（闲庭·图文故事创作）：独立 LLM 绑定 + 启停
	FuncSinEngine  string `json:"func_sin_engine,omitempty"`
	FuncSinModel   string `json:"func_sin_model,omitempty"`
	FuncSinEnabled *bool  `json:"func_sin_enabled,omitempty"`
	// 敏感域本地化开关（nil=默认开启，true=成本/报价 AI 走本地 Herdsman）
	SensitiveLocal *bool `json:"sensitive_local,omitempty"`
	// 办公本地优先开关（nil=默认开启，true=办公功能级 AI 调用走本地 Herdsman）
	OfficeLocal *bool `json:"office_local,omitempty"`
	// 读屏摘要开关（nil=默认开启，OCR 长文本本地摘要后朗读）
	ReadScreenSummary *bool `json:"read_screen_summary,omitempty"`
	// 读屏留档开关（nil=默认关闭，最近读屏截图覆盖存 exports）
	ReadScreenKeepLast *bool `json:"read_screen_keep_last,omitempty"`
	// 意图 LLM 兜底分类开关（nil=默认关闭）
	IntentsLLMFallback *bool `json:"intents_llm_fallback,omitempty"`
	// 意图 LLM 兜底硬超时毫秒（0=未配置回退默认 2000）
	IntentsLLMTimeoutMS int `json:"intents_llm_timeout_ms,omitempty"`
	// 全局离线模式开关（nil=默认关闭，路由只允许本地引擎）
	OfflineMode *bool `json:"offline_mode,omitempty"`
	// 引擎故障转移开关（C 刀 v0，nil=默认关闭）
	EngineFailoverEnabled *bool `json:"engine_failover_enabled,omitempty"`
	// 本地模型调度开关（T5-3a/b，nil=默认开启）
	KeepWarmEnabled *bool `json:"keep_warm_enabled,omitempty"` // 保活探针
	AutoPreload     *bool `json:"auto_preload,omitempty"`      // 启动自动预载
	// 晨报预载开关（v4.16 刀④，nil=默认开启）：只影响上下文注入，与晨报卡片无关。
	MorningPreload *bool `json:"morning_preload,omitempty"`
	// ProjectBrief 是项目本体注入开关（6.2，默认开）：work 空间新会话装配时
	// 把固化/项目/反馈决策带 [MEM:] 引用键注入缓存稳定前缀。
	ProjectBrief *bool `json:"project_brief,omitempty"`
	// 美元→人民币汇率（费用估算折算用；0=未配置，加载时回退默认 7.2）
	UsdCnyRate float64 `json:"usd_cny_rate,omitempty"`
	// GLM 目录覆盖文件路径（空=只用内嵌目录）
	GLMCatalogPath string `json:"glm_catalog_path,omitempty"`
	// GLM 目录远程热更新 URL（空=禁用）
	GLMCatalogURL string `json:"glm_catalog_url,omitempty"`
	// CosyVoice 本地 TTS 服务（T6-9.5）：路径/端口可配置，空值回退默认。
	CosyVoiceDir  string `json:"cosyvoice_dir,omitempty"`
	CosyVoicePort int    `json:"cosyvoice_port,omitempty"`
	// 实时语音 Realtime 档（S1）：provider/model + API Key（存储口径 =
	// secure.EncryptString 密文，config 层只存取字符串不做加解密）。
	RealtimeProvider string `json:"realtime_provider,omitempty"`
	RealtimeModel    string `json:"realtime_model,omitempty"`
	RealtimeAPIKey   string `json:"realtime_api_key,omitempty"`
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
	SinImageBackend   string // 原罪插图生图后端（空=跟随全局生图设置）
	SinImageModel     string // 原罪插图生图模型（空=跟随全局生图设置）
	ComfyUIPath       string // ComfyUI 安装目录（main.py 所在路径），空=需手动启动
	ComfyUIPythonPath string // Python 解释器路径（留空则自动查找）

	// 活跃模型引擎 ID（"xai" | "ollama" | "herdsman" | "deepseek"）
	ActiveEngineID string

	// DeepSeek API Key
	DeepseekAPIKey string

	// GLM (智谱) API Key（模型中心配置）
	GLMAPIKey string

	// OpenCode Go API Key（opencode.ai 订阅，模型中心配置）
	OpenCodeGoAPIKey string

	// OpenCode Zen API Key（按量付费，opencode.ai/auth 获取）
	OpenCodeZenAPIKey string

	// Unsloth Model Hub API Key（本地 Model Hub 引擎；Unsloth 设置 → API 创建）
	ModelHubAPIKey string

	// 自定义引擎 Key 库（A 刀自定义引擎）：engineID → secure.EncryptString
	// 密文。app 层启动时解密为明文后注入 modelengine.Manager（明文只存内存）。
	CustomEngineKeys map[string]string

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
	// 原罪板块（闲庭·图文故事创作）：故事文本生成的独立引擎/模型绑定。
	FuncSinEngine string
	FuncSinModel  string
	// 功能级启停（默认启用；停用后该功能路由回退全局）
	FuncChatEnabled    bool
	FuncNovelEnabled   bool
	FuncOfficeEnabled  bool
	FuncGaeaEnabled    bool
	FuncCharLibEnabled bool
	FuncRoutineEnabled bool
	FuncSinEnabled     bool

	// 敏感域本地化（S2-4/D8）：成本/报价类 AI 操作默认路由本地 Herdsman。
	// true=本地优先（默认）；false=按常规路由（可回云端）。
	SensitiveLocal bool
	OfficeLocal    bool
	// 读屏纵深（v4.8）：摘要/留档开关
	ReadScreenSummary  bool
	ReadScreenKeepLast bool
	// 意图 LLM 兜底（v4.8）：开关默认关；超时默认 2000ms
	IntentsLLMFallback  bool
	IntentsLLMTimeoutMS int
	// 全局离线模式（v4.8）：默认关；开启后路由只允许本地引擎
	OfflineMode bool

	// 引擎故障转移（C 刀 v0）：默认关；开启后聊天请求首请求失败（网络类错误或
	// HTTP 408/429/5xx）时换 enabled 且最近 Status.Connected 的 llm 引擎重试一次。
	EngineFailoverEnabled bool

	// 本地模型调度（T5-3a/b，默认开启）：
	//   KeepWarmEnabled：保活——周期性对已运行的本地模型发轻量探针，防止被
	//                     herdsman 空闲卸载/降温，保持「说用就能用」；
	//   AutoPreload：启动自动预载——按功能绑定（gaea→office→chat）预载一个
	//                 herdsman 模型，降低首次对话的冷启动等待。
	KeepWarmEnabled bool
	AutoPreload     bool

	// 晨报预载（v4.16 刀④）：work 空间会话装配时把高频工作记忆预装配进
	// agent 上下文（零 LLM、预算受限）。默认开启，只影响上下文注入。
	MorningPreload bool
	ProjectBrief   bool

	// 美元→人民币汇率（费用估算折算用，默认 7.2；模型中心可配置）
	UsdCnyRate float64

	// GLM 目录覆盖文件路径（模型中心成本层，空=只用内嵌目录）
	GLMCatalogPath string

	// GLM 目录远程热更新 URL（模型中心成本层，空=禁用；非空时启动异步
	// 拉取循环，v2 响应写缓存 glm_catalog_remote.json）
	GLMCatalogURL string

	// CosyVoice 本地 TTS 服务（T6-9.5）：路径/端口可配置，默认 C:\AI\cosyvoice / 8010。
	CosyVoiceDir  string
	CosyVoicePort int

	// 实时语音 Realtime 档（S1，internal/realtime seam）：三项来自落盘配置，
	// 默认空 = 未配置。RealtimeAPIKey 为 secure.EncryptString 密文（config 层
	// 只存取，不加减密）；app 层 initVoice 启动时经 secure.DecryptString 解出
	// 内存明文再注入语音运行时配置。
	RealtimeProvider string
	RealtimeModel    string
	RealtimeAPIKey   string
}
