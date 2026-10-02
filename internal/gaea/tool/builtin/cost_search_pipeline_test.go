package builtin

// 审计 GA6-04 tool 面钉子：
//   1. 单源断言：cost_search 发给 rerank 服务的文档串必须逐字节等于
//      cost.RerankDocText（统一后的唯一实现）——把统一串改回任一旧版
//      （app/builtin 各自的同构拼接、或 BM25 语料串）此测试即红；
//   2. limit 语义：未精排（模型不可用）时按 limit 截断；精排生效时不截断。
// 补召回阈值语义由既有 TestCostSearchSemanticRecall 覆盖，不重复。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
)

// TestCostSearchToolRerankDocsSingleSource 捕获 cost_search 实际发给精排
// 服务的 documents，与 cost.RerankDocText 的输出集合逐字节对账（单源）。
func TestCostSearchToolRerankDocsSingleSource(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)

	// 10 条全部命中「材料」（>8 条才触发精排）。
	for i := 0; i < 10; i++ {
		title := "材料 " + strings.Repeat(string(rune('A'+i)), 2)
		if i == 9 {
			title = "材料 HP300 高频液压振动锤"
		}
		if err := store.Save(cost.Entry{
			Name: cost.SlugName(title) + string(rune('a'+i)), Title: title, Category: "材料",
			Unit: "件", Price: float64(100 + i), Spec: "规格" + string(rune('A'+i)), Status: "现行",
		}); err != nil {
			t.Fatal(err)
		}
	}

	// 捕获 documents 并按原下标降序原样返回（精排生效路径）。
	var captured []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-reranker-v2-m3"}]}`))
			return
		}
		var req struct {
			Documents []string `json:"documents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		captured = req.Documents
		results := make([]map[string]any, 0, len(req.Documents))
		for i := range req.Documents {
			results = append(results, map[string]any{"index": i, "relevance_score": float64(100 - i)})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}))
	defer srv.Close()
	SetRerankerForTest(retrieval.New(srv.URL, "bge-reranker-v2-m3"))
	t.Cleanup(func() { SetRerankerForTest(nil) })

	ks := costSearch{}
	if _, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "材料"})); err != nil {
		t.Fatal(err)
	}
	if len(captured) != 10 {
		t.Fatalf("精排收到 %d 份文档, want 10", len(captured))
	}
	// 单源对账：每份文档必须是 cost.RerankDocText 对某条目的输出，且 10 份
	// 恰好覆盖全部条目（无一份来自本包/本文件旧实现）。
	want := map[string]bool{}
	list, lerr := store.Search("材料", "", "")
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(list) != 10 {
		t.Fatalf("关键词命中 %d 条, want 10", len(list))
	}
	for _, e := range list {
		want[cost.RerankDocText(e)] = true
	}
	for _, doc := range captured {
		if !want[doc] {
			t.Fatalf("精排文档串不在单源集合内（文档串实现漂移）: %q", doc)
		}
		delete(want, doc)
	}
	if len(want) != 0 {
		t.Fatalf("精排文档串缺 %d 份: %v", len(want), want)
	}
}

// TestCostSearchToolTruncatesWithoutRerank 未精排（模型不可用）时按 limit
// 截断；fired=false 语义经共享 Enhance 传回 tool 面。
func TestCostSearchToolTruncatesWithoutRerank(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	for i := 0; i < 12; i++ {
		title := "材料 第" + string(rune('一'+i)) + "号"
		if err := store.Save(cost.Entry{
			Name: cost.SlugName(title), Title: title, Category: "材料",
			Unit: "件", Price: float64(i + 1), Status: "现行",
		}); err != nil {
			t.Fatal(err)
		}
	}
	// 不可用的 rerank 服务（连接拒绝）→ 精排降级，走截断。
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	dead.Close()
	SetRerankerForTest(retrieval.New(dead.URL, "bge-reranker-v2-m3"))
	t.Cleanup(func() { SetRerankerForTest(nil) })

	ks := costSearch{}
	res, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "材料", "limit": 5}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res, "（5 条）") {
		t.Fatalf("未精排时应按 limit=5 截断: %s", res)
	}
	if got := strings.Count(res, "材料 第"); got != 5 {
		t.Fatalf("截断后数据行 = %d, want 5: %s", got, res)
	}

	// limit 放宽到 50：12 条全出（不截断）。
	res2, err := ks.Execute(context.Background(), toJSON(t, map[string]interface{}{"query": "材料", "limit": 50}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res2, "（12 条）") {
		t.Fatalf("limit=50 应返回全部 12 条: %s", res2)
	}
}
