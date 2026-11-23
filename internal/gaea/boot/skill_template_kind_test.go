package boot

import "testing"

// 技能名 → 子代理模板 kind 映射回归（同类子代理共享 L4 前缀缓存，V5.30
// 机制）。novel-agents→subagent_writing 映射已随 2026-11-23 技能退役删除。
func TestSubagentSkillToTemplateKind(t *testing.T) {
	if got := subagentSkillToTemplateKind("explore"); got != "subagent_explore" {
		t.Fatalf("explore kind = %q, want subagent_explore（既有映射回归）", got)
	}
	if got := subagentSkillToTemplateKind("novel-agents"); got != "" {
		t.Fatalf("novel-agents kind = %q, want empty（技能已退役）", got)
	}
	if got := subagentSkillToTemplateKind("no-such-skill"); got != "" {
		t.Fatalf("unknown skill kind = %q, want empty", got)
	}
}
