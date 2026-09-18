package boot

import "testing"

// v4.336：novel-agents（网文创作七角色卡）→ subagent_writing 模板映射——
// 同类子代理共享 L4 前缀缓存（V5.30 机制，oh-story T5 落库接线）。
func TestSubagentSkillToTemplateKind_NovelAgents(t *testing.T) {
	if got := subagentSkillToTemplateKind("novel-agents"); got != "subagent_writing" {
		t.Fatalf("novel-agents kind = %q, want subagent_writing", got)
	}
	if got := subagentSkillToTemplateKind("explore"); got != "subagent_explore" {
		t.Fatalf("explore kind = %q, want subagent_explore（既有映射回归）", got)
	}
	if got := subagentSkillToTemplateKind("no-such-skill"); got != "" {
		t.Fatalf("unknown skill kind = %q, want empty", got)
	}
}
