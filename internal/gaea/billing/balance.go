// Package billing queries a provider's wallet balance for the status line. The
// documented shape today is DeepSeek's GET /user/balance — registered as kind
// "deepseek"; other providers register their own shape (3.0 Step 3d #8：
// 余额查询按 kind 注册，不再只认 DeepSeek 形状). Balance is strictly optional:
// a provider with no balance_url is never queried — callers pass "" and get
// (nil, nil) back, and surfaces simply omit the readout. Kept tiny and
// dependency-free (net/http + encoding/json) so every frontend can share one fetch.
package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Balance is a wallet balance normalized for display.
type Balance struct {
	Available bool   // the provider reports the account can still serve API calls
	Infos     []Info // one entry per currency the provider returns
}

// Info is one currency's balance (DeepSeek returns one per currency).
type Info struct {
	Currency        string // "CNY" | "USD"
	TotalBalance    string // total available (granted + topped-up)
	GrantedBalance  string // unexpired promotional credit
	ToppedUpBalance string // paid-in credit
}

// deepseekResp mirrors the GET /user/balance response shape.
type deepseekResp struct {
	IsAvailable  bool `json:"is_available"`
	BalanceInfos []struct {
		Currency        string `json:"currency"`
		TotalBalance    string `json:"total_balance"`
		GrantedBalance  string `json:"granted_balance"`
		ToppedUpBalance string `json:"topped_up_balance"`
	} `json:"balance_infos"`
}

// httpClient bounds the balance query so a slow endpoint can't hang the status
// line; the per-call ctx still cancels it on shutdown.
var httpClient = netclient.NewSimpleClient(12 * time.Second)

// ── 余额查询 Provider seam（3.0 Step 3d #8）────────────────────
// 范式见 internal/gaea/provider/provider.go 与 internal/ai/image_backend.go 的
// Register/New/Kinds。DeepSeek 形状注册为 kind "deepseek"；其他 provider 可
// 自注册自己的余额端点形状。消费者（Fetch/FetchByKind）只依赖 Provider 接口
// 与 kind；切换余额后端只改配置（kind + url）、代码零改动。

// BalanceKindDeepSeek DeepSeek GET /user/balance 形状（is_available +
// balance_infos）。保持历史默认。
const BalanceKindDeepSeek = "deepseek"

// Provider 余额查询能力接口：按 url/apiKey 查询并返回归一化余额。
type Provider interface {
	Fetch(ctx context.Context, url, apiKey string) (*Balance, error)
}

// BalanceProviderFactory 构建余额查询提供者（kind → 实例）。
type BalanceProviderFactory func() Provider

// balanceProviderRegistry kind → 工厂注册表。各实现 init() 自注册；互斥注册，
// 重复即 panic（编译期接线错误）。
var balanceProviderRegistry = map[string]BalanceProviderFactory{}

func init() {
	RegisterBalanceProvider(BalanceKindDeepSeek, func() Provider { return deepseekProvider{} })
}

// RegisterBalanceProvider 注册余额查询后端 kind（如 "deepseek"）。供各实现
// init() 自注册；kind 为空或重复注册直接 panic。
func RegisterBalanceProvider(kind string, factory BalanceProviderFactory) {
	if kind == "" {
		panic("billing: balance provider kind must not be empty")
	}
	if _, dup := balanceProviderRegistry[kind]; dup {
		panic("billing: duplicate balance provider kind " + kind)
	}
	balanceProviderRegistry[kind] = factory
}

// NewBalanceProvider 按 kind 经注册表构建提供者；未知 kind 返回错误
// （fail-closed，附已注册 kind 列表）。
func NewBalanceProvider(kind string) (Provider, error) {
	factory, ok := balanceProviderRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("billing: unknown balance provider kind %q (registered: %v)", kind, BalanceProviderKinds())
	}
	p := factory()
	if p == nil {
		return nil, fmt.Errorf("billing: balance provider factory %q returned nil", kind)
	}
	return p, nil
}

// BalanceProviderKinds 返回已注册余额后端 kind 列表（排序，供诊断/校验）。
func BalanceProviderKinds() []string {
	out := make([]string, 0, len(balanceProviderRegistry))
	for k := range balanceProviderRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// deepseekProvider 实现 DeepSeek GET /user/balance 形状。
type deepseekProvider struct{}

func (deepseekProvider) Fetch(ctx context.Context, url, apiKey string) (*Balance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("balance: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var dr deepseekResp
	if err := json.Unmarshal(body, &dr); err != nil {
		return nil, fmt.Errorf("balance: decode: %w", err)
	}
	b := &Balance{Available: dr.IsAvailable}
	for _, bi := range dr.BalanceInfos {
		b.Infos = append(b.Infos, Info{
			Currency:        bi.Currency,
			TotalBalance:    bi.TotalBalance,
			GrantedBalance:  bi.GrantedBalance,
			ToppedUpBalance: bi.ToppedUpBalance,
		})
	}
	return b, nil
}

// FetchByKind 按 kind 经注册表查询余额。空 url 一律返回 (nil, nil)——"未配置"
// 而非错误——与 kind 无关；未知 kind fail-closed（返回错误，不静默降级）。
func FetchByKind(ctx context.Context, kind, url, apiKey string) (*Balance, error) {
	if strings.TrimSpace(url) == "" {
		return nil, nil
	}
	p, err := NewBalanceProvider(kind)
	if err != nil {
		return nil, err
	}
	return p.Fetch(ctx, url, apiKey)
}

// Fetch 查询余额（默认 DeepSeek 形状，kind=deepseek）。
// 兼容既有调用方（controller.Balance 等）与历史测试：语义与旧实现完全一致。
func Fetch(ctx context.Context, url, apiKey string) (*Balance, error) {
	return FetchByKind(ctx, BalanceKindDeepSeek, url, apiKey)
}

// symbol maps an ISO currency code to a compact symbol; an unknown code passes
// through with a trailing space ("XYZ 12.00").
func symbol(currency string) string {
	switch strings.ToUpper(currency) {
	case "CNY", "RMB":
		return "¥"
	case "USD":
		return "$"
	default:
		if currency == "" {
			return ""
		}
		return currency + " "
	}
}

// Display renders the primary balance compactly, e.g. "¥110.00". It prefers CNY,
// then the first currency reported. "" when there's nothing to show.
func (b *Balance) Display() string {
	if b == nil || len(b.Infos) == 0 {
		return ""
	}
	pick := b.Infos[0]
	for _, i := range b.Infos {
		if strings.EqualFold(i.Currency, "CNY") {
			pick = i
			break
		}
	}
	return symbol(pick.Currency) + strings.TrimSpace(pick.TotalBalance)
}
