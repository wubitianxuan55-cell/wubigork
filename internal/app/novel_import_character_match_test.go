package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// 角色名→角色库 ID 匹配（v4.290，拆书线欠账：反推条目只带角色名，
// 大纲节点 Characters 字段收角色 ID）。

func testChars() []types.Character {
	return []types.Character{
		{ID: "c1", Name: "林晚儿"},
		{ID: "c2", Name: "陈默"},
		{ID: "c3", Name: "玄天宗"},
	}
}

func TestMatchCharacterIDs(t *testing.T) {
	// 精确名
	if got := matchCharacterIDs([]string{"陈默"}, testChars()); len(got) != 1 || got[0] != "c2" {
		t.Fatalf("精确名应命中: %v", got)
	}
	// 双向包含：反推给「林晚」、库里「林晚儿」
	if got := matchCharacterIDs([]string{"林晚"}, testChars()); len(got) != 1 || got[0] != "c1" {
		t.Fatalf("包含式兜底应命中: %v", got)
	}
	// 未知名静默跳过（不编造角色）
	if got := matchCharacterIDs([]string{"路人甲"}, testChars()); got != nil {
		t.Fatalf("未知名不应产出 ID: %v", got)
	}
	// 去重保序
	if got := matchCharacterIDs([]string{"陈默", "陈默", "林晚"}, testChars()); len(got) != 2 || got[0] != "c2" || got[1] != "c1" {
		t.Fatalf("去重保序不符: %v", got)
	}
	// 空输入
	if matchCharacterIDs(nil, testChars()) != nil || matchCharacterIDs([]string{"陈默"}, nil) != nil {
		t.Fatal("空输入应返回 nil")
	}
}

// 端到端：反推条目带角色名 → apply 后命中章的节点 Characters 收 ID，
// 既有 ID 保留在前；角色库为空时节点不动。
func TestNovelOutlineReconstructApply_CharacterMatch(t *testing.T) {
	a := newSkeletonTestApp(t) // 60 章导入工程（走章级条目）
	if a == nil {
		t.Fatal("fixture")
	}
	// 建角色库：导入工程无角色，模拟用户后续建的两个角色
	if err := a.getPM().WriteCharacters(&types.CharacterFile{Characters: []types.Character{
		{ID: "ch-lin", Name: "林晚", RoleType: "protagonist", Status: "Alive"},
		{ID: "ch-chen", Name: "陈默", RoleType: "supporting", Status: "Alive"},
	}}); err != nil {
		t.Fatalf("写角色库: %v", err)
	}

	items := []OutlineReconstructItem{
		{ChapterNumber: 1, Title: "第1章 试炼", Summary: "开场。", Characters: []string{"林晚", "陈默"}},
		{ChapterNumber: 2, Title: "第2章 试炼", Summary: "推进。"},
	}
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.NovelOutlineReconstructApply(string(raw)); err != nil {
		t.Fatalf("应用: %v", err)
	}

	of, err := a.getPM().ReadOutlines()
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range of.Nodes {
		if node.ID != "imp-001" {
			continue
		}
		if len(node.Characters) != 2 || node.Characters[0] != "ch-lin" || node.Characters[1] != "ch-chen" {
			t.Fatalf("命中章应收到匹配 ID: %+v", node.Characters)
		}
	}
}

// 既有 ID 保留：节点已有角色 ID 时，匹配只增不删。
func TestNovelOutlineReconstructApply_CharacterMergePreservesExisting(t *testing.T) {
	a := newSkeletonTestApp(t)
	if err := a.getPM().WriteCharacters(&types.CharacterFile{Characters: []types.Character{
		{ID: "ch-chen", Name: "陈默", RoleType: "supporting", Status: "Alive"},
	}}); err != nil {
		t.Fatal(err)
	}
	// 预置节点既有角色
	of, _ := a.getPM().ReadOutlines()
	for i := range of.Nodes {
		if of.Nodes[i].ChapterFile == "001.md" {
			of.Nodes[i].Characters = []string{"ch-existing"}
		}
	}
	a.getPM().WriteOutlines(of)

	items := []OutlineReconstructItem{
		{ChapterNumber: 1, Title: "第1章 试炼", Summary: "开场。", Characters: []string{"陈默"}},
	}
	raw, _ := json.Marshal(items)
	if _, err := a.NovelOutlineReconstructApply(string(raw)); err != nil {
		t.Fatal(err)
	}
	of2, _ := a.getPM().ReadOutlines()
	for _, node := range of2.Nodes {
		if node.ChapterFile == "001.md" {
			joined := strings.Join(node.Characters, ",")
			if !strings.Contains(joined, "ch-existing") || !strings.Contains(joined, "ch-chen") {
				t.Fatalf("既有 ID 应保留且新增追加: %v", node.Characters)
			}
			if node.Characters[0] != "ch-existing" {
				t.Fatalf("既有 ID 应在前: %v", node.Characters)
			}
		}
	}
}
