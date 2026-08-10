package boot_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// chdirTemp 将测试工作目录切换到临时目录并自动恢复，避免 Build 在包源码目录
// 创建 .gaea/、AGENTS.md 等运行时副作用（go test 的 cwd 是包目录）。
func chdirTemp(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

// TestBuildSmoke 冒烟测试：注入 mock provider + 测试配置，Build 应返回可用的
// Controller（不依赖网络/外部模型），确保 boot 装配链（config→resolve→provider→agent→controller）不被破坏。
func TestBuildSmoke(t *testing.T) {
	chdirTemp(t)
	// 注册 mock provider（唯一名字避免与全局注册表冲突）
	const kind = "test-mock-boot"
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})

	// 注入测试配置（不写用户配置文件）
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []config.ProviderEntry{{
		Name:          "mock",
		Kind:          kind,
		Model:         "grok-3",
		ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	defer config.SetLoader(nil) // 恢复默认文件加载，避免污染其他测试

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:      "mock",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		SessionDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	if ctrl == nil {
		t.Fatal("Build 返回 nil Controller")
	}
	ctrl.Close()
}

// TestBuildUnknownModel 未知模型应快速失败（装配链的 resolve 步骤守卫）。
func TestBuildUnknownModel(t *testing.T) {
	chdirTemp(t)
	cfg := config.Default()
	cfg.DefaultModel = "no-such-model"
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	defer config.SetLoader(nil)

	_, err := boot.Build(context.Background(), boot.Options{
		Model: "no-such-model",
		Sink:  event.FuncSink(func(event.Event) {}),
	})
	if err == nil {
		t.Fatal("未知模型应返回错误")
	}
}

// TestBuildWorkspaceCommandsAndPins 验证办公引擎按工作区（而非进程目录）发现
// 命令：任务模板库落盘 .gaea/commands/*.md 后，/ 菜单与 Submit 即可解析；
// 固定资料文件存在且能随系统提示词组装（pins 包单测覆盖正文块，此处验证
// Build 不因清单存在而失败，并确认命令被发现）。
func TestBuildWorkspaceCommandsAndPins(t *testing.T) {
	chdirTemp(t)
	const kind = "test-mock-boot-cmds"
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})

	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []config.ProviderEntry{{Name: "mock", Kind: kind, Model: "grok-3", ContextWindow: 1_000_000}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	defer config.SetLoader(nil)

	ws := t.TempDir()
	// 任务模板命令文件（模拟 ensureTaskTemplateCommands 落盘结果）
	cmdDir := filepath.Join(ws, ".gaea", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmdBody := "---\ndescription: 结构化周报\n---\n\n帮我生成周报：按「本周进展 / 下周计划」撰写。"
	if err := os.WriteFile(filepath.Join(cmdDir, "weekly-report.md"), []byte(cmdBody), 0o644); err != nil {
		t.Fatal(err)
	}
	// 固定资料清单 + 文本文件
	pinDir := filepath.Join(ws, ".gaea")
	if err := os.WriteFile(filepath.Join(pinDir, "pinned.json"), []byte(`["docs/说明.md"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "docs", "说明.md"), []byte("这是固定资料正文。"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:      "mock",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		SessionDir: t.TempDir(),
		Cwd:        ws,
	})
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	defer ctrl.Close()

	found := false
	for _, c := range ctrl.Commands() {
		if c.Name == "weekly-report" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("工作区 .gaea/commands 下的模板命令未被发现")
	}
}

// TestBuildWorkspaceLanguageNotInjected 回归：办公文档工作区不得在系统提示词中
// 被标成 Go/代码工程（此前 Profile.Scan 无条件硬编码 Language="Go"，且扫描的是
// 进程目录而非工作区，导致通用办公 AI 表现得像编程 agent）。
func TestBuildWorkspaceLanguageNotInjected(t *testing.T) {
	chdirTemp(t)

	// 办公工作区：只有文档 + 少量辅助脚本，无任何工程清单
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "方案.md"), []byte("# 方案\n"), 0644)
	os.MkdirAll(filepath.Join(ws, "scripts"), 0755)
	os.WriteFile(filepath.Join(ws, "scripts", "check.py"), []byte("print(1)\n"), 0644)

	// 对照组：真实 Go 工程（go.mod）
	goDir := t.TempDir()
	os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module example.com/test\n"), 0644)

	buildAndRun := func(kind, cwd string) string {
		t.Helper()
		var mp *testutil.MockProvider
		provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
			mp = testutil.NewMock("mock",
				testutil.Turn{Text: "好的，收到。"},
				testutil.Turn{Text: "好的，收到。"},
				testutil.Turn{Text: "好的，收到。"},
				testutil.Turn{Text: "好的，收到。"},
				testutil.Turn{Text: "好的，收到。"},
			)
			return mp, nil
		})
		cfg := config.Default()
		cfg.DefaultModel = "mock"
		cfg.Providers = []config.ProviderEntry{{
			Name:          "mock",
			Kind:          kind,
			Model:         "grok-3",
			ContextWindow: 1_000_000,
		}}
		config.SetLoader(func() (*config.Config, error) { return cfg, nil })

		ctrl, err := boot.Build(context.Background(), boot.Options{
			Model:      "mock",
			RequireKey: false,
			Sink:       event.FuncSink(func(event.Event) {}),
			Stderr:     io.Discard,
			Cwd:        cwd,
			SessionDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("Build(%s) 失败: %v", cwd, err)
		}
		defer ctrl.Close()
		if err := ctrl.Run(context.Background(), "你好"); err != nil {
			t.Fatalf("Run(%s) 失败: %v", cwd, err)
		}
		if mp == nil || len(mp.Requests()) == 0 {
			t.Fatalf("(%s) 未捕获到模型请求", cwd)
		}
		return mp.Requests()[0].Messages[0].Content
	}
	defer config.SetLoader(nil)

	officeSys := buildAndRun("test-mock-boot-office", ws)
	if strings.Contains(officeSys, "l=Go") || strings.Contains(officeSys, "Language: Go") {
		t.Errorf("办公工作区系统提示词被标成 Go 工程:\n%s", officeSys)
	}
	// 工作区根目录（root=…）应来自 opts.Cwd，而不是进程目录
	if !strings.Contains(officeSys, "root="+filepath.Base(ws)) {
		t.Errorf("办公系统提示词缺少工作区根 root=%s:\n%s", filepath.Base(ws), officeSys)
	}

	goSys := buildAndRun("test-mock-boot-go", goDir)
	if !strings.Contains(goSys, "l=Go") && !strings.Contains(goSys, "Language: Go") {
		t.Errorf("Go 工程系统提示词未识别语言:\n%s", goSys)
	}
}
