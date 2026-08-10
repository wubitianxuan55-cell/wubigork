package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestSuggestSkillsFromMemories(t *testing.T) {
	ms := []memory.Memory{
		{
			Name: "cost-estimate-steps", Title: "成本测算步骤",
			Description: "成本测算先对科目再汇总", Kind: memory.KindProcedural,
			Body: "1. 对齐科目；2. 金额公式；3. 图表",
		},
		{
			Name: "cost-estimate-format", Title: "成本测算格式",
			Description: "成本测算表统一格式", Kind: memory.KindProcedural,
			Body: "科目/单位/数量/单价/金额",
		},
		{
			Name: "weekly-report", Title: "周报",
			Description: "周报结构", Kind: memory.KindProcedural,
			Body: "进展/数据/问题/计划",
		},
	}
	skills := suggestSkillsFromMemories(ms)
	if len(skills) == 0 {
		t.Fatal("应产生技能候选")
	}
	// 成本相关两条共用 estimate 主题词 → 候选含 workflow-estimate
	found := false
	for _, s := range skills {
		if s.Name == "workflow-estimate" {
			found = true
			if len(s.Evidence) != 2 {
				t.Fatalf("evidence 应为 2 条: %+v", s.Evidence)
			}
			if !strings.Contains(s.Body, "成本测算步骤") {
				t.Fatalf("候选正文应含记忆内容: %s", s.Body)
			}
		}
	}
	if !found {
		t.Fatalf("缺少 workflow-estimate 候选: %+v", skills)
	}
}

func TestSuggestSkillsNeedsTwo(t *testing.T) {
	ms := []memory.Memory{
		{Name: "only-one-rule", Title: "唯一规则", Description: "x", Kind: memory.KindProcedural},
		{Name: "unrelated-fact", Title: "无关", Description: "y", Kind: memory.KindSemantic},
	}
	if got := suggestSkillsFromMemories(ms); len(got) != 0 {
		t.Fatalf("少于 2 条共用主题词不应产生候选: %+v", got)
	}
}
