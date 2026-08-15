package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/auth"
	"github.com/gaea/gaea/internal/config"
	gaeacfg "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/netclient"
)

// Client xAI API 客户端，封装认证和 HTTP 通信
type Client struct {
	// mu 保护以下可变状态：activeEngineID、imageBackend/imageBackendType、token。
	// 读写必须经锁；持锁期间不得调用可能再取锁的方法（避免死锁）。
	mu sync.RWMutex

	cfg        *config.Config
	httpClient *http.Client
	tokenStore *auth.TokenStore
	token      *auth.Token

	// 本地图片生成后端（nil 时使用 xAI）
	imageBackend     ImageBackend
	imageBackendType string // "xai" | "comfyui" | "herdsman" | "ollama"

	// 多引擎支持
	engineMgr      *modelengine.Manager
	activeEngineID string // 当前活跃引擎 ID（空字符串="xai"）

	// OnEvent 可选回调：每次 API 调用前后触发，用于前端实时日志
	OnEvent func(eventType string, data map[string]interface{})

	// 并发控制信号量（限制同时进行的 API 调用数）
	sem chan struct{}

	// 非流式 Chat 请求重试退避序列（连接错误与 5xx；默认 2 次，1s/2s，
	// 复用流式 defaultStreamRetryBackoff 语义）。测试可覆盖为短间隔。
	chatRetryBackoff []time.Duration
	// 流式请求重试退避序列（连接错误与 5xx；默认 2 次，1s/2s）。测试可覆盖为短间隔。
	streamRetryBackoff []time.Duration
	// 流空闲超时：连接/首字节后超过该时长无任何数据视为失败（默认 60s）。测试可覆盖为短时长。
	streamIdleTimeout time.Duration
	// proxySpecOverride 代理配置覆盖（测试注入）；nil 时读取 gaea 生效配置。
	proxySpecOverride *netclient.ProxySpec
}

// 流式请求内部哨兵错误：需要整体重试（不记录 usage）。
var (
	errStreamRetry401     = errors.New("stream: retry after token refresh")
	errStreamDegradeUsage = errors.New("stream: retry without include_usage")
)

// errStreamIdleTimeout 流空闲超时哨兵错误（由 idleTimeoutBody 产生）。
var errStreamIdleTimeout = errors.New("stream idle timeout")

// defaultStreamRetryBackoff 连接错误与 5xx 的指数退避：1s / 2s 共 2 次重试。
var defaultStreamRetryBackoff = []time.Duration{time.Second, 2 * time.Second}

// defaultStreamIdleTimeout 流空闲超时默认值。
const defaultStreamIdleTimeout = 60 * time.Second

// chatBackoffOrDefault 返回非流式 Chat 生效的退避序列：未注入时复用流式共享默认值
// defaultStreamRetryBackoff（连接错误与 5xx 指数退避：1s/2s 共 2 次重试，语义与
// 流式一致）。测试可通过 chatRetryBackoff 字段注入短间隔。
func (c *Client) chatBackoffOrDefault() []time.Duration {
	if len(c.chatRetryBackoff) > 0 {
		return c.chatRetryBackoff
	}
	return defaultStreamRetryBackoff
}

// NewClient 创建 AI 客户端
func NewClient(cfg *config.Config) *Client {
	const maxConcurrency = 4 // SuperGrok 并发限制
	c := &Client{
		cfg:                cfg,
		tokenStore:         auth.NewTokenStore(cfg.TokenStorePath),
		sem:                make(chan struct{}, maxConcurrency),
		chatRetryBackoff:   append([]time.Duration(nil), defaultStreamRetryBackoff...),
		streamRetryBackoff: append([]time.Duration(nil), defaultStreamRetryBackoff...),
		streamIdleTimeout:  defaultStreamIdleTimeout,
	}
	c.httpClient = c.buildHTTPClient()
	return c
}

// currentProxySpec 返回当前生效的代理配置：优先测试注入的覆盖值，否则读取
// gaea 生效配置（与办公工具链 web_fetch/web_search 同一来源，保持一致）。
// 读取失败时回退空 spec（等效直连+环境代理），不阻断客户端构造。
func (c *Client) currentProxySpec() netclient.ProxySpec {
	if c.proxySpecOverride != nil {
		return *c.proxySpecOverride
	}
	gcfg, err := gaeacfg.Load()
	if err != nil {
		slog.Warn("读取代理配置失败，回退默认", "error", err)
		return netclient.ProxySpec{}
	}
	if gcfg == nil {
		return netclient.ProxySpec{}
	}
	return gcfg.NetworkProxySpec()
}

// proxySpec 返回按应用代理配置解析的 ProxySpec，并强制本地引擎直连：
// localhost / 127.0.0.1 / ::1（回环）一律不走代理（herdsman/ComfyUI/Ollama
// 都是本机服务），云端引擎（xAI/DeepSeek/MiMo 等外部域名）按配置走代理。
func (c *Client) proxySpec() netclient.ProxySpec {
	spec := c.currentProxySpec()
	spec.DirectHosts = append(spec.DirectHosts, "localhost", "127.0.0.1", "::1")
	return spec
}

// buildHTTPClient 按应用代理配置构建 AI/引擎流量使用的 HTTP 客户端，
// 保留默认传输的超时与连接池行为。代理配置无效时回退直连客户端。
func (c *Client) buildHTTPClient() *http.Client {
	spec := c.proxySpec()
	// 响应头超时兜底「连接 + 首字节等待」：代理或远端黑洞时避免无限挂起
	// （与流空闲超时同值；流开始后由 idleTimeoutBody 按空闲计）。
	cli, err := netclient.NewHTTPClient(spec, netclient.TransportOptions{
		ResponseHeaderTimeout: c.idleTimeout(),
	})
	if err != nil {
		slog.Warn("代理配置无效，AI 客户端回退直连", "error", err)
		return netclient.NewSimpleClient(0)
	}
	return cli
}

// idleTimeout 返回流空闲超时时长（测试可覆盖，默认 60s）。
func (c *Client) idleTimeout() time.Duration {
	if c.streamIdleTimeout > 0 {
		return c.streamIdleTimeout
	}
	return defaultStreamIdleTimeout
}

// SetEngineManager 设置模型引擎管理器（用于多引擎路由）
func (c *Client) SetEngineManager(mgr *modelengine.Manager) {
	c.engineMgr = mgr
}

// SetActiveEngine 设置当前活跃引擎（空字符串="xai"）
func (c *Client) SetActiveEngine(engineID string) {
	if engineID == "" {
		engineID = "xai"
	}
	c.mu.Lock()
	c.activeEngineID = engineID
	c.mu.Unlock()
	slog.Info("切换活跃引擎", "engine", engineID)
}

// ActiveEngineID 返回当前活跃引擎 ID
func (c *Client) ActiveEngineID() string {
	c.mu.RLock()
	id := c.activeEngineID
	c.mu.RUnlock()
	if id == "" {
		return "xai"
	}
	return id
}

// resolveChatEndpoint 根据活跃引擎解析 chat completions 的 URL 和 API key。
// 当活跃引擎是 xAI 时使用 OAuth token；其他引擎使用 engineManager。
func (c *Client) resolveChatEndpoint(engineID string) (endpoint string, apiKey string, err error) {
	if engineID == "" {
		engineID = c.ActiveEngineID()
	}

	if engineID != "xai" && c.engineMgr != nil {
		return c.engineMgr.BuildChatURL(engineID)
	}

	// xAI：使用 OAuth token
	token, err := c.GetToken()
	if err != nil {
		return "", "", err
	}
	endpoint = strings.TrimSuffix(c.cfg.XaiAPIBaseURL, "/") + "/chat/completions"
	return endpoint, token, nil
}

// resolveModelName 根据活跃引擎解析模型名称。
// 规则：
//   - xAI 引擎: 优先用请求指定的模型，否则用 cfg.Model
//   - 其他引擎: 优先用请求指定的模型（除非是 xAI 默认模型名），否则用引擎默认模型
func (c *Client) resolveModelName(reqModel string, engineID string) string {
	if engineID == "" {
		engineID = c.ActiveEngineID()
	}

	// xAI 引擎：保持原有逻辑
	if engineID == "xai" || c.engineMgr == nil {
		if reqModel != "" {
			return reqModel
		}
		return c.cfg.Model
	}

	// 非 xAI 引擎：获取引擎默认模型
	engineDefault := ""
	if def, err := c.engineMgr.GetDefaultModel(engineID); err == nil {
		engineDefault = def
	}

	// 如果请求的模型为空，或者是 xAI 的默认模型名（来自 Router 的硬编码），则替换
	if reqModel == "" || reqModel == c.cfg.Model {
		if engineDefault != "" {
			return engineDefault
		}
		// 引擎也没有默认模型，回退到 cfg.Model（会因 API 报错而提醒用户配置）
		return c.cfg.Model
	}

	// 用户显式指定了非 xAI 默认的模型名，保留
	return reqModel
}

// GetToken 获取有效 token（自动刷新）。内部加锁保证并发安全：
// 快速路径（token 未过期）只取读锁；刷新路径持写锁并在锁内二次判空
// （single-flight）——等锁期间其他调用方已完成刷新时直接复用，避免并发重复刷新。
func (c *Client) GetToken() (string, error) {
	c.mu.RLock()
	if c.token != nil && !c.token.IsExpired() {
		tok := c.token.AccessToken
		c.mu.RUnlock()
		return tok, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	// 锁内二次判空：等锁期间可能已有调用方完成刷新
	if c.token != nil && !c.token.IsExpired() {
		return c.token.AccessToken, nil
	}

	stored, err := c.tokenStore.Load()
	if err != nil {
		slog.Warn("token 加载失败", "error", err)
	}
	if stored != nil {
		c.token = stored
		if !stored.IsExpired() {
			return stored.AccessToken, nil
		}
		if stored.RefreshToken != "" {
			newToken, err := auth.RefreshAccessToken(c.cfg, stored.RefreshToken)
			if err != nil {
				return "", fmt.Errorf("token 刷新失败，请重新登录: %w", err)
			}
			c.token = newToken
			if err := c.tokenStore.Save(newToken); err != nil {
				slog.Error("保存刷新后的 token 失败", "error", err)
			}
			return newToken.AccessToken, nil
		}
	}
	return "", fmt.Errorf("登录已过期，请在右上角重新登录（当前 token 无法刷新）")
}

// EnsureToken 确保已登录
func (c *Client) EnsureToken() error {
	_, err := c.GetToken()
	if err == nil {
		return nil
	}
	// GetToken 失败，尝试直接从文件加载
	store := auth.NewTokenStore(c.cfg.TokenStorePath)
	tok, loadErr := store.Load()
	if loadErr != nil {
		slog.Warn("EnsureToken: token 加载失败", "error", loadErr)
	}
	if tok != nil && !tok.IsExpired() {
		c.mu.Lock()
		c.token = tok
		c.mu.Unlock()
		return nil
	}
	// token 不存在或已过期，返回原始错误
	if tok == nil {
		return fmt.Errorf("未登录：token 文件不存在")
	}
	return err
}

// tryRefreshToken 尝试刷新 token 并保存，成功返回 nil。
// 内部加锁保护 c.token 读写（刷新期间阻塞其他 token 访问，避免并发重复刷新）。
func (c *Client) tryRefreshToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == nil || c.token.RefreshToken == "" {
		return fmt.Errorf("无 refresh token")
	}
	newToken, err := auth.RefreshAccessToken(c.cfg, c.token.RefreshToken)
	if err != nil {
		return fmt.Errorf("token 刷新失败: %w", err)
	}
	c.token = newToken
	if err := c.tokenStore.Save(newToken); err != nil {
		slog.Error("保存刷新后的 token 失败", "error", err)
	}
	return nil
}

// ── Chat Completions（非流式）─────────────────────────────────

// Chat 非流式对话。
//
// 请求建立阶段失败的重试语义与流式一致：
//   - 连接错误（httpClient.Do 返回 err）与 5xx 响应按指数退避重试（默认 2 次
//     1s/2s，复用 defaultStreamRetryBackoff，测试可通过 chatRetryBackoff 注入）；
//   - 收到 200 后不再重试（非流式天然单次响应）；
//   - 仅 xAI 引擎对 401 刷新 token 后在同一函数内重发一次（不递归调用 Chat，
//     避免外层 defer releaseSem 未执行导致信号量占双槽）；
//   - 其余非 200 状态照旧直接返回错误。
//
// 重试期间保持信号量占用；usage 统计只在最终成功/失败时记录一次（与流式一致，
// 避免重试成功时重复累计失败）。
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := c.acquireSem(ctx); err != nil {
		return nil, fmt.Errorf("等待 API 槽位: %w", err)
	}
	defer c.releaseSem()

	start := time.Now()
	reqEngine := req.EngineID
	if reqEngine == "" {
		reqEngine = c.ActiveEngineID()
	}
	reqModel := req.Model
	if reqModel == "" {
		reqModel = c.resolveModelName("", req.EngineID)
	}

	endpoint, apiKey, err := c.resolveChatEndpoint(req.EngineID)
	if err != nil {
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// 如果请求中模型名为空，自动填充引擎默认模型
	if req.Model == "" {
		req.Model = c.resolveModelName("", req.EngineID)
	}

	body, err := json.Marshal(req)
	if err != nil {
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	backoff := c.chatBackoffOrDefault()
	var lastErr error
	refreshed := false // 401 刷新只做一次
	for attempt := 0; attempt <= len(backoff); attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoff[attempt-1]):
			case <-ctx.Done():
				errMsg := fmt.Sprintf("请求已取消: %v", ctx.Err())
				c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, errMsg)
				return nil, fmt.Errorf("%s", errMsg)
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
			return nil, fmt.Errorf("构造请求失败: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			// 连接建立失败：ctx 取消不重试，其余退避重试
			if ctx.Err() != nil {
				c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
				return nil, fmt.Errorf("API 请求失败: %w", err)
			}
			lastErr = fmt.Errorf("API 请求失败: %w", err)
			slog.Warn("Chat 请求失败，准备退避重试", "attempt", attempt+1, "error", lastErr)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, readErr.Error())
			return nil, fmt.Errorf("read response body: %w", readErr)
		}

		// 仅 xAI 引擎做 401 token 刷新重试：刷新后立即在同一函数内重发一次
		//（不递归调用 Chat，避免外层 defer releaseSem 未执行而占 2 个信号量槽）。
		if resp.StatusCode == 401 && reqEngine == "xai" && !refreshed {
			if err := c.tryRefreshToken(); err != nil {
				c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
				return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
			}
			refreshed = true
			newKey, err := c.GetToken()
			if err != nil {
				c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
				return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
			}
			apiKey = newKey
			attempt = -1 // 立即重发（attempt 自增回 0，不走退避等待）
			continue
		}

		// 5xx：服务端故障，响应未开始，退避重试
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(respBody), 500))
			slog.Warn("Chat 请求 5xx，准备退避重试", "attempt", attempt+1, "error", lastErr)
			continue
		}

		if resp.StatusCode != 200 {
			errMsg := fmt.Sprintf("API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(respBody), 500))
			c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}

		var chatResp ChatResponse
		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		if chatResp.Error != nil {
			errMsg := fmt.Sprintf("[%s] %s", chatResp.Error.Code, chatResp.Error.Message)
			c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}
		var inTok, outTok, cacheHit, cacheMiss int64
		if chatResp.Usage != nil {
			inTok, outTok = chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens
			cacheHit, cacheMiss = cacheSplitForUsage(chatResp.Usage)
		}
		c.recordUsage(reqEngine, reqModel, start, inTok, outTok, cacheHit, cacheMiss, true, "")
		return &chatResp, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("API 请求失败（重试耗尽）")
	}
	c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, lastErr.Error())
	return nil, lastErr
}

// ── SSE 流式 ──────────────────────────────────────────────────

// ChatStream 流式对话，通过 channel 返回 SSEChunk
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (<-chan SSEChunk, error) {
	// 并发控制
	if err := c.acquireSem(ctx); err != nil {
		return nil, fmt.Errorf("等待 API 槽位: %w", err)
	}

	start := time.Now()
	reqEngine := req.EngineID
	if reqEngine == "" {
		reqEngine = c.ActiveEngineID()
	}
	reqModel := req.Model
	if reqModel == "" {
		reqModel = c.resolveModelName("", req.EngineID)
	}

	endpoint, apiKey, err := c.resolveChatEndpoint(req.EngineID)
	if err != nil {
		c.releaseSem()
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// 如果请求中模型名为空，自动填充引擎默认模型
	if req.Model == "" {
		req.Model = c.resolveModelName("", req.EngineID)
	}

	req.Stream = true
	// 流式接口默认不返回 usage；显式请求 include_usage 以便统计 Token。
	// 部分服务端不支持该字段（400），会在下方去掉后重试一次。
	if !req.skipIncludeUsage && req.StreamOptions == nil {
		req.StreamOptions = &ChatStreamOptions{IncludeUsage: true}
	}
	body, err := json.Marshal(req)
	if err != nil {
		c.releaseSem()
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, fmt.Errorf("marshal stream request: %w", err)
	}

	// 流上下文：HTTP 请求与流空闲超时共用。空闲超时触发时取消该上下文以解除
	// 阻塞中的响应体读取（若请求使用调用方 ctx，取消流上下文将无法中断读取）。
	streamCtx, cancel := context.WithCancel(ctx)

	resp, err := c.doStreamRequest(streamCtx, endpoint, apiKey, body, reqEngine, req)
	switch {
	case errors.Is(err, errStreamRetry401):
		cancel()
		c.releaseSem()
		return c.ChatStream(ctx, req)
	case errors.Is(err, errStreamDegradeUsage):
		cancel()
		c.releaseSem()
		req.skipIncludeUsage = true
		req.StreamOptions = nil
		return c.ChatStream(ctx, req)
	case err != nil:
		cancel()
		c.releaseSem()
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// 流空闲超时：连接/首字节后超过 idleTimeout 无任何数据视为失败并返回错误。
	// 按空闲计而非总时长，慢速但持续输出的流不受影响。
	resp.Body = newIdleTimeoutBody(resp.Body, c.idleTimeout(), cancel)

	// 解析协程使用调用方 ctx（而非 streamCtx）：send 的取消守卫语义是
	// 「调用方取消时退出」；空闲超时取消的是 streamCtx（解除阻塞读取），
	// 若把 streamCtx 传入，超时后 send 的 ctx.Done 分支就绪会随机抢占，
	// 导致超时错误分块被丢弃。
	chunks := make(chan SSEChunk, 64)
	go c.parseStreamEvents(ctx, resp, chunks, reqEngine, reqModel, start)
	return chunks, nil
}

// doStreamRequest 发送流式请求并处理请求建立阶段的失败：
//   - 连接错误与 5xx 响应按指数退避重试（默认 2 次，1s/2s）；
//   - 401 刷新 token 后返回 errStreamRetry401，由调用方整体重试；
//   - 400 且服务端不支持 include_usage 时返回 errStreamDegradeUsage，由调用方降级重试；
//   - 收到 200 后直接返回响应（流已开始，不再重试，避免重复生成）。
//
// 重试期间保持信号量占用，不做 usage 统计（最终成功/失败由调用方统一记录）。
func (c *Client) doStreamRequest(ctx context.Context, endpoint, apiKey string, body []byte, reqEngine string, req *ChatRequest) (*http.Response, error) {
	backoff := c.streamRetryBackoff
	if len(backoff) == 0 {
		backoff = defaultStreamRetryBackoff
	}
	var lastErr error
	for attempt := 0; attempt <= len(backoff); attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoff[attempt-1]):
			case <-ctx.Done():
				return nil, fmt.Errorf("流式请求已取消: %w", ctx.Err())
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("构造流式请求失败: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		}
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			// 连接建立失败：ctx 取消不重试，其余退避重试
			if ctx.Err() != nil {
				return nil, fmt.Errorf("流式请求失败: %w", err)
			}
			lastErr = fmt.Errorf("流式请求失败: %w", err)
			slog.Warn("流式请求失败，准备退避重试", "attempt", attempt+1, "error", lastErr)
			continue
		}

		// 仅 xAI 引擎做 401 token 刷新重试
		if resp.StatusCode == 401 && reqEngine == "xai" {
			resp.Body.Close()
			if err := c.tryRefreshToken(); err != nil {
				return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
			}
			return nil, errStreamRetry401
		}

		// 5xx：服务端故障，流尚未开始，退避重试
		if resp.StatusCode >= 500 {
			respBody, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			msg := fmt.Sprintf("API 错误 (HTTP %d)", resp.StatusCode)
			if readErr == nil {
				msg += ": " + trimStr(string(respBody), 500)
			}
			lastErr = fmt.Errorf("%s", msg)
			slog.Warn("流式请求 5xx，准备退避重试", "attempt", attempt+1, "error", lastErr)
			continue
		}

		if resp.StatusCode != 200 {
			respBody, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			// 服务端不支持 stream_options.include_usage 时去掉该字段重试一次。
			if resp.StatusCode == 400 && req.StreamOptions != nil && readErr == nil &&
				(bytes.Contains(respBody, []byte("stream_options")) || bytes.Contains(respBody, []byte("include_usage"))) {
				return nil, errStreamDegradeUsage
			}
			errMsg := ""
			if readErr != nil {
				errMsg = fmt.Sprintf("API 错误 (HTTP %d): <无法读取响应体>", resp.StatusCode)
			} else {
				errMsg = fmt.Sprintf("API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(respBody), 500))
			}
			return nil, fmt.Errorf("%s", errMsg)
		}

		return resp, nil // 200：流已开始，不再重试
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("流式请求失败（重试耗尽）")
	}
	return nil, lastErr
}

// parseStreamEvents 解析 SSE 事件流并发送到 chunks channel
func (c *Client) parseStreamEvents(ctx context.Context, resp *http.Response, chunks chan SSEChunk, reqEngine, reqModel string, start time.Time) {
	defer resp.Body.Close()
	defer close(chunks)
	defer c.releaseSem()

	// send 在 ctx 取消或通道关闭时安全退出，避免消费者提前返回后永久阻塞泄漏
	send := func(ch SSEChunk) bool {
		select {
		case chunks <- ch:
			return true
		case <-ctx.Done():
			return false
		}
	}

	// 工具调用分片按 index 拼装
	toolPending := make(map[int]*ChatToolCall)
	var toolOrder []int
	var streamUsage *ChatUsage // 流结束块携带的用量（OpenAI 兼容 API 在最后一块带 usage）
	var streamOK bool          // 是否正常收到结束帧

	defer func() {
		if c.engineMgr == nil {
			return
		}
		var inTok, outTok, cacheHit, cacheMiss int64
		if streamUsage != nil {
			inTok = streamUsage.PromptTokens
			outTok = streamUsage.CompletionTokens
			cacheHit, cacheMiss = cacheSplitForUsage(streamUsage)
		}
		c.engineMgr.RecordCall(modelengine.ModelCallUsage{
			EngineID:        reqEngine,
			Model:           reqModel,
			InputTokens:     inTok,
			OutputTokens:    outTok,
			CacheHitTokens:  cacheHit,
			CacheMissTokens: cacheMiss,
			DurationMs:      time.Since(start).Milliseconds(),
			Success:         streamOK,
			FinishedAt:      time.Now().Format("2006-01-02 15:04:05"),
		})
	}()

	// 用 bufio.Reader 逐行读取：Scanner 默认 64KB 行上限，超长行（大工具参数/
	// 长推理内容）会触发 ErrTooLong 断流；ReadString 对超长行自动拼接，不丢数据。
	reader := bufio.NewReader(resp.Body)

	for {
		select {
		case <-ctx.Done():
			// 区分空闲超时（读取阻塞中由定时器触发）与调用方主动取消
			if tb, ok := resp.Body.(*idleTimeoutBody); ok && tb.isTimedOut() {
				if !send(SSEChunk{Error: fmt.Sprintf("流空闲超时：超过 %s 无数据", c.idleTimeout())}) {
					return
				}
				return
			}
			if !send(SSEChunk{Error: "请求已取消"}) {
				return
			}
			return
		default:
		}

		line, err := readSSELine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // 服务端正常结束
			}
			if errors.Is(err, errStreamIdleTimeout) {
				if !send(SSEChunk{Error: fmt.Sprintf("流空闲超时：超过 %s 无数据", c.idleTimeout())}) {
					return
				}
				return
			}
			errMsg := fmt.Sprintf("流读取错误: %v", err)
			if !send(SSEChunk{Error: errMsg}) {
				return
			}
			return
		}
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			if len(toolOrder) > 0 {
				send(SSEChunk{Done: true, ToolCalls: flushToolCalls(toolPending, toolOrder), Usage: streamUsage})
			} else {
				send(SSEChunk{Done: true, Usage: streamUsage})
			}
			streamOK = true
			return
		}

		var choice struct {
			Choices []ChatChoice `json:"choices"`
			Usage   *ChatUsage   `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &choice); err != nil {
			continue
		}
		if choice.Usage != nil {
			streamUsage = choice.Usage
		}

		if len(choice.Choices) > 0 {
			delta := choice.Choices[0].Delta
			if delta.Content != "" {
				send(SSEChunk{Content: delta.Content})
			}
			if delta.ReasoningContent != "" {
				send(SSEChunk{Reasoning: delta.ReasoningContent})
			}
			for _, tc := range delta.ToolCalls {
				p, ok := toolPending[tc.Index]
				if !ok {
					p = &ChatToolCall{ID: tc.ID, Type: tc.Type}
					toolPending[tc.Index] = p
					toolOrder = append(toolOrder, tc.Index)
				}
				if tc.Function.Name != "" {
					p.Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					p.Function.Arguments += tc.Function.Arguments
				}
			}
			if delta.FinishReason == "tool_calls" {
				if !send(SSEChunk{Done: true, ToolCalls: flushToolCalls(toolPending, toolOrder), Usage: streamUsage}) {
					return
				}
				streamOK = true
				return
			}
			if delta.FinishReason != "" {
				if !send(SSEChunk{Done: true, Usage: streamUsage}) {
					return
				}
				streamOK = true
				return
			}
		}
	}

	if len(toolOrder) > 0 {
		send(SSEChunk{Done: true, ToolCalls: flushToolCalls(toolPending, toolOrder), Usage: streamUsage})
	} else {
		send(SSEChunk{Done: true, Usage: streamUsage})
	}
	streamOK = true
}

// readSSELine 用 bufio.Reader 读取一行 SSE 数据：支持任意长度行（ReadString
// 自动拼接超长行）、\n 与 \r\n 结尾，去掉行尾换行符。数据结束（EOF 且无剩余
// 内容）返回 io.EOF。
func readSSELine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && line == "" {
		return "", io.EOF
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// idleTimeoutBody 包装流式响应体：每次成功读取都会重置空闲计时器，超过 idle
// 时长无任何数据时取消请求上下文（解除阻塞中的 Read）并返回 errStreamIdleTimeout。
// 慢速但持续输出的流不受影响。Close 停止计时器并释放请求上下文。
type idleTimeoutBody struct {
	body     io.ReadCloser
	idle     time.Duration
	cancel   context.CancelFunc
	mu       sync.Mutex
	timer    *time.Timer
	timedOut bool
}

func newIdleTimeoutBody(body io.ReadCloser, idle time.Duration, cancel context.CancelFunc) *idleTimeoutBody {
	return &idleTimeoutBody{body: body, idle: idle, cancel: cancel}
}

func (b *idleTimeoutBody) Read(p []byte) (int, error) {
	b.mu.Lock()
	if !b.timedOut {
		if b.timer == nil {
			b.timer = time.AfterFunc(b.idle, b.onTimeout)
		} else {
			b.timer.Reset(b.idle)
		}
	}
	b.mu.Unlock()

	n, err := b.body.Read(p)

	b.mu.Lock()
	timedOut := b.timedOut
	if !timedOut {
		b.timer.Stop()
	}
	b.mu.Unlock()
	if timedOut && err != nil {
		return 0, errStreamIdleTimeout
	}
	return n, err
}

func (b *idleTimeoutBody) onTimeout() {
	b.mu.Lock()
	b.timedOut = true
	cancel := b.cancel
	b.mu.Unlock()
	if cancel != nil {
		cancel() // 解除阻塞中的 Read（请求上下文取消会中断响应体读取）
	}
}

// isTimedOut 返回空闲超时是否已触发（供解析循环区分超时与调用方取消）。
func (b *idleTimeoutBody) isTimedOut() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.timedOut
}

func (b *idleTimeoutBody) Close() error {
	b.mu.Lock()
	if b.timer != nil {
		b.timer.Stop()
	}
	cancel := b.cancel
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return b.body.Close()
}

// flushToolCalls 按出现顺序输出拼装完成的工具调用列表。
func flushToolCalls(pending map[int]*ChatToolCall, order []int) []ChatToolCall {
	out := make([]ChatToolCall, 0, len(order))
	for _, idx := range order {
		if p := pending[idx]; p != nil {
			out = append(out, *p)
		}
	}
	return out
}

// recordUsage 上报一次模型调用统计（非流式路径 / 流式前置失败路径）。
func (c *Client) recordUsage(engineID, model string, start time.Time, inTok, outTok, cacheHit, cacheMiss int64, success bool, errMsg string) {
	if c.engineMgr == nil {
		return
	}
	c.engineMgr.RecordCall(modelengine.ModelCallUsage{
		EngineID:        engineID,
		Model:           model,
		InputTokens:     inTok,
		OutputTokens:    outTok,
		CacheHitTokens:  cacheHit,
		CacheMissTokens: cacheMiss,
		DurationMs:      time.Since(start).Milliseconds(),
		Success:         success,
		ErrorMessage:    errMsg,
		FinishedAt:      time.Now().Format("2006-01-02 15:04:05"),
	})
}

// cacheSplitForUsage 从 ChatUsage 提取 KV 缓存命中/未命中 token 数，归一两种形状：
// DeepSeek 顶层 prompt_cache_{hit,miss}_tokens 与 OpenAI/MiMo
// prompt_tokens_details.cached_tokens。命中取 CacheHitTokens()（两者兼容）；
// 未命中取 CacheMissTokens()（优先服务端显式上报，否则按 prompt - 命中推算，
// 下限 0）。防污染：服务端完全未上报缓存拆分（命中/未命中/详情都为空）时
// 返回 0/0——此时 CacheMissTokens() 会把全部 prompt 推成未命中，把未知情况
// 算作 100% 未命中会拉低缓存命中率，故归零。
func cacheSplitForUsage(u *ChatUsage) (hit, miss int64) {
	if u == nil {
		return 0, 0
	}
	hit = u.CacheHitTokens()
	miss = u.CacheMissTokens()
	if hit == 0 && u.PromptCacheMissTokens == 0 && u.PromptTokensDetails == nil {
		return 0, 0
	}
	return hit, miss
}

// ChatSimpleStream 简化流式对话，收集完整回复（5 分钟超时）。

// ChatSimpleStream 简化流式对话，收集完整回复（5 分钟超时）。
func (c *Client) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	return c.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userMsg, ChatSimpleOptions{})
}

// ChatSimpleStreamWithOptions 简化流式对话，支持参数覆盖。
func (c *Client) ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ChatSimpleOptions) (string, error) {
	content, _, err := c.ChatSimpleStreamDetailed(ctx, model, systemPrompt, userMsg, opts)
	return content, err
}

// ChatSimpleStreamDetailed 与 ChatSimpleStreamWithOptions 相同，但额外返回思考链。
// opts.EnableThinking 对本地 Qwen3 系模型（herdsman/ollama）开启思考模式，
// 服务端会流式下发 reasoning_content，函数累计后随正文一起返回。
func (c *Client) ChatSimpleStreamDetailed(ctx context.Context, model, systemPrompt, userMsg string, opts ChatSimpleOptions) (string, string, error) {
	chunks, cancel, err := c.ChatStreamChunks(ctx, model, systemPrompt, userMsg, opts)
	if err != nil {
		return "", "", err
	}
	defer cancel()

	var sb strings.Builder
	var rb strings.Builder
loop:
	for {
		select {
		case <-ctx.Done():
			return sb.String(), rb.String(), fmt.Errorf("请求超时或已取消")
		case chunk, ok := <-chunks:
			if !ok {
				break loop
			}
			if chunk.Error != "" {
				return sb.String(), rb.String(), fmt.Errorf("%s", chunk.Error)
			}
			if chunk.Done {
				break loop
			}
			sb.WriteString(chunk.Content)
			rb.WriteString(chunk.Reasoning)
		}
	}
	result := sb.String()
	reasoning := rb.String()
	c.emit("response", map[string]interface{}{
		"length":    len([]rune(result)),
		"content":   result,
		"reasoning": reasoning,
	})
	return result, reasoning, nil
}

// ChatStreamChunks 与 ChatSimpleStreamDetailed 相同的请求准备，但不消费底层 SSE
// 通道，而是把它原样返回给调用方逐块消费（用于聊天板块真实流式下发）。
// 调用方负责在消费完成后调用返回的 cancel（同时关闭超时定时器）。
func (c *Client) ChatStreamChunks(ctx context.Context, model, systemPrompt, userMsg string, opts ChatSimpleOptions) (<-chan SSEChunk, context.CancelFunc, error) {
	timeoutMinutes := opts.TimeoutMinutes
	if timeoutMinutes <= 0 {
		timeoutMinutes = 5
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)

	// 如果 model 为空，用引擎默认模型（功能级引擎优先）
	if model == "" {
		model = c.resolveModelName("", opts.EngineID)
	}
	reqEngine := opts.EngineID
	if reqEngine == "" {
		reqEngine = c.ActiveEngineID()
	}

	c.emit("request", map[string]interface{}{
		"model":     model,
		"system":    systemPrompt,
		"user":      userMsg,
		"reasoning": opts.ReasoningEffort,
		"engine":    reqEngine,
	})

	temperature := opts.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	req := &ChatRequest{
		Model:           model,
		EngineID:        opts.EngineID,
		Messages:        []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userMsg}},
		MaxTokens:       maxTokens,
		Temperature:     temperature,
		ReasoningEffort: opts.ReasoningEffort,
	}
	if opts.TopP > 0 {
		req.TopP = opts.TopP
	}
	if opts.EnableThinking {
		reqEngine := opts.EngineID
		if reqEngine == "" {
			reqEngine = c.ActiveEngineID()
		}
		if reqEngine == "herdsman" || reqEngine == "ollama" {
			t := true
			req.EnableThinking = &t
			req.ChatTemplateKwargs = map[string]any{"enable_thinking": true}
			// 测评结论：思考模式与正文共享 max_tokens，<4096 会出现
			// 「只有推理、无正文」（herdsman 模型测评报告 §8.1/§9）。
			// 守护：显式小预算抬到 4096。
			if req.MaxTokens > 0 && req.MaxTokens < 4096 {
				req.MaxTokens = 4096
			}
		}
	}

	chunks, err := c.ChatStream(ctx, req)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return chunks, cancel, nil
}

// SetImageBackend 设置图片生成后端（nil + backendType 回退到 xAI）
func (c *Client) SetImageBackend(backend ImageBackend, backendType string) {
	if backendType == "" {
		backendType = "xai"
	}
	c.mu.Lock()
	c.imageBackend = backend
	c.imageBackendType = backendType
	c.mu.Unlock()
}

// GetImageBackendType 获取当前图片后端类型
func (c *Client) GetImageBackendType() string {
	c.mu.RLock()
	t := c.imageBackendType
	c.mu.RUnlock()
	if t == "" {
		return "xai"
	}
	return t
}

// GetImageBackend 返回当前图片后端实例（可能为 nil）。
// 供取消中断等场景类型断言使用（如 *ComfyUIBackend 的 Interrupt/ResetCancel）。
func (c *Client) GetImageBackend() ImageBackend {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	b := c.imageBackend
	c.mu.RUnlock()
	return b
}

// ── Models ────────────────────────────────────────────────────
// ListModels 获取可用模型列表（始终从 xAI 获取，本地引擎通过 engineManager）
func (c *Client) ListModels(ctx context.Context) (*ModelsResponse, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimSuffix(c.cfg.XaiAPIBaseURL, "/") + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("models request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read models response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, trimStr(string(body), 500))
	}

	var models ModelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, fmt.Errorf("parse models response: %w", err)
	}
	return &models, nil
}

// ── 图片生成 ──────────────────────────────────────────────────

func (c *Client) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	c.mu.RLock()
	backend := c.imageBackend
	c.mu.RUnlock()
	if backend != nil {
		return backend.GenerateImage(ctx, req)
	}
	return c.generateImageXAI(ctx, req)
}

// generateImageXAI xAI 原生图片生成
func (c *Client) generateImageXAI(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	// 规范化模型名：如果传入的是 ComfyUI 模型名，替换为 xAI 默认模型
	if req.Model == "" || req.Model == "flux" || req.Model == "z-image-turbo" {
		req.Model = "grok-imagine-image"
	}

	// xAI API 只接受 model/prompt/n/response_format，清空不兼容字段
	req.Size = ""
	req.Negative = ""
	req.Seed = 0

	endpoint := strings.TrimSuffix(c.cfg.XaiAPIBaseURL, "/") + "/images/generations"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal image request: %w", err)
	}
	slog.Info("xAI图片请求", "body", string(body))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造图片请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("图片 API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read image response: %w", err)
	}

	if resp.StatusCode == 401 {
		if err := c.tryRefreshToken(); err != nil {
			return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
		}
		return c.generateImageXAI(ctx, req)
	}
	if resp.StatusCode != 200 {
		slog.Error("xAI图片生成失败", "status", resp.StatusCode, "body", trimStr(string(respBody), 500))
		// 解析错误详情，提供友好提示
		var errResp struct {
			Code  string `json:"code"`
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Code == "imagine:content-moderated" {
			return nil, fmt.Errorf("xAI 内容审核拦截，请修改提示词后重试")
		}
		return nil, fmt.Errorf("图片 API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(respBody), 500))
	}

	var imgResp ImageGenerationResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		slog.Error("解析xAI图片响应失败", "body", trimStr(string(respBody), 300), "error", err)
		return nil, fmt.Errorf("解析图片响应失败: %w", err)
	}
	return &imgResp, nil
}

// ── 工具函数 ──────────────────────────────────────────────────

func trimStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (c *Client) emit(eventType string, data map[string]interface{}) {
	if c.OnEvent != nil {
		c.OnEvent(eventType, data)
	}
}

func (c *Client) acquireSem(ctx context.Context) error {
	select {
	case c.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) releaseSem() {
	<-c.sem
}
