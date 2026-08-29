package app

import (
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper"
)

// GaeaWhisperGraphSubgraph 返回指定人格关系图谱中以 entity 为中心、hops 跳内
// 的邻接子图（v4.3b，play 空间会客厅）。会话不存在或图谱为空时返回空子图
// （只读查询，不创建会话、无副作用）；entity 不存在返回空子图。
func (a *whisperState) GaeaWhisperGraphSubgraph(personalityID, entity string, hops int) (whisper.Subgraph, error) {
	sessionID := "whisper_" + personalityID
	whisperSessionsMu.RLock()
	orch, ok := whisperSessions[sessionID]
	whisperSessionsMu.RUnlock()
	if !ok || orch == nil || orch.KG == nil {
		return whisper.Subgraph{}, nil
	}
	return orch.KG.QuerySubgraph(entity, hops), nil
}

// GaeaWhisperProactiveNow 手动触发一次主动关心评估（v4.3c，play 空间）：
// 用现成门控（EvaluateProactiveGate）+ 合成器（ComposeProactiveMessage）按当前
// 关系/情绪/时段评估，返回是否该发与消息类型/提示词。前端「轻语先开口」按钮
// 与定时器共用同一评估入口；定时器频控由前端/调用侧承担。
func (a *whisperState) GaeaWhisperProactiveNow(personalityID string) (map[string]interface{}, error) {
	orch := a.getOrCreateOrch(personalityID)
	if orch == nil {
		return nil, fmt.Errorf("轻语会话未就绪")
	}
	st := orch.State
	now := timeOfDayNow()
	gate := whisper.EvaluateProactiveGate(
		st.Emotion.Aff, st.Emotion.Aro, st.Emotion.Sec,
		st.Relationship.Trust, st.Relationship.Rifts,
		st.Relationship.Stage, now, orch.AdultMode)
	res := whisper.ComposeProactiveMessage(
		gate, st.Emotion.Aff, st.Emotion.Sec,
		st.Relationship.Trust, st.Relationship.Stage,
		now, 0, false, personalityID)
	if res == nil || !res.ShouldSend {
		return map[string]interface{}{"shouldSend": false}, nil
	}
	return map[string]interface{}{
		"shouldSend":  true,
		"messageType": string(res.MessageType),
		"promptHint":  res.PromptHint,
	}, nil
}

// timeOfDayNow 返回当前时段标签（late_night 23-5 点，其余空串——合成器仅
// 消费 late_night 分支）。
func timeOfDayNow() string {
	h := time.Now().Hour()
	if h >= 23 || h < 5 {
		return "late_night"
	}
	return ""
}
