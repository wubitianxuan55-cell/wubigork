package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// mustSeedForeshadowsWithSource 写入带来源标记的登记表（失败即终止）。
func mustSeedForeshadowsWithSource(t *testing.T, a *App, items ...types.Foreshadow) {
	t.Helper()
	mustSaveForeshadows(t, a, items)
}

// readForeshadowItemsForTest 读回登记表条目。
func readForeshadowItemsForTest(t *testing.T, a *App) []types.Foreshadow {
	t.Helper()
	pm := a.getPM()
	ff, err := pm.ReadForeshadows()
	if err != nil {
		t.Fatalf("读伏笔文件: %v", err)
	}
	return ff.Items
}

func analysisItem(id, plantedIn string) types.Foreshadow {
	return types.Foreshadow{
		ID: id, Category: "mystery", Title: id, Description: "分析条目 " + id,
		PlantedIn: plantedIn, Status: types.ForeshadowPlanted,
		SourceType: types.ForeshadowSourceAnalysis, SourceMemoryID: id,
	}
}

func manualItem(id, plantedIn string) types.Foreshadow {
	return types.Foreshadow{
		ID: id, Category: "plot", Description: "手工条目 " + id,
		PlantedIn: plantedIn, Status: types.ForeshadowPlanted, IsLongTerm: true,
	}
}

// TestDeleteChapterForeshadows_SourceGuard t1-P3.1：默认只删分析来源，
// 手动条目永不被批量删除；onlyAnalysisSource=false 才整章全删。
func TestDeleteChapterForeshadows_SourceGuard(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustSeedForeshadowsWithSource(t, a,
		analysisItem("an_plant", "002.md"),  // 埋入本章（分析）
		analysisItem("an_reveal", "001.md"), // 埋在他章
		manualItem("ma_plant", "002.md"),    // 埋入本章（手动）
		manualItem("ma_reveal", "001.md"),   // 埋在他章
	)
	// an_reveal / ma_reveal 补回收章 002 → 命中 RevealedIn 判据
	items := readForeshadowItemsForTest(t, a)
	for i := range items {
		if strings.HasPrefix(items[i].ID, "an_reveal") || strings.HasPrefix(items[i].ID, "ma_reveal") {
			items[i].RevealedIn = "002.md"
			items[i].Status = types.ForeshadowRevealed
		}
	}
	mustSaveForeshadows(t, a, items)

	res, err := a.DeleteChapterForeshadows("002.md", true)
	if err != nil {
		t.Fatalf("清理失败: %v", err)
	}
	if res.Deleted != 2 || res.RolledBack != 0 || res.ResetManual != 0 {
		t.Fatalf("应只删 2 条分析条目（埋入∨回收本章）: %+v", res)
	}
	left := readForeshadowItemsForTest(t, a)
	if len(left) != 2 {
		t.Fatalf("手动条目应全保留，剩 %d: %+v", len(left), left)
	}
	for _, it := range left {
		if !foreshadowIsManual(it) {
			t.Fatalf("分析条目漏删: %+v", it)
		}
	}

	// 整章全删（false）：手动条目也删
	mustSeedForeshadowsWithSource(t, a, analysisItem("an_plant", "002.md"), manualItem("ma_plant", "002.md"))
	res, err = a.DeleteChapterForeshadows("002.md", false)
	if err != nil {
		t.Fatalf("整章清理失败: %v", err)
	}
	if res.Deleted != 2 || len(readForeshadowItemsForTest(t, a)) != 0 {
		t.Fatalf("onlyAnalysisSource=false 应清空本章关联: %+v", res)
	}
}

// TestCleanChapterAnalysisForeshadows t1-P3.1：与删除入口语义严格区分——
// 只删「分析∧埋入本章」，本章回收的条目回退 planted 清回收痕迹。
func TestCleanChapterAnalysisForeshadows(t *testing.T) {
	a := newFingerprintTestApp(t)
	resolvedManual := manualItem("ma_res", "001.md")
	resolvedManual.Status = types.ForeshadowRevealed
	resolvedManual.RevealedIn = "003.md"
	resolvedManual.ResolvedAt = "2026-09-14T00:00:00Z"
	resolvedManual.ResolutionText = "铜匣开启"

	partialManual := manualItem("ma_part", "001.md")
	partialManual.Status = types.ForeshadowPartial
	partialManual.RevealedIn = "003.md"

	rollbackAnalysis := analysisItem("an_rb", "001.md")
	rollbackAnalysis.Status = types.ForeshadowRevealed
	rollbackAnalysis.RevealedIn = "003.md"

	mustSeedForeshadowsWithSource(t, a,
		analysisItem("an_plant", "003.md"), // 分析∧埋入本章 → 删
		resolvedManual,                     // 手动∧本章回收 → 回退 planted
		partialManual,                      // 部分回收∧本章 → 回退 planted
		rollbackAnalysis,                   // 分析∧埋他章∨回收本章 → 回退（不删）
		manualItem("ma_plant", "003.md"),   // 手动∧埋入本章 → 原样保留
	)

	res, err := a.CleanChapterAnalysisForeshadows("003.md")
	if err != nil {
		t.Fatalf("清理失败: %v", err)
	}
	if res.Deleted != 1 || res.RolledBack != 3 {
		t.Fatalf("应删 1 回退 3: %+v", res)
	}
	left := readForeshadowItemsForTest(t, a)
	if len(left) != 4 {
		t.Fatalf("应剩 4 条: %+v", left)
	}
	got := findByTestID(left, "ma_res")
	if got.Status != types.ForeshadowPlanted || got.RevealedIn != "" || got.ResolvedAt != "" || got.ResolutionText != "" {
		t.Fatalf("回收条目应回退 planted 并清空回收痕迹: %+v", got)
	}
	if got.PlantedIn != "001.md" || !got.IsLongTerm {
		t.Fatalf("回退不得改写埋入章/长线标记: %+v", got)
	}
	if got := findByTestID(left, "ma_part"); got.Status != types.ForeshadowPlanted || got.RevealedIn != "" {
		t.Fatalf("部分回收也应回退: %+v", got)
	}
	if got := findByTestID(left, "an_rb"); got.Status != types.ForeshadowPlanted || got.RevealedIn != "" {
		t.Fatalf("分析条目回收本章同样回退: %+v", got)
	}
	if got := findByTestID(left, "ma_plant"); got.Status != types.ForeshadowPlanted || got.PlantedIn != "003.md" {
		t.Fatalf("手动埋入本章条目应原样保留: %+v", got)
	}
}

// TestClearProjectForeshadowsForReset t1-P3.1：分析来源整批删，手动只重置不删除。
func TestClearProjectForeshadowsForReset(t *testing.T) {
	a := newFingerprintTestApp(t)
	manual := manualItem("ma_1", "004.md")
	manual.RevealedIn = "005.md"
	manual.TargetResolveIn = "006.md"
	manual.PlantedAt = "2026-09-01T00:00:00Z"
	manual.ResolvedAt = "2026-09-02T00:00:00Z"
	mustSeedForeshadowsWithSource(t, a,
		analysisItem("an_1", "001.md"),
		analysisItem("an_2", "002.md"),
		manual,
	)

	res, err := a.ClearProjectForeshadowsForReset()
	if err != nil {
		t.Fatalf("重置失败: %v", err)
	}
	if res.Deleted != 2 || res.ResetManual != 1 {
		t.Fatalf("应删 2 分析 + 重置 1 手动: %+v", res)
	}
	left := readForeshadowItemsForTest(t, a)
	if len(left) != 1 {
		t.Fatalf("手动条目记录本身应保留: %+v", left)
	}
	got := left[0]
	if got.Status != types.ForeshadowPending {
		t.Fatalf("手动条目应重置为 pending: %+v", got)
	}
	if got.PlantedIn != "" || got.RevealedIn != "" || got.TargetResolveIn != "" || got.PlantedAt != "" || got.ResolvedAt != "" {
		t.Fatalf("章节关联与时间戳应清空: %+v", got)
	}
	if got.Description != "手工条目 ma_1" || !got.IsLongTerm {
		t.Fatalf("内容与长线标记应保留: %+v", got)
	}
}

// TestGetForeshadowStats t1-P3.3：分状态计数 + 别名归一 + 超期数（ClassifyResolve 唯一入口）。
func TestGetForeshadowStats(t *testing.T) {
	a := newFingerprintTestApp(t)
	for i := 1; i <= 3; i++ {
		mustWriteChapter(t, a, i, "样文。")
	}
	aliasResolved := manualItem("ma_alias", "001.md")
	aliasResolved.Status = types.ForeshadowResolved // 外部数据别名 → 归一并入 resolved
	overdue := manualItem("ma_overdue", "001.md")
	overdue.Status = types.ForeshadowPlanted
	overdue.TargetResolveIn = "002.md"
	ontime := manualItem("ma_ontime", "001.md")
	ontime.TargetResolveIn = "003.md"
	partialOverdue := manualItem("ma_partial", "001.md")
	partialOverdue.Status = types.ForeshadowPartial
	partialOverdue.TargetResolveIn = "001.md"
	revealed := manualItem("ma_done", "001.md")
	revealed.Status = types.ForeshadowRevealed
	revealed.RevealedIn = "002.md"
	hinted := manualItem("ma_hint", "001.md") // hinted 无计划（存活，但缺计划不判超期）
	hinted.Status = types.ForeshadowHinted
	mustSeedForeshadowsWithSource(t, a,
		types.Foreshadow{ID: "pd_1", Description: "规划", Status: types.ForeshadowPending},
		overdue, ontime, partialOverdue, revealed, aliasResolved,
		hinted,
		types.Foreshadow{ID: "ab_1", Description: "废弃", Status: types.ForeshadowAbandoned, PlantedIn: "001.md"},
	)

	stats, err := a.GetForeshadowStats(0) // 0=自动按已写章节数（3）
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if stats.CurrentChapter != 3 {
		t.Fatalf("应自动取已写章节数 3: %+v", stats)
	}
	if stats.Pending != 1 || stats.Planted != 2 || stats.Hinted != 1 || stats.Resolved != 2 ||
		stats.PartiallyResolved != 1 || stats.Abandoned != 1 {
		t.Fatalf("分状态计数不对（resolved 别名应并入）: %+v", stats)
	}
	if stats.Total != 8 {
		t.Fatalf("Total 应为分桶之和 8: %+v", stats)
	}
	if stats.LongTermCount < 5 {
		t.Fatalf("长线计数不对: %+v", stats)
	}
	// 超期：ma_overdue（计划 002 < 3）与 ma_partial（计划 001 < 3）→ 2；
	// ma_ontime 计划 == 3 不算超期；hinted 无计划不算。
	if stats.OverdueCount != 2 {
		t.Fatalf("超期数应为 2: %+v", stats)
	}

	stats, err = a.GetForeshadowStats(5) // 显式当前章 5：计划 003/002/001 全部 <5 → 3 超期
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if stats.CurrentChapter != 5 || stats.OverdueCount != 3 {
		t.Fatalf("显式章号下超期口径不对: %+v", stats)
	}
}

// TestLintForeshadowItems_OverdueUnplanned t1-P3.4：⑥超期 ⑦无计划两新码，
// 既有五类不回归。
func TestLintForeshadowItems_OverdueUnplanned(t *testing.T) {
	// ── 超期（当前 5 章）──
	overdue := manualItem("od_1", "001.md")
	overdue.IsLongTerm = false
	overdue.TargetResolveIn = "002.md"
	ontime := manualItem("od_2", "001.md")
	ontime.TargetResolveIn = "005.md"
	done := manualItem("done_1", "001.md")
	done.Status = types.ForeshadowRevealed
	done.RevealedIn = "002.md"
	done.TargetResolveIn = "001.md" // 计划早于当前，但已回收 → 不报超期
	findings := lintForeshadowItems([]types.Foreshadow{overdue, ontime, done}, 5)

	if f := findLintCode(findings, "od_1", "overdue"); f == nil || f.Severity != "medium" ||
		!strings.Contains(f.Message, "超期 3 章") {
		t.Fatalf("超期检查失效: %+v", f)
	}
	if f := findLintCode(findings, "od_2", "overdue"); f != nil {
		t.Fatalf("计划章==当前章不应报超期: %+v", f)
	}
	if f := findLintCode(findings, "done_1", "overdue"); f != nil {
		t.Fatalf("已回收条目不应报超期: %+v", f)
	}

	// ── 无计划（当前 12 章，埋入 11 章达阈值）──
	unplanned := manualItem("up_1", "001.md") // 无计划 + 埋入 11 章
	unplanned.IsLongTerm = false
	longterm := manualItem("up_2", "001.md") // 长线豁免 unplanned/stale
	pending := types.Foreshadow{ID: "pd_1", Description: "规划", Status: types.ForeshadowPending, PlantedIn: "001.md"}
	findings = lintForeshadowItems([]types.Foreshadow{unplanned, longterm, pending}, 12)

	if f := findLintCode(findings, "up_1", "unplanned"); f == nil || f.Severity != "low" ||
		!strings.Contains(f.Message, "未填计划回收章") {
		t.Fatalf("无计划检查失效: %+v", f)
	}
	// up_1 同时满足 stale 与 unplanned（建议动作不同，两码并存）
	if f := findLintCode(findings, "up_1", "stale"); f == nil {
		t.Fatalf("既有 stale 不应回归: %+v", f)
	}
	for _, code := range []string{"unplanned", "stale"} {
		if f := findLintCode(findings, "up_2", code); f != nil {
			t.Fatalf("长线条目应豁免 %s: %+v", code, f)
		}
	}
	if f := findLintCode(findings, "pd_1", "unplanned"); f != nil {
		t.Fatalf("pending 未埋入不应报无计划: %+v", f)
	}

	// ── 既有五类在原口径上不回归 ──
	stal := manualItem("stal_x", "001.md")
	stal.IsLongTerm = false
	old := []types.Foreshadow{
		stal,
		{ID: "ord_x", Description: "怀表", PlantedIn: "003.md", RevealedIn: "001.md", Status: types.ForeshadowRevealed},
	}
	findings = lintForeshadowItems(old, 12)
	if findLintCode(findings, "stal_x", "stale") == nil || findLintCode(findings, "ord_x", "ordering") == nil {
		t.Fatalf("既有检查回归: %+v", findings)
	}
}

// TestForeshadowCleanupStatsWireShape 形状锁：清理/统计负载 marshal 键 camelCase。
func TestForeshadowCleanupStatsWireShape(t *testing.T) {
	cleanup, err := json.Marshal(ForeshadowCleanupResult{Deleted: 1, RolledBack: 2, ResetManual: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"deleted", "rolledBack", "resetManual"} {
		if !strings.Contains(string(cleanup), `"`+key+`"`) {
			t.Fatalf("清理结果缺键 %q: %s", key, cleanup)
		}
	}
	stats, err := json.Marshal(ForeshadowStatsReport{Total: 9, PartiallyResolved: 1, LongTermCount: 2, OverdueCount: 3, CurrentChapter: 4})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"total", "pending", "planted", "hinted", "resolved", "partiallyResolved", "abandoned", "longTermCount", "overdueCount", "currentChapter"} {
		if !strings.Contains(string(stats), `"`+key+`"`) {
			t.Fatalf("统计缺键 %q: %s", key, stats)
		}
	}
	if strings.Contains(string(stats), `"PartiallyResolved"`) || strings.Contains(string(cleanup), `"RolledBack"`) {
		t.Fatalf("出现 PascalCase 键: %s %s", cleanup, stats)
	}
}

// findByTestID 按 ID 查条目。
func findByTestID(items []types.Foreshadow, id string) types.Foreshadow {
	for _, it := range items {
		if it.ID == id {
			return it
		}
	}
	return types.Foreshadow{}
}
