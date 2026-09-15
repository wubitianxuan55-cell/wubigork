package characterstate

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func mkFile() *types.CharacterFile {
	return &types.CharacterFile{
		Characters: []types.Character{
			{ID: "c_lin", Name: "林晚", Status: "Alive"},
			{ID: "c_chen", Name: "陈铮", Status: "Alive"},
			{ID: "c_ghost", Name: "路人甲", Status: "Alive"},
		},
		Organizations: []types.Organization{
			{ID: "o_qy", Name: "青云宗", Members: []string{"c_lin"}, MemberList: []types.OrgMember{
				{CharacterID: "c_lin", Position: "弟子", Status: "active", Loyalty: 50},
			}},
		},
		Relationships: []types.Relationship{},
	}
}

// TestApplyChapterDiff_SurvivalCascade t5 核心级联（§3.4）：存活终态 + 水位 +
// 关系 past + 组织成员终态 + 心理短路（退场后不再改心理/关系）。
func TestApplyChapterDiff_SurvivalCascade(t *testing.T) {
	cf := mkFile()
	cf.Relationships = append(cf.Relationships, types.Relationship{
		FromID: "c_lin", ToID: "c_chen", Status: "active", Intimacy: 60,
	})

	res := ApplyChapterDiff(cf, 5, []types.CharacterStateDiff{
		{Name: "林晚", Status: "Dead", KeyEvent: "坠崖"},
		{Name: "林晚", NewState: "坚定", Status: "Dead"}, // 死亡后心理应被短路
	}, nil, nil)

	lin := &cf.Characters[0]
	if lin.Status != "Dead" || lin.StatusChangedChapter != 5 || lin.StateUpdatedChapter != 5 {
		t.Fatalf("存活终态/水位不对: %+v", lin)
	}
	if !strings.Contains(lin.CurrentState, "死亡") || !strings.Contains(lin.CurrentState, "第5章") {
		t.Fatalf("心理应文本化为「死亡（第5章）」: %q", lin.CurrentState)
	}
	// 级联 1：active 关系 → past + EndedAt
	if cf.Relationships[0].Status != "past" || cf.Relationships[0].EndedAt != "第5章" {
		t.Fatalf("关系级联不对: %+v", cf.Relationships[0])
	}
	// 级联 2：组织成员 → deceased + LeftAt + Notes 追加
	m := &cf.Organizations[0].MemberList[0]
	if m.Status != "deceased" || m.LeftAt != "第5章" || !strings.Contains(m.Notes, "[第5章]") {
		t.Fatalf("组织成员级联不对: %+v", m)
	}
	if res.StateUpdated < 1 {
		t.Fatalf("计数不对: %+v", res)
	}
}

// TestApplyChapterDiff_WatermarkGuards 水位单调守卫：重放低章节被拒。
func TestApplyChapterDiff_WatermarkGuards(t *testing.T) {
	cf := mkFile()
	cf.Characters[0].StateUpdatedChapter = 8
	cf.Characters[0].StatusChangedChapter = 8

	res := ApplyChapterDiff(cf, 5, []types.CharacterStateDiff{
		{Name: "林晚", NewState: "动摇"},
		{Name: "林晚", Status: "Dead"},
	}, nil, nil)

	if cf.Characters[0].CurrentState != "" || cf.Characters[0].Status != "Alive" {
		t.Fatalf("低章节重放不得改写: %+v", cf.Characters[0])
	}
	if len(res.Skipped) != 2 {
		t.Fatalf("两条都应记 Skipped: %+v", res.Skipped)
	}
}

// TestApplyChapterDiff_Relationships 关系差分：新建基线 50+delta；已存在双向
// 命中、描述追加、亲密度钳制。
func TestApplyChapterDiff_Relationships(t *testing.T) {
	cf := mkFile()
	rels := []types.RelationshipChange{
		{FromName: "林晚", ToName: "陈铮", ChangeDesc: "并肩作战后彼此信任"},
		{FromName: "陈铮", ToName: "林晚", ChangeDesc: "不信任加深"}, // 反向应命中同一条
	}
	res := ApplyChapterDiff(cf, 3, nil, rels, nil)

	if res.RelCreated != 1 || res.RelUpdated != 1 {
		t.Fatalf("应新建 1 更新 1: %+v", res)
	}
	rel := cf.Relationships[0]
	if rel.Status != "active" || rel.StartedAt != "第3章" {
		t.Fatalf("新建关系形状不对: %+v", rel)
	}
	// 反向更新：描述追加时间线 + 亲密度 = clamp(基线 + 「并肩作战后彼此信任」命中… + 「不信任」-15)
	if !strings.Contains(rel.Description, "[第3章] 并肩作战后彼此信任") ||
		!strings.Contains(rel.Description, "[第3章] 不信任加深") {
		t.Fatalf("变更时间线缺失: %q", rel.Description)
	}
	if len(rel.History) != 2 {
		t.Fatalf("History 应 2 条: %+v", rel.History)
	}
	if rel.Intimacy > 100 || rel.Intimacy < -100 {
		t.Fatalf("亲密度应钳 [-100,100]: %d", rel.Intimacy)
	}
}

// TestCalcIntimacyDelta 最长匹配优先（§7.3）：「不信任」=-15 不得被「信任」+10
// 稀释；钳制 ±30；无匹配 0。
func TestCalcIntimacyDelta(t *testing.T) {
	cases := []struct {
		desc string
		want int
	}{
		{"不信任加深", -15},   // 修 MuMu 子串累加缺陷（原 -5）
		{"彼此信任", 10},     // 普通命中
		{"彻底背叛", -30},    // 负向最长
		{"和解并加深信任", 20},  // 「和解」20 为最长命中
		{"天气不错", 0},      // 无匹配
		{"极度仇恨与决裂", -30}, // 多词命中取最长（-25/-30 → -30 最长命中钳制）
	}
	for _, tc := range cases {
		if got := CalcIntimacyDelta(tc.desc); got != tc.want {
			t.Fatalf("CalcIntimacyDelta(%q)=%d want %d", tc.desc, got, tc.want)
		}
	}
}

// TestApplyChapterDiff_OrgAndCareer 组织成员变更与职业推进（守卫/上限）。
func TestApplyChapterDiff_OrgAndCareer(t *testing.T) {
	cf := mkFile()
	pv := 88
	destroyed := false
	res := ApplyChapterDiff(cf, 4, []types.CharacterStateDiff{
		{Name: "林晚", CareerIncs: []types.CareerStageChange{
			{CareerID: "sword", IsMain: true, NewStage: 3},
			{CareerID: "alchemy", IsMain: false, NewStage: 2},
		}},
	}, nil, []types.OrgStateDiff{
		{OrgName: "青云宗", PowerValue: &pv, Destroyed: &destroyed, Members: []types.OrgMemberChange{
			{CharacterID: "c_chen", ChangeType: types.OrgMemberJoined, Position: "外门弟子", LoyaltyHint: intPtr(70)},
		}},
	})

	lin := &cf.Characters[0]
	if lin.MainCareerID != "sword" || lin.MainCareerStage != 3 {
		t.Fatalf("主职业推进不对: %+v", lin)
	}
	if len(lin.SubCareers) != 1 || lin.SubCareers[0].Stage != 2 || lin.SubCareers[0].UpdatedChapter != 4 {
		t.Fatalf("副职业推进/水位不对: %+v", lin.SubCareers)
	}
	org := &cf.Organizations[0]
	if org.PowerValue != 88 || org.Destroyed {
		t.Fatalf("组织状态不对: %+v", org)
	}
	if len(org.MemberList) != 2 || org.MemberList[1].CharacterID != "c_chen" || org.MemberList[1].Loyalty != 70 {
		t.Fatalf("成员加入不对: %+v", org.MemberList)
	}
	if res.CareerUpdated != 2 || res.OrgMemberUpdated != 1 || res.OrgStateUpdated != 1 {
		t.Fatalf("计数不对: %+v", res)
	}

	// 副职业上限 2：第三条副职业被拒
	cf.Characters[0].SubCareers = append(cf.Characters[0].SubCareers, types.CharacterCareerRef{CareerID: "talismans", Stage: 1, UpdatedChapter: 4})
	res2 := ApplyChapterDiff(cf, 5, []types.CharacterStateDiff{{
		Name: "林晚", CareerIncs: []types.CareerStageChange{{CareerID: "medicine", IsMain: false, NewStage: 1}},
	}}, nil, nil)
	if len(cf.Characters[0].SubCareers) != 2 {
		t.Fatalf("副职业应上限 2: %+v", cf.Characters[0].SubCareers)
	}
	if len(res2.Skipped) == 0 {
		t.Fatalf("超限应记 Skipped: %+v", res2.Skipped)
	}
}

func intPtr(v int) *int { return &v }

// TestApplyChapterDiff_OrgDestroyedShortCircuit 覆灭短路（MuMu :718-750，
// t5 第二刀钉死）：destroyed=true 时 power 清零、同条成员变更跳过——
// 组织已灭，加人/晋升无意义；低章节重放不「复活」已覆灭组织。
func TestApplyChapterDiff_OrgDestroyedShortCircuit(t *testing.T) {
	cf := &types.CharacterFile{
		Characters: []types.Character{{ID: "c1", Name: "甲", Status: "Alive"}},
		Organizations: []types.Organization{
			{Name: "魔教", PowerValue: 60, MemberList: []types.OrgMember{
				{CharacterID: "c1", Status: "active", Loyalty: 80},
			}},
		},
	}
	yes := true
	res := ApplyChapterDiff(cf, 9, nil, nil, []types.OrgStateDiff{{
		OrgName: "魔教", Destroyed: &yes, PowerValue: intPtr2(99),
		Members: []types.OrgMemberChange{{CharacterID: "c1", ChangeType: "promoted", Position: "护法"}},
	}})
	o := cf.Organizations[0]
	if !o.Destroyed || o.DestroyedChapter != 9 {
		t.Errorf("覆灭未落: %+v", o)
	}
	if o.PowerValue != 0 {
		t.Errorf("覆灭应清零势力值（MuMu :725），got %d", o.PowerValue)
	}
	if len(o.MemberList) != 1 || o.MemberList[0].Position == "护法" {
		t.Errorf("覆灭短路应跳过成员变更: %+v", o.MemberList)
	}
	if res.OrgStateUpdated != 1 || res.OrgMemberUpdated != 0 {
		t.Errorf("净产出不符: %+v", res)
	}

	// 低章节重放：destroyed=false 不清覆灭标记（false 与 null 都不走短路分支）
	no := false
	ApplyChapterDiff(cf, 3, nil, nil, []types.OrgStateDiff{{OrgName: "魔教", Destroyed: &no, PowerValue: intPtr2(10)}})
	if !cf.Organizations[0].Destroyed {
		t.Errorf("destroyed=false 不应复活已覆灭组织")
	}
	if cf.Organizations[0].PowerValue != 10 {
		t.Errorf("非覆灭分支 power 正常更新: %d", cf.Organizations[0].PowerValue)
	}
}

func intPtr2(v int) *int { return &v }
