package boot

// mode_isolation_assembly_test.go — 记忆按运行模式隔离的装配断言（2026-11-25
// 办公工具梳理批）+ 注入去重断言（同一批记忆在系统提示词里只列一遍）。
// 走真实装配（Build + SQLite 后端，脑目录显式传临时目录，不碰用户库），mock
// provider 同 memory_brain_isolation_test 纪律。

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// modeTestBuild 以给定会话运行模式装配一个引擎（同一脑目录、同一工作区）。
func modeTestBuild(t *testing.T, brainDir, projDir, mode string) *control.Controller {
	t.Helper()
	kind := brainTestKind("test-mock-mode")
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: kind, Model: "grok-3", ContextWindow: 1_000_000,
	}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	cfg.Memory.Enabled = true
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := Build(context.Background(), Options{
		Model:         "mock",
		RequireKey:    false,
		Sink:          event.FuncSink(func(event.Event) {}),
		Stderr:        io.Discard,
		SessionDir:    filepath.Join(projDir, ".gaea", "sessions"),
		Cwd:           projDir,
		MemoryUserDir: brainDir,
		MemoryMode:    mode,
	})
	if err != nil {
		t.Fatalf("Build(mode=%q): %v", mode, err)
	}
	t.Cleanup(ctrl.Close)
	return ctrl
}

func modeTestTool(t *testing.T, ctrl *control.Controller, name string) (tool.Tool, bool) {
	t.Helper()
	for _, tl := range ctrl.Tools() {
		if tl.Name() == name {
			return tl, true
		}
	}
	return nil, false
}

// modeTestHome 建一个脑目录 + 工作区，返回未收窄的存储。
func modeTestHome(t *testing.T) (brain, proj string, base memory.Store) {
	t.Helper()
	home := t.TempDir()
	brain = filepath.Join(home, "brain")
	proj = filepath.Join(home, "office")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	gdb := db.GetDatabase(brain)
	t.Cleanup(func() { _ = db.CloseDatabase(brain) })
	return brain, proj, memory.SQLiteStoreFor(gdb, brain, proj)
}

func TestBuildMemoryModeIsolation(t *testing.T) {
	brain, proj, base := modeTestHome(t)
	for _, m := range []memory.Memory{
		{Name: "general-fact", Description: "通用条目", Body: "b"},
		{Name: "cost-fact", Description: "造价条目", Body: "b", Mode: "cost"},
		{Name: "proposal-fact", Description: "方案条目", Body: "b", Mode: "proposal"},
	} {
		if _, err := base.Save(m); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		mode     string
		has      []string
		lacks    []string
		modeTags int
	}{
		{"", []string{"general-fact", "cost-fact", "proposal-fact"}, nil, 0},
		{"standard", []string{"general-fact"}, []string{"cost-fact", "proposal-fact"}, 0},
		{"cost", []string{"general-fact", "cost-fact"}, []string{"proposal-fact"}, 1},
		{"proposal", []string{"general-fact", "proposal-fact"}, []string{"cost-fact"}, 1},
	}
	for _, c := range cases {
		ctrl := modeTestBuild(t, brain, proj, c.mode)
		idx := ctrl.Memory().Index
		for _, want := range c.has {
			if !strings.Contains(idx, want+".md)") {
				t.Errorf("mode=%q 索引缺 %s：%s", c.mode, want, idx)
			}
		}
		for _, ban := range c.lacks {
			if strings.Contains(idx, ban) {
				t.Errorf("mode=%q 索引不该含 %s：%s", c.mode, ban, idx)
			}
		}
		if got := strings.Count(idx, "〔"); got != c.modeTags {
			t.Errorf("mode=%q 索引模式标签数 = %d, want %d：%s", c.mode, got, c.modeTags, idx)
		}
		if c.mode == "cost" && !strings.Contains(idx, "cost-fact.md)〔造价〕") {
			t.Errorf("造价档索引应给造价条目打「造价」标签：%s", idx)
		}

		// 注入去重：同一批记忆在系统提示词里只列一遍（晨报块/项目事实单已停注）。
		prompt := ctrl.SystemPrompt()
		if strings.Contains(prompt, "【工作记忆晨报】") || strings.Contains(prompt, "## Project fact sheet") {
			t.Errorf("mode=%q 系统提示词不该再注入晨报块/项目事实单（与索引重复）", c.mode)
		}
		// 索引行形如 [cost-fact](cost-fact.md)：按链接形态计数，一行一次。
		if n := strings.Count(prompt, "(cost-fact.md)"); (c.mode == "" || c.mode == "cost") && n != 1 {
			t.Errorf("mode=%q 系统提示词里 cost-fact 应只出现一次，实际 %d", c.mode, n)
		}
	}
}

// TestBuildRememberStampsMode 造价档引擎里 remember 存的项目记忆默认归造价；
// 方案档装配读不到那条；退役的三个记忆工具不再单独注册。
func TestBuildRememberStampsMode(t *testing.T) {
	brain, proj, base := modeTestHome(t)
	ctrl := modeTestBuild(t, brain, proj, "cost")
	remember, ok := modeTestTool(t, ctrl, "remember")
	if !ok {
		t.Fatal("造价档装配缺 remember")
	}
	if _, err := remember.Execute(context.Background(), json.RawMessage(
		`{"name":"cost-rule","description":"取费规矩","body":"管理费 5%","type":"project","kind":"procedural"}`)); err != nil {
		t.Fatalf("remember: %v", err)
	}
	if m, ok := base.Get("cost-rule"); !ok || m.Mode != "cost" {
		t.Fatalf("造价档存的过程规矩应归造价，得到 ok=%v mode=%q", ok, m.Mode)
	}

	proposal := modeTestBuild(t, brain, proj, "proposal")
	if strings.Contains(proposal.Memory().Index, "cost-rule") {
		t.Error("方案档索引不该含造价档记忆")
	}
	for _, gone := range []string{"forget", "memory_get", "promote_session_facts"} {
		if _, ok := modeTestTool(t, proposal, gone); ok {
			t.Errorf("%s 已并入 remember / memory_search，不应再单独注册", gone)
		}
	}
	if _, ok := modeTestTool(t, proposal, "memory_search"); !ok {
		t.Error("装配缺 memory_search")
	}
}
