// config.go — internal/config 核心骨架（P4 纯结构拆分）。
// 本文件仅保留：Load 加载装配、资源目录解析（resolveResourceDir /
// ResolveResourceDirForTest / DataRoot / dirExists）与 funcMu。
// 域拆分：配置键常量 → config_keys.go；configFile / Config 类型 → config_types.go；
// 功能级模型绑定 → config_features.go；布尔偏好 → config_prefs.go；
// 实时语音 Realtime → config_realtime.go；Save / saveSetters → config_save.go。
// 纯结构拆分：逻辑 / 字段 / 函数签名 / 默认值零改动，仅文件重组。

package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// funcMu 保护功能级模型绑定字段（GetFeatureModel/SetFeatureModel 并发读写）
var funcMu sync.RWMutex

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
		FuncSinEnabled:     true, // 原罪（闲庭·图文故事创作）默认启用：绑定即生效
		// S2-4/D8：敏感域（成本/报价）AI 默认本地优先。
		SensitiveLocal: true,
		// 2026-08-28：办公板块功能级 AI 调用默认本地优先（数据不出本机 + 省 token）。
		OfficeLocal: true,
		// 读屏纵深（v4.8）：摘要默认开（只走本地 Herdsman，失败退回截断）；
		// 留档默认关（exports 会进工位检索面）。
		ReadScreenSummary:  true,
		ReadScreenKeepLast: false,
		// 意图 LLM 兜底（v4.8）：默认关（宁漏勿误姿态 + 语音回路延迟）；硬超时 2s。
		IntentsLLMFallback:  false,
		IntentsLLMTimeoutMS: 2000,
		// 全局离线模式（v4.8）：默认关（云端引擎可用）。
		OfflineMode: false,
		// 引擎故障转移（C 刀 v0）：默认关（关闭=现状逐字节）。
		EngineFailoverEnabled: false,
		// T5-3a/b：本地模型保活 + 启动自动预载默认开启。
		KeepWarmEnabled: true,
		AutoPreload:     true,
		// 晨报预载（v4.16 刀④）：默认开启（高频工作记忆预装配进上下文）。
		MorningPreload: true,
		// 项目本体注入（6.2）：默认开启（固化/项目/反馈决策带引用注入）。
		ProjectBrief: true,
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
			if cf.SinImageBackend != "" {
				cfg.SinImageBackend = cf.SinImageBackend
			}
			if cf.SinImageModel != "" {
				cfg.SinImageModel = cf.SinImageModel
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
			if cf.GLMAPIKey != "" {
				cfg.GLMAPIKey = cf.GLMAPIKey
			}
			if cf.OpenCodeGoAPIKey != "" {
				cfg.OpenCodeGoAPIKey = cf.OpenCodeGoAPIKey
			}
			if cf.OpenCodeZenAPIKey != "" {
				cfg.OpenCodeZenAPIKey = cf.OpenCodeZenAPIKey
			}
			if cf.ModelHubAPIKey != "" {
				cfg.ModelHubAPIKey = cf.ModelHubAPIKey
			}
			if len(cf.CustomEngineKeys) > 0 {
				cfg.CustomEngineKeys = cf.CustomEngineKeys
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
			if cf.FuncSinEngine != "" {
				cfg.FuncSinEngine = cf.FuncSinEngine
			}
			if cf.FuncSinModel != "" {
				cfg.FuncSinModel = cf.FuncSinModel
			}
			if cf.FuncSinEnabled != nil {
				cfg.FuncSinEnabled = *cf.FuncSinEnabled
			}
			if cf.SensitiveLocal != nil {
				cfg.SensitiveLocal = *cf.SensitiveLocal
			}
			if cf.OfficeLocal != nil {
				cfg.OfficeLocal = *cf.OfficeLocal
			}
			if cf.ReadScreenSummary != nil {
				cfg.ReadScreenSummary = *cf.ReadScreenSummary
			}
			if cf.ReadScreenKeepLast != nil {
				cfg.ReadScreenKeepLast = *cf.ReadScreenKeepLast
			}
			if cf.IntentsLLMFallback != nil {
				cfg.IntentsLLMFallback = *cf.IntentsLLMFallback
			}
			if cf.IntentsLLMTimeoutMS > 0 {
				cfg.IntentsLLMTimeoutMS = cf.IntentsLLMTimeoutMS
			}
			if cf.OfflineMode != nil {
				cfg.OfflineMode = *cf.OfflineMode
			}
			if cf.EngineFailoverEnabled != nil {
				cfg.EngineFailoverEnabled = *cf.EngineFailoverEnabled
			}
			if cf.KeepWarmEnabled != nil {
				cfg.KeepWarmEnabled = *cf.KeepWarmEnabled
			}
			if cf.AutoPreload != nil {
				cfg.AutoPreload = *cf.AutoPreload
			}
			if cf.ProjectBrief != nil {
				cfg.ProjectBrief = *cf.ProjectBrief
			}
			if cf.MorningPreload != nil {
				cfg.MorningPreload = *cf.MorningPreload
			}
			if cf.UsdCnyRate != 0 {
				cfg.UsdCnyRate = cf.UsdCnyRate
			}
			if cf.GLMCatalogPath != "" {
				cfg.GLMCatalogPath = cf.GLMCatalogPath
			}
			if cf.GLMCatalogURL != "" {
				cfg.GLMCatalogURL = cf.GLMCatalogURL
			}
			if cf.CosyVoiceDir != "" {
				cfg.CosyVoiceDir = cf.CosyVoiceDir
			}
			if cf.CosyVoicePort != 0 {
				cfg.CosyVoicePort = cf.CosyVoicePort
			}
			if cf.RealtimeProvider != "" {
				cfg.RealtimeProvider = cf.RealtimeProvider
			}
			if cf.RealtimeModel != "" {
				cfg.RealtimeModel = cf.RealtimeModel
			}
			if cf.RealtimeAPIKey != "" {
				cfg.RealtimeAPIKey = cf.RealtimeAPIKey
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
