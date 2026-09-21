package boot_test

// v4.378 刀：固化正文注入装配断言（activation 二维）——记忆开关开 → 系统提示
// 词含固化块（pinned 正文随会话快照装配）；两空间各见各自的固化集合（activation
// 与 scope 正交，读端经 InSpace 收窄）；开关关/无固化 → 不注入（前缀逐字节
// 不变）。走真实 MemoryUserDir + SQLite 后端，与桌面端装配同源。

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

const pinnedFactsHeader = "【固化记忆】"

func TestPinnedFactsAssembly(t *testing.T) {
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

	bootCwd := t.TempDir()
	seed := memory.Load(memory.Options{CWD: bootCwd, UserDir: userDir, DB: gdb})
	save := func(name, space, body string) {
		t.Helper()
		if _, err := seed.Store.Save(memory.Memory{
			Name: name, Space: space, Type: memory.TypeProject,
			Kind: memory.KindSemantic, Description: name + " 摘要", Body: body,
		}); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}
	save("plain-work", "work", "普通工作记忆正文")
	save("pinned-work", "work", "固化工作记忆正文")
	if err := seed.Store.Pin("pinned-work"); err != nil {
		t.Fatalf("pin pinned-work: %v", err)
	}
	save("pinned-play", "play", "固化乐园记忆正文")
	if err := seed.Store.Pin("pinned-play"); err != nil {
		t.Fatalf("pin pinned-play: %v", err)
	}

	build := func(space string, memoryOn bool) string {
		t.Helper()
		registerSpaceMockKind()
		cfg := config.Default()
		cfg.DefaultModel = "mock"
		cfg.Tools.Enabled = nil
		cfg.Session.Space = space
		cfg.Memory.Enabled = memoryOn
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
			t.Fatalf("Build(%s, memoryOn=%v): %v", space, memoryOn, err)
		}
		t.Cleanup(func() { ctrl.Close() })
		return ctrl.SystemPrompt()
	}

	// work：注入固化块，含 work 固化正文、绝不含 play 固化正文与未固化正文。
	got := build("work", true)
	if !strings.Contains(got, pinnedFactsHeader) {
		t.Fatalf("work 系统提示词应含固化块\n---\n%s", got)
	}
	if !strings.Contains(got, "固化工作记忆正文") {
		t.Fatalf("work 固化块应含 pinned-work 正文\n---\n%s", got)
	}
	if strings.Contains(got, "固化乐园记忆正文") {
		t.Fatalf("work 不应读到 play 固化正文（双空间红线）\n---\n%s", got)
	}
	if strings.Contains(got, "普通工作记忆正文") {
		t.Fatalf("未固化正文只应检索可达，不随快照\n---\n%s", got)
	}

	// play：各见各自的固化集合（activation 与 scope 正交）。
	got = build("play", true)
	if !strings.Contains(got, "固化乐园记忆正文") {
		t.Fatalf("play 固化块应含 pinned-play 正文\n---\n%s", got)
	}
	if strings.Contains(got, "固化工作记忆正文") {
		t.Fatalf("play 不应读到 work 固化正文\n---\n%s", got)
	}

	// 记忆开关关：不注入。
	if got := build("work", false); strings.Contains(got, pinnedFactsHeader) {
		t.Fatalf("记忆开关关不应注入固化块: %s", got)
	}
}
