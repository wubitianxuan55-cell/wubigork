package novelgate

// ── 变更声明门禁 五类正反测试（阶段八 8.5 出口判据）──────────
//
// 「门禁五类检查各有正反测试」：每类一正一反；拒收路径人可读=Message 中文点名。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func gateChars() []types.Character {
	return []types.Character{
		{ID: "c1", Name: "秦昭", Aliases: []string{"昭哥"}, Status: "Alive", StatusChangedChapter: 3, StateUpdatedChapter: 3},
		{ID: "c2", Name: "沈砚", Aliases: []string{"沈公子"}},
		{ID: "c3", Name: "王爷", ExcludeHits: []string{"老王爷"}},
	}
}

func gateFFs() []types.Foreshadow {
	return []types.Foreshadow{
		{ID: "ab12cd34", Title: "玄铁剑的来历", Status: types.ForeshadowPlanted, PlantedIn: "003.md"},
		{ID: "ef987654", Title: "断指旧誓", Status: types.ForeshadowRevealed, PlantedIn: "002.md", RevealedIn: "007.md"},
	}
}

func declCodes(issues []Issue) map[string]bool {
	m := map[string]bool{}
	for _, it := range issues {
		m[it.Code] = true
	}
	return m
}

// 正例：全合法声明零 S2（仅 timeline 缺失 S3 或全绿）。
func TestChangeDeclaration_CleanPass(t *testing.T) {
	decl := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{
			{Name: "昭哥", New: "与沈砚决裂", Kind: "survival"}, // 别名命中+正文提及+水位允许
		},
		Foreshadows: []types.ChangeDeclForeshadow{
			{ID: "ab12cd34", Action: "resolved"},       // 挂回既有未回收条目
			{Action: "planted", Description: "井底徽记之谜"}, // 新增带描述
		},
		Timeline: "三日之后",
	}
	content := "秦昭——不，昭哥——转身离井。沈砚在雨中沉默。"
	issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, content)
	if len(DeclBlockingIssues(issues)) != 0 {
		t.Fatalf("合法声明不得有 S2: %+v", issues)
	}
}

// ① 未知实体：名字不在角色卡（反）→ 正（改用卡内名即消）。
func TestChangeDeclaration_UnknownEntity(t *testing.T) {
	decl := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{{Name: "林晚", New: "黑化"}},
	}
	issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭转身。")
	codes := declCodes(issues)
	if !codes["decl_unknown_entity"] {
		t.Fatalf("应有 decl_unknown_entity: %+v", issues)
	}
	if issues[0].Severity != "S2" || !strings.Contains(issues[0].Message, "林晚") {
		t.Fatalf("拒收须人可读（S2+点名）: %+v", issues[0])
	}
	// 正：卡内名
	decl.CharacterStates[0].Name = "秦昭"
	issues = ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭转身。")
	if declCodes(issues)["decl_unknown_entity"] {
		t.Fatalf("卡内名不应报未知实体: %+v", issues)
	}
}

// ② 引用存在：resolved 空 ID/坏 ID/重复回收（反）→ 正（挂回既有未回收条目）。
func TestChangeDeclaration_MissingReference(t *testing.T) {
	cases := []struct {
		name string
		fs   types.ChangeDeclForeshadow
	}{
		{"空 ID", types.ChangeDeclForeshadow{Action: "resolved", Title: "断誓"}},
		{"坏 ID", types.ChangeDeclForeshadow{ID: "deadbeef", Action: "resolved"}},
		{"重复回收", types.ChangeDeclForeshadow{ID: "ef987654", Action: "resolved"}},
	}
	for _, tc := range cases {
		decl := &types.ChapterChangeDeclaration{Foreshadows: []types.ChangeDeclForeshadow{tc.fs}}
		issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭。")
		if !declCodes(issues)["decl_missing_reference"] {
			t.Fatalf("%s: 应有 decl_missing_reference: %+v", tc.name, issues)
		}
	}
	// 正
	decl := &types.ChapterChangeDeclaration{Foreshadows: []types.ChangeDeclForeshadow{{ID: "ab12cd34", Action: "resolved"}}}
	issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭。")
	if declCodes(issues)["decl_missing_reference"] {
		t.Fatalf("挂回既有未回收条目不应报: %+v", issues)
	}
}

// ③ 状态连续：水位回退（反：第 3 章已变更，第 2 章声明变更）→ 正（第 5 章声明）。
func TestChangeDeclaration_StateRegression(t *testing.T) {
	decl := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{{Name: "秦昭", New: "重伤", Kind: "survival"}},
	}
	issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 2, "秦昭倒下。")
	if !declCodes(issues)["decl_state_regression"] {
		t.Fatalf("水位回退应有 decl_state_regression: %+v", issues)
	}
	// 正：章号前进
	issues = ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭倒下。")
	if declCodes(issues)["decl_state_regression"] {
		t.Fatalf("章号前进不应报回退: %+v", issues)
	}
	// 心理水位单查：存活水位高但心理查的是 StateUpdatedChapter
	d2 := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{{Name: "秦昭", New: "心灰意冷"}}, // 空 kind=心理
	}
	issues = ChangeDeclarationIssues(d2, gateChars(), gateFFs(), 2, "秦昭。")
	if !declCodes(issues)["decl_state_regression"] {
		t.Fatalf("心理水位回退应有 decl_state_regression: %+v", issues)
	}
}

// ④ 出场对账：声明变更但正文零提及（反）→ 正（别名命中也算提及）。
func TestChangeDeclaration_CastMismatch(t *testing.T) {
	decl := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{{Name: "沈砚", New: "离队"}},
	}
	issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭独自前行，无人同行。")
	if !declCodes(issues)["decl_cast_mismatch"] {
		t.Fatalf("正文零提及应有 decl_cast_mismatch: %+v", issues)
	}
	// 正：别名命中
	issues = ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭想起沈公子的断指。")
	if declCodes(issues)["decl_cast_mismatch"] {
		t.Fatalf("别名命中不应报出场对账: %+v", issues)
	}
	// 排除词：角色卡排除「老王爷」后，正文的「老王爷」不再命中该卡（排除词
	// 正是为了把「老王爷」从「王爷」的命中域里剔掉）——如实报出场对账。
	decl3 := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{{Name: "王爷", New: "驾崩", Kind: "survival"}},
	}
	issues = ChangeDeclarationIssues(decl3, gateChars(), gateFFs(), 5, "老王爷驾崩了。")
	if !declCodes(issues)["decl_cast_mismatch"] {
		t.Fatalf("排除词剔除后老王爷不应算王爷命中（这正是排除词语义）: %+v", issues)
	}
	// 正：正文出现的是「王爷」本尊
	issues = ChangeDeclarationIssues(decl3, gateChars(), gateFFs(), 5, "王爷亲征，驾崩于途。")
	if declCodes(issues)["decl_cast_mismatch"] {
		t.Fatalf("本尊命中不应报: %+v", issues)
	}
}

// ⑤ 伏笔动作：枚举外/planted 缺描述（反）→ 正（补描述即消）。
func TestChangeDeclaration_InvalidAction(t *testing.T) {
	decl := &types.ChapterChangeDeclaration{Foreshadows: []types.ChangeDeclForeshadow{
		{Action: "vanished"},
		{Action: "planted", Title: "无描述"},
	}}
	issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭。")
	if got := declCodes(issues)["decl_invalid_action"]; !got {
		t.Fatalf("应有 decl_invalid_action: %+v", issues)
	}
	n := 0
	for _, it := range issues {
		if it.Code == "decl_invalid_action" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("两条非法各报一条，得 %d: %+v", n, issues)
	}
	// 正
	decl.Foreshadows[0].Action = "hinted"
	decl.Foreshadows[1].Description = "井底徽记"
	issues = ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭。")
	if declCodes(issues)["decl_invalid_action"] {
		t.Fatalf("修正后不应报: %+v", issues)
	}
}

// 软降级口径：nil 声明=零问题（不是违规）；timeline 缺失=S3 不进拒收集。
func TestChangeDeclaration_SoftDegradation(t *testing.T) {
	if issues := ChangeDeclarationIssues(nil, gateChars(), gateFFs(), 5, ""); issues != nil {
		t.Fatalf("nil 声明应零问题: %+v", issues)
	}
	decl := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{{Name: "秦昭", New: "出发"}},
	}
	issues := ChangeDeclarationIssues(decl, gateChars(), gateFFs(), 5, "秦昭出发。")
	if len(DeclBlockingIssues(issues)) != 0 {
		t.Fatalf("仅缺 timeline 不得拒收: %+v", issues)
	}
	found := false
	for _, it := range issues {
		if it.Code == "decl_timeline_missing" && it.Severity == "S3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("缺 timeline 应有 S3 提示: %+v", issues)
	}
}
