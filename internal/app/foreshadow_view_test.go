package app

import (
	"testing"

	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/types"
)

// TestGetForeshadows_UrgencyProjection t1-P4：GetForeshadows 附带运行时紧急度
// 投影（A5 不落库；D7 阈值后端算）——超期条目 level=3，已回收条目不带 urgency 键。
func TestGetForeshadows_UrgencyProjection(t *testing.T) {
	a := newFingerprintTestApp(t)
	for i := 1; i <= 3; i++ {
		mustWriteChapter(t, a, i, "样文。")
	}
	overdue := manualItem("od_1", "001.md")
	overdue.TargetResolveIn = "002.md" // 计划 002 < 当前 3 → 超期
	near := manualItem("near_1", "001.md")
	near.TargetResolveIn = "004.md" // 还有 1 章 → 需关注（remind 缺省 5）
	noplan := manualItem("np_1", "001.md")
	done := manualItem("done_1", "001.md")
	done.Status = types.ForeshadowRevealed
	done.RevealedIn = "002.md"
	mustSaveForeshadows(t, a, []types.Foreshadow{overdue, near, noplan, done})

	res := a.GetForeshadows()
	if res == nil {
		t.Fatalf("应返回载荷")
	}
	if cur, ok := res["currentChapter"].(int); !ok || cur != 3 {
		t.Fatalf("currentChapter 应自动=3: %v", res["currentChapter"])
	}
	items, ok := res["items"].([]map[string]interface{})
	if !ok || len(items) != 4 {
		t.Fatalf("应 4 条: %v", res["items"])
	}
	byID := map[string]map[string]interface{}{}
	for _, it := range items {
		byID[it["id"].(string)] = it
	}
	urgencyOf := func(id string) *types.ForeshadowUrgency {
		u, _ := byID[id]["urgency"].(*types.ForeshadowUrgency)
		return u
	}
	// 超期：level 3 + overdueChapters 1 + resolveStatus=overdue
	if u := urgencyOf("od_1"); u == nil || u.Level != 3 || u.OverdueChapters != 1 ||
		u.ResolveStatus != types.ResolveOverdue {
		t.Fatalf("超期投影不对: %v", byID["od_1"]["urgency"])
	}
	// 剩余 1 章 ≤ ForeshadowUrgentRemaining(2) → level 2 急需回收（时机仍是 not_yet）
	if u := urgencyOf("near_1"); u == nil || u.Level != 2 || u.OverdueChapters != 0 ||
		u.ResolveStatus != types.ResolveNotYet {
		t.Fatalf("急需投影不对: %v", byID["near_1"]["urgency"])
	}
	// 无计划：存活但 level 0，仍投影（口径：非回收态都带）
	if u := urgencyOf("np_1"); u == nil || u.Level != 0 || u.ResolveStatus != types.ResolveNoPlan {
		t.Fatalf("无计划投影不对: %v", byID["np_1"]["urgency"])
	}
	// 已回收：不带 urgency 键（回收态无压力）
	if _, has := byID["done_1"]["urgency"]; has {
		t.Fatalf("已回收条目不应带 urgency: %v", byID["done_1"])
	}
}

// TestGetLastForeshadowSync t1-P4：SyncResult 上绑定面——分析同步后可取，
// 尚未分析时显式报错（D3 跳过原因可见不静默）。
func TestGetLastForeshadowSync(t *testing.T) {
	a := newFingerprintTestApp(t)
	a.analysisAgent = analysis.New(nil, a.getPM(), &config.Config{}, nil)

	if _, err := a.GetLastForeshadowSync(); err == nil {
		t.Fatalf("尚未分析应显式报错")
	}

	// 直接驱动一轮同步（不调 LLM）：1 回收（内容匹配）+ 1 跳过（无效引用）
	mustSaveForeshadows(t, a, []types.Foreshadow{
		{ID: "manual_1", Category: "plot", Description: "祠堂地砖下的铜匣",
			PlantedIn: "001.md", Status: types.ForeshadowPlanted},
	})
	res := a.analysisAgent.SyncForeshadows(2, []types.ForeshadowHit{
		{Type: "resolved", Content: "祠堂地砖下的铜匣"},
		{Type: "resolved", ReferenceStableID: "gone", Content: "任意"},
	})

	out, err := a.GetLastForeshadowSync()
	if err != nil {
		t.Fatalf("同步后应可取: %v", err)
	}
	if out["resolvedCount"].(int) != 1 || out["matchedByContent"].(int) != 1 {
		t.Fatalf("同步计数不对: %v", out)
	}
	reasons, _ := out["skippedReasons"].([]analysis.SkipReason)
	if len(reasons) != 1 || reasons[0].Kind != "invalid_reference" {
		t.Fatalf("跳过原因应可见（D3）: %v", out["skippedReasons"])
	}
	if res.ResolvedCount != 1 {
		t.Fatalf("同步返回值应一致: %+v", res)
	}
}
