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

// TestBuildBranchCastSection 选角会议区段：名册命中者全量档案+主角关系缀行；
// 未命中（库挑入/新建）按负载自带信息成行；未圈定角色不渲染；空负载退回全量；
// 纯字符串数组兼容；坏 JSON 不报错。
func TestBuildBranchCastSection(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}

	got := a.buildBranchCastSection(pm, parseBranchCast(`[{"name":"林晚","relation":"师妹"},{"name":"玄铁","relation":"死敌","note":"库·冷面刀客"}]`))
	if !strings.Contains(got, "【本轮分支登场角色（作者圈定；关系为主角视角）】") {
		t.Errorf("缺选角区段头:\n%s", got)
	}
	if !strings.Contains(got, "- 林晚：主角") || !strings.Contains(got, "与主角关系：师妹") {
		t.Errorf("名册命中应全量档案+关系缀行:\n%s", got)
	}
	if !strings.Contains(got, "- 玄铁：新增角色·库·冷面刀客·与主角关系：死敌") {
		t.Errorf("未命中应按负载成行:\n%s", got)
	}
	if strings.Contains(got, "沈青") {
		t.Errorf("未圈定角色不应渲染:\n%s", got)
	}

	// 纯字符串数组兼容
	got = a.buildBranchCastSection(pm, parseBranchCast(`["林晚"]`))
	if !strings.Contains(got, "- 林晚：主角") || strings.Contains(got, "与主角关系：") {
		t.Errorf("字符串负载=只圈名字不带关系:\n%s", got)
	}
	// 空负载/坏 JSON → 全量名册
	if full := a.buildCharacterSummary(pm); a.buildBranchCastSection(pm, parseBranchCast("")) != full {
		t.Errorf("空负载应退回全量名册")
	}
	if full := a.buildCharacterSummary(pm); a.buildBranchCastSection(pm, parseBranchCast("not-json")) != full {
		t.Errorf("坏 JSON 应退回全量名册")
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
