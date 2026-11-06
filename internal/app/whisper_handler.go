package app

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
	"github.com/gaea/gaea/internal/whisper/db/repos"
	qrcode "github.com/skip2/go-qrcode"
)

// Chat 实现 whisper.LlmClient 接口（接入 gaea 模型中心）
func (a *whisperState) Chat(systemPrompt, userPrompt string) (string, error) {
	if a.app == nil {
		return "", fmt.Errorf("聊天模型路由未初始化")
	}
	eng, model, source, err := a.app.resolveChatModelRoute() // 2.x 聊天/轻语合并：统一走 chat 路由
	if err != nil {
		return "", err
	}
	return a.clientRef().ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "chat", DisableFailover: source == "feature",
	})
}

// GetEngineList 返回模型中心全部引擎 ID（轻语设置面板引擎选择器用）
func (a *whisperState) GetEngineList() []string {
	if a.engineMgr == nil {
		return []string{"default"}
	}
	engs := a.engineMgr.GetEngines()
	ids := make([]string, 0, len(engs))
	for _, e := range engs {
		ids = append(ids, e.ID)
	}
	if len(ids) == 0 {
		return []string{"default"}
	}
	return ids
}

var (
	whisperSessions   = map[string]*whisper.Orchestrator{}
	whisperSessionsMu sync.RWMutex
)

const (
	voiceContextCharBudget = 5000
	voiceHistoryTurns      = 4
	voiceMaxOutputTokens   = 768
)

// truncateVoiceContext 保留人格状态、情绪与高优先级记忆的头部，尾部低权重
// 召回内容在语音低延迟模式下截断。文字聊天不经过此函数。
func truncateVoiceContext(s string) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= voiceContextCharBudget {
		return string(runes)
	}
	return string(runes[:voiceContextCharBudget]) + "\n\n【语音低延迟模式：低优先级上下文已压缩】"
}

func (a *whisperState) getOrCreateOrch(personalityID string) *whisper.Orchestrator {
	sessionID := "whisper_" + personalityID
	whisperSessionsMu.RLock()
	if orch, ok := whisperSessions[sessionID]; ok {
		whisperSessionsMu.RUnlock()
		return orch
	}
	whisperSessionsMu.RUnlock()

	var preset *whisper.PersonalityPreset
	if personalityID == "" || personalityID == "plain" {
		// 普通对话：聊天板块 plain 模式的语音回复使用中性助手，不套用任何角色
		preset = &whisper.PersonalityPreset{
			ID:         "plain",
			Label:      "普通对话",
			Gender:     "neutral",
			Dims:       whisper.PersonalityDims{T: 60, I: 50, S: 50, O: 60, R: 50},
			VoiceGuide: "普通对话：你是自然、直接、务实的 AI 助手，不扮演任何角色。语气平和清晰，先准确理解问题，再给出简洁有用的回答。",
		}
	} else {
		preset = whisper.GetPreset(personalityID)
		// 全局角色库优先：库内角色（含可编辑的内置人格）直接作为聊天人格，
		// 且库卡是唯一事实源——ToPreset 已折叠性格/背景/口吻/行为规则/情感
		// 逻辑/对话样本，不再被助手记录的精简快照整体覆盖（旧序会把厚人格
		// 换成只剩 voiceGuide 的薄壳，性格背景全部丢失）。助手记录只在库内
		// 无此人格时兜底（小说导入等历史路径）。
		fromLib := false
		if a.charLib != nil {
			if c, err := a.charLib.Get(personalityID); err == nil && c != nil && !c.Hidden && c.ChatEnabled {
				preset = c.ToPreset()
				fromLib = true
			}
		}
		if preset == nil {
			preset = whisper.GetPreset("gaea")
		}
		if preset == nil {
			preset = &whisper.PersonalityPresets[0]
		}
		// 小说角色导入的自定义人格（不在角色库）：助手记录带 voiceGuide 时兜底
		if !fromLib {
			if ast, ok := a.assistantMgr.FindByPersonality(personalityID); ok && ast.VoiceGuide != "" {
				dims := ast.Dims
				if dims.T == 0 && dims.I == 0 && dims.S == 0 && dims.O == 0 && dims.R == 0 {
					dims = whisper.PersonalityDims{T: 50, I: 50, S: 50, O: 50, R: 50}
				}
				preset = &whisper.PersonalityPreset{
					ID:         ast.PersonalityID,
					Label:      ast.Name,
					Gender:     ast.Gender,
					Dims:       dims,
					Tags:       ast.Tags,
					VoiceGuide: ast.VoiceGuide,
				}
				slog.Info("使用自定义人格角色", "personalityID", personalityID, "name", ast.Name)
			}
		}
	}
	orch := whisper.NewOrchestrator(sessionID, *preset)
	orch.DataRoot = a.whisperDataRoot
	// KV 前缀稳定化二刀：历史走 append-only messages（见 whisperChat 调用处注释）
	orch.HistoryInMessages = true
	// FTS 全文检索回调：buildTierBBlock 经此走 hermes.db FTS5 索引（含 LIKE 中文降级）
	orch.FTSSearch = func(query string, limit int) []string {
		ids, err := repos.SearchFactIDsFTS(a.whisperDataRoot, query, limit)
		if err != nil {
			return nil
		}
		return ids
	}

	// 众神殿刀4：先恢复后发布——状态恢复、生日自愈与历史回灌全部完成后才进
	// whisperSessions，并发首轮与旁路读（追踪页等）不再可能拿到半成品实例
	//（PreLLMTurn 读到空 WM/空 FactStore 的窗口关闭）。双建代价从「浪费一次
	// 构建」变为「浪费一次构建+恢复」，同键并发首轮罕见，可接受；构建段无
	// 全局副作用，先建者让位的语义不变。
	if err := restoreWhisperState(orch); err != nil {
		slog.Error("[whisper] 会话状态恢复失败（继续用全新状态）", "sessionID", orch.SessionID, "error", err)
	}
	a.healImplicitBirthday(orch)
	a.warmPromptHistory(orch)

	whisperSessionsMu.Lock()
	if existing, ok := whisperSessions[sessionID]; ok {
		// 审计 P0#13（源 AP6-02）：查表（RLock）→ 建实例 → 回写三段跨锁，
		// 并发同键会双建 orchestrator 且后写者覆盖先写者。入库走写锁二次查表：
		// 先建者让位，已入库的胜出，本地重复品直接丢弃。
		whisperSessionsMu.Unlock()
		return existing
	}
	whisperSessions[sessionID] = orch
	whisperSessionsMu.Unlock()
	return orch
}

func (a *whisperState) WhisperGetPersonalities() []whisper.PersonalityPreset {
	// 统一人格列表 = 角色库中可聊天角色（内置人格已种子化进库，可在库内编辑）。
	// 库读失败回退内置预设但 warn 留痕（2026-09-19 审计）：不再吞错返回空列表。
	if a.charLib != nil {
		items, err := a.charLib.ListChatEnabled()
		if err != nil {
			slog.Warn("[whisper] 角色库人格列表读取失败（回退内置人格）", "error", err)
		} else {
			out := make([]whisper.PersonalityPreset, 0, len(items))
			for i := range items {
				if p := items[i].ToPreset(); p != nil {
					out = append(out, *p)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return whisper.PersonalityPresets
}

func (a *whisperState) WhisperChat(userMsg string, personalityID string, thinking bool) (result map[string]interface{}, err error) {
	// 绑定签名不可变：委托内部实现，无助手名注入（assistantName 空=不碰该字段）
	return a.whisperChat(userMsg, personalityID, "", thinking, false, false)
}

// whisperChat WhisperChat 的内部实现。assistantName 非空时在 LockTurn 持锁窗口内
// 写 orch.AssistantName：同人格多助手共享同一 orchestrator（"whisper_"+personalityID
// 全局缓存），微信回调按助手身份注入名字必须在回合锁内完成——锁外直写会互相覆盖，
// 且与 PreLLMTurn 锁内读（BuildAckemCanonBlock 身份块）构成数据竞争。
// searchEnabled/forceSearch 控制搜索决策（含待补槽位续接），增广块只进模型输入。
func (a *whisperState) whisperChat(userMsg, personalityID, assistantName string, thinking, searchEnabled, forceSearch bool) (result map[string]interface{}, err error) {
	return a.whisperChatCore(a.ctx, userMsg, personalityID, assistantName, thinking, false, searchEnabled, forceSearch, nil, nil)
}

// whisperChatCore 是阻塞/流式两条输入路径共用的角色回合实现。onDelta 与
// onReasoning 非空时，模型分片一到达就交给调用方；否则只聚合最终文本。
// 众神殿刀3 双通道：userMsg 恒为用户原文（进 TurnPlan/PreLLMTurn/WM/
// FinalizeTurn/记忆摄入），搜索增广块只拼进本轮模型输入（llmUserMsg →
// messages）；B3 净行刀后 RecordPromptTurn 亦记原句——动态块与搜索结果
// 不再经历史跨轮重放（同进程/重启/语音三口径一致）。
// 多轮续接刀（action-7）：搜索决策收进 core（需 orch 读 PendingSearch）——
// 正常触发照旧；未触发但上一轮刚搜过且本句是实体短答时，做一次「上一轮
// 查询词+本句实体」的合并检索（明天天气→四川成都），未消费即清。
func (a *whisperState) whisperChatCore(
	ctx context.Context,
	userMsg, personalityID, assistantName string,
	thinking bool,
	lowLatency bool,
	searchEnabled, forceSearch bool,
	onDelta, onReasoning func(string),
) (result map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[whisper] PANIC", "panic", fmt.Sprintf("%v", r))
			result = nil
			err = fmt.Errorf("whisper panic: %v", r)
		}
	}()
	if ctx == nil {
		ctx = a.ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}

	msgPreview := userMsg
	if len(userMsg) > 80 {
		// 按 rune 截断预览（按字节会把汉字切成半截，日志出现 \xe6\x81 伪影，
		// 曾误导排查为提示词损坏）
		r := []rune(userMsg)
		if len(r) > 40 {
			msgPreview = string(r[:40]) + "…"
		} else {
			msgPreview = userMsg
		}
	}
	slog.Info("[whisper] WhisperChat start", "userMsg", msgPreview, "personality", personalityID)

	orch := a.getOrCreateOrch(personalityID)

	// T7-1.1 会话并发安全：同一角色 GUI/微信/语音三入口串行化，
	// 防止状态机并发撕裂（TotalTurns/情绪/成人 FSM）。
	orch.LockTurn()
	defer orch.UnlockTurn()

	// 助手名注入必须在持锁窗口内：PreLLMTurn 持同一回合锁读该字段组装身份块
	if assistantName != "" {
		orch.AssistantName = assistantName
	}

	// v4.16 离线裂缝收口：persona 主链路与 plain 聊天同源走 routeModel（功能绑定 →
	// 全局 → 兜底，离线模式自动滤云端）；无可用时 model 为空，由客户端按既有
	// 「模型不可用」路径处理（与 v4.15 plain 聊天一致）。
	// B6 前移：路由解析提到搜索决策前——模型拟查询（formulateSearchQuery）
	// 需要引擎/模型，一次解析两处消费。
	if a.app == nil {
		return nil, fmt.Errorf("聊天模型路由未初始化")
	}
	engine, model, source, routeErr := a.app.resolveChatModelRoute()
	if routeErr != nil {
		return nil, routeErr
	}
	// per-call 引擎覆盖：不影响全局激活引擎，多会话并发安全
	if a.clientRef() == nil {
		slog.Error("[whisper] client is nil")
		return nil, fmt.Errorf("model client not initialized")
	}

	// 搜索决策（双通道 + 待补槽位续接）：状态机/记忆永远只吃原文，增广块
	// 只进本轮模型输入。正常触发照旧；未触发但上一轮刚搜过（PendingSearch）
	// 且本句是实体短答 → 一次合并检索；本轮回合结束即消费/清除，不做悬挂。
	// B6 拟查询刀：触发仍走规则（关键词门控，便宜确定），查询词交给模型从
	// 对话语境拟写（指代/省略/连接词规则修不完——「所以，明天的天气」实机
	// 病例），失败有界降级回原句清洗。
	llmAppendix := ""
	searchTriggered := false
	if !isIdentityQuestion(userMsg) && (forceSearch || (searchEnabled && shouldSearchWeb(userMsg))) {
		q := a.formulateSearchQuery(engine, model, userMsg, "")
		in := a.prepareWhisperChatInput(userMsg, q, true)
		llmAppendix = in.LLMAppendix
		searchTriggered = true
	} else if searchEnabled || forceSearch {
		if p := orch.State.PendingSearch; p != nil && whisper.IsSearchFollowupAnswer(userMsg) {
			merged := strings.TrimSpace(p.Query) + " " + strings.TrimSpace(userMsg)
			q := a.formulateSearchQuery(engine, model, userMsg, p.Query)
			if q == "" {
				q = merged
			}
			slog.Info("[whisper] 待补槽位合并检索", "prevQuery", truncateRunes(p.Query, 40), "followup", userMsg, "query", truncateRunes(q, 40))
			out, err := chatWebSearch(q)
			if err != nil {
				slog.Warn("[whisper] 续接检索适配器故障，注入如实占位", "error", err)
				llmAppendix = "\n\n[联网搜索暂不可用，请基于已有知识回答，并如实说明未获取到实时信息]"
			} else {
				llmAppendix = chatSearchAppendix(out)
			}
		}
	}

	turnPlan := whisper.BuildTurnPlanFromRules(userMsg, whisper.BuildTurnPlanRulePriors(userMsg))
	slog.Info("[whisper] TurnPlan", "routing", turnPlan.Routing, "goal", turnPlan.Goal)

	preResult := orch.PreLLMTurn(userMsg)
	slog.Info("[whisper] PreLLMTurn done", "hasSystemPrompt", len(preResult.SystemPrompt) > 0, "eventType", preResult.Event.Type,
		"dynCtxChars", len(preResult.DynamicContext))

	systemPrompt := preResult.SystemPrompt
	// KV 前缀稳定化（2026-11-01 提速刀，两段式）：
	// ① system 只留跨轮稳定段（orchestrator 动态上下文挪出）；
	// ② 会话历史走真实 append-only messages（RecentExchanges）——system+旧历史
	// 跨轮字节不变，引擎前缀缓存（llama-server --cache-reuse）跨轮命中，每轮
	// 只 prefill 新增轮次（动态上下文+本轮消息），长会话每轮 10~26s 冷 prefill
	// 降至 2~4s 量级。
	dynamicCtx := strings.TrimSpace(preResult.DynamicContext)
	if turnPlan.Routing == whisper.RouteStructuredChat && turnPlan.FormatHint != "" {
		dynamicCtx = strings.TrimSpace(dynamicCtx + "\n\n【本轮格式要求】\n" + turnPlan.FormatHint)
	}
	llmUserMsg := userMsg + llmAppendix
	if dynamicCtx != "" {
		llmUserMsg = "【本轮上下文】\n" + dynamicCtx + "\n【本轮上下文结束】\n\n用户消息：" + userMsg + llmAppendix
	}
	if lowLatency {
		dynamicCtx = truncateVoiceContext(dynamicCtx)
		if turnPlan.Routing == whisper.RouteStructuredChat && turnPlan.FormatHint != "" {
			formatBlock := "【本轮格式要求】\n" + turnPlan.FormatHint
			if !strings.Contains(dynamicCtx, formatBlock) {
				dynamicCtx = strings.TrimSpace(dynamicCtx + "\n\n" + formatBlock)
			}
		}
		llmUserMsg = userMsg + llmAppendix
		if dynamicCtx != "" {
			llmUserMsg = "【本轮上下文】\n" + dynamicCtx + "\n【本轮上下文结束】\n\n用户消息：" + userMsg + llmAppendix
		}
	}
	historyExchanges := preResult.RecentExchanges
	if lowLatency && len(historyExchanges) > voiceHistoryTurns {
		historyExchanges = historyExchanges[len(historyExchanges)-voiceHistoryTurns:]
	}
	messages := []ai.ChatMessage{{Role: "system", Content: systemPrompt}}
	for _, ex := range historyExchanges {
		if strings.TrimSpace(ex.UserText) != "" {
			messages = append(messages, ai.ChatMessage{Role: "user", Content: ex.UserText})
		}
		if strings.TrimSpace(ex.AssistantText) != "" {
			messages = append(messages, ai.ChatMessage{Role: "assistant", Content: ex.AssistantText})
		}
	}
	messages = append(messages, ai.ChatMessage{Role: "user", Content: llmUserMsg})

	// 路由解析已前移至搜索决策前（B6 拟查询刀，一次解析两处消费）。
	// S1.5-B play 内容护栏：persona_lock 在人格一致性参数（dims/voiceGuide）
	// 注入时追加人格锁定段（防系统层覆盖人格段）并锁温度上限；
	// max_output_tokens 钳制输出。未配置 = 零值 = 请求与现状逐字节一致。
	opts := ai.ChatSimpleOptions{EngineID: engine, Feature: "chat", EnableThinking: thinking, DisableThinking: !thinking}
	opts.DisableFailover = source == "feature"
	if lowLatency && opts.MaxTokens <= 0 {
		opts.MaxTokens = voiceMaxOutputTokens
	}
	systemPrompt = applyWhisperGuardrails(&opts, systemPrompt, orch.Preset, playGuardrails())
	// 前缀命中观测（KV 前缀稳定化验证）：system 跨轮哈希应恒定、历史轮数递增。
	// 交付合同观测（众神殿 2026-11-06 裁决风险#3）：护栏改的是局部 systemPrompt
	// 而实际发送的是 :326 构造的 messages——记录 messages[0] 的合同命中数与
	// 独立哈希，合同若缺位即当场暴露（sysHash 不能证明模型收到什么）。
	sysSum := sha1.Sum([]byte(systemPrompt))
	msg0Sum := sha1.Sum([]byte(messages[0].Content))
	contractHits := strings.Count(messages[0].Content, whisper.DeliveryContractClause)
	slog.Info("[whisper] calling LLM", "engine", engine, "model", model,
		"sysHash", fmt.Sprintf("%x", sysSum)[:8], "histTurns", len(historyExchanges),
		"dynCtxChars", len([]rune(dynamicCtx)), "lowLatency", lowLatency,
		"contractHits", contractHits, "msg0Hash", fmt.Sprintf("%x", msg0Sum)[:8])
	requestStart := time.Now()
	// 众神殿刀1（进度计划/gaea-chat-persona-optimization-20261105.md §1.1③）：
	// 空正文/取消守卫。非取消空正文有界重试一次，仍空则显式报错——空流/仅
	// reasoning/首 token 前静默夭折不得伪装成成功回合。流式出口仅在用户不可见
	// 输出（无正文且 reasoning 未外发）时重试，避免 UI 重复拼接。
	var reply, reasoning string
	var usage *ai.ChatUsage
	var firstTokenAt time.Time
	streamPending := ""
	emitDelta := func(delta string) {
		if onDelta == nil {
			return
		}
		streamPending += delta
		for {
			markerAt := strings.Index(streamPending, whisper.SplitMarker)
			if markerAt < 0 {
				break
			}
			if markerAt > 0 {
				onDelta(streamPending[:markerAt])
			}
			onDelta("\n")
			streamPending = streamPending[markerAt+len(whisper.SplitMarker):]
		}
		hold := 0
		for n := len(whisper.SplitMarker) - 1; n > 0; n-- {
			if strings.HasSuffix(streamPending, whisper.SplitMarker[:n]) {
				hold = n
				break
			}
		}
		if len(streamPending) > hold {
			cut := len(streamPending) - hold
			onDelta(streamPending[:cut])
			streamPending = streamPending[cut:]
		}
	}
	var finishedAt time.Time
	var cancelled bool
	for attempt := 1; ; attempt++ {
		chunks, cancel, callErr := a.clientRef().ChatStreamMessages(ctx, model, messages, opts, nil)
		if callErr != nil && ctx.Err() != nil {
			// 首 token 前取消（建连即取消）：按取消终态收尾，不算故障——
			// 空回复由下方守卫挡住落库与记账。建连失败时 cancel 为 nil
			//（ChatStreamMessages 错误路径返回 nil, nil, err），无需清理。
			finishedAt = time.Now()
			cancelled = true
			reply, reasoning = "", ""
			break
		}
		if callErr != nil {
			slog.Error("[whisper] LLM call failed", "error", callErr)
			return nil, callErr
		}
		var replyBuilder, reasoningBuilder strings.Builder
		usage = nil
		firstTokenAt = time.Time{}
		streamPending = ""
		for chunk := range chunks {
			if chunk.Error != "" {
				callErr = errors.New(chunk.Error)
				break
			}
			if chunk.Done {
				usage = chunk.Usage
				break
			}
			if (chunk.Content != "" || chunk.Reasoning != "") && firstTokenAt.IsZero() {
				firstTokenAt = time.Now()
			}
			if chunk.Content != "" {
				replyBuilder.WriteString(chunk.Content)
				emitDelta(chunk.Content)
			}
			if chunk.Reasoning != "" {
				reasoningBuilder.WriteString(chunk.Reasoning)
				if onReasoning != nil {
					onReasoning(chunk.Reasoning)
				}
			}
		}
		if onDelta != nil && streamPending != "" {
			onDelta(streamPending)
		}
		cancel()
		finishedAt = time.Now()
		cancelled = ctx.Err() != nil
		if callErr != nil && !cancelled {
			slog.Error("[whisper] LLM call failed", "error", callErr)
			return nil, callErr
		}
		reply = replyBuilder.String()
		reasoning = reasoningBuilder.String()
		if reply != "" || cancelled || attempt >= 2 || (reasoning != "" && onReasoning != nil) {
			break
		}
		slog.Warn("[whisper] 空回复，有界重试", "attempt", attempt)
	}
	if strings.TrimSpace(reply) == "" && !cancelled {
		return nil, fmt.Errorf("模型返回空回复（重试后仍为空）")
	}
	slog.Info("[whisper] LLM reply", "len", len(reply))
	streamStats := []any{"lowLatency", lowLatency, "totalMs", finishedAt.Sub(requestStart).Milliseconds()}
	if !firstTokenAt.IsZero() {
		streamStats = append(streamStats, "ttftMs", firstTokenAt.Sub(requestStart).Milliseconds())
	}
	slog.Info("[whisper] LLM stream stats", streamStats...)
	// 探针归因日志（众神殿人格论证刀0）：start 日志只预览 40 rune（搜索块被截掉），
	// 此处补 raw/llm 两形态长度+哈希、历史面与终态，供五桶探针逐轮取材。
	rawSum := sha1.Sum([]byte(userMsg))
	llmSum := sha1.Sum([]byte(llmUserMsg))
	slog.Info("[whisper] turn probe",
		"personality", personalityID,
		"rawUserMsgChars", len([]rune(userMsg)), "rawHash", fmt.Sprintf("%x", rawSum)[:8],
		"llmUserMsgChars", len([]rune(llmUserMsg)), "llmHash", fmt.Sprintf("%x", llmSum)[:8],
		"searchAppendixChars", len([]rune(llmAppendix)),
		"replyChars", len([]rune(reply)), "emptyReply", strings.TrimSpace(reply) == "",
		"cancelled", cancelled,
		"promptHistoryLen", len(orch.PromptHistory),
		"recentExchangesLen", len(preResult.RecentExchanges),
		"eventType", string(preResult.Event.Type),
		"gateLevel", preResult.Trace.L3.GateLevel,
		"strategyLevel", preResult.Trace.L3.StrategyLevel,
		"clampInjected", preResult.Trace.L3.ClampInjected,
		"ttftMs", finishedAt.Sub(requestStart).Milliseconds())

	// 节奏标记归一（v4.9.1）：chatter 模式要求模型用 [SPLIT] 分条连发，但该
	// 标记是内部格式协议——本函数是 GUI/微信/语音三个出口的共同上游，在这
	// 里把标记归一为换行（分条可读性保留），任何出口都不应再见到裸标记
	//（此前微信/语音直接发送原文，[SPLIT] 直接漏给用户）。裸 SPLIT 降级
	// 变体同剥（2026-11-06 复现探针 R4 泄漏实证，NormalizeReplySplitLines）。
	reply = whisper.NormalizeReplySplitLines(reply)

	sentences := whisper.SplitIntoSentences(reply)
	if len(sentences) == 0 {
		sentences = []string{reply}
	}

	costCNY := 0.0
	if usage != nil {
		usdCny := 0.0
		if a.cfg != nil {
			usdCny = a.cfg.UsdCnyRate
		}
		costCNY = modelengine.EstimateCostCNY(engine, model, usage.PromptTokens, usage.CompletionTokens, usdCny)
	}
	answeredBy := map[string]interface{}{
		"engine": engine, "model": model, "source": source, "cost_cny": costCNY,
	}
	metrics := streamOutputMetrics(requestStart, firstTokenAt, finishedAt, usage)
	if len(metrics) > 0 {
		answeredBy["metrics"] = metrics
	}
	if usage != nil && !firstTokenAt.IsZero() {
		if tps := streamTokensPerSecond(usage.CompletionTokens, finishedAt.Sub(firstTokenAt)); tps > 0 {
			answeredBy["tokens_per_second"] = tps
		}
	}

	// 众神殿刀1：空轮/取消轮不得伪装成完整回合记账——不推 WM/PromptHistory、
	// 不 FinalizeTurn（SharedEventsCount++/轻量事实/companion_reply 全在其内）、
	// 不入记忆队列。PreLLMTurn 侧状态演化随 persistStateAsync 照常落盘，保持
	// 内存/磁盘一致；轮次追踪仍落库（探针诚实记录，L4.Wrote=false）。
	skipBookkeeping := cancelled || strings.TrimSpace(reply) == ""
	if !skipBookkeeping {
		orch.WM.Push(orch.SessionID, whisper.Exchange{
			TurnIndex: orch.State.Counters.TotalTurns, UserText: userMsg, AssistantText: reply,
			Source: whisper.ExchangeSourceRaw,
		})
		// 净行刀（2026-11-06 B3 批，病灶①收口）：提示历史改记**原句**（userMsg
		// 无动态包裹与搜索增广）。旧「实发串」契约让【本轮上下文】动态块+搜索
		// 结果以 user 角色跨轮重放 4~8 轮=格式模仿老师+token 浪费；重启回灌
		//（warmPromptHistory）与语音出口（voice_v2:405）本就记原句，本轮刀后
		// 同进程/重启/语音三口径一致。KV 前缀代价=每轮缓存复用止于上一行
		//（原句行与实发行差一截），重 prefill 的只有一行短原句，可接受。
		if orch.HistoryInMessages {
			orch.RecordPromptTurn(orch.State.Counters.TotalTurns, userMsg, reply)
		}
	}

	l1Snap := orch.State.Relationship
	l2Snap := orch.State.Emotion
	turns := orch.State.Counters.TotalTurns

	if !skipBookkeeping {
		whisper.FinalizeTurn(orch, whisper.PostTurnContext{
			SessionID: orch.SessionID, TurnIndex: turns,
			UserMsg: userMsg, AssistantText: reply,
			Event: preResult.Event, AdultMode: orch.AdultMode,
		})
	}
	// 轮次追踪持久化（按会话归属，供角色库「追踪」页查看）
	if err := repos.AppendTurnTraceToDB(orch.DataRoot, orch.SessionID, preResult.Trace); err != nil {
		slog.Error("[whisper] 轮次追踪落库失败", "sessionID", orch.SessionID, "error", err)
	}

	if !skipBookkeeping {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("whisper: memory write goroutine panic recovered", "panic", r)
					a.recordMemoryWriteError(orch.SessionID, "panic", fmt.Errorf("memory write panic: %v", r))
				}
			}()
			// T6-5.3：错误回传（LLM 失败/JSON 解析失败/panic）计入 WriteErrors 计数
			whisper.EnqueueMemoryWrite(a, whisper.MemoryWritePayload{
				SessionID: orch.SessionID, TurnIndex: turns,
				UserMsg: userMsg, AssistantText: reply,
				L1: l1Snap, L2: l2Snap,
				FactStore: orch.FactStore, TotalTurns: turns, KG: orch.KG,
				EpisodicStore:   orch.EpisodicStore,
				RecentExchanges: buildRecentExchanges(orch),
				// 时间锚点写出口：摄入命中锚点策略 → 锁内落 State.TemporalAnchors，
				// 随 companion_state 持久化（「重访雨夜」数据源接线）
				TemporalAnchorSink: func(a whisper.TemporalAnchor) { orch.AddTemporalAnchor(a) },
				AdultMode:          orch.AdultMode,
			}, a.recordMemoryWriteError)
		}()
	}

	// 槽位续接状态机：本轮触发搜索→记录（下一轮可合并检索）；否则消费/清除
	//（宁漏勿误，不做多轮悬挂）。取消/空轮不触碰（保持重试语义）。
	if !skipBookkeeping {
		if searchTriggered {
			orch.State.PendingSearch = &whisper.PendingSearch{Query: userMsg, TurnIndex: turns}
		} else {
			orch.State.PendingSearch = nil
		}
	}

	go a.persistStateAsync(orch)

	slog.Info("[whisper] WhisperChat done", "replyLen", len(reply), "emotion", orch.State.Emotion.PrimaryLabel)
	return map[string]interface{}{
		"reply":        reply,
		"reasoning":    reasoning,
		"sentences":    sentences,
		"turnPlan":     turnPlan,
		"stage":        string(orch.State.Relationship.Stage),
		"emotion":      orch.State.Emotion.PrimaryLabel,
		"mood":         orch.State.Emotion.Mood,
		"trust":        orch.State.Relationship.Trust,
		"aff":          orch.State.Emotion.Aff,
		"sec":          orch.State.Emotion.Sec,
		"aro":          orch.State.Emotion.Aro,
		"dom":          orch.State.Emotion.Dom,
		"rifts":        orch.State.Relationship.Rifts,
		"totalTurns":   orch.State.Counters.TotalTurns,
		"desireSlots":  buildDesireSlots(orch.State.DesireStack),
		"trace":        preResult.Trace,
		"facts":        buildFactsList(orch.FactStore),
		"sharedEvents": orch.State.Counters.SharedEventsCount,
		"answered_by":  answeredBy,
		"cancelled":    cancelled,
	}, nil
}

func buildDesireSlots(stack whisper.DesireStack) []map[string]interface{} {
	var slots []map[string]interface{}
	for _, d := range stack.Slots {
		if d == nil {
			slots = append(slots, nil)
			continue
		}
		slots = append(slots, map[string]interface{}{
			"id": d.ID, "topic": d.Topic, "category": d.Category,
			"urgency": d.Urgency, "status": d.Status,
		})
	}
	return slots
}

func buildFactsList(fs *whisper.FactStore) []map[string]interface{} {
	active := fs.ListActive()
	facts := make([]map[string]interface{}, 0, len(active))
	for _, f := range active {
		fact := map[string]interface{}{
			"id":           f.ID,
			"domain":       f.Domain,
			"subcategory":  f.Subcategory,
			"subject":      f.Subject,
			"summary":      f.Summary,
			"weight":       f.Weight,
			"confidence":   f.Confidence,
			"createdAt":    f.CreatedAt.Format("2006-01-02 15:04"),
			"updatedAt":    f.UpdatedAt.Format("2006-01-02 15:04"),
			"tier":         f.RawTier,
			"triggers":     f.Triggers,
			"sensitivity":  f.Sensitivity,
			"privacyLevel": f.PrivacyLevel,
		}
		if f.EmotionalContext != nil {
			fact["emotionalContext"] = map[string]interface{}{
				"valence":   f.EmotionalContext.Valence,
				"intensity": f.EmotionalContext.Intensity,
				"trust":     f.EmotionalContext.Trust,
				"relStage":  string(f.EmotionalContext.RelStage),
			}
		}
		facts = append(facts, fact)
	}
	return facts
}

// WhisperGetFacts 独立获取当前会话的记忆列表
func (a *whisperState) WhisperGetFacts(personalityID string) []map[string]interface{} {
	orch := a.getOrCreateOrch(personalityID)
	if orch == nil {
		return nil
	}
	return buildFactsList(orch.FactStore)
}

// WhisperGetTraces 获取角色的轮次追踪（角色库「追踪」页；按会话从 hermes.db 读取）
func (a *whisperState) WhisperGetTraces(personalityID string) []whisper.TurnTrace {
	if a.whisperDataRoot == "" {
		return nil
	}
	traces, err := repos.LoadTurnTracesFromDBSession(a.whisperDataRoot, "whisper_"+personalityID, 80)
	if err != nil {
		return nil
	}
	return traces
}

// WhisperDeleteFact 删除指定记忆
func (a *whisperState) WhisperDeleteFact(personalityID string, factID string) error {
	whisperSessionsMu.RLock()
	orch, ok := whisperSessions["whisper_"+personalityID]
	whisperSessionsMu.RUnlock()
	if !ok {
		return fmt.Errorf("no active session")
	}
	orch.FactStore.RetireFact(factID)
	slog.Info("[whisper] fact deleted", "id", factID)
	return nil
}

// WhisperUpdateFact 更新记忆字段
func (a *whisperState) WhisperUpdateFact(personalityID string, factID string, updates map[string]interface{}) error {
	whisperSessionsMu.RLock()
	orch, ok := whisperSessions["whisper_"+personalityID]
	whisperSessionsMu.RUnlock()
	if !ok {
		return fmt.Errorf("no active session")
	}
	orch.FactStore.UpdateFact(factID, updates)
	slog.Info("[whisper] fact updated", "id", factID)
	return nil
}

func (a *whisperState) WhisperGetState(personalityID string) map[string]interface{} {
	orch := a.getOrCreateOrch(personalityID)
	if orch == nil {
		return map[string]interface{}{"error": "no active session"}
	}
	state := orch.State
	return map[string]interface{}{
		"relationship": map[string]interface{}{
			"stage":      string(state.Relationship.Stage),
			"trust":      state.Relationship.Trust,
			"rifts":      state.Relationship.Rifts,
			"atmosphere": string(state.Relationship.Atmosphere),
		},
		"emotion": map[string]interface{}{
			"aff": state.Emotion.Aff, "sec": state.Emotion.Sec, "aro": state.Emotion.Aro,
			"dom": state.Emotion.Dom, "label": state.Emotion.PrimaryLabel, "locked": state.Emotion.IsLocked,
		},
		"totalTurns":  state.Counters.TotalTurns,
		"desireStack": state.DesireStack,
		"personality": map[string]interface{}{
			"id": state.Personality.PresetID, "T": state.Personality.T,
			"I": state.Personality.I, "S": state.Personality.S,
			"O": state.Personality.O, "R": state.Personality.R,
		},
		"engineID":  orch.EngineID,
		"adultMode": orch.AdultMode,
	}
}

func (a *whisperState) WhisperSetEngine(engineID string) error {
	// 写改字段必须持写锁（审计 P1 AP6-06：RLock 下改写 orchestrator 字段，
	// 并发回合读到撕裂值，race 必红）。
	whisperSessionsMu.Lock()
	defer whisperSessionsMu.Unlock()
	for _, orch := range whisperSessions {
		orch.EngineID = engineID
	}
	return nil
}

func (a *whisperState) WhisperGetEngine() string {
	whisperSessionsMu.RLock()
	defer whisperSessionsMu.RUnlock()
	for _, orch := range whisperSessions {
		if orch.EngineID != "" {
			return orch.EngineID
		}
	}
	return a.clientRef().ActiveEngineID()
}

func (a *whisperState) WhisperSetModel(engineID, modelName string) error {
	whisperSessionsMu.Lock()
	defer whisperSessionsMu.Unlock()
	for _, orch := range whisperSessions {
		orch.EngineID = engineID
		orch.ModelName = modelName
	}
	return nil
}

func (a *whisperState) WhisperGetModel() string {
	whisperSessionsMu.RLock()
	defer whisperSessionsMu.RUnlock()
	for _, orch := range whisperSessions {
		if orch.ModelName != "" {
			return orch.ModelName
		}
	}
	return a.GetActiveModel()
}

func (a *whisperState) WhisperSetImageModel(modelName string) error {
	whisperSessionsMu.Lock()
	defer whisperSessionsMu.Unlock()
	for _, orch := range whisperSessions {
		orch.ImageModelName = modelName
	}
	return nil
}

func (a *whisperState) WhisperGetImageModel() string {
	whisperSessionsMu.RLock()
	defer whisperSessionsMu.RUnlock()
	for _, orch := range whisperSessions {
		if orch.ImageModelName != "" {
			return orch.ImageModelName
		}
	}
	return ""
}

func (a *whisperState) WhisperGetConfig() map[string]interface{} {
	return map[string]interface{}{
		"engine":       a.WhisperGetEngine(),
		"model":        a.WhisperGetModel(),
		"imageModel":   a.WhisperGetImageModel(),
		"engines":      a.GetEngines(),
		"activeEngine": a.GetActiveEngine(),
		"activeModel":  a.GetActiveModel(),
	}
}

func (a *whisperState) WhisperGetEngines() []modelengine.EngineConfig {
	return a.GetEngines()
}

func (a *whisperState) WhisperClearSession(personalityID string) error {
	sessionID := "whisper_" + personalityID
	whisperSessionsMu.Lock()
	delete(whisperSessions, sessionID)
	whisperSessionsMu.Unlock()
	return nil
}

// ─── 上网查询 ──────────────────────────────────────────────────

// WhisperWebSearch 执行上网查询（只读）。众神殿刀1 四入口收敛：委托统一检索
// 核 adapter（与 plain/人格注入同一条链），按状态回执——旧实现直连
// whisper.WebSearch 且 err 分支把 err.Error() 塞 map 却返回 nil error（结合旧栈
// 「暂无结果」伪成功，error 字段恒死），现按状态如实回填。
func (a *whisperState) WhisperWebSearch(query string) (map[string]interface{}, error) {
	slog.Info("[whisper] web search", "query", query)
	out, err := chatWebSearch(query)
	if err != nil {
		return map[string]interface{}{
			"query":  query,
			"state":  string(builtin.SearchStateAllFailed),
			"result": "",
			"error":  err.Error(),
		}, nil
	}
	m := map[string]interface{}{
		"query": query,
		"state": string(out.State),
	}
	if out.State == builtin.SearchStateOK {
		m["result"] = out.Text
	} else {
		m["result"] = ""
		m["error"] = "联网搜索未获得可用结果：" + out.Engines
	}
	return m, nil
}

// WhisperChatWithSearch 带搜索增强的对话：自动检测是否需要上网查询
func (a *whisperState) WhisperChatWithSearch(userMsg string, personalityID string, thinking, forceSearch bool) (map[string]interface{}, error) {
	// 绑定签名不可变：委托内部实现，无助手名注入
	return a.whisperChatWithSearch(userMsg, personalityID, "", thinking, forceSearch)
}

// whisperChatInput 用户原文与搜索增广的双通道载荷（众神殿刀3）：Raw 恒为
// 用户原话（进状态机/工作记忆/FinalizeTurn/记忆摄入），LLMAppendix 只拼进
// 本轮模型输入。此前增广串作为唯一 userMsg 全链下传，事件分类、FTS 召回与
// 记忆抽取都把网页摘要当用户原话（搜索桶「答非所问」的结构性根源）。
type whisperChatInput struct {
	Raw         string
	LLMAppendix string
}

// whisperChatStream 角色聊天流式入口：搜索增强、人格预编排与最终状态收尾
// 与阻塞路径共用 whisperChatCore，只把模型正文/推理分片实时交给调用方。
func (a *whisperState) whisperChatStream(
	ctx context.Context,
	userMsg, personalityID string,
	searchEnabled, thinking, forceSearch bool,
	lowLatency bool,
	onDelta, onReasoning func(string),
) (map[string]interface{}, error) {
	return a.whisperChatCore(ctx, userMsg, personalityID, "", thinking, lowLatency, searchEnabled, forceSearch, onDelta, onReasoning)
}

// whisperChatWithSearch 搜索增强的内部实现：assistantName 非空时透传给 whisperChat
// 在其 LockTurn 持锁窗口内注入 orch.AssistantName。
func (a *whisperState) whisperChatWithSearch(userMsg, personalityID, assistantName string, thinking, forceSearch bool) (map[string]interface{}, error) {
	return a.whisperChat(userMsg, personalityID, assistantName, thinking, true, forceSearch)
}

// prepareWhisperChatInput 在请求模型前完成可选联网搜索，返回双通道载荷。
// Raw 恒为用户原文；LLMAppendix 为四态注入头的增广块（空 = 未触发搜索）。
// 众神殿刀3：搜索结果只进入本轮模型上下文——状态机、工作记忆、PromptHistory
// 之外的落库面与记忆摄入全部改吃原文（旧注释「不污染落库的用户原文」只对
// chat.db 成立，对状态机与记忆不成立，本切片闭合该缺口）。
// formulateSearchQuery B6 拟查询刀（2026-11-06）：触发仍走规则门控，查询词
// 交给模型从对话语境拟写——指代/省略/连接词/相对日期规则修不完（「所以，
// 明天的天气」实机病例：规则清洗后仍搜出罗大佑《所以》）。有界设计：单次
// 调用 4s 超时、MaxTokens 48、DisableThinking；任何失败返回 ""（调用方降级
// 回原句清洗路径，行为同 B5 前）。prevQuery 非空时作为上文槽位提示注入。
func (a *whisperState) formulateSearchQuery(engine, model, userMsg, prevQuery string) string {
	if a.clientRef() == nil || engine == "" || model == "" {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("把用户最新的话改写成一条网页搜索关键词")
	if prevQuery != "" {
		fmt.Fprintf(&sb, "（结合此前的检索主题「%s」补全指代与省略）", truncateRunes(prevQuery, 60))
	}
	sb.WriteString("。只输出关键词本身：一行、不带解释、不带引号、不带标点结尾。")
	today := time.Now().Format("2006-01-02")
	sys := fmt.Sprintf("你是搜索查询拟写器。今天是 %s。把对话中的最新请求改写成适合搜索引擎的关键词组合；结合上文补全指代（如「所以/那/它」指代的内容），去掉口语连接词，保留关键实体（地点/对象/日期词）。%s", today, sb.String())
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	opts := ai.ChatSimpleOptions{EngineID: engine, Feature: "chat-search", DisableThinking: true, MaxTokens: 48}
	out, _, err := a.clientRef().ChatSimpleMessagesStreamDetailed(ctx, model, []ai.ChatMessage{
		{Role: "system", Content: sys},
		{Role: "user", Content: userMsg},
	}, opts)
	if err != nil {
		slog.Warn("[whisper] 拟查询降级回原句", "error", err)
		return ""
	}
	q := strings.TrimSpace(out)
	q = strings.Trim(q, " \t\r\n\"「」『』'。．.，,")
	if q == "" || len([]rune(q)) > 60 || strings.ContainsAny(q, "\n") {
		return ""
	}
	slog.Info("[whisper] 拟查询", "query", truncateRunes(q, 40), "raw", truncateRunes(userMsg, 40))
	return q
}

func (a *whisperState) prepareWhisperChatInput(userMsg, searchQuery string, forceSearch bool) whisperChatInput {
	// 检测搜索意图（身份类问题除外——「你是谁/你会什么」问的是助手自己，
	// 联网搜索只会注入无关网页摘要，模型把英文片段混进回复反而出乱码）
	if !isIdentityQuestion(userMsg) && (forceSearch || shouldSearchWeb(userMsg)) {
		// 预览按 rune 截断——字节切片会把多字节中文切成 mojibake 进日志（AP6-07）
		slog.Info("[whisper] auto-search triggered", "msg", truncateRunes(userMsg, 60), "query", truncateRunes(searchQuery, 40))
		if strings.TrimSpace(searchQuery) == "" {
			searchQuery = userMsg
		}
		out, err := chatWebSearch(searchQuery)
		if err != nil {
			slog.Warn("[whisper] 搜索适配器故障，注入如实占位", "error", err)
			return whisperChatInput{Raw: userMsg, LLMAppendix: "\n\n[联网搜索暂不可用，请基于已有知识回答，并如实说明未获取到实时信息]"}
		}
		return whisperChatInput{Raw: userMsg, LLMAppendix: chatSearchAppendix(out)}
	}
	return whisperChatInput{Raw: userMsg}
}

// whisperChatAsAssistant 以指定助手身份发起对话（微信回调专用内部入口，不进 Wails 绑定面）：
// 复用 WhisperChatWithSearch 的搜索意图判断；assistantName 经 whisperChat 在
// WhisperChat 的 LockTurn 持锁窗口内写入 orch.AssistantName——多助手同人格共享
// orchestrator 时，锁内赋值保证每个回合用自己助手的名字且无数据竞争。
func (a *whisperState) whisperChatAsAssistant(userMsg, personalityID, assistantName string, thinking bool) (map[string]interface{}, error) {
	return a.whisperChatWithSearch(userMsg, personalityID, assistantName, thinking, false)
}

// searchTriggers 搜索意图触发词（v4.9.1 收窄：动词锚定 + 硬时效词）。
// 教训：旧表含「什么/怎么/为什么/在哪/什么时候/介绍一下/告诉我/帮我找/
// 最新/最近/实时」等对话高频词（朴素子串匹配），日常对话大量误触发——
// 网页摘要被灌进角色回复，是「答非所问」的主力来源之一。纪律：宁漏勿误
// ——漏了走对话管道无害，误了把无关网页注入回复必然劣化。
var searchTriggers = []string{
	// 显式命令（动词锚定）
	"帮我搜索", "帮我搜", "帮我查", "搜一下", "查一下", "查查", "上网查", "搜索",
	// 硬时效信息（对话闲聊里几乎不出现，误触发率低）
	"天气", "股价", "汇率", "金价", "油价", "比分", "新闻",
}

func shouldSearchWeb(msg string) bool {
	for _, t := range searchTriggers {
		if len(msg) >= len(t) {
			for i := 0; i <= len(msg)-len(t); i++ {
				if msg[i:i+len(t)] == t {
					return true
				}
			}
		}
	}
	return false
}

// isIdentityQuestion 识别「问助手自己」的问题（你是谁/你叫什么/你会什么…）。
// 这类问题联网搜索必错：搜索引擎返回的是无关词条，注入后既拖慢回复又把
// 网页英文片段混进角色扮演回复（微信实测：夹杂随机字母的根因）。
func isIdentityQuestion(msg string) bool {
	for _, re := range identityPatterns {
		if re.MatchString(msg) {
			return true
		}
	}
	return false
}

var identityPatterns = []*regexp.Regexp{
	regexp.MustCompile(`你(是谁|叫什么|叫啥|会什么|能干什么|能做什么|会干什么|是什么)`),
	regexp.MustCompile(`(介绍|说说|讲讲)(一下|下)?你(自己)?`),
}

// ─── 虚拟助手 CRUD ────────────────────────────────────────────

func (a *whisperState) WhisperAssistantList() []assistant.Assistant {
	if a.assistantMgr == nil {
		return nil
	}
	return a.assistantMgr.List()
}

func (a *whisperState) WhisperAssistantSave(ast assistant.Assistant) error {
	if a.assistantMgr == nil {
		return fmt.Errorf("not ready")
	}
	// 防御：拒绝保存脱敏 Token（含 * 说明是占位回显，会致 getUpdates 认证失败）
	if strings.Contains(ast.WxToken, "*") {
		return fmt.Errorf("Token 无效（不能包含 *），请重新扫码绑定")
	}
	existing := a.assistantMgr.Get(ast.ID)
	if existing != nil {
		// 防御：空 token / 空绑定用户保留现值——「启停切换只传 id+enabled」的
		// 部分保存不得清空扫码凭据（清空后 getUpdates 认证失败，需重新扫码）。
		if ast.WxToken == "" {
			ast.WxToken = existing.WxToken
		}
		if ast.WxUserID == "" {
			ast.WxUserID = existing.WxUserID
		}
		if err := a.assistantMgr.Update(ast.ID, ast); err != nil {
			return err
		}
		a.stopAssistantWx(ast.ID)
		if ast.Enabled && ast.WxToken != "" {
			a.startAssistantWx(ast)
		}
		return nil
	}
	if err := a.assistantMgr.Add(ast); err != nil {
		return err
	}
	if ast.Enabled && ast.WxToken != "" {
		a.startAssistantWx(ast)
	}
	return nil
}

func (a *whisperState) WhisperAssistantDelete(id string) error {
	if a.assistantMgr == nil {
		return fmt.Errorf("not ready")
	}
	a.stopAssistantWx(id)
	return a.assistantMgr.Delete(id)
}

// WhisperWeixinGetQR 获取微信扫码登录二维码
func (a *whisperState) WhisperWeixinGetQR() (map[string]interface{}, error) {
	qr, err := weixin.GetQRCode()
	if err != nil {
		return nil, err
	}
	// 将微信扫码链接生成为二维码图片（用 qrcode_img_content，不是 qrcode 会话token）
	png, err := qrcode.Encode(qr.QrcodeImgContent, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("生成二维码图片失败: %w", err)
	}
	imageUrl := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	return map[string]interface{}{
		"qrcode":   qr.Qrcode,
		"imageUrl": imageUrl,
	}, nil
}

// WhisperWeixinQRStatus 轮询二维码状态
func (a *whisperState) WhisperWeixinQRStatus(qrcode string) (map[string]interface{}, error) {
	status, err := weixin.PollQRStatus(qrcode)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"status": status.Status,
	}
	if status.RedirectHost != "" {
		result["redirectHost"] = status.RedirectHost
	}
	if status.VerifyCode != "" {
		result["verifyCode"] = status.VerifyCode
	}
	if status.BotToken != "" {
		result["botToken"] = status.BotToken
		result["botId"] = status.ILinkBotID
		result["baseUrl"] = status.BaseURL
		result["userId"] = status.ILinkUserID
	}
	return result, nil
}

// WhisperWeixinQRStatusWithCode 带手机配对码轮询二维码状态（need_verifycode 状态时使用）
func (a *whisperState) WhisperWeixinQRStatusWithCode(qrcode, verifyCode string) (map[string]interface{}, error) {
	status, err := weixin.PollQRStatusWithCode(qrcode, verifyCode)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"status": status.Status,
	}
	if status.RedirectHost != "" {
		result["redirectHost"] = status.RedirectHost
	}
	if status.BotToken != "" {
		result["botToken"] = status.BotToken
		result["botId"] = status.ILinkBotID
		result["baseUrl"] = status.BaseURL
		result["userId"] = status.ILinkUserID
	}
	return result, nil
}

func (a *whisperState) WhisperWeixinStatus() []map[string]interface{} {
	result := []map[string]interface{}{}
	if a.assistantMgr == nil {
		return result
	}
	for _, ast := range a.assistantMgr.List() {
		item := map[string]interface{}{
			"id": ast.ID, "name": ast.Name, "personalityId": ast.PersonalityID,
			"enabled": ast.Enabled, "hasToken": ast.WxToken != "", "wxRunning": false,
		}
		a.weixinMu.Lock()
		if srv, ok := a.weixinServers[ast.ID]; ok {
			item["wxRunning"] = srv.IsRunning() && !srv.SessionExpired()
			item["wxSessionExpired"] = srv.SessionExpired()
		}
		a.weixinMu.Unlock()
		result = append(result, item)
	}
	return result
}

// healImplicitBirthday 众神殿刀2 存量自愈：旧版初始化把 nil 生日写成首聊时刻
// （orchestrator PreLLMTurn 旧逻辑），存量会话 6/6 ackemBirthday==firstMetDate，
// 首聊日周年必注入假事实「我的生日」。等值判定天然幂等：非 gaea 清 nil（永不
// 注入），gaea 改为内置生日 06-11。只动等值行，用户显式配置的生日不受影响；
// 首次动刀前对 hermes.db 做一次性文件级备份（主库文件快照，WAL 尾部增量不在
// 内——自愈本身等值门控幂等，备份只是兜底保险）。
func (a *whisperState) healImplicitBirthday(orch *whisper.Orchestrator) {
	st := orch.State
	if st.AckemBirthday == nil || st.FirstMetDate == nil || !st.AckemBirthday.Equal(*st.FirstMetDate) {
		return
	}
	dbPath := db.DatabasePath(orch.DataRoot)
	bakPath := dbPath + ".bak-birthday-heal"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		if raw, rerr := os.ReadFile(dbPath); rerr == nil {
			if werr := os.WriteFile(bakPath, raw, 0o600); werr == nil {
				slog.Info("[whisper] 生日自愈前已备份 hermes.db", "backup", bakPath)
			}
		}
	}
	if orch.Preset.ID == "gaea" {
		if bd, err := whisper.ParseAckemBirthdayMMDD(whisper.AckemBirthdayMMDD, *st.FirstMetDate); err == nil {
			st.AckemBirthday = &bd
		} else {
			slog.Warn("[whisper] gaea 内置生日解析失败，清空隐式生日", "sessionID", orch.SessionID, "error", err)
			st.AckemBirthday = nil
		}
	} else {
		st.AckemBirthday = nil
	}
	orch.State = st
	if err := a.persistStateSync(orch); err != nil {
		slog.Warn("[whisper] 生日自愈落库失败（内存已修，下轮重试）", "sessionID", orch.SessionID, "error", err)
		return
	}
	slog.Info("[whisper] 隐式生日自愈完成", "sessionID", orch.SessionID, "isGaea", orch.Preset.ID == "gaea")
}

// transientHistoryMarkers 已知动态包裹标记（众神殿刀4）：命中任一的历史行
// 视为「中间形态」脏行，重启回灌时跳过（不做模糊清洗，避免误删用户原文）。
// 「【本轮上下文】」为防御性——该前缀只存在于 llmUserMsg，chat_history 本不会存。
var transientHistoryMarkers = []string{
	"【本轮上下文】",
	"[以下是关于此问题的实时搜索结果",
	"[联网搜索已降级",
	"[联网搜索已完成",
	"[联网搜索暂不可用",
}

func containsTransientMarker(s string) bool {
	for _, m := range transientHistoryMarkers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

// warmPromptHistory 众神殿刀4 重启回灌：PromptHistory 只在内存，进程重启后
// HistoryInMessages 模式下模型只看 system+当前轮（chat_history 只回填 WM，而
// WM 近期块在该模式下不进 messages），重启即失忆。从 hermes chat_history 取
// 最近 ≤HistoryKeepTurns 轮「无动态包裹标记」的干净行回灌 RecordPromptTurn
// （非搜索轮 chat_history 的 userText 即原文，无需跨库映射）。接受一次 KV
// 前缀缓存 miss（重启本来就要重新 prefill）。B3 净行刀后 RecordPromptTurn
// 本身即记原句，此处与主链口径完全一致（旧「实发串」契约已退役）。
func (a *whisperState) warmPromptHistory(orch *whisper.Orchestrator) {
	if orch.DataRoot == "" || len(orch.PromptHistory) > 0 {
		return
	}
	rows, err := repos.LoadChatHistoryFromDB(orch.DataRoot, orch.SessionID)
	if err != nil || len(rows) == 0 {
		return
	}
	type histRow struct {
		turnIndex               int
		userText, assistantText string
	}
	var clean []histRow
	for _, r := range rows {
		ti, okTi := r["turnIndex"].(float64)
		ut, _ := r["userText"].(string)
		at, _ := r["assistantText"].(string)
		if !okTi {
			continue
		}
		if src, _ := r["source"].(string); src != "" && src != whisper.ExchangeSourceRaw {
			continue // 来源契约：未知形态不回灌（raw 之外的标记留给未来契约扩展）
		}
		if containsTransientMarker(ut) || containsTransientMarker(at) {
			continue
		}
		clean = append(clean, histRow{turnIndex: int(ti), userText: ut, assistantText: at})
	}
	if len(clean) > whisper.HistoryKeepTurns {
		clean = clean[len(clean)-whisper.HistoryKeepTurns:]
	}
	for _, h := range clean {
		orch.RecordPromptTurn(h.turnIndex, h.userText, h.assistantText)
	}
	if len(clean) > 0 {
		slog.Info("[whisper] 重启历史回灌完成", "sessionID", orch.SessionID, "turns", len(clean))
	}
}

func restoreWhisperState(orch *whisper.Orchestrator) error {
	if orch.DataRoot == "" {
		return nil
	}
	if _, err := db.GetDatabase(orch.DataRoot); err != nil {
		return fmt.Errorf("初始化 hermes.db 失败: %w", err)
	}
	state, err := repos.LoadCompanionStateFromDB(orch.DataRoot, orch.SessionID)
	if err != nil {
		return err
	}
	if state != nil {
		personality := orch.State.Personality
		orch.State = *state
		orch.State.Personality = personality
	}
	rows, err := repos.LoadChatHistoryFromDB(orch.DataRoot, orch.SessionID)
	if err != nil {
		return err
	}
	for _, r := range rows {
		ti, okTi := r["turnIndex"].(float64)
		ut, _ := r["userText"].(string)
		at, _ := r["assistantText"].(string)
		src, _ := r["source"].(string)
		if !okTi {
			slog.Warn("[whisper] restore: turnIndex type assertion failed", "row", r)
			continue
		}
		orch.WM.Push(orch.SessionID, whisper.Exchange{
			TurnIndex: int(ti), UserText: ut, AssistantText: at, Source: src,
		})
	}
	// 角色记忆隔离：事实/情节按会话加载，知识图谱按归属事实过滤——
	// 每个角色只恢复自己的记忆，不串其他角色
	facts := repos.LoadFactsFromDBForSession(orch.DataRoot, orch.SessionID)
	if len(facts) > 0 {
		orch.FactStore.Restore(facts)
	}
	ownFactIDs := make(map[string]bool, len(facts))
	for _, f := range facts {
		ownFactIDs[f.ID] = true
	}
	if eps, err := repos.LoadEpisodesFromDBForSession(orch.DataRoot, orch.SessionID); err == nil && len(eps) > 0 {
		for _, ep := range eps {
			orch.EpisodicStore.Add(ep)
		}
	}
	if tris, err := repos.LoadTriplesFromDB(orch.DataRoot); err == nil && len(tris) > 0 {
		var mine []whisper.Triple
		for _, t := range tris {
			if tripleOwnedBySession(t, ownFactIDs) {
				mine = append(mine, t)
			}
		}
		if len(mine) > 0 {
			orch.KG.Restore(mine)
		}
	}
	return nil
}

// persistMu 全局持久化互斥（T7-1.1 跨会话正确性）：persistFactsToDB/
// persistEpisodesToDB/persistKGToDB 都是「全表读 → 内存合并 → 全表替换」，
// SQLite 单连接只串行化单条语句，「读→改→写」跨语句交错会让后完成者
// 用旧快照覆盖先完成者的写入（多会话并发聊天时记忆互相丢失）。
// 持久化整体持锁，串行化所有会话的落库。
var persistMu sync.Mutex

// persistStateSync 同步持久化会话状态（T7-1.1 ③）：先在回合锁内取状态快照
// （不阻塞下一轮对话），再在全局单写锁（persistMu）内落库，串行化所有会话的
// 「全表读→内存合并→全表替换」，避免多会话并发互相覆盖（H3）。返回落库错误。
// 供 persistStateAsync（异步）与 drainAndPersistAll（Shutdown 末轮）共用，
// 测试也可直接调用做确定性断言。
func (a *whisperState) persistStateSync(orch *whisper.Orchestrator) error {
	orch.LockTurn()
	stateSnap := whisper.CloneFullState(orch.State)
	orch.UnlockTurn()

	persistMu.Lock()
	defer persistMu.Unlock()
	return persistWhisperStateWithSnapshot(orch, stateSnap)
}

// persistStateAsync 异步持久化会话状态（fire-and-forget，T6-5.3）：
// 落库失败与 panic 统一计入 WriteErrors 计数并记录日志，不再静默丢弃。
func (a *whisperState) persistStateAsync(orch *whisper.Orchestrator) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("whisper: persist state goroutine panic recovered", "panic", r)
			a.recordMemoryWriteError(orch.SessionID, "persist_panic", fmt.Errorf("persist state panic: %v", r))
		}
	}()
	if err := a.persistStateSync(orch); err != nil {
		a.recordMemoryWriteError(orch.SessionID, "persist", err)
	}
}

// drainAndPersistAll 末轮落库（T7-1.1 ③）：Shutdown 时先 drain 全部异步记忆
// 写入队列（保证末轮 LLM 抽取的事实已进入内存 FactStore，H4），再按全局单写锁
// 串行持久化所有会话，避免进程退出时末轮记忆/状态丢失。
func (a *whisperState) drainAndPersistAll() {
	// V2 已预推进但尚未收到 response.done 的回合属于未提交状态；退出时先
	// 回滚，避免把幽灵情绪/轮次写进 companion_state。
	a.rollbackVoiceV2Prepared()
	whisper.DrainAllMemoryWriteJobs()

	whisperSessionsMu.RLock()
	orchs := make([]*whisper.Orchestrator, 0, len(whisperSessions))
	for _, o := range whisperSessions {
		orchs = append(orchs, o)
	}
	whisperSessionsMu.RUnlock()

	for _, o := range orchs {
		if o.DataRoot == "" {
			continue
		}
		if err := a.persistStateSync(o); err != nil {
			slog.Error("[whisper] shutdown 末轮落库失败", "sessionID", o.SessionID, "error", err)
		}
	}
}

// persistWhisperState 持久化会话状态（同伴状态/聊天历史/事实/情节/图谱）。
// 任一落库失败返回合并错误（不中断后续写回），由调用方记录与计数。
// 测试直接调用本函数（以 orch.State 当前值落库）；异步路径请用
// persistWhisperStateWithSnapshot 避免与主流程竞争。
func persistWhisperState(orch *whisper.Orchestrator) error {
	return persistWhisperStateWithSnapshot(orch, orch.State)
}

// persistWhisperStateWithSnapshot 与 persistWhisperState 同语义，
// 但同伴状态取调用方传入的快照（T7-1.1：异步持久化在回合锁外执行，
// orch.State 可能已被新回合改写，必须用回合锁内快照）。
func persistWhisperStateWithSnapshot(orch *whisper.Orchestrator, stateSnap whisper.FullState) error {
	if orch.DataRoot == "" {
		return nil
	}
	var errs []error
	if err := repos.SaveCompanionStateToDB(orch.DataRoot, orch.SessionID, stateSnap); err != nil {
		slog.Error("[whisper] 同伴状态落库失败", "sessionID", orch.SessionID, "error", err)
		errs = append(errs, err)
	}
	exchanges := orch.WM.GetAll(orch.SessionID)
	if len(exchanges) > 0 {
		rows := make([]map[string]interface{}, len(exchanges))
		for i, e := range exchanges {
			rows[i] = map[string]interface{}{
				"turnIndex": e.TurnIndex, "userText": e.UserText, "assistantText": e.AssistantText,
			}
			// 来源字段契约：只给有标记的行写 source（空标记旧行零改写，
			// 重落库后仍无该键，遗留形态可审计）
			if e.Source != "" {
				rows[i]["source"] = e.Source
			}
		}
		if err := repos.SaveChatHistoryToDB(orch.DataRoot, orch.SessionID, rows); err != nil {
			slog.Error("[whisper] 聊天历史落库失败", "sessionID", orch.SessionID, "error", err)
			errs = append(errs, err)
		}
	}
	// 记忆贯通：事实/情节/图谱均按会话合并写回，其他角色的记忆不被覆盖
	if err := persistFactsToDB(orch); err != nil {
		errs = append(errs, err)
	}
	if err := persistEpisodesToDB(orch); err != nil {
		errs = append(errs, err)
	}
	if err := persistKGToDB(orch); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// persistFactsToDB 事实合并写回：本会话 ID 用内存版替换（含退役态），其他会话保留 DB 版。
// 落库/FTS 重建失败均记录日志并返回错误（T6-5.3 不再吞错）。
func persistFactsToDB(orch *whisper.Orchestrator) error {
	all := orch.FactStore.ListAll()
	mine := make(map[string]bool, len(all))
	for _, f := range all {
		mine[f.ID] = true
	}
	dbFacts := repos.LoadFactsFromDB(orch.DataRoot)
	merged := make([]whisper.MemoryFact, 0, len(dbFacts)+len(all))
	for _, f := range dbFacts {
		if mine[f.ID] {
			continue // 本会话事实由内存版提供
		}
		merged = append(merged, f)
	}
	for _, f := range all {
		merged = append(merged, f.MemoryFact)
	}
	var errs []error
	if err := repos.ReplaceFactsInDB(orch.DataRoot, merged); err != nil {
		slog.Error("[whisper] 事实落库失败", "sessionID", orch.SessionID, "error", err)
		errs = append(errs, err)
	}
	// FTS 全文索引重建：事实写回后让 memory_facts_fts 与主表同步
	if err := repos.RebuildFactsFTS(orch.DataRoot); err != nil {
		slog.Error("[whisper] FTS 事实索引重建失败", "sessionID", orch.SessionID, "error", err)
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// persistEpisodesToDB 情节按会话合并写回：本会话以内存为准，其他会话保留 DB 版。
// 落库/FTS 重建失败均记录日志并返回错误（T6-5.3 不再吞错）。
func persistEpisodesToDB(orch *whisper.Orchestrator) error {
	eps := orch.EpisodicStore.ListAll()
	dbEps, err := repos.LoadEpisodesFromDB(orch.DataRoot)
	merged := make([]whisper.Episode, 0, len(dbEps)+len(eps))
	if err == nil {
		for _, e := range dbEps {
			if e.SourceSessionID == orch.SessionID {
				continue // 本会话情节由内存版提供
			}
			merged = append(merged, e)
		}
	}
	merged = append(merged, eps...)
	var errs []error
	if len(merged) > 0 {
		if err := repos.ReplaceEpisodesInDB(orch.DataRoot, merged); err != nil {
			slog.Error("[whisper] 情节落库失败", "sessionID", orch.SessionID, "error", err)
			errs = append(errs, err)
		}
		// FTS 全文索引重建：情节写回后让 episodes_fts 与主表同步
		if err := repos.RebuildEpisodesFTS(orch.DataRoot); err != nil {
			slog.Error("[whisper] FTS 情节索引重建失败", "sessionID", orch.SessionID, "error", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// persistKGToDB 知识图谱按归属合并写回：三元组从 facts 派生，
// 归属本会话（source_fact_ids 命中本会话事实）的以内存为准，其余保留 DB 版。
// 落库失败记录日志并返回错误（T6-5.3 不再吞错）。
func persistKGToDB(orch *whisper.Orchestrator) error {
	tris := orch.KG.ListAll()
	ownFactIDs := make(map[string]bool)
	for _, f := range orch.FactStore.ListAll() {
		ownFactIDs[f.ID] = true
	}
	dbTris, err := repos.LoadTriplesFromDB(orch.DataRoot)
	merged := make([]whisper.Triple, 0, len(dbTris)+len(tris))
	if err == nil {
		for _, t := range dbTris {
			if tripleOwnedBySession(t, ownFactIDs) {
				continue // 本会话三元组由内存版提供
			}
			merged = append(merged, t)
		}
	}
	merged = append(merged, tris...)
	if len(merged) > 0 {
		if err := repos.ReplaceTriplesInDB(orch.DataRoot, merged); err != nil {
			slog.Error("[whisper] 图谱落库失败", "sessionID", orch.SessionID, "error", err)
			return err
		}
	}
	return nil
}

// tripleOwnedBySession 判断三元组是否归属某会话（source_fact_ids 命中该会话任一事实）。
// 无来源事实的三元组视为全局遗留，不归属任何会话。
func tripleOwnedBySession(t whisper.Triple, factIDs map[string]bool) bool {
	if len(t.SourceFactIDs) == 0 {
		return false
	}
	for _, id := range t.SourceFactIDs {
		if factIDs[id] {
			return true
		}
	}
	return false
}
func buildRecentExchanges(orch *whisper.Orchestrator) []whisper.ExchangePair {
	exs := orch.WM.GetAll(orch.SessionID)
	pairs := make([]whisper.ExchangePair, 0, len(exs))
	for _, e := range exs {
		pairs = append(pairs, whisper.ExchangePair{User: e.UserText, Assistant: e.AssistantText})
	}
	return pairs
}
