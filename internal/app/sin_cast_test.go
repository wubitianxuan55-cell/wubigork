package app

// 原罪 × 角色库（v4.257）：角色选择持久化 + 提示词注入 + 硬隔离根目录断言。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/characterlib"
)

// newSinCastTestApp 在原罪测试 App 上接角色库（临时目录），并把用户配置目录
// 重定向到临时 home——sinRoot()（数据/插图/角色配置）必须落在临时目录，
// 否则测试会写进真实 %AppData%\gaea。
func newSinCastTestApp(t *testing.T) (*App, *characterlib.Store) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)         // Windows: os.UserConfigDir 来源
	t.Setenv("XDG_CONFIG_HOME", home) // Linux/macOS 兼容

	a := newSinTestApp(t)
	lib := characterlib.NewStore(filepath.Join(t.TempDir(), "characterlib"))
	t.Cleanup(func() { _ = lib.Close() })
	a.charLib = lib
	return a, lib
}

func seedCharacter(t *testing.T, lib *characterlib.Store, id, name string) {
	t.Helper()
	if err := lib.Upsert(&characterlib.Character{
		ID: id, Name: name, Gender: "female", Age: "27",
		Appearance: "短发，右眉骨有一道旧疤", Figure: "高挑",
		Personality: "冷静、话说得少", Background: "夜班记者",
		VoiceGuide: "句子短，不解释",
	}); err != nil {
		t.Fatalf("Upsert(%s): %v", id, err)
	}
}

// TestSinCastRoundTrip 角色选择往返：去重/悬空 id 丢弃/保序；清空即移除。
func TestSinCastRoundTrip(t *testing.T) {
	a, lib := newSinCastTestApp(t)
	seedCharacter(t, lib, "c1", "林夏")
	seedCharacter(t, lib, "c2", "周野")
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}

	saved, err := a.SinCastSet(story.ID, []string{"c2", "c1", "c1", "ghost", ""})
	if err != nil {
		t.Fatalf("SinCastSet: %v", err)
	}
	if strings.Join(saved, ",") != "c2,c1" {
		t.Fatalf("生效清单 = %v, want [c2 c1]（去重保序 + 悬空 id 丢弃）", saved)
	}
	got, err := a.SinCastGet(story.ID)
	if err != nil {
		t.Fatalf("SinCastGet: %v", err)
	}
	if strings.Join(got, ",") != "c2,c1" {
		t.Fatalf("读回 = %v, want [c2 c1]", got)
	}

	if _, err := a.SinCastSet(story.ID, nil); err != nil {
		t.Fatalf("清空: %v", err)
	}
	got, err = a.SinCastGet(story.ID)
	if err != nil {
		t.Fatalf("SinCastGet(清空后): %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("清空后 = %v, want 空", got)
	}
}

// TestSinCastGuardsUnknownTopic 非 sin 话题（聊天话题）不得写/读角色配置。
func TestSinCastGuardsUnknownTopic(t *testing.T) {
	a, lib := newSinCastTestApp(t)
	seedCharacter(t, lib, "c1", "林夏")
	chatTopic, err := a.ChatTopicCreate("闲聊", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if _, err := a.SinCastSet(chatTopic.ID, []string{"c1"}); err == nil {
		t.Error("SinCastSet(聊天话题) 应报错")
	}
	if _, err := a.SinCastGet(chatTopic.ID); err == nil {
		t.Error("SinCastGet(聊天话题) 应报错")
	}
}

// TestSinCastCapsPerStory 单故事角色数封顶（防提示词被角色设定吃满）。
func TestSinCastCapsPerStory(t *testing.T) {
	a, lib := newSinCastTestApp(t)
	ids := make([]string, 0, sinCastMaxPerStory+3)
	for i := 0; i < sinCastMaxPerStory+3; i++ {
		id := "c" + string(rune('a'+i))
		seedCharacter(t, lib, id, "角色"+id)
		ids = append(ids, id)
	}
	story, err := a.SinTopicCreate("群像")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	saved, err := a.SinCastSet(story.ID, ids)
	if err != nil {
		t.Fatalf("SinCastSet: %v", err)
	}
	if len(saved) != sinCastMaxPerStory {
		t.Fatalf("生效数 = %d, want %d", len(saved), sinCastMaxPerStory)
	}
}

// TestSinCastBlockInjectsCharacterProfile 角色库内容进提示词：名字与设定字段
// 必须在「前情回顾」之前出现（写作锚点优先），空字段不占位。
func TestSinCastBlockInjectsCharacterProfile(t *testing.T) {
	a, lib := newSinCastTestApp(t)
	seedCharacter(t, lib, "c1", "林夏")
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinCastSet(story.ID, []string{"c1"}); err != nil {
		t.Fatalf("SinCastSet: %v", err)
	}
	cast := a.sinCastCharacters(story.ID)
	if len(cast) != 1 || cast[0].Name != "林夏" {
		t.Fatalf("角色详情 = %+v", cast)
	}
	block := sinCastBlock(cast)
	for _, want := range []string{"林夏", "短发，右眉骨有一道旧疤", "冷静、话说得少", "夜班记者", "句子短，不解释"} {
		if !strings.Contains(block, want) {
			t.Errorf("角色块缺少 %q:\n%s", want, block)
		}
	}
	prompt := buildSinUserPrompt(nil, "继续写", cast)
	if !strings.Contains(prompt, "林夏") {
		t.Fatalf("单轮提示未带角色设定:\n%s", prompt)
	}
	if idx := strings.Index(prompt, "本故事角色"); idx < 0 {
		t.Fatalf("缺少角色块标题:\n%s", prompt)
	} else if idx > strings.Index(prompt, "【本次指令】") {
		t.Fatalf("角色块应排在本次指令之前:\n%s", prompt)
	}
	// 未选角色 = 与旧行为逐字一致（无角色块）
	if plain := buildSinUserPrompt(nil, "继续写", nil); strings.Contains(plain, "本故事角色") {
		t.Fatalf("未选角色不应出现角色块:\n%s", plain)
	}
}

// TestSinStorageRootsAreIsolated 硬隔离证据：原罪数据/插图目录落在用户配置
// 目录下，不在办公工作区内（不写 .gaea、不依赖办公 ImageSaveDir）。
func TestSinStorageRootsAreIsolated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	root := sinRoot()
	art := sinArtDir()
	if !strings.HasPrefix(root, home) {
		t.Fatalf("原罪数据根应落在用户配置目录: %s（home=%s）", root, home)
	}
	if !strings.HasPrefix(art, root) {
		t.Fatalf("插图目录应在原罪数据根内: %s", art)
	}
	if strings.Contains(root, string(os.PathSeparator)+".gaea") {
		t.Fatalf("原罪数据根不得落在办公工作区 .gaea 下: %s", root)
	}
	if !strings.HasSuffix(filepath.ToSlash(art), "/gaea/sin/art") {
		t.Fatalf("插图目录命名应为 <配置目录>/gaea/sin/art: %s", art)
	}
}
