package app

// gaea_ocr_chain_test.go — 审计 U54/IN3-09 钉子：GaeaOCRText 四腿编排顺序与
// 两链配置口径互不越界。任何「把 herdsman 接进 docmd OCRProvider seam / 删
// 四腿串联」的重构碰坏其中一条即红（必须回 docmd/ocr.go seam 注释的口径边界
// 段重新论证分工后再动）。全部 httptest 假服务，无真机依赖。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// newOCRTestImage 写一张假图片文件（GaeaOCRText 只做 os.Stat，不解码）。
func newOCRTestImage(t *testing.T) string {
	t.Helper()
	img := filepath.Join(t.TempDir(), "sample.png")
	if err := os.WriteFile(img, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	return img
}

// clearHerdsmanOCREnv 清空 HERDSMAN_* env，保证模型选择走引擎模型列表而非
// 开发机环境（hermetic）。
func clearHerdsmanOCREnv(t *testing.T) {
	t.Helper()
	t.Setenv("HERDSMAN_OCR_MODEL", "")
	t.Setenv("HERDSMAN_PARSE_MODEL", "")
	t.Setenv("HERDSMAN_PARSE_MODE", "")
}

// TestGaeaOCRText_ActiveOCRBindingLeg1ParseRouting 腿 1（模型中心「设为 OCR」
// 绑定）优先且按模型名分流：绑定 minerU 模型 → /v1/documents/parse，不得先
// 打 /v1/ocr。
func TestGaeaOCRText_ActiveOCRBindingLeg1ParseRouting(t *testing.T) {
	clearHerdsmanOCREnv(t)
	var ocrHits, parseHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/ocr":
			ocrHits++
			http.NotFound(w, r)
		case "/v1/documents/parse":
			parseHits++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"model":"minerU","text":"绑定 MinerU 解析文本","markdown":"绑定 MinerU 解析文本"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		BaseURL: srv.URL + "/v1",
		Enabled: true,
		Models:  []modelengine.ModelInfo{{ID: "paddleocr-ppocrv5-server"}, {ID: "minerU"}},
	}); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{engineMgr: mgr, activeOCREngine: "herdsman", activeOCRModel: "minerU"}}

	got, err := a.GaeaOCRText(newOCRTestImage(t))
	if err != nil {
		t.Fatalf("GaeaOCRText: %v", err)
	}
	if got != "绑定 MinerU 解析文本" {
		t.Errorf("got %q, want 绑定 MinerU 解析文本", got)
	}
	if parseHits != 1 || ocrHits != 0 {
		t.Errorf("腿 1 应按模型名走 parse 且不先打 /v1/ocr（parse=%d ocr=%d）", parseHits, ocrHits)
	}
}

// TestGaeaOCRText_GAEA_OCR_ENGINEDoesNotHijackHerdsmanLegs docmd 的
// GAEA_OCR_ENGINE 只管第四腿（本地链），不得劫持 herdsman 腿：显式
// GAEA_OCR_ENGINE=tesseract 时，herdsman 可用仍应走 /v1/ocr 并返回其文本。
func TestGaeaOCRText_GAEA_OCR_ENGINEDoesNotHijackHerdsmanLegs(t *testing.T) {
	clearHerdsmanOCREnv(t)
	t.Setenv("GAEA_OCR_ENGINE", "tesseract")

	var ocrHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ocr" {
			http.NotFound(w, r)
			return
		}
		ocrHits++
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"Paddle 文本"}`))
	}))
	defer srv.Close()

	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		BaseURL: srv.URL + "/v1",
		Enabled: true,
		Models:  []modelengine.ModelInfo{{ID: "paddleocr-ppocrv5-server"}},
	}); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{engineMgr: mgr}} // 无 activeOCR 绑定

	got, err := a.GaeaOCRText(newOCRTestImage(t))
	if err != nil {
		t.Fatalf("GaeaOCRText: %v", err)
	}
	if got != "Paddle 文本" {
		t.Errorf("got %q, want Paddle 文本（GAEA_OCR_ENGINE 不应劫持远端腿）", got)
	}
	if ocrHits != 1 {
		t.Errorf("/v1/ocr 命中 %d 次, want 1", ocrHits)
	}
}

// TestGaeaOCRText_ParseLegAfterOCRModelMissing 腿内顺序：引擎只有 MinerU 模型
// 时，腿 2 挑不到 OCR 模型 → 腿 3 文档解析兜底（PaddleOCR 先于 MinerU）。
func TestGaeaOCRText_ParseLegAfterOCRModelMissing(t *testing.T) {
	clearHerdsmanOCREnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/documents/parse" {
			t.Errorf("path = %q, want /v1/documents/parse（/v1/ocr 无 OCR 模型不应被调用）", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"minerU","text":"MinerU 兜底文本","markdown":"MinerU 兜底文本"}`))
	}))
	defer srv.Close()

	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		BaseURL: srv.URL + "/v1",
		Enabled: true,
		Models:  []modelengine.ModelInfo{{ID: "minerU"}},
	}); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{engineMgr: mgr}}

	got, err := a.GaeaOCRText(newOCRTestImage(t))
	if err != nil {
		t.Fatalf("GaeaOCRText: %v", err)
	}
	if got != "MinerU 兜底文本" {
		t.Errorf("got %q, want MinerU 兜底文本", got)
	}
}

// TestGaeaOCRText_LocalLegHonorsGAEA_OCR_ENGINE 第四腿透传 GAEA_OCR_ENGINE 到
// docmd seam：herdsman 引擎不存在时落到本地链，显式未知 kind fail-closed
// （docmd 报「未知 OCR 引擎」，app 侧逐腿收集加「本地 OCR」前缀，不静默降级）。
func TestGaeaOCRText_LocalLegHonorsGAEA_OCR_ENGINE(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", "paddle") // 远端名不是本地 kind → fail-closed

	mgr := modelengine.NewManager("", "") // 无 herdsman 引擎：腿 2/3 报「Herdsman 引擎未启用」
	a := &App{core: &core{engineMgr: mgr}}

	_, err := a.GaeaOCRText(newOCRTestImage(t))
	if err == nil {
		t.Fatal("全链不可用应报错")
	}
	for _, want := range []string{"本地 OCR", "未知 OCR 引擎", "paddle"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误 %q 缺少 %q（第四腿应透传 GAEA_OCR_ENGINE 且逐腿收集错误）", err.Error(), want)
		}
	}
}
