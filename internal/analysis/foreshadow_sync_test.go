package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newSyncTestAgent 构造带临时项目的分析 Agent（syncForeshadows 不依赖 LLM，client 可为 nil）。
func newSyncTestAgent(t *testing.T) *Agent {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	return New(nil, pm, &config.Config{}, nil)
}

// seedForeshadows 写入初始伏笔文件（失败即终止）。
func seedForeshadows(t *testing.T, a *Agent, items ...types.Foreshadow) {
	t.Helper()
	if err := a.pm.WriteForeshadows(&types.ForeshadowFile{Items: items}); err != nil {
		t.Fatalf("写伏笔文件: %v", err)
	}
}

// readForeshadowItems 读回伏笔条目（失败即终止）。
func readForeshadowItems(t *testing.T, a *Agent) []types.Foreshadow {
	t.Helper()
	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		t.Fatalf("读伏笔文件: %v", err)
	}
	return ff.Items
}

// findByID 按 ID 查找条目。
func findByID(items []types.Foreshadow, id string) *types.Foreshadow {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

// skipKinds 汇总一轮同步的跳过类型。
func skipKinds(res SyncResult) map[string]int {
	m := map[string]int{}
	for _, s := range res.SkippedReasons {
		m[s.Kind]++
	}
	return m
}

// TestSyncForeshadows_MergesByIDAndKeepsManual 覆盖核心合并语义：
// 回收按 精确ID/内容兜底 推进既有条目（手工条目同样可被推进且字段不被冲掉）、
// planted 追加新条目带 v2 字段。
func TestSyncForeshadows_MergesByIDAndKeepsManual(t *testing.T) {
	a := newSyncTestAgent(t)
	manual := types.Foreshadow{
		ID: "manual_1725000000000", Category: "plot", Description: "神秘铜匣的钥匙",
		PlantedIn: "001.md", Status: types.ForeshadowPlanted, IsLongTerm: true,
	}
	aiA := types.Foreshadow{
		ID: "plot_001_abc", Category: "character", Description: "主角左臂旧伤",
		PlantedIn: "001.md", Status: types.ForeshadowPlanted,
	}
	seedForeshadows(t, a, manual, aiA)

	res := a.syncForeshadows(2, []types.ForeshadowHit{
		// 新伏笔：第 2 章新埋设（带预估回收章与评分）
		{Type: "planted", Category: "item", Title: "星门钥匙", Content: "星门在月圆之夜开启",
			Strength: 8, EstimateResolve: 12},
		// 既有 AI 条目：按精确 ID 回收
		{Type: "resolved", Title: "左臂旧伤", Content: "旧伤迸裂", ReferenceStableID: "plot_001_abc"},
		// 手工条目：无 ID，按内容兜底匹配被回收（期望：状态推进但 ID/长线标记保留）
		{Type: "resolved", Title: "铜匣钥匙回收", Content: "神秘铜匣的钥匙"},
	})

	items := readForeshadowItems(t, a)
	if len(items) != 3 {
		t.Fatalf("应有 3 条伏笔（2 既有 + 1 新增），实际 %d: %+v", len(items), items)
	}
	if res.ResolvedCount != 2 || res.CreatedCount != 1 || res.MatchedByContent != 1 {
		t.Fatalf("同步计数不符: %+v", res)
	}

	// 手工条目：ID 与 is_long_term 原样保留，仅状态被内容匹配推进到 revealed/002.md
	got := findByID(items, "manual_1725000000000")
	if got == nil {
		t.Fatalf("手工条目被 syncForeshadows 冲掉: %+v", items)
	}
	if got.Status != types.ForeshadowRevealed || got.RevealedIn != "002.md" {
		t.Fatalf("手工条目应被内容匹配推进到 revealed/002.md: %+v", got)
	}
	if !got.IsLongTerm || got.PlantedIn != "001.md" || got.Category != "plot" {
		t.Fatalf("手工条目其他字段不应被改写: %+v", got)
	}

	// 既有 AI 条目：按精确 ID 回收
	got = findByID(items, "plot_001_abc")
	if got == nil || got.Status != types.ForeshadowRevealed || got.RevealedIn != "002.md" {
		t.Fatalf("既有 AI 条目应按 ID 回收: %+v", got)
	}

	// 新增条目：planted 在 002.md，稳定 ID + v2 字段齐备
	stableID := GenerateStableID("item", "002.md", "星门在月圆之夜开启")
	got = findByID(items, stableID)
	if got == nil {
		t.Fatalf("新增伏笔缺失: %+v", items)
	}
	if got.Status != types.ForeshadowPlanted || got.PlantedIn != "002.md" {
		t.Fatalf("新增伏笔状态/埋入章错误: %+v", got)
	}
	if got.SourceType != types.ForeshadowSourceAnalysis || got.SourceMemoryID != stableID {
		t.Fatalf("新增伏笔来源字段缺失: %+v", got)
	}
	if got.Title != "星门钥匙" || got.TargetResolveIn != "012.md" {
		t.Fatalf("标题/计划回收章映射错误: %+v", got)
	}
	if got.Importance != 0.8 || got.Strength != 8 {
		t.Fatalf("Importance 应=min(8/10,1)=0.8: %+v", got)
	}
	if got.AutoRemind == nil || !*got.AutoRemind || got.RemindBeforeChapters != types.ForeshadowRemindBeforeDefault {
		t.Fatalf("注入控制缺省语义错误: %+v", got)
	}
	if got.CreatedAt == "" || got.PlantedAt == "" {
		t.Fatalf("时间戳缺失: %+v", got)
	}
}

// TestSyncForeshadows_ReplayNoDuplicateNoReset 同章同内容重复分析（同一 planted
// 动作重放）不应产生重复条目（防重第一道：稳定 ID），也不应重置既有条目状态；
// 更新路径不占每章新建额度。
func TestSyncForeshadows_ReplayNoDuplicateNoReset(t *testing.T) {
	a := newSyncTestAgent(t)
	stableID := GenerateStableID("plot", "003.md", "铜匣出现")
	hit := types.ForeshadowHit{Type: "planted", Category: "plot", Content: "铜匣出现"}

	res1 := a.syncForeshadows(3, []types.ForeshadowHit{hit})
	if res1.CreatedCount != 1 {
		t.Fatalf("首次同步应新建 1 条: %+v", res1)
	}
	// 模拟已有推进：手工把该条目标成 hinted，再次重放同一 planted 动作
	items := readForeshadowItems(t, a)
	items[0].Status = types.ForeshadowHinted
	seedForeshadows(t, a, items...)

	res2 := a.syncForeshadows(3, []types.ForeshadowHit{hit})

	items = readForeshadowItems(t, a)
	if len(items) != 1 {
		t.Fatalf("重放 planted 不应产生重复条目，实际 %d: %+v", len(items), items)
	}
	if items[0].ID != stableID || items[0].Status != types.ForeshadowHinted {
		t.Fatalf("重放不应重置既有条目: %+v", items[0])
	}
	if res2.CreatedCount != 0 || res2.PlantedCount != 0 {
		t.Fatalf("重放更新路径不应计入新建: %+v", res2)
	}
}

// TestSyncForeshadows_CorruptFileNotOverwritten 伏笔文件损坏（读取失败）时放弃同步，
// 不得用空数据覆盖既有文件。
func TestSyncForeshadows_CorruptFileNotOverwritten(t *testing.T) {
	a := newSyncTestAgent(t)
	path := filepath.Join(a.pm.Dir, "foreshadows.json")
	corrupt := []byte("{ not valid json !!")
	if err := os.WriteFile(path, corrupt, 0644); err != nil {
		t.Fatalf("写损坏文件: %v", err)
	}

	res := a.syncForeshadows(1, []types.ForeshadowHit{
		{Type: "planted", Category: "plot", Content: "不应落盘"},
	})

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读回损坏文件: %v", err)
	}
	if string(after) != string(corrupt) {
		t.Fatalf("损坏文件应原样保留（不被覆盖）: %q", after)
	}
	if len(res.Errors) == 0 {
		t.Fatalf("同步放弃应记入 Errors（D3 不静默）: %+v", res)
	}
}

// TestSyncForeshadows_FreshProject 新项目（文件不存在）首次同步正常建档。
func TestSyncForeshadows_FreshProject(t *testing.T) {
	a := newSyncTestAgent(t)
	a.syncForeshadows(1, []types.ForeshadowHit{
		{Type: "planted", Category: "identity", Content: "主角的旧玉佩"},
	})

	items := readForeshadowItems(t, a)
	if len(items) != 1 || items[0].Status != types.ForeshadowPlanted || items[0].PlantedIn != "001.md" {
		t.Fatalf("新项目首次同步应登记 1 条 planted: %+v", items)
	}
}

// TestSyncForeshadows_InvalidRefNoFallback t1-P2 关键回归：reference_stable_id
// 无效时禁止回落内容匹配（spec §1.2 原则 3：宁可漏收不可错收）——即使正文
// 与既有条目完全一致也不得推进。
func TestSyncForeshadows_InvalidRefNoFallback(t *testing.T) {
	a := newSyncTestAgent(t)
	seedForeshadows(t, a, types.Foreshadow{
		ID: "plot_001_ok", Category: "plot", Description: "祠堂地砖下的铜匣",
		PlantedIn: "001.md", Status: types.ForeshadowPlanted,
	})

	res := a.syncForeshadows(2, []types.ForeshadowHit{
		{Type: "resolved", Title: "铜匣", Content: "祠堂地砖下的铜匣", ReferenceStableID: "plot_999_gone"},
	})

	items := readForeshadowItems(t, a)
	if got := findByID(items, "plot_001_ok"); got == nil || got.Status != types.ForeshadowPlanted {
		t.Fatalf("无效引用不得回落内容匹配，条目应保持 planted: %+v", got)
	}
	kinds := skipKinds(res)
	if kinds[skipInvalidReference] != 1 {
		t.Fatalf("应记一次 invalid_reference 跳过: %+v", res.SkippedReasons)
	}
	if res.ResolvedCount != 0 {
		t.Fatalf("不应有回收: %+v", res)
	}
}

// TestSyncForeshadows_ResolvedNoMatchNoCreate 回收无匹配不建新记录
// （spec §1.2 原则 1：只有埋入会创建记录）。
func TestSyncForeshadows_ResolvedNoMatchNoCreate(t *testing.T) {
	a := newSyncTestAgent(t)
	seedForeshadows(t, a, types.Foreshadow{
		ID: "plot_001_a", Description: "左臂旧伤", PlantedIn: "001.md", Status: types.ForeshadowPlanted,
	})

	res := a.syncForeshadows(2, []types.ForeshadowHit{
		{Type: "resolved", Title: "毫不相干", Content: "北方边境爆发战乱，粮草告急"},
	})

	items := readForeshadowItems(t, a)
	if len(items) != 1 {
		t.Fatalf("resolved 无匹配不得新建记录，实际 %d: %+v", len(items), items)
	}
	kinds := skipKinds(res)
	if kinds[skipNoMatch] != 1 || res.SkippedResolveCount != 1 {
		t.Fatalf("应记 no_match 跳过: %+v", res)
	}
}

// TestSyncForeshadows_AlreadyResolvedNoReset 已回收条目不重置（不因重复分析改写
// 回收章），跳过原因如实区分本章/历史。
func TestSyncForeshadows_AlreadyResolvedNoReset(t *testing.T) {
	a := newSyncTestAgent(t)
	seedForeshadows(t, a, types.Foreshadow{
		ID: "plot_001_a", Description: "左臂旧伤", PlantedIn: "001.md",
		Status: types.ForeshadowRevealed, RevealedIn: "001.md",
	})

	res := a.syncForeshadows(5, []types.ForeshadowHit{
		{Type: "resolved", ReferenceStableID: "plot_001_a", Content: "旧伤迸裂"},
	})

	items := readForeshadowItems(t, a)
	got := findByID(items, "plot_001_a")
	if got == nil || got.RevealedIn != "001.md" {
		t.Fatalf("已回收条目不得被重写回收章: %+v", got)
	}
	kinds := skipKinds(res)
	if kinds[skipAlreadyResolved] != 1 {
		t.Fatalf("应记 already_resolved 跳过: %+v", res.SkippedReasons)
	}
	if res.SkippedReasons[0].Message == "已在本章回收过" {
		t.Fatalf("第 5 章重复回收第 1 章条目应报历史章而非本章: %+v", res.SkippedReasons[0])
	}
}

// TestSyncForeshadows_NotPlantedSkip pending（未埋入）条目不可被回收。
func TestSyncForeshadows_NotPlantedSkip(t *testing.T) {
	a := newSyncTestAgent(t)
	seedForeshadows(t, a, types.Foreshadow{
		ID: "plot_001_p", Description: "计划中的预言", PlantedIn: "001.md", Status: types.ForeshadowPending,
	})

	res := a.syncForeshadows(2, []types.ForeshadowHit{
		{Type: "resolved", ReferenceStableID: "plot_001_p", Content: "预言应验"},
	})

	got := findByID(readForeshadowItems(t, a), "plot_001_p")
	if got.Status != types.ForeshadowPending {
		t.Fatalf("pending 条目不得被回收: %+v", got)
	}
	if skipKinds(res)[skipNotPlanted] != 1 {
		t.Fatalf("应记 not_planted 跳过: %+v", res.SkippedReasons)
	}
}

// TestSyncForeshadows_HintedResolvable D15 口径：hinted 视同可回收状态，
// 可被精确 ID 推进到 revealed。
func TestSyncForeshadows_HintedResolvable(t *testing.T) {
	a := newSyncTestAgent(t)
	seedForeshadows(t, a, types.Foreshadow{
		ID: "plot_001_h", Description: "左臂旧伤", PlantedIn: "001.md", Status: types.ForeshadowHinted,
	})

	res := a.syncForeshadows(2, []types.ForeshadowHit{
		{Type: "resolved", ReferenceStableID: "plot_001_h", Content: "旧伤迸裂"},
	})

	got := findByID(readForeshadowItems(t, a), "plot_001_h")
	if got == nil || got.Status != types.ForeshadowRevealed || got.RevealedIn != "002.md" {
		t.Fatalf("hinted 条目应可被回收: %+v", got)
	}
	if res.ResolvedCount != 1 {
		t.Fatalf("回收计数错误: %+v", res)
	}
}

// TestSyncForeshadows_CapFivePerChapter 每章新建上限 5 只约束新建（更新不占额度），
// 超出如实记 limit_reached 不静默丢弃。
func TestSyncForeshadows_CapFivePerChapter(t *testing.T) {
	a := newSyncTestAgent(t)
	hits := make([]types.ForeshadowHit, 0, 6)
	for i := 1; i <= 6; i++ {
		hits = append(hits, types.ForeshadowHit{
			Type: "planted", Category: "mystery",
			Title: fmt.Sprintf("%d号信物", i), Content: fmt.Sprintf("第%d枚信物在火场出现", i),
		})
	}
	res := a.syncForeshadows(4, hits)

	items := readForeshadowItems(t, a)
	if len(items) != types.ForeshadowMaxNewPerChapter {
		t.Fatalf("每章新建应封顶 %d，实际 %d", types.ForeshadowMaxNewPerChapter, len(items))
	}
	if res.CreatedCount != types.ForeshadowMaxNewPerChapter {
		t.Fatalf("创建计数应封顶: %+v", res)
	}
	if skipKinds(res)[skipLimitReached] != 1 {
		t.Fatalf("第 6 条应记 limit_reached: %+v", res.SkippedReasons)
	}

	// 更新不占额度：同章重放 1 条既有 + 再来 5 条全新 → 5 条新建
	replay := []types.ForeshadowHit{hits[0]}
	for i := 7; i <= 11; i++ {
		replay = append(replay, types.ForeshadowHit{
			Type: "planted", Category: "mystery",
			Title: fmt.Sprintf("后加%d", i), Content: fmt.Sprintf("后续补充的第%d枚信物", i),
		})
	}
	a.syncForeshadows(5, replay) // 注意：新章（005.md）重置额度，仅验证重放零新建
	items = readForeshadowItems(t, a)
	// 004 章 5 条 + 005 章重放命中 004 既有条目（更新不新建）+ 5 条新章新建 = 11
	if len(items) != 5+5 {
		t.Fatalf("重放应走更新不新建（004 章 5 条 + 005 章 5 条），实际 %d", len(items))
	}
}

// TestSyncForeshadows_EmptyContentSkip 埋入内容为空如实跳过。
func TestSyncForeshadows_EmptyContentSkip(t *testing.T) {
	a := newSyncTestAgent(t)
	res := a.syncForeshadows(1, []types.ForeshadowHit{
		{Type: "planted", Category: "mystery", Content: "   "},
	})
	if len(readForeshadowItems(t, a)) != 0 {
		t.Fatalf("空内容不得建档")
	}
	if skipKinds(res)[skipEmptyContent] != 1 {
		t.Fatalf("应记 empty_content 跳过: %+v", res.SkippedReasons)
	}
}

// TestSyncForeshadows_PlantedDedupByTitleGuard 防重第二道：同章同题的分析条目
// （内容已改写、稳定 ID 不同）命中更新而非重复建档。
func TestSyncForeshadows_PlantedDedupByTitleGuard(t *testing.T) {
	a := newSyncTestAgent(t)
	seedForeshadows(t, a, types.Foreshadow{
		ID: "plot_001_old", Category: "mystery", Title: "古钟自鸣",
		Description: "村口古钟每晚自鸣", PlantedIn: "003.md",
		Status: types.ForeshadowPlanted, SourceType: types.ForeshadowSourceAnalysis,
	})

	// 同章同题但内容改写 → 稳定 ID 变了，第二道防重应命中
	res := a.syncForeshadows(3, []types.ForeshadowHit{
		{Type: "planted", Category: "mystery", Title: "古钟自鸣", Content: "村口古钟每到子夜自鸣三声"},
	})

	items := readForeshadowItems(t, a)
	if len(items) != 1 {
		t.Fatalf("同章同题分析条目应更新而非重复建档，实际 %d: %+v", len(items), items)
	}
	got := findByID(items, "plot_001_old")
	if got.Description != "村口古钟每到子夜自鸣三声" {
		t.Fatalf("既有条目应被更新描述: %+v", got)
	}
	if res.CreatedCount != 0 {
		t.Fatalf("更新路径不得计入创建: %+v", res)
	}
}

// TestSyncForeshadows_SameBatchNoDoubleMatch 同批两条 resolved（无 ID）匹配同一
// 埋入条目：第一条回收后候选池移除，第二条记 no_match 不重复消费。
func TestSyncForeshadows_SameBatchNoDoubleMatch(t *testing.T) {
	a := newSyncTestAgent(t)
	seedForeshadows(t, a, types.Foreshadow{
		ID: "plot_001_x", Description: "一枚成对的黑玉扳指", PlantedIn: "001.md", Status: types.ForeshadowPlanted,
	})

	res := a.syncForeshadows(2, []types.ForeshadowHit{
		{Type: "resolved", Title: "扳指", Content: "一枚成对的黑玉扳指"},
		{Type: "resolved", Title: "扳指", Content: "一枚成对的黑玉扳指"},
	})

	if res.ResolvedCount != 1 || res.MatchedByContent != 1 {
		t.Fatalf("同批同条目只应回收一次: %+v", res)
	}
	if skipKinds(res)[skipNoMatch] != 1 {
		t.Fatalf("第二条应记 no_match: %+v", res.SkippedReasons)
	}
}

// TestSyncForeshadows_UnknownTypeError 未知类型如实进 Errors，不中断整批。
func TestSyncForeshadows_UnknownTypeError(t *testing.T) {
	a := newSyncTestAgent(t)
	res := a.syncForeshadows(1, []types.ForeshadowHit{
		{Type: "hinted", Content: "旧动作残留"},
		{Type: "planted", Content: "正常建档"},
	})
	items := readForeshadowItems(t, a)
	if len(items) != 1 {
		t.Fatalf("未知类型条目不得建档，正常条目应继续: %+v", items)
	}
	if len(res.Errors) != 1 {
		t.Fatalf("未知类型应记入 Errors: %+v", res.Errors)
	}
}

// TestSyncForeshadows_ScoreDerivedDefaults 评分派生唯一规则与缺省语义：
// Importance=min(Strength/10,1)；Strength/Subtlety 0→缺省 5；>10 封顶。
func TestSyncForeshadows_ScoreDerivedDefaults(t *testing.T) {
	a := newSyncTestAgent(t)
	a.syncForeshadows(1, []types.ForeshadowHit{
		{Type: "planted", Content: "缺省评分条目"},
		{Type: "planted", Content: "超限评分条目", Strength: 15, Subtlety: -3},
	})
	items := readForeshadowItems(t, a)
	var def, clamp *types.Foreshadow
	for i := range items {
		switch items[i].Description {
		case "缺省评分条目":
			def = &items[i]
		case "超限评分条目":
			clamp = &items[i]
		}
	}
	if def == nil || def.Importance != 0.5 || def.Strength != 5 || def.Subtlety != 5 {
		t.Fatalf("缺省语义应为 5/5/0.5: %+v", def)
	}
	if clamp == nil || clamp.Importance != 1.0 || clamp.Strength != 10 || clamp.Subtlety != 5 {
		t.Fatalf("评分应钳制 1-10（负值→缺省 5）: %+v", clamp)
	}
}
