package types

import (
	"testing"
)

// ── ToMarkdown ────────────────────────────────────────────

func TestWorldviewFile_ToMarkdown_Normal(t *testing.T) {
	wf := &WorldviewFile{
		Sections: []WorldviewSection{
			{Title: "时代背景", Content: "这是时代背景内容。"},
			{Title: "地理环境", Content: "这是地理环境内容。"},
		},
	}
	got := wf.ToMarkdown()
	want := "## 时代背景\n\n这是时代背景内容。\n\n## 地理环境\n\n这是地理环境内容。\n\n"
	if got != want {
		t.Fatalf("ToMarkdown mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestWorldviewFile_ToMarkdown_SkipsEmptyContent(t *testing.T) {
	wf := &WorldviewFile{
		Sections: []WorldviewSection{
			{Title: "时代背景", Content: "有内容"},
			{Title: "空节", Content: ""},
			{Title: "文化", Content: "文化内容"},
		},
	}
	got := wf.ToMarkdown()
	// 空节应被跳过
	if got != "## 时代背景\n\n有内容\n\n## 文化\n\n文化内容\n\n" {
		t.Fatalf("空节未被跳过: %q", got)
	}
}

func TestWorldviewFile_ToMarkdown_AllEmpty(t *testing.T) {
	wf := &WorldviewFile{
		Sections: []WorldviewSection{
			{Title: "无内容节", Content: ""},
		},
	}
	if got := wf.ToMarkdown(); got != "" {
		t.Fatalf("全空应返回空串: %q", got)
	}
}

func TestWorldviewFile_ToMarkdown_NilSections(t *testing.T) {
	wf := &WorldviewFile{}
	if got := wf.ToMarkdown(); got != "" {
		t.Fatalf("nil Sections 应返回空串: %q", got)
	}
}

// ── E02：PromptView 剧照剥离 ──────────────────────────────

func TestCharacterPromptView_StripsPortrait(t *testing.T) {
	orig := Character{ID: "mc", Name: "林晚", PortraitURL: "data:image/png;base64,AAAA"}
	view := orig.PromptView()
	if view.PortraitURL != "" {
		t.Errorf("PromptView.PortraitURL = %q, want 空", view.PortraitURL)
	}
	if view.Name != "林晚" {
		t.Errorf("PromptView.Name = %q", view.Name)
	}
	if orig.PortraitURL != "data:image/png;base64,AAAA" {
		t.Error("PromptView 不应修改原角色")
	}
}

func TestCharacterFilePromptView_DeepCopy(t *testing.T) {
	cf := &CharacterFile{
		Characters: []Character{
			{ID: "a", Name: "甲", PortraitURL: "data:image/png;base64,BBBB"},
			{ID: "b", Name: "乙"},
		},
		Organizations: []Organization{{ID: "org1", Name: "青云宗"}},
		Relationships: []Relationship{{FromID: "a", ToID: "b", RelationType: "friend"}},
	}
	view := cf.PromptView()
	if view.Characters[0].PortraitURL != "" {
		t.Error("PromptView 未剥离剧照")
	}
	if len(view.Organizations) != 1 || view.Organizations[0].Name != "青云宗" {
		t.Errorf("组织未保留: %+v", view.Organizations)
	}
	if len(view.Relationships) != 1 {
		t.Errorf("关系未保留: %+v", view.Relationships)
	}
	if cf.Characters[0].PortraitURL != "data:image/png;base64,BBBB" {
		t.Error("PromptView 修改了原文件")
	}
}

// ── 常量枚举 ──────────────────────────────────────────────

func TestOutlineNodeStatus_Constants(t *testing.T) {
	if OutlinePlanned != "planned" {
		t.Fatalf("OutlinePlanned: got %q", OutlinePlanned)
	}
	if OutlineWriting != "writing" {
		t.Fatalf("OutlineWriting: got %q", OutlineWriting)
	}
	if OutlineDone != "done" {
		t.Fatalf("OutlineDone: got %q", OutlineDone)
	}
	if OutlineAbandoned != "abandoned" {
		t.Fatalf("OutlineAbandoned: got %q", OutlineAbandoned)
	}
}

func TestForeshadowStatus_Constants(t *testing.T) {
	if ForeshadowPlanted != "planted" {
		t.Fatalf("ForeshadowPlanted: got %q", ForeshadowPlanted)
	}
	if ForeshadowHinted != "hinted" {
		t.Fatalf("ForeshadowHinted: got %q", ForeshadowHinted)
	}
	if ForeshadowRevealed != "revealed" {
		t.Fatalf("ForeshadowRevealed: got %q", ForeshadowRevealed)
	}
}

func TestSceneStatus_Constants(t *testing.T) {
	if SceneDraft != "draft" {
		t.Fatalf("SceneDraft: got %q", SceneDraft)
	}
	if SceneRevising != "revising" {
		t.Fatalf("SceneRevising: got %q", SceneRevising)
	}
	if SceneDone != "done" {
		t.Fatalf("SceneDone: got %q", SceneDone)
	}
	if ScenePaused != "paused" {
		t.Fatalf("ScenePaused: got %q", ScenePaused)
	}
}

func TestContextPriority_Constants(t *testing.T) {
	if PriorityP0 != "P0" {
		t.Fatalf("PriorityP0: got %q", PriorityP0)
	}
	if PriorityP1 != "P1" {
		t.Fatalf("PriorityP1: got %q", PriorityP1)
	}
	if PriorityP2 != "P2" {
		t.Fatalf("PriorityP2: got %q", PriorityP2)
	}
}
