package ocr

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestClient_RecognizeImageFile(t *testing.T) {
	var got struct {
		Model       string `json:"model"`
		ImageBase64 string `json:"image_base64"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ocr" {
			t.Errorf("path = %q, want /v1/ocr", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"text":"识别出的整页文本",
			"lines":[{"text":"第一行","score":0.98,"box":[[0,0],[10,0],[10,20],[0,20]]}],
			"image_width":640,
			"image_height":360,
			"elapsed_ms":1327
		}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	img := filepath.Join(dir, "sample.png")
	payload := []byte("fake-png-bytes")
	if err := os.WriteFile(img, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(srv.URL+"/v1", "paddleocr-ppocrv5-server")
	res, err := c.RecognizeImageFile(img)
	if err != nil {
		t.Fatalf("RecognizeImageFile: %v", err)
	}
	if res.Text != "识别出的整页文本" {
		t.Errorf("Text = %q", res.Text)
	}
	if len(res.Lines) != 1 || res.Lines[0].Text != "第一行" {
		t.Errorf("Lines = %+v", res.Lines)
	}
	if got.Model != "paddleocr-ppocrv5-server" {
		t.Errorf("model = %q", got.Model)
	}
	wantPrefix := "data:image/png;base64," + base64.StdEncoding.EncodeToString(payload)
	if got.ImageBase64 != wantPrefix {
		t.Errorf("image_base64 未按预期编码")
	}
}

func TestClient_ParseDocumentJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/documents/parse" {
			t.Errorf("path = %q, want /v1/documents/parse", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req["model"] != "minerU" || req["mode"] != "pipeline" {
			t.Errorf("请求参数不符: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"minerU",
			"text":"提取出的完整文档文本",
			"markdown":"提取出的完整文档文本",
			"pages":[{"page_number":1,"text":"单页提取文本"}],
			"metadata":{"page_count":1,"elapsed_ms":1520,"ocr_enabled":true}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", DefaultOCRModel)
	res, err := c.ParseDocument(ParseOptions{Model: "minerU", Path: "C:\\tmp\\sample.pdf"})
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if res.Text != "提取出的完整文档文本" || res.Metadata.PageCount != 1 {
		t.Errorf("解析结果不符: %+v", res)
	}
}

func TestClient_ParseDocumentText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("# 文档标题\n\n正文内容"))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", DefaultOCRModel)
	res, err := c.ParseDocument(ParseOptions{Model: "minerU", Path: "sample.docx", Format: "markdown"})
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if res.Text != "# 文档标题\n\n正文内容" {
		t.Errorf("Text = %q", res.Text)
	}
}
