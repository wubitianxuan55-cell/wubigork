package app

import (
	"encoding/base64"
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

	preset := whisper.GetPreset(personalityID)
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

	_ = restoreWhisperState(orch)
	return orch
}

func (a *whisperState) WhisperGetPersonalities() []whisper.PersonalityPreset {
	return whisper.PersonalityPresets
}

func (a *whisperState) WhisperChat(userMsg string, personalityID string) (result map[string]interface{}, err error) {
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
	slog.Info("[whisper] calling LLM", "engine", engine, "model", model)
	reply, callErr := a.client.ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userMsg, ai.ChatSimpleOptions{EngineID: engine})
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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("whisper: memory write goroutine panic recovered", "panic", r)
			}
		}()
		whisper.EnqueueMemoryWrite(a, whisper.MemoryWritePayload{
			SessionID: orch.SessionID, TurnIndex: turns,
			UserMsg: userMsg, AssistantText: reply,
			L1: l1Snap, L2: l2Snap,
			FactStore: orch.FactStore, TotalTurns: turns, KG: orch.KG,
			EpisodicStore:   orch.EpisodicStore,
			RecentExchanges: buildRecentExchanges(orch),
			AdultMode:       orch.AdultMode,
		})
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("whisper: persist state goroutine panic recovered", "panic", r)
			}
		}()
		persistWhisperState(orch)
	}()

	slog.Info("[whisper] WhisperChat done", "replyLen", len(reply), "emotion", orch.State.Emotion.PrimaryLabel)
	return map[string]interface{}{
		"reply":        reply,
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
	whisperSessionsMu.RLock()
	orch, ok := whisperSessions["whisper_"+personalityID]
	whisperSessionsMu.RUnlock()
	if !ok {
		return nil
	}
	return buildFactsList(orch.FactStore)
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
	whisperSessionsMu.RLock()
	orch, ok := whisperSessions["whisper_"+personalityID]
	whisperSessionsMu.RUnlock()
	if !ok {
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
		"totalTurns": state.Counters.TotalTurns,
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

func (a *whisperState) WhisperSetAdultMode(personalityID string, enabled bool) error {
	whisperSessionsMu.RLock()
	orch, ok := whisperSessions["whisper_"+personalityID]
	whisperSessionsMu.RUnlock()
	if !ok {
		return nil
	}
	// 私人非商用：成人内容始终开启，忽略开关参数，保留方法以兼容旧调用
	orch.AdultMode = true
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
func (a *whisperState) WhisperChatWithSearch(userMsg string, personalityID string) (map[string]interface{}, error) {
	// 检测搜索意图
	if shouldSearchWeb(userMsg) {
		slog.Info("[whisper] auto-search triggered", "msg", userMsg[:min(60, len(userMsg))])
		searchResult, err := whisper.WebSearch(userMsg)
		if err == nil && searchResult != "" {
			// 将搜索结果注入为增强的 userMsg
			enhancedMsg := fmt.Sprintf("%s\n\n[以下是关于此问题的实时搜索结果，请参考这些信息回答]\n%s", userMsg, searchResult)
			return a.WhisperChat(enhancedMsg, personalityID)
		}
	}
	return a.WhisperChat(userMsg, personalityID)
}

// searchTriggers 搜索意图触发词（精简版 — 去掉宽泛词，添加精确模式）
var searchTriggers = []string{
	// 显式命令
	"搜索", "查一下", "查查", "帮我查", "帮我搜", "搜一下", "上网查",
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
	db.GetDatabase(orch.DataRoot)
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
	// 记忆贯通：始终尝试恢复事实库/情节库/知识图谱（不受 state/rows 缺失影响）
	if facts := repos.LoadFactsFromDB(orch.DataRoot); len(facts) > 0 {
		orch.FactStore.Restore(facts)
	}
	if eps, err := repos.LoadEpisodesFromDB(orch.DataRoot); err == nil && len(eps) > 0 {
		for _, ep := range eps {
			orch.EpisodicStore.Add(ep)
		}
	}
	if tris, err := repos.LoadTriplesFromDB(orch.DataRoot); err == nil && len(tris) > 0 {
		orch.KG.Restore(tris)
	}
	return nil
}
func persistWhisperState(orch *whisper.Orchestrator) {
	if orch.DataRoot == "" {
		return
	}
	_ = repos.SaveCompanionStateToDB(orch.DataRoot, orch.SessionID, orch.State)
	exchanges := orch.WM.GetAll(orch.SessionID)
	if len(exchanges) > 0 {
		rows := make([]map[string]interface{}, len(exchanges))
		for i, e := range exchanges {
			rows[i] = map[string]interface{}{
				"turnIndex": e.TurnIndex, "userText": e.UserText, "assistantText": e.AssistantText,
			}
		}
		_ = repos.SaveChatHistoryToDB(orch.DataRoot, orch.SessionID, rows)
	}
	// 记忆贯通：事实库合并写回（本会话事实以内存为准，保留其他会话），情节/图谱全量写回
	persistFactsToDB(orch)
	persistEpisodesToDB(orch)
	persistKGToDB(orch)
}

// persistFactsToDB 事实合并写回：本会话 ID 用内存版替换（含退役态），其他会话保留 DB 版
func persistFactsToDB(orch *whisper.Orchestrator) {
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
	_ = repos.ReplaceFactsInDB(orch.DataRoot, merged)
	// FTS 全文索引重建：事实写回后让 memory_facts_fts 与主表同步
	_ = repos.RebuildFactsFTS(orch.DataRoot)
}

// persistEpisodesToDB 情节全量写回（每会话 store 均为全局快照 + 新增，冲突窗口极小）
func persistEpisodesToDB(orch *whisper.Orchestrator) {
	if eps := orch.EpisodicStore.ListAll(); len(eps) > 0 {
		_ = repos.ReplaceEpisodesInDB(orch.DataRoot, eps)
		// FTS 全文索引重建：情节写回后让 episodes_fts 与主表同步
		_ = repos.RebuildEpisodesFTS(orch.DataRoot)
	}
}
// persistKGToDB 知识图谱全量写回（三元组从 facts 派生，每会话 KG 为全局快照 + 增量）
func persistKGToDB(orch *whisper.Orchestrator) {
	if tris := orch.KG.ListAll(); len(tris) > 0 {
		_ = repos.ReplaceTriplesInDB(orch.DataRoot, tris)
	}
}
func buildRecentExchanges(orch *whisper.Orchestrator) []whisper.ExchangePair {
	exs := orch.WM.GetAll(orch.SessionID)
	pairs := make([]whisper.ExchangePair, 0, len(exs))
	for _, e := range exs {
		pairs = append(pairs, whisper.ExchangePair{User: e.UserText, Assistant: e.AssistantText})
	}
	return pairs
}
