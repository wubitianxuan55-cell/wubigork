package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
	"github.com/gaea/gaea/internal/whisper/db/repos"
	qrcode "github.com/skip2/go-qrcode"
)

// Chat 实现 whisper.LlmClient 接口（接入 gaea 模型中心）
func (a *whisperState) Chat(systemPrompt, userPrompt string) (string, error) {
	eng, model, _ := a.routeModel("chat") // 2.x 聊天/轻语合并：统一走 chat 路由
	return a.client.ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{EngineID: eng})
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
		// 全局角色库优先：库内角色（含可编辑的内置人格）直接作为聊天人格
		if a.charLib != nil {
			if c, err := a.charLib.Get(personalityID); err == nil && c != nil && !c.Hidden && c.ChatEnabled {
				preset = c.ToPreset()
			}
		}
		if preset == nil {
			preset = whisper.GetPreset("gaea")
		}
		if preset == nil {
			preset = &whisper.PersonalityPresets[0]
		}
		// 小说角色导入的自定义人格：助手记录带 voiceGuide 时覆盖预设
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
	orch := whisper.NewOrchestrator(sessionID, *preset)
	orch.DataRoot = a.whisperDataRoot
	// FTS 全文检索回调：buildTierBBlock 经此走 hermes.db FTS5 索引（含 LIKE 中文降级）
	orch.FTSSearch = func(query string, limit int) []string {
		ids, err := repos.SearchFactIDsFTS(a.whisperDataRoot, query, limit)
		if err != nil {
			return nil
		}
		return ids
	}

	whisperSessionsMu.Lock()
	whisperSessions[sessionID] = orch
	whisperSessionsMu.Unlock()

	if err := restoreWhisperState(orch); err != nil {
		slog.Error("[whisper] 会话状态恢复失败（继续用全新状态）", "sessionID", orch.SessionID, "error", err)
	}
	return orch
}

func (a *whisperState) WhisperGetPersonalities() []whisper.PersonalityPreset {
	// 统一人格列表 = 角色库中可聊天角色（内置人格已种子化进库，可在库内编辑）
	if a.charLib != nil {
		items := a.charLib.ListChatEnabled()
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
	return whisper.PersonalityPresets
}

func (a *whisperState) WhisperChat(userMsg string, personalityID string, thinking bool) (result map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[whisper] PANIC", "panic", fmt.Sprintf("%v", r))
			result = nil
			err = fmt.Errorf("whisper panic: %v", r)
		}
	}()

	msgPreview := userMsg
	if len(userMsg) > 80 {
		msgPreview = userMsg[:80]
	}
	slog.Info("[whisper] WhisperChat start", "userMsg", msgPreview, "personality", personalityID)

	orch := a.getOrCreateOrch(personalityID)

	// T7-1.1 会话并发安全：同一角色 GUI/微信/语音三入口串行化，
	// 防止状态机并发撕裂（TotalTurns/情绪/成人 FSM）。
	orch.LockTurn()
	defer orch.UnlockTurn()

	turnPlan := whisper.BuildTurnPlanFromRules(userMsg, whisper.BuildTurnPlanRulePriors(userMsg))
	slog.Info("[whisper] TurnPlan", "routing", turnPlan.Routing, "goal", turnPlan.Goal)

	preResult := orch.PreLLMTurn(userMsg)
	slog.Info("[whisper] PreLLMTurn done", "hasSystemPrompt", len(preResult.SystemPrompt) > 0, "eventType", preResult.Event.Type)

	systemPrompt := preResult.SystemPrompt
	if turnPlan.Routing == whisper.RouteStructuredChat && turnPlan.FormatHint != "" {
		systemPrompt = systemPrompt + "\n\n【本轮格式要求】\n" + turnPlan.FormatHint
	}

	// 功能级绑定：聊天/轻语合并后统一用 chat 绑定（未绑定则沿用 orch/全局）
	featEng, featModel := a.featureModel("chat")
	engine := orch.EngineID
	if featEng != "" {
		engine = featEng
	}
	model := orch.ModelName
	if featModel != "" {
		model = featModel
	}
	// per-call 引擎覆盖：不影响全局激活引擎，多会话并发安全
	if a.client == nil {
		slog.Error("[whisper] client is nil")
		return nil, fmt.Errorf("model client not initialized")
	}
	// S1.5-B play 内容护栏：persona_lock 在人格一致性参数（dims/voiceGuide）
	// 注入时追加人格锁定段（防系统层覆盖人格段）并锁温度上限；
	// max_output_tokens 钳制输出。未配置 = 零值 = 请求与现状逐字节一致。
	opts := ai.ChatSimpleOptions{EngineID: engine, EnableThinking: thinking}
	systemPrompt = applyWhisperGuardrails(&opts, systemPrompt, orch.Preset, playGuardrails())
	slog.Info("[whisper] calling LLM", "engine", engine, "model", model)
	reply, reasoning, callErr := a.client.ChatSimpleStreamDetailed(a.ctx, model, systemPrompt, userMsg, opts)
	if callErr != nil {
		slog.Error("[whisper] LLM call failed", "error", callErr)
		return nil, callErr
	}
	slog.Info("[whisper] LLM reply", "len", len(reply))

	sentences := whisper.SplitIntoSentences(reply)
	if len(sentences) == 0 {
		sentences = []string{reply}
	}

	orch.WM.Push(orch.SessionID, whisper.Exchange{
		TurnIndex: orch.State.Counters.TotalTurns, UserText: userMsg, AssistantText: reply,
	})

	l1Snap := orch.State.Relationship
	l2Snap := orch.State.Emotion
	turns := orch.State.Counters.TotalTurns

	whisper.FinalizeTurn(orch, whisper.PostTurnContext{
		SessionID: orch.SessionID, TurnIndex: turns,
		UserMsg: userMsg, AssistantText: reply,
		Event: preResult.Event, AdultMode: orch.AdultMode,
	})
	// 轮次追踪持久化（按会话归属，供角色库「追踪」页查看）
	if err := repos.AppendTurnTraceToDB(orch.DataRoot, orch.SessionID, preResult.Trace); err != nil {
		slog.Error("[whisper] 轮次追踪落库失败", "sessionID", orch.SessionID, "error", err)
	}

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
			AdultMode:       orch.AdultMode,
		}, a.recordMemoryWriteError)
	}()

	go a.persistStateAsync(orch)

	slog.Info("[whisper] WhisperChat done", "replyLen", len(reply), "emotion", orch.State.Emotion.PrimaryLabel)
	return map[string]interface{}{
		"reply":        reply,
		"reasoning":    reasoning,
		"sentences":    sentences,
		"turnPlan":     turnPlan,
		"stage":        string(orch.State.Relationship.Stage),
		"emotion":      orch.State.Emotion.PrimaryLabel,
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
	whisperSessionsMu.RLock()
	defer whisperSessionsMu.RUnlock()
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
	return a.client.ActiveEngineID()
}

func (a *whisperState) WhisperSetModel(engineID, modelName string) error {
	whisperSessionsMu.RLock()
	defer whisperSessionsMu.RUnlock()
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
	whisperSessionsMu.RLock()
	defer whisperSessionsMu.RUnlock()
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

// WhisperWebSearch 执行上网查询（只读）
func (a *whisperState) WhisperWebSearch(query string) (map[string]interface{}, error) {
	slog.Info("[whisper] web search", "query", query)
	result, err := whisper.WebSearch(query)
	if err != nil {
		return map[string]interface{}{
			"query":  query,
			"result": "",
			"error":  err.Error(),
		}, nil
	}
	return map[string]interface{}{
		"query":  query,
		"result": result,
	}, nil
}

// WhisperChatWithSearch 带搜索增强的对话：自动检测是否需要上网查询
func (a *whisperState) WhisperChatWithSearch(userMsg string, personalityID string, thinking, forceSearch bool) (map[string]interface{}, error) {
	// 检测搜索意图
	if forceSearch || shouldSearchWeb(userMsg) {
		slog.Info("[whisper] auto-search triggered", "msg", userMsg[:min(60, len(userMsg))])
		searchResult, err := whisper.WebSearch(userMsg)
		if err == nil && searchResult != "" {
			// 将搜索结果注入为增强的 userMsg
			enhancedMsg := fmt.Sprintf("%s\n\n[以下是关于此问题的实时搜索结果，请参考这些信息回答]\n%s", userMsg, searchResult)
			return a.WhisperChat(enhancedMsg, personalityID, thinking)
		}
	}
	return a.WhisperChat(userMsg, personalityID, thinking)
}

// searchTriggers 搜索意图触发词（精简版 — 去掉宽泛词，添加精确模式）
var searchTriggers = []string{
	// 显式命令
	"帮我搜索", "帮我搜", "帮我查", "搜一下", "查一下", "查查", "上网查",
	// 实时/时效信息
	"最新", "最近", "新闻", "天气", "股价", "汇率", "比赛", "实时", "今天天气",
	// 知识查询
	"是谁", "什么是", "多少钱", "在哪里", "什么时候", "为什么", "怎么",
	"如何", "介绍一下", "告诉我", "帮我找",
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
		if !okTi {
			slog.Warn("[whisper] restore: turnIndex type assertion failed", "row", r)
			continue
		}
		orch.WM.Push(orch.SessionID, whisper.Exchange{
			TurnIndex: int(ti), UserText: ut, AssistantText: at,
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
