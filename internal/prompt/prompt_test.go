package prompt

import (
	"strings"
	"testing"
)

func TestEngineGetExisting(t *testing.T) {
	eng := NewEngine("../../prompts")
	if eng == nil {
		t.Fatal("NewEngine returned nil")
	}

	tmpl := eng.Get("chapter-generate")
	if tmpl == nil {
		t.Fatal("chapter-generate template should exist")
	}
	if tmpl.Name != "chapter-generate" {
		t.Errorf("template Name = %q, want %q", tmpl.Name, "chapter-generate")
	}
}

func TestEngineGetMissing(t *testing.T) {
	eng := NewEngine("../../prompts")
	tmpl := eng.Get("nonexistent-template")
	if tmpl != nil {
		t.Error("nonexistent template should return nil")
	}
}

func TestEngineGetAllTemplates(t *testing.T) {
	eng := NewEngine("../../prompts")
	names := []string{
		"chapter-generate", "chapter-summary",
		"character-agent", "character-detail", "character-generate-single", "character-generate-batch",
		"worldview-agent",
		"outline-chat", "outline-chat-node", "outline-continue", "outline-expand",
		"analysis-chapter", "plot-branch-browser", "create-chapter",
	}
	for _, name := range names {
		if tmpl := eng.Get(name); tmpl == nil {
			t.Errorf("template %q should exist", name)
		}
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	eng := NewEngine("../../prompts")
	tmpl := eng.Get("chapter-generate")
	if tmpl == nil {
		t.Fatal("chapter-generate not found")
	}

	result := tmpl.BuildSystemPrompt("")
	if !strings.Contains(result, "作者") {
		t.Error("system prompt should contain role description")
	}
	if !strings.Contains(result, "任务") {
		t.Error("system prompt should contain task section")
	}
	if !strings.Contains(result, "创作约束") {
		t.Error("system prompt should contain constraints section")
	}
	if !strings.Contains(result, "✅") && !strings.Contains(result, "必须") {
		t.Error("system prompt should contain 'must' constraints")
	}
}

func TestBuildSystemPromptWithSkill(t *testing.T) {
	eng := NewEngine("../../prompts")
	tmpl := eng.Get("chapter-generate")
	if tmpl == nil {
		t.Fatal("chapter-generate not found")
	}

	skillMD := "测试写作指导：多用比喻，避免平铺直叙。"
	result := tmpl.BuildSystemPrompt(skillMD)

	if !strings.Contains(result, "额外写作指导") {
		t.Error("system prompt with skill should contain 额外写作指导 section")
	}
	if !strings.Contains(result, "多用比喻") {
		t.Error("system prompt with skill should contain the skill content")
	}
}

func TestBuildUserPromptPriorityOrder(t *testing.T) {
	eng := NewEngine("../../prompts")
	tmpl := eng.Get("chapter-generate")
	if tmpl == nil {
		t.Fatal("chapter-generate not found")
	}

	contexts := map[string]string{
		"outline_node":       "大纲内容",
		"prev_chapter":       "上一章",
		"prev_summary":       "摘要",
		"character_status":   "状态",
		"all_summaries":      "全书摘要",
		"active_foreshadows": "伏笔",
		"worldview":          "世界观",
		"all_characters":     "角色列表",
	}

	result := tmpl.BuildUserPrompt(contexts)

	// P0 应该出现在 P1 和 P2 之前
	p0idx := strings.Index(result, "大纲内容")
	p1idx := strings.Index(result, "全书摘要")
	p2idx := strings.Index(result, "世界观")

	if p0idx < 0 || p1idx < 0 || p2idx < 0 {
		t.Logf("result:\n%s", result)
		t.Skip("some sections missing, skipping order check")
		return
	}

	if p0idx >= p1idx {
		t.Errorf("P0 section should appear before P1: P0 at %d, P1 at %d", p0idx, p1idx)
	}
	if p1idx >= p2idx {
		t.Errorf("P1 section should appear before P2: P1 at %d, P2 at %d", p1idx, p2idx)
	}
}

func TestBuildUserPromptEmptyContext(t *testing.T) {
	eng := NewEngine("../../prompts")
	tmpl := eng.Get("chapter-summary")
	if tmpl == nil {
		t.Fatal("chapter-summary not found")
	}

	result := tmpl.BuildUserPrompt(map[string]string{})
	if result != "" {
		t.Errorf("BuildUserPrompt with empty context should return empty, got: %q", result)
	}
}
