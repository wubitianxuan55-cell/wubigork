package app

// v4.158.0 AI 组价复核闭环(Go 侧)应用层用例:确认即留痕 + 回看查询。
// 不真调 LLM——GaeaCostCompose 的 LLM 路径不在本刀范围,手工构造
// CostComposeView 走 GaeaCostComposeApply(确认回写 + 留痕),再用
// GaeaCostComposeRecords 回读快照验证证据链没有当场即丢。

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

// newComposeRecordApp 隔离成本库环境(临时目录 + 注入,不触碰真实用户库)。
func newComposeRecordApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	SetCostStoreForTest(cost.Open(gdb))
	t.Cleanup(ResetCostStoreForTest)
	return &App{}
}

// sampleComposeView 手工构造的确认视图:双行人材机 + 三条证据链 + 一条 warn 校验。
func sampleComposeView(desc string) CostComposeView {
	return CostComposeView{
		Description:      desc,
		Unit:             "m²",
		RecommendedPrice: 480.5,
		Reason:           "相似条目中位数",
		Components: []CostComponentView{
			{Kind: "人工", Title: "摊铺人工", Unit: "工日", Quantity: 0.5, Price: 150, Amount: 75},
			{Kind: "材料", Title: "C30 混凝土", Unit: "m³", Quantity: 1.02, Price: 380, Amount: 387.6},
		},
		ComponentsNote: "AI 拆解完成,请核对含量与单价",
		LLMUsed:        true,
		Evidence: []CostComposeEvidence{
			{Name: "e1", Title: "路面摊铺 2025-11", Price: 470, Source: "信息价", Region: "成都", PriceDate: "2025-11", PriceType: "到场价"},
			{Name: "e2", Title: "路面摊铺 2025-12", Price: 485, Source: "信息价", Region: "成都", PriceDate: "2025-12", PriceType: "到场价"},
			{Name: "e3", Title: "路面摊铺 2026-01", Price: 490, Source: "信息价", Region: "成都", PriceDate: "2026-01", PriceType: "到场价"},
		},
		Checks: []cost.ComposeCheck{{Level: "warn", Row: 0, Msg: "金额 75.00 ≠ 含量×单价 75.00"}},
	}
}

// TestGaeaCostComposeApplyRecordsRoundTrip 闭环主路径:Apply 成功后按条目回看,
// 快照的描述/推荐价/证据链长度与确认视图吻合(LLMUsed/Checks 一并留痕)。
func TestGaeaCostComposeApplyRecordsRoundTrip(t *testing.T) {
	a := newComposeRecordApp(t)
	v := sampleComposeView("C30 混凝土路面摊铺\n(机械摊铺,含切缝)")
	name, err := a.GaeaCostComposeApply(v)
	if err != nil {
		t.Fatal(err)
	}
	if name != cost.SlugName("C30 混凝土路面摊铺") {
		t.Fatalf("条目键异常: %q", name)
	}
	// 确认回写主意图:成本库出现该条目(来源标注 AI组价)。
	e, err := a.hubCostStore().Get(name)
	if err != nil || e == nil {
		t.Fatalf("回写成本库失败: %v", err)
	}
	if e.Source != "AI组价" {
		t.Fatalf("Source = %q, want AI组价", e.Source)
	}

	// 回看:该条目恰 1 条留痕,快照与确认视图吻合。
	records, err := a.GaeaCostComposeRecords(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("want 1 条留痕, got %d", len(records))
	}
	rec := records[0]
	if !rec.LLMUsed {
		t.Error("llmUsed 应为 true")
	}
	if rec.Snapshot == nil {
		t.Fatal("快照反序列化不应失败")
	}
	if rec.Snapshot.Description != v.Description {
		t.Errorf("快照描述不符: %q vs %q", rec.Snapshot.Description, v.Description)
	}
	if rec.Snapshot.RecommendedPrice != 480.5 {
		t.Errorf("快照推荐价不符: %v", rec.Snapshot.RecommendedPrice)
	}
	if len(rec.Snapshot.Evidence) != 3 {
		t.Errorf("证据链长度不符: %d", len(rec.Snapshot.Evidence))
	}
	if len(rec.Snapshot.Components) != 2 || len(rec.Snapshot.Checks) != 1 {
		t.Errorf("组件/校验留痕不符: %d/%d", len(rec.Snapshot.Components), len(rec.Snapshot.Checks))
	}
	if rec.CreatedAt == "" {
		t.Error("createdAt 不应为空")
	}
}

// TestGaeaCostComposeRecordsLatestFirst 空 entryName 返回全库最近 20 条,
// 同秒以 id 兜底保证「后确认先出现」。
func TestGaeaCostComposeRecordsLatestFirst(t *testing.T) {
	a := newComposeRecordApp(t)
	nameA, err := a.GaeaCostComposeApply(sampleComposeView("人行道砖铺设"))
	if err != nil {
		t.Fatal(err)
	}
	nameB, err := a.GaeaCostComposeApply(sampleComposeView("路缘石安装"))
	if err != nil {
		t.Fatal(err)
	}

	all, err := a.GaeaCostComposeRecords("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 条留痕, got %d", len(all))
	}
	if all[0].EntryName != nameB || all[1].EntryName != nameA {
		t.Fatalf("应按最近确认倒序: [%s %s]", all[0].EntryName, all[1].EntryName)
	}
	for _, rec := range all {
		if rec.Snapshot == nil {
			t.Fatalf("条目 %s 快照不应为 nil", rec.EntryName)
		}
	}

	// 按条目过滤互不串扰。
	onlyA, err := a.GaeaCostComposeRecords(nameA)
	if err != nil || len(onlyA) != 1 || onlyA[0].EntryName != nameA {
		t.Fatalf("按条目回看异常: %v %+v", err, onlyA)
	}
	// 无留痕条目返回空而非报错。
	none, err := a.GaeaCostComposeRecords("从未组价条目")
	if err != nil || len(none) != 0 {
		t.Fatalf("无留痕应返回空: %v %+v", err, none)
	}
}

// TestGaeaCostComposeRecordsBadSnapshot 坏 snapshot 行如实标 Snapshot=nil,不整批失败。
func TestGaeaCostComposeRecordsBadSnapshot(t *testing.T) {
	a := newComposeRecordApp(t)
	if _, err := a.GaeaCostComposeApply(sampleComposeView("侧平石安砌")); err != nil {
		t.Fatal(err)
	}
	// 直插一条坏快照(模拟历史脏数据/截断)。
	gdb := a.hubCostStore().DB()
	if _, err := gdb.Exec(
		`INSERT INTO cost_compose_records(entry_name, created_at, llm_used, snapshot) VALUES('bad-entry', '2026-01-01T00:00:00Z', 1, '{broken')`); err != nil {
		t.Fatal(err)
	}

	bad, err := a.GaeaCostComposeRecords("bad-entry")
	if err != nil {
		t.Fatalf("坏快照不应导致查询失败: %v", err)
	}
	if len(bad) != 1 || bad[0].Snapshot != nil || !bad[0].LLMUsed {
		t.Fatalf("坏快照行应 Snapshot=nil 且 llmUsed=true: %+v", bad)
	}

	// 混合批次:好快照照常解析,坏快照降级,互不影响。
	all, err := a.GaeaCostComposeRecords("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 条, got %d", len(all))
	}
	nilCount, okCount := 0, 0
	for _, rec := range all {
		if rec.Snapshot == nil {
			nilCount++
		} else {
			okCount++
		}
	}
	if nilCount != 1 || okCount != 1 {
		t.Fatalf("好/坏快照应各 1 条: nil=%d ok=%d", nilCount, okCount)
	}
}

// TestGaeaCostComposeRecordsUnavailable 成本库不可用时查询报错(不静默返回空)。
func TestGaeaCostComposeRecordsUnavailable(t *testing.T) {
	SetCostStoreForTest(cost.Open(nil))
	t.Cleanup(ResetCostStoreForTest)
	a := &App{}
	if _, err := a.GaeaCostComposeRecords("c30"); err == nil {
		t.Fatal("不可用库应返回错误")
	}
}

// TestComposeChecksContentBaseline v4.209.0 含量对照接线:池=相似条目组件明细,
// 拆解含量带外时 Checks 追加含量 warn(与金额自洽结论同列,零新结构零新绑定);
// 带内/池空静默。不真调 LLM——直接单测汇总函数。
func TestComposeChecksContentBaseline(t *testing.T) {
	a := newComposeRecordApp(t)
	store := a.hubCostStore()
	// 池:3 条同类条目,同名资源「C30 混凝土」含量 1.00~1.02(P25=P75 同值带)。
	for i, q := range []float64{1.00, 1.01, 1.02} {
		name := fmt.Sprintf("pool-entry-%d", i)
		err := store.Save(cost.Entry{
			Name: name, Title: name, Category: "路面", CategoryPath: "市政/道路/路面",
			Unit: "m²", Price: 480, Status: "现行",
			Components: []cost.Component{
				{Kind: "材料", Title: "C30 混凝土", Unit: "m³", Quantity: q, Price: 380, Amount: q * 380},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	similar := []cost.Summary{{Name: "pool-entry-0"}, {Name: "pool-entry-1"}, {Name: "pool-entry-2"}}

	// 带外:含量 1.5(同值带 1.01±5% 之外)→ 含量偏高 warn;同时人材机合计
	// 570 > 480×1.05 → R3 全局 warn,共 2 条。
	outBand := []cost.Component{
		{Kind: "材料", Title: "C30 混凝土", Unit: "m³", Quantity: 1.5, Price: 380, Amount: 570},
	}
	checks := a.composeChecks(similar, outBand, 480)
	if len(checks) != 2 {
		t.Fatalf("want R3+含量共 2 条, got %+v", checks)
	}
	var content *cost.ComposeCheck
	for i := range checks {
		if checks[i].Row == 0 {
			content = &checks[i]
		}
	}
	if content == nil || !strings.Contains(content.Msg, "高") {
		t.Fatalf("row0 应有含量偏高 warn, got %+v", checks)
	}

	// 带内:含量 1.01、金额 383.8 在 (240,504) 内 → 金额自洽与含量对照双静默。
	inBand := []cost.Component{
		{Kind: "材料", Title: "C30 混凝土", Unit: "m³", Quantity: 1.01, Price: 380, Amount: 383.8},
	}
	if got := a.composeChecks(similar, inBand, 480); len(got) != 0 {
		t.Fatalf("带内应双静默, got %+v", got)
	}

	// 池空(相似条目无组件明细)→ 含量对照层静默,只剩金额自洽结论。
	empty := []cost.Summary{{Name: "pool-entry-miss"}}
	if got := a.composeChecks(empty, outBand, 480); len(got) != 1 || got[0].Row != -1 {
		t.Fatalf("池空应只剩 R3 全局一条, got %+v", got)
	}
}
