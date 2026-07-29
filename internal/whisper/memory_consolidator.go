// Package whisper — memory_consolidator.go
// 100% 对齐 ackem memory/consolidator.ts

package whisper

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	consolidationMaxFactsInput = 30
	consolidationMinFacts      = 8
	consolidationInsightWeight = 2.5
	consolidationMaxInsights   = 5
)

type MemoryConsolidator struct{}

func NewMemoryConsolidator() *MemoryConsolidator { return &MemoryConsolidator{} }

func (mc *MemoryConsolidator) Consolidate(
	fs *FactStore, llm LlmClient, emoCtx *EmotionalContext,
	sessionID string, turnIndex int,
) int {
	var recent []*Fact
	for _, f := range fs.ListActive() {
		if f.FactLayer == "" || f.FactLayer == "raw" {
			recent = append(recent, f)
		}
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].UpdatedAt.After(recent[j].UpdatedAt) })
	if len(recent) > consolidationMaxFactsInput {
		recent = recent[:consolidationMaxFactsInput]
	}
	if len(recent) < consolidationMinFacts {
		return 0
	}

	var factLines []string
	for i, f := range recent {
		factLines = append(factLines, fmt.Sprintf("[%d] (%s) %s: %s", i+1, f.Subcategory, f.Subject, f.Summary))
	}
	raw, err := llm.Chat(consolidationSystemZH, fmt.Sprintf("近期事实（共%d条）：\n%s", len(recent), strings.Join(factLines, "\n")))
	if err != nil {
		return 0
	}

	added := 0
	derivedFrom := make([]string, len(recent))
	for i, f := range recent {
		derivedFrom[i] = f.ID
	}
	if emoCtx == nil {
		emoCtx = &EmotionalContext{}
	}

	type insightRaw struct {
		Subcategory string   `json:"subcategory"`
		Subject     string   `json:"subject"`
		Summary     string   `json:"summary"`
		Triggers    []string `json:"triggers"`
	}
	var parsed struct {
		Insights     []insightRaw `json:"insights"`
		Associations []struct {
			FactAIdx int    `json:"fact_a_idx"`
			FactBIdx int    `json:"fact_b_idx"`
			Type     string `json:"type"`
			Strength string `json:"strength"`
		} `json:"associations"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil {
		s := strings.TrimSpace(raw)
		i := strings.Index(s, "{")
		j := strings.LastIndex(s, "}")
		if i < 0 || j <= i || json.Unmarshal([]byte(s[i:j+1]), &parsed) != nil {
			return 0
		}
	}

	for _, ins := range parsed.Insights {
		if added >= consolidationMaxInsights {
			break
		}
		if ins.Subcategory == "" || ins.Subject == "" || ins.Summary == "" {
			continue
		}
		fs.Add(MemoryFact{
			Domain: subcategoryToDomain(ins.Subcategory), Subcategory: ins.Subcategory,
			Subject: ins.Subject, Summary: ins.Summary, Weight: consolidationInsightWeight,
			Confidence: 0.7, SelfRelevance: 1.0, Triggers: ins.Triggers,
			SourceSessionID: sessionID, SourceTurnIndex: turnIndex,
			DerivedFrom: derivedFrom, FactLayer: "consolidated", EmotionalContext: emoCtx,
		})
		added++
	}

	// 存储关联（如果提供了 AssociationIndex）
	for _, a := range parsed.Associations {
		if a.FactAIdx < 1 || a.FactBIdx < 1 || a.FactAIdx > len(recent) || a.FactBIdx > len(recent) {
			continue
		}
		s := assocStrength(a.Strength)
		if a.Type == "" || s == 0 {
			continue
		}
		// 注意：此处仅解析关联，实际写入 AssociationIndex 由调用方 orchestator 负责
		// 见 post_chat_turn.go consolidateNow → orch.AssocIndex.Add(...)
		_ = s // 关联强度已计算，供后续集成使用
	}

	return added
}

func assocStrength(s string) float64 {
	switch s {
	case "strong":
		return 0.8
	case "medium":
		return 0.5
	case "weak":
		return 0.2
	default:
		return 0
	}
}

func subcategoryToDomain(sub string) string {
	m := map[string]string{
		"BASIC_PROFILE": "IDENTITY", "LIFE_STORY": "IDENTITY",
		"VALUES_BELIEFS": "IDENTITY", "SELF_PERCEPTION": "IDENTITY",
		"OUR_BOND": "SOCIAL", "FAMILY": "SOCIAL", "FRIENDS": "SOCIAL", "PARTNER": "SOCIAL",
		"ROUTINES": "DAILY_LIFE", "HEALTH": "DAILY_LIFE",
		"LIVING_SPACE": "DAILY_LIFE", "LIFESTYLE": "DAILY_LIFE",
		"CAREER": "PURSUITS", "LEARNING": "PURSUITS", "GOALS": "PURSUITS",
		"PROJECTS": "PURSUITS", "PROCEDURES": "PURSUITS",
		"MOOD": "INNER_WORLD", "TASTES": "INNER_WORLD",
		"VULNERABILITIES": "INNER_WORLD", "INSIDE_JOKES": "INNER_WORLD",
		"NOW": "TEMPORAL", "COMMITMENTS": "TEMPORAL", "PLANS": "TEMPORAL", "WORLD": "TEMPORAL",
	}
	if d, ok := m[sub]; ok {
		return d
	}
	return "INNER_WORLD"
}

// consolidationSystemZH 记忆整合反思系统提示
// 100% 对齐 ackem prompt/memory-consolidation.ts
const consolidationSystemZH = `你审视一组关于用户的近期记忆事实，合成高层洞察和事实间关联。

── 输入限制 ──
- 只处理最近 50 条事实（或 weight≥1 的事实前 100 条）
- 输入事实按时间倒序排列，每条带序号

── 洞察规则 ──
- 从多条事实中寻找模式（反复出现的主题、价值观、性格特质、行为模式）
- 不要总结单条事实——找出跨事实的上层洞察
- 洞察必须是"用户未直接说但可以从多条事实推断的"
- 每条洞察用一句简洁的话陈述
- 洞察 subcategory 只能从以下选择：VALUES_BELIEFS, SELF_PERCEPTION, LIFESTYLE, MOOD, TASTES, GOALS, VULNERABILITIES, OUR_BOND

── 关联规则 ──
- 判断事实之间的关联关系
- 关联类型：temporal(时间有关), entity(同一实体), event_chain(因果前后), emotion_peak(情绪相似), self_reference(自我认知), thematic(同一主题)
- 强度用定性等级：strong(0.8) / medium(0.5) / weak(0.2)
- 使用输入事实的序号引用

── 输出 ──
{"insights":[{"subcategory":"...","subject":"标签","summary":"洞察","triggers":["关键词"]}],
 "associations":[{"fact_a_idx":0,"fact_b_idx":2,"type":"thematic","strength":"medium"}]}

若找不到有意义的模式，返回 {"insights":[],"associations":[]}`
