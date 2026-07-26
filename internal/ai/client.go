package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/auth"
	"github.com/wubigork/wubigork/internal/config"
	"github.com/wubigork/wubigork/internal/modelengine"
	"github.com/wubigork/wubigork/internal/util"
)

// Client xAI API 客户端，封装认证和 HTTP 通信
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
	tokenStore *auth.TokenStore
	token      *auth.Token

	// 本地图片生成后端（nil 时使用 xAI）
	imageBackend ImageBackend

	// 多引擎支持
	engineMgr      *modelengine.Manager
	activeEngineID string // 当前活跃引擎 ID（空字符串="xai"）

	// OnEvent 可选回调：每次 API 调用前后触发，用于前端实时日志
	OnEvent func(eventType string, data map[string]interface{})

	// 并发控制信号量（限制同时进行的 API 调用数）
	sem chan struct{}
}

// NewClient 创建 AI 客户端
func NewClient(cfg *config.Config) *Client {
	const maxConcurrency = 4 // SuperGrok 并发限制
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 0},
		tokenStore: auth.NewTokenStore(cfg.TokenStorePath),
		sem:        make(chan struct{}, maxConcurrency),
	}
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
	c.activeEngineID = engineID
	slog.Info("切换活跃引擎", "engine", engineID)
}

// ActiveEngineID 返回当前活跃引擎 ID
func (c *Client) ActiveEngineID() string {
	if c.activeEngineID == "" {
		return "xai"
	}
	return c.activeEngineID
}

// resolveChatEndpoint 根据活跃引擎解析 chat completions 的 URL 和 API key。
// 当活跃引擎是 xAI 时使用 OAuth token；其他引擎使用 engineManager。
func (c *Client) resolveChatEndpoint() (endpoint string, apiKey string, err error) {
	engineID := c.ActiveEngineID()

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
func (c *Client) resolveModelName(reqModel string) string {
	engineID := c.ActiveEngineID()

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

// GetToken 获取有效 token（自动刷新）
func (c *Client) GetToken() (string, error) {
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
	if err != nil {
		store := auth.NewTokenStore(c.cfg.TokenStorePath)
		tok, err := store.Load()
		if err != nil {
			slog.Warn("EnsureToken: token 加载失败", "error", err)
		}
		if tok != nil && !tok.IsExpired() {
			c.token = tok
			return nil
		}
		return err
	}
	return nil
}

// tryRefreshToken 尝试刷新 token 并保存，成功返回 nil
func (c *Client) tryRefreshToken() error {
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

// Chat 非流式对话
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := c.acquireSem(ctx); err != nil {
		return nil, fmt.Errorf("等待 API 槽位: %w", err)
	}
	defer c.releaseSem()

	endpoint, apiKey, err := c.resolveChatEndpoint()
	if err != nil {
		return nil, err
	}

	// 如果请求中模型名为空，自动填充引擎默认模型
	if req.Model == "" {
		req.Model = c.resolveModelName("")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// 仅 xAI 引擎做 401 token 刷新重试
	if resp.StatusCode == 401 && c.ActiveEngineID() == "xai" {
		c.releaseSem()
		if err := c.tryRefreshToken(); err != nil {
			return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
		}
		return c.Chat(ctx, req)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(respBody), 500))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("[%s] %s", chatResp.Error.Code, chatResp.Error.Message)
	}
	return &chatResp, nil
}

// ── SSE 流式 ──────────────────────────────────────────────────

// ChatStream 流式对话，通过 channel 返回 SSEChunk
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (<-chan SSEChunk, error) {
	// 并发控制
	if err := c.acquireSem(ctx); err != nil {
		return nil, fmt.Errorf("等待 API 槽位: %w", err)
	}

	endpoint, apiKey, err := c.resolveChatEndpoint()
	if err != nil {
		c.releaseSem()
		return nil, err
	}

	// 如果请求中模型名为空，自动填充引擎默认模型
	if req.Model == "" {
		req.Model = c.resolveModelName("")
	}

	req.Stream = true
	body, err := json.Marshal(req)
	if err != nil {
		c.releaseSem()
		return nil, fmt.Errorf("marshal stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		c.releaseSem()
		return nil, fmt.Errorf("构造流式请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.releaseSem()
		return nil, fmt.Errorf("流式请求失败: %w", err)
	}

	// 仅 xAI 做 401 重试
	if resp.StatusCode == 401 && c.ActiveEngineID() == "xai" {
		resp.Body.Close()
		c.releaseSem()
		if err := c.tryRefreshToken(); err != nil {
			return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
		}
		return c.ChatStream(ctx, req)
	}
	if resp.StatusCode != 200 {
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		c.releaseSem()
		if readErr != nil {
			return nil, fmt.Errorf("API 错误 (HTTP %d): <无法读取响应体>", resp.StatusCode)
		}
		return nil, fmt.Errorf("API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(body), 500))
	}

	chunks := make(chan SSEChunk, 64)
	go c.parseStreamEvents(ctx, resp, chunks)
	return chunks, nil
}

// parseStreamEvents 解析 SSE 事件流并发送到 chunks channel
func (c *Client) parseStreamEvents(ctx context.Context, resp *http.Response, chunks chan SSEChunk) {
	defer resp.Body.Close()
	defer close(chunks)
	defer c.releaseSem()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			chunks <- SSEChunk{Error: "请求已取消"}
			return
		default:
		}

		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			chunks <- SSEChunk{Done: true}
			return
		}

		var choice struct {
			Choices []ChatChoice `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &choice); err != nil {
			continue
		}

		if len(choice.Choices) > 0 {
			delta := choice.Choices[0].Delta
			if delta.Content != "" {
				chunks <- SSEChunk{Content: delta.Content}
			}
			if delta.FinishReason != "" {
				chunks <- SSEChunk{Done: true}
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		chunks <- SSEChunk{Error: fmt.Sprintf("流读取错误: %v", err)}
		return
	}

	chunks <- SSEChunk{Done: true}
}

// ChatSimpleStream 简化流式对话，收集完整回复（5 分钟超时）。
func (c *Client) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	return c.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userMsg, ChatSimpleOptions{})
}

// ChatSimpleStreamWithOptions 简化流式对话，支持参数覆盖。
func (c *Client) ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ChatSimpleOptions) (string, error) {
	timeoutMinutes := opts.TimeoutMinutes
	if timeoutMinutes <= 0 {
		timeoutMinutes = 5
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	// 如果 model 为空，用引擎默认模型
	if model == "" {
		model = c.resolveModelName("")
	}

	c.emit("request", map[string]interface{}{
		"model":     model,
		"system":    util.Truncate(systemPrompt, 80),
		"user":      util.Truncate(userMsg, 80),
		"reasoning": opts.ReasoningEffort,
		"engine":    c.ActiveEngineID(),
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
		Messages:        []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userMsg}},
		MaxTokens:       maxTokens,
		Temperature:     temperature,
		ReasoningEffort: opts.ReasoningEffort,
	}
	if opts.TopP > 0 {
		req.TopP = opts.TopP
	}

	chunks, err := c.ChatStream(ctx, req)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
loop:
	for {
		select {
		case <-ctx.Done():
			return sb.String(), fmt.Errorf("请求超时或已取消")
		case chunk, ok := <-chunks:
			if !ok {
				break loop
			}
			if chunk.Error != "" {
				return sb.String(), fmt.Errorf("%s", chunk.Error)
			}
			if chunk.Done {
				break loop
			}
			sb.WriteString(chunk.Content)
		}
	}
	result := sb.String()
	c.emit("response", map[string]interface{}{
		"length":  len([]rune(result)),
		"content": result,
	})
	return result, nil
}

// SetImageBackend 设置本地图片生成后端（nil 回退到 xAI）
func (c *Client) SetImageBackend(backend ImageBackend) {
	c.imageBackend = backend
}

// GetImageBackendType 获取当前图片后端配置
func (c *Client) GetImageBackendType() string {
	if c.imageBackend != nil {
		return "comfyui"
	}
	return "xai"
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

// GenerateImage 调用 /v1/images/generations 生成图片
func (c *Client) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	if c.imageBackend != nil {
		return c.imageBackend.GenerateImage(ctx, req)
	}
	return c.generateImageXAI(ctx, req)
}

// generateImageXAI xAI 原生图片生成
func (c *Client) generateImageXAI(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	req.Size = ""

	endpoint := strings.TrimSuffix(c.cfg.XaiAPIBaseURL, "/") + "/images/generations"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal image request: %w", err)
	}

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
