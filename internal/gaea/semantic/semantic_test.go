package semantic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return Open(gdb)
}

func fakeEmbedServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-m3"}]}`))
		case "/v1/embeddings":
			var req struct {
				Input []string `json:"input"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			data := make([]map[string]any, 0, len(req.Input))
			for i, s := range req.Input {
				vec := []float32{0, 0, 0}
				if strings.Contains(s, "锤") || strings.Contains(s, "液压") {
					vec[0] = 1
				}
				if strings.Contains(s, "水泥") {
					vec[1] = 1
				}
				if strings.Contains(s, "挖掘机") {
					vec[2] = 1
				}
				data = append(data, map[string]any{"index": i, "embedding": vec})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestEnsureIncrementalAndSearch(t *testing.T) {
	srv := fakeEmbedServer(t)
	defer srv.Close()
	e := retrieval.NewEmbedder(srv.URL, "bge-m3")
	ctx := context.Background()
	st := newTestStore(t)

	docs := []Doc{
		{ID: "hp300", Text: "HP300 高频液压振动锤 台班 3200"},
		{ID: "cement", Text: "P.O 42.5 水泥 吨 480"},
	}
	n, err := st.Ensure(ctx, e, "cost", docs)
	if err != nil || n != 2 {
		t.Fatalf("Ensure = %d, %v; want 2", n, err)
	}
	// 再次 Ensure：已存在 → 不重复向量化。
	n, _ = st.Ensure(ctx, e, "cost", docs)
	if n != 0 {
		t.Errorf("Ensure should be incremental, added %d", n)
	}

	hits, err := st.Search(ctx, e, "cost", docs, "打桩锤", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != "hp300" {
		t.Errorf("search = %+v, want hp300", hits)
	}
}

func TestSearchManyMergesKinds(t *testing.T) {
	srv := fakeEmbedServer(t)
	defer srv.Close()
	e := retrieval.NewEmbedder(srv.URL, "bge-m3")
	ctx := context.Background()
	st := newTestStore(t)

	kindDocs := map[string][]Doc{
		"cost":      {{ID: "hp300", Text: "HP300 高频液压振动锤 台班 3200"}},
		"knowledge": {{ID: "桩基-要点", Text: "桩基施工要点 振动锤选型"}},
		"office":    {{ID: "fact-1", Text: "甲方偏好保守报价"}},
	}
	hits, err := st.SearchMany(ctx, e, kindDocs, "打桩锤", 2, 5)
	if err != nil {
		t.Fatalf("SearchMany failed: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 semantic hits, got %+v", hits)
	}
	kinds := map[string]bool{}
	for _, h := range hits {
		kinds[h.Kind] = true
	}
	if !kinds["cost"] || !kinds["knowledge"] {
		t.Errorf("should merge cost+knowledge, got %+v", hits)
	}
}

func TestStale(t *testing.T) {
	srv := fakeEmbedServer(t)
	defer srv.Close()
	e := retrieval.NewEmbedder(srv.URL, "bge-m3")
	ctx := context.Background()
	st := newTestStore(t)
	docs := []Doc{
		{ID: "a", Text: "液压振动锤"},
		{ID: "b", Text: "水泥"},
	}
	if _, err := st.Ensure(ctx, e, "cost", docs); err != nil {
		t.Fatal(err)
	}
	del, err := st.Stale("cost", map[string]bool{"a": true})
	if err != nil || del != 1 {
		t.Fatalf("Stale = %d, %v; want 1", del, err)
	}
	hits, _ := st.SearchReady(ctx, e, "cost", "液压", 5)
	if len(hits) != 1 || hits[0].ID != "a" {
		t.Errorf("after stale, search = %+v", hits)
	}
}

func TestEnsureRefreshesChangedDoc(t *testing.T) {
	srv := fakeEmbedServer(t)
	defer srv.Close()
	e := retrieval.NewEmbedder(srv.URL, "bge-m3")
	ctx := context.Background()
	st := newTestStore(t)

	docs := []Doc{{ID: "a", Text: "液压振动锤 台班 3200"}}
	if n, err := st.Ensure(ctx, e, "cost", docs); err != nil || n != 1 {
		t.Fatalf("first Ensure = %d, %v; want 1", n, err)
	}
	// 内容变化 → 重新向量化（count 1），查询能命中新内容。
	docs[0].Text = "挖掘机 台班 2600"
	if n, err := st.Ensure(ctx, e, "cost", docs); err != nil || n != 1 {
		t.Fatalf("refresh Ensure = %d, %v; want 1", n, err)
	}
	hits, err := st.SearchReady(ctx, e, "cost", "挖掘机", 5)
	if err != nil || len(hits) != 1 || hits[0].ID != "a" {
		t.Errorf("search after refresh = %+v, err=%v", hits, err)
	}
}
