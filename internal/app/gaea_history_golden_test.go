package app

// 3.0 Step 1 兼容红线：GaeaHistory 输出必须逐字节不变。
// 这是唯一能直接调用真实 GaeaHistory 的位置（session/event/boot 包导入
// internal/app 会构成 import cycle），故作为新增测试文件放在 internal/app，
// 不修改任何既有文件、不新增 Wails 绑定。golden 文件缺失时自动生成基线；
// 改造完成后再次运行本测试必须逐字节通过。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// goldenFixtureSession 构造覆盖 GaeaHistory 全部分支的固定会话：
// system / user / assistant(含工具调用) / tool 结果 / 纯 assistant 文本。
// 输出必须逐字节稳定，任何一次改动都必须重录基线并评审。
func goldenFixtureSession() *agent.Session {
	s := agent.NewSession("golden system prompt")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "请帮我调试 auth 模块"})
	s.Add(provider.Message{
		Role:    provider.RoleAssistant,
		Content: "好的，我先读取相关文件。",
		ToolCalls: []provider.ToolCall{
			{ID: "call-1", Name: "read_file", Arguments: `{"path":"internal/auth.go"}`},
		},
	})
	s.Add(provider.Message{
		Role:       provider.RoleTool,
		Content:    "package auth\n\nfunc Login() {}",
		ToolCallID: "call-1",
		Name:       "read_file",
	})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "找到问题了：Login 缺少参数校验。"})
	return s
}

// TestGaeaHistoryGolden 用固定 fixture 断言 GaeaHistory 输出逐字节不变。
// 事件日志/投影/恢复改造不得改变该输出（前端恢复/回退/兜底全链依赖它）。
func TestGaeaHistoryGolden(t *testing.T) {
	// 与 TestGaeaBootBuild 一致：chdir 到临时目录避免会话/归档污染仓库，
	// 但不改 APPDATA（boot.Build 会打开全局 Hephaestus.db，临时 APPDATA 目录
	// 会被句柄锁住导致 cleanup 失败）。
	oldwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldwd) }()
	_ = os.Chdir(t.TempDir())

	// 用 mock provider 构建办公控制器（不依赖网络/真实模型）。
	const kind = "test-mock-gaea-history"
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})
	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name:          "mock",
		Kind:          kind,
		Model:         "grok-3",
		ContextWindow: 1_000_000,
	}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	defer gaeaConfig.SetLoader(nil)

	ctrl, err := gaeaBoot.Build(context.Background(), gaeaBoot.Options{
		Model:      "mock",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		SessionDir: gaeaConfig.WorkspaceSessionDir(""),
	})
	if err != nil {
		t.Fatalf("boot.Build: %v", err)
	}
	defer ctrl.Close()

	path := filepath.Join(gaeaConfig.WorkspaceSessionDir(""), "golden-session.jsonl")
	ctrl.Resume(goldenFixtureSession(), path)

	oldCtrl := ga.ctrl
	ga.mu.Lock()
	ga.ctrl = ctrl
	ga.mu.Unlock()
	defer func() {
		ga.mu.Lock()
		ga.ctrl = oldCtrl
		ga.mu.Unlock()
	}()

	a := &App{}
	got, err := json.Marshal(a.GaeaHistory())
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}

	_, thisFile, _, _ := runtime.Caller(0)
	goldenPath := filepath.Join(filepath.Dir(thisFile), "testdata", "gaea_history.golden.json")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden file generated: %s (%d bytes)", goldenPath, len(got))
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("GaeaHistory output drifted from golden:\n got: %s\nwant: %s", got, want)
	}
}
