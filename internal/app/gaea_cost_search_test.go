package app

// gaea_cost_search_test.go — 造价库混合检索的应用层接线：cost_project 的三个检索动作与
// 两个界面列表绑定。模型依赖用测试注入缝换成确定性的假向量化（同义词落同一维）。

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/vecindex"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// costBagEmbedder 测试用的「语义」向量化：同义词落同一维（one-hot），都不命中的落最后一维。
type costBagEmbedder struct{}

var costBuckets = [][]string{{"挖掘机", "挖机"}, {"混凝土", "砼"}}

func (costBagEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, len(costBuckets)+1)
		v[len(costBuckets)] = 1
		for b, words := range costBuckets {
			for _, w := range words {
				if strings.Contains(t, w) {
					v = make([]float32, len(costBuckets)+1)
					v[b] = 1
					break
				}
			}
		}
		out[i] = v
	}
	return out, nil
}

// costSearchApp 建带索引的测试环境：临时库 + 三类索引建好 + 假模型注入。
// restore 用来撤掉注入（验证「索引没建时保持原列表口径」）。
func costSearchApp(t *testing.T) (a *App, pid int64, restore func(), cleanup func()) {
	t.Helper()
	a, gdb, closeDB := workcostTestAppDB(t)
	s := a.hubWorkcostStore()
	for _, r := range []workcost.Resource{
		{Kind: workcost.KindMachine, Title: "单斗挖掘机 挖土", Spec: "斗容 1m3", Unit: "m3"},
		{Kind: workcost.KindMaterial, Title: "商品混凝土", Unit: "m3"},
	} {
		if _, err := s.SaveResource(r); err != nil {
			t.Fatalf("建工料机失败: %v", err)
		}
	}
	if _, err := s.SaveQuota(workcost.Quota{Code: "Q1", Title: "挖掘机挖土", Unit: "m3"}, nil); err != nil {
		t.Fatalf("建定额失败: %v", err)
	}
	pid, err := s.UpsertBillProject("测算P", "", "", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "WP01", Title: "混凝土垫层", Unit: "m3", Quantity: 10}); err != nil {
		t.Fatalf("建清单行失败: %v", err)
	}
	vs := vecindex.Open(gdb)
	for kind, src := range s.IndexSources() {
		if _, err := vs.Apply(context.Background(), kind, "bge-m3", src, costBagEmbedder{}, 128); err != nil {
			t.Fatalf("建索引 %s: %v", kind, err)
		}
	}
	restore = SetCostIndexForTest(vs, func() workcost.SearchEmbedder { return costBagEmbedder{} }, nil)
	return a, pid, restore, func() {
		vecindex.Forget(gdb)
		closeDB()
	}
}

func costProjectRun(t *testing.T, a *App, args map[string]any) string {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	out, err := (costProjectTool{a: a}).Execute(context.Background(), b)
	if err != nil {
		t.Fatalf("cost_project %v: %v", args, err)
	}
	return out
}

// 同义说法经向量路召回并标〈语义〉；名称含查询词的不标；编码精确标〈编码〉且只返回它。
func TestCostProjectSearch_HybridMarks(t *testing.T) {
	a, _, restore, cleanup := costSearchApp(t)
	defer func() { restore(); cleanup() }()

	out := costProjectRun(t, a, map[string]any{"action": "resources", "keyword": "挖机"})
	if !strings.Contains(out, "单斗挖掘机 挖土") || !strings.Contains(out, "〈语义〉") {
		t.Fatalf("「挖机」应经向量路召回并标〈语义〉:\n%s", out)
	}
	if out = costProjectRun(t, a, map[string]any{"action": "resources", "keyword": "挖掘机"}); strings.Contains(out, "〈") {
		t.Fatalf("名称含查询词的关键词命中不该标:\n%s", out)
	}
	if out = costProjectRun(t, a, map[string]any{"action": "quotas", "keyword": "q1"}); !strings.Contains(out, "[Q1]") ||
		!strings.Contains(out, "〈编码〉") || strings.Contains(out, "[Q2]") {
		t.Fatalf("编码精确应只返回那一条并标〈编码〉:\n%s", out)
	}
	// 向量路不可用（撤掉注入 → 用户库上没有索引）：退化成关键词，「挖机」对不上 LIKE，返回空
	restore()
	if out = costProjectRun(t, a, map[string]any{"action": "resources", "keyword": "挖机"}); !strings.Contains(out, "无匹配") {
		t.Fatalf("索引没建时应退化为关键词检索:\n%s", out)
	}
	if out = costProjectRun(t, a, map[string]any{"action": "resources", "keyword": "商品"}); !strings.Contains(out, "商品混凝土") {
		t.Fatalf("关键词路应照常: %s", out)
	}
}

// bills：跨项目清单行检索；project_id / keyword 照常。
func TestCostProjectSearch_BillsAcrossProjects(t *testing.T) {
	a, pid, _, cleanup := costSearchApp(t)
	defer cleanup()

	out := costProjectRun(t, a, map[string]any{"action": "bills", "keyword": "砼"})
	if !strings.Contains(out, "混凝土垫层") || !strings.Contains(out, "〈语义〉") ||
		!strings.Contains(out, "项目 #"+strconv.FormatInt(pid, 10)) {
		t.Fatalf("bills 应跨项目召回并带项目号:\n%s", out)
	}
	if out = costProjectRun(t, a, map[string]any{"action": "bills", "keyword": "垫层"}); !strings.Contains(out, "WP01") ||
		strings.Contains(out, "〈语义〉") {
		t.Fatalf("bills 关键词命中: %s", out)
	}
	if out = costProjectRun(t, a, map[string]any{"action": "bills", "keyword": "垫层", "project_id": 999}); !strings.Contains(out, "无匹配") {
		t.Fatalf("project_id 过滤: %s", out)
	}
	// 空查询词 = 跨项目列表
	if out = costProjectRun(t, a, map[string]any{"action": "bills"}); !strings.Contains(out, "WP01") {
		t.Fatalf("bills 不带关键词应列清单: %s", out)
	}
}

// 界面列表绑定：索引就绪且带关键词走混合检索；只是浏览（没给关键词）保持全量列表；
// 索引没建时按原 LIKE 口径。
func TestCostSearchBindings_UseIndexOnlyWhenReady(t *testing.T) {
	a, _, restore, cleanup := costSearchApp(t)
	defer func() { restore(); cleanup() }()

	if got := a.GaeaWorkcostResourceList("", "挖机"); len(got) != 1 || !strings.HasPrefix(got[0].Title, "单斗挖掘机") {
		t.Fatalf("索引就绪时界面检索应召回同义说法: %+v", got)
	}
	if got := a.GaeaWorkcostResourceList("", ""); len(got) != 2 {
		t.Fatalf("只是浏览应全量返回 2 条: %+v", got)
	}
	if got := a.GaeaWorkcostResourceList(workcost.KindMaterial, ""); len(got) != 1 || got[0].Title != "商品混凝土" {
		t.Fatalf("类型过滤: %+v", got)
	}
	if got := a.GaeaWorkcostQuotaList("", "挖机"); len(got) != 1 || got[0].Code != "Q1" {
		t.Fatalf("定额列表检索: %+v", got)
	}
	// 索引没建：原口径（LIKE），「挖机」对不上
	restore()
	if got := a.GaeaWorkcostResourceList("", "挖机"); len(got) != 0 {
		t.Fatalf("索引没建时应按原 LIKE 口径返回空: %+v", got)
	}
	if got := a.GaeaWorkcostQuotaList("", "挖掘机"); len(got) != 1 {
		t.Fatalf("索引没建时关键词路照常: %+v", got)
	}
}
