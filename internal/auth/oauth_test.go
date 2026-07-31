package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

func TestPKCE(t *testing.T) {
	pair, err := newPKCE()
	if err != nil {
		t.Fatalf("newPKCE() failed: %v", err)
	}
	if pair.Verifier == "" {
		t.Error("code_verifier is empty")
	}
	if pair.Challenge == "" {
		t.Error("code_challenge is empty")
	}

	// 验证: SHA256(verifier) == base64url(challenge)
	h := sha256.Sum256([]byte(pair.Verifier))
	expected := base64.RawURLEncoding.EncodeToString(h[:])
	if pair.Challenge != expected {
		t.Errorf("code_challenge mismatch:\n  got:      %s\n  expected: %s", pair.Challenge, expected)
	}

	// Verifier 应为 43 字符 (32 bytes base64url)
	if len(pair.Verifier) != 43 {
		t.Errorf("code_verifier length = %d, expected 43", len(pair.Verifier))
	}
}

func TestBuildAuthURL(t *testing.T) {
	cfg := config.Load()
	redirectURI := "http://127.0.0.1:56121/callback"

	authURL := buildAuthURL("https://auth.x.ai/oauth2/authorize", cfg, redirectURI, "test_challenge", "test_state", "test_nonce")

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("buildAuthURL returned invalid URL: %v", err)
	}

	q := parsed.Query()

	// 必需参数检查
	checks := map[string]string{
		"response_type":         "code",
		"client_id":             cfg.XaiClientID,
		"redirect_uri":          redirectURI,
		"code_challenge_method": "S256",
		"code_challenge":        "test_challenge",
		"state":                 "test_state",
		"nonce":                 "test_nonce",
		"plan":                  "generic", // 关键参数！
		"referrer":              "gaea",
		"scope":                 "openid profile email offline_access grok-cli:access api:access",
	}

	for key, want := range checks {
		got := q.Get(key)
		if got != want {
			t.Errorf("query param %q: got %q, want %q", key, got, want)
		}
	}

	// plan=generic 是关键参数——明确验证
	if q.Get("plan") != "generic" {
		t.Errorf("CRITICAL: 'plan=generic' is missing — xAI will reject loopback OAuth without it")
	}
}

func TestValidateInferenceBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "https://api.x.ai/v1"},
		{"https://api.x.ai/v1", "https://api.x.ai/v1"},
		{"https://api.x.ai/v1/", "https://api.x.ai/v1"},
		{"https://staging.x.ai/v1", "https://staging.x.ai/v1"},
		// 非 xAI 域 → 回退到默认
		{"https://attacker.example/v1", "https://api.x.ai/v1"},
		{"https://evil.x.ai.com/v1", "https://api.x.ai/v1"}, // x.ai.com ≠ *.x.ai
		// 非 HTTPS → 回退
		{"http://api.x.ai/v1", "https://api.x.ai/v1"},
		// 无 hostname → 回退
		{"https:///v1", "https://api.x.ai/v1"},
	}

	for _, tc := range tests {
		got := validateInferenceBaseURL(tc.input)
		if got != tc.expected {
			t.Errorf("validateInferenceBaseURL(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestOIDCDiscovery(t *testing.T) {
	// 真实网络调用：验证 xAI OIDC Discovery 端点的实际响应
	disc, err := DiscoverEndpoints()
	if err != nil {
		t.Logf("OIDC Discovery 可能因网络问题失败: %v", err)
		return
	}
	if disc.AuthorizationEndpoint == "" {
		t.Error("authorization_endpoint is empty")
	}
	if disc.TokenEndpoint == "" {
		t.Error("token_endpoint is empty")
	}

	// 验证端点格式
	if !strings.HasPrefix(disc.AuthorizationEndpoint, "https://auth.x.ai/") {
		t.Errorf("unexpected authorization_endpoint: %s", disc.AuthorizationEndpoint)
	}
	if !strings.HasPrefix(disc.TokenEndpoint, "https://auth.x.ai/") {
		t.Errorf("unexpected token_endpoint: %s", disc.TokenEndpoint)
	}

	t.Logf("✓ authorization_endpoint: %s", disc.AuthorizationEndpoint)
	t.Logf("✓ token_endpoint: %s", disc.TokenEndpoint)
}

func TestTokenExpired(t *testing.T) {
	// 空 token → 过期
	var tok *Token
	if !tok.IsExpired() {
		t.Error("nil token should be expired")
	}

	tok = &Token{AccessToken: "", ExpiresIn: 3600}
	if !tok.IsExpired() {
		t.Error("empty access_token should be expired")
	}

	// 无过期信息 → 未过期
	tok = &Token{AccessToken: "xxx", ExpiresIn: 0}
	if tok.IsExpired() {
		t.Error("token with no expiry info should not be expired")
	}
}

func TestTokenValidate(t *testing.T) {
	if err := (*Token)(nil).Validate(); err == nil {
		t.Error("nil token should fail validation")
	}

	tok := &Token{AccessToken: "", RefreshToken: "r"}
	if err := tok.Validate(); err == nil {
		t.Error("empty access_token should fail validation")
	}

	tok = &Token{AccessToken: "a", RefreshToken: ""}
	if err := tok.Validate(); err != nil {
		t.Errorf("token without refresh_token should still validate: %v", err)
	}
}
