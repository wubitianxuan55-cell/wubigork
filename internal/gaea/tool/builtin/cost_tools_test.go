package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// TestCostSaveAndSearch 验证 cost_save 沉淀 + cost_search 引用闭环：
// 同名标题自动生成稳定 name，重复保存覆盖更新而非新增。
func TestCostSaveAndSearch(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	SetCostStoreForTest(cost.Open(gdb))

	cs := costSave{}
	result, err := cs.Execute(context.Background(), toJSON(t, map[string]interface{}{
		"title": "HP300 高频液压振动锤", "category": "机械", "unit": "台班",
		"price": 3200, "spec": "300kW", "source": "市场询价", "tags": "振动锤,桩基",
	}))
	if err != nil {
		t.Fatalf("costSave failed: %v", err)
	}
	if !strings.Contains(result, "已新增") || !strings.Contains(result, "3200.00") {
		t.Errorf("expected 新增 result with price, got: %s", result)
	}

	// 同名覆盖更新：价格变化，名称保持稳定。
	result, err = cs.Execute(context.Background(), toJSON(t, map[string]interface{}{
		"title": "HP300 高频液压振动锤", "category": "机械", "unit": "台班",
		"price": 3400, "spec": "300kW", "source": "市场询价",
	}))
	if err != nil {
		t.Fatalf("costSave upsert failed: %v", err)
	}
	if !strings.Contains(result, "覆盖更新") {
		t.Errorf("expected 覆盖更新, got: %s", result)
	}

	// 搜索命中且价格已更新。
	ks := costSearch{}
	searchResult, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "振动锤"}))
	if err != nil {
		t.Fatalf("costSearch failed: %v", err)
	}
	if !strings.Contains(searchResult, "HP300") || !strings.Contains(searchResult, "3400.00") {
		t.Errorf("expected updated entry in search, got: %s", searchResult)
	}

	// 分类过滤。
	filtered, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"category": "材料"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(filtered, "HP300") {
		t.Errorf("材料 filter should exclude 机械 entry: %s", filtered)
	}
}

// TestCostSearchOverview 空参数返回成本库概览。
func TestCostSearchOverview(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	SetCostStoreForTest(cost.Open(gdb))

	cs := costSave{}
	if _, err := cs.Execute(context.Background(), toJSON(t, map[string]interface{}{
		"title": "P.O 42.5 水泥", "category": "材料", "unit": "吨", "price": 480, "source": "定额",
	})); err != nil {
		t.Fatal(err)
	}

	ks := costSearch{}
	ov, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{}))
	if err != nil {
		t.Fatalf("costSearch overview failed: %v", err)
	}
	if !strings.Contains(ov, "成本库概览") || !strings.Contains(ov, "材料") {
		t.Errorf("expected 成本库概览 with 材料 category, got: %s", ov)
	}
}

// TestSlugName 标题生成稳定唯一键：大小写折叠、符号折叠、可重复。
func TestSlugName(t *testing.T) {
	a := cost.SlugName("P.O 42.5 水泥")
	b := cost.SlugName("P.O 42.5 水泥")
	if a != b || a == "" {
		t.Fatalf("name should be stable and non-empty: %q vs %q", a, b)
	}
	if strings.Contains(a, ".") || strings.Contains(a, " ") {
		t.Errorf("name should not contain punctuation/space: %q", a)
	}
}

// TestCostSearchRerank 验证候选多时 cost_search 走本地精排：
// rerank 把最后一条排到最前，搜索结果顺序跟随精排；服务失败时回退 SQL。
func TestCostSearchRerank(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)

	// 造 10 条（全部命中 LIKE 查询），其中一条与「振动锤」语义最相关。
	for i := 0; i < 10; i++ {
		title := "材料 " + strings.Repeat(string(rune('A'+i)), 2)
		if i == 9 {
			title = "材料 HP300 高频液压振动锤"
		}
		if err := store.Save(cost.Entry{
			Name: cost.SlugName(title), Title: title, Category: "材料",
			Unit: "件", Price: float64(100 + i), Status: "现行",
		}); err != nil {
			t.Fatal(err)
		}
	}

	// rerank 服务器：含「振动锤」的文档得分最高（模拟语义相关性）。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-reranker-v2-m3"}]}`))
			return
		}
		var req struct {
			Documents []string `json:"documents"`
			TopN      int      `json:"top_n"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		results := []map[string]any{}
		idx := make([]int, len(req.Documents))
		for i := range idx {
			idx[i] = i
		}
		// 按得分降序：含振动锤的排最前。
		score := func(i int) float64 {
			if strings.Contains(req.Documents[i], "振动锤") {
				return 100
			}
			return 0
		}
		for i := 0; i < len(idx); i++ {
			for j := i + 1; j < len(idx); j++ {
				if score(idx[j]) > score(idx[i]) {
					idx[i], idx[j] = idx[j], idx[i]
				}
			}
		}
		for i, v := range idx {
			if i >= req.TopN {
				break
			}
			results = append(results, map[string]any{"index": v, "relevance_score": score(v)})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}))
	defer srv.Close()
	SetRerankerForTest(retrieval.New(srv.URL, "bge-reranker-v2-m3"))

	ks := costSearch{}
	res, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "材料", "limit": 3}))
	if err != nil {
		t.Fatalf("costSearch failed: %v", err)
	}
	if !strings.Contains(res, "HP300 高频液压振动锤") {
		t.Fatalf("reranked result should contain 振动锤 entry: %s", res)
	}
	// 精排后 HP300 应排在第一行（表格首行数据）。
	if strings.Index(res, "材料 HP300 高频液压振动锤") > strings.Index(res, "材料 BB") {
		t.Errorf("HP300 应排最前（精排生效）：%s", res)
	}

	// 服务失败回退：结果仍返回且不报错。
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	SetRerankerForTest(retrieval.New(srv2.URL, "bge-reranker-v2-m3"))
	res2, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "材料", "limit": 3}))
	if err != nil {
		t.Fatalf("fallback should not error: %v", err)
	}
	if !strings.Contains(res2, "材料") {
		t.Errorf("fallback result should contain entries: %s", res2)
	}
	SetRerankerForTest(nil)
}

// TestCostSearchRerankLiveHerdsman 真实 Herdsman bge-reranker-v2-m3 端到端验证
// （仅 RERANK_LIVE=1 时运行，需要本机 Herdsman 已装该模型）。
func TestCostSearchRerankLiveHerdsman(t *testing.T) {
	if os.Getenv("RERANK_LIVE") != "1" {
		t.Skip("RERANK_LIVE=1 时运行真实 Herdsman 验证")
	}
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	entries := []cost.Entry{
		{Name: "cement", Title: "P.O 42.5 水泥", Category: "材料", Unit: "吨", Price: 480, Status: "现行"},
		{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Spec: "300kW", Status: "现行"},
		{Name: "excavator", Title: "挖掘机 220", Category: "机械", Unit: "台班", Price: 2600, Status: "现行"},
	}
	for _, e := range entries {
		if err := store.Save(e); err != nil {
			t.Fatal(err)
		}
	}
	SetRerankerForTest(retrieval.New("http://localhost:8080", "bge-reranker-v2-m3"))

	ks := costSearch{}
	res, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "液压振动锤 台班", "limit": 3}))
	if err != nil {
		t.Fatalf("live rerank search failed: %v", err)
	}
	if !strings.Contains(res, "HP300 高频液压振动锤") {
		t.Fatalf("live rerank should surface 振动锤: %s", res)
	}
	t.Logf("live pipeline OK（多词召回 + Herdsman 连通）: %s", res[:160])
	SetRerankerForTest(nil)
}

// TestCostSearchSemanticRecall 验证关键词召回不足时走本地 embedding 补召回：
// 「打桩锤」不子串命中任何标题，但语义上应召回 HP300 液压振动锤。
func TestCostSearchSemanticRecall(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	_ = store.Save(cost.Entry{Name: "cement", Title: "P.O 42.5 水泥", Category: "材料", Unit: "吨", Price: 480, Status: "现行"})
	_ = store.Save(cost.Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Status: "现行"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-m3"}]}`))
			return
		}
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		data := make([]map[string]any, 0, len(req.Input))
		for i, s := range req.Input {
			vec := []float32{0, 0}
			if strings.Contains(s, "锤") || strings.Contains(s, "液压") {
				vec[0] = 1
			}
			if strings.Contains(s, "水泥") {
				vec[1] = 1
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()
	SetEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))
	SetSemanticStoreForTest(semantic.Open(gdb))

	ks := costSearch{}
	res, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "打桩锤"}))
	if err != nil {
		t.Fatalf("costSearch failed: %v", err)
	}
	if !strings.Contains(res, "HP300 高频液压振动锤") {
		t.Fatalf("semantic recall should surface 振动锤 entry: %s", res)
	}

	// embedding 服务不可用 → 回退「未找到」提示，不报错。
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	SetEmbedderForTest(retrieval.NewEmbedder(srv2.URL, "bge-m3"))
	res2, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "打桩锤"}))
	if err != nil {
		t.Fatalf("fallback should not error: %v", err)
	}
	if !strings.Contains(res2, "未找到匹配") {
		t.Errorf("expected not-found fallback: %s", res2)
	}
	SetEmbedderForTest(nil)
	SetSemanticStoreForTest(nil)
}

// TestCostSearchSemanticLiveHerdsman 真实 Herdsman bge-m3 语义召回端到端
// （仅 SEMANTIC_LIVE=1 时运行）。
func TestCostSearchSemanticLiveHerdsman(t *testing.T) {
	if os.Getenv("SEMANTIC_LIVE") != "1" {
		t.Skip("SEMANTIC_LIVE=1 时运行真实 bge-m3 验证")
	}
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	_ = store.Save(cost.Entry{Name: "cement", Title: "P.O 42.5 水泥", Category: "材料", Unit: "吨", Price: 480, Status: "现行"})
	_ = store.Save(cost.Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Spec: "300kW", Status: "现行"})
	SetEmbedderForTest(retrieval.NewEmbedder("http://localhost:8080", "bge-m3"))

	ks := costSearch{}
	res, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "打桩设备 台班价"}))
	if err != nil {
		t.Fatalf("live semantic search failed: %v", err)
	}
	if !strings.Contains(res, "HP300 高频液压振动锤") {
		t.Fatalf("live bge-m3 should recall 振动锤: %s", res)
	}
	t.Logf("live semantic recall OK: %s", res[:160])
	SetEmbedderForTest(nil)
}
