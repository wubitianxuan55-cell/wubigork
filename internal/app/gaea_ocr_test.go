package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

func TestGaeaOCRText_UsesHerdsmanPaddleOCR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ocr" {
			t.Errorf("path = %q, want /v1/ocr", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req["model"] != "paddleocr-ppocrv5-server" {
			t.Errorf("model = %v", req["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"Herdsman OCR 文本"}`))
	}))
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "sample.png")
	if err := os.WriteFile(img, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		BaseURL: srv.URL + "/v1",
		Enabled: true,
		Models:  []modelengine.ModelInfo{{ID: "paddleocr-ppocrv5-server"}},
	}); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{engineMgr: mgr}}

	got, err := a.GaeaOCRText(img)
	if err != nil {
		t.Fatalf("GaeaOCRText: %v", err)
	}
	if got != "Herdsman OCR 文本" {
		t.Errorf("got %q", got)
	}
}
