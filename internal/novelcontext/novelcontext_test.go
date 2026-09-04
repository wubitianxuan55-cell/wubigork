package novelcontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/graph"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newTestProject 建一个最小 v4 项目：
// 角色甲（主角，藏着秘密）、角色乙（配角）、世界观、伏笔、大纲主线、一个场景。
// povID 指定该场景的 POV 角色 ID。返回 pm 与场景对象。
func newTestProject(t *testing.T, povID string) (*project.Manager, *types.Scene) {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试小说", "玄幻", "冷峻", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	if err := pm.WriteCharacters(&types.CharacterFile{Characters: []types.Character{
		{ID: "char-a", Name: "甲", RoleType: "protagonist", Status: "Alive"},
		{ID: "char-b", Name: "乙", RoleType: "supporting", Status: "Alive"},
	}}); err != nil {
		t.Fatalf("写角色: %v", err)
	}

	if err := pm.WriteWorldviewFile(&types.WorldviewFile{Sections: []types.WorldviewSection{
		{ID: "era", Title: "时代背景", Content: "灵气复苏三百年，九州大陆王朝更迭频繁", Order: 1},
	}}); err != nil {
		t.Fatalf("写世界观: %v", err)
	}

	if err := pm.WriteForeshadows(&types.ForeshadowFile{Items: []types.Foreshadow{
		{ID: "f1", Category: "plot", Description: "主角背上有一道古剑胎记", PlantedIn: "001.md", Status: types.ForeshadowPlanted, IsLongTerm: true},
		{ID: "f2", Category: "character", Description: "酒馆老板总在打听商队路线", PlantedIn: "002.md", Status: types.ForeshadowHinted},
		{ID: "f3", Category: "world", Description: "北境冰湖下的旧神（已回收）", PlantedIn: "001.md", RevealedIn: "003.md", Status: types.ForeshadowRevealed},
	}}); err != nil {
		t.Fatalf("写伏笔: %v", err)
	}

	if err := pm.WriteOutlines(&types.OutlineFile{
		StoryThread: "主角寻找失落的剑心，重铸剑宗荣光",
		Nodes:       []types.OutlineNode{},
	}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	// 章节 cast：让两名角色都在本章子图内（即使未写入正文）。
	if err := pm.WriteChapterSummary(1, &types.ChapterSummary{
		Title: "第一章 重逢", Summary: "甲与乙重逢于青阳城。",
		CharactersAppeared: []string{"甲", "乙"}, EmotionTone: "凝重",
	}); err != nil {
		t.Fatalf("写章摘要: %v", err)
	}

	// 实体数据库：甲带一条秘密（known_by=甲，即只有甲知道）。
	db := &graph.EntityDB{Entities: []graph.Entity{
		{ID: "char-a", Name: "甲", Type: graph.EntityCharacter, Properties: map[string]string{
			"role_type": "protagonist", "status": "Alive", "known_by": "甲", "secret": "圣物在古井",
		}},
		{ID: "char-b", Name: "乙", Type: graph.EntityCharacter, Properties: map[string]string{
			"role_type": "supporting", "status": "Alive", "location": "青阳城",
		}},
	}}
	if err := db.Save(pm.Dir); err != nil {
		t.Fatalf("写实体库: %v", err)
	}

	// 创建场景并设置 POV。
	sm := pm.SceneManager(1)
	s, err := sm.Create("opening", "开场")
	if err != nil {
		t.Fatalf("创建场景: %v", err)
	}
	s.Meta.POVCharID = povID
	s.Meta.Location = "青阳城"
	s.Meta.TimeOfDay = "黄昏"
	s.Meta.Emotion = "凝重"
	s.Meta.Tags = []string{"dialogue", "reunion"}
	s.Content = "乙与甲并肩站在青阳城头。风吹过，乙沉默不语。"
	if err := sm.Write(s); err != nil {
		t.Fatalf("写场景: %v", err)
	}

	// 回读，确保落盘一致。
	got, err := sm.Read(s.Meta.ID)
	if err != nil {
		t.Fatalf("读场景: %v", err)
	}
	return pm, got
}

func TestCompileSceneBible_Basic(t *testing.T) {
	pm, s := newTestProject(t, "char-b")

	b, err := CompileSceneBible(pm, 1, s)
	if err != nil {
		t.Fatalf("编译场景圣经: %v", err)
	}
	if b == nil {
		t.Fatalf("返回的圣经不应为 nil")
	}

	// 世界观要点
	if !strings.Contains(b.Setting, "时代背景") || !strings.Contains(b.Setting, "灵气复苏") {
		t.Errorf("Setting 应含世界观要点，got: %q", b.Setting)
	}
	// 出场角色
	if len(b.Characters) == 0 {
		t.Fatalf("应至少有一个出场角色")
	}
	foundA, foundB := false, false
	for _, c := range b.Characters {
		if c.Name == "甲" {
			foundA = true
			if c.RoleType != "protagonist" && c.RoleType != "主角" {
				t.Errorf("甲的定位应为 protagonist/主角，got %q", c.RoleType)
			}
		}
		if c.Name == "乙" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Errorf("出场角色应包含甲/乙，got %+v", b.Characters)
	}
	// 未回收伏笔（planted/hinted 注入，revealed 过滤）
	if len(b.Foreshadows) == 0 {
		t.Fatalf("应有未回收伏笔")
	}
	for _, f := range b.Foreshadows {
		if strings.Contains(f, "旧神") {
			t.Errorf("已回收伏笔不应出现: %q", f)
		}
	}
	// 时间锚点
	if !strings.Contains(b.TimeAnchor, "黄昏") {
		t.Errorf("时间锚点应含场景时间，got %q", b.TimeAnchor)
	}
	// 故事主线
	if !strings.Contains(b.Thread, "剑心") {
		t.Errorf("故事主线应注入，got %q", b.Thread)
	}

	// Render 非空且不超预算
	rendered := b.Render(2000)
	if rendered == "" {
		t.Fatalf("Render 不应为空")
	}
	if n := len([]rune(rendered)); n > 2000 {
		t.Errorf("Render 超预算: got %d rune, want <= 2000", n)
	}
	// 空区段不输出空标题
	if strings.Contains(rendered, "## \n") {
		t.Errorf("Render 出现空区段标题:\n%s", rendered)
	}
}

// TestPOVMask_HidesSecret 核心差异化断言：
// 甲知道秘密 S；当 POV=乙（B）时，S 必须出现在 HiddenFacts 且不进 POVView；
// 当 POV=甲（A）时，S 进 POVView 且不进 HiddenFacts。
func TestPOVMask_HidesSecret(t *testing.T) {
	secret := "圣物在古井"

	// POV = 乙（未被告知秘密）
	pmB, sceneB := newTestProject(t, "char-b")
	bB, err := CompileSceneBible(pmB, 1, sceneB)
	if err != nil {
		t.Fatalf("编译(乙POV): %v", err)
	}
	if containsAny(bB.POVView, secret) {
		t.Errorf("POV=乙 时 POVView 不应含秘密 %q，实际:\n%s", secret, bB.POVView)
	}
	if !sliceContains(bB.HiddenFacts, secret) {
		t.Errorf("POV=乙 时 HiddenFacts 应含秘密 %q，实际: %v", secret, bB.HiddenFacts)
	}

	// POV = 甲（知晓秘密）
	pmA, sceneA := newTestProject(t, "char-a")
	bA, err := CompileSceneBible(pmA, 1, sceneA)
	if err != nil {
		t.Fatalf("编译(甲POV): %v", err)
	}
	if !containsAny(bA.POVView, secret) {
		t.Errorf("POV=甲 时 POVView 应含秘密 %q，实际:\n%s", secret, bA.POVView)
	}
	if sliceContains(bA.HiddenFacts, secret) {
		t.Errorf("POV=甲 时 HiddenFacts 不应含秘密 %q，实际: %v", secret, bA.HiddenFacts)
	}
}

// TestCompile_CorruptRead 文件缺失 / 损坏时必须静默降级：不 panic、不报错、
// 返回可用的空字段圣经。
func TestCompile_CorruptRead(t *testing.T) {
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "损坏测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	// 损坏 / 缺失各类文件
	if err := os.WriteFile(filepath.Join(pm.Dir, "foreshadows.json"), []byte("{broken"), 0644); err != nil {
		t.Fatalf("写损坏伏笔: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pm.Dir, "worldview.json"), []byte("not json"), 0644); err != nil {
		t.Fatalf("写损坏世界观: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pm.Dir, "characters.json"), []byte("[[]]"), 0644); err != nil {
		t.Fatalf("写损坏角色: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pm.Dir, "outline.json"), []byte("oops"), 0644); err != nil {
		t.Fatalf("写损坏大纲: %v", err)
	}

	// 场景正文缺失（只有 meta，无对应 .md）也要能编译。
	sm := pm.SceneManager(1)
	s, err := sm.Create("broken", "残缺场景")
	if err != nil {
		t.Fatalf("创建场景: %v", err)
	}
	s.Meta.POVCharID = "ghost"
	s.Meta.Location = "无"
	if err := sm.Write(s); err != nil {
		t.Fatalf("写场景: %v", err)
	}
	// 删掉正文文件，模拟缺失。
	if err := os.Remove(filepath.Join(pm.Dir, "chapters", "001", "scenes", s.Meta.ID+".md")); err != nil {
		t.Fatalf("删正文: %v", err)
	}

	b, err := CompileSceneBible(pm, 1, s)
	if err != nil {
		t.Fatalf("损坏数据下编译不应报错，got: %v", err)
	}
	if b == nil {
		t.Fatalf("损坏数据下仍应返回非 nil 圣经")
	}
	if b.Setting != "" {
		t.Errorf("损坏世界观应降级为空 Setting，got %q", b.Setting)
	}
	if len(b.Foreshadows) != 0 {
		t.Errorf("损坏伏笔应降级为空，got %v", b.Foreshadows)
	}
	if len(b.Characters) != 0 {
		t.Errorf("损坏角色应降级为空，got %v", b.Characters)
	}
	// Render 在空字段下也要能安全调用。
	rendered := b.Render(2000)
	if n := len([]rune(rendered)); n > 2000 {
		t.Errorf("损坏数据下 Render 超预算: got %d", n)
	}
}

// TestBuildSceneBibleFromChapter 无 scene 时用章信息合成整章圣经，不崩溃。
func TestBuildSceneBibleFromChapter(t *testing.T) {
	pm, _ := newTestProject(t, "char-b")
	b, err := BuildSceneBibleFromChapter(pm, 1)
	if err != nil {
		t.Fatalf("整章编译: %v", err)
	}
	if b == nil {
		t.Fatalf("整章编译不应返回 nil")
	}
	if len(b.Characters) == 0 {
		// 合成场景时可能未解析出角色，但不应崩溃
		t.Logf("整章合成未解析出角色（可接受）")
	}
	if b.Render(0) == "" {
		t.Errorf("整章圣经 Render 应非空")
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if n != "" && strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

func sliceContains(haystack []string, needle string) bool {
	for _, n := range haystack {
		if n != "" && strings.Contains(n, needle) {
			return true
		}
	}
	return false
}
