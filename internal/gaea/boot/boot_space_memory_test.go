package boot_test

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// v4.5.1a 红线补课：系统提示词记忆索引按装配空间收窄——play 空间构建的控制器
// 画像/索引只含 play 记忆（work 事实不可见），兑现「记忆分区互不检索」注入侧。
// 走真实 MemoryUserDir（重定向 APPDATA/XDG_CONFIG_HOME）+ SQLite 后端，与桌面
// 端装配同源（boot.Build 读同一 userDir）。
func TestBuildSystemPromptMemorySpaceScoped(t *testing.T) {
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

	// 与 boot.Build 的 Cwd 同源（facts.project 唯一键的一部分）：桌面端真实
	// 路径是用户选定的工作空间，这里用同一临时目录做装配级对照。
	bootCwd := t.TempDir()
	seed := memory.Load(memory.Options{CWD: bootCwd, UserDir: userDir, DB: gdb})
	if _, err := seed.Store.Save(memory.Memory{
		Name: "work-fact", Space: "work", Type: memory.TypeProject,
		Kind: memory.KindSemantic, Description: "工位事实", Body: "预算口径",
	}); err != nil {
		t.Fatalf("save work-fact: %v", err)
	}
	if _, err := seed.Store.Save(memory.Memory{
		Name: "play-fact", Space: "play", Type: memory.TypeProject,
		Kind: memory.KindSemantic, Description: "乐园事实", Body: "游戏偏好",
	}); err != nil {
		t.Fatalf("save play-fact: %v", err)
	}

	build := func(space string) *control.Controller {
		t.Helper()
		registerSpaceMockKind()
		cfg := config.Default()
		cfg.DefaultModel = "mock"
		cfg.Tools.Enabled = nil
		cfg.Session.Space = space
		cfg.Providers = []config.ProviderEntry{{
			Name: "mock", Kind: "test-mock-space-assembly", Model: "m1", ContextWindow: 1_000_000,
		}}
		config.SetLoader(func() (*config.Config, error) { return cfg, nil })
		t.Cleanup(func() { config.SetLoader(nil) })
		ctrl, err := boot.Build(context.Background(), boot.Options{
			Model:      "mock",
			RequireKey: false,
			Sink:       event.FuncSink(func(event.Event) {}),
			Stderr:     io.Discard,
			Cwd:        bootCwd,
			SessionDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("Build(%s): %v", space, err)
		}
		t.Cleanup(func() { ctrl.Close() })
		return ctrl
	}

	// play 空间装配：控制器记忆只含 play（work 事实被隔离）
	play := build("play")
	got := play.Memory()
	if got == nil {
		t.Fatal("play 控制器 Memory() nil")
	}
	names := make([]string, 0, 2)
	for _, m := range got.Store.List() {
		names = append(names, m.Name)
	}
	if len(names) != 1 || names[0] != "play-fact" {
		t.Fatalf("play 系统提示词记忆索引 = %v, want [play-fact]", names)
	}
	// 逐轮注入同样收窄（RecallBlock 走同一 Store 视图）
	if block := got.RecallBlock("预算口径 工位", 0); block != "" {
		t.Fatalf("play 会话 RecallBlock 泄露 work 事实: %q", block)
	}

	// work 空间装配：只含 work
	work := build("work")
	names = names[:0]
	for _, m := range work.Memory().Store.List() {
		names = append(names, m.Name)
	}
	if len(names) != 1 || names[0] != "work-fact" {
		t.Fatalf("work 系统提示词记忆索引 = %v, want [work-fact]", names)
	}
}
