package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newCastTestProject 建一个带三名角色的测试项目（主角/配角/反派）。
func newCastTestProject(t *testing.T) *project.Manager {
	t.Helper()
	pm := newStatesTestProject(t)
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive",
				Personality: "清冷", Background: "没落剑修", Motivation: "查灭门真相"},
			{ID: "c2", Name: "沈青", RoleType: "supporting", Status: "Alive", Personality: "圆滑"},
			{ID: "c3", Name: "鬼手", RoleType: "antagonist", Status: "Alive", Personality: "阴鸷"},
		},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	return pm
}

// TestBuildChapterCastSection_FocusTiers 计划有角色焦点时装配两级：焦点角色
// 全量档案（登场），其余降名册（备查不登场）。
func TestBuildChapterCastSection_FocusTiers(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	plan := &types.ChapterPlan{CharacterFocus: []string{"林晚"}}

	got := a.buildChapterCastSection(pm, plan)
	if !strings.Contains(got, "【本章出场角色（按计划角色焦点，剧情围绕他们展开）】") {
		t.Errorf("缺焦点区段头:\n%s", got)
	}
	if !strings.Contains(got, "- 林晚：主角") || !strings.Contains(got, "查灭门真相") {
		t.Errorf("焦点角色应给全量档案:\n%s", got)
	}
	if !strings.Contains(got, "【其余名册（备查") {
		t.Errorf("缺名册区段头:\n%s", got)
	}
	if !strings.Contains(got, "- 沈青：配角") || !strings.Contains(got, "- 鬼手：反派") {
		t.Errorf("其余角色应降名册:\n%s", got)
	}
	// 名册是紧凑行：不携带焦点级档案字段（动机）
	if strings.Contains(got, "名册") && strings.Count(got, "目标：") > 1 {
		t.Errorf("名册行不应携带完整档案:\n%s", got)
	}
}

// TestBuildChapterCastSection_FocusTolerantMatch 焦点项带缀饰也能匹配（作者手写
// 「林晚（主角）」这类形态）。
func TestBuildChapterCastSection_FocusTolerantMatch(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	plan := &types.ChapterPlan{CharacterFocus: []string{"林晚（主角）"}}

	got := a.buildChapterCastSection(pm, plan)
	if !strings.Contains(got, "查灭门真相") {
		t.Errorf("缀饰焦点应命中角色档案:\n%s", got)
	}
}

// TestBuildChapterCastSection_NoPlanOrNoMatch 无计划/焦点为空/焦点没命中任何
// 角色时退回全量名册（旧口径），绝不空窗。
func TestBuildChapterCastSection_NoPlanOrNoMatch(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}

	full := a.buildCharacterSummary(pm)
	if got := a.buildChapterCastSection(pm, nil); got != full {
		t.Errorf("无计划应等于旧全量口径")
	}
	if got := a.buildChapterCastSection(pm, &types.ChapterPlan{}); got != full {
		t.Errorf("空焦点应等于旧全量口径")
	}
	noHit := a.buildChapterCastSection(pm, &types.ChapterPlan{CharacterFocus: []string{"路人甲"}})
	if !strings.Contains(noHit, "- 林晚") || !strings.Contains(noHit, "- 沈青") || !strings.Contains(noHit, "- 鬼手") {
		t.Errorf("焦点未命中应退回全量名册:\n%s", noHit)
	}
}
