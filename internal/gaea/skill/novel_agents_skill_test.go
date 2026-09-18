package skill

import (
	"strings"
	"testing"
)

// v4.336：oh-story T5 落库接线验收钉——novel-agents（网文创作七角色卡）
// 从静态资产升格为可派发的 runAs=subagent 技能（规格
// docs/gaea-novel-ohstory-distill-2026-09.md §3 T5；资产 dca0936b）。
// 仓库根 .gaea/skills/ 即项目技能根，这里以仓库根为 ProjectRoot 直读。

func TestNovelAgentsSkill_DispatchableWiring(t *testing.T) {
	st := New(Options{ProjectRoot: "../../..", HomeDir: t.TempDir()})
	sk, ok := st.Read("novel-agents")
	if !ok {
		t.Fatal("project-scope novel-agents skill not found (expected .gaea/skills/novel-agents/SKILL.md)")
	}
	if sk.Scope != ScopeProject {
		t.Errorf("novel-agents Scope = %s, want project", sk.Scope)
	}
	if sk.RunAs != RunSubagent {
		t.Errorf("novel-agents RunAs = %s, want subagent（T5 接线核心：run_skill 可派发）", sk.RunAs)
	}

	// 派发契约锚点：子代理执行序（解析 role → 读卡 → 按卡执行）不可丢
	anchors := []string{
		"派发契约",
		"role=",
		"read_file",
		".gaea/skills/novel-agents/agents/",
	}
	for _, a := range anchors {
		if !strings.Contains(sk.Body, a) {
			t.Errorf("body missing dispatch anchor %q", a)
		}
	}

	// 七张角色卡逐一在册（合法 role 清单与 agents/ 目录一致）
	roles := []string{
		"story-architect", "character-designer", "narrative-writer",
		"consistency-checker", "chapter-extractor", "story-explorer", "story-researcher",
	}
	for _, r := range roles {
		if !strings.Contains(sk.Body, r) {
			t.Errorf("body missing role %q", r)
		}
	}
}
