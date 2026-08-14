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

	"github.com/gaea/gaea/internal/auth"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/netclient"
)

// Client xAI API 客户端，封装认证和 HTTP 通信
type Client struct {
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
}

// NewClient 创建 AI 客户端
func NewClient(cfg *config.Config) *Client {
	const maxConcurrency = 4 // SuperGrok 并发限制
	return &Client{
		cfg:        cfg,
		httpClient: netclient.NewSimpleClient(0),
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
		c.token = tok
		return nil
	}
	// token 不存在或已过期，返回原始错误
	if tok == nil {
		return fmt.Errorf("未登录：token 文件不存在")
	}
	return err
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
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// 仅 xAI 引擎做 401 token 刷新重试
	if resp.StatusCode == 401 && reqEngine == "xai" {
		if err := c.tryRefreshToken(); err != nil {
			c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
			return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
		}
		return c.Chat(ctx, req)
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		c.releaseSem()
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
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
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, fmt.Errorf("流式请求失败: %w", err)
	}

	// 仅 xAI 做 401 重试
	if resp.StatusCode == 401 && reqEngine == "xai" {
		resp.Body.Close()
		c.releaseSem()
		if err := c.tryRefreshToken(); err != nil {
			c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
			return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
		}
		return c.ChatStream(ctx, req)
	}
	if resp.StatusCode != 200 {
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		c.releaseSem()
		// 服务端不支持 stream_options.include_usage 时去掉该字段重试一次。
		if resp.StatusCode == 400 && req.StreamOptions != nil && readErr == nil &&
			(bytes.Contains(body, []byte("stream_options")) || bytes.Contains(body, []byte("include_usage"))) {
			req.skipIncludeUsage = true
			req.StreamOptions = nil
			return c.ChatStream(ctx, req)
		}
		errMsg := ""
		if readErr != nil {
			errMsg = fmt.Sprintf("API 错误 (HTTP %d): <无法读取响应体>", resp.StatusCode)
		} else {
			errMsg = fmt.Sprintf("API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(body), 500))
		}
		c.recordUsage(reqEngine, reqModel, start, 0, 0, 0, 0, false, errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	chunks := make(chan SSEChunk, 64)
	go c.parseStreamEvents(ctx, resp, chunks, reqEngine, reqModel, start)
	return chunks, nil
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

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			if !send(SSEChunk{Error: "请求已取消"}) {
				return
			}
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

	if err := scanner.Err(); err != nil {
		errMsg := fmt.Sprintf("流读取错误: %v", err)
		if !send(SSEChunk{Error: errMsg}) {
			return
		}
		return
	}

	if len(toolOrder) > 0 {
		send(SSEChunk{Done: true, ToolCalls: flushToolCalls(toolPending, toolOrder), Usage: streamUsage})
	} else {
		send(SSEChunk{Done: true, Usage: streamUsage})
	}
	streamOK = true
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
	c.imageBackend = backend
	if backendType == "" {
		backendType = "xai"
	}
	c.imageBackendType = backendType
}

// GetImageBackendType 获取当前图片后端类型
func (c *Client) GetImageBackendType() string {
	if c.imageBackendType == "" {
		return "xai"
	}
	return c.imageBackendType
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
