package control

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/billing"
)

// balanceKindTestProviderKind 是本文件注册的自定义余额后端 kind（3.0 Wave 4
// Step 3 收官：BalanceKind 贯通）。与 billing 包测试内的 "custom" 区分——
// 跨包测试需自注册新 kind，避免重名 panic。
const balanceKindTestProviderKind = "test-kind-ctrl"

// registerBalanceKindTestProvider 注册一个返回固定余额的自定义 provider
// （不依赖网络）。sync.Once 保证同一测试进程内只注册一次（-count 重跑安全）。
var registerBalanceKindTestProvider = sync.OnceFunc(func() {
	billing.RegisterBalanceProvider(balanceKindTestProviderKind, func() billing.Provider {
		return balanceKindTestProvider{}
	})
})

// balanceKindTestProvider 自定义余额形状：直接返回固定余额（参考 billing 包
// balance_registry_test.go 的 customProvider 模式）。
type balanceKindTestProvider struct{}

func (balanceKindTestProvider) Fetch(ctx context.Context, url, apiKey string) (*billing.Balance, error) {
	_ = ctx
	return &billing.Balance{Available: true, Infos: []billing.Info{{Currency: "USD", TotalBalance: "42.00"}}}, nil
}

// TestBalanceKind_CustomKindRoutes 自定义 kind 贯通 controller.Balance：
// Options.BalanceKind → billing 注册表 → 该形状的返回结果（Display "$42.00"）。
func TestBalanceKind_CustomKindRoutes(t *testing.T) {
	registerBalanceKindTestProvider()
	c := New(Options{
		BalanceURL:  "http://example.invalid/balance",
		BalanceKey:  "k",
		BalanceKind: balanceKindTestProviderKind,
	})
	b, err := c.Balance(context.Background())
	if err != nil {
		t.Fatalf("Balance(custom kind): %v", err)
	}
	if b == nil || !b.Available || len(b.Infos) != 1 || b.Infos[0].Currency != "USD" {
		t.Fatalf("自定义 kind 结果 = %+v", b)
	}
	if got := b.Display(); got != "$42.00" {
		t.Errorf("Display = %q, want $42.00", got)
	}
}

// TestBalanceKind_EmptyKindDefaultsToDeepSeek 空 kind = 历史默认 deepseek：
// 对必然连不上的地址发起真实 HTTP → 连接类错误，而不是 "unknown balance
// provider kind"（证明走了 deepseek provider，而非未查注册表）。
func TestBalanceKind_EmptyKindDefaultsToDeepSeek(t *testing.T) {
	c := New(Options{BalanceURL: "http://127.0.0.1:1/balance"})
	b, err := c.Balance(context.Background())
	if err == nil {
		t.Fatal("空 kind 应发起真实 HTTP（连接错误），got nil")
	}
	if b != nil {
		t.Errorf("错误时不应返回余额: %+v", b)
	}
	if strings.Contains(err.Error(), "unknown balance provider kind") {
		t.Errorf("不应报 unknown kind：%v", err)
	}
}

// TestBalanceKind_NoURLDoesNotConsultRegistry 无 balance_url：返回 (nil, nil)，
// 不查注册表——即使 kind 未注册也不报错。
func TestBalanceKind_NoURLDoesNotConsultRegistry(t *testing.T) {
	c := New(Options{BalanceKey: "k", BalanceKind: "no-such-kind"})
	b, err := c.Balance(context.Background())
	if err != nil || b != nil {
		t.Fatalf("Balance(no url) = (%v, %v), want (nil, nil)", b, err)
	}
}

// TestBalanceKind_UnknownKindFailsClosed 未知 kind + 非空 url：fail-closed，
// 返回明确错误（附已注册 kind 列表）。
func TestBalanceKind_UnknownKindFailsClosed(t *testing.T) {
	c := New(Options{BalanceURL: "http://example.invalid/balance", BalanceKind: "no-such-kind"})
	_, err := c.Balance(context.Background())
	if err == nil {
		t.Fatal("未知 kind 应报错（fail-closed）")
	}
	if !strings.Contains(err.Error(), "unknown balance provider kind") {
		t.Errorf("错误应点名 unknown kind: %v", err)
	}
}
