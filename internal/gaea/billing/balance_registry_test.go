package billing

import (
	"context"
	"strings"
	"testing"
)

// TestBalanceRegistry_AllKinds "deepseek" kind 经注册表构建。
func TestBalanceRegistry_AllKinds(t *testing.T) {
	kinds := BalanceProviderKinds()
	if len(kinds) != 1 || kinds[0] != BalanceKindDeepSeek {
		t.Fatalf("BalanceProviderKinds = %v, want [deepseek]", kinds)
	}
	p, err := NewBalanceProvider(BalanceKindDeepSeek)
	if err != nil {
		t.Fatalf("NewBalanceProvider(deepseek): %v", err)
	}
	if _, ok := p.(deepseekProvider); !ok {
		t.Fatalf("kind=deepseek 应返回 deepseekProvider, got %T", p)
	}
}

// TestBalanceRegistry_ConfigRouting 同形配置 + 不同 kind 得到不同实现：
// 切换余额后端只改 kind，消费方（Provider 接口）零改动。
func TestBalanceRegistry_ConfigRouting(t *testing.T) {
	var consumer func(kind string) (Provider, error)
	consumer = func(kind string) (Provider, error) {
		return NewBalanceProvider(kind)
	}
	p, err := consumer(BalanceKindDeepSeek)
	if err != nil {
		t.Fatalf("consumer(deepseek): %v", err)
	}
	if _, ok := p.(deepseekProvider); !ok {
		t.Errorf("consumer(deepseek) 应返回 deepseekProvider, got %T", p)
	}
}

// TestBalanceRegistry_UnknownKindError 未知 kind fail-closed（附已注册列表）。
func TestBalanceRegistry_UnknownKindError(t *testing.T) {
	_, err := NewBalanceProvider("no-such-provider")
	if err == nil {
		t.Fatal("未知 kind 应报错")
	}
	if !strings.Contains(err.Error(), BalanceKindDeepSeek) {
		t.Errorf("错误应附已注册 kind 列表: %v", err)
	}
}

// TestBalanceRegistry_DuplicateKindPanics 互斥注册：重复即 panic。
func TestBalanceRegistry_DuplicateKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic")
		}
	}()
	RegisterBalanceProvider(BalanceKindDeepSeek, func() Provider { return nil })
}

// TestBalanceRegistry_EmptyKindPanics 空 kind 注册直接 panic。
func TestBalanceRegistry_EmptyKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("空 kind 应 panic")
		}
	}()
	RegisterBalanceProvider("", func() Provider { return nil })
}

// TestFetchByKind_EmptyURL 空 url 与 kind 无关地返回 (nil, nil)：未配置而非错误。
func TestFetchByKind_EmptyURL(t *testing.T) {
	b, err := FetchByKind(context.Background(), BalanceKindDeepSeek, "", "key")
	if err != nil || b != nil {
		t.Fatalf("FetchByKind(\"\") = (%v, %v), want (nil, nil)", b, err)
	}
	// 未知 kind + 空 url：仍是未配置（先判 url，不查注册表）。
	b2, err2 := FetchByKind(context.Background(), "no-such", "", "key")
	if err2 != nil || b2 != nil {
		t.Fatalf("FetchByKind(unknown, \"\") = (%v, %v), want (nil, nil)", b2, err2)
	}
}

// TestFetchByKind_UnknownKindFailsClosed 未知 kind + 非空 url：fail-closed 报错。
func TestFetchByKind_UnknownKindFailsClosed(t *testing.T) {
	_, err := FetchByKind(context.Background(), "no-such", "http://x/balance", "k")
	if err == nil {
		t.Fatal("未知 kind 应报错（fail-closed）")
	}
	if !strings.Contains(err.Error(), "no-such") {
		t.Errorf("错误应点名未知 kind: %v", err)
	}
}

// TestFetch_DefaultKindDeepSeek Fetch 默认走 kind=deepseek（历史行为不变）。
func TestFetch_DefaultKindDeepSeek(t *testing.T) {
	_, err := Fetch(context.Background(), "http://x/balance", "k")
	if err == nil {
		// 服务器不存在 → 连接错误（证明已进入 deepseek provider 的 Fetch）。
		t.Error("Fetch 应尝试连接（错误非空），got nil")
	}
}

// TestBalanceRegistry_CustomProvider 自定义 kind 注册后经 FetchByKind 可用：
// 「按 kind 注册」验收——其他 provider 形状不经代码改动接入。
func TestBalanceRegistry_CustomProvider(t *testing.T) {
	RegisterBalanceProvider("custom", func() Provider {
		return customProvider{}
	})
	b, err := FetchByKind(context.Background(), "custom", "http://example.invalid/balance", "k")
	if err != nil {
		t.Fatalf("FetchByKind(custom): %v", err)
	}
	if b == nil || !b.Available || len(b.Infos) != 1 || b.Infos[0].Currency != "USD" {
		t.Fatalf("custom provider 结果 = %+v", b)
	}
	if got := b.Display(); got != "$42.00" {
		t.Errorf("Display = %q, want $42.00", got)
	}
}

// customProvider 自定义余额形状：直接返回固定余额（不依赖网络）。
type customProvider struct{}

func (customProvider) Fetch(ctx context.Context, url, apiKey string) (*Balance, error) {
	_ = ctx
	return &Balance{Available: true, Infos: []Info{{Currency: "USD", TotalBalance: "42.00"}}}, nil
}
