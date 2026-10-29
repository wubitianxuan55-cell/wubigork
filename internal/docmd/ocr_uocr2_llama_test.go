package docmd

// ocr_uocr2_llama_test.go — uocr2 引擎 llama 后端（llama-server + Q8 GGUF）单测：
// chat 协议形状（两条铁律）、单/批识别解码与拆页、截断/页数不齐 fail-closed、
// 后端腿序（auto 偏好 llama / 回退 python / 显式指定 / 未知值 fail-closed）。
// 全部 httptest 假服务，不依赖真实 llama-server 或 python 服务。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeLlamaServer 起 httptest 服务模拟 llama-server：/health 探活 +
// /v1/chat/completions 返回注入应答并计数。
func fakeLlamaServer(t *testing.T, content, finishReason string, status int) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/v1/chat/completions":
			atomic.AddInt32(&hits, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			resp, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{
					"finish_reason": finishReason,
					"message":       map[string]string{"content": content},
				}},
			})
			_, _ = w.Write(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// TestUocr2LlamaChatContentShape 铁律一：content = [<PAGE> 文本, 图片]×N + 收尾
// 文本块（"Multi page parsing."）——最后一块必须是文本，图片永远不收尾。
func TestUocr2LlamaChatContentShape(t *testing.T) {
	content := uocr2LlamaChatContent([]string{"AAA", "BBB"})
	if len(content) != 5 { // 2×(页标+图) + 1 收尾
		t.Fatalf("content 应 5 块，got %d", len(content))
	}
	if content[0].Type != "text" || content[0].Text != "<PAGE>" {
		t.Errorf("首块应为 <PAGE> 文本: %+v", content[0])
	}
	if content[1].Type != "image_url" || !strings.HasPrefix(content[1].ImageURL.URL, "data:image/png;base64,AAA") {
		t.Errorf("第二块应为图片: %+v", content[1])
	}
	last := content[len(content)-1]
	if last.Type != "text" || last.Text != uocr2LlamaMultiPagePrompt {
		t.Fatalf("末块必须是收尾文本块（铁律一），got %+v", last)
	}
	if content[len(content)-2].Type != "image_url" {
		t.Errorf("倒数第二块应为图片: %+v", content[len(content)-2])
	}
}

// TestBuildUocr2LlamaServerCmd 铁律二：启动参数必须含 --ubatch-size 4096；工件
// 缺失返回 ok=false。
func TestBuildUocr2LlamaServerCmd(t *testing.T) {
	t.Setenv("GAEA_UOCR2_LLAMA_DIR", t.TempDir()) // 空目录：工件缺失
	if _, _, ok := buildUocr2LlamaServerCmd(); ok {
		t.Fatal("工件缺失应返回 ok=false")
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		filepath.Join(dir, "bin", "llama-server.exe"),
		filepath.Join(dir, "uocr2-Q8_0.gguf"),
		filepath.Join(dir, "mmproj-uocr2-F16.gguf"),
	} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GAEA_UOCR2_LLAMA_DIR", dir)
	cmd, logf, ok := buildUocr2LlamaServerCmd()
	if !ok || cmd == nil {
		t.Fatal("工件齐全应返回 ok=true")
	}
	if logf != nil { // Windows 开着的文件删不掉：必须关闭否则 TempDir 清理假红
		t.Cleanup(func() { _ = logf.Close() })
	}
	joined := strings.Join(cmd.Args, " ")
	for _, want := range []string{
		"--ubatch-size 4096", // 铁律二：多图丢图根修，勿删
		"--mmproj", "-c 65536", "-np 1", "--temp 0", "--top-k 1",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("启动参数缺少 %q: %s", want, joined)
		}
	}
}

// TestUocr2LlamaPageOCRDecodesAndCleans 单页：chat 应答原始标记输出 → 清洗。
func TestUocr2LlamaPageOCRDecodesAndCleans(t *testing.T) {
	srv, _ := fakeLlamaServer(t,
		"<PAGE><BLOCK>table[1, 2, 3, 4]<FORK>水温\n<BLOCK>header[5, 6, 7, 8]<FORK>pH 值",
		"stop", http.StatusOK)
	got, err := uocr2LlamaPageOCR(srv.URL, mustTempPNG(t))
	if err != nil {
		t.Fatalf("uocr2LlamaPageOCR: %v", err)
	}
	if got != "水温\npH 值" {
		t.Errorf("text = %q, want 水温\\npH 值", got)
	}
}

// TestUocr2LlamaPagesOCRSplitsAndCleans 多页：输出按 <PAGE> 拆分与输入等长同序。
func TestUocr2LlamaPagesOCRSplitsAndCleans(t *testing.T) {
	srv, _ := fakeLlamaServer(t,
		"<PAGE><BLOCK>table[1, 2, 3, 4]<FORK>第一页表格<PAGE><BLOCK>header[5, 6, 7, 8]<FORK>第二页标题",
		"stop", http.StatusOK)
	got, err := uocr2LlamaPagesOCR(srv.URL, []string{mustTempPNG(t), mustTempPNG(t)})
	if err != nil {
		t.Fatalf("uocr2LlamaPagesOCR: %v", err)
	}
	if len(got) != 2 || got[0] != "第一页表格" || got[1] != "第二页标题" {
		t.Errorf("应与输入等长同序且已清洗: %+v", got)
	}
}

// TestUocr2LlamaPagesOCRCountMismatchFailsClosed 页数不齐 fail-closed。
func TestUocr2LlamaPagesOCRCountMismatchFailsClosed(t *testing.T) {
	srv, _ := fakeLlamaServer(t, "<PAGE>只有一页", "stop", http.StatusOK)
	_, err := uocr2LlamaPagesOCR(srv.URL, []string{mustTempPNG(t), mustTempPNG(t)})
	if err == nil || !strings.Contains(err.Error(), "页数不齐") {
		t.Fatalf("页数不齐应 fail-closed: %v", err)
	}
}

// TestUocr2LlamaPagesOCRLeadingTextFailsClosed 首段非空=结构异常 fail-closed。
func TestUocr2LlamaPagesOCRLeadingTextFailsClosed(t *testing.T) {
	srv, _ := fakeLlamaServer(t, "前言文字<PAGE>第一页", "stop", http.StatusOK)
	_, err := uocr2LlamaPagesOCR(srv.URL, []string{mustTempPNG(t)})
	if err == nil || !strings.Contains(err.Error(), "首段非空") {
		t.Fatalf("首段非空应 fail-closed: %v", err)
	}
}

// TestUocr2LlamaTruncationFailsClosed finish_reason=length 截断 fail-closed
// （对齐 python 后端 422 口径）。
func TestUocr2LlamaTruncationFailsClosed(t *testing.T) {
	srv, _ := fakeLlamaServer(t, "<PAGE>残缺文本", "length", http.StatusOK)
	_, err := uocr2LlamaChat(srv.URL, []string{mustTempPNG(t)})
	if err == nil || !strings.Contains(err.Error(), "截断") {
		t.Fatalf("截断应 fail-closed: %v", err)
	}
}

// assertLlamaChatRequest 在假服务 handler 内联断言请求形状（铁律一）：
// content 末块为收尾文本、每图前有 <PAGE> 文本块、max_tokens=8000×页数。
func assertLlamaChatRequest(t *testing.T, req map[string]any, nPages int) {
	t.Helper()
	msgs, _ := req["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages 应 1 条, got %d", len(msgs))
	}
	msg := msgs[0].(map[string]any)
	content, _ := msg["content"].([]any)
	if len(content) != nPages*2+1 {
		t.Fatalf("content 应 %d 块（页标+图×%d+收尾）, got %d", nPages*2+1, nPages, len(content))
	}
	last := content[len(content)-1].(map[string]any)
	if last["type"] != "text" || last["text"] != uocr2LlamaMultiPagePrompt {
		t.Errorf("content 末块必须是收尾文本块（铁律一）: %v", last)
	}
	if mt, _ := req["max_tokens"].(float64); int(mt) != uocr2PageMaxTokens*nPages {
		t.Errorf("max_tokens = %v, want %d", req["max_tokens"], uocr2PageMaxTokens*nPages)
	}
}

// TestUocr2ProviderAutoPrefersLlama 默认腿序（auto）：llama 假服务健康即命中，
// python 端点不得被探活（llama 成功后短路）。
func TestUocr2ProviderAutoPrefersLlama(t *testing.T) {
	var llamaHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/v1/chat/completions":
			atomic.AddInt32(&llamaHits, 1)
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			assertLlamaChatRequest(t, req, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"<PAGE><BLOCK>t[1]<FORK>llama 文本"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GAEA_UOCR2_LLAMA_URL", srv.URL)
	t.Setenv("GAEA_UOCR2_URL", "http://127.0.0.1:1") // python 侧死值：llama 成功即不应触达

	p := &uocr2OCRProvider{}
	got, err := p.ExtractImage(mustTempPNG(t))
	if err != nil {
		t.Fatalf("ExtractImage: %v", err)
	}
	if got != "llama 文本" {
		t.Errorf("text = %q, want llama 文本", got)
	}
	if atomic.LoadInt32(&llamaHits) != 1 {
		t.Errorf("llama chat 应命中 1 次, got %d", llamaHits)
	}
	if p.backend != uocr2BackendLlama {
		t.Errorf("backend = %q, want llama", p.backend)
	}
}

// TestUocr2ProviderAutoFallsBackToPython auto 腿序：llama 探活失败+拉起工件缺失
// → 回退 python 后端（/ocr 协议）。
func TestUocr2ProviderAutoFallsBackToPython(t *testing.T) {
	t.Setenv("GAEA_UOCR2_LLAMA_URL", "http://127.0.0.1:1")                        // 死端口：探活必失败
	t.Setenv("GAEA_UOCR2_LLAMA_DIR", filepath.Join(t.TempDir(), "no-such-llama")) // 工件缺失：不真拉起
	pySrv := fakeUocr2Server(t, http.StatusOK, mustJSON(t, map[string]string{"text": "python 兜底文本"}))
	t.Setenv("GAEA_UOCR2_URL", pySrv.URL)

	p := &uocr2OCRProvider{}
	got, err := p.ExtractImage(mustTempPNG(t))
	if err != nil {
		t.Fatalf("ExtractImage: %v", err)
	}
	if got != "python 兜底文本" {
		t.Errorf("text = %q, want python 兜底文本", got)
	}
	if p.backend != uocr2BackendPython {
		t.Errorf("backend = %q, want python", p.backend)
	}
}

// TestUocr2ProviderExplicitPythonSkipsLlama 显式 GAEA_UOCR2_BACKEND=python：
// 即使 llama 假服务活着也不得触达（llama 0 命中）。
func TestUocr2ProviderExplicitPythonSkipsLlama(t *testing.T) {
	var llamaHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		atomic.AddInt32(&llamaHits, 1)
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GAEA_UOCR2_BACKEND", "python")
	t.Setenv("GAEA_UOCR2_LLAMA_URL", srv.URL)
	pySrv := fakeUocr2Server(t, http.StatusOK, mustJSON(t, map[string]string{"text": "指定 python"}))
	t.Setenv("GAEA_UOCR2_URL", pySrv.URL)

	p := &uocr2OCRProvider{}
	got, err := p.ExtractImage(mustTempPNG(t))
	if err != nil || got != "指定 python" {
		t.Fatalf("ExtractImage = %q, %v", got, err)
	}
	if atomic.LoadInt32(&llamaHits) != 0 {
		t.Errorf("显式 python 时 llama 不得被调（命中 %d 次）", llamaHits)
	}
}

// TestUocr2ProviderBackendUnknownFailsClosed 未知 GAEA_UOCR2_BACKEND fail-closed：
// 报错含原值，不静默回退任何后端。
func TestUocr2ProviderBackendUnknownFailsClosed(t *testing.T) {
	t.Setenv("GAEA_UOCR2_BACKEND", "bogus-backend")
	p := &uocr2OCRProvider{}
	_, err := p.ExtractImage(mustTempPNG(t))
	if err == nil {
		t.Fatal("未知后端值应 fail-closed 报错")
	}
	for _, want := range []string{"GAEA_UOCR2_BACKEND", "bogus-backend"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误 %q 缺少 %q", err.Error(), want)
		}
	}
}
