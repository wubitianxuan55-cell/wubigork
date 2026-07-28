// Package whisper — 欲望栈 P2-1（100% 对齐 ackem engine/desire.ts）
package whisper

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// ─── 欲望触发概率表 ───────────────────────────────────────────

type desireTrigger struct {
	chance     float64
	categories []string
}

var desireTriggers = map[EventType]desireTrigger{
	EvtVulnerable: {0.20, []string{"concern", "share"}},
	EvtQuestion:   {0.12, []string{"curiosity", "suggest"}},
	EvtPraise:     {0.10, []string{"share", "tease"}},
	EvtTease:      {0.15, []string{"tease", "curiosity"}},
	EvtCasualChat: {0.06, []string{"curiosity", "share", "suggest"}},
	EvtApology:    {0.08, []string{"concern"}},
	EvtCold:       {0.12, []string{"concern", "curiosity"}},
	EvtHurtful:    {0.03, []string{"concern"}},
}

// ─── 话题提取 ─────────────────────────────────────────────────

func extractTopic(userMsg string) string {
	clean := strings.Map(func(r rune) rune {
		if strings.ContainsRune("，。！？、的了我你是", r) {
			return ' '
		}
		return r
	}, userMsg)
	words := strings.Fields(clean)
	var result []string
	for _, w := range words {
		if len([]rune(w)) >= 2 {
			result = append(result, w)
		}
	}
	if len(result) == 0 {
		return "近况"
	}
	// 取前3个
	n := 3
	if len(result) < n {
		n = len(result)
	}
	return strings.Join(result[:n], "")
}

func normalizeTopicKey(s string) string {
	prefixes := []string{"搜一下", "帮我搜", "帮我查", "查一下", "介绍一下", "介绍", "讲讲", "说说", "了解", "想了解"}
	for _, p := range prefixes {
		s = strings.TrimPrefix(s, p)
	}
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune("，。！？、 ", r) {
			return -1
		}
		return r
	}, s)
	return strings.ToLower(s)
}

// DesireTopicMatchesKnowledge 知识整理主题是否与欲望 topic 相关
func DesireTopicMatchesKnowledge(desireTopic, knowledgeTopic string) bool {
	a := normalizeTopicKey(desireTopic)
	b := normalizeTopicKey(knowledgeTopic)
	if len(a) < 2 || len(b) < 2 {
		return false
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// ─── 欲望→提示转换 ───────────────────────────────────────────

func desireToHint(d Desire) string {
	switch d.Category {
	case "concern":
		return fmt.Sprintf("有点担心ta的%s，想问问", d.Topic)
	case "curiosity":
		return fmt.Sprintf("对ta说的%s很好奇，想了解更多", d.Topic)
	case "share":
		return fmt.Sprintf("想和ta分享关于%s的事", d.Topic)
	case "tease":
		return fmt.Sprintf("想在%s上小小捉弄ta一下", d.Topic)
	case "suggest":
		return fmt.Sprintf("有个关于%s的建议想告诉ta", d.Topic)
	}
	return ""
}

// ─── 欲望生成 ─────────────────────────────────────────────────

func generateDesire(userMsg string, event Event, turnIndex int, stage RelationshipStage) *Desire {
	trigger, ok := desireTriggers[event.Type]
	if !ok {
		return nil
	}

	stageBonus := 1.0
	if stage == StageIntimate {
		stageBonus = 1.5
	} else if stage == StageFamiliar {
		stageBonus = 1.2
	}
	intensityBonus := 0.5 + event.Intensity*0.5
	chance := trigger.chance * stageBonus * intensityBonus

	if rand.Float64() > chance {
		return nil
	}

	topic := extractTopic(userMsg)
	category := trigger.categories[rand.Intn(len(trigger.categories))]

	return &Desire{
		ID:        fmt.Sprintf("d_%d_%d", turnIndex, rand.Intn(10000)),
		Topic:     topic,
		Category:  category,
		Urgency:   1 + event.Intensity*2,
		Status:    "active",
		SourceTurn: turnIndex,
		CreatedAt: time.Now(),
	}
}

// ─── 沉淀规则 ─────────────────────────────────────────────────

func applySettleRules(slots []*Desire, turnIndex int) {
	for i := 0; i < DesireMaxSlots && i < len(slots); i++ {
		d := slots[i]
		if d == nil || d.Status == "settled" {
			continue
		}

		if d.Status == "expressed" {
			expressedAt := d.SourceTurn
			if d.ExpressedAtTurn != nil {
				expressedAt = *d.ExpressedAtTurn
			}
			if turnIndex-expressedAt >= DesireExpressedSettleAfterTurns {
				d.Status = "settled"
				d.Urgency = 0
				slots[i] = d
			}
			continue
		}

		if d.Status != "active" {
			continue
		}

		idleTurns := mathMax(0, float64(turnIndex-d.SourceTurn))
		if d.Urgency <= 0 || idleTurns >= DesireIdleSettleTurns {
			d.Status = "settled"
			d.Urgency = 0
			slots[i] = d
		}
	}
}

// ─── UpdateDesireStack 主入口 ─────────────────────────────────

// UpdateDesireStack 每轮更新欲望栈
func UpdateDesireStack(stack DesireStack, userMsg string, event Event, l1 L1State, turnIndex int) (DesireStack, []string) {
	slots := make([]*Desire, DesireMaxSlots)
	copy(slots, stack.Slots[:DesireMaxSlots])

	// 确保至少有 DesireMaxSlots 个槽位
	for len(slots) < DesireMaxSlots {
		slots = append(slots, nil)
	}

	// 1. 衰减存量
	for i := 0; i < DesireMaxSlots; i++ {
		d := slots[i]
		if d == nil || d.Status == "settled" || d.Status == "expressed" {
			continue
		}
		d.Urgency = math.Max(0, d.Urgency-DesireDecayPerTurn)
		slots[i] = d
	}

	// 2. 沉淀
	applySettleRules(slots, turnIndex)

	// 3. 生成新欲望
	newD := generateDesire(userMsg, event, turnIndex, l1.Stage)
	if newD != nil {
		emptyIdx := -1
		for i := 0; i < DesireMaxSlots; i++ {
			if slots[i] == nil || slots[i].Status == "settled" {
				emptyIdx = i
				break
			}
		}
		if emptyIdx >= 0 {
			slots[emptyIdx] = newD
		} else {
			minIdx := 0
			minUrgency := math.MaxFloat64
			for i := 0; i < DesireMaxSlots; i++ {
				d := slots[i]
				if d == nil || d.Status == "settled" {
					continue
				}
				if d.Urgency < minUrgency {
					minUrgency = d.Urgency
					minIdx = i
				}
			}
			slots[minIdx] = newD
		}
	}

	// 4. 收集表达提示
	var hints []string
	for i := 0; i < DesireMaxSlots; i++ {
		d := slots[i]
		if d == nil || d.Status != "active" {
			continue
		}
		if d.Urgency >= DesireExpressThreshold {
			hints = append(hints, desireToHint(*d))
			d.Status = "expressed"
			d.Urgency = 0
			at := turnIndex
			d.ExpressedAtTurn = &at
			slots[i] = d
		}
	}

	applySettleRules(slots, turnIndex)

	// 截断到 DesireMaxSlots
	if len(slots) > DesireMaxSlots {
		slots = slots[:DesireMaxSlots]
	}

	return DesireStack{Slots: slots}, hints
}

// DefaultDesireStack 默认空欲望栈
func DefaultDesireStack() DesireStack {
	return DesireStack{Slots: make([]*Desire, DesireMaxSlots)}
}
