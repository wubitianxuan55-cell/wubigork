package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

func newChatEnginePreflightTestApp(t *testing.T, boundModel string) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	c := &core{
		cfg:       &config.Config{Model: "grok-4.7"},
		engineMgr: modelengine.NewManager("", ""),
	}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "llama",
		Enabled:      true,
		DefaultModel: "qwen35",
		Models:       []modelengine.ModelInfo{{ID: "qwen35", Kind: "llm"}},
	}); err != nil {
		t.Fatalf("SaveEngine(llama): %v", err)
	}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "xai",
		Enabled:      true,
		DefaultModel: "grok-4.7",
		Models:       []modelengine.ModelInfo{{ID: "grok-4.7", Kind: "llm"}},
	}); err != nil {
		t.Fatalf("SaveEngine(xai): %v", err)
	}
	c.cfg.SetFeatureModel("chat", "llama", boundModel)
	c.cfg.SetFeatureModelEnabled("chat", true)
	return &App{core: c}
}

func TestResolveChatModelRouteStartsBoundEngineAndNormalizesLlamaModel(t *testing.T) {
	stale := `C:\AI\gaea-voice-v2\models\Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2.gguf`
	a := newChatEnginePreflightTestApp(t, stale)
	original := ensureChatBindingEngineFn
	t.Cleanup(func() { ensureChatBindingEngineFn = original })

	var ensuredEngine, ensuredModel string
	ensureChatBindingEngineFn = func(_ *App, engineID, model string) error {
		ensuredEngine, ensuredModel = engineID, model
		return nil
	}

	engine, model, source, err := a.resolveChatModelRoute()
	if err != nil {
		t.Fatalf("resolveChatModelRoute: %v", err)
	}
	if engine != "llama" || model != "qwen35" || source != "feature" {
		t.Fatalf("route = (%q,%q,%q), want (llama,qwen35,feature)", engine, model, source)
	}
	if ensuredEngine != "llama" || ensuredModel != "qwen35" {
		t.Fatalf("ensure call = (%q,%q), want (llama,qwen35)", ensuredEngine, ensuredModel)
	}
	if _, got := a.cfg.GetFeatureModel("chat"); got != "qwen35" {
		t.Fatalf("normalized binding = %q, want qwen35", got)
	}
}

func TestResolveChatModelRouteFailsInsteadOfGlobalFallbackWhenBoundEngineCannotStart(t *testing.T) {
	a := newChatEnginePreflightTestApp(t, "qwen35")
	original := ensureChatBindingEngineFn
	t.Cleanup(func() { ensureChatBindingEngineFn = original })
	ensureChatBindingEngineFn = func(_ *App, _, _ string) error {
		return errors.New("35B 300s 未就绪")
	}

	_, _, _, err := a.resolveChatModelRoute()
	if err == nil {
		t.Fatal("绑定引擎启动失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "聊天绑定的模型 llama/qwen35 未能启动") {
		t.Fatalf("error = %q, want bound-engine context", err)
	}
	if strings.Contains(err.Error(), "xai") {
		t.Fatalf("不得静默回退全局 xAI: %q", err)
	}
}

func TestChatPlainSystemPromptDeclaresGaeaIdentity(t *testing.T) {
	if !strings.Contains(chatPlainSystemPrompt, "Gaea 桌面 AI 助手") {
		t.Fatalf("chatPlainSystemPrompt 缺少 Gaea 身份: %q", chatPlainSystemPrompt)
	}
	if !strings.Contains(chatPlainSystemPrompt, "不要把自己说成 xAI") {
		t.Fatalf("chatPlainSystemPrompt 缺少厂商身份约束: %q", chatPlainSystemPrompt)
	}
}
