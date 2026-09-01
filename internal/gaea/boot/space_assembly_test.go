package boot_test

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// spaceTestTool 桌面端 ExtraTools 形态的最小工具桩。
type spaceTestTool struct {
	name string
	tag  string // 非空 = 实现 SpaceTaggedTool 自声明
}

func (t spaceTestTool) Name() string        { return t.name }
func (t spaceTestTool) ReadOnly() bool      { return true }
func (t spaceTestTool) Description() string { return "test stub" }
func (t spaceTestTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}
func (t spaceTestTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "", nil
}
func (t spaceTestTool) SpaceTag() string { return t.tag }

func toolNames(ctrl *control.Controller) map[string]bool {
	out := map[string]bool{}
	for _, t := range ctrl.Tools() {
		out[t.Name()] = true
	}
	return out
}

// registerSpaceMockKind 注册本文件的 mock LLM kind（provider.Register 重复注册
// 会 panic，用 sync.Once 保证同一测试进程只注册一次）。
var spaceMockOnce sync.Once

func registerSpaceMockKind() {
	spaceMockOnce.Do(func() {
		provider.Register("test-mock-space-assembly", func(cfg provider.Config) (provider.Provider, error) {
			return testutil.NewMock("mock"), nil
		})
	})
}

func buildWithSpaceConfig(t *testing.T, mutate func(*config.Config), extra []tool.Tool) *control.Controller {
	t.Helper()
	registerSpaceMockKind()
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	// 桌面端形态：Enabled=nil 全量注册内置工具（gaea_handler.go 同款），空间
	// 过滤完全由装配期决定。
	cfg.Tools.Enabled = nil
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: "test-mock-space-assembly", Model: "m1", ContextWindow: 1_000_000,
	}}
	if mutate != nil {
		mutate(cfg)
	}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:      "mock",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		SessionDir: t.TempDir(),
		ExtraTools: extra,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Cleanup(func() { ctrl.Close() })
	return ctrl
}

// TestBuildSpaceToolFiltering 装配期按空间物理过滤（S1.3-B）：work 不含
// image_gen、play 不含 edit 系/bash、shared 两空间都有、mode=off 全量回退。
func TestBuildSpaceToolFiltering(t *testing.T) {
	extra := []tool.Tool{
		spaceTestTool{name: "image_gen"},                       // play（分类表）
		spaceTestTool{name: "office_extra", tag: "work"},        // work（自声明）
		spaceTestTool{name: "generic_extra"},                    // 未归类 → shared
	}

	// work（session.space 缺省 → work）：办公工具齐全，无生图。
	work := buildWithSpaceConfig(t, nil, extra)
	names := toolNames(work)
	for _, want := range []string{"edit_file", "bash", "write_file", "grep", "office_extra", "generic_extra", "memory_search", "todo_write", "ask"} {
		if !names[want] {
			t.Errorf("work 缺少工具 %q", want)
		}
	}
	if names["image_gen"] {
		t.Error("work 不应包含 image_gen（play 域工具）")
	}

	// play：edit 系/bash 全部缺席，生图与 shared/meta 工具保留。
	play := buildWithSpaceConfig(t, func(c *config.Config) { c.Session.Space = "play" }, extra)
	names = toolNames(play)
	for _, absent := range []string{"edit_file", "multi_edit", "edit_lines", "move_file", "bash", "write_file", "grep", "ls", "web_fetch", "office_extra", "cost_save"} {
		if names[absent] {
			t.Errorf("play 不应包含 %q（work 域工具）", absent)
		}
	}
	for _, want := range []string{"image_gen", "generic_extra", "memory_search", "todo_write", "complete_step", "read_skill", "ask", "task"} {
		if !names[want] {
			t.Errorf("play 缺少工具 %q", want)
		}
	}

	// mode=off：整体回退现状——全量注册，工作空间分区工具一个不少。
	off := buildWithSpaceConfig(t, func(c *config.Config) {
		c.Space.Mode = "off"
		c.Session.Space = "play" // mode=off 时忽略
	}, extra)
	names = toolNames(off)
	for _, want := range []string{"edit_file", "bash", "image_gen", "office_extra", "generic_extra"} {
		if !names[want] {
			t.Errorf("mode=off 应全量注册 %q（现状回退）", want)
		}
	}
}

// TestBuildSpaceProfileModel S1.3-A：控制器按空间取 profile，gaea 键经
// ResolveModel 解析为执行器 entry（Label 随 entry.Name），缺省/不可解析回退现状。
func TestBuildSpaceProfileModel(t *testing.T) {
	chdirTemp(t)
	kindA := testKind("test-mock-space-model-a")
	kindB := testKind("test-mock-space-model-b")
	provider.Register(kindA, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("a"), nil
	})
	provider.Register(kindB, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("b"), nil
	})

	newCfg := func() *config.Config {
		cfg := config.Default()
		cfg.Providers = []config.ProviderEntry{
			{Name: "mock-a", Kind: kindA, Model: "ma", ContextWindow: 1_000_000},
			{Name: "mock-b", Kind: kindB, Model: "mb", ContextWindow: 1_000_000},
		}
		return cfg
	}

	build := func(t *testing.T, cfg *config.Config) *control.Controller {
		t.Helper()
		config.SetLoader(func() (*config.Config, error) { return cfg, nil })
		t.Cleanup(func() { config.SetLoader(nil) })
		ctrl, err := boot.Build(context.Background(), boot.Options{
			Model: "mock-a", RequireKey: false,
			Sink: event.FuncSink(func(event.Event) {}), Stderr: io.Discard,
			SessionDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		t.Cleanup(func() { ctrl.Close() })
		return ctrl
	}

	// play 空间 + profile.gaea=mock-b → 执行器切换到 mock-b。
	cfg := newCfg()
	cfg.Session.Space = "play"
	cfg.SpaceProfiles = map[string]config.SpaceProfile{"play": {Gaea: "mock-b"}}
	if got := build(t, cfg).Label(); got != "mock-b" {
		t.Fatalf("play profile 应切换执行器到 mock-b, Label = %q", got)
	}

	// 无 profile → 现状（mock-a）。
	cfg = newCfg()
	cfg.Session.Space = "play"
	if got := build(t, cfg).Label(); got != "mock-a" {
		t.Fatalf("无 profile 应回退现状模型, Label = %q", got)
	}

	// profile 引用不可解析 → 告警回退现状（不阻断装配）。
	cfg = newCfg()
	cfg.Session.Space = "play"
	cfg.SpaceProfiles = map[string]config.SpaceProfile{"play": {Gaea: "no-such-provider"}}
	if got := build(t, cfg).Label(); got != "mock-a" {
		t.Fatalf("不可解析引用应回退现状, Label = %q", got)
	}

	// mode=off → 忽略 profile（现状）。
	cfg = newCfg()
	cfg.Space.Mode = "off"
	cfg.Session.Space = "play"
	cfg.SpaceProfiles = map[string]config.SpaceProfile{"play": {Gaea: "mock-b"}}
	if got := build(t, cfg).Label(); got != "mock-a" {
		t.Fatalf("mode=off 应忽略 profile, Label = %q", got)
	}
}
