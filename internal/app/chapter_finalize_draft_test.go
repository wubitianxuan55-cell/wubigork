package app

// ── 定稿草案（FinalizeChapterDraft）假 LLM 测试 ───────────────
//
// 核心断言=建议制红线：草案提取（ExtractOnly）成功后**零落盘**——分析路五件
// （analysis-v2/记忆/标注/伏笔登记表/角色状态）一个都不许出现，落库只发生在
// Apply（作者确认）之后。

import (
	"fmt"
	"testing"

	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/types"
)

// 合法 V2 分析载荷（最小骨架）：一条角色状态 + 一条伏笔 planted + 阶段收官所需摘要。
// 围栏包裹：RetryJSON 对裸 JSON 有已知的桩误判（app 测试既坑），草案链同一待遇。
const finDraftReply = "```json\n" + `{
  "hooks": [], "foreshadows": [
    {"title": "井底徽记", "content": "玄铁剑刻着秦家徽记", "type": "planted", "category": "item"}
  ],
  "character_states": [
    {"name": "秦昭", "old_state": "隐忍", "new_state": "决裂出走"}
  ],
  "conflict": {"types": ["人与人"], "description": "决裂"},
  "plot_points": [], "scenes": [], "suggestions": [],
  "scores": {"pacing": 7, "engagement": 7, "coherence": 7}
}` + "\n```"

func TestFinalizeChapterDraft_ExtractOnlyZeroPersist(t *testing.T) {
	env := newPlanStubEnv(t, finDraftReply)
	a := env.a
	pm := env.pm

	// 前置：正文 + 角色卡（名字对齐）+ 第 1~9 章摘要（阶段合账素材，第 10 章=收官）
	if err := pm.WriteChapter(10, "秦昭夜探听风阁，与沈砚决裂。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteCharacters(&types.CharacterFile{Characters: []types.Character{
		{ID: "c1", Name: "秦昭"},
	}}); err != nil {
		t.Fatal(err)
	}
	var nodes []types.OutlineNode
	for i := 1; i <= 10; i++ {
		nodes = append(nodes, types.OutlineNode{
			ID: fmt.Sprintf("n%d", i), Title: fmt.Sprintf("第%d章", i),
			OrderIndex: i, Status: types.OutlineDone, Summary: fmt.Sprintf("第%d章摘要", i),
		})
	}
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nodes}); err != nil {
		t.Fatal(err)
	}

	// 无 client 的 analysisAgent 换成假 LLM 版（ExtractOnly 走模型；SyncForeshadows 不触发）
	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.AnalysisTemperature = 0.2
	a.analysisAgent = analysis.New(a.core.client, pm, cfg, a.eng)

	res, err := a.FinalizeChapterDraft(10, "")
	if err != nil {
		t.Fatalf("定稿草案: %v", err)
	}
	if res["finalized"] != false {
		t.Fatalf("未定稿章 finalized 应为 false: %v", res["finalized"])
	}

	// 草案内容：角色条目按名对齐 ID；伏笔透传；阶段合账在列
	chars := res["characters"].([]finalizeCharDraft)
	if len(chars) != 1 || chars[0].ID != "c1" || chars[0].Text != "决裂出走" {
		t.Fatalf("角色草案: %+v", chars)
	}
	hits := res["foreshadows"].([]types.ForeshadowHit)
	if len(hits) != 1 || hits[0].Title != "井底徽记" {
		t.Fatalf("伏笔草案: %+v", hits)
	}
	sr, ok := res["stageRecap"].(map[string]interface{})
	if !ok || sr["text"] == "" || sr["start"] != 1 || sr["end"] != 10 {
		t.Fatalf("阶段合账草案: %+v", res["stageRecap"])
	}

	// ── 建议制红线：五路落盘一件都不许有 ──
	if _, ok, _ := pm.ReadChapterFinalization(10, ""); ok {
		t.Fatal("草案阶段不得写定稿档")
	}
	if e, ok, _ := pm.ReadStageRecap(1); ok && e != nil {
		t.Fatal("草案阶段不得写合账缓存")
	}
	if ff, err := pm.ReadForeshadows(); err == nil && ff != nil && len(ff.Items) > 0 {
		t.Fatalf("草案阶段不得写伏笔登记表: %+v", ff.Items)
	}
	if cf, _ := pm.ReadCharacters(); len(cf.Characters[0].Progressions) != 0 {
		t.Fatalf("草案阶段不得写角色状态卡: %+v", cf.Characters[0].Progressions)
	}
	if av2, err := pm.ReadAnalysisV2File(); err == nil && av2 != nil && len(av2.Items) > 0 {
		t.Fatal("草案阶段不得写 analysis-v2.json")
	}
	if mf, err := pm.ReadChapterMemories(10); err == nil && mf != nil && len(mf.Items) > 0 {
		t.Fatal("草案阶段不得写章节记忆")
	}
}

func TestFinalizeChapterDraft_AlreadyFinalizedIdempotent(t *testing.T) {
	env := newPlanStubEnv(t, finDraftReply)
	a := env.a
	pm := env.pm
	if err := pm.WriteChapter(6, "正文。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteCharacters(&types.CharacterFile{Characters: []types.Character{{ID: "c1", Name: "秦昭"}}}); err != nil {
		t.Fatal(err)
	}
	// 预置已定稿
	if err := pm.WriteChapterFinalization(&types.ChapterFinalization{
		Version: 1, ChapterNum: 6, FinalizedAt: "2026-10-14T23:00:00+08:00", Note: "首定",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := a.FinalizeChapterDraft(6, "")
	if err != nil {
		t.Fatalf("已定稿章再草案: %v", err)
	}
	if res["finalized"] != true {
		t.Fatalf("应返回 finalized=true: %v", res["finalized"])
	}
	if _, ok := res["finalization"]; !ok {
		t.Fatal("应回带既有定稿档")
	}
	// 非收官章：无合账草案
	if _, ok := res["stageRecap"]; ok {
		t.Fatalf("第6章非阶段收官，不应有合账草案: %v", res["stageRecap"])
	}
}
