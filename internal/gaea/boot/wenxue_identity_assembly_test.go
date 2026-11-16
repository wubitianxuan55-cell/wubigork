package boot_test

// 2026-11-16 图文身份分板装配断言：Options.SystemPrompt 非空 → L1 基底顶替
// 全局配置解析（板块引擎不吃办公人格）；空 → 现状逐字节（office default）。

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
)

func buildWithPromptOverride(t *testing.T, override string) string {
	t.Helper()
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	})
	// Build 会开脑 SQLite（Hephaestus.db 句柄不放，TempDir RemoveAll 会炸
	// Windows 文件锁）——与 morning_preload_assembly 同款收尾。
	userDir := config.MemoryUserDir()
	t.Cleanup(func() { db.CloseDatabase(userDir) })

	registerSpaceMockKind()
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Tools.Enabled = nil
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: "test-mock-space-assembly", Model: "m1", ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:        "mock",
		RequireKey:   false,
		Sink:         event.FuncSink(func(event.Event) {}),
		Stderr:       io.Discard,
		Cwd:          t.TempDir(),
		SessionDir:   t.TempDir(),
		SystemPrompt: override,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Cleanup(func() { ctrl.Close() })
	return ctrl.SystemPrompt()
}

// TestSystemPromptOverrideAssembly 覆写位装配断言：覆写非空顶替办公默认
// 人格；空保持现状（office DefaultSystemPrompt 仍生效）。
func TestSystemPromptOverrideAssembly(t *testing.T) {
	got := buildWithPromptOverride(t, "你是图文板块测试人格 MARKER-WX。")
	if !strings.Contains(got, "MARKER-WX") {
		t.Fatalf("覆写应作为 L1 基底装配\n---\n%s", got[:min(len(got), 400)])
	}
	if strings.Contains(got, "通用办公 AI 助手") {
		t.Fatal("覆写生效时不得再含办公默认人格")
	}
	// 装配块照常叠加：语言策略仍拼接（覆写只换身份段，不动底座装配）。
	if !strings.Contains(got, "所有思考过程和输出必须使用中文") {
		t.Fatal("语言策略装配块应照常叠加")
	}

	got = buildWithPromptOverride(t, "")
	if !strings.Contains(got, "通用办公 AI 助手") {
		t.Fatalf("空覆写应保持现状（office 默认人格）\n---\n%s", got[:min(len(got), 400)])
	}
}
