package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/config"
)

// PKCE 生成 code_verifier 和 code_challenge
type pkcePair struct {
	Verifier  string
	Challenge string
}

func newPKCE() (*pkcePair, error) {
	// 生成 32 字节随机数 → code_verifier
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("生成随机数失败: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)

	// code_challenge = SHA256(code_verifier) → base64url
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	return &pkcePair{Verifier: verifier, Challenge: challenge}, nil
}

// OAuthResult 登录完成后的结果
type OAuthResult struct {
	Token      *Token
	RawPayload []byte
	BaseURL    string
}

// DoLogin 执行 OAuth PKCE loopback 登录
//
// 完整对齐 hermes-agent 的 _xai_oauth_loopback_login + _xai_oauth_discovery 流程:
//  1. OIDC Discovery 获取真实端点
//  2. 生成 PKCE code_verifier / code_challenge
//  3. 打开浏览器让用户授权 (含 plan=generic / referrer=wubigork /
//     nonce 等 xAI 必要参数)
//  4. 本机 HTTP server 接收 callback
//  5. 用 code + code_verifier + code_challenge 换取 token
func DoLogin(cfg *config.Config) (*OAuthResult, error) {
	// ── 1. OIDC Discovery ──────────────────────────────────────────
	disc, err := DiscoverEndpoints()
	if err != nil {
		return nil, fmt.Errorf("获取 OIDC 端点失败: %w", err)
	}

	// ── 2. PKCE ────────────────────────────────────────────────────
	pkce, err := newPKCE()
	if err != nil {
		return nil, err
	}

	// 生成随机 state（防 CSRF）+ nonce（OIDC 必需）
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("生成 state 失败: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

	// ── 3. 启动本地 HTTP server 接收 callback ──────────────────────
	addr := net.JoinHostPort(cfg.RedirectHost, cfg.RedirectPort)
	redirectURI := fmt.Sprintf("http://%s/callback", addr)
	resultCh := make(chan *OAuthResult, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	var server *http.Server

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// 校验 state
		gotState := q.Get("state")
		if gotState != state {
			errCh <- fmt.Errorf("state 不匹配，可能的 CSRF 攻击")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(oauthErrorHTML("State 校验失败")))
			return
		}

		// 检查错误
		if errDesc := q.Get("error_description"); errDesc != "" {
			errCh <- fmt.Errorf("授权失败: %s", errDesc)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(oauthErrorHTML(errDesc)))
			return
		}
		if errParam := q.Get("error"); errParam != "" {
			errCh <- fmt.Errorf("授权失败: %s", errParam)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(oauthErrorHTML(errParam)))
			return
		}

		code := q.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("未收到授权码")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(oauthErrorHTML("未收到授权码")))
			return
		}

		// 后台换取 token（不阻塞 HTTP handler）
		go func() {
			token, raw, baseURL, err := exchangeCodeForToken(cfg, disc.TokenEndpoint, code, redirectURI, pkce)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- &OAuthResult{Token: token, RawPayload: raw, BaseURL: baseURL}
		}()

		w.Write([]byte(oauthSuccessHTML()))
	})

	server = &http.Server{Addr: addr, Handler: mux}

	// 启动 server
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("启动回调服务器失败 (端口 %s 被占用?): %w", cfg.RedirectPort, err)
	}
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("回调服务器错误: %w", err)
		}
	}()

	// 确保 server 被关闭
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// ── 4. 构建授权 URL 并打开浏览器 ───────────────────────────────
	authURL := buildAuthURL(disc.AuthorizationEndpoint, cfg, redirectURI, pkce.Challenge, state, nonce)
	fmt.Println("\n📱 请在浏览器中登录 xAI 账号：")
	fmt.Printf("   %s\n\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Println("   ⚠️  无法自动打开浏览器，请手动复制上面的链接")
	}

	// ── 5. 等待结果（超时 5 分钟）──────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	select {
	case result := <-resultCh:
		fmt.Println("✅ 登录成功！")
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("登录超时（5 分钟），请重试")
	}
}

// buildAuthURL 构建授权 URL
//
// 对齐 hermes-agent 的 _xai_oauth_build_authorize_url：
// — plan=generic 是 xAI 的关键参数：缺少它，非白名单客户端会被拒绝
// — referrer=wubigork 让 xAI 可以归因 OAuth 来源
// — nonce 是 OIDC 规范要求
func buildAuthURL(authEndpoint string, cfg *config.Config, redirectURI, challenge, state, nonce string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.XaiClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {"openid profile email offline_access grok-cli:access api:access"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"nonce":                 {nonce},
		"plan":                  {"generic"},  // 关键！没有此参数 xAI 拒绝 loopback OAuth
		"referrer":              {"wubigork"}, // 归因标识
	}

	return authEndpoint + "?" + params.Encode()
}

// exchangeCodeForToken 用 authorization code 换取 access token
//
// 对齐 hermes-agent 的 _xai_oauth_exchange_code_for_tokens：
// — 同时发送 code_verifier 和 code_challenge（xAI 在 token 步骤也会验证 challenge）
// — 缺少 code_challenge 会导致 "code_challenge is required" 错误
func exchangeCodeForToken(cfg *config.Config, tokenEndpoint, code, redirectURI string, pkce *pkcePair) (*Token, []byte, string, error) {
	if pkce.Verifier == "" {
		return nil, nil, "", fmt.Errorf("PKCE code_verifier 为空，这是一个 bug")
	}

	payload := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.XaiClientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkce.Verifier},
		// 防御性参数：xAI 在 token 步骤也会验证 challenge
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {"S256"},
	}

	resp, err := http.PostForm(tokenEndpoint, payload)
	if err != nil {
		return nil, nil, "", fmt.Errorf("请求 token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return nil, nil, "", fmt.Errorf("读取 token 响应失败: %w", err)
	}

	if resp.StatusCode == 403 {
		return nil, nil, "", fmt.Errorf(
			"换取 token 被拒 (HTTP 403): 此 xAI 账号未获 API 访问授权。"+
				"如果您是 SuperGrok 订阅用户但仍遇此问题，xAI 可能对部分订阅等级限制了 OAuth API 访问。"+
				"可尝试设置 XAI_API_KEY 环境变量改用 API Key 模式。"+
				"详情: %s", string(body))
	}

	if resp.StatusCode != 200 {
		return nil, nil, "", fmt.Errorf("换取 token 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, nil, "", fmt.Errorf("解析 token 响应失败: %w\n原始响应: %s", err, string(body))
	}
	token.ObtainedAt = time.Now()

	// 安全校验 inference base URL（防 credential leak）
	baseURL := validateInferenceBaseURL(cfg.XaiAPIBaseURL)

	return &token, body, baseURL, nil
}

// RefreshAccessToken 使用 refresh_token 刷新 access token
//
// 对齐 hermes-agent 的 refresh_xai_oauth_pure：
// — 通过 OIDC Discovery 获取 token_endpoint
// — 发送 client_id + refresh_token
func RefreshAccessToken(cfg *config.Config, refreshToken string) (*Token, error) {
	disc, err := DiscoverEndpoints()
	if err != nil {
		return nil, fmt.Errorf("获取 OIDC 端点失败: %w", err)
	}

	payload := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {cfg.XaiClientID},
		"refresh_token": {refreshToken},
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.PostForm(disc.TokenEndpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("刷新 token 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return nil, fmt.Errorf("读取刷新响应失败: %w", err)
	}

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf(
			"刷新 token 被拒 (HTTP 403): 此 xAI 账号未获 API 访问授权 — " +
				"xAI 可能对部分 SuperGrok 等级限制了 OAuth API 访问。" +
				"重新登录无法解决此问题。可尝试设置 XAI_API_KEY 环境变量改用 API Key 模式。")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("刷新 token 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("解析刷新响应失败: %w", err)
	}
	token.ObtainedAt = time.Now()

	return &token, nil
}

// validateInferenceBaseURL 安全校验：确保 inference URL 指向 xAI 官方域
//
// 对齐 hermes-agent 的 _xai_validate_inference_base_url：
// — 防止恶意 .env 将 token 发送到攻击者服务器
// — 只允许 https:// 且域名为 x.ai 或其子域名
func validateInferenceBaseURL(rawURL string) string {
	candidate := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if candidate == "" {
		return "https://api.x.ai/v1"
	}

	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme == "" {
		slog.Warn("忽略无效的 base URL，使用默认值", "url", candidate)
		return "https://api.x.ai/v1"
	}

	if parsed.Scheme != "https" {
		slog.Warn("拒绝非 HTTPS 的 base URL（会泄露 token），使用默认值", "url", candidate)
		return "https://api.x.ai/v1"
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "x.ai" && !strings.HasSuffix(host, ".x.ai") {
		slog.Warn("拒绝非 xAI 域的 base URL（会泄露 token），使用默认值", "url", candidate, "host", host)
		return "https://api.x.ai/v1"
	}

	return candidate
}

func oauthSuccessHTML() string {
	return `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>登录成功 — wubigork</title>
<style>body{font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#0d0d0d;color:#e0e0e0}div{text-align:center}h1{color:#4ade80}p{color:#9ca3af}</style></head>
<body><div><h1>✅ 登录成功</h1><p>您可以关闭此页面，回到 wubigork 终端继续写作 ✍️</p></div></body></html>`
}

func oauthErrorHTML(msg string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>登录失败 — wubigork</title>
<style>body{font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#0d0d0d;color:#e0e0e0}div{text-align:center}h1{color:#f87171}p{color:#9ca3af}</style></head>
<body><div><h1>❌ 登录失败</h1><p>%s</p></div></body></html>`, msg)
}
