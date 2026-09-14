package novelcontext

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/graph"
	"github.com/gaea/gaea/internal/types"
)

// TestEntityCharacterStateRoundtrip 状态机字段走既有实体属性管道
// （t5 §7.4-1）：entityFromCharacter 写入 → sceneCharFromEntity 读回 →
// formatSceneChar 渲染，全程零新增管道。
func TestEntityCharacterStateRoundtrip(t *testing.T) {
	c := types.Character{
		ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive",
		CurrentState:    "剑心重铸，坚定（第12章）",
		MainCareerID:    "剑修",
		MainCareerStage: 3,
		SubCareers:      []types.CharacterCareerRef{{CareerID: "炼丹师", Stage: 2}},
	}
	e := entityFromCharacter(c, c.ID)
	if e.Properties["current_state"] != c.CurrentState {
		t.Errorf("current_state 未写入实体属性: %q", e.Properties["current_state"])
	}
	if e.Properties["career_main"] != "剑修·3阶" {
		t.Errorf("career_main = %q, want 剑修·3阶", e.Properties["career_main"])
	}

	charByName := map[string]types.Character{c.Name: c}
	sc := sceneCharFromEntity(e, charByName)
	if sc.CurrentState != c.CurrentState {
		t.Errorf("CurrentState 读回 = %q", sc.CurrentState)
	}
	if sc.CareerMain != "剑修·3阶" {
		t.Errorf("CareerMain 读回 = %q", sc.CareerMain)
	}
	if len(sc.CareerSub) != 1 || sc.CareerSub[0] != "炼丹师·2阶" {
		t.Errorf("CareerSub 读回 = %v", sc.CareerSub)
	}

	line := formatSceneChar(sc)
	for _, want := range []string{"当前状态: 剑心重铸，坚定（第12章）", "主职业: 剑修·3阶", "副职业: 炼丹师·2阶"} {
		if !strings.Contains(line, want) {
			t.Errorf("渲染缺 %q，got: %s", want, line)
		}
	}
}

// TestSceneCharStateFallback 角色文件兜底：实体属性缺失（实体库未回灌）时
// 从 characters.json 取状态机字段。
func TestSceneCharStateFallback(t *testing.T) {
	c := types.Character{
		ID: "c2", Name: "沈青", RoleType: "antagonist", Status: "Alive",
		CurrentState:    "多疑渐深（第8章）",
		MainCareerID:    "魔道",
		MainCareerStage: 5,
	}
	e := graph.Entity{ID: "e2", Name: "沈青", Type: graph.EntityCharacter,
		Properties: map[string]string{"role_type": "antagonist"}} // 无状态机属性
	sc := sceneCharFromEntity(e, map[string]types.Character{c.Name: c})
	if sc.CurrentState != c.CurrentState || sc.CareerMain != "魔道·5阶" {
		t.Errorf("兜底失败: state=%q career=%q", sc.CurrentState, sc.CareerMain)
	}
}

// TestFormatSceneCharNoStateNoise 无状态机字段的老角色渲染零噪声
// （不出现空的「当前状态:」行）。
func TestFormatSceneCharNoStateNoise(t *testing.T) {
	sc := SceneChar{Name: "路人", RoleType: "minor", Status: "Alive"}
	line := formatSceneChar(sc)
	if strings.Contains(line, "当前状态") || strings.Contains(line, "主职业") {
		t.Errorf("无状态角色不应渲染状态行: %s", line)
	}
}
