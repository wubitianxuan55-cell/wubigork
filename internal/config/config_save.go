// config_save.go — 持久化写入域（P4 纯结构拆分）。
// Save / saveConfigFile / saveSetters / parseBoolPtr 原样保留。

package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

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
// 默认用 fileutil.RenameWithRetry：Windows 上 AV/索引器瞬时持有目标文件时
// 权限类 rename 失败按退避重试（v4.293），最终失败仍返回错误。
var renameFile = fileutil.RenameWithRetry

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
	KeyTTSBackend:        func(cf *configFile, v string) error { cf.TTSBackend = v; return nil },
	KeyActiveEngineID:    func(cf *configFile, v string) error { cf.ActiveEngineID = v; return nil },
	KeyModel:             func(cf *configFile, v string) error { cf.Model = v; return nil },
	KeyDeepseekAPIKey:    func(cf *configFile, v string) error { cf.DeepseekAPIKey = v; return nil },
	KeyGLMAPIKey:         func(cf *configFile, v string) error { cf.GLMAPIKey = v; return nil },
	KeyOpencodeGoAPIKey:  func(cf *configFile, v string) error { cf.OpenCodeGoAPIKey = v; return nil },
	KeyOpencodeZenAPIKey: func(cf *configFile, v string) error { cf.OpenCodeZenAPIKey = v; return nil },
	KeyModelHubAPIKey:    func(cf *configFile, v string) error { cf.ModelHubAPIKey = v; return nil },
	KeyCustomEngineKeys: func(cf *configFile, v string) error {
		// 值为 JSON map[string]string（engineID → secure 密文）；空串 = 清空。
		if v == "" {
			cf.CustomEngineKeys = nil
			return nil
		}
		m := map[string]string{}
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return fmt.Errorf("custom_engine_keys 需为 JSON 对象（engineID → 密文）: %w", err)
		}
		cf.CustomEngineKeys = m
		return nil
	},
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
	KeyFuncSinEngine:     func(cf *configFile, v string) error { cf.FuncSinEngine = v; return nil },
	KeyFuncSinModel:      func(cf *configFile, v string) error { cf.FuncSinModel = v; return nil },
	KeyFuncSinEnabled: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.FuncSinEnabled = b
		return nil
	},
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
	KeyOfficeLocal: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.OfficeLocal = b
		return nil
	},
	KeyReadScreenSummary: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.ReadScreenSummary = b
		return nil
	},
	KeyReadScreenKeepLast: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.ReadScreenKeepLast = b
		return nil
	},
	KeyIntentsLLMFallback: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.IntentsLLMFallback = b
		return nil
	},
	KeyIntentsLLMTimeoutMS: func(cf *configFile, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		if n < 200 || n > 60000 {
			return fmt.Errorf("意图兜底超时须在 200-60000 毫秒之间（当前值: %s）", v)
		}
		cf.IntentsLLMTimeoutMS = n
		return nil
	},
	KeyOfflineMode: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.OfflineMode = b
		return nil
	},
	KeyEngineFailover: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.EngineFailoverEnabled = b
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
	KeyMorningPreload: func(cf *configFile, v string) error {
		b, err := parseBoolPtr(v)
		if err != nil {
			return err
		}
		cf.MorningPreload = b
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
	KeyRealtimeProvider: func(cf *configFile, v string) error {
		// 空 = 关闭实时语音档（允许清空回未配置态）；非空只允许已注册 kind
		// "openai"（与 internal/realtime 注册表一致，非法值保存即拒绝）。
		if v != "" && v != "openai" {
			return fmt.Errorf("实时语音 provider 仅支持 openai（当前值: %s）", v)
		}
		cf.RealtimeProvider = v
		return nil
	},
	KeyRealtimeModel:  func(cf *configFile, v string) error { cf.RealtimeModel = v; return nil },
	KeyGLMCatalogPath: func(cf *configFile, v string) error { cf.GLMCatalogPath = v; return nil },
	KeyGLMCatalogURL:  func(cf *configFile, v string) error { cf.GLMCatalogURL = v; return nil },
	KeyRealtimeAPIKey: func(cf *configFile, v string) error { cf.RealtimeAPIKey = v; return nil },
}

// parseBoolPtr 解析 "true"/"1"/"0" 等布尔值并返回指针（用于 *bool 配置项）。
func parseBoolPtr(v string) (*bool, error) {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
