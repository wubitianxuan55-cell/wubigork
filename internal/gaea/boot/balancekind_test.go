package boot_test

import (
	"context"
	"io"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/billing"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// bootBalanceKind 是本文件注册的自定义余额后端 kind（3.0 Wave 4 Step 3
// 收官：ProviderEntry.balance_kind → boot → controller 贯通）。与 billing /
// control 包测试的 kind 名区分，避免重名 panic。init() 每个测试进程只跑一次。
const bootBalanceKind = "test-kind-boot"

func init() {
	billing.RegisterBalanceProvider(bootBalanceKind, func() billing.Provider {
		return bootBalanceProvider{}
	})
}

// bootBalanceProvider 自定义余额形状：直接返回固定余额（不依赖网络）。
type bootBalanceProvider struct{}

func (bootBalanceProvider) Fetch(ctx context.Context, url, apiKey string) (*billing.Balance, error) {
	_ = ctx
	return &billing.Balance{Available: true, Infos: []billing.Info{{Currency: "USD", TotalBalance: "42.00"}}}, nil
}

// TestBuildBalanceKindRouting 走完整 Build 装配链（config → resolve → boot →
// controller），验证 ProviderEntry.BalanceKind 贯通到 ctrl.Balance：结果按
// 自定义 kind 路由（Display "$42.00"），而非历史默认 deepseek。
func TestBuildBalanceKindRouting(t *testing.T) {
	chdirTemp(t)
	// 注册 mock LLM provider（唯一名字避免与全局注册表冲突）
	const llmKind = "test-mock-boot-balance"
	provider.Register(llmKind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})

	// 注入测试配置：provider 带 balance_url + balance_kind
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []config.ProviderEntry{{
		Name:          "mock",
		Kind:          llmKind,
		Model:         "grok-3",
		ContextWindow: 1_000_000,
		BalanceURL:    "http://example.invalid/balance",
		BalanceKind:   bootBalanceKind,
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
	defer ctrl.Close()

	b, err := ctrl.Balance(context.Background())
	if err != nil {
		t.Fatalf("ctrl.Balance: %v", err)
	}
	if b == nil || !b.Available || len(b.Infos) != 1 || b.Infos[0].Currency != "USD" {
		t.Fatalf("余额未按自定义 kind 路由: %+v", b)
	}
	if got := b.Display(); got != "$42.00" {
		t.Errorf("Display = %q, want $42.00", got)
	}
}
