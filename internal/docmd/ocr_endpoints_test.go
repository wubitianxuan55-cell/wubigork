package docmd

// ocr_endpoints_test.go — 本地服务被动观测面与显式引擎入口（模型中心配套刀）。
import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalServiceHealth(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer up.Close()
	if !LocalServiceHealth(up.URL) {
		t.Fatal("/health ok 应判就绪")
	}
	// 非 /health（404）与无应答 → false。
	if LocalServiceHealth(up.URL + "/other") {
		t.Fatal("404 不应判就绪")
	}
	if LocalServiceHealth("") {
		t.Fatal("空地址应 false")
	}
	if LocalServiceHealth("http://127.0.0.1:1") {
		t.Fatal("拒连端口应 false")
	}
}

func TestOCRImageTextWithEngine_UnknownKind(t *testing.T) {
	_, err := OCRImageTextWithEngine("whatever.png", "bogus-kind")
	if err == nil || !strings.Contains(err.Error(), "未知本地 OCR 引擎") {
		t.Fatalf("未知 kind 必须 fail-closed: %v", err)
	}
}

func TestOCRImageTextWithEngine_FailClosed(t *testing.T) {
	// 未知 kind 已单测；此处验显式 kind 不可用时 fail-closed（错误非空文本）。
	// 用替身引擎：uocr2 的 resolve 路径会「按需拉起服务」，单元测试不得触发真实
	// 进程（拉起行为由 live 门控测试覆盖）。
	useFakeOCR(t, "", errors.New("替身引擎不可用"))
	if _, err := OCRImageTextWithEngine("whatever.png", ocrTestKind); err == nil ||
		!strings.Contains(err.Error(), ocrTestKind) {
		t.Fatalf("显式引擎不可用必须 fail-closed 并说明引擎: %v", err)
	}
}

// 已退役引擎（旧配置残留）：如实报「已退役」并指路，不静默换挡。
func TestOCRImageTextWithEngine_RetiredKind(t *testing.T) {
	runs := useFakeOCR(t, "不应被使用", nil)
	for _, retired := range []string{"ovis", "tesseract"} {
		if _, err := OCRImageTextWithEngine("whatever.png", retired); err == nil ||
			!strings.Contains(err.Error(), "退役") || !strings.Contains(err.Error(), "uocr2") {
			t.Fatalf("已退役引擎 %s 应报错并指路 uocr2: %v", retired, err)
		}
	}
	if *runs != 0 {
		t.Fatalf("已退役引擎不得落到替身引擎（runs=%d）", *runs)
	}
}
