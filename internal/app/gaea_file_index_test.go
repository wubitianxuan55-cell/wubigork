package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

func TestGaeaFileIndexRebuildAndSearch(t *testing.T) {
	root := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	_ = os.WriteFile(filepath.Join(root, "说明.md"), []byte("振动锤选型要点 桩基施工"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "水泥.md"), []byte("P.O 42.5 水泥 吨 480"), 0o644)

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
			if strings.Contains(s, "锤") || strings.Contains(s, "桩") {
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

	dbDir := t.TempDir()
	gdb := db.GetDatabase(dbDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(dbDir)
		SetAppSemanticStoreForTest(nil)
		SetAppEmbedderForTest(nil)
	})
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	SetAppEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))

	a := &App{}
	st, err := a.GaeaFileIndexRebuild()
	if err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}
	if st.Total != 2 {
		t.Fatalf("indexed %d, want 2", st.Total)
	}

	hits, err := a.GaeaFileSemanticSearch("打桩锤", 5)
	if err != nil {
		t.Fatalf("semantic search failed: %v", err)
	}
	if len(hits) != 1 || !strings.Contains(hits[0].Path, "说明.md") {
		t.Errorf("hits = %+v, want 说明.md", hits)
	}
}
