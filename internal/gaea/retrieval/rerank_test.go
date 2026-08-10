package retrieval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestRerankServer(t *testing.T, order []int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"bge-reranker-v2-m3"},{"id":"other"}]}`))
		case "/v1/rerank":
			var req struct {
				Model     string   `json:"model"`
				Query     string   `json:"query"`
				Documents []string `json:"documents"`
				TopN      int      `json:"top_n"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Model != "bge-reranker-v2-m3" || len(req.Documents) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			results := make([]map[string]any, 0, len(order))
			for i, idx := range order {
				if i >= req.TopN {
					break
				}
				results = append(results, map[string]any{
					"index": idx, "relevance_score": 10 - float64(i),
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRerankerAvailableAndRerank(t *testing.T) {
	srv := newTestRerankServer(t, []int{2, 0, 1})
	defer srv.Close()
	r := New(srv.URL, "bge-reranker-v2-m3")
	ctx := context.Background()
	if !r.Available(ctx) {
		t.Fatal("model should be available")
	}
	docs := []string{"水泥", "钢筋", "液压振动锤"}
	got, err := r.Rerank(ctx, "振动锤", docs, 2)
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}
	if len(got) != 2 || got[0].Index != 2 || got[0].Content != "液压振动锤" || got[1].Index != 0 {
		t.Errorf("rerank result wrong: %+v", got)
	}
}

func TestRerankerUnavailable(t *testing.T) {
	r := New("http://127.0.0.1:1", "bge-reranker-v2-m3") // 连接拒绝
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	if r.Available(ctx) {
		t.Error("unreachable server should not be available")
	}
}

func TestRerankHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	r := New(srv.URL, "bge-reranker-v2-m3")
	_, err := r.Rerank(context.Background(), "q", []string{"a", "b"}, 2)
	if err == nil {
		t.Error("expected error on 500")
	}
}
