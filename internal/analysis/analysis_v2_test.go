package analysis

import (
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// TestNormalizeAnalysisV2 t4-C1 服务端权威归一：overall 重算、钳位、建议联动裁剪。
func TestNormalizeAnalysisV2(t *testing.T) {
	v2 := &types.AnalysisResultV2{}
	v2.Scores = types.AnalysisScores{
		Pacing: 12.0, Engagement: -3.0, Coherence: 7.0,
		Overall: 10.0, // 模型自报值必须被重算覆盖
	}
	v2.Suggestions = []string{"一", "二", "三", "四", "五"}

	got := normalizeAnalysisV2(v2)
	if got != v2 {
		t.Fatalf("应就地改写返回同指针")
	}
	// 钳位：12→10，-3→1
	if v2.Scores.Pacing != 10.0 || v2.Scores.Engagement != 1.0 {
		t.Fatalf("三维应钳位 [1,10]: %+v", v2.Scores)
	}
	// overall 权威重算：(10+1+7)/3 = 6.0（模型自报 10 被覆盖）
	if v2.Scores.Overall != 6.0 {
		t.Fatalf("overall 应重算=6.0，实际 %v", v2.Scores.Overall)
	}
	// 建议联动：overall 6.0 不满足 <6.0 → 落 [2,3] 档，5 条裁到 3 条（只裁上限不代拟）
	if len(v2.Suggestions) != 3 {
		t.Fatalf("建议应按联动上限裁到 3 条，实际 %d", len(v2.Suggestions))
	}
}

// TestNormalizeAnalysisV2_SuggestionBands 建议数与 overall 分档联动矩阵。
func TestNormalizeAnalysisV2_SuggestionBands(t *testing.T) {
	cases := []struct {
		p, e, c float64
		wantMax int
	}{
		{2.0, 2.0, 2.0, 5}, // overall 2.0 → [4,5]
		{5.0, 5.0, 5.0, 4}, // overall 5.0 → [3,4]
		{7.0, 7.0, 7.0, 3}, // overall 7.0 → [2,3]
		{9.0, 9.0, 9.0, 1}, // overall 9.0 → [0,1]
	}
	for _, tc := range cases {
		v2 := &types.AnalysisResultV2{Suggestions: []string{"a", "b", "c", "d", "e"}}
		v2.Scores = types.AnalysisScores{Pacing: tc.p, Engagement: tc.e, Coherence: tc.c}
		normalizeAnalysisV2(v2)
		if len(v2.Suggestions) > tc.wantMax {
			t.Fatalf("P/E/C=%v overall=%v 建议应 ≤%d，实际 %d",
				tc.p, v2.Scores.Overall, tc.wantMax, len(v2.Suggestions))
		}
	}
}

// TestDeriveLegacyAnalysis 旧 wire 映射：AnalyzeChapter 返回键零变化。
func TestDeriveLegacyAnalysis(t *testing.T) {
	v2 := &types.AnalysisResultV2{
		Hooks: []types.Hook{
			{Type: "悬念", Content: "铜匣在雷雨夜自己打开了", Strength: 8, Position: "开头"},
		},
		Conflict: types.Conflict{Description: "父女对峙升级", Level: 7},
		EmotionalArc: types.EmotionalArc{
			PrimaryEmotion: "悲怆", Curve: "压抑到爆发", Intensity: 8,
		},
		PlotPoints: []types.PlotPoint{
			{Content: "事件一"}, {Content: "事件二"}, {Content: "事件三"},
			{Content: "事件四"}, {Content: "事件五"}, {Content: "事件六·应被裁"},
		},
		Pacing: "fast",
		Scores: types.AnalysisScores{Pacing: 8, Engagement: 7, Coherence: 6},
		CharacterStates: []types.CharacterStateChangeV2{
			{Name: "林晚", OldState: "怀疑", NewState: "确认真相"},
		},
		Foreshadows: []types.ForeshadowHit{
			{Type: "planted", Content: "星门钥匙", ReferenceStableID: ""},
		},
	}
	normalizeAnalysisV2(v2) // overall = 7.0
	old := deriveLegacyAnalysis(v2)

	if old.Hook != "铜匣在雷雨夜自己打开了" {
		t.Fatalf("hook 应取首钩内容: %q", old.Hook)
	}
	if old.Conflict != "父女对峙升级" {
		t.Fatalf("conflict 应取描述: %q", old.Conflict)
	}
	if old.EmotionCurve != "悲怆：压抑到爆发" {
		t.Fatalf("情感曲线映射不对: %q", old.EmotionCurve)
	}
	if len(old.KeyEvents) != 5 || old.KeyEvents[4] != "事件五" {
		t.Fatalf("关键事件应取前 5 条: %v", old.KeyEvents)
	}
	if old.SceneRhythm != "节奏偏快" {
		t.Fatalf("节奏应映射中文: %q", old.SceneRhythm)
	}
	if old.QualityScore != 7 {
		t.Fatalf("质量分应=round(7.0)=7，实际 %d", old.QualityScore)
	}
	if len(old.CharacterStates) != 1 || old.CharacterStates[0].Name != "林晚" ||
		old.CharacterStates[0].NewState != "确认真相" {
		t.Fatalf("角色状态映射不对: %+v", old.CharacterStates)
	}
	if len(old.Foreshadows) != 1 || old.Foreshadows[0].Content != "星门钥匙" {
		t.Fatalf("伏笔应原样透传（t1-P2 消费契约不变）: %+v", old.Foreshadows)
	}
}

// TestDeriveLegacyAnalysis_EmptySafe 空载荷派生不 panic、字段归零。
func TestDeriveLegacyAnalysis_EmptySafe(t *testing.T) {
	old := deriveLegacyAnalysis(&types.AnalysisResultV2{})
	if old == nil {
		t.Fatal("应返回非 nil")
	}
	if old.Hook != "" || old.QualityScore != 0 || len(old.KeyEvents) != 0 {
		t.Fatalf("空载荷应字段归零: %+v", old)
	}
	if deriveLegacyAnalysis(nil) != nil {
		t.Fatal("nil 入应返回 nil")
	}
}

// TestPersistAnalysisV2_Upsert 落盘按章号 upsert：同章覆盖、异章追加、章号升序。
func TestPersistAnalysisV2_Upsert(t *testing.T) {
	a := newSyncTestAgent(t)

	mk := func(n int, summary string) types.ChapterAnalysisResult {
		v2 := types.AnalysisResultV2{Summary: summary}
		v2.Scores = types.AnalysisScores{Pacing: 8, Engagement: 8, Coherence: 8, Overall: 8}
		return types.ChapterAnalysisResult{
			ChapterNum: n, ChapterFile: "003.md", AnalyzerSource: "llm", Result: v2,
		}
	}
	if err := a.pm.UpsertAnalysisV2(mk(3, "第一版")); err != nil {
		t.Fatalf("首次落盘: %v", err)
	}
	if err := a.pm.UpsertAnalysisV2(mk(1, "第一章")); err != nil {
		t.Fatalf("异章追加: %v", err)
	}
	if err := a.pm.UpsertAnalysisV2(mk(3, "重分析覆盖版")); err != nil {
		t.Fatalf("同章覆盖: %v", err)
	}

	f, err := a.pm.ReadAnalysisV2File()
	if err != nil {
		t.Fatalf("读回: %v", err)
	}
	if len(f.Items) != 2 {
		t.Fatalf("同章应覆盖不追加，应 2 条实际 %d: %+v", len(f.Items), f.Items)
	}
	if f.Items[0].ChapterNum != 1 || f.Items[1].ChapterNum != 3 {
		t.Fatalf("应按章号升序: %+v", f.Items)
	}
	if f.Items[1].Result.Summary != "重分析覆盖版" {
		t.Fatalf("同章重分析应覆盖: %+v", f.Items[1])
	}

	// persistAnalysisV2 容错路径：正常项目上跑通（引擎/模型缺省为空串）
	a.persistAnalysisV2(2, &types.AnalysisResultV2{})
	f, err = a.pm.ReadAnalysisV2File()
	if err != nil || len(f.Items) != 3 {
		t.Fatalf("persist 后应 3 条: %v %d", err, len(f.Items))
	}
}

// TestSyncCharacterStatesV2_OrgDiff 组织顶层差分端到端（t5 第二刀）：
// V2 载荷 organization_states（名称式引用）→ 名称→ID 精确匹配 →
// ApplyChapterDiff 落库。覆盖：成员加入带忠诚度/晋升/覆灭短路/
// 不存在组织与角色跳过/destroyed 时其他字段忽略。
func TestSyncCharacterStatesV2_OrgDiff(t *testing.T) {
	a := newSyncTestAgent(t)
	if err := a.pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive"},
			{ID: "c2", Name: "沈青", RoleType: "antagonist", Status: "Alive"},
		},
		Organizations: []types.Organization{
			{Name: "青云宗", MemberList: []types.OrgMember{
				{CharacterID: "c1", Position: "外门弟子", Status: "active", Loyalty: 50},
			}},
			{Name: "丹盟"},
		},
	}); err != nil {
		t.Fatalf("写角色库: %v", err)
	}

	loyal := 88
	destroyed := true
	v2 := &types.AnalysisResultV2{
		OrganizationStates: []types.OrganizationStateChangeV2{
			{OrgName: "青云宗", PowerValue: intPtr(72), MemberChanges: []types.OrgMemberChangeV2{
				{CharacterName: "林晚", ChangeType: "promoted", Position: "内门弟子", LoyaltyHint: &loyal},
				{CharacterName: "沈青", ChangeType: "joined", Position: "客卿"},
				{CharacterName: "不存在的人", ChangeType: "joined"}, // 名称匹配失败跳过
			}},
			{OrgName: "不存在组织", Destroyed: &destroyed},                      // 组织不存在跳过
			{OrgName: "丹盟", Destroyed: &destroyed, PowerValue: intPtr(10)}, // 覆灭短路：power 忽略
		},
	}
	a.syncCharacterStatesV2(5, v2)

	cf, err := a.pm.ReadCharacters()
	if err != nil {
		t.Fatalf("读回: %v", err)
	}
	orgByName := map[string]types.Organization{}
	for _, o := range cf.Organizations {
		orgByName[o.Name] = o
	}
	qy := orgByName["青云宗"]
	if qy.PowerValue != 72 {
		t.Errorf("青云宗 PowerValue = %d, want 72", qy.PowerValue)
	}
	if len(qy.MemberList) != 2 {
		t.Fatalf("青云宗成员应 2 人（不存在的人被跳过）: %+v", qy.MemberList)
	}
	if qy.MemberList[0].Position != "内门弟子" || qy.MemberList[0].Loyalty != 88 || qy.MemberList[0].UpdatedChapter != 5 {
		t.Errorf("林晚晋升未落: %+v", qy.MemberList[0])
	}
	if qy.MemberList[1].CharacterID != "c2" || qy.MemberList[1].Loyalty != 50 || qy.MemberList[1].JoinedAt != "第5章" {
		t.Errorf("沈青加入未落（ID 匹配/缺省忠诚 50/JoinedAt）: %+v", qy.MemberList[1])
	}
	dm := orgByName["丹盟"]
	if !dm.Destroyed || dm.DestroyedChapter != 5 {
		t.Errorf("丹盟覆灭未落: %+v", dm)
	}
	if dm.PowerValue != 0 {
		t.Errorf("覆灭短路后 PowerValue 应被忽略（保持 0）: %d", dm.PowerValue)
	}
}

// TestSyncCharacterStatesV2_OrgOnlyEmpty 载荷只有空组织差分时零写盘
// （早退短路，不触发 ReadCharacters 失败路径）。
func TestSyncCharacterStatesV2_OrgOnlyEmpty(t *testing.T) {
	a := newSyncTestAgent(t)
	a.syncCharacterStatesV2(3, &types.AnalysisResultV2{}) // 全空：不应 panic 也不应写盘
	cf, err := a.pm.ReadCharacters()
	if err != nil || cf == nil {
		t.Fatalf("空项目读回应为空文件非错误: %v", err)
	}
	if len(cf.Organizations) != 0 || len(cf.Characters) != 0 {
		t.Errorf("空差分不应写入任何数据: %+v", cf)
	}
}

func intPtr(v int) *int { return &v }
