package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/gaea/genui"
	"github.com/gaea/gaea/internal/modelengine"
)

// chatPlainSystemPrompt 普通对话（聊天板块 plain 模式）的系统提示词。
// 众神殿刀1 三层契约第三层：删「不要说自己无法联网搜索」的反向施压——搜索
// 状态由注入头如实携带（ok/empty/失败三分叉，chat_search_adapter.go），提示词
// 只要求未获取到实时信息时如实说明。仅 plain 有这段联网说明，人格路径没有、勿加。
var chatPlainSystemPrompt = "你是 Gaea 桌面 AI 助手，用中文与用户进行日常对话。用户问你是谁时，明确回答你是 Gaea 的 AI 助手；不要把自己说成 xAI、Grok、OpenAI 或任何底层模型厂商的官方助手。当联网功能开启时，你可以调用 web_search 工具获取实时信息（天气/新闻/价格/影视等时效性问题建议先搜索），基于检索结果作答并注明信息来源；搜索失败或无结果时如实告知用户未获取到实时信息，基于已有知识回答并注明不确定，不要假装已搜索到。联网关闭时直接基于已有知识回答并注明可能过时。你不具备 web_search 以外的任何工具，任何时候都不要在回复正文里输出工具调用格式的文本。\n\n" + genui.ChatRule

// ChatSend 统一聊天入口（聊天/轻语板块合并，后端）。
// mode 为空或 "plain" → 普通对话；否则按人格 ID 走轻语 Orchestrator（记忆/情绪）。
// searchEnabled 打开时自动检测搜索意图并注入实时搜索结果（普通对话同样生效）。
// thinking 打开时对本地 Qwen3 系模型开启思考模式，回复附带思考链。
func (a *App) ChatSend(topicID, message, mode string, searchEnabled, thinking, forceSearch bool) (map[string]interface{}, error) {
	if topicID == "" {
		return nil, fmt.Errorf("话题 ID 不能为空")
	}
	if mode == "" || mode == "plain" {
		return a.chatSendPlain(topicID, message, searchEnabled, thinking, forceSearch)
	}
	return a.chatSendPersona(topicID, message, mode, searchEnabled, thinking, forceSearch)
}

// preparePlainChatMessage 联网搜索增强：命中搜索意图/强制搜索时，经统一检索
// 核 adapter（chat_search_adapter.go）注入——注入头按状态选文案（三层契约
// 第二层）：OK=实时结果；EMPTY=已搜索无相关结果；ALL_FAILED=联网暂不可用。
func (a *App) chatSendPlain(topicID, message string, searchEnabled, thinking, forceSearch bool) (map[string]interface{}, error) {
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	userMessage := message
	eng, model, source, err := a.resolveChatModelRoute()
	if err != nil {
		return nil, err
	}
	opts := ai.ChatSimpleOptions{
		EngineID: eng, Feature: "chat", EnableThinking: thinking, DisableThinking: !thinking,
		DisableFailover: source == "feature",
	}
	var reply, reasoning string
	// B7 工具循环（按钮语义=允许联网总开关）：开着即带 web_search 真工具，
	// 要不要搜/搜什么由模型判断（关键词门控退役）；关闭=不传工具零变化。
	if searchEnabled || forceSearch {
		res, loopErr := a.runBoundedSearchToolLoop(a.ctx, eng, model,
			[]ai.ChatMessage{{Role: "system", Content: chatPlainSystemPrompt}, {Role: "user", Content: userMessage}},
			opts, forceSearch, nil, nil)
		if loopErr != nil {
			return nil, loopErr
		}
		reply, reasoning = res.Reply, res.Reasoning
	} else {
		reply, reasoning, err = a.clientRef().ChatSimpleStreamDetailed(a.ctx, model,
			chatPlainSystemPrompt, userMessage, opts)
		if err != nil {
			return nil, err
		}
		reply = stripToolCallMarkup(reply)
	}
	extra := ""
	if reasoning != "" {
		if b, err := json.Marshal(map[string]interface{}{"reasoning": reasoning}); err == nil {
			extra = string(b)
		}
	}
	// T6-3.2：落库失败必须透传给调用方（前端可见），不再静默吞错。
	if err := a.appendChatExchange(topicID, userMessage, reply, extra); err != nil {
		return nil, err
	}
	// answered_by 回显：engine/model/source 来自 routeModel（离线过滤 + 全局/
	// 兜底已生效），source 即「为何」。ChatSimpleStreamDetailed 不返回本次 usage
	// （用量内部已上报统计）→ usage 在此点不可达，cost_cny 诚实记 0，不虚报。
	answeredBy := map[string]interface{}{"engine": eng, "model": model, "source": source, "cost_cny": 0.0}
	return map[string]interface{}{"reply": reply, "reasoning": reasoning, "mode": "plain", "topicID": topicID, "answered_by": answeredBy}, nil
}

// newChatStreamRunID 生成流式 runID（测试可替换为固定值以订阅固定事件名）。
// 毫秒时间戳尾部拼进程内序号：同毫秒两路并发（双面板连发/程序化重试）不再
// 碰撞——碰撞即两 goroutine 向同一 "chat-stream:<runID>" 通道交错发帧。
var chatStreamRunSeq atomic.Int64

var newChatStreamRunID = func() string {
	return fmt.Sprintf("cs_%d_%d", time.Now().UnixMilli(), chatStreamRunSeq.Add(1))
}

// streamTokensPerSecond 生成速度（tok/s）＝ completion tokens ÷ 实测流式耗时，
// 四舍五入到 0.1；tokens 或耗时非正值返回 0（调用方按 0 = 不下发不落库，
// 不虚报）。聊天与原罪两条流式链路共用同一口径。
func streamTokensPerSecond(tokens int64, elapsed time.Duration) float64 {
	if tokens <= 0 || elapsed <= 0 {
		return 0
	}
	return math.Round(float64(tokens)/elapsed.Seconds()*10) / 10
}

// streamOutputMetrics 汇总一次流式生成的可见性能指标。
// 首字延迟从请求发出计到首个正文/推理分片；prefill_tps 使用 prompt tokens
// 除以首字耗时，completion_tps 使用 completion tokens 除以首字后的生成时段。
// 这是通用 OpenAI 兼容流能稳定获得的口径，不把网络排队时间伪装成 GPU 预填时间。
func streamOutputMetrics(requestStart, firstTokenAt, finishedAt time.Time, usage *ai.ChatUsage) map[string]interface{} {
	metrics := map[string]interface{}{}
	if !requestStart.IsZero() && !finishedAt.IsZero() && finishedAt.After(requestStart) {
		metrics["duration_ms"] = finishedAt.Sub(requestStart).Milliseconds()
	}
	if !requestStart.IsZero() && !firstTokenAt.IsZero() && firstTokenAt.After(requestStart) {
		firstMS := firstTokenAt.Sub(requestStart).Milliseconds()
		metrics["first_token_ms"] = firstMS
		metrics["prefill_ms"] = firstMS
		if usage != nil && usage.PromptTokens > 0 {
			metrics["prefill_tps"] = math.Round(float64(usage.PromptTokens)/firstTokenAt.Sub(requestStart).Seconds()*10) / 10
		}
	}
	if !firstTokenAt.IsZero() && !finishedAt.IsZero() && finishedAt.After(firstTokenAt) {
		decodeMS := finishedAt.Sub(firstTokenAt).Milliseconds()
		metrics["decode_ms"] = decodeMS
		if usage != nil && usage.CompletionTokens > 0 {
			metrics["completion_tps"] = streamTokensPerSecond(usage.CompletionTokens, finishedAt.Sub(firstTokenAt))
		}
	}
	if usage != nil {
		metrics["prompt_tokens"] = usage.PromptTokens
		metrics["completion_tokens"] = usage.CompletionTokens
		total := usage.TotalTokens
		if total <= 0 {
			total = usage.PromptTokens + usage.CompletionTokens
		}
		metrics["total_tokens"] = total
		metrics["cache_hit_tokens"] = usage.CacheHitTokens()
		metrics["cache_miss_tokens"] = usage.CacheMissTokens()
		if tps, ok := metrics["completion_tps"].(float64); ok && tps > 0 {
			metrics["tokens_per_second"] = tps
		}
	}
	return metrics
}

// ChatStreamPlain 普通对话真实流式入口：先完成聊天绑定引擎预检，再返回 runID，
// 前端订阅 "chat-stream:<runID>" 事件流（delta / reasoning / done / error），完成后落库。
// 联网搜索注入只进入模型上下文，落库仍保留用户原文。
//
// T6-3.1 时序保障：delta 帧在 goroutine 内就绪即发，首帧可能早于前端订阅建立；
// 「订阅先行 + 首帧可重放」由本刀前端（ChatPage 订阅竞态修复）负责，后端不做
// emit 前等待。后端只保证终态纪律：任一失败路径（启动失败/流错误/落库失败/panic）
// 必 emit error，正常完成必 emit done；无帧超时由前端兜底（等待超时按失败展示）。
func (a *App) ChatStreamPlain(topicID, message string, searchEnabled, thinking, forceSearch bool) (string, error) {
	if topicID == "" {
		return "", fmt.Errorf("话题 ID 不能为空")
	}
	if a.clientRef() == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	eng, model, source, err := a.resolveChatModelRoute()
	if err != nil {
		return "", err
	}
	runID := newChatStreamRunID()
	// P1-5 对话条：请求级 ctx 登记进取消表（ChatStreamCancel 可中断——长回合
	// 不再只能等 30s 静默超时）；协程退出独占清表（chapterGen 同款纪律）。
	parent := a.ctx
	if parent == nil {
		parent = context.Background() // 裸构造（测试桩）防御；生产 a.ctx 恒就绪
	}
	streamCtx, cancel := context.WithCancel(parent)
	a.chatStreamRegister(runID, cancel)
	go a.runChatStreamPlain(streamCtx, runID, topicID, message, eng, model, source, searchEnabled, thinking, forceSearch)
	return runID, nil
}

// ChatStreamPersona 角色对话真实流式入口。人格链路仍由 Whisper 编排器完成
// 情绪、记忆与状态收尾，但模型正文不再等整段完成后才回前端：首个 delta 直接
// 通过 chat-stream:<runID> 下发，前端无需再模拟打字。
func (a *App) ChatStreamPersona(topicID, message, mode string, searchEnabled, thinking, forceSearch bool) (string, error) {
	if topicID == "" {
		return "", fmt.Errorf("话题 ID 不能为空")
	}
	if strings.TrimSpace(mode) == "" || mode == "plain" {
		return "", fmt.Errorf("角色对话模式无效")
	}
	if a.clientRef() == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	// 角色流同样先完成绑定引擎预检，再返回 runID。否则冷启动期间前端只握着
	// 一个尚无事件的 runID，会先触发 30s 流超时。
	if _, _, _, err := a.resolveChatModelRoute(); err != nil {
		return "", err
	}
	runID := newChatStreamRunID()
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	streamCtx, cancel := context.WithCancel(parent)
	a.chatStreamRegister(runID, cancel)
	go a.runChatStreamPersona(streamCtx, runID, topicID, message, mode, searchEnabled, thinking, forceSearch)
	return runID, nil
}

// chatStreamRegister 登记进行中的对话流取消函数。
func (a *App) chatStreamRegister(runID string, cancel context.CancelFunc) {
	a.chatStreamMu.Lock()
	defer a.chatStreamMu.Unlock()
	if a.chatStreamCancels == nil {
		a.chatStreamCancels = make(map[string]context.CancelFunc)
	}
	a.chatStreamCancels[runID] = cancel
}

// chatStreamUnregister 协程退出时独占清表（幂等）。
func (a *App) chatStreamUnregister(runID string) {
	a.chatStreamMu.Lock()
	delete(a.chatStreamCancels, runID)
	a.chatStreamMu.Unlock()
}

// ChatStreamCancel 取消指定对话流（P1-5）：已生成部分照常落库并以 cancelled
// 终态收尾（对齐章节生成取消语义——保留部分回复）；runID 未知（已结束/不存在）
// 返回 false 不算错。
func (a *App) ChatStreamCancel(runID string) bool {
	a.chatStreamMu.Lock()
	cancel, running := a.chatStreamCancels[runID]
	a.chatStreamMu.Unlock()
	if !running || cancel == nil {
		return false
	}
	cancel()
	return true
}

func (a *App) runChatStreamPlain(ctx context.Context, runID, topicID, userMessage, eng, model, source string, searchEnabled, thinking, forceSearch bool) {
	defer a.chatStreamUnregister(runID)
	defer func() {
		if r := recover(); r != nil {
			slog.Error("chat stream plain panic", "panic", r)
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": "流式生成异常"})
		}
	}()

	// B7 工具循环（按钮语义=允许联网总开关）：开着即带 web_search 真工具，
	// 要不要搜/搜什么由模型判断（关键词门控退役）；关闭=不传工具零变化。
	toolMode := searchEnabled || forceSearch
	requestStart := time.Now()
	var reply strings.Builder
	var reasoning strings.Builder
	// usage 由流结束块携带（OpenAI 兼容 API 最后一块带 usage，SSEChunk.Usage）；
	// 服务端未带 usage 时保持 nil → 费用诚实记 0。
	var usage *ai.ChatUsage
	// 生成速度计时：从首个内容/思考 delta 起表（排队与首字延迟不计入 tok/s）。
	var firstTokenAt time.Time
	if toolMode {
		res, loopErr := a.runBoundedSearchToolLoop(ctx, eng, model,
			[]ai.ChatMessage{{Role: "system", Content: chatPlainSystemPrompt}, {Role: "user", Content: userMessage}},
			ai.ChatSimpleOptions{
				EngineID: eng, Feature: "chat", EnableThinking: thinking, DisableThinking: !thinking,
				DisableFailover: source == "feature",
			},
			forceSearch,
			func(delta string) {
				reply.WriteString(delta)
				a.emit("chat-stream:"+runID, map[string]interface{}{"type": "delta", "content": delta})
			},
			func(r string) {
				reasoning.WriteString(r)
				a.emit("chat-stream:"+runID, map[string]interface{}{"type": "reasoning", "content": r})
			})
		if loopErr != nil {
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": loopErr.Error()})
			return
		}
		usage = res.Usage
		firstTokenAt = res.FirstToken
	} else {
		chunks, cancel, err := a.clientRef().ChatStreamChunks(ctx, model, chatPlainSystemPrompt, userMessage,
			ai.ChatSimpleOptions{
				EngineID: eng, Feature: "chat", EnableThinking: thinking, DisableThinking: !thinking,
				DisableFailover: source == "feature",
			})
		if err != nil {
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": err.Error()})
			return
		}
		defer cancel()
		mf := &markupDeltaFilter{} // B7 残刀三：非工具轮标记流式吞除
		for chunk := range chunks {
			if chunk.Error != "" {
				a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": chunk.Error})
				return
			}
			if chunk.Done {
				usage = chunk.Usage
				break
			}
			if (chunk.Content != "" || chunk.Reasoning != "") && firstTokenAt.IsZero() {
				firstTokenAt = time.Now()
			}
			if chunk.Content != "" {
				safe := mf.feed(chunk.Content)
				if safe != "" {
					reply.WriteString(safe)
					a.emit("chat-stream:"+runID, map[string]interface{}{"type": "delta", "content": safe})
				}
			}
			if chunk.Reasoning != "" {
				reasoning.WriteString(chunk.Reasoning)
				a.emit("chat-stream:"+runID, map[string]interface{}{"type": "reasoning", "content": chunk.Reasoning})
			}
		}
		if rest := mf.flush(); rest != "" {
			reply.WriteString(rest)
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "delta", "content": rest})
		}
	}

	// P1-5 用户取消：已生成部分照常走落库与终态——按 cancelled（而非 done/error）
	// 如实告知前端（保留部分回复，对齐章节生成取消语义）。
	cancelled := ctx.Err() != nil
	finishedAt := time.Now()
	replyStr := reply.String()
	reasoningStr := reasoning.String()
	// answered_by 回显：费用按本次实际 token（流结束块 usage）经 EstimateCostCNY
	// 估算，汇率取配置（与 GaeaGetUsdCnyRate 同源；非法值由 EstimateCostCNY 内
	// 回退默认 7.2）；usage 不可达时 cost_cny=0（诚实，不虚报）。
	costCNY := 0.0
	if usage != nil {
		usdCny := 0.0
		if a.cfg != nil {
			usdCny = a.cfg.UsdCnyRate
		}
		costCNY = modelengine.EstimateCostCNY(eng, model, usage.PromptTokens, usage.CompletionTokens, usdCny)
	}
	finalType := "done"
	if cancelled {
		finalType = "cancelled"
	}
	// 生成速度（tok/s）随 answered_by 下发：usage 缺失/无正文时不带该键（不虚报）。
	answeredBy := map[string]interface{}{
		"engine": eng, "model": model, "source": source, "cost_cny": costCNY,
	}
	metrics := streamOutputMetrics(requestStart, firstTokenAt, finishedAt, usage)
	if len(metrics) > 0 {
		answeredBy["metrics"] = metrics
	}
	if usage != nil && !firstTokenAt.IsZero() {
		if s := streamTokensPerSecond(usage.CompletionTokens, finishedAt.Sub(firstTokenAt)); s > 0 {
			answeredBy["tokens_per_second"] = s
		}
	}
	persistedExtra := map[string]interface{}{"answered_by": answeredBy}
	if reasoningStr != "" {
		persistedExtra["reasoning"] = reasoningStr
	}
	extra := ""
	if b, err := json.Marshal(persistedExtra); err == nil {
		extra = string(b)
	}
	// T6-3.2 落库错误透传：delta 流不受影响，仅在最终落库失败时 emit error 终态
	// 而非 done——前端可见失败（消息已生成但未持久化）。
	if err := a.appendChatExchange(topicID, userMessage, replyStr, extra); err != nil {
		slog.Error("chat stream plain 落库失败", "runID", runID, "topicID", topicID, "error", err)
		a.emit("chat-stream:"+runID, map[string]interface{}{
			"type":  "error",
			"error": "回复生成完成但消息保存失败: " + err.Error(),
		})
		return
	}
	a.emit("chat-stream:"+runID, map[string]interface{}{
		"type":        finalType,
		"reply":       replyStr,
		"reasoning":   reasoningStr,
		"topicID":     topicID,
		"answered_by": answeredBy,
	})
}

func (a *App) runChatStreamPersona(ctx context.Context, runID, topicID, userMessage, mode string, searchEnabled, thinking, forceSearch bool) {
	defer a.chatStreamUnregister(runID)
	defer func() {
		if r := recover(); r != nil {
			slog.Error("chat stream persona panic", "panic", r)
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": "角色流式生成异常"})
		}
	}()

	out, err := a.whisperChatStream(
		ctx,
		userMessage,
		mode,
		searchEnabled,
		thinking,
		forceSearch,
		false,
		func(delta string) {
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "delta", "content": delta})
		},
		func(delta string) {
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "reasoning", "content": delta})
		},
	)
	if err != nil {
		a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": err.Error()})
		return
	}

	reply := fmt.Sprint(out["reply"])
	if reply == "<nil>" {
		reply = ""
	}
	reasoning := fmt.Sprint(out["reasoning"])
	if reasoning == "<nil>" {
		reasoning = ""
	}
	answeredBy := out["answered_by"]
	persistedExtra := map[string]interface{}{
		"emotion": out["emotion"], "trust": out["trust"], "stage": out["stage"],
		"totalTurns": out["totalTurns"], "answered_by": answeredBy,
	}
	if reasoning != "" {
		persistedExtra["reasoning"] = reasoning
	}
	extra := ""
	if b, marshalErr := json.Marshal(persistedExtra); marshalErr == nil {
		extra = string(b)
	}

	finalType := "done"
	if cancelled, _ := out["cancelled"].(bool); cancelled {
		finalType = "cancelled"
	}
	// 众神殿刀1：空 assistant 一律不落库（取消空轮/异常空流不再伪造成功交换）；
	// 取消但保留了部分文本仍照常落库（产品语义：保留已生成内容）。
	if reply != "" {
		if err := a.appendChatExchange(topicID, userMessage, reply, extra); err != nil {
			slog.Error("chat stream persona 落库失败", "runID", runID, "topicID", topicID, "error", err)
			a.emit("chat-stream:"+runID, map[string]interface{}{
				"type":  "error",
				"error": "回复生成完成但消息保存失败: " + err.Error(),
			})
			return
		}
	} else {
		slog.Warn("chat stream persona 空回复不落库", "runID", runID, "finalType", finalType)
	}
	a.emit("chat-stream:"+runID, map[string]interface{}{
		"type":        finalType,
		"reply":       reply,
		"reasoning":   reasoning,
		"topicID":     topicID,
		"mode":        mode,
		"emotion":     out["emotion"],
		"aff":         out["aff"],
		"aro":         out["aro"],
		"answered_by": answeredBy,
	})
}

func (a *App) chatSendPersona(topicID, message, mode string, searchEnabled, thinking, forceSearch bool) (map[string]interface{}, error) {
	var out map[string]interface{}
	var err error
	if forceSearch || searchEnabled {
		out, err = a.WhisperChatWithSearch(message, mode, thinking, forceSearch)
	} else {
		out, err = a.WhisperChat(message, mode, thinking)
	}
	if err != nil {
		return nil, err
	}
	extra := ""
	if b, err := json.Marshal(map[string]interface{}{
		"emotion": out["emotion"], "trust": out["trust"], "stage": out["stage"], "totalTurns": out["totalTurns"],
		"reasoning": out["reasoning"], "answered_by": out["answered_by"],
	}); err == nil {
		extra = string(b)
	}
	// T6-3.2：persona 路径同样把落库失败透传给调用方。
	// 众神殿刀1：空 assistant 不落库、不上报成功（与流式路径同一守卫口径）。
	reply := fmt.Sprint(out["reply"])
	if reply == "<nil>" {
		reply = ""
	}
	if reply == "" {
		return nil, fmt.Errorf("模型返回空回复")
	}
	if err := a.appendChatExchange(topicID, message, reply, extra); err != nil {
		return nil, err
	}
	out["mode"] = mode
	out["topicID"] = topicID
	return out, nil
}

// appendChatExchange 把「用户消息 + 助手消息」原子落库；失败时返回 error
// （调用方决定透传还是转 error 事件），不再静默吞错。
func (a *App) appendChatExchange(topicID, userMsg, reply, extra string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	if err := a.chatStore.AppendExchange(topicID, userMsg, reply, extra); err != nil {
		slog.Error("chat exchange 落库失败", "topicID", topicID, "error", err)
		return err
	}
	return nil
}

// ── 话题 CRUD（统一会话存储）─────────────────────────────────

// ChatTopicsList 列出聊天板块话题（T6-3.2：读错返回 error，前端可见失败而非空列表）。
// 原罪（sin）故事话题存同一张表（mode=sin）但属于闲庭·原罪板块：这里过滤
// 掉——聊天板块的 persona 分支不认识 sin 模式，放进去既污染列表也会走错
// 生成路径；原罪侧经 SinTopicsList 读自己的话题（见 sin_handler.go）。
func (a *App) ChatTopicsList() ([]chat.Topic, error) {
	if a.chatStore == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	all, err := a.chatStore.ListTopics()
	if err != nil {
		slog.Error("chat 话题列表读取失败", "error", err)
		return nil, err
	}
	topics := make([]chat.Topic, 0, len(all))
	for _, t := range all {
		if t.Mode == sinTopicMode {
			continue
		}
		topics = append(topics, t)
	}
	return topics, nil
}

func (a *App) ChatTopicCreate(title, mode string) (chat.Topic, error) {
	if a.chatStore == nil {
		return chat.Topic{}, fmt.Errorf("chat store 未初始化")
	}
	id := fmt.Sprintf("t_%d", time.Now().UnixMilli())
	if err := a.chatStore.CreateTopic(id, title, mode); err != nil {
		return chat.Topic{}, err
	}
	return a.chatStore.GetTopic(id)
}

func (a *App) ChatTopicRename(id, title string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.RenameTopic(id, title)
}

// ChatTopicSetMode 切换话题模式（plain ↔ personaID），前端模式切换器使用。
func (a *App) ChatTopicSetMode(id, mode string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.SetMode(id, mode)
}

// ChatTopicClear 清空话题消息（前端「清空当前对话」）。
func (a *App) ChatTopicClear(id string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.ClearMessages(id)
}

func (a *App) ChatTopicDelete(id string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.DeleteTopic(id)
}

// ChatMessagesList 列出话题全部消息（T6-3.2：读错返回 error，前端可见失败而非空列表）。
// ChatMessagesPageResult 分页拉取结果：Messages 升序（旧→新），HasMore 表示
// 是否还存在更早的历史（v4.370 聊天历史分页）。
type ChatMessagesPageResult struct {
	Messages  []chat.Message `json:"messages"`
	HasMore   bool           `json:"hasMore"`
	OldestSeq int64          `json:"oldestSeq"`
}

// ChatMessagesPage 游标式分页拉取话题消息：beforeSeq<=0 从最新一条向前取
// limit 条；否则取 seq<beforeSeq 的最新 limit 条。前端首屏取最新 200 条，
// 向上翻页时以本批最早 seq 为游标继续拉。
func (a *App) ChatMessagesPage(topicID string, limit int, beforeSeq int64) (ChatMessagesPageResult, error) {
	if a.chatStore == nil {
		return ChatMessagesPageResult{}, fmt.Errorf("chat store 未初始化")
	}
	msgs, hasMore, err := a.chatStore.ListMessagesPage(topicID, limit, beforeSeq)
	if err != nil {
		slog.Error("chat 消息分页读取失败", "topicID", topicID, "error", err)
		return ChatMessagesPageResult{}, err
	}
	var oldest int64
	if len(msgs) > 0 {
		oldest = int64(msgs[0].Seq)
	}
	return ChatMessagesPageResult{Messages: msgs, HasMore: hasMore, OldestSeq: oldest}, nil
}

func (a *App) ChatMessagesList(topicID string) ([]chat.Message, error) {
	if a.chatStore == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	msgs, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		slog.Error("chat 消息列表读取失败", "topicID", topicID, "error", err)
		return nil, err
	}
	return msgs, nil
}

// ChatAppendMessages 把一组消息追加到已有话题（T6-3.3 语音消息持久化）：
// 单事务批量写入，全部成功或全部回滚；role 仅接受 user/assistant，其余跳过。
// 语音消息目前只在前端内存——前端可在收到语音识别结果后调用本绑定落库。
func (a *App) ChatAppendMessages(topicID string, messages []ChatMessageInput) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	inputs := make([]chat.MessageInput, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		inputs = append(inputs, chat.MessageInput{Role: role, Content: m.Content, Extra: m.Extra})
	}
	return a.chatStore.AppendMessagesTx(topicID, inputs)
}

// ChatTopicExportMarkdown 把话题全部消息导出为 Markdown 文件，写到用户数据目录
// 的 exports/chat 下，返回绝对路径（前端可展示/复制路径）。
func (a *App) ChatTopicExportMarkdown(topicID string) (string, error) {
	if a.chatStore == nil {
		return "", fmt.Errorf("chat store 未初始化")
	}
	topic, err := a.chatStore.GetTopic(topicID)
	if err != nil {
		return "", err
	}
	title := topic.Title
	msgs, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	b.WriteString("> 导出时间：" + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
	for _, m := range msgs {
		role := "用户"
		if m.Role == "assistant" {
			role = "AI"
		}
		b.WriteString("## " + role + "\n\n")
		// T6-3.5：消息原文写前转义 Markdown 结构敏感字符，避免破坏导出文档
		// （如行首井号生成新标题）；保留换行与加粗/斜体等基础格式。
		b.WriteString(escapeMarkdownContent(m.Content) + "\n\n")
	}

	dir := filepath.Join(a.whisperDataRoot, "exports", "chat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s_%d.md", sanitizeChatFilename(title), time.Now().UnixMilli())
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// windowsReservedNames Windows 保留设备名（大小写不敏感，含扩展名同样保留）。
var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// escapeMarkdownContent 转义消息原文中会破坏 Markdown 文档结构的字符
// （行首井号、反引号、尖括号、竖线），保留换行与基础格式（加粗/斜体/列表不受影响）。
func escapeMarkdownContent(s string) string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(escapeMarkdownLine(line))
	}
	return b.String()
}

// escapeMarkdownLine 单行转义：行首（可带前导空格）的 # 转义为 \#（避免生成标题），
// 反引号/尖括号/竖线全部转义为 \X（避免代码块/HTML/表格结构被破坏）。
func escapeMarkdownLine(line string) string {
	runes := []rune(line)
	firstNonSpace := 0
	for firstNonSpace < len(runes) && runes[firstNonSpace] == ' ' {
		firstNonSpace++
	}
	var b strings.Builder
	for i, r := range runes {
		switch {
		case i == firstNonSpace && r == '#':
			b.WriteString("\\#")
		case r == '`' || r == '<' || r == '>' || r == '|':
			b.WriteString("\\")
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitizeChatFilename 把话题标题规整为安全的文件名片段：
// 非法字符替换为 _、去首尾空白与尾部点号、截断 40 字符、规避 Windows 保留名。
func sanitizeChatFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' || r < 32:
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), " .")
	if out == "" {
		out = "chat"
	}
	runes := []rune(out)
	if len(runes) > 40 {
		out = string(runes[:40])
	}
	base := out
	if i := strings.LastIndexByte(out, '.'); i > 0 {
		base = out[:i]
	}
	if windowsReservedNames[strings.ToUpper(base)] {
		out = "_" + out
	}
	return out
}

// ChatMessageInput 历史消息导入输入（旧 localStorage 话题迁入 chat.db）。
type ChatMessageInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Extra   string `json:"extra"`
}

// ChatImportTopic 创建话题并按序导入历史消息（迁移旧 localStorage 会话，不调用 AI）。
// T6-3.4：建话题 + 全部消息改走单事务（ImportTopicTx），中途失败整体回滚，
// 不再出现「话题已建但消息只落了一半」的脏状态。
func (a *App) ChatImportTopic(title, mode string, messages []ChatMessageInput) (chat.Topic, error) {
	if a.chatStore == nil {
		return chat.Topic{}, fmt.Errorf("chat store 未初始化")
	}
	id := fmt.Sprintf("t_%d", time.Now().UnixMilli())
	inputs := make([]chat.MessageInput, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		inputs = append(inputs, chat.MessageInput{Role: role, Content: m.Content, Extra: m.Extra})
	}
	if err := a.chatStore.ImportTopicTx(id, title, mode, inputs); err != nil {
		return chat.Topic{}, err
	}
	return a.chatStore.GetTopic(id)
}
