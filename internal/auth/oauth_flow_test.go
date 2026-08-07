package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

// newTokenEndpoint 返回记录表单参数并回送指定 JSON 的 token 端点。
func newTokenEndpoint(t *testing.T, status int, body string, gotForm *url.Values) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if gotForm != nil {
			*gotForm = r.Form
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestDiscoverEndpoints_UsesConfigURL Discovery URL 必须来自配置
// （修复：此前硬编码 auth.x.ai，cfg.OIDCDiscoveryURL 与 XAI_OIDC_DISCOVERY_URL 环境变量完全失效）。
func TestDiscoverEndpoints_UsesConfigURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 "https://issuer.example",
			"authorization_endpoint": "https://auth.example/authorize",
			"token_endpoint":         "https://auth.example/token",
		})
	}))
	t.Cleanup(srv.Close)

	cfg := &config.Config{OIDCDiscoveryURL: srv.URL}
	disc, err := DiscoverEndpoints(cfg)
	if err != nil {
		t.Fatalf("DiscoverEndpoints: %v", err)
	}
	if disc.AuthorizationEndpoint != "https://auth.example/authorize" {
		t.Errorf("authorization_endpoint = %q", disc.AuthorizationEndpoint)
	}
	if disc.TokenEndpoint != "https://auth.example/token" {
		t.Errorf("token_endpoint = %q", disc.TokenEndpoint)
	}
}

// TestExchangeCodeForToken_Success 授权码换 token：必须携带 verifier + challenge，解析 token 并记录获取时间。
func TestExchangeCodeForToken_Success(t *testing.T) {
	var form url.Values
	srv := newTokenEndpoint(t, 200,
		`{"access_token":"acc-1","refresh_token":"ref-1","token_type":"Bearer","expires_in":21600,"scope":"openid"}`,
		&form)

	cfg := &config.Config{XaiClientID: "test-client", XaiAPIBaseURL: "https://api.x.ai/v1"}
	pkce, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	token, raw, baseURL, err := exchangeCodeForToken(cfg, srv.URL, "auth-code", "http://127.0.0.1:56121/callback", pkce)
	if err != nil {
		t.Fatalf("exchangeCodeForToken: %v", err)
	}
	if token.AccessToken != "acc-1" || token.RefreshToken != "ref-1" {
		t.Errorf("token = %+v", token)
	}
	if token.ObtainedAt.IsZero() {
		t.Error("ObtainedAt 应被设置为当前时间")
	}
	if len(raw) == 0 || baseURL != "https://api.x.ai/v1" {
		t.Errorf("raw/baseURL = %q/%q", raw, baseURL)
	}
	// xAI 在 token 步骤也验证 challenge：两个参数都必须存在
	if form.Get("code_verifier") != pkce.Verifier {
		t.Errorf("code_verifier = %q, want %q", form.Get("code_verifier"), pkce.Verifier)
	}
	if form.Get("code_challenge") != pkce.Challenge {
		t.Errorf("code_challenge = %q", form.Get("code_challenge"))
	}
	if form.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q", form.Get("code_challenge_method"))
	}
	if form.Get("grant_type") != "authorization_code" || form.Get("code") != "auth-code" {
		t.Errorf("grant_type/code = %q/%q", form.Get("grant_type"), form.Get("code"))
	}
}

// TestExchangeCodeForToken_500 E04 场景：token 端点 500 必须给出可诊断错误（含状态码与响应体）。
func TestExchangeCodeForToken_500(t *testing.T) {
	srv := newTokenEndpoint(t, 500, `{"error":"server_error"}`, nil)
	cfg := &config.Config{XaiClientID: "test-client", XaiAPIBaseURL: "https://api.x.ai/v1"}
	pkce, _ := newPKCE()
	_, _, _, err := exchangeCodeForToken(cfg, srv.URL, "code", "http://127.0.0.1:56121/callback", pkce)
	if err == nil {
		t.Fatal("500 应报错")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "server_error") {
		t.Errorf("错误信息 = %q, want 含 500 与响应体", err.Error())
	}
}

// TestExchangeCodeForToken_403 账号未获授权：给出明确引导（而不是裸 403）。
func TestExchangeCodeForToken_403(t *testing.T) {
	srv := newTokenEndpoint(t, 403, `{"error":"access_denied"}`, nil)
	cfg := &config.Config{XaiClientID: "test-client", XaiAPIBaseURL: "https://api.x.ai/v1"}
	pkce, _ := newPKCE()
	_, _, _, err := exchangeCodeForToken(cfg, srv.URL, "code", "http://127.0.0.1:56121/callback", pkce)
	if err == nil {
		t.Fatal("403 应报错")
	}
	if !strings.Contains(err.Error(), "未获 API 访问授权") {
		t.Errorf("错误信息 = %q, want 含授权引导", err.Error())
	}
}

// TestRefreshAccessToken_Success 刷新 token：discovery 用配置 URL，表单带 refresh_token + client_id。
func TestRefreshAccessToken_Success(t *testing.T) {
	var form url.Values
	tokenSrv := newTokenEndpoint(t, 200,
		`{"access_token":"new-acc","refresh_token":"new-ref","token_type":"Bearer","expires_in":21600}`,
		&form)
	discSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "https://auth.example/authorize",
			"token_endpoint":         tokenSrv.URL,
		})
	}))
	t.Cleanup(discSrv.Close)

	cfg := &config.Config{XaiClientID: "test-client", OIDCDiscoveryURL: discSrv.URL}
	token, err := RefreshAccessToken(cfg, "old-refresh")
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if token.AccessToken != "new-acc" || token.RefreshToken != "new-ref" {
		t.Errorf("token = %+v", token)
	}
	if token.ObtainedAt.IsZero() {
		t.Error("ObtainedAt 应为当前时间")
	}
	if form.Get("grant_type") != "refresh_token" {
		t.Errorf("grant_type = %q", form.Get("grant_type"))
	}
	if form.Get("refresh_token") != "old-refresh" {
		t.Errorf("refresh_token = %q", form.Get("refresh_token"))
	}
	if form.Get("client_id") != "test-client" {
		t.Errorf("client_id = %q", form.Get("client_id"))
	}
}

func TestRefreshAccessToken_500(t *testing.T) {
	tokenSrv := newTokenEndpoint(t, 500, `{"error":"refresh_failed"}`, nil)
	discSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "https://auth.example/authorize",
			"token_endpoint":         tokenSrv.URL,
		})
	}))
	t.Cleanup(discSrv.Close)

	_, err := RefreshAccessToken(&config.Config{XaiClientID: "c", OIDCDiscoveryURL: discSrv.URL}, "ref")
	if err == nil {
		t.Fatal("500 应报错")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "refresh_failed") {
		t.Errorf("错误信息 = %q", err.Error())
	}
}

func TestRefreshAccessToken_403(t *testing.T) {
	tokenSrv := newTokenEndpoint(t, 403, `{"error":"access_denied"}`, nil)
	discSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "https://auth.example/authorize",
			"token_endpoint":         tokenSrv.URL,
		})
	}))
	t.Cleanup(discSrv.Close)

	_, err := RefreshAccessToken(&config.Config{XaiClientID: "c", OIDCDiscoveryURL: discSrv.URL}, "ref")
	if err == nil {
		t.Fatal("403 应报错")
	}
	if !strings.Contains(err.Error(), "未获 API 访问授权") {
		t.Errorf("错误信息 = %q, want 含授权引导", err.Error())
	}
}

// TestExchangeCodeForToken_EmptyVerifier 空 code_verifier 必须显式报错（防 bug 静默）。
func TestExchangeCodeForToken_EmptyVerifier(t *testing.T) {
	_, _, _, err := exchangeCodeForToken(&config.Config{XaiClientID: "c"}, "https://example.invalid/token",
		"code", "http://127.0.0.1:56121/callback", &pkcePair{})
	if err == nil {
		t.Fatal("空 verifier 应报错")
	}
	if !strings.Contains(err.Error(), "code_verifier") {
		t.Errorf("错误信息 = %q", err.Error())
	}
}
