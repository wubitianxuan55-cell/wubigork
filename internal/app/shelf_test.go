package app

// T7-2 可见性收口：SaveConfig 写盘后内存同步补齐测试（含整数/浮点/布尔与
// 未知键报错路径）。

import (
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

// shelfTestApp 构造带 cfg 与裸 writingState 的 App（getPM/closePM 可用但
// 无项目；config.Save 写入临时 USERPROFILE/HOME 下的 .gaea_config.json）。
func shelfTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	return &App{
		core:         &core{cfg: &config.Config{}},
		writingState: &writingState{mu: sync.RWMutex{}},
	}
}

// TestSaveConfig_SyncsStringNumericKeys 字符串/整数/浮点键写盘后同步内存。
func TestSaveConfig_SyncsStringNumericKeys(t *testing.T) {
	a := shelfTestApp(t)

	cases := []struct {
		key, value string
		check      func() bool
	}{
		{config.KeyModel, "grok-4.21", func() bool { return a.cfg.Model == "grok-4.21" }},
		{config.KeyDefaultTemperature, "0.5", func() bool { return a.cfg.DefaultTemperature == 0.5 }},
		{config.KeyAnalysisTemperature, "0.2", func() bool { return a.cfg.AnalysisTemperature == 0.2 }},
		{config.KeyHTTPTimeoutSeconds, "240", func() bool { return a.cfg.HTTPTimeoutSeconds == 240 }},
		{config.KeyQualityThreshold, "8", func() bool { return a.cfg.QualityThreshold == 8 }},
		{config.KeyQualityMaxRetries, "3", func() bool { return a.cfg.QualityMaxRetries == 3 }},
		{config.KeyTTSSpeed, "1.5", func() bool { return a.cfg.TTSSpeed == 1.5 }},
		{config.KeyTTSPort, "8888", func() bool { return a.cfg.TTSPort == 8888 }},
		{config.KeyUsdCnyRate, "7.5", func() bool { return a.cfg.UsdCnyRate == 7.5 }},
		{config.KeyCosyVoicePort, "8011", func() bool { return a.cfg.CosyVoicePort == 8011 }},
		{config.KeyReasoningEffort, "low", func() bool { return a.cfg.ReasoningEffort == "low" }},
		{config.KeyTTSBinaryPath, "C:/tts/run.exe", func() bool { return a.cfg.TTSBinaryPath == "C:/tts/run.exe" }},
	}
	for _, c := range cases {
		if err := a.SaveConfig(c.key, c.value); err != nil {
			t.Errorf("SaveConfig(%q,%q): %v", c.key, c.value, err)
			continue
		}
		if !c.check() {
			t.Errorf("SaveConfig(%q,%q) 后内存未同步", c.key, c.value)
		}
	}
}

// TestSaveConfig_SyncsBoolKeys 布尔开关键写盘后同步内存（含置 false）。
func TestSaveConfig_SyncsBoolKeys(t *testing.T) {
	a := shelfTestApp(t)

	for _, k := range []string{
		config.KeyKeepWarm, config.KeyAutoPreload, config.KeySensitiveLocal,
		config.KeyFuncChatEnabled, config.KeyFuncNovelEnabled, config.KeyFuncOfficeEnabled,
		config.KeyFuncGaeaEnabled, config.KeyFuncCharLibEnabled, config.KeyFuncRoutineEnabled,
	} {
		if err := a.SaveConfig(k, "false"); err != nil {
			t.Errorf("SaveConfig(%q,false): %v", k, err)
			continue
		}
	}
	if a.cfg.KeepWarmEnabled || a.cfg.AutoPreload || a.cfg.SensitiveLocal {
		t.Error("布尔开关应全部同步为 false")
	}
	if a.cfg.FuncChatEnabled || a.cfg.FuncNovelEnabled || a.cfg.FuncOfficeEnabled ||
		a.cfg.FuncGaeaEnabled || a.cfg.FuncCharLibEnabled || a.cfg.FuncRoutineEnabled {
		t.Error("功能级启停开关应全部同步为 false")
	}

	// 回写 true 再次确认。
	if err := a.SaveConfig(config.KeyKeepWarm, "true"); err != nil {
		t.Fatalf("SaveConfig keep_warm true: %v", err)
	}
	if !a.cfg.KeepWarmEnabled {
		t.Error("KeepWarmEnabled 应同步为 true")
	}
}

// TestSaveConfig_NovelsDirUpdatesMemory novels_dir 写盘后同步内存（旧目录
// 无打开项目时不做 closePM）。
func TestSaveConfig_NovelsDirUpdatesMemory(t *testing.T) {
	a := shelfTestApp(t)
	if err := a.SaveConfig(config.KeyNovelsDir, "C:/AI/books2"); err != nil {
		t.Fatalf("SaveConfig novels_dir: %v", err)
	}
	if a.cfg.NovelsDir != "C:/AI/books2" {
		t.Errorf("NovelsDir = %q, want C:/AI/books2", a.cfg.NovelsDir)
	}
}

// TestSaveConfig_UnknownKeyReturnsError 不支持的配置项必须把 config.Save 的
// 错误透传给调用方（不静默假装成功）。
func TestSaveConfig_UnknownKeyReturnsError(t *testing.T) {
	a := shelfTestApp(t)
	if err := a.SaveConfig("no_such_setting", "x"); err == nil {
		t.Fatal("未知配置键应返回错误")
	}
}
