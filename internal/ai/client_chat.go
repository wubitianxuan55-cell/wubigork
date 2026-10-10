// client_chat.go — 非流式对话、用量记账与请求准备域（2026-10-04 自 client.go
// 纯搬移：Chat/chatOnce/flushToolCalls/recordUsage/思考预算钳制与
// prepareStreamRequest；配方照批 25 websearch.go）。

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/modelengine"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Chat 非流式对话。
//
// C 刀故障转移 v0：chatOnce（单引擎完整重试语义，见下）失败且满足可转移
// 条件（网络类错误或 HTTP 408/429/5xx，开关开启，存在候选引擎）时，换候选
// 引擎用其 default_model 重试一次；仍失败返回原错误。不递归——chatOnce 内
// 无转移逻辑，最多一次转移。失败的请求与重试请求照常逐笔记账（各按实际
// 引擎/模型）。开关关闭时本函数与原实现行为逐字节一致。
//
// chatOnce 请求建立阶段失败的重试语义与流式一致：
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

	resp, err := c.chatOnce(ctx, req)
	if err == nil {
		return resp, nil
	}
	from := req.EngineID
	if from == "" {
		from = c.ActiveEngineID()
	}
	if to, toModel, ok := c.failoverTarget(from, err); ok {
		slog.Warn("聊天请求失败，故障转移重试", "from_engine", from, "to_engine", to, "model", toModel, "error", err)
		retryReq := *req
		retryReq.EngineID = to
		retryReq.Model = toModel
		retryReq.failoverDone = true
		if c.OnFailover != nil {
			c.OnFailover(from, to, toModel)
		}
		if resp2, err2 := c.chatOnce(ctx, &retryReq); err2 == nil {
			return resp2, nil
		}
	}
	return nil, err
}

// chatOnce 单引擎一次完整聊天尝试（原 Chat 主体：引擎/模型解析 + 退避重试 +
// 401 刷新 + usage 记账），不含故障转移。由 Chat（外层）在信号量内调用。
func (c *Client) chatOnce(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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
		c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// 如果请求中模型名为空，自动填充引擎默认模型
	if req.Model == "" {
		req.Model = c.resolveModelName("", req.EngineID)
	}

	// 非流式同漏斗兜底：Chat 与 ChatStream 各自是手装 ChatRequest 的两个入口，
	// 两条都要过预算守护（同 applyThinkingBudgetGuard 注释）。
	c.applyThinkingBudgetGuard(req, reqEngine)

	body, err := json.Marshal(req)
	if err != nil {
		c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
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
				c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, errMsg)
				return nil, fmt.Errorf("%s", errMsg)
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
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
				c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
				return nil, fmt.Errorf("API 请求失败: %w", err)
			}
			lastErr = fmt.Errorf("API 请求失败: %w", err)
			slog.Warn("Chat 请求失败，准备退避重试", "attempt", attempt+1, "error", lastErr)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, readErr.Error())
			return nil, fmt.Errorf("read response body: %w", readErr)
		}

		// 仅 xAI 引擎做 401 token 刷新重试：刷新后立即在同一函数内重发一次
		//（不递归调用 Chat，避免外层 defer releaseSem 未执行而占 2 个信号量槽）。
		if resp.StatusCode == 401 && reqEngine == "xai" && !refreshed {
			if err := c.tryRefreshToken(); err != nil {
				c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
				return nil, fmt.Errorf("认证失败 (HTTP 401): %w", err)
			}
			refreshed = true
			newKey, err := c.GetToken()
			if err != nil {
				c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
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
			c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}

		var chatResp ChatResponse
		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, err.Error())
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		if chatResp.Error != nil {
			errMsg := fmt.Sprintf("[%s] %s", chatResp.Error.Code, chatResp.Error.Message)
			c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}
		var inTok, outTok, cacheHit, cacheMiss int64
		if chatResp.Usage != nil {
			inTok, outTok = chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens
			cacheHit, cacheMiss = cacheSplitForUsage(chatResp.Usage)
		}
		c.recordUsage(req.Feature, reqEngine, reqModel, start, inTok, outTok, cacheHit, cacheMiss, true, "")
		return &chatResp, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("API 请求失败（重试耗尽）")
	}
	c.recordUsage(req.Feature, reqEngine, reqModel, start, 0, 0, 0, 0, false, lastErr.Error())
	return nil, lastErr
}

// ── SSE 流式 ──────────────────────────────────────────────────

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
// feature 为账目功能域标签（7.1-2 A 线，空=未标记/历史）。
func (c *Client) recordUsage(feature, engineID, model string, start time.Time, inTok, outTok, cacheHit, cacheMiss int64, success bool, errMsg string) {
	if c.engineMgr == nil {
		return
	}
	c.engineMgr.RecordCall(modelengine.ModelCallUsage{
		EngineID:        engineID,
		Model:           model,
		Feature:         feature,
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

// minThinkingBudget 思考模式最小生成预算（IN2-14 单点）：思考与正文共享
// max_tokens，低于该值会出现「只有推理、无正文」（herdsman 模型测评报告
// §8.1/§9；modelhub Studio 同款表现）。流式请求 max_tokens 的默认值与其同源。
const minThinkingBudget = 4096

// clampThinkingBudget 思考预算守护（modelhub / herdsman / ollama / strata 各
// 分支共用，IN2-14 收口——原来两处逐字复制）：开启思考且显式给出小预算时抬到
// minThinkingBudget；未开思考或未显式配置（<=0）不动。
func clampThinkingBudget(req *ChatRequest, wantThinking bool) {
	if wantThinking {
		req.MaxTokens = ClampThinkingMaxTokens(req.MaxTokens)
	}
}

// ClampThinkingMaxTokens 思考型引擎的生成预算守护（纯函数，导出给 gaea provider
// bridge 复用）。思考模式与正文共享同一份 max_tokens：预算小于 minThinkingBudget
// 时推理会把预算烧光，正文被截断或整个为空（本包 strata 真机实测：160 token
// 预算下 finish_reason=length、正文只剩 8 字）。n<=0（未显式配置，上游已按默认
// 值兜底）原样返回。
//
// bridge 手装 ChatRequest、不走 prepareStreamRequest，此前 4096 在 client_chat.go
// 与 bridge.go 各写一遍——同值不同源是漂移隐患，故收口到本函数单一来源。
func ClampThinkingMaxTokens(n int) int {
	if n > 0 && n < minThinkingBudget {
		return minThinkingBudget
	}
	return n
}

// applyThinkingBudgetGuard 思考原生引擎的生成预算兜底，落在 ChatStream/chatOnce
// 两个漏斗入口上（流式与非流式各一个）。
//
// 为什么不放在 prepareStreamRequest：那只覆盖 ChatSimple* 家族。手装 ChatRequest
// 的调用方绕过它，且都是短预算高风险点——章节直连（applyChapterGuardrails 把
// max_tokens 设成 play 护栏上限）、copilot CmdK 续写（256）、wx_agent（4096）、
// gaea provider bridge（办公智能体多轮工具循环）。这些形态在思考原生引擎上会
// 稳定复现「推理吃光预算、正文截断或整个为空」。
//
// 判据取引擎类型（非引擎名），仅 Strata 一类思考原生引擎需要：它的服务端默认开
// 思考，调用方不表态就是开。调用方显式 ctk=false 时不抬预算（推理不再抢预算，
// 该尊重调用方自己的上限）。
//
// 幂等：经 prepareStreamRequest 装配的请求在此为二次应用，同值。MaxTokens<=0
// （调用方刻意不发上限）原样保留——真机实测 Strata 无 max_tokens 时按服务端
// 宽上限正常收尾（393 token 完整），不替调用方凭空造一个上限。
func (c *Client) applyThinkingBudgetGuard(req *ChatRequest, engineID string) {
	if req == nil || c.engineMgr == nil {
		return
	}
	eng, ok := c.engineMgr.GetEngine(engineID)
	if !ok || eng.Type != modelengine.EngineStrata {
		return
	}
	// 显式关思考（ctk false）：预算不守护。nil map 读取安全，缺键即「未表态」。
	if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); ok && !v {
		return
	}
	req.MaxTokens = ClampThinkingMaxTokens(req.MaxTokens)
}

// prepareStreamRequest 装配一轮流式请求：模型名兜底、default 采样、modelhub 让位、
// Qwen3 系思考预算守护。ChatStreamChunks 与 ChatStreamMessages 共用同一份实现。
//
// 注意：本函数只覆盖 ChatSimple* 家族。手装 ChatRequest 直接调 ChatStream/Chat 的
// 调用方（章节直连、copilot 续写、wx_agent、gaea provider bridge）绕过这里，
// 引擎级兜底由 ChatStream/chatOnce 漏斗上的 applyThinkingBudgetGuard 负责。
func (c *Client) prepareStreamRequest(model string, messages []ChatMessage, opts ChatSimpleOptions) *ChatRequest {
	// 如果 model 为空，用引擎默认模型（功能级引擎优先）
	if model == "" {
		model = c.resolveModelName("", opts.EngineID)
	}
	reqEngine := opts.EngineID
	if reqEngine == "" {
		reqEngine = c.ActiveEngineID()
	}

	// 引擎类型判定（modelhub / strata 两条本地分支按 Type 而非 ID 查证：自定义
	// 引擎 Type 恒为 custom，内置同型引擎各只有一个，按 Type 判定可免名单漂移）。
	// reqEngine 已做活跃引擎兜底（与 resolveChatEndpoint 同口径）——此前用未兜底
	// 的 opts.EngineID 查表：功能级未指定引擎、而活跃引擎恰是 modelhub 时，
	// isModelHub 判定为假，采样让位与思考让位双双落空。
	var engType modelengine.EngineType
	if c.engineMgr != nil {
		if eng, ok := c.engineMgr.GetEngine(reqEngine); ok {
			engType = eng.Type
		}
	}
	isModelHub := engType == modelengine.EngineModelHub

	temperature := opts.Temperature
	// MH1 采样让位：modelhub 下用户未显式配置（<=0）时不传 temperature
	// （零值经 omitempty 丢弃），把默认采样让给 Studio 自动调优；其余引擎
	// 维持 0.7 兜底。
	if temperature <= 0 && !isModelHub {
		temperature = 0.7
	}
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		// 与思考预算守护同一常量（IN2-14）：默认 4096 本就是「思考+正文」
		// 够用的最低档，两处同值不同源是漂移隐患。
		maxTokens = minThinkingBudget
	}

	req := &ChatRequest{
		Model:           model,
		EngineID:        opts.EngineID,
		Feature:         opts.Feature,
		Messages:        messages,
		MaxTokens:       maxTokens,
		Temperature:     temperature,
		ReasoningEffort: opts.ReasoningEffort,
	}
	if opts.TopP > 0 {
		req.TopP = opts.TopP
	}

	switch {
	case isModelHub:
		// 实测（蒸馏规划 §七）：unsloth 服务端默认开思考，token 全烧在
		// reasoning（正文 0 字、finish=length）——gaea 专业秘书人设默认显式
		// 关思考；显式开启（乐园人格等）时传 true 并同样抬预算守护。只走
		// chat_template_kwargs（A/B 实测证实的唯一有效通道），不发顶层
		// enable_thinking。
		req.ChatTemplateKwargs = map[string]any{"enable_thinking": opts.EnableThinking}
		clampThinkingBudget(req, opts.EnableThinking)

	case engType == modelengine.EngineStrata:
		// Strata（Qwen3.8-Flash-Next 125B MoE）：思考原生引擎——服务端默认开
		// 思考，reasoning_content 与正文共享 max_tokens。与 herdsman/ollama
		// 「缺省关、显式开」相反，本分支缺省即开（要关得由调用方显式表态，
		// 见 ChatSimpleOptions.DisableThinking 三态语义）。
		//
		// 通道（2026-10-10 真机逐项对照）：顶层 enable_thinking 无效——传
		// false 仍产出 reasoning_content；chat_template_kwargs 是唯一有效
		// 通道，故本分支只发 ctk、不发顶层字段。
		//
		// 预算守护必需（同批实测）：160 token 预算下推理烧光预算，正文被
		// 截断成 8 字且 finish_reason=length（「忽长忽短/偶尔空」的真身）；
		// 抬到 4096 后同一请求 finish=stop 完整收尾。短预算调用点密集
		// （意图分类 64 / 幽灵建议 160 / 章节阶段 700 / 人格护栏 256·2048），
		// 不守护即稳定复现截断。
		//
		// 与 play 护栏的优先序（有意为之，非疏漏）：applyChatSimpleMaxTokens
		// 以 4096 为基线「只降不升」，本守护在此「只升不降」——两者相对，
		// 即思考优先于输出上限，与 modelhub/herdsman 的 clampThinkingBudget
		// 同一条先例。要按护栏上限出字须显式关思考（DisableThinking，见下），
		// 那时本守护不生效、上限被尊重。
		wantThinking := !opts.DisableThinking
		req.ChatTemplateKwargs = map[string]any{"enable_thinking": wantThinking}
		clampThinkingBudget(req, wantThinking)

	case opts.EnableThinking:
		if reqEngine == "herdsman" || reqEngine == "ollama" {
			t := true
			req.EnableThinking = &t
			req.ChatTemplateKwargs = map[string]any{"enable_thinking": true}
			// 守护：思考与正文共享 max_tokens，显式小预算抬到 minThinkingBudget
			//（同 modelhub 分支，IN2-14 收口）。
			clampThinkingBudget(req, true)
		}
	}

	return req
}
