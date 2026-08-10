package retrieval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
)

// fakeEmbedServer 按文档是否含「振动锤/水泥」返回 1-hot 向量，模拟 bge-m3。
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
				if strings.Contains(s, "振动锤") || strings.Contains(s, "液压") {
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
			_ = json.NewEncoder(w).Encode(map[string]any{"model": "bge-m3", "data": data})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestEmbedderEmbedAndCosine(t *testing.T) {
	srv := fakeEmbedServer(t)
	defer srv.Close()
	e := NewEmbedder(srv.URL, "bge-m3")
	ctx := context.Background()
	if !e.Available(ctx) {
		t.Fatal("embedder should be available")
	}
	vecs, err := e.Embed(ctx, []string{"液压振动锤 台班", "水泥 吨"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vecs) != 2 || len(vecs[0]) != 3 {
		t.Fatalf("vecs = %+v", vecs)
	}
	if c := Cosine(vecs[0], vecs[0]); c < 0.99 {
		t.Errorf("self cosine = %v, want ~1", c)
	}
	if c := Cosine(vecs[0], vecs[1]); c != 0 {
		t.Errorf("orthogonal cosine = %v, want 0", c)
	}
}

func TestSemanticRecall(t *testing.T) {
	srv := fakeEmbedServer(t)
	defer srv.Close()
	e := NewEmbedder(srv.URL, "bge-m3")
	ctx := context.Background()
	full := []cost.Summary{
		{Name: "cement", Title: "P.O 42.5 水泥", Unit: "吨", Price: 480},
		{Name: "hp300", Title: "HP300 高频液压振动锤", Unit: "台班", Price: 3200},
	}
	got := SemanticRecall(ctx, e, "液压振动锤", full, nil, 1)
	if len(got) != 1 || got[0].Name != "hp300" {
		t.Errorf("semantic recall = %+v, want hp300", got)
	}
}

func TestSemanticRecallExcludesExisting(t *testing.T) {
	srv := fakeEmbedServer(t)
	defer srv.Close()
	e := NewEmbedder(srv.URL, "bge-m3")
	ctx := context.Background()
	full := []cost.Summary{
		{Name: "cement", Title: "P.O 42.5 水泥", Unit: "吨", Price: 480},
		{Name: "hp300", Title: "HP300 高频液压振动锤", Unit: "台班", Price: 3200},
	}
	// 已有 hp300（关键词命中），语义召回只补水泥之外的语义相近项。
	got := SemanticRecall(ctx, e, "液压振动锤", full, []cost.Summary{{Name: "hp300"}}, 5)
	if len(got) != 1 || got[0].Name != "hp300" {
		t.Errorf("exclude logic wrong: %+v", got)
	}
}

func TestEmbedderUnavailable(t *testing.T) {
	e := NewEmbedder("http://127.0.0.1:1", "bge-m3")
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	if e.Available(ctx) {
		t.Error("unreachable server should not be available")
	}
}
