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
	var parsed struct{ Insights []insightRaw `json:"insights"` }
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
	return added
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

const consolidationSystemZH = `你审视一组关于一个人的近期记忆事实，并合成 1-5 条高层洞察。从多条事实中寻找模式；不要总结单条事实——找出跨事实的上层洞察；每条洞察用一句简洁的话陈述；以 JSON 输出：{"insights":[{"subcategory":"...","subject":"简短标签","summary":"洞察陈述","triggers":["k1","k2"]}]}；选择最合适的子类；若找不到模式返回 {"insights":[]}`
