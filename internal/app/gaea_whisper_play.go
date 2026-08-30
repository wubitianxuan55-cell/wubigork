package app

import (
	"fmt"
	"strings"
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
	sub := orch.KG.QuerySubgraph(entity, hops)
	enrichSubgraphWithAssociations(orch, &sub)
	return sub, nil
}

// assocTypeLabels 记忆关联类型 → 图谱边中文标签（v4.9 事件链/关系入图）。
var assocTypeLabels = map[string]string{
	"event_chain":    "因果",
	"temporal":       "时间",
	"entity":         "同实体",
	"emotion_peak":   "情绪相似",
	"self_reference": "自我",
	"thematic":       "主题",
}

// enrichSubgraphWithAssociations 把记忆关联（fact↔fact，含 event_chain 因果链）
// 以「事实 Subject 实体」映射成图边并入子图。数据早已存在（memory_associations +
// AssocIndex），此前只活在索引里、图谱面板不可见（审计 §C「推理仅邻接遍历」补口）。
// 只读、无副作用；只并入至少一端已在子图内的关联（保持以查询实体为中心），
// 与 KG 边按 From|Type|To 去重，关联边权重 = strength。
func enrichSubgraphWithAssociations(orch *whisper.Orchestrator, sub *whisper.Subgraph) {
	if orch == nil || orch.AssocIndex == nil || orch.FactStore == nil {
		return
	}
	assocs := orch.AssocIndex.ListAll()
	if len(assocs) == 0 {
		return
	}
	subjectByFact := map[string]string{}
	for _, f := range orch.FactStore.ListActive() {
		if s := strings.TrimSpace(f.Subject); s != "" {
			subjectByFact[f.ID] = s
		}
	}
	existing := map[string]bool{}
	for _, e := range sub.Edges {
		existing[e.From+"\x00"+e.Type+"\x00"+e.To] = true
	}
	nodeSet := map[string]bool{}
	for _, n := range sub.Nodes {
		nodeSet[n.ID] = true
	}
	for _, a := range assocs {
		sa, okA := subjectByFact[a.FactIDA]
		sb, okB := subjectByFact[a.FactIDB]
		if !okA || !okB || sa == sb {
			continue
		}
		// 只并入与当前子图连通的关联（一端已是节点）
		if !nodeSet[sa] && !nodeSet[sb] {
			continue
		}
		label := assocTypeLabels[a.AssociationType]
		if label == "" {
			label = a.AssociationType
		}
		key := sa + "\x00" + label + "\x00" + sb
		if existing[key] {
			continue
		}
		existing[key] = true
		if !nodeSet[sa] {
			nodeSet[sa] = true
			sub.Nodes = append(sub.Nodes, whisper.GraphNode{ID: sa, Name: sa, Weight: a.Strength})
		}
		if !nodeSet[sb] {
			nodeSet[sb] = true
			sub.Nodes = append(sub.Nodes, whisper.GraphNode{ID: sb, Name: sb, Weight: a.Strength})
		}
		sub.Edges = append(sub.Edges, whisper.GraphEdge{
			From: sa, To: sb, Type: label, Weight: a.Strength,
		})
	}
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
// 消费 late_night 分支）。v4.3c：实现收敛到 timeOfDayFor（定时推送可注入 now）。
func timeOfDayNow() string {
	return timeOfDayFor(time.Now())
}

// timeOfDayFor 由指定时间计算时段标签（测试可注入 now）。
func timeOfDayFor(t time.Time) string {
	h := t.Hour()
	if h >= 23 || h < 5 {
		return "late_night"
	}
	return ""
}
