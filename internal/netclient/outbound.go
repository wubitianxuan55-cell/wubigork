// outbound.go 是出网许可层（codex 蒸馏 B9 刀1，第四批 P4）。
//
// 机制同构 codex network_policy 的「发送前 acquire + 确定性四态拒因」，
// 但落在 gaea 既有的唯一 HTTP 工厂 netclient 里——不另起平行层：
//   - 出网策略 OutboundPolicy：unrestricted（默认，行为与现状逐字节等价）
//     / restricted（https 白名单 + 回环豁免）；
//   - OutboundGuard 持策略与版本号：Acquire 发放带版本的许可，SetPolicy
//     换策略即撤销全部旧许可（revoked）；CheckURL 供传输层逐跳复查；
//   - InstallOutbound 给 http.Client 装重定向纪律：≤10 跳、每跳过闸、
//     跨源剥离 Authorization/Cookie/Proxy-Authorization 并补 Referer。
//
// B7「构造即验证」纪律：NewOutboundGuard 在构造期校验策略（restricted
// 必须带白名单），SetPolicy 同样校验——不存在「默认宽、后设窄」的豁免口。
package netclient

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 出网模式。
const (
	OutboundUnrestricted = "unrestricted"
	OutboundRestricted   = "restricted"
)

// 四态拒因（确定性拒绝，与可重试的网络故障分家——可重试归 retry 层管）。
const (
	// DenyUnavailable：策略层不可用（guard 未装配/未初始化就被要求裁决）。
	DenyUnavailable = "unavailable"
	// DenyDestination：目标主机或协议不在白名单。
	DenyDestination = "destination"
	// DenyRevoked：策略已变更，旧许可失效。
	DenyRevoked = "revoked"
	// DenyUnsupportedTransport：传输形态不受支持（ws:// 等非 HTTP(S) 请求面）。
	DenyUnsupportedTransport = "unsupported_transport"
)

// MaxRedirectHops 是重定向上限（第 11 跳即终止，不发第 11 次请求）。
const MaxRedirectHops = 10

// OutboundPolicy 是一份出网策略。AllowedHosts 精确主机或子域后缀匹配
//（口径同 withDirectHosts）；回环（localhost/127.0.0.1/::1）恒放行——
// 本机引擎（herdsman/llama/ComfyUI）不是「出网」，restricted 也不该断本地。
type OutboundPolicy struct {
	Mode         string
	AllowedHosts []string
}

// OutboundError 是一次确定性的出网拒绝：Reason 是四态之一，URL 是被拒
// 目标，Detail 给出可读依据。错误即值——调用方按 Reason 分类，不解析文本。
type OutboundError struct {
	Reason string
	URL    string
	Detail string
}

func (e *OutboundError) Error() string {
	return fmt.Sprintf("outbound denied (%s): %s: %s", e.Reason, e.URL, e.Detail)
}

// OutboundGuard 是带版本号的出网闸门。零值不可用（B7：必须经
// NewOutboundGuard 构造并校验，不存在默认放行的零值形态）。
type OutboundGuard struct {
	mu       sync.RWMutex
	policy   OutboundPolicy
	revision uint64
	ready    bool
}

// NewOutboundGuard 构造并验证一份出网闸门（构造即验证）：
// restricted 模式必须给出非空白名单，否则拒绝构造——宁可建不出来，
// 不给「restricted 却全放行」的静默宽口。
func NewOutboundGuard(policy OutboundPolicy) (*OutboundGuard, error) {
	if err := ValidateOutboundPolicy(policy); err != nil {
		return nil, err
	}
	return &OutboundGuard{policy: normalizedPolicy(policy), revision: 1, ready: true}, nil
}

// ValidateOutboundPolicy 校验策略自洽性（构造与换策略共用同一判据）。
func ValidateOutboundPolicy(policy OutboundPolicy) error {
	switch strings.ToLower(strings.TrimSpace(policy.Mode)) {
	case "", OutboundUnrestricted:
		return nil
	case OutboundRestricted:
		if len(policy.AllowedHosts) == 0 {
			return fmt.Errorf("outbound restricted policy requires allowed_hosts (empty whitelist would deny or mislead)")
		}
		return nil
	default:
		return fmt.Errorf("outbound mode %q: must be unrestricted|restricted", policy.Mode)
	}
}

func normalizedPolicy(policy OutboundPolicy) OutboundPolicy {
	mode := strings.ToLower(strings.TrimSpace(policy.Mode))
	if mode == "" {
		mode = OutboundUnrestricted
	}
	hosts := make([]string, 0, len(policy.AllowedHosts))
	for _, h := range policy.AllowedHosts {
		if h = normalizeHost(h); h != "" {
			hosts = append(hosts, h)
		}
	}
	return OutboundPolicy{Mode: mode, AllowedHosts: hosts}
}

// normalizeHost 归一主机名：小写 + 剥末尾点（"api.example.com." 与
// "api.example.com" 同一主机——DNS 尾点合法形态，见验收 N2）。
func normalizeHost(h string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(h)), ".")
}

// OutboundPermit 是一次许可。持有版本号，策略变更后 Valid() 为假。
type OutboundPermit struct {
	guard    *OutboundGuard
	revision uint64
	URL      string
}

// Valid 报告许可是否仍然有效（策略未变更）。
func (p OutboundPermit) Valid() bool {
	if p.guard == nil {
		return false
	}
	p.guard.mu.RLock()
	defer p.guard.mu.RUnlock()
	return p.guard.ready && p.guard.revision == p.revision
}

// SetPolicy 换策略并撤销全部既有许可（旧许可 Valid() 即假）。同样走
// 构造期判据——非法策略换不进来。
func (g *OutboundGuard) SetPolicy(policy OutboundPolicy) error {
	if err := ValidateOutboundPolicy(policy); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.policy = normalizedPolicy(policy)
	g.revision++
	g.ready = true
	return nil
}

// Acquire 对目标 URL 发放许可。被拒返回 *OutboundError（四态拒因之一）。
func (g *OutboundGuard) Acquire(rawURL string) (OutboundPermit, error) {
	if g == nil {
		return OutboundPermit{}, &OutboundError{Reason: DenyUnavailable, URL: rawURL, Detail: "no outbound guard configured"}
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.ready {
		return OutboundPermit{}, &OutboundError{Reason: DenyUnavailable, URL: rawURL, Detail: "guard not initialized"}
	}
	if err := g.evaluateLocked(rawURL); err != nil {
		return OutboundPermit{}, err
	}
	return OutboundPermit{guard: g, revision: g.revision, URL: rawURL}, nil
}

// CheckURL 是传输层逐跳复查（无许可对象，按当前策略裁决）。
func (g *OutboundGuard) CheckURL(rawURL string) error {
	if g == nil {
		return &OutboundError{Reason: DenyUnavailable, URL: rawURL, Detail: "no outbound guard configured"}
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.ready {
		return &OutboundError{Reason: DenyUnavailable, URL: rawURL, Detail: "guard not initialized"}
	}
	return g.evaluateLocked(rawURL)
}

func (g *OutboundGuard) evaluateLocked(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return &OutboundError{Reason: DenyDestination, URL: rawURL, Detail: "unparseable URL"}
	}
	if g.policy.Mode != OutboundRestricted {
		return nil
	}
	scheme := strings.ToLower(u.Scheme)
	host := normalizeHost(u.Hostname())
	if isLoopbackHost(host) {
		return nil // 本机引擎豁免：不出网
	}
	switch scheme {
	case "https":
		// 唯一放行的出网协议，继续过白名单。
	case "http", "ws", "wss", "ftp", "gopher":
		return &OutboundError{Reason: DenyDestination, URL: rawURL,
			Detail: "restricted outbound allows https only (loopback exempt)"}
	case "":
		return &OutboundError{Reason: DenyUnsupportedTransport, URL: rawURL, Detail: "no scheme"}
	default:
		return &OutboundError{Reason: DenyUnsupportedTransport, URL: rawURL,
			Detail: "unsupported transport scheme " + scheme}
	}
	for _, allowed := range g.policy.AllowedHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}
	return &OutboundError{Reason: DenyDestination, URL: rawURL,
		Detail: "host not in outbound allowed_hosts"}
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]"
}

// InstallOutbound 给客户端装出网纪律（guard 为 nil = 不装，行为与现状
// 逐字节一致）：
//   - 重定向 ≤ MaxRedirectHops 跳，第 11 跳直接终止并报错；
//   - 每一跳（含首跳由 Transport 层复查，见 OutboundTransport）过闸；
//   - 跨源跳转剥离 Authorization / Cookie / Proxy-Authorization，并补
//     Referer 指向发起方——凭据不出 origen（验收 N4）；同源不删。
func InstallOutbound(cli *http.Client, guard *OutboundGuard) {
	if cli == nil || guard == nil {
		return
	}
	cli.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= MaxRedirectHops {
			return fmt.Errorf("outbound: stopped after %d redirects (last: %s)", MaxRedirectHops, req.URL.String())
		}
		if err := guard.CheckURL(req.URL.String()); err != nil {
			return err
		}
		prev := via[len(via)-1]
		if sameOrigin(prev.URL, req.URL) {
			return nil
		}
		// 跨源：凭据头不跟随（服务器侧 Set-Cookie 由 cookiejar 决定，这里只管
		// 请求头），补 Referer 供目标侧归因——凭据不出源站。
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
		req.Header.Del("Proxy-Authorization")
		req.Header.Set("Referer", prev.URL.String())
		return nil
	}
	cli.Transport = OutboundTransport(cli.Transport, guard)
}

// sameOrigin 判定两个 URL 是否同源（scheme+host+port）。
func sameOrigin(a, b *url.URL) bool {
	ap := canonicalPort(a)
	bp := canonicalPort(b)
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Hostname(), b.Hostname()) && ap == bp
}

func canonicalPort(u *url.URL) string {
	p := u.Port()
	if p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

// OutboundTransport 包一层传输：首跳（以及未被 CheckRedirect 覆盖的路径）
// 在拨号前过闸。base 为 nil 时用 http.DefaultTransport。
func OutboundTransport(base http.RoundTripper, guard *OutboundGuard) http.RoundTripper {
	if guard == nil {
		return base
	}
	if base == nil {
		base = http.DefaultTransport
	}
	return &outboundTransport{base: base, guard: guard}
}

type outboundTransport struct {
	base  http.RoundTripper
	guard *OutboundGuard
}

func (t *outboundTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.guard.CheckURL(req.URL.String()); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req)
}
