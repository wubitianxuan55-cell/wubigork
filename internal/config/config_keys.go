// config_keys.go — 配置键常量域（P4 纯结构拆分：从 config.go 按域切出）。
// 只含 Key* 与 Default* 常量，无任何逻辑。

package config

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
	// 原罪（闲庭·图文故事创作板块）：独立 LLM 绑定，便于给故事创作单挂
	// 无审查/长上下文模型，与聊天/小说互不干扰。
	KeyFuncSinEngine = "func_sin_engine"
	KeyFuncSinModel  = "func_sin_model"
	// 功能级启停（FeatureModelBar 启停语义：只影响该功能的路由，不影响整个引擎）
	KeyFuncChatEnabled    = "func_chat_enabled"
	KeyFuncNovelEnabled   = "func_novel_enabled"
	KeyFuncOfficeEnabled  = "func_office_enabled"
	KeyFuncGaeaEnabled    = "func_gaea_enabled"
	KeyFuncCharLibEnabled = "func_characterlib_enabled"
	KeyFuncRoutineEnabled = "func_routine_enabled"
	KeyFuncSinEnabled     = "func_sin_enabled"
	// 敏感域本地化（S2-4/D8）：成本/报价类 AI 操作默认路由本地 Herdsman，
	// 可配置回云端。默认开启。
	KeySensitiveLocal = "sensitive_local"
	// 办公本地优先（2026-08-28）：办公板块功能级 AI 调用（文档/表格编辑、
	// 资料摘要、知识导入、记忆整理）默认路由本地 Herdsman，数据不出本机、
	// 省 token；Herdsman 停用/不可用时回退常规路由。默认开启。
	KeyOfficeLocal = "office_local"
	// 读屏摘要（读屏纵深 v4.8）：「读一下屏幕」OCR 文本超过 300 字时用本地
	// 模型压缩成 ≤200 字口语化摘要再朗读；只走本地 Herdsman（路由回退云端
	// 时放弃摘要退回截断，屏幕内容不出本机）。默认开启。
	KeyReadScreenSummary = "read_screen_summary"
	// 读屏留档（读屏纵深 v4.8）：把最近一次读屏截图以「屏幕-最近.png」滚动
	// 覆盖存进 .gaea/exports（会进工位检索面）。默认关闭。
	KeyReadScreenKeepLast = "read_screen_keep_last"
	// 意图 LLM 兜底分类（v4.8）：规则引擎未命中时用轻量 LLM 分类（白名单
	// navigate/status/read_screen + 0.75 置信门）。默认关闭——本开关把
	// 「宁可漏判」姿态交给模型且给语音回路加延迟。
	KeyIntentsLLMFallback = "intents_llm_fallback"
	// 意图 LLM 兜底硬超时（毫秒，默认 2000）：超时立即走聊天管道不重试。
	KeyIntentsLLMTimeoutMS = "intents_llm_timeout_ms"
	// 全局离线模式（v4.8）：开启后所有 AI 路由只允许本地引擎
	// （ollama/herdsman/cosyvoice/modelhub），云端引擎（xai/deepseek/opencode-*）
	// 一律跳过——数据不出本机的总闸。默认关闭。
	KeyOfflineMode = "offline_mode"
	// 引擎故障转移（C 刀 v0）：开启后聊天请求链（流式/非流式）首请求失败且为
	// 网络类错误或 HTTP 408/429/5xx 时，换一个 enabled 且最近 Status.Connected
	// 的 llm 引擎用其 default_model 重试一次（401/403/400/404 等配置类错误
	// 不转移）。默认关闭——关闭即现状，成功路径行为零改动。
	KeyEngineFailover = "engine_failover_enabled"
	// 本地模型调度（T5-3a/b）：保活 + 启动自动预载，默认开启。
	KeyKeepWarm    = "keep_warm_enabled" // 保活：周期性探活已运行的本地模型，防卸载/降温
	KeyAutoPreload = "auto_preload"      // 启动自动预载：按功能绑定预载 herdsman 模型
	// 晨报预载（v4.16 刀④）：work 空间会话装配时把高频工作记忆确定性聚合为
	// 「晨报预载」块预装配进 agent 上下文（零 LLM、预算受限、work 只读）。
	// 只影响上下文注入，与首页晨报卡片（GaeaMemoryMorningBrief）无关。默认开启。
	KeyMorningPreload    = "morning_preload"
	KeyProjectBrief      = "project_brief"
	KeyDeepseekAPIKey    = "deepseek_api_key"
	KeyGLMAPIKey         = "glm_api_key"
	KeyOpencodeGoAPIKey  = "opencode_go_api_key"
	KeyOpencodeZenAPIKey = "opencode_zen_api_key"
	// Unsloth Model Hub 本地引擎 Key（模型中心配置；sk-unsloth- 开头）。
	KeyModelHubAPIKey = "modelhub_api_key"
	// 自定义引擎 Key 库（A 刀自定义引擎）：值为 JSON map[string]string
	// （engineID → secure.EncryptString 密文）。config 层只存取字符串/JSON、
	// 不做加解密（明文↔密文由 app 层 secure 负责，先例 realtime_api_key）。
	KeyCustomEngineKeys = "custom_engine_keys"
	// 美元→人民币汇率（费用估算折算用，默认 7.2，可在模型中心配置）
	KeyUsdCnyRate = "usd_cny_rate"
	// GLM 目录覆盖文件路径（模型中心成本层）：非空时 GLM 静态目录在内嵌
	// 目录基础上按该文件热更新合并（同 ID 替换 + 新 ID 追加）。默认空=只用内嵌。
	KeyGLMCatalogPath = "glm_catalog_path"
	// GLM 目录远程热更新 URL（B 刀，模型中心成本层）：非空时 app 启动异步
	// 拉取 + 每 24h 周期，v2 响应写缓存 glm_catalog_remote.json（与
	// engines.json 同目录），生效优先级 覆盖文件 > 远程缓存 > 内嵌。
	// 远程目录仅影响展示与费用估算，不影响请求路由/alias 判定/鉴权。
	// 默认空=禁用。
	KeyGLMCatalogURL = "glm_catalog_url"
	// CosyVoice 本地 TTS 服务（T6-9.5）：路径/端口可配置，默认与历史硬编码一致。
	KeyCosyVoiceDir  = "cosyvoice_dir"
	KeyCosyVoicePort = "cosyvoice_port"
	// 实时语音 Realtime 档（S1，internal/realtime seam）：provider/model/key
	// 三项落盘。provider 空字符串 = 未配置（实时语音档关闭，走现拼接管线）；
	// 非空只允许 "openai"。realtime_api_key 存储口径 = secure.EncryptString
	// 的密文——config 层不做加解密、只存取字符串（明文↔密文转换由 app 层
	// secure 负责，先例 model_engine_handler.go SetOpencodeZenKey）。
	KeyRealtimeProvider = "realtime_provider"
	KeyRealtimeModel    = "realtime_model"
	KeyRealtimeAPIKey   = "realtime_api_key"
)

// DefaultUsdCnyRate 美元→人民币汇率默认值（费用估算折算口径）。
const DefaultUsdCnyRate = 7.2

// CosyVoice 本地服务默认值（T6-9.5：路径/端口可配置，未配置时与历史硬编码一致）。
const (
	DefaultCosyVoiceDir  = `C:\AI\cosyvoice`
	DefaultCosyVoicePort = 8010
)
