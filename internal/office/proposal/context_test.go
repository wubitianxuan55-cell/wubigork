package proposal

import (
	"context"
	"testing"
)

func TestSectionContext_IncludesAllLayers(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "soil-remediation-bid", "需求", "环保工程", proj.ID, []ProposalSection{
		{Title: "第一章", Level: 1, WordTarget: 5000, Children: []ProposalSection{
			{Title: "1.1 项目背景", Level: 2, WordTarget: 2500},
			{Title: "1.2 修复目标", Level: 2, WordTarget: 2500},
		}},
		{Title: "第二章", Level: 1},
	})
	p.BidSummary = &BidSummary{
		Overview: "污染场地修复", Duration: "90 日历天",
		TechScoring: []ScoringItem{{Name: "施工方案", MaxScore: "20", Requirement: "完整合理"}},
		RedLines:    []string{"未签字盖章"},
		Format:      []BidItem{{Name: "装订", Content: "A4 双面"}},
		DarkRules:   []BidItem{{Name: "暗标", Content: "不得出现单位名称"}},
	}
	_ = s.store.SaveProjectFacts(proj.ID, map[string]string{"业主单位": "某区生态环境局", "修复目标": "砷 ≤ 60 mg/kg"})
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	targetID := p.Sections[0].Children[0].ID
	sc, err := s.SectionContext(context.Background(), p.ID, targetID)
	if err != nil {
		t.Fatalf("SectionContext: %v", err)
	}
	if sc.WordTarget != 2500 {
		t.Errorf("WordTarget = %d, want 2500", sc.WordTarget)
	}
	joined := sc.SystemPrompt + "\n" + sc.UserPrompt
	for _, want := range []string{"施工方案", "90 日历天", "未签字盖章", "A4 双面", "不得出现单位名称", "业主单位", "砷 ≤ 60 mg/kg", "1.2 修复目标"} {
		if !containsAny(joined, want) {
			t.Errorf("上下文缺少 %q", want)
		}
	}
}
