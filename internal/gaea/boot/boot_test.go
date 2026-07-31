package boot_test

import (
	"context"
	"io"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// TestBuildSmoke 冒烟测试：注入 mock provider + 测试配置，Build 应返回可用的
// Controller（不依赖网络/外部模型），确保 boot 装配链（config→resolve→provider→agent→controller）不被破坏。
func TestBuildSmoke(t *testing.T) {
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
