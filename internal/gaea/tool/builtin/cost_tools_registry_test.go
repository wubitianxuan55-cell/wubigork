package builtin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaea/gaea/internal/gaea/retrieval"
)

// saveRetrievalRuntimeGlobal 快照并恢复包级 retrievalRuntime（config 注入）。
func saveRetrievalRuntimeGlobal(t *testing.T) {
	t.Helper()
	old := retrievalRuntime
	t.Cleanup(func() { retrievalRuntime = old })
}

// TestCostEmbedderReranker_ConfigRouting 3.0 Step 3d #2/#3：embed/rerank 后端
// 由 SetRetrievalRuntime 注入的 config 路由（kind/base/model），不再读
// HERDSMAN_BASE_URL 环境变量；切换后端只改配置、消费方零改动。
func TestCostEmbedderReranker_ConfigRouting(t *testing.T) {
	saveRetrievalRuntimeGlobal(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-m3"},{"id":"bge-reranker-v2-m3"}]}`))
		case "/v1/embeddings":
			_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0,0]}]}`))
		case "/v1/rerank":
			_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// 未注入 config 时：默认 kind=openai + localhost:8080 + 各自默认模型。
	e := costEmbedder()
	if e == nil {
		t.Fatal("默认 config 下 costEmbedder 不应为 nil")
	}
	if e.BaseURL != "http://localhost:8080/v1" || e.Model != "bge-m3" {
		t.Errorf("默认 embedder = %s/%s, want localhost:8080/v1 + bge-m3", e.BaseURL, e.Model)
	}
	r0 := costReranker()
	if r0 == nil {
		t.Fatal("默认 config 下 costReranker 不应为 nil")
	}
	if r0.BaseURL != "http://localhost:8080/v1" || r0.Model != "bge-reranker-v2-m3" {
		t.Errorf("默认 reranker = %s/%s, want localhost:8080/v1 + bge-reranker-v2-m3", r0.BaseURL, r0.Model)
	}

	// 注入 config：kind/base/model 全部生效（指向测试服务器）。
	SetRetrievalRuntime(RetrievalRuntime{
		EmbedKind:     retrieval.EmbedderKindOpenAI,
		EmbedBaseURL:  srv.URL,
		EmbedModel:    "bge-m3",
		RerankKind:    retrieval.RerankerKindOpenAI,
		RerankBaseURL: srv.URL,
		RerankModel:   "bge-reranker-v2-m3",
	})
	e2 := costEmbedder()
	if e2 == nil || e2.BaseURL != srv.URL+"/v1" {
		t.Fatalf("config 路由 embedder = %+v, want base %s", e2, srv.URL+"/v1")
	}
	r2 := costReranker()
	if r2 == nil || r2.BaseURL != srv.URL+"/v1" {
		t.Fatalf("config 路由 reranker = %+v, want base %s", r2, srv.URL+"/v1")
	}
	// 端到端：路由后的客户端可用且可调用（对测试服务器）。
	ctx := t.Context()
	if !e2.Available(ctx) {
		t.Error("config 路由的 embedder 应可用")
	}
	if !r2.Available(ctx) {
		t.Error("config 路由的 reranker 应可用")
	}
	vecs, err := e2.Embed(ctx, []string{"测试"})
	if err != nil || len(vecs) != 1 {
		t.Errorf("config 路由 embed 失败: %v %v", vecs, err)
	}
	scored, err := r2.Rerank(ctx, "q", []string{"d1"}, 1)
	if err != nil || len(scored) != 1 {
		t.Errorf("config 路由 rerank 失败: %v %v", scored, err)
	}
}

// TestCostEmbedder_UnknownKindFailsClosed 未知 kind fail-closed：
// 返回 nil（调用方按服务不可用回退 SQL），不静默降级到其他后端。
func TestCostEmbedder_UnknownKindFailsClosed(t *testing.T) {
	saveRetrievalRuntimeGlobal(t)
	SetRetrievalRuntime(RetrievalRuntime{EmbedKind: "no-such-embedder"})
	if e := costEmbedder(); e != nil {
		t.Fatalf("未知 kind 应返回 nil（fail-closed），got %+v", e)
	}
	SetRetrievalRuntime(RetrievalRuntime{RerankKind: "no-such-reranker"})
	if r := costReranker(); r != nil {
		t.Fatalf("未知 kind 应返回 nil（fail-closed），got %+v", r)
	}
}
