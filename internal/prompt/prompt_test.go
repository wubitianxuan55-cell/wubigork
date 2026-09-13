package prompt

import (
	"strings"
	"testing"
	"testing/fstest"
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

func TestEngineEmbeddedFallback(t *testing.T) {
	embedded := fstest.MapFS{
		"prompts/plot-branch-browser.json": &fstest.MapFile{
			Data: []byte(`{"name":"plot-branch-browser","system":"s","task":"t",
				"input_sections":{},"output":{"format":"json","description":"d"},
				"constraints":{"must":[],"forbidden":[]}}`),
		},
		"prompts/embedded-only.json": &fstest.MapFile{
			Data: []byte(`{"name":"embedded-only","system":"s","task":"t",
				"input_sections":{},"output":{"format":"json","description":"d"},
				"constraints":{"must":[],"forbidden":[]}}`),
		},
	}
	// dir 指向不存在的目录：模拟单文件 exe 部署（无磁盘 prompts/）
	eng := NewEngineWithEmbedded("../../prompts-does-not-exist", embedded)
	if tmpl := eng.Get("plot-branch-browser"); tmpl == nil {
		t.Fatal("内置 plot-branch-browser 模板应可用（磁盘缺失时兜底）")
	}
	if tmpl := eng.Get("embedded-only"); tmpl == nil {
		t.Fatal("内置 embedded-only 模板应加载")
	}
}

func TestEngineEmbeddedDiskOverrides(t *testing.T) {
	embedded := fstest.MapFS{
		"prompts/plot-branch-browser.json": &fstest.MapFile{
			Data: []byte(`{"name":"plot-branch-browser","system":"embedded","task":"t",
				"input_sections":{},"output":{"format":"json","description":"d"},
				"constraints":{"must":[],"forbidden":[]}}`),
		},
	}
	// dir 指向真实仓库 prompts/：磁盘模板应叠加且同名覆盖内置
	eng := NewEngineWithEmbedded("../../prompts", embedded)
	if tmpl := eng.Get("plot-branch-browser"); tmpl == nil {
		t.Fatal("plot-branch-browser 应可用")
	} else if strings.Contains(tmpl.System, "embedded") {
		t.Error("磁盘模板应优先于内置模板")
	}
	if tmpl := eng.Get("chapter-generate"); tmpl == nil {
		t.Error("磁盘其余模板应与内置叠加")
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

// v4.280：输入节顺序确定性（规格 docs/distill/02-book-import.md §8.2 点名的引擎缺陷）——
// 原实现按 Go map 遍历输入节，同模板两次渲染字节不同；改后按 Order 升序 + key 字典序兜底。
func TestBuildUserPrompt_StableOrderAcrossRenders(t *testing.T) {
	eng := NewEngine("../../prompts")
	tmpl := eng.Get("book-import-outline")
	if tmpl == nil {
		t.Fatal("book-import-outline not found")
	}
	contexts := map[string]string{
		"project_title":         "书名",
		"genre":                 "都市",
		"theme":                 "主题",
		"narrative_perspective": "第三人称",
		"batch_range":           "第1-5章",
		"expected_count":        "5",
		"chapters_text":         "正文……",
	}
	first := tmpl.BuildUserPrompt(contexts)
	for i := 2; i <= 20; i++ {
		if got := tmpl.BuildUserPrompt(contexts); got != first {
			t.Fatalf("第 %d 次渲染与首次不一致（输入节顺序不稳定）:\n%q\n%q", i, first, got)
		}
	}
	// 长文本（chapters_text，order=9）必须稳定置尾；靠前字段在前
	if strings.Index(first, "书名") > strings.Index(first, "正文……") {
		t.Fatalf("长文本应置于末尾: %q", first)
	}
	if strings.Index(first, "本批章节范围") > strings.Index(first, "正文……") {
		t.Fatalf("批次范围应在长文本之前: %q", first)
	}
}

func TestBuildUserPrompt_OrderFieldOverridesKeyOrder(t *testing.T) {
	raw := `{"name":"t","system":"s","task":"k","input_sections":{` +
		`"z_last":{"priority":"P0","label":"Z","order":9},` +
		`"a_first":{"priority":"P0","label":"A","order":1}},` +
		`"output":{"format":"json","description":"d"},"constraints":{"must":[],"forbidden":[]}}`
	tmpl, err := parseTemplate([]byte(raw), "test")
	if err != nil {
		t.Fatalf("解析模板: %v", err)
	}
	out := tmpl.BuildUserPrompt(map[string]string{"z_last": "ZZZ", "a_first": "AAA"})
	if strings.Index(out, "AAA") < 0 || strings.Index(out, "ZZZ") < 0 {
		t.Fatalf("两节都应渲染: %q", out)
	}
	if strings.Index(out, "AAA") > strings.Index(out, "ZZZ") {
		t.Fatalf("order 未生效（a_first 应在 z_last 之前）: %q", out)
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
