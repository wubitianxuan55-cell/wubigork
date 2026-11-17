package context

import (
	"strings"
	"testing"
)

// T7（codex 蒸馏）验收五例（asset-08 §五）。

// guidanceSection 是测试与真实场景共用的字符串章节（asset-08
// context_window_guidance 同构）。
type guidanceSection struct{ msg string }

func (g guidanceSection) ID() string       { return "context_window_guidance" }
func (g guidanceSection) Snapshot() string { return g.msg }
func (g guidanceSection) Body() string     { return g.msg }

// ① 同一章节连续三轮快照不变：只有第一轮产生片段，后两轮零片段。
func TestSectionDiffThreeRoundsStable(t *testing.T) {
	store := NewSectionStore()
	sec := guidanceSection{msg: "输出控制在 200 行以内"}

	out1 := store.RenderAll([]Section{sec})
	if len(out1) != 1 || !strings.Contains(out1[0].Body, "200 行") {
		t.Fatalf("首轮应直出内容: %+v", out1)
	}
	for round := 2; round <= 3; round++ {
		if out := store.RenderAll([]Section{sec}); len(out) != 0 {
			t.Fatalf("第 %d 轮应零片段: %+v", round, out)
		}
	}
}

// ② 快照从「有」变「空」：恰好一条移除通知。
func TestSectionDiffRemovalNotice(t *testing.T) {
	store := NewSectionStore()
	if out := store.RenderAll([]Section{guidanceSection{"有指引"}}); len(out) != 1 {
		t.Fatalf("首轮应直出")
	}
	out := store.RenderAll([]Section{guidanceSection{""}})
	if len(out) != 1 {
		t.Fatalf("撤销应恰好一条片段: %+v", out)
	}
	if !strings.Contains(out[0].Body, sectionNoticeRemove) {
		t.Fatalf("应是移除通知: %q", out[0].Body)
	}
	// 之后保持空：不再发言。
	if out := store.RenderAll([]Section{guidanceSection{""}}); len(out) != 0 {
		t.Fatalf("持续为空不应发言: %+v", out)
	}
}

// ③ Unknown 且当前非空 → 替换通知 + 内容；Unknown 且当前空 → 移除通知。
func TestSectionDiffUnknownLegacy(t *testing.T) {
	store := NewSectionStore()
	store.MarkUnknown("context_window_guidance")
	out := store.RenderAll([]Section{guidanceSection{"新指引"}})
	if len(out) != 1 || !strings.Contains(out[0].Body, sectionNoticeReplace) || !strings.Contains(out[0].Body, "新指引") {
		t.Fatalf("Unknown+非空应替换通知+内容: %+v", out)
	}
	// 未知且撤销：保守发移除（历史可能有旧指引）。
	store2 := NewSectionStore()
	store2.MarkUnknown("context_window_guidance")
	out2 := store2.RenderAll([]Section{guidanceSection{""}})
	if len(out2) != 1 || !strings.Contains(out2[0].Body, sectionNoticeRemove) {
		t.Fatalf("Unknown+空应移除通知: %+v", out2)
	}
}

// ④ 快照改一个字符 → 恰好一条替换通知，内容完整。
func TestSectionDiffOneCharChange(t *testing.T) {
	store := NewSectionStore()
	store.RenderAll([]Section{guidanceSection{"输出控制在 200 行以内"}})
	out := store.RenderAll([]Section{guidanceSection{"输出控制在 201 行以内"}})
	if len(out) != 1 || !strings.Contains(out[0].Body, sectionNoticeReplace) {
		t.Fatalf("一字之差应替换通知: %+v", out)
	}
	if !strings.Contains(out[0].Body, "201 行") {
		t.Fatalf("替换内容应完整: %q", out[0].Body)
	}
}

// ⑤ 多章节同时变更：各出各的片段，顺序按 ID 字典序稳定（输入顺序翻转不影响）。
func TestSectionDiffMultiSectionStableOrder(t *testing.T) {
	store := NewSectionStore()
	// 首轮注册两章节。
	if out := store.RenderAll([]Section{guidanceSection{"v1"}, projectSection{root: "/w", lang: "go"}}); len(out) != 2 {
		t.Fatalf("首轮两片段: %+v", out)
	}
	// 双变更 + 输入顺序翻转：各自出替换片段，顺序仍按 ID 字典序。
	out := store.RenderAll([]Section{
		projectSection{root: "/w2", lang: "rust"},
		guidanceSection{"v2"},
	})
	out2 := store.RenderAll([]Section{
		guidanceSection{"v3"},
		projectSection{root: "/w3", lang: "go"},
	})
	if len(out) != 2 || len(out2) != 2 {
		t.Fatalf("双变更各两片段: %+v / %+v", out, out2)
	}
	var ids1, ids2 []string
	for _, f := range out {
		ids1 = append(ids1, f.Kind)
	}
	for _, f := range out2 {
		ids2 = append(ids2, f.Kind)
	}
	if strings.Join(ids1, ",") != strings.Join(ids2, ",") {
		t.Fatalf("顺序应随输入翻转保持稳定: %v vs %v", ids1, ids2)
	}
	// 字典序断言：context_window_guidance < project.state。
	if ids1[0] != "section.context_window_guidance" || ids1[1] != "section.project.state" {
		t.Fatalf("应按 ID 字典序: %v", ids1)
	}
}

// T6 联动：章节碎片 user 角色 + ProvHarness → 不可作授权证据；MatchesText 可识别。
func TestSectionFragmentContract(t *testing.T) {
	f := SectionFragment("guidance", "内容")
	if f.Prov != ProvHarness || f.Role != "user" {
		t.Fatalf("章节碎片应为 user 角色 + harness 归属: %+v", f)
	}
	if CanAuthorize(f) {
		t.Fatalf("harness 章节碎片不可作授权证据")
	}
	if !MatchesText(f.Kind, f.Render()) {
		t.Fatalf("渲染后应可被 MatchesText 识别")
	}
}

// projectSection 是 L2 项目块的章节化形状（真实消费方向的样板）。
type projectSection struct {
	root string
	lang string
}

func (p projectSection) ID() string { return "project.state" }
func (p projectSection) Snapshot() string {
	return p.root + "\x00" + p.lang
}
func (p projectSection) Body() string {
	var b strings.Builder
	b.WriteString("## Project\n")
	if p.root != "" {
		b.WriteString("- Root: " + p.root + "\n")
	}
	if p.lang != "" {
		b.WriteString("- Language: " + p.lang + "\n")
	}
	return b.String()
}
