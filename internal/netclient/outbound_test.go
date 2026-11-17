package netclient

// B9 刀1（第四批 P4）出网许可层验收。口径=asset-53 §12 N1–N5（N6 TLS
// 回退是 codex rustls 双后端特有，Go 标准库无此概念，不搬；N7 模型目录
// 缓存不在 P4 落地面）+ B7 构造即验证。

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mustGuard(t *testing.T, policy OutboundPolicy) *OutboundGuard {
	t.Helper()
	g, err := NewOutboundGuard(policy)
	if err != nil {
		t.Fatalf("NewOutboundGuard: %v", err)
	}
	return g
}

func denyReason(t *testing.T, err error) string {
	t.Helper()
	oe, ok := err.(*OutboundError)
	if !ok {
		t.Fatalf("err = %v, want *OutboundError", err)
	}
	return oe.Reason
}

// N1：Restricted 下 http:// 与 ws:// 一律 Destination 拒绝；https 白名单
// 主机放行；回环豁免（本机引擎不是出网）。
func TestOutboundRestrictedSchemesAndHosts(t *testing.T) {
	g := mustGuard(t, OutboundPolicy{Mode: OutboundRestricted, AllowedHosts: []string{"api.deepseek.com"}})

	if err := g.CheckURL("http://api.deepseek.com/v1/chat"); denyReason(t, err) != DenyDestination {
		t.Errorf("http:// want destination deny, got %v", err)
	}
	if err := g.CheckURL("ws://api.deepseek.com/socket"); denyReason(t, err) != DenyDestination {
		t.Errorf("ws:// want destination deny, got %v", err)
	}
	if err := g.CheckURL("https://api.deepseek.com/v1/chat"); err != nil {
		t.Errorf("https allowed host: %v", err)
	}
	if err := g.CheckURL("https://evil.example.com/x"); denyReason(t, err) != DenyDestination {
		t.Errorf("non-whitelisted host want destination deny, got %v", err)
	}
	// 子域后缀匹配。
	if err := g.CheckURL("https://cn.api.deepseek.com/v1"); err != nil {
		t.Errorf("subdomain of allowed host: %v", err)
	}
	// 回环豁免：本机引擎（http://127.0.0.1:8080）不属出网。
	if err := g.CheckURL("http://127.0.0.1:8080/v1"); err != nil {
		t.Errorf("loopback http must be exempt, got %v", err)
	}
	if err := g.CheckURL("http://localhost:8080/v1"); err != nil {
		t.Errorf("localhost http must be exempt, got %v", err)
	}
	// 无 scheme → unsupported transport。
	if err := g.CheckURL("api.deepseek.com/v1"); denyReason(t, err) != DenyUnsupportedTransport {
		t.Errorf("schemeless want unsupported_transport, got %v", err)
	}
}

// N2：主机名末尾带点仍匹配（DNS 尾点合法形态）。
func TestOutboundTrailingDotHost(t *testing.T) {
	g := mustGuard(t, OutboundPolicy{Mode: OutboundRestricted, AllowedHosts: []string{"api.deepseek.com."}})
	if err := g.CheckURL("https://api.deepseek.com/v1"); err != nil {
		t.Errorf("trailing-dot whitelist vs plain host: %v", err)
	}
	g2 := mustGuard(t, OutboundPolicy{Mode: OutboundRestricted, AllowedHosts: []string{"api.deepseek.com"}})
	if err := g2.CheckURL("https://api.deepseek.com./v1"); err != nil {
		t.Errorf("plain whitelist vs trailing-dot host: %v", err)
	}
}

// N3：策略变更后旧许可失效（Valid=false），新请求按新策略判定。
func TestOutboundPermitRevokedOnPolicyChange(t *testing.T) {
	g := mustGuard(t, OutboundPolicy{Mode: OutboundRestricted, AllowedHosts: []string{"a.example.com"}})
	permit, err := g.Acquire("https://a.example.com/x")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !permit.Valid() {
		t.Fatal("fresh permit must be valid")
	}
	if err := g.SetPolicy(OutboundPolicy{Mode: OutboundRestricted, AllowedHosts: []string{"b.example.com"}}); err != nil {
		t.Fatalf("SetPolicy: %v", err)
	}
	if permit.Valid() {
		t.Error("permit must be revoked after policy change")
	}
	if err := g.CheckURL("https://a.example.com/x"); denyReason(t, err) != DenyDestination {
		t.Errorf("old host must now deny, got %v", err)
	}
	if err := g.CheckURL("https://b.example.com/x"); err != nil {
		t.Errorf("new host must pass, got %v", err)
	}
}

// B7 构造即验证：restricted 空白名单拒绝构造/换入；未知 mode 拒绝；
// 零值 guard 裁决 unavailable（不存在默认放行的零值形态）。
func TestOutboundConstructTimeValidation(t *testing.T) {
	if _, err := NewOutboundGuard(OutboundPolicy{Mode: OutboundRestricted}); err == nil {
		t.Error("restricted with empty whitelist must fail construction")
	}
	if _, err := NewOutboundGuard(OutboundPolicy{Mode: "yolo"}); err == nil {
		t.Error("unknown mode must fail construction")
	}
	g := mustGuard(t, OutboundPolicy{Mode: OutboundUnrestricted})
	if err := g.SetPolicy(OutboundPolicy{Mode: OutboundRestricted}); err == nil {
		t.Error("SetPolicy must reject empty-whitelist restricted")
	}
	var nilGuard *OutboundGuard
	if err := nilGuard.CheckURL("https://a.example.com/"); denyReason(t, err) != DenyUnavailable {
		t.Errorf("nil guard want unavailable, got %v", err)
	}
}

// N4：跨源 302 剥 Authorization/Cookie/Proxy-Authorization 并补 Referer；
// 同源 302 不删头。
func TestOutboundCrossOriginRedirectStripsCredentials(t *testing.T) {
	var gotAuth, gotCookie, gotReferer string
	cross := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		gotReferer = r.Header.Get("Referer")
		w.WriteHeader(http.StatusOK)
	}))
	defer cross.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, cross.URL+"/sink", http.StatusFound)
	}))
	defer origin.Close()

	g := mustGuard(t, OutboundPolicy{Mode: OutboundUnrestricted})
	cli := &http.Client{}
	InstallOutbound(cli, g)
	req, _ := http.NewRequest("GET", origin.URL+"/start", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "session=1")
	req.Header.Set("Proxy-Authorization", "Basic xyz")
	resp, err := cli.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if gotAuth != "" || gotCookie != "" {
		t.Errorf("cross-origin hop kept credentials: auth=%q cookie=%q", gotAuth, gotCookie)
	}
	if gotReferer != origin.URL+"/start" {
		t.Errorf("Referer = %q, want origin start URL", gotReferer)
	}
}

func TestOutboundSameOriginRedirectKeepsCredentials(t *testing.T) {
	var sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hop" {
			http.Redirect(w, r, "/end", http.StatusFound)
			return
		}
		sawAuth = r.Header.Get("Authorization") != ""
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := mustGuard(t, OutboundPolicy{Mode: OutboundUnrestricted})
	cli := &http.Client{}
	InstallOutbound(cli, g)
	req, _ := http.NewRequest("GET", srv.URL+"/hop", nil)
	req.Header.Set("Authorization", "Bearer keepme")
	resp, err := cli.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if !sawAuth {
		t.Error("same-origin redirect must keep Authorization")
	}
}

// N5：重定向 11 跳终止并给出明确错误，不发第 11 次请求。
func TestOutboundRedirectHopLimit(t *testing.T) {
	hops := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops++
		http.Redirect(w, r, "/next", http.StatusFound)
	}))
	defer srv.Close()

	g := mustGuard(t, OutboundPolicy{Mode: OutboundUnrestricted})
	cli := &http.Client{}
	InstallOutbound(cli, g)
	_, err := cli.Get(srv.URL + "/start")
	if err == nil {
		t.Fatal("expected redirect-limit error")
	}
	if !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Errorf("err = %v, want hop-limit message", err)
	}
	if hops != MaxRedirectHops {
		t.Errorf("server saw %d hops, want %d（第 11 跳不发）", hops, MaxRedirectHops)
	}
}

// Transport 层首跳过闸：restricted 下白名单外主机在拨号前被拒。
func TestOutboundTransportChecksFirstHop(t *testing.T) {
	g := mustGuard(t, OutboundPolicy{Mode: OutboundRestricted, AllowedHosts: []string{"allowed.example.com"}})
	tr := OutboundTransport(http.DefaultTransport, g)
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "denied.example.com"}}
	if _, err := tr.RoundTrip(req); denyReason(t, err) != DenyDestination {
		t.Errorf("first hop want destination deny, got %v", err)
	}
}

// 首跳在 redirect hook 之前被 Transport 拦截时，InstallOutbound(nil guard)
// 与不装完全等价（现状红线）。
func TestInstallOutboundNilGuardNoop(t *testing.T) {
	cli := &http.Client{}
	InstallOutbound(cli, nil)
	if cli.CheckRedirect != nil || cli.Transport != nil {
		t.Error("nil guard must not touch the client")
	}
}
