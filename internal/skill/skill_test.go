package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValidSkillMD(t *testing.T) {
	// 创建一个临时 SKILL.md
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	content := `---
name: test-style
description: 测试写作风格
version: "1.0"
applies_to: [chapter, outline]
---
# 测试风格
多用短句，节奏明快。
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

	// 加载
	loader := NewLoader(dir)
	skill := loader.Get("test-style")
	if skill == nil {
		t.Fatal("skill should be found")
	}
	if skill.Name != "test-style" {
		t.Errorf("Name = %q, want %q", skill.Name, "test-style")
	}
	if skill.Description != "测试写作风格" {
		t.Errorf("Description = %q", skill.Description)
	}
	if skill.Version != "1.0" {
		t.Errorf("Version = %q", skill.Version)
	}
	if len(skill.AppliesTo) != 2 || skill.AppliesTo[0] != "chapter" {
		t.Errorf("AppliesTo = %v", skill.AppliesTo)
	}
	if !strings.Contains(skill.Body, "多用短句") {
		t.Error("Body should contain skill instruction")
	}
}

func TestNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "bad-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("没有 frontmatter，直接正文"), 0644)

	loader := NewLoader(dir)
	if skill := loader.Get("bad-skill"); skill != nil {
		t.Error("skill without frontmatter should not be loaded")
	}
}

func TestListAllSkills(t *testing.T) {
	dir := t.TempDir()
	makeSkill(t, dir, "style-a", "风格A", "1.0", "chapter")
	makeSkill(t, dir, "style-b", "风格B", "2.0", "outline")

	loader := NewLoader(dir)
	skills := loader.List()
	if len(skills) != 2 {
		t.Errorf("List() returned %d skills, want 2", len(skills))
	}
}

func TestFilterByAppliesTo(t *testing.T) {
	dir := t.TempDir()
	makeSkill(t, dir, "chapter-only", "仅章节", "1.0", "chapter")
	makeSkill(t, dir, "outline-only", "仅大纲", "1.0", "outline")
	makeSkill(t, dir, "both", "两者", "1.0", "chapter", "outline")

	loader := NewLoader(dir)

	chapterSkills := loader.FilterByAppliesTo("chapter")
	if len(chapterSkills) != 2 {
		t.Errorf("chapter skills = %d, want 2", len(chapterSkills))
	}

	outlineSkills := loader.FilterByAppliesTo("outline")
	if len(outlineSkills) != 2 {
		t.Errorf("outline skills = %d, want 2", len(outlineSkills))
	}
}

func TestInjectSkill(t *testing.T) {
	dir := t.TempDir()
	makeSkill(t, dir, "inject-test", "注入测试", "1.0", "chapter")

	loader := NewLoader(dir)
	base := "你是一位小说作家。"

	result := loader.InjectSkill(base, "inject-test")
	if !strings.Contains(result, base) {
		t.Error("injected prompt should start with base prompt")
	}
	if !strings.Contains(result, "inject-test") {
		t.Error("injected prompt should contain skill name")
	}
	if !strings.Contains(result, "写作指导") {
		t.Error("injected prompt should contain 写作指导 header")
	}
}

func TestInjectSkillMissing(t *testing.T) {
	loader := NewLoader("")
	base := "你是一位小说作家。"
	result := loader.InjectSkill(base, "nonexistent")
	if result != base {
		t.Errorf("injecting nonexistent skill should return base unchanged, got %q", result)
	}
}

// ── 辅助 ────────────────────────────────────────────────────

func makeSkill(t *testing.T, dir, name, desc, version string, appliesTo ...string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	os.MkdirAll(skillDir, 0755)

	applies := "["
	for i, a := range appliesTo {
		if i > 0 {
			applies += ", "
		}
		applies += a
	}
	applies += "]"

	content := "---\nname: " + name + "\ndescription: " + desc + "\nversion: \"" + version + "\"\napplies_to: " + applies + "\n---\n\n## " + desc + "\n\n这是 " + name + " 的写作指导内容。\n"
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
}
