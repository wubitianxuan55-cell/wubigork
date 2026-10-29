package app

// gaea_ocr_chain_test.go — 审计 U54/IN3-09 钉子：GaeaOCRText 腿链编排顺序与
// 两链配置口径互不越界。任何「把 herdsman 接进 docmd OCRProvider seam / 删
// 腿链串联」的重构碰坏其中一条即红（必须回 docmd/ocr.go seam 注释的口径边界
// 段重新论证分工后再动）。2026-10-29 增钉：腿 2~4 相对顺序默认 local-first
// （同图三引擎实测拍板：uocr2 质量全面第一），GAEA_OCR_LEG_ORDER=herdsman-first
// 保旧序，未知值 fail-closed。全部 httptest 假服务，无真机依赖。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
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

// killLocalOCRLeg 让本地腿 fail-closed（未知 GAEA_OCR_ENGINE kind 在
// ocrEngineOrder 即刻报错，不探活不拉起）——默认 local-first 序下本地腿先行，
// herdsman 腿测试必须杀本地腿，防真拉本机 uocr2/ovis 服务（8138 在跑时会真打）。
func killLocalOCRLeg(t *testing.T) {
	t.Helper()
	t.Setenv("GAEA_OCR_ENGINE", "paddle") // 远端名不是本地 kind → fail-closed
}

// newFakeUocr2 起一个假 Unlimited-OCR-2 llama 后端服务（GAEA_UOCR2_LLAMA_URL 注入
// docmd；2026-10-29 起默认 auto 腿序=llama 先行）：/health 应答 ok，
// /v1/chat/completions 返回 <PAGE>+BLOCK 标记包裹的固定文本并计数（清洗后即
// 预期文本）。resolveBase 探 /health 即成功并缓存，不触真实拉起。
func newFakeUocr2(t *testing.T, text string) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	content := "<PAGE><BLOCK>t[1, 2, 3, 4]<FORK>" + text
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/v1/chat/completions":
			atomic.AddInt32(&hits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":` + strconv.Quote(content) + `}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// TestGaeaOCRText_ActiveOCRBindingLeg1ParseRouting 腿 1（模型中心「设为 OCR」
// 绑定）优先且按模型名分流：绑定 minerU 模型 → /v1/documents/parse，不得先
// 打 /v1/ocr。
func TestGaeaOCRText_ActiveOCRBindingLeg1ParseRouting(t *testing.T) {
	clearHerdsmanOCREnv(t)
	killLocalOCRLeg(t) // 腿 1 命中即返回；防御：万一穿透也不真拉本地服务
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
// GAEA_OCR_ENGINE 只管本地腿的引擎选择，不得劫持腿序：显式 GAEA_OCR_ENGINE=tesseract
// 时本地腿先行但假图必失败（tesseract 未装/坏图均报错），穿透到 herdsman 腿
// 并返回其文本。
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
// 时，/v1/ocr 挑不到 OCR 模型 → 文档解析兜底（PaddleOCR 先于 MinerU）。默认
// local-first 序下本地腿先行，杀本地腿后才轮到 herdsman 两腿。
func TestGaeaOCRText_ParseLegAfterOCRModelMissing(t *testing.T) {
	clearHerdsmanOCREnv(t)
	killLocalOCRLeg(t) // 必须杀：否则本地腿先行，本机 8138 真服务会被打
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

// TestGaeaOCRText_LocalLegHonorsGAEA_OCR_ENGINE 本地腿透传 GAEA_OCR_ENGINE 到
// docmd seam：默认 local-first 序下本地腿先行，显式未知 kind fail-closed
// （docmd 报「未知 OCR 引擎」，app 侧逐腿收集加「本地 OCR」前缀，不静默降级），
// 穿透后 herdsman 腿错误一并聚合。
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
			t.Errorf("错误 %q 缺少 %q（本地腿应透传 GAEA_OCR_ENGINE 且逐腿收集错误）", err.Error(), want)
		}
	}
}

// TestGaeaOCRText_LocalLegFirstByDefault 默认腿序（2026-10-29 同图三引擎实测
// 拍板，质量优先）：本地链（uocr2 假服务经 GAEA_UOCR2_URL 注入，GAEA_OCR_ENGINE
// 保持 auto）先于 herdsman 两腿——本地腿命中即返回，herdsman 两端点不得被调。
func TestGaeaOCRText_LocalLegFirstByDefault(t *testing.T) {
	clearHerdsmanOCREnv(t)
	uocr2, uocr2Hits := newFakeUocr2(t, "uocr2 本地文本")
	t.Setenv("GAEA_UOCR2_LLAMA_URL", uocr2.URL)      // llama 主后端注入假服务
	t.Setenv("GAEA_UOCR2_URL", "http://127.0.0.1:1") // python 兜底侧死值：llama 成功即不应触达
	t.Setenv("GAEA_UOCR2_LLAMA_DIR", filepath.Join(t.TempDir(), "no-such-llama"))

	var ocrHits, parseHits int
	herd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/ocr":
			ocrHits++
		case "/v1/documents/parse":
			parseHits++
		}
		http.NotFound(w, r)
	}))
	defer herd.Close()

	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		BaseURL: herd.URL + "/v1",
		Enabled: true,
		Models:  []modelengine.ModelInfo{{ID: "paddleocr-ppocrv5-server"}, {ID: "minerU"}},
	}); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{engineMgr: mgr}}

	got, err := a.GaeaOCRText(newOCRTestImage(t))
	if err != nil {
		t.Fatalf("GaeaOCRText: %v", err)
	}
	if got != "uocr2 本地文本" {
		t.Errorf("got %q, want uocr2 本地文本", got)
	}
	if atomic.LoadInt32(uocr2Hits) != 1 {
		t.Errorf("uocr2 假服务命中 %d 次, want 1", atomic.LoadInt32(uocr2Hits))
	}
	if ocrHits != 0 || parseHits != 0 {
		t.Errorf("本地腿先行命中后 herdsman 不得被调（ocr=%d parse=%d）", ocrHits, parseHits)
	}
}

// TestGaeaOCRText_HerdsmanFirstOptIn GAEA_OCR_LEG_ORDER=herdsman-first 保旧序：
// 即使本地 uocr2 可用（假服务活着），herdsman /v1/ocr 先行命中，本地腿不得被调
// ——开关真实移动腿序，而非仅改文档。
func TestGaeaOCRText_HerdsmanFirstOptIn(t *testing.T) {
	clearHerdsmanOCREnv(t)
	t.Setenv("GAEA_OCR_LEG_ORDER", "herdsman-first")
	uocr2, uocr2Hits := newFakeUocr2(t, "uocr2 本地文本")
	t.Setenv("GAEA_UOCR2_LLAMA_URL", uocr2.URL)
	t.Setenv("GAEA_UOCR2_URL", "http://127.0.0.1:1")

	var ocrHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ocr" {
			http.NotFound(w, r)
			return
		}
		ocrHits++
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
	a := &App{core: &core{engineMgr: mgr}}

	got, err := a.GaeaOCRText(newOCRTestImage(t))
	if err != nil {
		t.Fatalf("GaeaOCRText: %v", err)
	}
	if got != "Paddle 文本" {
		t.Errorf("got %q, want Paddle 文本", got)
	}
	if ocrHits != 1 {
		t.Errorf("/v1/ocr 命中 %d 次, want 1", ocrHits)
	}
	if atomic.LoadInt32(uocr2Hits) != 0 {
		t.Errorf("herdsman-first 序下本地腿不得被调（uocr2 命中 %d 次）", atomic.LoadInt32(uocr2Hits))
	}
}

// TestGaeaOCRText_LegOrderFailClosed 未知 GAEA_OCR_LEG_ORDER fail-closed：
// 立即报错且不打任何腿（herdsman 假服务零命中）。
func TestGaeaOCRText_LegOrderFailClosed(t *testing.T) {
	clearHerdsmanOCREnv(t)
	t.Setenv("GAEA_OCR_LEG_ORDER", "bogus-order")

	var ocrHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/ocr" {
			ocrHits++
		}
		http.NotFound(w, r)
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
	a := &App{core: &core{engineMgr: mgr}}

	_, err := a.GaeaOCRText(newOCRTestImage(t))
	if err == nil {
		t.Fatal("未知腿序值应 fail-closed 报错")
	}
	for _, want := range []string{"GAEA_OCR_LEG_ORDER", "bogus-order"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误 %q 缺少 %q", err.Error(), want)
		}
	}
	if ocrHits != 0 {
		t.Errorf("fail-closed 前不得打任何腿（/v1/ocr 命中 %d 次）", ocrHits)
	}
}
