package app

// ── 章节定稿回写 测试（阶段八 8.4）───────────────────────────
//
// 覆盖（出口判据对应）：
//   - 主线 Apply 全链：角色 Progressions 追加（8.2 状态卡对齐）+ 同章重定稿幂等
//     （不重复堆积）+ 阶段合账落缓存 + 定稿档回执留痕；
//   - 「定稿后下一章读到最新状态」：阶段开篇章 stageRecapSection 命中缓存不再
//     烧 LLM（无 client 下回落链被缓存截断，注入文本=缓存文本）；
//   - 分支章：只标记+Skipped 申报（登记表/合账/状态卡主线口径）；
//   - 空章拒收 / 非法分支拒收 / 未定稿 Get 正常态。

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/config"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newFinalizeTestApp 最小 App（无 AI client：Draft 诚实报错；Apply 的确定性三件
// 路不依赖模型——伏笔件走 SyncForeshadows 是纯确定性匹配）。
func newFinalizeTestApp(t *testing.T) (*App, *project.Manager) {
	t.Helper()
	a := &App{core: &core{}}
	a.writingState = &writingState{core: a.core, app: a}
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "定稿测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	// 无 client 的 analysisAgent：伏笔件 SyncForeshadows 是纯确定性匹配（零 LLM），
	// Draft 的 LLM 提取路会如实报错——恰好覆盖两条纪律。
	a.analysisAgent = analysis.New(nil, pm, &config.Config{}, nil)
	return a, pm
}

func TestFinalizeApply_MainlineFullChain(t *testing.T) {
	a, pm := newFinalizeTestApp(t)

	// 前置：正文 + 两角色 + 阶段收官素材（第 10 章=首阶段收官）
	if err := pm.WriteChapter(10, "秦昭夜探听风阁，与沈砚决裂。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteCharacters(&types.CharacterFile{Characters: []types.Character{
		{ID: "c1", Name: "秦昭"},
		{ID: "c2", Name: "沈砚"},
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

	payload := map[string]interface{}{
		"note": "本卷收官",
		"characters": []map[string]interface{}{
			{"id": "c1", "name": "秦昭", "field": "status", "text": "与沈砚决裂，独走", "note": ""},
		},
		"foreshadows": []map[string]interface{}{
			{"title": "玄铁剑的来历", "content": "井底玄铁剑刻着秦家徽记", "type": "planted", "category": "item"},
		},
		"stage_recap": map[string]interface{}{"stage": 1, "start": 1, "end": 10, "text": "第一阶段：秦昭崛起与决裂。"},
	}
	raw, _ := json.Marshal(payload)
	res, err := a.FinalizeChapterApply(10, "", string(raw))
	if err != nil {
		t.Fatalf("定稿落库: %v", err)
	}
	if res["finalized"] != true {
		t.Fatalf("回执 finalized: %v", res["finalized"])
	}

	// 件一：角色卡 Progressions 追加（8.2 状态卡对齐，「下一章读到最新状态」）
	cf, _ := pm.ReadCharacters()
	if len(cf.Characters[0].Progressions) != 1 {
		t.Fatalf("秦昭应有一条状态演进: %+v", cf.Characters[0].Progressions)
	}
	p := cf.Characters[0].Progressions[0]
	if p.Chapter != 10 || p.Text != "与沈砚决裂，独走" {
		t.Fatalf("演进条目不符: %+v", p)
	}

	// 件二：伏笔登记表 planted
	ff, _ := pm.ReadForeshadows()
	if len(ff.Items) != 1 || ff.Items[0].Status != types.ForeshadowPlanted {
		t.Fatalf("伏笔登记表应有一条 planted: %+v", ff.Items)
	}

	// 件三：合账缓存
	e, ok, _ := pm.ReadStageRecap(1)
	if !ok || e.Text != "第一阶段：秦昭崛起与决裂。" || e.SourceChapter != 10 {
		t.Fatalf("合账缓存: %+v ok=%v", e, ok)
	}

	// 定稿档留痕
	file, ok, err := pm.ReadChapterFinalization(10, "")
	if err != nil || !ok || file.Applied == nil || file.Applied.Characters != 1 {
		t.Fatalf("定稿档: %+v ok=%v err=%v", file, ok, err)
	}

	// 幂等：同章重定稿（改文本）不重复堆积
	payload["characters"] = []map[string]interface{}{
		{"id": "c1", "name": "秦昭", "field": "status", "text": "决裂后远走北境", "note": ""},
	}
	raw, _ = json.Marshal(payload)
	if _, err := a.FinalizeChapterApply(10, "", string(raw)); err != nil {
		t.Fatalf("重定稿: %v", err)
	}
	cf, _ = pm.ReadCharacters()
	if len(cf.Characters[0].Progressions) != 1 || cf.Characters[0].Progressions[0].Text != "决裂后远走北境" {
		t.Fatalf("重定稿应整批替换: %+v", cf.Characters[0].Progressions)
	}
}

func TestFinalizeApply_CacheServesStageOpener(t *testing.T) {
	a, pm := newFinalizeTestApp(t)

	// 第 1~11 章大纲+摘要（第 11 章=阶段开篇章，素材来自第 1~10 章摘要）
	var nodes []types.OutlineNode
	for i := 1; i <= 11; i++ {
		nodes = append(nodes, types.OutlineNode{
			ID: fmt.Sprintf("n%d", i), Title: fmt.Sprintf("第%d章", i),
			OrderIndex: i, Status: types.OutlineDone, Summary: fmt.Sprintf("第%d章摘要", i),
		})
	}
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nodes}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 10; i++ {
		if err := pm.WriteChapterSummary(i, &types.ChapterSummary{Title: fmt.Sprintf("第%d章", i), Summary: fmt.Sprintf("第%d章摘要", i)}); err != nil {
			t.Fatal(err)
		}
	}
	// 第 10 章定稿落合账缓存（绕过 Apply 直接 upsert——Apply 链已在上一用例覆盖）
	if err := pm.UpsertStageRecap(1, types.StageRecapEntry{
		Start: 1, End: 10, Text: "定稿合账：秦昭崛起。", SourceChapter: 10,
	}); err != nil {
		t.Fatal(err)
	}

	// 无 client：synthesizeStageRecap 走不通会回落 digest——缓存命中必须先于
	// 回落发生，注入文本=缓存文本（8.4 出口判据：定稿后下一章读到最新状态）。
	sec := a.stageRecapSection(pm, 11)
	if !strings.Contains(sec, "定稿合账：秦昭崛起。") {
		t.Fatalf("阶段开篇章应命中合账缓存:\n%s", sec)
	}

	// 缓存区间失配=脏数据，弃用（回落 digest——无 client 下即逐章摘要清单）
	if err := pm.UpsertStageRecap(1, types.StageRecapEntry{Start: 2, End: 9, Text: "脏区间"}); err != nil {
		t.Fatal(err)
	}
	sec = a.stageRecapSection(pm, 11)
	if strings.Contains(sec, "脏区间") || !strings.Contains(sec, "第1章摘要") {
		t.Fatalf("区间失配应弃用缓存回落摘要清单:\n%s", sec)
	}
}

func TestFinalizeApply_BranchOnlyMarks(t *testing.T) {
	a, pm := newFinalizeTestApp(t)
	if err := pm.WriteChapterBranch(5, "a", "分支线正文。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteCharacters(&types.CharacterFile{Characters: []types.Character{{ID: "c1", Name: "秦昭"}}}); err != nil {
		t.Fatal(err)
	}

	payload := map[string]interface{}{
		"characters": []map[string]interface{}{
			{"id": "c1", "name": "秦昭", "text": "分支状态", "note": ""},
		},
	}
	raw, _ := json.Marshal(payload)
	res, err := a.FinalizeChapterApply(5, "a", string(raw))
	if err != nil {
		t.Fatalf("分支定稿: %v", err)
	}
	if res["skipped"] == "" {
		t.Fatalf("分支定稿应有 Skipped 申报: %v", res)
	}

	// 定稿档只标记，三件套不落
	cf, ok, _ := pm.ReadChapterFinalization(5, "a")
	if !ok || cf.Skipped == "" || cf.Applied != nil {
		t.Fatalf("分支定稿档: %+v ok=%v", cf, ok)
	}
	charFile, _ := pm.ReadCharacters()
	if len(charFile.Characters[0].Progressions) != 0 {
		t.Fatalf("分支章不得写主线状态卡: %+v", charFile.Characters[0].Progressions)
	}
	if _, ok, _ := pm.ReadStageRecap(1); ok {
		t.Fatal("分支章不得写合账缓存")
	}
}

func TestFinalizeGuards(t *testing.T) {
	a, pm := newFinalizeTestApp(t)

	// 空章拒收
	if _, err := a.FinalizeChapterApply(9, "", "{}"); err == nil {
		t.Fatal("空章应拒收")
	}
	// 非法分支拒收
	if _, err := a.FinalizeChapterApply(9, "z", "{}"); err == nil {
		t.Fatal("非法分支应拒收")
	}
	// 坏 JSON 拒收
	if err := pm.WriteChapter(9, "有正文"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.FinalizeChapterApply(9, "", "{bad"); err == nil {
		t.Fatal("坏载荷应拒收")
	}
	// Draft 无 client 诚实报错（建议制：无模型不落库）
	if _, err := a.FinalizeChapterDraft(9, ""); err == nil {
		t.Fatal("无 client 的 Draft 应报错")
	}
	// Get：未定稿正常态
	got, err := a.ChapterFinalizationGet(9, "")
	if err != nil || got != nil {
		t.Fatalf("未定稿 Get 应 nil,nil: %+v err=%v", got, err)
	}
	// 阶段收官判定：第 10/20 章是，第 1/11 章不是
	for _, tc := range []struct {
		num  int
		want bool
	}{{10, true}, {20, true}, {1, false}, {11, false}, {5, false}} {
		if got := isStageCloser(tc.num); got != tc.want {
			t.Fatalf("isStageCloser(%d)=%v want %v", tc.num, got, tc.want)
		}
	}
	if s, e := stageCloserRange(10); s != 1 || e != 10 {
		t.Fatalf("阶段区间: %d~%d", s, e)
	}
}
