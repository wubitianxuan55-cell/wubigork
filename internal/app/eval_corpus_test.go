package app

// 评测回归夹具：3×30 章病历（v4.435 留池头名，v4.445 落地）。
//
// 三本书 × 30 章的确定性语料，病理画像可区分，给评测链（快照/基线/对比）做
// 规模级回归保护。断言全部走**相对指标**（病历书 vs 基准书的指标差）——对
// 打分器内部参数变化鲁棒，对链条退化敏感（打分归零/聚合错位/章漏扫即红）：
//
//   healthy 基准书：自然行文（对话+情态标记、无 AI 腔模式、无预告尾），
//                   伏笔 10 埋 8 收（回收率 0.8）——各项指标的「好」端基线。
//   flavor   病历书：行文同 healthy 的伏笔表（10 埋 8 收），正文换重 AI 腔
//                   （重复感叹+否定翻转+解释腔）——隔离「AI 味」单一维度。
//   hollow   病历书：正文同 healthy + 每章一条超长段（>400 rune，gate
//                   paragraph_too_long=S3 确定命中），伏笔全部埋设不回收
//                   ——隔离「回收率/质量闸」维度。

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

const evalCorpusChapters = 30

// evalHealthyChapter 基准正文：开场/对话/情态随章号轮换（同书内不重复），无
// novelstyle 负模式、无 novelgate 预告尾标记。
func evalHealthyChapter(i int) string {
	weather := []string{"落了一夜的雨", "刮了半日的风", "晴得发白"}[i%3]
	dialog := []string{
		"「你还回来。」她说，声音没有起伏。",
		"「路上冷。」他把姜汤推过去，没看她的眼睛。",
		"「账都清了。」她说完这句，屋里只剩灯芯的响。",
	}[i%3]
	feel := []string{
		"她攥紧了袖口。",
		"他喉头一紧，把到嘴的话咽了回去。",
		"灯花跳了一下，两个人都没动。",
	}[(i/3)%3]
	return fmt.Sprintf("第%d章的雨%s。他把伞收在门边，水顺着伞尖滴成一排。%s他没接话，把那张字条压在茶杯底下。%s",
		i, weather, dialog, feel)
}

// evalFlavorChapter 病历正文：重复感叹句 + 否定翻转 + 解释腔（novelstyle
// 负模式全家桶），AI 味应显著高于基准。
func evalFlavorChapter(i int) string {
	heavy := strings.Repeat("他感到无比的震惊。此外，值得注意的是，这一切仿佛命运的安排。", 8)
	return fmt.Sprintf("%s这不是偶然，而是必然。殊不知，真正的原因还藏在暗处。（%d）", heavy, i)
}

// evalCorpusForeshadows 伏笔表：total 埋、revealedCount 收（状态口径与
// evalTestProject 同源）。
func evalCorpusForeshadows(total, revealedCount int) []types.Foreshadow {
	items := make([]types.Foreshadow, 0, total)
	for i := 0; i < total; i++ {
		st := types.ForeshadowPlanted
		if i < revealedCount {
			st = types.ForeshadowRevealed
		}
		items = append(items, types.Foreshadow{
			ID:          fmt.Sprintf("f%d", i+1),
			Category:    "plot",
			Description: fmt.Sprintf("伏笔%d", i+1),
			PlantedIn:   fmt.Sprintf("%03d.md", i%evalCorpusChapters+1),
			Status:      st,
		})
	}
	return items
}

// buildEvalCorpusBook 按画像生成一本书（30 章 + 伏笔表）并切换为当前项目。
func buildEvalCorpusBook(t *testing.T, a *App, name string, chapter func(i int) string, foreshadows []types.Foreshadow) *project.Manager {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "corpus", name), name, "都市", "", "")
	if err != nil {
		t.Fatalf("创建 %s: %v", name, err)
	}
	for i := 1; i <= evalCorpusChapters; i++ {
		if err := pm.WriteChapter(i, chapter(i)); err != nil {
			t.Fatalf("%s 写章 %d: %v", name, i, err)
		}
	}
	if err := pm.WriteForeshadows(&types.ForeshadowFile{SchemaVersion: 2, Items: foreshadows}); err != nil {
		t.Fatalf("%s 写伏笔: %v", name, err)
	}
	a.setPM(pm)
	return pm
}

// TestEvalCorpusSnapshotDeterministic 三本书各自：同一状态两次快照逐字节相同
// （§4 确定性验收线在 30 章规模复验），章数恒 30（无漏扫）。
func TestEvalCorpusSnapshotDeterministic(t *testing.T) {
	a := newGateEmptyApp()
	books := []struct {
		name    string
		chapter func(int) string
		fb      []types.Foreshadow
	}{
		{"healthy", evalHealthyChapter, evalCorpusForeshadows(10, 8)},
		{"flavor", evalFlavorChapter, evalCorpusForeshadows(10, 8)},
		{"hollow", func(i int) string { return evalHealthyChapter(i) + strings.Repeat("这一段长得没有道理，句号迟迟不来，念头一个接一个地往下坠，像是把整章的呼吸都压进了一口锅里，直到读者忘了换气。", 8) }, evalCorpusForeshadows(10, 0)},
	}
	for _, b := range books {
		pm := buildEvalCorpusBook(t, a, b.name, b.chapter, b.fb)
		j1, _ := json.Marshal(a.buildEvalSnapshot(pm))
		j2, _ := json.Marshal(a.buildEvalSnapshot(pm))
		if string(j1) != string(j2) {
			t.Fatalf("%s 快照必须确定性：\n%s\n%s", b.name, j1, j2)
		}
		var body evalSnapshotBody
		if err := json.Unmarshal(j1, &body); err != nil {
			t.Fatal(err)
		}
		if body.Chapters != evalCorpusChapters {
			t.Fatalf("%s 章数应 %d got %d（漏扫）", b.name, evalCorpusChapters, body.Chapters)
		}
		if body.Chars <= 0 {
			t.Fatalf("%s 字数应非零", b.name)
		}
	}
}

// TestEvalCorpusProfilesSeparate 病理画像可区分：AI 味病历 > 基准（均值守恒
// 失效=打分链退化）；断链书回收率 0 且 S1 每章一条；基准书 S1 干净。
func TestEvalCorpusProfilesSeparate(t *testing.T) {
	a := newGateEmptyApp()

	pmHealthy := buildEvalCorpusBook(t, a, "healthy", evalHealthyChapter, evalCorpusForeshadows(10, 8))
	healthy := a.buildEvalSnapshot(pmHealthy)

	pmFlavor := buildEvalCorpusBook(t, a, "flavor", evalFlavorChapter, evalCorpusForeshadows(10, 8))
	flavor := a.buildEvalSnapshot(pmFlavor)

	pmHollow := buildEvalCorpusBook(t, a, "hollow", func(i int) string {
		return evalHealthyChapter(i) + strings.Repeat("这一段长得没有道理，句号迟迟不来，念头一个接一个地往下坠，像是把整章的呼吸都压进了一口锅里，直到读者忘了换气。", 8)
	}, evalCorpusForeshadows(10, 0))
	hollow := a.buildEvalSnapshot(pmHollow)

	if flavor.Taste.Mean <= healthy.Taste.Mean {
		t.Fatalf("AI 味病历书均值必须高于基准：flavor=%v healthy=%v", flavor.Taste.Mean, healthy.Taste.Mean)
	}
	if hollow.Foreshadow.Recall != 0 {
		t.Fatalf("断链书回收率必须为 0: %v", hollow.Foreshadow.Recall)
	}
	if healthy.Foreshadow.Recall != 0.8 {
		t.Fatalf("基准书回收率应 0.8: %v", healthy.Foreshadow.Recall)
	}
	if hollow.Quality.S3 < evalCorpusChapters {
		t.Fatalf("断链书每章超长段应各记 S3（≥30）: %v", hollow.Quality.S3)
	}
	if healthy.Quality.S3 > hollow.Quality.S3-evalCorpusChapters {
		t.Fatalf("基准书 S3 应显著少于断链书: %v vs %v", healthy.Quality.S3, hollow.Quality.S3)
	}
}

// TestEvalCorpusBaselineCompareScale 基线/对比在 30 章规模：基线后无变化对比
// 无 worse 项；单章病变（第 15 章换重 AI 腔）→ AI 味均值 worse 方向。
func TestEvalCorpusBaselineCompareScale(t *testing.T) {
	a := newGateEmptyApp()
	pm := buildEvalCorpusBook(t, a, "healthy", evalHealthyChapter, evalCorpusForeshadows(10, 8))

	if _, err := a.NovelEvalBaselineSet(); err != nil {
		t.Fatalf("设基线: %v", err)
	}
	res, err := a.NovelEvalCompare()
	if err != nil {
		t.Fatalf("无变化对比: %v", err)
	}
	if res["stale"] != false {
		t.Fatalf("同 prompts 不应 stale: %v", res["stale"])
	}
	for _, it := range res["items"].([]map[string]interface{}) {
		if it["dir"] == "worse" {
			t.Fatalf("无变化不得出现 worse 项: %v", it)
		}
	}

	// 单章病变：第 15 章换重 AI 腔 → 均值上升=worse
	if err := pm.WriteChapter(15, evalFlavorChapter(15)); err != nil {
		t.Fatal(err)
	}
	res2, err := a.NovelEvalCompare()
	if err != nil {
		t.Fatalf("病变对比: %v", err)
	}
	found := false
	for _, it := range res2["items"].([]map[string]interface{}) {
		if it["metric"] == "AI 味均值" && it["dir"] == "worse" {
			found = true
		}
	}
	if !found {
		t.Fatalf("第 15 章病变应使 AI 味均值 worse: %v", res2["items"])
	}
}
