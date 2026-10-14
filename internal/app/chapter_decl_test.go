package app

// ── 章级变更声明 批B 测试（阶段八 8.5）──────────────────────
//
// 覆盖：切段/围栏剥离/坏 JSON 软降级；过闸随摘要落盘（拒收不落）；
// 定稿草案声明优先（零 LLM 预填充——无 client 也出草案）与回落现提取。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/novelgate"

	"github.com/gaea/gaea/internal/types"
)

const declJSON = "```json\n{\"character_states\":[{\"name\":\"秦昭\",\"old\":\"隐忍\",\"new\":\"决裂出走\",\"kind\":\"survival\"}],\"foreshadows\":[{\"id\":\"ab12cd34\",\"action\":\"resolved\"}],\"timeline\":\"三日后\"}\n```"

func TestChapterChanges_ParseAndGate(t *testing.T) {
	summaryRegion := "---CHAPTER_SUMMARY---\n关键事件：决裂。\n" + chapterChangesMarker + "\n" + declJSON

	// 切段：摘要区不含声明 JSON
	summaryPart, declPart := splitDeclSegment(summaryRegion)
	if strings.Contains(summaryPart, "character_states") {
		t.Fatalf("摘要段被声明污染: %q", summaryPart)
	}
	if !strings.Contains(declPart, "秦昭") {
		t.Fatalf("声明段提取失败: %q", declPart)
	}

	// 解析+校验全链（合法声明，角色在卡+正文提及+引用存在）
	chars := []types.Character{{ID: "c1", Name: "秦昭"}}
	ffs := []types.Foreshadow{{ID: "ab12cd34", Title: "玄铁剑", Status: types.ForeshadowPlanted, PlantedIn: "003.md"}}
	decl, issues := resolveChapterChanges(summaryRegion, "秦昭转身离去。", chars, ffs, 5)
	if decl == nil {
		t.Fatal("合法声明应解析成功")
	}
	if len(issues) != 0 {
		t.Fatalf("合法声明应零问题: %+v", issues)
	}

	// 过闸随摘要落盘
	summary := &types.ChapterSummary{Title: "第5章", Summary: "决裂。"}
	if !attachDeclToSummary(summary, decl, issues) {
		t.Fatal("过闸声明应落盘")
	}
	if summary.DeclChanges == nil || len(summary.DeclChanges.CharacterStates) != 1 {
		t.Fatalf("DeclChanges 未落: %+v", summary.DeclChanges)
	}

	// 拒收：声明的角色不在卡（unknown_entity S2）→ 不落盘
	bad := &types.ChapterChangeDeclaration{
		CharacterStates: []types.ChangeDeclState{{Name: "路人甲", New: "出场"}},
	}
	badIssues := novelgate.ChangeDeclarationIssues(bad, chars, ffs, 5, "秦昭。")
	if attachDeclToSummary(&types.ChapterSummary{Title: "x"}, bad, badIssues) {
		t.Fatal("拒收声明不得落盘")
	}
}

// 未声明/坏 JSON：软降级（decl=nil，零 issues——不计违规）。
func TestChapterChanges_SoftDegradation(t *testing.T) {
	decl, issues := resolveChapterChanges("---CHAPTER_SUMMARY---\n只有摘要。", "正文。", nil, nil, 5)
	if decl != nil || issues != nil {
		t.Fatalf("未声明应 (nil,nil): %+v %+v", decl, issues)
	}
	region := "---CHAPTER_SUMMARY---\n摘要。" + chapterChangesMarker + "\n{坏 JSON"
	decl, issues = resolveChapterChanges(region, "正文。", nil, nil, 5)
	if decl != nil || issues != nil {
		t.Fatalf("坏 JSON 应软降级: %+v %+v", decl, issues)
	}
	// 裸 JSON（无围栏）也解析
	region = chapterChangesMarker + "\n{\"character_states\":[],\"timeline\":\"次日\"}"
	decl, _ = resolveChapterChanges(region, "", nil, nil, 5)
	if decl == nil || decl.Timeline != "次日" {
		t.Fatalf("裸 JSON 应解析: %+v", decl)
	}
}

// TestFinalizeDraft_FromDeclaration 定稿草案声明优先：摘要里有过闸声明时零 LLM
// 出草案（无 client 也照出——声明路径不依赖模型），fromDeclaration 标记回带。
func TestFinalizeDraft_FromDeclaration(t *testing.T) {
	a, pm := newFinalizeTestApp(t)
	if err := pm.WriteChapter(5, "秦昭决裂。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteCharacters(&types.CharacterFile{Characters: []types.Character{{ID: "c1", Name: "秦昭"}}}); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapterSummary(5, &types.ChapterSummary{
		Title: "第5章", Summary: "决裂。",
		DeclChanges: &types.ChapterChangeDeclaration{
			CharacterStates: []types.ChangeDeclState{{Name: "秦昭", New: "决裂出走", Kind: "survival"}},
			Foreshadows:     []types.ChangeDeclForeshadow{{ID: "ab12cd34", Action: "resolved", Title: "玄铁剑"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	res, err := a.FinalizeChapterDraft(5, "")
	if err != nil {
		t.Fatalf("声明路径草案: %v", err)
	}
	if res["fromDeclaration"] != true {
		t.Fatalf("应回带 fromDeclaration: %v", res)
	}
	chars := res["characters"].([]finalizeCharDraft)
	if len(chars) != 1 || chars[0].ID != "c1" || chars[0].Text != "决裂出走" {
		t.Fatalf("角色草案: %+v", chars)
	}
	fs := res["foreshadows"].([]types.ForeshadowHit)
	if len(fs) != 1 || fs[0].Type != "resolved" || fs[0].ReferenceStableID != "ab12cd34" {
		t.Fatalf("伏笔草案: %+v", fs)
	}

	// 无声明 → 走 ExtractOnly 兜底（无 client 报错，路径区分正确）
	if _, err := a.FinalizeChapterDraft(6, ""); err == nil {
		t.Fatal("无声明无 client 应走模型路径并报错")
	}
}
