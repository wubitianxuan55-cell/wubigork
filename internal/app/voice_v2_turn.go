// voice_v2_turn.go — V2 sidecar 已完成回合的 Gaea 权威收尾。
//
// sidecar 负责生成，Gaea 负责聊天事务、人格状态、轮次追踪与长期记忆抽取。
// response_id 同时写入 assistant extra，WebSocket 重试时可在进程重启后仍判重。
package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

var (
	voiceV2EnqueueMemory = func(a *whisperState, payload whisper.MemoryWritePayload) {
		whisper.EnqueueMemoryWrite(a, payload, a.recordMemoryWriteError)
	}
	voiceV2PersistState = func(a *whisperState, orch *whisper.Orchestrator) {
		go a.persistStateAsync(orch)
	}
)

type voiceV2PreparedTurn struct {
	personalityID string
	userText      string
	itemID        string
	turnID        string
	turnRevision  int
	orch          *whisper.Orchestrator
	snapshot      whisper.PreTurnSnapshot
	pre           whisper.PreLLMResult
}

// prepareVoiceV2Turn 在 sidecar 调 4B 前推进一次完整轻语状态，并返回生成
// 所需稳定 system + 本轮动态上下文。提交阶段消费同一 pre，绝不再调第二次。
func (a *whisperState) prepareVoiceV2Turn(
	personalityID, userText, itemID, turnID string, turnRevision int,
) *voiceV2PreparedTurn {
	personalityID = strings.TrimSpace(personalityID)
	userText = strings.TrimSpace(userText)
	itemID = strings.TrimSpace(itemID)
	turnID = strings.TrimSpace(turnID)
	if personalityID == "" || personalityID == "plain" || userText == "" {
		return nil
	}
	a.voiceV2PrepareMu.Lock()
	defer a.voiceV2PrepareMu.Unlock()

	// 同一语音台只允许一个待提交回合；新一轮抢先到来时先还原上一轮预推进。
	a.rollbackVoiceV2PreparedLocked()
	orch := a.getOrCreateOrch(personalityID)
	if orch == nil {
		return nil
	}
	orch.LockTurn()
	snapshot := orch.SnapshotPreTurn()
	var pre whisper.PreLLMResult
	var panicErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = fmt.Errorf("PreLLMTurn panic: %v", r)
			}
		}()
		pre = orch.PreLLMTurn(userText)
	}()
	orch.UnlockTurn()
	if panicErr != nil {
		orch.LockTurn()
		orch.RestorePreTurn(snapshot)
		orch.UnlockTurn()
		slog.Warn("Voice V2 生成前预推进失败", "personality", personalityID, "error", panicErr)
		return nil
	}

	prepared := &voiceV2PreparedTurn{
		personalityID: personalityID,
		userText:      userText,
		itemID:        itemID,
		turnID:        turnID,
		turnRevision:  turnRevision,
		orch:          orch,
		snapshot:      snapshot,
		pre:           pre,
	}
	a.voiceV2Prepared = prepared
	return prepared
}

// rollbackVoiceV2PreparedTurn 用于 4B HTTP/SSE 失败、客户端取消和下一轮抢先。
func (a *whisperState) rollbackVoiceV2PreparedTurn(prepared *voiceV2PreparedTurn) bool {
	a.voiceV2PrepareMu.Lock()
	defer a.voiceV2PrepareMu.Unlock()
	p := a.voiceV2Prepared
	if p == nil || p != prepared {
		return false
	}
	a.restoreVoiceV2PreparedLocked(p)
	a.voiceV2Prepared = nil
	return true
}

func (a *whisperState) rollbackVoiceV2Prepared() {
	a.voiceV2PrepareMu.Lock()
	defer a.voiceV2PrepareMu.Unlock()
	a.rollbackVoiceV2PreparedLocked()
}

func (a *whisperState) rollbackVoiceV2PreparedLocked() {
	if a.voiceV2Prepared == nil {
		return
	}
	a.restoreVoiceV2PreparedLocked(a.voiceV2Prepared)
	a.voiceV2Prepared = nil
}

func (a *whisperState) restoreVoiceV2PreparedLocked(p *voiceV2PreparedTurn) {
	if p == nil || p.orch == nil {
		return
	}
	p.orch.LockTurn()
	p.orch.RestorePreTurn(p.snapshot)
	p.orch.UnlockTurn()
}

func (a *whisperState) takeVoiceV2PreparedTurn(
	personalityID, userText, itemID string,
) (prepared *voiceV2PreparedTurn, blockedByLaterTurn bool) {
	a.voiceV2PrepareMu.Lock()
	defer a.voiceV2PrepareMu.Unlock()
	p := a.voiceV2Prepared
	if p == nil {
		return nil, false
	}
	itemID = strings.TrimSpace(itemID)
	matches := false
	if p.itemID != "" && itemID != "" {
		// Protocol item ids are the authoritative correlation key. This also
		// distinguishes repeated identical utterances that text matching cannot.
		matches = p.itemID == itemID
	} else {
		// Compatibility path for an older sidecar that does not forward item ids.
		matches = p.personalityID == personalityID && p.userText == strings.TrimSpace(userText)
	}
	if !matches {
		return nil, true
	}
	a.voiceV2Prepared = nil
	return p, false
}

// VoiceV2CommitTurn 由语音台在 response.done(completed) 后调用。话题 mode 是
// 人格唯一事实源；plain 只落聊天，不把无角色会话污染进某个人格记忆。
func (a *whisperState) VoiceV2CommitTurn(
	topicID, userText, assistantText, responseID, itemID string,
) (map[string]interface{}, error) {
	topicID = strings.TrimSpace(topicID)
	responseID = strings.TrimSpace(responseID)
	itemID = strings.TrimSpace(itemID)
	userText = strings.TrimSpace(userText)
	assistantText = strings.TrimSpace(assistantText)
	if topicID == "" || responseID == "" {
		return nil, fmt.Errorf("VoiceV2CommitTurn: topicID/responseID 不能为空")
	}
	if userText == "" || assistantText == "" {
		return nil, fmt.Errorf("VoiceV2CommitTurn: 用户/助手文本不能为空")
	}
	if len([]rune(userText)) > 16000 || len([]rune(assistantText)) > 32000 {
		return nil, fmt.Errorf("VoiceV2CommitTurn: 回合文本超出上限")
	}
	if a.chatStore == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}

	a.voiceV2CommitMu.Lock()
	defer a.voiceV2CommitMu.Unlock()

	topic, err := a.chatStore.GetTopic(topicID)
	if err != nil {
		return nil, err
	}
	duplicate, err := a.voiceV2ResponseCommitted(topicID, responseID)
	if err != nil {
		return nil, err
	}
	if duplicate {
		return map[string]interface{}{
			"committed":       true,
			"duplicate":       true,
			"memoryScheduled": false,
		}, nil
	}

	extra, err := json.Marshal(map[string]interface{}{
		"source":               "voice_v2",
		"voice_v2_response_id": responseID,
		"voice_v2_item_id":     itemID,
	})
	if err != nil {
		return nil, err
	}
	if err := a.chatStore.AppendExchange(topicID, userText, assistantText, string(extra)); err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"committed": true,
		"duplicate": false,
	}
	if topic.Mode == "" || topic.Mode == "plain" {
		result["memoryScheduled"] = false
		return result, nil
	}

	info, warning := a.finalizeVoiceV2PersonaTurn(topic.Mode, userText, assistantText, itemID)
	result["memoryScheduled"] = warning == nil
	for k, v := range info {
		result[k] = v
	}
	if warning != nil {
		result["warning"] = "聊天已保存，但人格记忆收尾失败: " + warning.Error()
		slog.Warn("Voice V2 人格记忆收尾失败（聊天已落库）",
			"topicID", topicID, "personality", topic.Mode, "responseID", responseID, "error", warning)
		a.emit("voice-v2-commit-warning", map[string]interface{}{
			"topicID": topicID, "responseID": responseID,
			"personality": topic.Mode, "error": warning.Error(),
		})
	}
	return result, nil
}

func (a *whisperState) voiceV2ResponseCommitted(topicID, responseID string) (bool, error) {
	// response 重试只可能紧邻当前尾部；只扫最近 100 条，避免长话题每轮全表读取。
	messages, _, err := a.chatStore.ListMessagesPage(topicID, 100, 0)
	if err != nil {
		return false, err
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "assistant" || strings.TrimSpace(messages[i].Extra) == "" {
			continue
		}
		var meta struct {
			Source     string `json:"source"`
			ResponseID string `json:"voice_v2_response_id"`
		}
		if json.Unmarshal([]byte(messages[i].Extra), &meta) == nil &&
			meta.Source == "voice_v2" && meta.ResponseID == responseID {
			return true, nil
		}
	}
	return false, nil
}

// finalizeVoiceV2PersonaTurn 复用 whisperChatCore 的回合尾链，但不重复调用模型。
// 这样 V2 sidecar 生成的回答能推进同一套关系/情绪/工作记忆/FactStore。
func (a *whisperState) finalizeVoiceV2PersonaTurn(
	personalityID, userText, assistantText, itemID string,
) (info map[string]interface{}, warning error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Voice V2 回合收尾 PANIC", "personality", personalityID, "panic", r)
			info = nil
			warning = fmt.Errorf("panic: %v", r)
		}
	}()

	orch := a.getOrCreateOrch(personalityID)
	if orch == nil {
		return nil, fmt.Errorf("人格会话未初始化")
	}
	prepared, blocked := a.takeVoiceV2PreparedTurn(personalityID, userText, itemID)
	if blocked {
		return nil, fmt.Errorf("后续语音回合已预推进人格状态，当前迟到回合不重复推进")
	}
	if prepared != nil && prepared.orch == orch {
		orch.LockTurn()
		defer orch.UnlockTurn()
		return a.completeVoiceV2TurnLocked(orch, prepared.pre, userText, assistantText)
	}

	orch.LockTurn()
	defer orch.UnlockTurn()

	pre := orch.PreLLMTurn(userText)
	return a.completeVoiceV2TurnLocked(orch, pre, userText, assistantText)
}

func (a *whisperState) completeVoiceV2TurnLocked(
	orch *whisper.Orchestrator,
	pre whisper.PreLLMResult,
	userText, assistantText string,
) (map[string]interface{}, error) {
	turns := orch.State.Counters.TotalTurns
	orch.WM.Push(orch.SessionID, whisper.Exchange{
		TurnIndex: turns, UserText: userText, AssistantText: assistantText,
	})
	if orch.HistoryInMessages {
		orch.RecordPromptTurn(turns, userText, assistantText)
	}

	l1Snap := orch.State.Relationship
	l2Snap := orch.State.Emotion
	recent := buildRecentExchanges(orch)
	whisper.FinalizeTurn(orch, whisper.PostTurnContext{
		SessionID: orch.SessionID, TurnIndex: turns,
		UserMsg: userText, AssistantText: assistantText,
		Event: pre.Event, AdultMode: orch.AdultMode,
	})
	if err := repos.AppendTurnTraceToDB(orch.DataRoot, orch.SessionID, pre.Trace); err != nil {
		slog.Warn("Voice V2 轮次追踪落库失败", "sessionID", orch.SessionID, "error", err)
	}

	voiceV2EnqueueMemory(a, whisper.MemoryWritePayload{
		SessionID: orch.SessionID, TurnIndex: turns,
		UserMsg: userText, AssistantText: assistantText,
		L1: l1Snap, L2: l2Snap,
		FactStore: orch.FactStore, TotalTurns: turns, KG: orch.KG,
		EpisodicStore: orch.EpisodicStore, RecentExchanges: recent,
		TemporalAnchorSink: func(anchor whisper.TemporalAnchor) { orch.AddTemporalAnchor(anchor) },
		AdultMode:          orch.AdultMode,
	})
	voiceV2PersistState(a, orch)

	return map[string]interface{}{
		"emotion":    orch.State.Emotion.PrimaryLabel,
		"stage":      string(orch.State.Relationship.Stage),
		"totalTurns": turns,
	}, nil
}
