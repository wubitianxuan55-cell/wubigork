package app

// 审计 AP2-05 键覆盖对照测试（god-func 大拆试水第一刀的现状钉）：
// SaveConfig 全部 63 个内存同步键逐键 set→断言内存值，另钉「盘上有、内存无」
// 键的 default 分支（Warn 文案 + 内存不同步）与非法值错误路径。表驱动化改造
// 后本文件一字不改全绿 = 逐字段等价证明。

import (
	"testing"

	"github.com/gaea/gaea/internal/config"
)

// TestSaveConfig_AllSyncedKeys 现状钉：SaveConfig 支持内存同步的 63 个键
// （42 字符串直赋 + 5 整数 + 4 浮点 + 10 布尔 + model（SetModelMem）+
// novels_dir（书架特判）），每键一例 set→断言内存字段精确值。各键用互异值，
// 键→字段接错（如 tts_port 写进 cosyvoice_port）即红。
func TestSaveConfig_AllSyncedKeys(t *testing.T) {
	a := shelfTestApp(t)

	cases := []struct {
		key, value string
		check      func(*config.Config) bool
	}{
		// ── 字符串直赋键（42）──
		{config.KeyXaiClientID, "client-xyz-1", func(c *config.Config) bool { return c.XaiClientID == "client-xyz-1" }},
		{config.KeyReasoningEffort, "low", func(c *config.Config) bool { return c.ReasoningEffort == "low" }},
		{config.KeyTTSBinaryPath, "C:/tts/bin.exe", func(c *config.Config) bool { return c.TTSBinaryPath == "C:/tts/bin.exe" }},
		{config.KeyTTSModelPath, "C:/tts/model.gguf", func(c *config.Config) bool { return c.TTSModelPath == "C:/tts/model.gguf" }},
		{config.KeyTTSBackend, "cuda", func(c *config.Config) bool { return c.TTSBackend == "cuda" }},
		{config.KeyImageBackend, "comfyui", func(c *config.Config) bool { return c.ImageBackend == "comfyui" }},
		{config.KeyComfyUIURL, "http://127.0.0.1:8188", func(c *config.Config) bool { return c.ComfyUIURL == "http://127.0.0.1:8188" }},
		{config.KeyImageSaveDir, "C:/imgs-out", func(c *config.Config) bool { return c.ImageSaveDir == "C:/imgs-out" }},
		{config.KeyImageModel, "flux", func(c *config.Config) bool { return c.ImageModel == "flux" }},
		{config.KeyPortraitBackend, "herdsman", func(c *config.Config) bool { return c.PortraitBackend == "herdsman" }},
		{config.KeyPortraitModel, "portrait-m", func(c *config.Config) bool { return c.PortraitModel == "portrait-m" }},
		{config.KeySinImageBackend, "ollama", func(c *config.Config) bool { return c.SinImageBackend == "ollama" }},
		{config.KeySinImageModel, "sin-img-m", func(c *config.Config) bool { return c.SinImageModel == "sin-img-m" }},
		{config.KeyComfyUIPath, "C:/comfyui", func(c *config.Config) bool { return c.ComfyUIPath == "C:/comfyui" }},
		{config.KeyComfyUIPythonPath, "C:/python/python.exe", func(c *config.Config) bool { return c.ComfyUIPythonPath == "C:/python/python.exe" }},
		{config.KeyActiveEngineID, "deepseek", func(c *config.Config) bool { return c.ActiveEngineID == "deepseek" }},
		{config.KeyDeepseekAPIKey, "sk-deepseek-1", func(c *config.Config) bool { return c.DeepseekAPIKey == "sk-deepseek-1" }},
		{config.KeyOpencodeGoAPIKey, "sk-ocgo-1", func(c *config.Config) bool { return c.OpenCodeGoAPIKey == "sk-ocgo-1" }},
		{config.KeyOpencodeZenAPIKey, "sk-oczen-1", func(c *config.Config) bool { return c.OpenCodeZenAPIKey == "sk-oczen-1" }},
		{config.KeyActiveASREngine, "herdsman", func(c *config.Config) bool { return c.ActiveASREngine == "herdsman" }},
		{config.KeyActiveASRModel, "asr-m", func(c *config.Config) bool { return c.ActiveASRModel == "asr-m" }},
		{config.KeyActiveTTSEngine, "cosyvoice", func(c *config.Config) bool { return c.ActiveTTSEngine == "cosyvoice" }},
		{config.KeyActiveTTSModel, "tts-m", func(c *config.Config) bool { return c.ActiveTTSModel == "tts-m" }},
		{config.KeyTTSVoice, "轻语", func(c *config.Config) bool { return c.TTSVoice == "轻语" }},
		{config.KeyActiveOCREngine, "paddle", func(c *config.Config) bool { return c.ActiveOCREngine == "paddle" }},
		{config.KeyActiveOCRModel, "ocr-m", func(c *config.Config) bool { return c.ActiveOCRModel == "ocr-m" }},
		{config.KeyVoicePersonality, "gaea", func(c *config.Config) bool { return c.VoicePersonality == "gaea" }},
		{config.KeyFuncChatVoiceEngine, "edge", func(c *config.Config) bool { return c.FuncChatVoiceEngine == "edge" }},
		{config.KeyFuncChatVoiceModel, "fcvm", func(c *config.Config) bool { return c.FuncChatVoiceModel == "fcvm" }},
		{config.KeyFuncChatEngine, "herdsman", func(c *config.Config) bool { return c.FuncChatEngine == "herdsman" }},
		{config.KeyFuncChatModel, "chat-m", func(c *config.Config) bool { return c.FuncChatModel == "chat-m" }},
		{config.KeyFuncNovelEngine, "deepseek", func(c *config.Config) bool { return c.FuncNovelEngine == "deepseek" }},
		{config.KeyFuncNovelModel, "novel-m", func(c *config.Config) bool { return c.FuncNovelModel == "novel-m" }},
		{config.KeyFuncOfficeEngine, "herdsman", func(c *config.Config) bool { return c.FuncOfficeEngine == "herdsman" }},
		{config.KeyFuncOfficeModel, "office-m", func(c *config.Config) bool { return c.FuncOfficeModel == "office-m" }},
		{config.KeyFuncGaeaEngine, "ollama", func(c *config.Config) bool { return c.FuncGaeaEngine == "ollama" }},
		{config.KeyFuncGaeaModel, "gaea-m", func(c *config.Config) bool { return c.FuncGaeaModel == "gaea-m" }},
		{config.KeyFuncCharLibEngine, "herdsman", func(c *config.Config) bool { return c.FuncCharLibEngine == "herdsman" }},
		{config.KeyFuncCharLibModel, "charlib-m", func(c *config.Config) bool { return c.FuncCharLibModel == "charlib-m" }},
		{config.KeyFuncRoutineEngine, "ollama", func(c *config.Config) bool { return c.FuncRoutineEngine == "ollama" }},
		{config.KeyFuncRoutineModel, "routine-m", func(c *config.Config) bool { return c.FuncRoutineModel == "routine-m" }},
		{config.KeyCosyVoiceDir, "C:/cosyvoice-x", func(c *config.Config) bool { return c.CosyVoiceDir == "C:/cosyvoice-x" }},
		// ── 整数键（5；含 0 值一例钉「合法值不得被跳过」）──
		{config.KeyHTTPTimeoutSeconds, "233", func(c *config.Config) bool { return c.HTTPTimeoutSeconds == 233 }},
		{config.KeyHTTPTimeoutSeconds, "-5", func(c *config.Config) bool { return c.HTTPTimeoutSeconds == -5 }},
		{config.KeyQualityThreshold, "7", func(c *config.Config) bool { return c.QualityThreshold == 7 }},
		{config.KeyQualityThreshold, "0", func(c *config.Config) bool { return c.QualityThreshold == 0 }},
		{config.KeyQualityMaxRetries, "4", func(c *config.Config) bool { return c.QualityMaxRetries == 4 }},
		{config.KeyTTSPort, "8971", func(c *config.Config) bool { return c.TTSPort == 8971 }},
		{config.KeyCosyVoicePort, "8023", func(c *config.Config) bool { return c.CosyVoicePort == 8023 }},
		// ── 浮点键（4）──
		{config.KeyDefaultTemperature, "0.63", func(c *config.Config) bool { return c.DefaultTemperature == 0.63 }},
		{config.KeyAnalysisTemperature, "0.17", func(c *config.Config) bool { return c.AnalysisTemperature == 0.17 }},
		{config.KeyTTSSpeed, "1.25", func(c *config.Config) bool { return c.TTSSpeed == 1.25 }},
		{config.KeyUsdCnyRate, "7.14", func(c *config.Config) bool { return c.UsdCnyRate == 7.14 }},
		// ── 布尔键（10，先全量预置 true 再逐键置 false）──
		{config.KeyFuncChatEnabled, "false", func(c *config.Config) bool { return !c.FuncChatEnabled }},
		{config.KeyFuncNovelEnabled, "false", func(c *config.Config) bool { return !c.FuncNovelEnabled }},
		{config.KeyFuncOfficeEnabled, "false", func(c *config.Config) bool { return !c.FuncOfficeEnabled }},
		{config.KeyFuncGaeaEnabled, "false", func(c *config.Config) bool { return !c.FuncGaeaEnabled }},
		{config.KeyFuncCharLibEnabled, "false", func(c *config.Config) bool { return !c.FuncCharLibEnabled }},
		{config.KeyFuncRoutineEnabled, "false", func(c *config.Config) bool { return !c.FuncRoutineEnabled }},
		{config.KeySensitiveLocal, "false", func(c *config.Config) bool { return !c.SensitiveLocal }},
		{config.KeyOfflineMode, "false", func(c *config.Config) bool { return !c.OfflineMode }},
		{config.KeyKeepWarm, "false", func(c *config.Config) bool { return !c.KeepWarmEnabled }}, // 注意：keep_warm_enabled → KeepWarmEnabled
		{config.KeyAutoPreload, "false", func(c *config.Config) bool { return !c.AutoPreload }},
		// ── model（SetModelMem 并发安全写）与 novels_dir（书架特判）──
		{config.KeyModel, "grok-9", func(c *config.Config) bool { return c.Model == "grok-9" }},
		{config.KeyNovelsDir, "C:/books-x", func(c *config.Config) bool { return c.NovelsDir == "C:/books-x" }},
	}

	// 布尔键全量预置 true：置 false 后逐键可证「确是本键生效」。
	a.cfg.FuncChatEnabled = true
	a.cfg.FuncNovelEnabled = true
	a.cfg.FuncOfficeEnabled = true
	a.cfg.FuncGaeaEnabled = true
	a.cfg.FuncCharLibEnabled = true
	a.cfg.FuncRoutineEnabled = true
	a.cfg.SensitiveLocal = true
	a.cfg.OfflineMode = true
	a.cfg.KeepWarmEnabled = true
	a.cfg.AutoPreload = true

	for _, tc := range cases {
		if err := a.SaveConfig(tc.key, tc.value); err != nil {
			t.Errorf("SaveConfig(%q,%q): %v", tc.key, tc.value, err)
			continue
		}
		if !tc.check(a.cfg) {
			t.Errorf("SaveConfig(%q,%q) 后内存未按现状同步", tc.key, tc.value)
		}
	}
}

// TestSaveConfig_DiskOnlyKeysWarnNoSync 盘上有 setter、内存 switch 无 case 的键
// （default 分支消费集）：写盘成功、记「需重启生效」Warn（文案逐字）、内存
// 字段一字不动。表驱动化后多同步/漏告警任一偏离即红。
func TestSaveConfig_DiskOnlyKeysWarnNoSync(t *testing.T) {
	a := shelfTestApp(t)
	// 预置非零值：任何「误加内存同步」都会被断言抓到。
	a.cfg.GLMAPIKey = "old-glm"
	a.cfg.ModelHubAPIKey = "old-hub"
	a.cfg.FuncSinEngine = "old-sin-e"
	a.cfg.FuncSinModel = "old-sin-m"
	a.cfg.FuncSinEnabled = true
	a.cfg.OfficeLocal = true
	a.cfg.ReadScreenSummary = true
	a.cfg.ReadScreenKeepLast = true
	a.cfg.IntentsLLMFallback = true
	a.cfg.IntentsLLMTimeoutMS = 1999
	a.cfg.EngineFailoverEnabled = true
	a.cfg.MorningPreload = true
	a.cfg.GLMCatalogPath = "old-path"
	a.cfg.GLMCatalogURL = "old-url"
	a.cfg.RealtimeProvider = "old"
	a.cfg.RealtimeModel = "old-rt-m"
	a.cfg.RealtimeAPIKey = "old-rt-k"

	cases := []struct {
		key, value string
		unchanged  func(*config.Config) bool
	}{
		{config.KeyGLMAPIKey, "sk-glm-2", func(c *config.Config) bool { return c.GLMAPIKey == "old-glm" }},
		{config.KeyModelHubAPIKey, "sk-hub-2", func(c *config.Config) bool { return c.ModelHubAPIKey == "old-hub" }},
		{config.KeyCustomEngineKeys, `{"e1":"k1"}`, func(c *config.Config) bool { return c.CustomEngineKeys == nil }},
		{config.KeyFuncSinEngine, "sin-e-2", func(c *config.Config) bool { return c.FuncSinEngine == "old-sin-e" }},
		{config.KeyFuncSinModel, "sin-m-2", func(c *config.Config) bool { return c.FuncSinModel == "old-sin-m" }},
		{config.KeyFuncSinEnabled, "false", func(c *config.Config) bool { return c.FuncSinEnabled }},
		{config.KeyOfficeLocal, "false", func(c *config.Config) bool { return c.OfficeLocal }},
		{config.KeyReadScreenSummary, "false", func(c *config.Config) bool { return c.ReadScreenSummary }},
		{config.KeyReadScreenKeepLast, "true", func(c *config.Config) bool { return c.ReadScreenKeepLast }},
		{config.KeyIntentsLLMFallback, "true", func(c *config.Config) bool { return c.IntentsLLMFallback }},
		{config.KeyIntentsLLMTimeoutMS, "3000", func(c *config.Config) bool { return c.IntentsLLMTimeoutMS == 1999 }},
		{config.KeyEngineFailover, "true", func(c *config.Config) bool { return c.EngineFailoverEnabled }},
		{config.KeyMorningPreload, "false", func(c *config.Config) bool { return c.MorningPreload }},
		{config.KeyGLMCatalogPath, "new-path", func(c *config.Config) bool { return c.GLMCatalogPath == "old-path" }},
		{config.KeyGLMCatalogURL, "http://cat.example", func(c *config.Config) bool { return c.GLMCatalogURL == "old-url" }},
		{config.KeyRealtimeProvider, "openai", func(c *config.Config) bool { return c.RealtimeProvider == "old" }},
		{config.KeyRealtimeModel, "rt-m-2", func(c *config.Config) bool { return c.RealtimeModel == "old-rt-m" }},
		{config.KeyRealtimeAPIKey, "rt-k-2", func(c *config.Config) bool { return c.RealtimeAPIKey == "old-rt-k" }},
	}
	for _, tc := range cases {
		records := captureLogs(t, func() {
			if err := a.SaveConfig(tc.key, tc.value); err != nil {
				t.Errorf("SaveConfig(%q,%q): %v", tc.key, tc.value, err)
				return
			}
		})
		if !logContainsMsg(records, "SaveConfig: 配置项已持久化，无内存同步项（重启后生效）") {
			t.Errorf("SaveConfig(%q,%q) 未记「无内存同步项」Warn, got %v", tc.key, tc.value, logMsgs(records))
		}
		if !tc.unchanged(a.cfg) {
			t.Errorf("SaveConfig(%q,%q) 不应同步内存字段", tc.key, tc.value)
		}
	}
}

// TestSaveConfig_InvalidValueErrorsBeforeMemSync 非法值：config.Save 先行校验
// 失败 → 错误透传、内存字段不动、且不产生任何「内存同步跳过」Warn（同步侧
// 根本不执行）。钉的是「盘校验先于内存同步」的顺序现状。
func TestSaveConfig_InvalidValueErrorsBeforeMemSync(t *testing.T) {
	a := shelfTestApp(t)
	a.cfg.HTTPTimeoutSeconds = 30
	a.cfg.TTSSpeed = 1.0
	a.cfg.CosyVoicePort = 8010
	a.cfg.FuncChatEnabled = true

	cases := []struct {
		key, value string
		unchanged  func(*config.Config) bool
	}{
		{config.KeyHTTPTimeoutSeconds, "abc", func(c *config.Config) bool { return c.HTTPTimeoutSeconds == 30 }},
		{config.KeyTTSSpeed, "9.9", func(c *config.Config) bool { return c.TTSSpeed == 1.0 }}, // 越界 0.25-4.0
		{config.KeyCosyVoicePort, "0", func(c *config.Config) bool { return c.CosyVoicePort == 8010 }},
		{config.KeyFuncChatEnabled, "maybe", func(c *config.Config) bool { return c.FuncChatEnabled }},
	}
	for _, tc := range cases {
		records := captureLogs(t, func() {
			if err := a.SaveConfig(tc.key, tc.value); err == nil {
				t.Errorf("SaveConfig(%q,%q) 应透传盘校验错误", tc.key, tc.value)
			}
		})
		for _, m := range logMsgs(records) {
			if containsAny(m, "内存同步跳过") {
				t.Errorf("SaveConfig(%q,%q) 盘校验失败后不应进入内存同步侧, got warn %q", tc.key, tc.value, m)
			}
		}
		if !tc.unchanged(a.cfg) {
			t.Errorf("SaveConfig(%q,%q) 失败后内存字段不得变动", tc.key, tc.value)
		}
	}
}
