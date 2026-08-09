package docmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOvisServerClient 用 httptest 校验健康检查与图片 OCR 请求/解析。
func TestOvisServerClient(t *testing.T) {
	var gotBody map[string]any
	var gotImageData bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Write([]byte(`{"status":"ok"}`))
		case "/v1/chat/completions":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			content, _ := json.Marshal(gotBody["messages"])
			gotImageData = strings.Contains(string(content), "data:image/png;base64,")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"choices":[{"message":{"content":"项目周报：营收 120 万元"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &http.Client{Timeout: 3 * time.Second}
	if !ovisServerHealthy(c, srv.URL) {
		t.Fatal("health should be ok")
	}
	png := filepath.Join(t.TempDir(), "p.png")
	if err := os.WriteFile(png, []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}
	text, err := ovisPageOCR(srv.URL, png)
	if err != nil {
		t.Fatalf("ovisPageOCR: %v", err)
	}
	if !strings.Contains(text, "项目周报") {
		t.Fatalf("unexpected OCR text: %q", text)
	}
	if !gotImageData {
		t.Fatal("request should carry base64 image data URI")
	}
	if _, ok := gotBody["max_tokens"]; !ok {
		t.Fatal("request should set max_tokens")
	}
}

// TestOvisServerHealthyNegative 服务不可达时健康检查应快速失败。
func TestOvisServerHealthyNegative(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &http.Client{Timeout: 2 * time.Second}
	if ovisServerHealthy(c, srv.URL) {
		t.Fatal("non-200 should be unhealthy")
	}
	if ovisServerHealthy(c, "http://127.0.0.1:1") {
		t.Fatal("unreachable should be unhealthy")
	}
}

// TestOCRPDFUsesOvisServer 端到端：真实扫描 PDF → pdftoppm 渲染 → 常驻 OvisOCR2。
// 依赖本机 OvisOCR2 服务（127.0.0.1:8137）与 pdftoppm；设置 GAEA_TEST_SCAN_PDF
// 指向扫描件 PDF 后运行。
func TestOCRPDFUsesOvisServer(t *testing.T) {
	pdf := os.Getenv("GAEA_TEST_SCAN_PDF")
	if pdf == "" {
		t.Skip("设置 GAEA_TEST_SCAN_PDF 指向扫描件 PDF 后运行端到端 OCR 测试")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm 不可用")
	}
	c := &http.Client{Timeout: 2 * time.Second}
	if !ovisServerHealthy(c, "http://127.0.0.1:8137") {
		t.Skip("OvisOCR2 服务未运行")
	}
	md, err := Convert(pdf, "")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.Contains(md, "项目周报") || !strings.Contains(md, "120") {
		t.Fatalf("OCR 结果缺少预期内容:\n%s", md)
	}
}

// TestOCRImageTextE2E 端到端：单张图片 → 常驻 OvisOCR2 服务提取文字。
// 依赖本机 OvisOCR2 服务；设置 GAEA_TEST_OCR_IMAGE 指向含文字的图片后运行。
func TestOCRImageTextE2E(t *testing.T) {
	img := os.Getenv("GAEA_TEST_OCR_IMAGE")
	if img == "" {
		t.Skip("设置 GAEA_TEST_OCR_IMAGE 指向含文字的图片后运行")
	}
	c := &http.Client{Timeout: 2 * time.Second}
	if !ovisServerHealthy(c, "http://127.0.0.1:8137") {
		t.Skip("OvisOCR2 服务未运行")
	}
	text, err := OCRImageText(img)
	if err != nil {
		t.Fatalf("OCRImageText: %v", err)
	}
	if !strings.Contains(text, "项目周报") || !strings.Contains(text, "120") {
		t.Fatalf("OCR 结果缺少预期内容: %q", text)
	}
}
