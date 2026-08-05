package proposal

import (
	"context"
	"testing"
)

func TestAllocateWordBudget_LeavesSumToTotal(t *testing.T) {
	sections := []ProposalSection{
		{Title: "第一章", Level: 1, Children: []ProposalSection{
			{Title: "1.1", Level: 2},
			{Title: "1.2", Level: 2},
		}},
		{Title: "第二章", Level: 1, Children: []ProposalSection{
			{Title: "2.1", Level: 2},
		}},
		{Title: "第三章", Level: 1},
	}
	AllocateWordBudget(sections, 150000)
	sum := 0
	var walk func(ss []ProposalSection)
	walk = func(ss []ProposalSection) {
		for _, sec := range ss {
			if len(sec.Children) == 0 {
				sum += sec.WordTarget
			}
			walk(sec.Children)
		}
	}
	walk(sections)
	if sum != 150000 {
		t.Fatalf("叶子字数合计 = %d, want 150000", sum)
	}
	if sections[0].WordTarget <= 0 {
		t.Errorf("章字数目标未设置: %+v", sections[0])
	}
}

func TestOutlineStrategyPrompt(t *testing.T) {
	if !containsAny(outlineStrategyPrompt("scoring"), "评分标准", "严格按") {
		t.Error("scoring 策略提示词缺少评分标准约束")
	}
	if !containsAny(outlineStrategyPrompt("format"), "投标文件格式") {
		t.Error("format 策略提示词缺少格式要求约束")
	}
	if !containsAny(outlineStrategyPrompt("reference"), "参考") {
		t.Error("reference 策略提示词异常")
	}
}

func TestGenerateOutline_TotalWordsFallback(t *testing.T) {
	ai := &mockAI{replies: map[string]string{"需求描述": `{"title":"方案","sections":[{"title":"第一章","level":1}]}`}}
	s := newServiceAt(t, t.TempDir(), ai)
	p, _ := s.Create("方案", "blank", "需求描述", "其他")
	p.BidSummary = &BidSummary{TotalWords: 80000}
	_ = s.store.Update(p)
	got, err := s.GenerateOutline(context.Background(), p.ID, "需求描述", OutlineStrategyReference, 0)
	if err != nil {
		t.Fatalf("GenerateOutline: %v", err)
	}
	if got.Sections[0].WordTarget != 80000 {
		t.Errorf("WordTarget = %d, want 80000（应取招标要求）", got.Sections[0].WordTarget)
	}
}
