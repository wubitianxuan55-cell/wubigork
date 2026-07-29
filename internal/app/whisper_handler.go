package app

import (
	"github.com/wubigork/wubigork/internal/whisper"
)

// Chat 实现 whisper.LlmClient 接口（接入 wubigrok 模型中心）
func (a *App) Chat(systemPrompt, userPrompt string) (string, error) {
	return a.client.ChatSimpleStream(a.ctx, "", systemPrompt, userPrompt)
}

var whisperSessions = map[string]*whisper.Orchestrator{}

func getOrCreateOrch(personalityID string) *whisper.Orchestrator {
	sessionID := "whisper_" + personalityID // 每人格独立会话
	if orch, ok := whisperSessions[sessionID]; ok {
		return orch
	}
	preset := whisper.GetPreset(personalityID)
	if preset == nil {
		preset = whisper.GetPreset("deredere")
	}
	if preset == nil {
		// 极端兜底
		preset = &whisper.PersonalityPresets[0]
	}
	orch := whisper.NewOrchestrator(sessionID, *preset)
	whisperSessions[sessionID] = orch
	return orch
}

func (a *App) WhisperGetPersonalities() []whisper.PersonalityPreset {
	return whisper.PersonalityPresets
}

func (a *App) WhisperChat(userMsg string, personalityID string) (map[string]interface{}, error) {
	orch := getOrCreateOrch(personalityID)

	result := orch.PreLLMTurn(userMsg)

	// 临时切换引擎
	origEngine := ""
	if orch.EngineID != "" {
		origEngine = a.client.ActiveEngineID()
		if orch.EngineID != origEngine {
			a.client.SetActiveEngine(orch.EngineID)
		}
	}
	reply, err := a.client.ChatSimpleStream(a.ctx, "", result.SystemPrompt, userMsg)
	if orch.EngineID != "" && orch.EngineID != origEngine {
		a.client.SetActiveEngine(origEngine)
	}
	if err != nil {
		return nil, err
	}

	orch.WM.Push(orch.SessionID, whisper.Exchange{
		TurnIndex: orch.State.Counters.TotalTurns, UserText: userMsg, AssistantText: reply,
	})

	// P2: post-turn 管线（同步）
	l1Snap := orch.State.Relationship
	l2Snap := orch.State.Emotion
	turns := orch.State.Counters.TotalTurns

	whisper.FinalizeTurn(orch, whisper.PostTurnContext{
		SessionID:     orch.SessionID,
		TurnIndex:     turns,
		UserMsg:       userMsg,
		AssistantText: reply,
		Event:         result.Event,
		AdultMode:     orch.AdultMode,
	})

	// 异步记忆摄入管线（注入 wubigrok 模型中心）
	go func() {
		whisper.EnqueueMemoryWrite(a, whisper.MemoryWritePayload{
			SessionID: orch.SessionID, TurnIndex: turns,
			UserMsg: userMsg, AssistantText: reply,
			L1: l1Snap, L2: l2Snap,
			FactStore: orch.FactStore, TotalTurns: turns, KG: orch.KG,
			EpisodicStore: nil, AdultMode: orch.AdultMode,
		})
	}()

	return map[string]interface{}{
		"desireSlots":  buildDesireSlots(orch.State.DesireStack),
		"trace":        result.Trace,
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
			"stage": string(state.Relationship.Stage), "trust": state.Relationship.Trust,
			"rifts": state.Relationship.Rifts, "atmosphere": string(state.Relationship.Atmosphere),
		},
		"emotion": map[string]interface{}{
			"aff": state.Emotion.Aff, "sec": state.Emotion.Sec, "aro": state.Emotion.Aro,
			"dom": state.Emotion.Dom, "label": state.Emotion.PrimaryLabel, "locked": state.Emotion.IsLocked,
		},
		"totalTurns": state.Counters.TotalTurns,
		"personality": map[string]interface{}{
			"id": state.Personality.PresetID, "T": state.Personality.T, "I": state.Personality.I,
			"S": state.Personality.S, "O": state.Personality.O, "R": state.Personality.R,
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
