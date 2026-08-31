package boot_test

// v4.16 刀④：晨报预载块装配断言——work 空间 + 开关开 → 系统提示词含晨报预载
// 块（高频工作记忆预装配进 agent 上下文，与画像/记忆索引并列）；play/mode=off
// /开关关/空记忆 → 不注入（双空间红线 + 预算诚实）。走真实 MemoryUserDir
// （重定向 APPDATA/XDG_CONFIG_HOME）+ SQLite 后端，与桌面端装配同源
// （boot.Build 读同一 userDir）。

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
	"github.com/gaea/gaea/internal/gaea/memory"
)

// morningPreloadHeader 与实现对齐的块头（断言注入存在性）。
const morningPreloadHeader = "【工作记忆晨报】"

// TestMorningPreloadAssembly 装配注入断言：work 注入（含 work 记忆、不含 play
// 记忆）/ play 不注入 / mode=off 不注入 / 开关关不注入。
func TestMorningPreloadAssembly(t *testing.T) {
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	userDir := config.MemoryUserDir()
	if userDir == "" {
		t.Fatal("MemoryUserDir 空（配置目录重定向失败）")
	}
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(userDir) })

	// 与 boot.Build 的 Cwd 同源（facts.project 唯一键的一部分）。
	bootCwd := t.TempDir()
	seed := memory.Load(memory.Options{CWD: bootCwd, UserDir: userDir, DB: gdb})
	if _, err := seed.Store.Save(memory.Memory{
		Name: "work-fact", Space: "work", Type: memory.TypeProject,
		Kind: memory.KindSemantic, Description: "预算口径", Body: "高频工作记忆",
	}); err != nil {
		t.Fatalf("save work-fact: %v", err)
	}
	if _, err := seed.Store.Save(memory.Memory{
		Name: "play-fact", Space: "play", Type: memory.TypeProject,
		Kind: memory.KindSemantic, Description: "游戏偏好", Body: "乐园事实",
	}); err != nil {
		t.Fatalf("save play-fact: %v", err)
	}

	build := func(space string, morningPreload, spaceModeOff bool) string {
		t.Helper()
		registerSpaceMockKind()
		cfg := config.Default()
		cfg.DefaultModel = "mock"
		cfg.Tools.Enabled = nil
		cfg.Session.Space = space
		if spaceModeOff {
			// space.mode=off：EffectiveSessionSpace 返回 ""（平铺形态）
			cfg.Space.Mode = "off"
		}
		cfg.Providers = []config.ProviderEntry{{
			Name: "mock", Kind: "test-mock-space-assembly", Model: "m1", ContextWindow: 1_000_000,
		}}
		config.SetLoader(func() (*config.Config, error) { return cfg, nil })
		t.Cleanup(func() { config.SetLoader(nil) })
		ctrl, err := boot.Build(context.Background(), boot.Options{
			Model:          "mock",
			RequireKey:     false,
			Sink:           event.FuncSink(func(event.Event) {}),
			Stderr:         io.Discard,
			Cwd:            bootCwd,
			SessionDir:     t.TempDir(),
			MorningPreload: morningPreload,
		})
		if err != nil {
			t.Fatalf("Build(%s, preload=%v, modeOff=%v): %v", space, morningPreload, spaceModeOff, err)
		}
		t.Cleanup(func() { ctrl.Close() })
		return ctrl.SystemPrompt()
	}

	// work + 开关开：注入预载块，含 work 记忆、绝不含 play 记忆（双空间隔离）
	if got := build("work", true, false); !strings.Contains(got, morningPreloadHeader) {
		t.Fatalf("work + 开关开：系统提示词应含晨报预载块\n---\n%s", got)
	} else {
		if !strings.Contains(got, "work-fact") {
			t.Fatalf("work 预载块应含 work 记忆 work-fact: %s", got)
		}
		if strings.Contains(got, "play-fact") {
			t.Fatalf("work 预载块不应含 play 记忆（双空间红线）: %s", got)
		}
	}
	// play 空间：不注入（play 会话不读 work 记忆，装配点按空间取数天然隔离）
	if got := build("play", true, false); strings.Contains(got, morningPreloadHeader) {
		t.Fatalf("play 空间不应注入晨报预载块: %s", got)
	}
	// space.mode=off（Space=""）：平铺形态不注入（仅 work 空间装配）
	if got := build("work", true, true); strings.Contains(got, morningPreloadHeader) {
		t.Fatalf("mode=off 不应注入晨报预载块: %s", got)
	}
	// 开关关（morning_preload=false）：不注入，前缀维持原状
	if got := build("work", false, false); strings.Contains(got, morningPreloadHeader) {
		t.Fatalf("开关关不应注入晨报预载块: %s", got)
	}
}

// TestMorningPreloadAssembly_EmptyMemory 空记忆：work + 开关开也不注入
// （纯函数返回空串，前缀逐字节不变）。
func TestMorningPreloadAssembly_EmptyMemory(t *testing.T) {
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	userDir := config.MemoryUserDir()
	if userDir == "" {
		t.Fatal("MemoryUserDir 空（配置目录重定向失败）")
	}
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(userDir) })

	bootCwd := t.TempDir() // 不种任何事实：空记忆
	registerSpaceMockKind()
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Tools.Enabled = nil
	cfg.Session.Space = "work"
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: "test-mock-space-assembly", Model: "m1", ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })
	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:          "mock",
		RequireKey:     false,
		Sink:           event.FuncSink(func(event.Event) {}),
		Stderr:         io.Discard,
		Cwd:            bootCwd,
		SessionDir:     t.TempDir(),
		MorningPreload: true,
	})
	if err != nil {
		t.Fatalf("Build(empty, work, preload=true): %v", err)
	}
	t.Cleanup(func() { ctrl.Close() })
	if got := ctrl.SystemPrompt(); strings.Contains(got, morningPreloadHeader) {
		t.Fatalf("空记忆不应注入晨报预载块: %s", got)
	}
}
