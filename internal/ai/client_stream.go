// client_stream.go — 流式对话域（2026-10-04 自 client.go 纯搬移：
// ChatStream/doStreamRequest/parseStreamEvents/SSE 逐行与 idleTimeoutBody、
// ChatSimple* 包装族；配方照批 25 websearch.go）。

package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gaea/gaea/internal/modelengine"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

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
		c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// 如果请求中模型名为空，自动填充引擎默认模型
	if req.Model == "" {
		req.Model = c.resolveModelName("", req.EngineID)
	}

	// 思考型引擎的预算兜底：必须落在本函数（流式唯一漏斗）而不是
	// prepareStreamRequest——后者只覆盖 ChatSimple* 家族，手装 ChatRequest 的
	// 调用方绕过它（见 applyThinkingBudgetGuard 注释）。放在序列化之前。
	c.applyThinkingBudgetGuard(req, reqEngine)

	req.Stream = true
	// 流式接口默认不返回 usage；显式请求 include_usage 以便统计 Token。
	// 部分服务端不支持该字段（400），会在下方去掉后重试一次。
	if !req.skipIncludeUsage && req.StreamOptions == nil {
		req.StreamOptions = &ChatStreamOptions{IncludeUsage: true}
	}
	body, err := json.Marshal(req)
	if err != nil {
		c.releaseSem()
		c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
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
		// 失败请求照常记账（原引擎/模型）。
		c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		// C 刀故障转移 v0：doStreamRequest 只在「尚未收到任何响应字节」时返回
		// 错误（连接失败/非 200；200 返回即流已开始，不进本分支）。可转移时换
		// 候选引擎重试一次；重试请求带 failoverDone 标记，不再二次转移。
		if req != nil && !req.failoverDone {
			if to, toModel, ok := c.failoverTarget(reqEngine, err); ok {
				slog.Warn("流式请求失败，故障转移重试", "from_engine", reqEngine, "to_engine", to, "model", toModel, "error", err)
				retryReq := *req
				retryReq.EngineID = to
				retryReq.Model = toModel
				retryReq.failoverDone = true
				if c.OnFailover != nil {
					c.OnFailover(reqEngine, to, toModel)
				}
				return c.ChatStream(ctx, &retryReq)
			}
		}
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
	// 解析协程 panic 防线：每轮 LLM 调用必经。通道 close 的责任在本层（不在
	// parseStreamEvents）：panic 时先补发错误块再 close——此前 recover 只写
	// 日志、close 已由被调方执行，消费方按「正常收帧」处理拿到的部分内容，
	// 收到的是空成功（审计 P1 IN2-10）。
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("SSE 流解析 panic", "engine", reqEngine, "model", reqModel, "recover", r)
				select {
				case chunks <- SSEChunk{Error: fmt.Sprintf("SSE 流解析异常，请求中断: %v", r)}:
				case <-ctx.Done():
				}
			}
			close(chunks)
		}()
		c.parseStreamEvents(ctx, resp, chunks, reqEngine, reqModel, req.Feature, start)
	}()
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
// feature 为账目功能域标签（7.1-2 A 线，由 ChatStream 从 req.Feature 透传）。
func (c *Client) parseStreamEvents(ctx context.Context, resp *http.Response, chunks chan SSEChunk, reqEngine, reqModel, feature string, start time.Time) {
	defer resp.Body.Close()
	// close(chunks) 由 ChatStream 的解析协程包装层负责（panic 路径要先补发
	// 错误块再 close，IN2-10）；本函数被测试直接调用时由调用方 close。
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
			Feature:         feature,
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

	req := c.prepareStreamRequest(model, []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}, opts)

	reqEngine := opts.EngineID
	if reqEngine == "" {
		reqEngine = c.ActiveEngineID()
	}

	c.emit("request", map[string]interface{}{
		"model":     req.Model,
		"system":    systemPrompt,
		"user":      userMsg,
		"reasoning": opts.ReasoningEffort,
		"engine":    reqEngine,
	})

	chunks, err := c.ChatStream(ctx, req)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return chunks, cancel, nil
}

// ChatStreamMessages 与 ChatStreamChunks 同一份装配（超时/模型解析/default 采样/
// modelhub 让位/思考预算守护），但接受完整消息数组与工具定义：多轮工具循环要逐轮
// 追加 assistant(tool_calls) 与 tool 消息，两段式「system + user」表达不了。
// 装配逻辑只留在 prepareStreamRequest 一处，两条入口不会漂移。
func (c *Client) ChatStreamMessages(ctx context.Context, model string, messages []ChatMessage, opts ChatSimpleOptions, tools []ChatToolSchema) (<-chan SSEChunk, context.CancelFunc, error) {
	timeoutMinutes := opts.TimeoutMinutes
	if timeoutMinutes <= 0 {
		timeoutMinutes = 5
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)

	req := c.prepareStreamRequest(model, messages, opts)
	req.Tools = tools

	chunks, err := c.ChatStream(ctx, req)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return chunks, cancel, nil
}
