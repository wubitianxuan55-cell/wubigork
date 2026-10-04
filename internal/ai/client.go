package ai

import (
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
	imageBackendType string // 取值域 ImageBackendType*（含 "glm"）；与注册表 kind 两套口径，对照表见 image_backend.go

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

	// engineFailoverFn 引擎故障转移开关读取函数（C 刀 v0；app 启动/重建 client
	// 后注入 cfg.GetEngineFailover，照 engineMgr 注入先例——避免逐请求读
	// config）。nil 或返回 false = 关闭（现状，请求路径零改动）。
	engineFailoverFn func() bool

	// OnFailover 故障转移发生回调（C 刀 v0；app 层接线 emit "model-failover"，
	// 同 modelengine.SetHealthNotifyFunc 注入模式；nil=忽略）。
	OnFailover func(fromEngine, toEngine, model string)
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

// SetEngineFailoverFunc 注入引擎故障转移开关读取函数（C 刀 v0；app 在
// configureClient 接线，client 重建后随 OnEvent 一起恢复）。nil=永久关闭。
func (c *Client) SetEngineFailoverFunc(fn func() bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.engineFailoverFn = fn
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

	// cfg.Model 与 UI 写路径（切引擎/设默认模型）共享实例，读经 config.GetModelMem
	// 持锁（写点 SetModelMem 同锁），消除「请求热路径 × UI 写路径」裸字段竞态。
	fallbackModel := config.GetModelMem(c.cfg)

	// xAI 引擎：保持原有逻辑
	if engineID == "xai" || c.engineMgr == nil {
		if reqModel != "" {
			return reqModel
		}
		return fallbackModel
	}

	// 非 xAI 引擎：获取引擎默认模型
	engineDefault := ""
	if def, err := c.engineMgr.GetDefaultModel(engineID); err == nil {
		engineDefault = def
	}

	// 如果请求的模型为空，或者是 xAI 的默认模型名（来自 Router 的硬编码），则替换
	if reqModel == "" || reqModel == fallbackModel {
		if engineDefault != "" {
			return engineDefault
		}
		// 引擎也没有默认模型，回退到 cfg.Model（会因 API 报错而提醒用户配置）
		return fallbackModel
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

// SetImageBackend 设置图片生成后端（nil + backendType 回退到 xAI）。
// backendType 取值域是 ImageBackendType* 运行时类型名（非注册表 kind，
// 两套口径对照表见 image_backend.go）。
func (c *Client) SetImageBackend(backend ImageBackend, backendType string) {
	if backendType == "" {
		backendType = ImageBackendTypeXAI
	}
	c.mu.Lock()
	c.imageBackend = backend
	c.imageBackendType = backendType
	c.mu.Unlock()
}

// GetImageBackendType 获取当前图片后端类型（ImageBackendType* 取值域，
// 空值回退 xAI；经 app GetImageBackend 绑定下发前端与配置恢复）。
func (c *Client) GetImageBackendType() string {
	c.mu.RLock()
	t := c.imageBackendType
	c.mu.RUnlock()
	if t == "" {
		return ImageBackendTypeXAI
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
	return c.generateImageXAI(ctx, req, false)
}

// generateImageXAI xAI 图片生成（3a Image seam）：
// 走注册表 openai 兼容后端（kind = ImageBackendKindOpenAI），保留 xAI 特有参数清理；
// 401 时刷新 token 重试一次（retried 守卫单次重试，避免持续 401 无限递归），
// 内容审核拦截给出友好提示。
func (c *Client) generateImageXAI(ctx context.Context, req *ImageGenerationRequest, retried bool) (*ImageGenerationResponse, error) {
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

	backend, err := NewImageBackend(ImageBackendKindOpenAI, ImageBackendConfig{
		BaseURL: c.cfg.XaiAPIBaseURL,
		APIKey:  token,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("xAI图片请求", "model", req.Model, "n", req.N)

	resp, err := backend.GenerateImage(ctx, req)
	if err != nil {
		msg := err.Error()
		// 内容审核拦截：错误体含 imagine:content-moderated 时给出友好提示
		if strings.Contains(msg, "imagine:content-moderated") {
			return nil, fmt.Errorf("xAI 内容审核拦截，请修改提示词后重试")
		}
		// 401 认证失败：刷新 token 后重试一次（注册表后端以 "HTTP 401" 透传状态码）。
		// retried 守卫：仅当尚未重试过才刷新并递归，持续 401 直接返回错误（不无限递归）。
		if strings.Contains(msg, "HTTP 401") && !retried {
			if rerr := c.tryRefreshToken(); rerr != nil {
				return nil, fmt.Errorf("认证失败 (HTTP 401): %w", rerr)
			}
			return c.generateImageXAI(ctx, req, true)
		}
		return nil, err
	}
	return resp, nil
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
