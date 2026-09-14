package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

func newStatesTestProject(t *testing.T) *project.Manager {
	t.Helper()
	pm, err := project.Create(t.TempDir()+"/novel", "状态回灌测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	return pm
}

// TestBuildCharacterStatesSection_NoiseReduction 降噪矩阵（§7.4-3 照搬
// MuMu chapters.py:794/821）：intimacy==50 与 loyalty==50 不注入；
// active 组织成员才显示；past 关系跳过；死亡角色带 emoji。
func TestBuildCharacterStatesSection_NoiseReduction(t *testing.T) {
	pm := newStatesTestProject(t)
	a := &writingState{}
	err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive",
				CurrentState: "坚定（第12章）", MainCareerID: "剑修", MainCareerStage: 3},
			{ID: "c2", Name: "沈青", RoleType: "antagonist", Status: "Dead",
				StatusChangedChapter: 11, StateUpdatedChapter: 11},
			{ID: "c3", Name: "路人甲", RoleType: "minor", Status: "Alive"}, // 存量零 v2 字段
		},
		Organizations: []types.Organization{
			{Name: "青云宗", MemberList: []types.OrgMember{
				{CharacterID: "c1", Position: "内门弟子", Loyalty: 72, Status: "active"},
				{CharacterID: "c2", Position: "长老", Loyalty: 50, Status: "active"},
				{CharacterID: "c3", Position: "杂役", Loyalty: 90, Status: "active"},
				{CharacterID: "c1", Position: "客卿", Loyalty: 80, Status: "retired"}, // 非 active 不显示
			}},
		},
		Relationships: []types.Relationship{
			{FromID: "c1", ToID: "c2", RelationType: "rival", Description: "夺剑之仇", Intimacy: 20, Status: "past"},    // past 跳过
			{FromID: "c1", ToID: "c3", RelationType: "friend", Description: "同门之谊", Intimacy: 50, Status: "active"}, // 默认 50 降噪
			{FromID: "c3", ToID: "c2", RelationType: "enemy", Description: "旧怨", Intimacy: -30, Status: "active"},
		},
	})
	if err != nil {
		t.Fatalf("写角色: %v", err)
	}

	out := a.buildCharacterStatesSection(pm)
	if out == "" {
		t.Fatal("有 v2 状态数据时不应返回空串")
	}

	// 降噪断言
	if strings.Contains(out, "忠诚度:50") || strings.Contains(out, "[50]") {
		t.Errorf("默认值 50 应降噪不注入:\n%s", out)
	}
	if !strings.Contains(out, "忠诚度:72") {
		t.Errorf("非默认忠诚度应注入:\n%s", out)
	}
	// past 关系整条不出现（含其亲密度 20 与描述「夺剑之仇」；
	// 「与沈青」不能断言——c3→c2 的 active「旧怨」合法地渲染同样前缀）
	if strings.Contains(out, "夺剑之仇") || strings.Contains(out, "[-20]") || strings.Contains(out, "[20]") {
		t.Errorf("past 关系不应注入:\n%s", out)
	}
	// c3 只有组织+关系 v2 数据（无心理/职业）→ 应出现（组织成员算 v2 字段）
	if !strings.Contains(out, "【路人甲】") {
		t.Errorf("有组织归属的角色应渲染:\n%s", out)
	}
	// 死亡角色 emoji
	if !strings.Contains(out, "【沈青】(💀已死亡)") {
		t.Errorf("死亡角色应带 emoji 标记:\n%s", out)
	}
	// 存量零 v2 角色整行跳过；c3 行应带组织信息（职位+非默认忠诚度）
	if !strings.Contains(out, "青云宗（杂役）[忠诚度:90]") {
		t.Errorf("c3 组织行应渲染:\n%s", out)
	}
	if !strings.Contains(out, "主职业: 剑修·3阶") {
		t.Errorf("主职业标签应渲染:\n%s", out)
	}
}

// TestBuildCharacterStatesSection_LegacyEmpty 存量项目（无任何 v2 字段）返回
// 空串——BuildUserPrompt 不渲染空 context，零噪声。
func TestBuildCharacterStatesSection_LegacyEmpty(t *testing.T) {
	pm := newStatesTestProject(t)
	a := &writingState{}
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive", Personality: "清冷"},
		},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	if out := a.buildCharacterStatesSection(pm); out != "" {
		t.Errorf("存量项目应返回空串，got:\n%s", out)
	}
}

// TestBuildCharacterStatesSection_EmptyProject 读档失败/空文件容错返回空串。
func TestBuildCharacterStatesSection_EmptyProject(t *testing.T) {
	pm := newStatesTestProject(t)
	a := &writingState{}
	if out := a.buildCharacterStatesSection(pm); out != "" {
		t.Errorf("空项目应返回空串，got: %q", out)
	}
	if out := a.buildCharacterStatesSection(nil); out != "" {
		t.Errorf("nil manager 应返回空串，got: %q", out)
	}
}

// TestCreateChapterTemplateRendersCharacterStates 模板联通：真实 create-chapter
// 模板有 character_states P1 槽位，注入内容渲染在 P0 区段之后；空内容不渲染
// 标题（存量项目零噪声）。
func TestCreateChapterTemplateRendersCharacterStates(t *testing.T) {
	eng := prompt.NewEngine("../../prompts")
	tmpl := eng.Get("create-chapter")
	if tmpl == nil {
		t.Fatal("缺少 create-chapter 模板")
	}
	if def, ok := tmpl.Inputs["character_states"]; !ok {
		t.Fatal("create-chapter 模板缺 character_states 槽位")
	} else if def.Priority != "P1" {
		t.Errorf("character_states priority = %q, want P1", def.Priority)
	}

	rendered := tmpl.BuildUserPrompt(map[string]string{
		"characters":       "- 林晚：主角·清冷",
		"character_states": "【林晚】\n  当前状态: 坚定（第12章）",
	})
	iChars := strings.Index(rendered, "已有角色")
	iStates := strings.Index(rendered, "角色当前状态")
	if iChars < 0 || iStates < 0 {
		t.Fatalf("渲染缺区段:\n%s", rendered)
	}
	if iStates < iChars {
		t.Errorf("P1 character_states 应渲染在 P0 characters 之后:\n%s", rendered)
	}
	if !strings.Contains(rendered, "坚定（第12章）") {
		t.Errorf("状态内容未进入渲染:\n%s", rendered)
	}

	empty := tmpl.BuildUserPrompt(map[string]string{"characters": "- 林晚"})
	if strings.Contains(empty, "角色当前状态") {
		t.Errorf("空 context 不应渲染标题:\n%s", empty)
	}
}
