package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 向后兼容：旧 foreshadows.json 原样可解析 ──────────────────
//
// 权威要求（acceptance 第 2 条 / handoff §2-3）：旧的 v1 JSON 必须可继续读取，
// 写入口径仍是 revealed；resolved 只作读取别名归一化为 revealed，写回不丢字段。

const legacyForeshadowsJSON = `{
  "items": [
    {
      "id": "planted_001_a3f9c2b4",
      "category": "plot",
      "description": "主角初见时注意到对方一缕异常绿发",
      "planted_in": "001.md",
      "revealed_in": "015.md",
      "status": "revealed",
      "is_long_term": true
    },
    {
      "id": "hinted_002_bb11cc22",
      "category": "character",
      "description": "旧账本里少了一页",
      "planted_in": "002.md",
      "status": "hinted",
      "is_long_term": false
    }
  ]
}`

func TestForeshadowFile_LegacyJSONParses(t *testing.T) {
	var ff ForeshadowFile
	if err := json.Unmarshal([]byte(legacyForeshadowsJSON), &ff); err != nil {
		t.Fatalf("旧 v1 foreshadows.json 解析失败（零迁移被破坏）: %v", err)
	}
	if len(ff.Items) != 2 {
		t.Fatalf("条目数 = %d, want 2", len(ff.Items))
	}
	// 旧文件无 schema_version → 0，读取层应视为 v1。
	if ff.SchemaVersion != 0 {
		t.Errorf("旧文件 schema_version = %d, want 0（缺省视为 v1）", ff.SchemaVersion)
	}
	got := ff.Items[0]
	if got.ID != "planted_001_a3f9c2b4" || got.Category != "plot" || got.PlantedIn != "001.md" {
		t.Errorf("既有字段解析错位: %+v", got)
	}
	if got.Status != ForeshadowRevealed {
		t.Errorf("status = %q, want %q（兼容别名必须原样读出）", got.Status, ForeshadowRevealed)
	}
	if !got.IsLongTerm {
		t.Error("is_long_term 丢失")
	}
	// 归一化后仍应是写入口径 revealed（存量数据零改写）。
	if n := NormalizeForeshadowStatus(got.Status); n != ForeshadowRevealed {
		t.Errorf("归一化 = %q, want %q", n, ForeshadowRevealed)
	}
}

func TestForeshadowFile_LegacyJSONRoundTripsAllFields(t *testing.T) {
	var ff ForeshadowFile
	if err := json.Unmarshal([]byte(legacyForeshadowsJSON), &ff); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(&ff)
	if err != nil {
		t.Fatal(err)
	}
	var back ForeshadowFile
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Items) != len(ff.Items) {
		t.Fatalf("往返条目数 = %d, want %d", len(back.Items), len(ff.Items))
	}
	for i := range ff.Items {
		if back.Items[i].ID != ff.Items[i].ID ||
			back.Items[i].Description != ff.Items[i].Description ||
			back.Items[i].Status != ff.Items[i].Status ||
			back.Items[i].IsLongTerm != ff.Items[i].IsLongTerm ||
			back.Items[i].PlantedIn != ff.Items[i].PlantedIn ||
			back.Items[i].RevealedIn != ff.Items[i].RevealedIn {
			t.Errorf("条目 %d 往返丢字段:\n got %+v\nwant %+v", i, back.Items[i], ff.Items[i])
		}
	}
	// 新增可选字段在旧数据上不得被写成非零值（omitempty 生效）。
	if strings.Contains(string(out), "target_resolve_in") {
		t.Errorf("旧数据写回不应凭空产出 target_resolve_in: %s", out)
	}
}

func TestForeshadow_V2FieldsRoundTrip(t *testing.T) {
	yes := true
	f := Foreshadow{
		ID: "planted_015_9c1d", Category: ForeshadowMystery,
		Title: "绿头发的视觉符号", Description: "血脉伏笔",
		HintText: "鬓角有一缕绿", ResolutionText: "血脉揭晓",
		SourceType: ForeshadowSourceAnalysis, SourceMemoryID: "planted_015_9c1d",
		PlantedIn: "015.md", TargetResolveIn: "030.md", RevealedIn: "",
		Status:     ForeshadowPlanted,
		Importance: 0.8, Strength: 8, Subtlety: 7,
		IsLongTerm:         true,
		RelatedCharacters:  []string{"林雪"},
		RelatedForeshadows: []string{"planted_001_a3f9c2b4"},
		Tags:               []string{"身世", "血脉"},
		Notes:              "创作用", ResolutionNotes: "血脉揭晓时回收",
		AutoRemind: &yes, RemindBeforeChapters: 5, IncludeInContext: &yes,
		CreatedAt: "2026-01-19T10:05:40Z", UpdatedAt: "2026-01-19T10:05:40Z",
		PlantedAt: "2026-01-19T10:05:40Z",
	}
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var back Foreshadow
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.TargetResolveIn != "030.md" || back.Importance != 0.8 || back.Strength != 8 || back.Subtlety != 7 {
		t.Errorf("v2 字段往返失真: %+v", back)
	}
	if back.AutoRemind == nil || !*back.AutoRemind {
		t.Error("auto_remind 指针语义丢失")
	}
	if len(back.RelatedForeshadows) != 1 || back.RelatedForeshadows[0] != "planted_001_a3f9c2b4" {
		t.Errorf("伏笔链丢失: %+v", back.RelatedForeshadows)
	}
}

func TestForeshadowStatus_UnionAndAlias(t *testing.T) {
	if got := len(AllForeshadowStatuses()); got != 6 {
		t.Fatalf("并集状态数 = %d, want 6", got)
	}
	for _, s := range AllForeshadowStatuses() {
		if !IsForeshadowStatusValid(s) {
			t.Errorf("%q 应属合法写入状态", s)
		}
		if IsForeshadowStatusAlias(s) {
			t.Errorf("%q 不应被判为别名", s)
		}
	}
	if !IsForeshadowStatusAlias(ForeshadowResolved) {
		t.Error("resolved 必须被判为兼容别名")
	}
	if IsForeshadowStatusValid(ForeshadowResolved) {
		t.Error("resolved 不得作为合法写入状态")
	}
	// 常量字面量必须保持与 v1 一致，否则既有比较语义静默漂移。
	if string(ForeshadowRevealed) != "revealed" {
		t.Fatalf("ForeshadowRevealed = %q, want \"revealed\"（v1 兼容）", ForeshadowRevealed)
	}
	if string(ForeshadowResolved) != "resolved" {
		t.Fatalf("ForeshadowResolved = %q, want \"resolved\"", ForeshadowResolved)
	}
	if string(ForeshadowPartial) != "partially_resolved" {
		t.Fatalf("ForeshadowPartial = %q", ForeshadowPartial)
	}
}

func TestNormalizeForeshadowStatus(t *testing.T) {
	cases := []struct {
		in, want ForeshadowStatus
	}{
		{ForeshadowResolved, ForeshadowRevealed}, // 读取别名 → 写入口径
		{ForeshadowRevealed, ForeshadowRevealed}, // 写入口径幂等
		{ForeshadowPlanted, ForeshadowPlanted},
		{ForeshadowPending, ForeshadowPending},
		{"weird_unknown", "weird_unknown"}, // 未知值原样返回，不猜
	}
	for _, c := range cases {
		if got := NormalizeForeshadowStatus(c.in); got != c.want {
			t.Errorf("NormalizeForeshadowStatus(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsResolvedStatus_CoversBothWireValues(t *testing.T) {
	// 「是否已回收」必须两套 wire 值都判得对：gaea 存量 revealed + 外部导入 resolved。
	for _, s := range []ForeshadowStatus{ForeshadowRevealed, ForeshadowResolved} {
		if !IsResolvedStatus(s) {
			t.Errorf("IsResolvedStatus(%q) = false, want true", s)
		}
	}
	for _, s := range []ForeshadowStatus{ForeshadowPending, ForeshadowPlanted, ForeshadowHinted,
		ForeshadowPartial, ForeshadowAbandoned, ""} {
		if IsResolvedStatus(s) {
			t.Errorf("IsResolvedStatus(%q) = true, want false", s)
		}
	}
}

// ── 角色：向后兼容 ────────────────────────────────────────────

const legacyCharactersJSON = `{
  "characters": [
    {
      "id": "c1",
      "name": "林雪",
      "role_type": "protagonist",
      "gender": "女",
      "status": "Alive",
      "notes": "剑修",
      "portrait_url": "data:image/png;base64,AAAA"
    }
  ],
  "organizations": [
    {"id": "o1", "name": "青云宗", "type": "门派", "power_level": "一流", "members": ["c1"]}
  ],
  "relationships": [
    {"from_id": "c1", "to_id": "c2", "relation_type": "friend", "intimacy": 30}
  ]
}`

func TestCharacterFile_LegacyJSONParses(t *testing.T) {
	var cf CharacterFile
	if err := json.Unmarshal([]byte(legacyCharactersJSON), &cf); err != nil {
		t.Fatalf("旧 characters.json 解析失败: %v", err)
	}
	c := cf.Characters[0]
	if c.Name != "林雪" || c.Status != "Alive" || c.PortraitURL == "" {
		t.Fatalf("既有字段解析错位: %+v", c)
	}
	// 新增字段零值（缺省）。
	if c.StatusChangedChapter != 0 || c.CurrentState != "" || c.StateUpdatedChapter != 0 ||
		c.MainCareerID != "" || c.MainCareerStage != 0 || len(c.SubCareers) != 0 {
		t.Errorf("旧数据新增字段应为零值: %+v", c)
	}
	org := cf.Organizations[0]
	if org.PowerLevel != "一流" || len(org.Members) != 1 {
		t.Errorf("组织既有字段丢失: %+v", org)
	}
	if org.MemberList != nil || org.PowerValue != 0 {
		t.Errorf("组织新增字段应为零值: %+v", org)
	}
	rel := cf.Relationships[0]
	if rel.Intimacy != 30 || rel.RelationType != "friend" {
		t.Errorf("关系既有字段丢失: %+v", rel)
	}
	if rel.Status != "" || len(rel.History) != 0 {
		t.Errorf("关系新增字段应为零值: %+v", rel)
	}
}

func TestCharacterFile_NewFieldsRoundTripAndOmitOld(t *testing.T) {
	cf := CharacterFile{
		Characters: []Character{{
			ID: "c1", Name: "林雪", RoleType: "protagonist", Status: "Alive",
			StatusChangedChapter: 12, CurrentState: "心怀戒备", StateUpdatedChapter: 12,
			MainCareerID: "career_sword", MainCareerStage: 3,
			SubCareers: []CharacterCareerRef{{CareerID: "career_alchemy", Stage: 1, UpdatedChapter: 9}},
		}},
		Organizations: []Organization{{
			ID: "o1", Name: "青云宗", PowerLevel: "一流",
			PowerValue: 72, MemberList: []OrgMember{{CharacterID: "c1", Position: "外门弟子", Status: "active", Loyalty: 60}},
		}},
		Relationships: []Relationship{{
			FromID: "c1", ToID: "c2", RelationType: "friend", Intimacy: 40,
			Status: "active", History: []RelChange{{Chapter: 12, Description: "并肩作战"}},
		}},
	}
	raw, err := json.Marshal(cf)
	if err != nil {
		t.Fatal(err)
	}
	var back CharacterFile
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Characters[0].StatusChangedChapter != 12 || back.Characters[0].MainCareerStage != 3 {
		t.Errorf("角色水位字段往返失真: %+v", back.Characters[0])
	}
	if len(back.Characters[0].SubCareers) != 1 || back.Characters[0].SubCareers[0].UpdatedChapter != 9 {
		t.Errorf("副职业引用往返失真: %+v", back.Characters[0].SubCareers)
	}
	if back.Organizations[0].PowerValue != 72 || len(back.Organizations[0].MemberList) != 1 {
		t.Errorf("组织 v2 字段往返失真: %+v", back.Organizations[0])
	}
	if back.Relationships[0].Status != "active" || len(back.Relationships[0].History) != 1 {
		t.Errorf("关系 v2 字段往返失真: %+v", back.Relationships[0])
	}
}

// ── worldview.json 零迁移（方案 C 只是新增 section）──────────

func TestWorldviewFile_LegacyJSONParsesAndCareersSectionIsOptional(t *testing.T) {
	legacy := `{"sections":[{"id":"era","title":"时代背景","content":"近未来","order":1}]}`
	var wf WorldviewFile
	if err := json.Unmarshal([]byte(legacy), &wf); err != nil {
		t.Fatalf("旧 worldview.json 解析失败: %v", err)
	}
	for _, s := range wf.Sections {
		if s.ID == CareersSectionID {
			t.Error("旧数据不应凭空出现 careers section")
		}
	}
	// 方案 C：职业tree 只是新增一个 section，无需新模型、无需迁移。
	wf.Sections = append(wf.Sections, WorldviewSection{
		ID: CareersSectionID, Title: "职业体系", Content: "3 主 2 副", Order: 9,
	})
	raw, err := json.Marshal(&wf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"id":"careers"`) {
		t.Errorf("careers section 未落盘: %s", raw)
	}
}

func TestActiveMemberCount_Derived(t *testing.T) {
	list := []OrgMember{
		{CharacterID: "a", Status: "active"},
		{CharacterID: "b", Status: ""},         // 缺省视为 active
		{CharacterID: "c", Status: "retired"},  // 退休不计
		{CharacterID: "d", Status: "expelled"}, // 逐出不计
		{CharacterID: "e", Status: "deceased"}, // 身故不计
	}
	if got := ActiveMemberCount(list); got != 2 {
		t.Fatalf("ActiveMemberCount = %d, want 2（派生值，不得累加）", got)
	}
	if got := ActiveMemberCount(nil); got != 0 {
		t.Fatalf("空列表 = %d, want 0", got)
	}
	// 关键不变量：加入再离开不得虚高（MuMu 只增不减）。
	list = append(list, OrgMember{CharacterID: "f", Status: "active"})
	if got := ActiveMemberCount(list); got != 3 {
		t.Fatalf("加入后 = %d, want 3", got)
	}
	list[len(list)-1].Status = "left"
	if got := ActiveMemberCount(list); got != 2 {
		t.Fatalf("离开后 = %d, want 2（不得虚高）", got)
	}
}

func TestCareerStageAdvanceAllowed_Watermark(t *testing.T) {
	ref := CharacterCareerRef{CareerID: "c", Stage: 3, UpdatedChapter: 20}
	// 重跑低章节必须拒绝（MuMu 会重复升阶）。
	if CareerStageAdvanceAllowed(ref, 15) {
		t.Error("低章节重放必须被水位守卫拒绝")
	}
	// 同章幂等允许。
	if !CareerStageAdvanceAllowed(ref, 20) {
		t.Error("同章应允许幂等覆盖")
	}
	// 前进允许。
	if !CareerStageAdvanceAllowed(ref, 21) {
		t.Error("更高章节应允许推进")
	}
	// 无水位记录（0）视为最低水位。
	fresh := CharacterCareerRef{CareerID: "c"}
	if !CareerStageAdvanceAllowed(fresh, 1) {
		t.Error("无水位记录时应允许首次推进")
	}
	// 非法章号一律拒绝。
	if CareerStageAdvanceAllowed(fresh, 0) || CareerStageAdvanceAllowed(fresh, -3) {
		t.Error("章号 <= 0 必须拒绝")
	}
}

func TestCharacterStateWatermarks(t *testing.T) {
	c := Character{Status: "Alive", StatusChangedChapter: 10, CurrentState: "平静", StateUpdatedChapter: 12}
	if SurvivalStatusAdvanceAllowed(c, 9) {
		t.Error("存活状态：低章节重放应拒绝")
	}
	if !SurvivalStatusAdvanceAllowed(c, 10) {
		t.Error("存活状态：同章应允许")
	}
	if StateAdvanceAllowed(c, 11) {
		t.Error("心理状态：低章节重放应拒绝")
	}
	if !StateAdvanceAllowed(c, 13) {
		t.Error("心理状态：更高章节应允许")
	}
}

func TestClampHelpers(t *testing.T) {
	if ClampIntimacy(150) != 100 || ClampIntimacy(-150) != -100 || ClampIntimacy(7) != 7 {
		t.Error("ClampIntimacy 边界错")
	}
	if ClampDeltaHint(99) != 30 || ClampDeltaHint(-99) != -30 || ClampDeltaHint(0) != 0 {
		t.Error("ClampDeltaHint 边界错（±30）")
	}
}

// ── 评分契约 ─────────────────────────────────────────────────

func TestComputeOverallScore(t *testing.T) {
	cases := []struct {
		p, e, c, want float64
	}{
		{6, 6, 6, 6},
		{5, 6, 7, 6},
		{7.5, 8, 8.5, 8},
		{1, 1, 1, 1},
		{10, 9, 8, 9},
		// 6.666… 必须收敛成一位小数。
		{6, 7, 7, 6.7},
	}
	for _, c := range cases {
		if got := ComputeOverallScore(c.p, c.e, c.c); got != c.want {
			t.Errorf("ComputeOverallScore(%v,%v,%v) = %v, want %v", c.p, c.e, c.c, got, c.want)
		}
	}
}

func TestSuggestionCountForOverall_HardLinkage(t *testing.T) {
	cases := []struct {
		overall  float64
		min, max int
	}{
		{1.0, 4, 5},
		{3.9, 4, 5},
		{4.0, 3, 4},
		{5.9, 3, 4},
		{6.0, 2, 3},
		{7.9, 2, 3},
		{8.0, 0, 1},
		{10.0, 0, 1},
	}
	for _, c := range cases {
		min, max := SuggestionCountForOverall(c.overall)
		if min != c.min || max != c.max {
			t.Errorf("SuggestionCountForOverall(%v) = (%d,%d), want (%d,%d)",
				c.overall, min, max, c.min, c.max)
		}
	}
}

func TestValidKeywordRuneCount(t *testing.T) {
	if ValidKeywordRuneCount(7) || ValidKeywordRuneCount(26) {
		t.Error("区间外必须为非法")
	}
	if !ValidKeywordRuneCount(8) || !ValidKeywordRuneCount(25) || !ValidKeywordRuneCount(15) {
		t.Error("区间内必须为合法（含端点）")
	}
}

func TestAnalysisResultV2_JSONRoundTrip(t *testing.T) {
	r := AnalysisResultV2{
		Hooks: []Hook{{
			Type: "悬念", Content: "开头抛出异常", Strength: 8, Position: "开头",
			Keyword: Keyword{Text: "她鬓角有一缕颜色深得不自然的绿", Pos: 120, Length: 16},
		}},
		Foreshadows: []ForeshadowHit{{
			Title: "绿发", Content: "血脉暗示", Type: "planted", Strength: 8, Subtlety: 7,
			Category: "identity", IsLongTerm: true, RelatedChars: []string{"林雪"},
			EstimateResolve: 30, ReferenceChapter: 15, ReferenceStableID: "planted_015_9c1d",
			Keyword: Keyword{Text: "一缕颜色深得不自然的绿", Pos: 3, Length: 12},
		}},
		Conflict:        Conflict{Types: []string{"人与己"}, Level: 7, Description: "自我怀疑", ResolutionProgress: 0.3},
		EmotionalArc:    EmotionalArc{PrimaryEmotion: "压抑", Intensity: 6, Curve: "先抑后扬", SecondaryEmotions: []string{"不甘"}},
		CharacterStates: []CharacterStateChangeV2{{Name: "林雪", OldState: "平静", NewState: "戒备", SurvivalStatus: "active"}},
		PlotPoints:      []PlotPoint{{Content: "识破血脉", Type: "revelation", Importance: 0.9, Impact: "主线推进"}},
		Scenes:          []AnalysisScene{{Location: "破庙", Atmosphere: "阴冷", Duration: "半小时"}},
		Pacing:          "moderate",
		DialogueRatio:   0.4, DescriptionRatio: 0.6,
		Scores:      AnalysisScores{6, 7, 7, 6.7, "节奏稳、悬念足"},
		PlotStage:   "rise",
		Suggestions: []string{"补一处感官细节"},
		Summary:     "本章推进主线",
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back AnalysisResultV2
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Scores.Overall != 6.7 || back.Hooks[0].Keyword.Pos != 120 {
		t.Errorf("评分/keyword 往返失真: %+v", back.Scores)
	}
	if back.Foreshadows[0].ReferenceStableID != "planted_015_9c1d" {
		t.Errorf("reference_stable_id 往返失真: %+v", back.Foreshadows[0])
	}
	if len(back.CharacterStates) != 1 || back.CharacterStates[0].NewState != "戒备" {
		t.Errorf("角色状态往返失真: %+v", back.CharacterStates)
	}
}

func TestChapterAnalysisResult_RoundTrip(t *testing.T) {
	r := ChapterAnalysisResult{
		ChapterNum: 15, ChapterFile: "015.md",
		AnalyzedAt: "2026-01-19T10:05:40Z", Engine: "grok", Model: "grok-4",
		AnalyzerSource: "llm",
		Result:         AnalysisResultV2{Scores: AnalysisScores{Overall: 6.7}},
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back ChapterAnalysisResult
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.ChapterNum != 15 || back.Result.Scores.Overall != 6.7 {
		t.Errorf("往返失真: %+v", back)
	}
}

// ── ChapterPlan / RewriteVersion ─────────────────────────────

func TestChapterPlan_RoundTrip(t *testing.T) {
	p := ChapterPlan{
		SubIndex: 3, Title: "破庙粥",
		PlotSummary: "林雪在破庙分粥", KeyEvents: []string{"分粥", "识破绿发"},
		CharacterFocus: []string{"林雪"}, EmotionalTone: "压抑",
		NarrativeGoal: "引出绿发伏笔", ConflictType: "人与己",
		EndingType: string(EndingSuspense), EstimatedWords: 3000,
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var back ChapterPlan
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.SubIndex != 3 || len(back.KeyEvents) != 2 || back.EndingType != "悬念" {
		t.Errorf("ChapterPlan 往返失真: %+v", back)
	}
	if back.ConflictType == "" || back.NarrativeGoal == "" {
		t.Errorf("ChapterPlan 契约字段缺失: %+v", back)
	}
}

func TestRewriteVersion_RoundTripCarriesSnapshot(t *testing.T) {
	v := RewriteVersion{
		ID: "v1", ChapterNum: 15, Mode: RewriteModePartial, Status: RewriteApplied,
		Source: RewriteSourceMixed, SuggestionIdxs: []int{0, 2},
		CustomInstr: "少用形容词", FocusAreas: []string{"对话"},
		Preserve: &PreserveConfig{Structure: true, Dialogues: []string{"分粥"}},
		StartPos: 100, EndPos: 400, LengthMode: LengthSimilar, TargetWords: 3000,
		OriginalContent: "全文快照", OriginalWordCount: 3000,
		NewContent: "改写后", NewWordCount: 3050, Similarity: 82.5,
		BeforeAIScore: 40, AfterAIScore: 71,
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	// 空时间零值不应炸；restore 所需的核心字段必须齐全。
	var back RewriteVersion
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.OriginalContent != "全文快照" {
		t.Fatalf("原稿快照丢失 → 无法恢复: %+v", back)
	}
	if back.Mode != RewriteModePartial || back.StartPos != 100 || back.EndPos != 400 {
		t.Errorf("partial 区间丢失: %+v", back)
	}
	if back.Preserve == nil || !back.Preserve.Structure {
		t.Errorf("保留元素丢失: %+v", back.Preserve)
	}
	if back.Status != RewriteApplied || back.BeforeAIScore != 40 || back.AfterAIScore != 71 {
		t.Errorf("状态/评分丢失: %+v", back)
	}
}

func TestRewriteVersionIndex_OmitsContent(t *testing.T) {
	idx := RewriteVersionIndex{ID: "v1", ChapterNum: 15, Mode: RewriteModeWhole, Status: RewriteCompleted, Similarity: 60}
	raw, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "original_content") || strings.Contains(string(raw), "new_content") {
		t.Errorf("列表索引不得携带全文（MuMu 式 1.5MB 下发）: %s", raw)
	}
}

func TestAnnotationRoundTrip(t *testing.T) {
	af := AnnotationFile{ChapterNum: 15, Items: []Annotation{
		{Type: "hook", Title: "开头钩子", Content: "绿发", Importance: 0.8, Pos: 12, Length: 2, Tags: []string{"悬念"}},
		{Type: "suggestion", Content: "补细节", Pos: -1},
	}}
	raw, err := json.Marshal(af)
	if err != nil {
		t.Fatal(err)
	}
	var back AnnotationFile
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Items[0].Pos != 12 || back.Items[1].Pos != -1 {
		t.Errorf("标注偏移往返失真: %+v", back.Items)
	}
}

// ── 接口契约：模板 / 上下文 / Lint ───────────────────────────

func TestPromptTemplate_LegacyJSONHasNoNewFields(t *testing.T) {
	legacy := `{
      "name": "create-chapter",
      "system": "sys",
      "task": "写{word_count}字",
      "input_sections": {"plot_req": {"priority": "P0", "label": "本章剧情要求"}},
      "output": {"format": "markdown", "description": "desc"},
      "constraints": {"must": ["a"], "forbidden": ["b"], "style": ["c"]}
    }`
	var tmpl PromptTemplate
	if err := json.Unmarshal([]byte(legacy), &tmpl); err != nil {
		t.Fatalf("旧模板 JSON 必须原样可读（15 个模板零迁移）: %v", err)
	}
	if tmpl.Name != "create-chapter" || tmpl.Inputs["plot_req"].Priority != "P0" {
		t.Errorf("模板既有字段解析错位: %+v", tmpl)
	}
	if tmpl.Inputs["plot_req"].Order != 0 {
		t.Errorf("旧模板 order 缺省应为 0，实际 %d", tmpl.Inputs["plot_req"].Order)
	}
	if tmpl.Version != "" || tmpl.Category != "" || tmpl.Description != "" {
		t.Errorf("新增元数据在旧模板上应为零值: %+v", tmpl)
	}
}

// stubResolver 验证接口可被包外实现（依赖倒置的最小证据）。
type stubResolver struct {
	tmpl PromptTemplate
	src  TemplateSource
}

func (s stubResolver) Resolve(name string) (ResolvedTemplate, bool) {
	if name != s.tmpl.Name {
		return ResolvedTemplate{}, false
	}
	return ResolvedTemplate{Template: s.tmpl, Source: s.src}, true
}

func (s stubResolver) List() []string { return []string{s.tmpl.Name} }

func (s stubResolver) ResolveTemplate(name string) (PromptTemplate, TemplateSource, error) {
	return s.tmpl, s.src, nil
}

func (s stubResolver) ValidateTemplate(PromptTemplate) []ValidationIssue { return nil }

func (s stubResolver) ExtractPlaceholders(text string) []string { return nil }

func (s stubResolver) RenderTemplate(text string, _ map[string]string) TemplateRenderResult {
	return TemplateRenderResult{Text: text}
}

func (s stubResolver) LintForeshadowItems(_ []Foreshadow, opts ForeshadowLintOptions) ForeshadowLintReport {
	return ForeshadowLintReport{TotalChapters: opts.TotalChapters, Findings: []ForeshadowLintFinding{}}
}

// 编译期断言：一个结构体可同时满足三个接口（模板引擎 + Lint 的组合能力）。
var (
	_ TemplateParser    = stubResolver{}
	_ TemplateResolver  = stubResolver{}
	_ TemplateValidator = stubResolver{}
	_ TemplateRenderer  = stubResolver{}
	_ TemplateEngine    = stubResolver{}
	_ ForeshadowLinter  = stubResolver{}
)

// stubProvider 验证上下文提供方可被包外实现。
type stubProvider struct {
	name string
	sec  []ContextSection
}

func (p stubProvider) Name() string                 { return p.name }
func (p stubProvider) Provide(int) []ContextSection { return p.sec }

var _ ContextProvider = stubProvider{}

// stubAssembler 验证上下文组装接口可被包外实现（novelcontext.SceneBible 即天然实现）。
type stubAssembler struct{ text string }

func (a stubAssembler) Render(maxRunes int) string { return a.text }

var _ ContextAssembler = stubAssembler{}

func TestTemplateSourceRank(t *testing.T) {
	order := []TemplateSource{
		TemplateSourceProjectOverride,
		TemplateSourceGlobalOverride,
		TemplateSourceDisk,
		TemplateSourceEmbedded,
		TemplateSourceBuiltin,
	}
	for i := 1; i < len(order); i++ {
		if TemplateSourceRank(order[i-1]) >= TemplateSourceRank(order[i]) {
			t.Errorf("来源优先序错: %v 应优先于 %v", order[i-1], order[i])
		}
	}
}

func TestSortContextSections_Deterministic(t *testing.T) {
	in := []ContextSection{
		{ID: "z", Priority: SectionP0, Order: 2},
		{ID: "a", Priority: SectionP0, Order: 2},
		{ID: "m", Priority: SectionP1, Order: 0},
		{ID: "b", Priority: SectionP0, Order: 1},
		{ID: "x", Priority: SectionP2},
	}
	SortContextSections(in)
	want := []string{"b", "a", "z", "m", "x"}
	for i, id := range want {
		if in[i].ID != id {
			t.Fatalf("排序结果 = %v, want %v", ids(in), want)
		}
	}
}

func ids(s []ContextSection) []string {
	out := make([]string, len(s))
	for i := range s {
		out[i] = s[i].ID
	}
	return out
}

func TestChapterNumOf(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"001.md", 1},
		{"015.md", 15},
		{"001a.md", 1},
		{"", 0},
		{"abc.md", 0},
		{".md", 0},
	}
	for _, c := range cases {
		if got := ChapterNumOf(c.in); got != c.want {
			t.Errorf("ChapterNumOf(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestDefaultContextMaxRunes_MatchesNovelcontextBaseline(t *testing.T) {
	// handoff §4-1 指定 novelcontext.DefaultMaxRunes = 2000，契约层声明必须同值，
	// 不得另立第二套阈值。
	if DefaultContextMaxRunes != 2000 {
		t.Fatalf("DefaultContextMaxRunes = %d, want 2000", DefaultContextMaxRunes)
	}
}

// ── 落盘兼容：真实文件级往返（含未知字段不丢）───────────────

func TestForeshadowFile_UnknownFieldsAreIgnoredNotFatal(t *testing.T) {
	raw := `{"schema_version":2,"items":[{"id":"a","description":"d","planted_in":"001.md","status":"resolved","unknown_future_field":123}]}`
	var ff ForeshadowFile
	if err := json.Unmarshal([]byte(raw), &ff); err != nil {
		t.Fatalf("含未知字段的 JSON 必须可解析（前向兼容）: %v", err)
	}
	if ff.SchemaVersion != 2 || len(ff.Items) != 1 || ff.Items[0].Status != ForeshadowResolved {
		t.Errorf("解析结果异常: %+v", ff)
	}
}

func TestForeshadowFile_WriteReadViaDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foreshadows.json")
	original := []byte(legacyForeshadowsJSON)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ff ForeshadowFile
	if err := json.Unmarshal(data, &ff); err != nil {
		t.Fatalf("磁盘旧文件读取失败: %v", err)
	}
	out, err := json.MarshalIndent(&ff, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ff2 ForeshadowFile
	if err := json.Unmarshal(again, &ff2); err != nil {
		t.Fatalf("写回后再读失败: %v", err)
	}
	if len(ff2.Items) != 2 || ff2.Items[0].ID != ff.Items[0].ID {
		t.Errorf("磁盘往返丢条目: %+v", ff2)
	}
}
