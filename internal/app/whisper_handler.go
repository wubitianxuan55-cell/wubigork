package app

import (
	"fmt"
	"log/slog"

	"github.com/wubigork/wubigork/internal/modelengine"
	"github.com/wubigork/wubigork/internal/whisper"
	"github.com/wubigork/wubigork/internal/whisper/db"
	"github.com/wubigork/wubigork/internal/whisper/db/repos"
)

// Chat 实现 whisper.LlmClient 接口（接入 wubigrok 模型中心）
func (a *App) Chat(systemPrompt, userPrompt string) (string, error) {
	return a.client.ChatSimpleStream(a.ctx, "", systemPrompt, userPrompt)
}

var whisperSessions = map[string]*whisper.Orchestrator{}

func (a *App) getOrCreateOrch(personalityID string) *whisper.Orchestrator {
	sessionID := "whisper_" + personalityID
	if orch, ok := whisperSessions[sessionID]; ok {
		return orch
	}
	preset := whisper.GetPreset(personalityID)
	if preset == nil {
		preset = whisper.GetPreset("deredere")
	}
	if preset == nil {
		preset = &whisper.PersonalityPresets[0]
	}
	orch := whisper.NewOrchestrator(sessionID, *preset)
	orch.DataRoot = a.whisperDataRoot
	whisperSessions[sessionID] = orch
	_ = restoreWhisperState(orch)
	return orch
}

func (a *App) WhisperGetPersonalities() []whisper.PersonalityPreset {
	return whisper.PersonalityPresets
}

func (a *App) WhisperChat(userMsg string, personalityID string) (result map[string]interface{}, err error) {
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

	origEngine := ""
	if orch.EngineID != "" {
		origEngine = a.client.ActiveEngineID()
		slog.Info("[whisper] engine switch", "from", origEngine, "to", orch.EngineID)
		if orch.EngineID != origEngine {
			a.client.SetActiveEngine(orch.EngineID)
		}
	}

	model := orch.ModelName
	if a.client == nil {
		slog.Error("[whisper] client is nil")
		return nil, fmt.Errorf("model client not initialized")
	}
	slog.Info("[whisper] calling LLM", "engine", orch.EngineID, "model", model)
	reply, callErr := a.client.ChatSimpleStream(a.ctx, model, systemPrompt, userMsg)

	if orch.EngineID != "" && orch.EngineID != origEngine {
		a.client.SetActiveEngine(origEngine)
	}
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
		whisper.EnqueueMemoryWrite(a, whisper.MemoryWritePayload{
			SessionID: orch.SessionID, TurnIndex: turns,
			UserMsg: userMsg, AssistantText: reply,
			L1: l1Snap, L2: l2Snap,
			FactStore: orch.FactStore, TotalTurns: turns, KG: orch.KG,
			EpisodicStore: nil, AdultMode: orch.AdultMode,
		})
	}()

	go persistWhisperState(orch)

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
		facts = append(facts, map[string]interface{}{
			"id": f.ID, "domain": f.Domain, "subject": f.Subject,
			"summary": f.Summary, "weight": f.Weight,
			"confidence": f.Confidence, "createdAt": f.CreatedAt.Format("2006-01-02"),
			"tier": f.RawTier,
		})
	}
	return facts
}

func (a *App) WhisperGetState(personalityID string) map[string]interface{} {
	orch, ok := whisperSessions["whisper_"+personalityID]
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

func (a *App) WhisperSetEngine(engineID string) error {
	for _, orch := range whisperSessions {
		orch.EngineID = engineID
	}
	return nil
}

func (a *App) WhisperGetEngine() string {
	for _, orch := range whisperSessions {
		if orch.EngineID != "" {
			return orch.EngineID
		}
	}
	return a.client.ActiveEngineID()
}

func (a *App) WhisperSetModel(engineID, modelName string) error {
	for _, orch := range whisperSessions {
		orch.EngineID = engineID
		orch.ModelName = modelName
	}
	return nil
}

func (a *App) WhisperGetModel() string {
	for _, orch := range whisperSessions {
		if orch.ModelName != "" {
			return orch.ModelName
		}
	}
	return a.GetActiveModel()
}

func (a *App) WhisperSetImageModel(modelName string) error {
	for _, orch := range whisperSessions {
		orch.ImageModelName = modelName
	}
	return nil
}

func (a *App) WhisperGetImageModel() string {
	for _, orch := range whisperSessions {
		if orch.ImageModelName != "" {
			return orch.ImageModelName
		}
	}
	return ""
}

func (a *App) WhisperGetConfig() map[string]interface{} {
	return map[string]interface{}{
		"engine":       a.WhisperGetEngine(),
		"model":        a.WhisperGetModel(),
		"imageModel":   a.WhisperGetImageModel(),
		"engines":      a.GetEngines(),
		"activeEngine": a.GetActiveEngine(),
		"activeModel":  a.GetActiveModel(),
	}
}

func (a *App) WhisperGetEngines() []modelengine.EngineConfig {
	return a.GetEngines()
}

func (a *App) WhisperClearSession(personalityID string) error {
	sessionID := "whisper_" + personalityID
	delete(whisperSessions, sessionID)
	return nil
}

func (a *App) WhisperSetAdultMode(personalityID string, enabled bool) error {
	orch, ok := whisperSessions["whisper_"+personalityID]
	if !ok {
		return nil
	}
	orch.AdultMode = enabled
	return nil
}

func restoreWhisperState(orch *whisper.Orchestrator) error {
	if orch.DataRoot == "" {
		return nil
	}
	db.GetDatabase(orch.DataRoot)
	state, err := repos.LoadCompanionStateFromDB(orch.DataRoot, orch.SessionID)
	if err != nil || state == nil {
		return err
	}
	personality := orch.State.Personality
	orch.State = *state
	orch.State.Personality = personality
	rows, err := repos.LoadChatHistoryFromDB(orch.DataRoot, orch.SessionID)
	if err != nil || rows == nil {
		return err
	}
	for _, r := range rows {
		ti, _ := r["turnIndex"].(float64)
		ut, _ := r["userText"].(string)
		at, _ := r["assistantText"].(string)
		orch.WM.Push(orch.SessionID, whisper.Exchange{
			TurnIndex: int(ti), UserText: ut, AssistantText: at,
		})
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
}
