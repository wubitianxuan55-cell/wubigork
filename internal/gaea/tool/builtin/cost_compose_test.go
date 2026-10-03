package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

// toJSON 测试参数助手（cost_tools_test.go 同款形态）。
func composeToJSON(t *testing.T, v map[string]interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestCostComposeBandAndEvidence 组价主链：相似条目 → 价格带+推荐价+证据链，
// 只读（不落库），并引导 cost_save 确认沉淀。
func TestCostComposeBandAndEvidence(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	defer SetCostStoreForTest(nil)

	// 造 4 条同单位相似样本（置信度=中），价格 3000/3200/3400/3600。
	for i, price := range []float64{3000, 3200, 3400, 3600} {
		if err := store.Save(cost.Entry{
			Name:  fmt.Sprintf("hp300-%d", i),
			Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班",
			Price: price, Spec: "300kW", Source: "历史项目", Status: "现行",
			Body: strings.Repeat("x", i+1), // body 微差避免同 name 互相覆盖
		}); err != nil {
			t.Fatal(err)
		}
	}

	cc := costCompose{}
	out, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "HP300 高频液压振动锤", "unit": "台班",
	}))
	if err != nil {
		t.Fatalf("cost_compose failed: %v", err)
	}
	for _, want := range []string{
		"AI 组价", "价格带", "推荐价", "证据链", "置信度", "中位数",
		"三档对照", "历史项目", "cost_save",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output 缺 %q: %s", want, out)
		}
	}

	// 只读：库不被写（条目数不变）。
	if all, _ := store.List(); len(all) != 4 {
		t.Errorf("cost_compose 应只读，条目数 %d != 4", len(all))
	}
}

// TestCostComposeEmptyAndEdge 空库引导、description 必填、单位全排除 band=nil。
func TestCostComposeEmptyAndEdge(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	defer SetCostStoreForTest(nil)

	cc := costCompose{}

	// 空库：诚实说无法组价并引导沉淀。
	out, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "挖机土方",
	}))
	if err != nil {
		t.Fatalf("empty store failed: %v", err)
	}
	if !strings.Contains(out, "相似条目") || !strings.Contains(out, "cost_save") {
		t.Errorf("空库应引导沉淀: %s", out)
	}

	// description 必填。
	if _, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{})); err == nil {
		t.Error("缺 description 应报错")
	}

	// 有样本但单位过滤全排除 → band=nil 提示去单位重试。
	if err := store.Save(cost.Entry{Name: "steel-bar", Title: "钢筋", Category: "材料", Unit: "吨", Price: 4200, Status: "现行"}); err != nil {
		t.Fatal(err)
	}
	out, err = cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "钢筋", "unit": "m³",
	}))
	if err != nil {
		t.Fatalf("unit-filter failed: %v", err)
	}
	if !strings.Contains(out, "去掉 unit") {
		t.Errorf("单位全排除应提示重试: %s", out)
	}
}

// TestCostComposeMode 多方案对照：mode=p25 推荐走 P25；未知 mode 回落中位数。
func TestCostComposeMode(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	defer SetCostStoreForTest(nil)

	for i, price := range []float64{1000, 1200, 1400, 1600} {
		if err := store.Save(cost.Entry{
			Name:  fmt.Sprintf("c30-%d", i),
			Title: "C30 混凝土", Category: "材料", Unit: "m³",
			Price: price, Status: "现行",
			Body: strings.Repeat("x", i+1),
		}); err != nil {
			t.Fatal(err)
		}
	}

	cc := costCompose{}
	p25Out, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "C30 混凝土", "unit": "m³", "mode": "p25",
	}))
	if err != nil {
		t.Fatalf("mode=p25: %v", err)
	}
	if !strings.Contains(p25Out, "P25 分位") {
		t.Errorf("mode=p25 推荐理由应含 P25 分位: %s", p25Out)
	}

	unk, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "C30 混凝土", "unit": "m³", "mode": "nope",
	}))
	if err != nil {
		t.Fatalf("mode=nope: %v", err)
	}
	if !strings.Contains(unk, "中位数") {
		t.Errorf("未知 mode 应回落中位数: %s", unk)
	}
}

// TestCostComposeRecallBelowThreshold <3 补召回编排（批 18 留池收口）：关键词
// 命中 < cost.RecallBelow 时 Enhance 触发 Recall 钩子并把召回结果并入候选。
// 注入 semanticCostRecall 桩验证双向口径：<3 必触发且合并（证据链出现召回
// 条目）、≥3 不触发；触发次数恒 1（单次编排单次召回）。
func TestCostComposeRecallBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	defer SetCostStoreForTest(nil)

	// 语料：HP300×2（<RecallBelow=3 的命中组）、塔吊×2（不命中，召回桩货源）、
	// C30×3（≥3 的命中组，反向用）。
	seed := []struct {
		name, title, spec string
		price             float64
	}{
		{"hp300-0", "HP300 高频液压振动锤", "300kW", 3000},
		{"hp300-1", "HP300 高频液压振动锤", "300kW", 3200},
		{"tower-0", "塔式起重机", "QTZ80", 800},
		{"tower-1", "塔式起重机", "QTZ80", 900},
		{"c30-0", "C30 混凝土", "P.O 42.5", 380},
		{"c30-1", "C30 混凝土", "P.O 42.5", 400},
		{"c30-2", "C30 混凝土", "P.O 42.5", 420},
	}
	for i, e := range seed {
		if err := store.Save(cost.Entry{
			Name: e.name, Title: e.title, Category: "机械", Unit: "台班",
			Price: e.price, Spec: e.spec, Source: "历史项目", Status: "现行",
			Body: strings.Repeat("x", i+1),
		}); err != nil {
			t.Fatal(err)
		}
	}

	calls := 0
	orig := semanticCostRecall
	t.Cleanup(func() { semanticCostRecall = orig })
	semanticCostRecall = func(ctx context.Context, query string, have []cost.Summary, store *cost.Store, topN int) []cost.Summary {
		calls++
		out := append([]cost.Summary{}, have...)
		out = append(out,
			cost.Summary{Name: "tower-0", Title: "塔式起重机", Unit: "台班", Price: 800, Source: "历史项目"},
			cost.Summary{Name: "tower-1", Title: "塔式起重机", Unit: "台班", Price: 900, Source: "历史项目"})
		return out
	}

	cc := costCompose{}
	// 正向：命中 2 < 3 → 触发召回，塔吊条目并入证据链。
	out, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "HP300 高频液压振动锤", "unit": "台班",
	}))
	if err != nil {
		t.Fatalf("cost_compose failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("命中 <3 应触发补召回恰 1 次, got %d", calls)
	}
	if !strings.Contains(out, "塔式起重机") {
		t.Fatalf("召回结果未并入候选（证据链无塔吊条目）:\n%s", out)
	}

	// 反向：命中 3 ≥ 3 → 不触发（阈值下界不抖动）。
	calls = 0
	out, err = cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "C30 混凝土", "unit": "台班",
	}))
	if err != nil {
		t.Fatalf("cost_compose(C30) failed: %v", err)
	}
	if calls != 0 {
		t.Fatalf("命中 ≥3 不应触发补召回, got %d 次", calls)
	}
	if strings.Contains(out, "塔式起重机") {
		t.Fatalf("未触发时召回桩条目不应出现:\n%s", out)
	}
}
