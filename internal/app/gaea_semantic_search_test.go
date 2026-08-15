package app

// gaea_semantic_search_test.go — T7-3：语义检索按需/缓存，避免每查询先 Ensure
// 全量扫描。用计数 embedding 服务验证：索引就绪后二次查询只嵌 query，
// 不重嵌任何文档；数据增删时才按需增量 Ensure。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// countingEmbedEnv 搭建带「计数 embedding 服务」的检索环境：记录每次被嵌入的
// 文本，供断言「二次查询不重嵌文档」。
func countingEmbedEnv(t *testing.T) (a *App, embedded *[]string, mu *sync.Mutex) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	var recorded []string
	var mu2 sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("{\"data\":[{\"id\":\"bge-m3\"}]}"))
			return
		}
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		mu2.Lock()
		for _, s := range req.Input {
			recorded = append(recorded, s)
		}
		mu2.Unlock()
		data := make([]map[string]any, 0, len(req.Input))
		for _, s := range req.Input {
			vec := make([]float32, 4)
			if strings.Contains(s, "锤") {
				vec[0] = 1
			}
			if strings.Contains(s, "水泥") {
				vec[1] = 1
			}
			if strings.Contains(s, "桩") {
				vec[2] = 1
			}
			if strings.Contains(s, "规范") {
				vec[3] = 1
			}
			data = append(data, map[string]any{"index": len(data), "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(srv.Close)

	gdb := db.GetDatabase(home)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(home)
		SetAppSemanticStoreForTest(nil)
		SetAppEmbedderForTest(nil)
		knowledge.SetStoreForTest(nil)
		ResetOfficeStoreForTest()
		ResetCostStoreForTest()
	})
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	SetAppEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))
	SetOfficeStoreForTest(memory.SQLiteStoreFor(gdb, home, home))
	SetCostStoreForTest(cost.Open(gdb))

	isoStore, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(isoStore)

	a = &App{core: &core{}}
	return a, &recorded, &mu2
}

// seedSemanticData 灌入三类可命中「振动锤」的跨库条目。
func seedSemanticData(t *testing.T, a *App) {
	t.Helper()
	ks, err := knowledge.Global().Store()
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.Save(knowledge.Entry{
		Name: "pile-guide", Title: "桩基施工要点", Category: knowledge.CatCase,
		Body: "振动锤选型需匹配地质，桩基施工工艺详见规范。",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.hubCostStore().Save(cost.Entry{
		Name: "pile-rent", Title: "振动锤租赁台班", Category: "机械", Unit: "台班", Price: 6500,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.hubOfficeStore().Save(memory.Memory{
		Name: "office-pile", Title: "打桩机械台账", Description: "振动锤与静压桩机配置", Body: "记录现场机械调度。",
	}); err != nil {
		t.Fatal(err)
	}
}

// TestGaeaSemanticSearchNoFullEnsure 索引就绪后二次查询不触发全量 Ensure：
// 只嵌 query（3 个有向量的 kind 各 1 次），不重嵌任何文档文本。
func TestGaeaSemanticSearchNoFullEnsure(t *testing.T) {
	a, recorded, mu := countingEmbedEnv(t)
	seedSemanticData(t, a)

	hits, err := a.GaeaSemanticSearch("振动锤")
	if err != nil {
		t.Fatalf("GaeaSemanticSearch: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("首查应命中跨库条目")
	}

	// 首查：Ensure 嵌入 3 个文档 + 3 个 kind 的 query 嵌入 = 6 次嵌入。
	mu.Lock()
	first := len(*recorded)
	mu.Unlock()
	if first != 6 {
		t.Fatalf("首查嵌入次数 = %d, want 6（3 文档 + 3 query）", first)
	}

	// 二次查询：只嵌 query，绝不重嵌文档 → 3 次嵌入且全部为 query 文本。
	mu.Lock()
	*recorded = nil
	mu.Unlock()
	hits2, err := a.GaeaSemanticSearch("振动锤")
	if err != nil {
		t.Fatalf("二次 GaeaSemanticSearch: %v", err)
	}
	if len(hits2) == 0 {
		t.Fatal("二次查询应命中相同条目")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*recorded) != 3 {
		t.Fatalf("二次查询应只嵌 query（3 次），实际 %d 次: %v", len(*recorded), *recorded)
	}
	for _, s := range *recorded {
		if s != "振动锤" {
			t.Fatalf("二次查询不应重嵌文档，出现非 query 文本: %q", s)
		}
	}
}

// TestGaeaSemanticSearchOnDemandEnsureAfterDataChange 数据增删时按需增量 Ensure：
// 新增条目被嵌入，未变文档不重嵌（非全量重建）。
func TestGaeaSemanticSearchOnDemandEnsureAfterDataChange(t *testing.T) {
	a, recorded, mu := countingEmbedEnv(t)
	seedSemanticData(t, a)
	if _, err := a.GaeaSemanticSearch("振动锤"); err != nil {
		t.Fatalf("首查: %v", err)
	}

	// 新增一条知识（含「水泥」）
	ks, err := knowledge.Global().Store()
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.Save(knowledge.Entry{
		Name: "cement-guide", Title: "混凝土配比", Category: knowledge.CatCase,
		Body: "水泥用量与标号关系，配合比设计要点。",
	}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	*recorded = nil
	mu.Unlock()
	if _, err := a.GaeaSemanticSearch("水泥"); err != nil {
		t.Fatalf("变更后查询: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	// 只应嵌入新条目正文（1 次）+ 3 个 kind 的 query = 4 次；旧文档不重嵌。
	if len(*recorded) != 4 {
		t.Fatalf("变更后嵌入次数 = %d, want 4（1 新文档 + 3 query）: %v", len(*recorded), *recorded)
	}
	foundNew := false
	for _, s := range *recorded {
		if strings.Contains(s, "水泥用量与标号关系") {
			foundNew = true
		}
		if strings.Contains(s, "振动锤选型需匹配地质") {
			t.Fatalf("未变更文档不应重嵌: %q", s)
		}
	}
	if !foundNew {
		t.Fatalf("新条目应被按需嵌入: %v", *recorded)
	}
}

// TestGaeaSemanticSearchEmbeddingUnavailable embedding 不可用：返回空不报错。
func TestGaeaSemanticSearchEmbeddingUnavailable(t *testing.T) {
	a, _, _ := countingEmbedEnv(t)
	SetAppEmbedderForTest(nil)

	hits, err := a.GaeaSemanticSearch("振动锤")
	if err != nil {
		t.Fatalf("embedding 不可用不应报错: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("embedding 不可用应返回空: %+v", hits)
	}
}

// TestGaeaSemanticSearchEmptyQuery 空查询直接返回空。
func TestGaeaSemanticSearchEmptyQuery(t *testing.T) {
	a, _, _ := countingEmbedEnv(t)
	hits, err := a.GaeaSemanticSearch("   ")
	if err != nil {
		t.Fatalf("空查询不应报错: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("空查询应返回空: %+v", hits)
	}
}
