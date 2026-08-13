package herdsman

// health_test.go：H0-2 健康检查的单元测试。
// 全部隔离：端口关闭分支用「先监听再释放」的空闲本机端口，存活分支用
// httptest 本地服务器，不依赖真实 herdsman / 8080 服务。

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// healthTestFreeAddr 返回一个当前空闲的本机地址（先监听 127.0.0.1:0 取端口再释放），
// 供「端口关闭」分支使用。
func healthTestFreeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("预留空闲端口失败: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// healthTestModelsServer 返回一个 /models 返回空列表的本地存活服务。
func healthTestModelsServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[]}`))
	}))
}

// TestHealthCheckPortClosed 端口关闭分支：未监听端口 → PortOpen=false、
// APIReachable=false（跳过 API 探测）、APIError="port closed"、Healthy=false，
// Summary 含「未监听」且列出全部缺失能力。
func TestHealthCheckPortClosed(t *testing.T) {
	addr := healthTestFreeAddr(t)
	result := HealthCheck("http://"+addr, nil, time.Second)

	if result.PortOpen {
		t.Fatalf("PortOpen = true，期望 false（端口 %s 未监听）", addr)
	}
	if result.APIReachable {
		t.Fatal("APIReachable = true，端口关闭时应跳过 API 探测")
	}
	if result.APIError != "port closed" {
		t.Fatalf("APIError = %q，期望 %q", result.APIError, "port closed")
	}
	if result.Healthy {
		t.Fatal("Healthy = true，端口关闭时不应健康")
	}
	for _, k := range capabilityKeys {
		if result.Capabilities[k] {
			t.Fatalf("Capabilities[%s] = true，端口关闭时不应有任何可用能力", k)
		}
	}
	joined := strings.Join(result.Summary, "\n")
	if !strings.Contains(joined, "未监听") {
		t.Fatalf("Summary 缺少端口未监听提示: %v", result.Summary)
	}
	// 问题清单应完整：端口 1 行 + 全部缺失能力 10 行。
	if len(result.Summary) != 1+len(capabilityKeys) {
		t.Fatalf("Summary 行数 = %d，期望 %d: %v", len(result.Summary), 1+len(capabilityKeys), result.Summary)
	}
}

// TestHealthCheckAPIAlive httptest 存活分支：服务在线且各能力模型齐备 →
// 端口/API 通过、10 个能力全部就绪、Healthy=true、Summary 为空，且结果可 JSON 序列化。
func TestHealthCheckAPIAlive(t *testing.T) {
	srv := healthTestModelsServer()
	defer srv.Close()

	models := []ModelInfo{
		{ID: "qwen2.5-7b-instruct", Status: "running"},                 // chat（ID 关键词）
		{ID: "qwen2.5-vl-7b", Capability: "vision", Status: "running"}, // vision（显式 Capability 字段）
		{ID: "bge-m3", Status: ""},                                     // embedding（空状态视为可用）
		{ID: "bge-reranker-v2-m3", Status: "running"},                  // rerank（优先于 embedding）
		{ID: "paddleocr-v4", Status: "running"},                        // ocr
		{ID: "mineru-1.9", Status: "running"},                          // parse
		{ID: "sherpa-onnx-paraformer", Status: "running"},              // asr
		{ID: "cosyvoice2-0.5b", Status: "running"},                     // tts
		{ID: "zimage-v1", Status: "running"},                           // imagegen
		{ID: "qwen2.5-mt-7b", Status: "running"},                       // translation（mt 优先于 qwen）
	}

	result := HealthCheck(srv.URL, models, time.Second)
	if !result.PortOpen {
		t.Fatalf("PortOpen = false（%s），httptest 服务应可达", result.PortError)
	}
	if !result.APIReachable {
		t.Fatalf("APIReachable = false（%s）", result.APIError)
	}
	for _, k := range capabilityKeys {
		if !result.Capabilities[k] {
			t.Fatalf("Capabilities[%s] = false，模型齐备时应为 true", k)
		}
	}
	if !result.Healthy {
		t.Fatal("Healthy = false，端口/API/聊天模型齐备时应健康")
	}
	if len(result.Summary) != 0 {
		t.Fatalf("Summary 应为空（全部就绪），实际: %v", result.Summary)
	}

	// 结果须可 JSON 序列化（前端展示契约）；summary 为 omitempty，
	// 健康时不应出现在 JSON 中。
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("HealthResult JSON 序列化失败: %v", err)
	}
	for _, key := range []string{"port_open", "api_reachable", "capabilities", "healthy"} {
		if !strings.Contains(string(data), key) {
			t.Fatalf("JSON 缺少字段 %q: %s", key, data)
		}
	}
	if strings.Contains(string(data), "summary") {
		t.Fatalf("健康时 JSON 不应包含 summary 字段: %s", data)
	}
}

// TestHealthCheckStatusRules 状态规则：running（含大小写变体）与空视为可用，
// stopped/unknown 不计入任何能力。
func TestHealthCheckStatusRules(t *testing.T) {
	srv := healthTestModelsServer()
	defer srv.Close()

	models := []ModelInfo{
		{ID: "qwen2.5-7b", Status: "running"},
		{ID: "llama-3.1-8b", Status: ""},
		{ID: "gemma-2-9b", Status: "stopped"},
		{ID: "hermes-3-8b", Status: "unknown"},
		{ID: "qwen2.5-3b", Status: "RUNNING"},
	}
	result := HealthCheck(srv.URL, models, time.Second)
	if !result.Capabilities["chat"] {
		t.Fatal("应有可用聊天模型（running / 空状态 / 大小写不敏感）")
	}
	for _, k := range []string{"embedding", "vision", "ocr", "asr"} {
		if result.Capabilities[k] {
			t.Fatalf("Capabilities[%s] = true，stopped/unknown 模型不应计为可用", k)
		}
	}
	if !result.Healthy {
		t.Fatal("Healthy = false，端口/API/聊天模型齐备时应健康")
	}
}

// TestHealthCheckAPINon2xx 非 2xx 响应视为 API 不可达（APIError 含状态码）。
func TestHealthCheckAPINon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	result := HealthCheck(srv.URL, nil, time.Second)
	if !result.PortOpen {
		t.Fatalf("PortOpen = false（%s）", result.PortError)
	}
	if result.APIReachable {
		t.Fatal("APIReachable = true，500 响应应视为不可达")
	}
	if !strings.Contains(result.APIError, "500") {
		t.Fatalf("APIError = %q，应包含 HTTP 状态码", result.APIError)
	}
	if result.Healthy {
		t.Fatal("Healthy = true，API 不可达时不应健康")
	}
}

// TestHealthCheckAPITimeout 超时分支：慢响应超过 timeout → API 不可达。
func TestHealthCheckAPITimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	result := HealthCheck(srv.URL, nil, 50*time.Millisecond)
	if !result.PortOpen {
		t.Fatalf("PortOpen = false（%s）", result.PortError)
	}
	if result.APIReachable {
		t.Fatal("APIReachable = true，超时应不可达")
	}
	if result.APIError == "" || result.APIError == "port closed" {
		t.Fatalf("APIError = %q，应为超时错误", result.APIError)
	}
}

// TestHealthCheckSummary Summary 生成：缺失能力逐条列出（中文），
// stopped 模型不计入能力，就绪的能力不出现在问题清单。
func TestHealthCheckSummary(t *testing.T) {
	srv := healthTestModelsServer()
	defer srv.Close()

	models := []ModelInfo{
		{ID: "qwen2.5-7b", Status: "running"},   // chat 就绪
		{ID: "paddleocr-v4", Status: "stopped"}, // OCR 已装但未运行 → 缺失
	}
	result := HealthCheck(srv.URL, models, time.Second)
	if !result.Capabilities["chat"] {
		t.Fatal("聊天模型应可用")
	}
	if result.Capabilities["ocr"] {
		t.Fatal("stopped 的 OCR 模型不应计为可用")
	}
	joined := strings.Join(result.Summary, "\n")
	if !strings.Contains(joined, "未发现可用 OCR 模型") {
		t.Fatalf("Summary 缺少 OCR 缺失提示: %v", result.Summary)
	}
	if strings.Contains(joined, "聊天") {
		t.Fatalf("Summary 不应包含聊天缺失提示: %v", result.Summary)
	}
}

// TestClassifyModelCapability 关键词归类各分支（大小写、优先级、未识别）。
func TestClassifyModelCapability(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		// chat
		{"qwen2.5-7b-instruct", "chat"},
		{"llama-3.1-8b", "chat"},
		{"gemma-2-9b", "chat"},
		{"hermes-3-8b", "chat"},
		{"QWEN2.5-14B", "chat"}, // 大小写不敏感
		// embedding
		{"bge-m3", "embedding"},
		{"qwen3-embedding-0.6b", "embedding"}, // 含 qwen 但须命中 embedding（优先级高于 chat）
		{"text-embedding-v3", "embedding"},
		// rerank（优先级高于 embedding：bge-reranker 不应归为 embedding）
		{"bge-reranker-v2-m3", "rerank"},
		{"jina-reranker-v2", "rerank"},
		// ocr / parse
		{"paddleocr-v4", "ocr"},
		{"ocr-server", "ocr"},
		{"mineru-1.9", "parse"},
		{"mineru-pdf-parse", "parse"},
		// asr
		{"sherpa-onnx-paraformer-zh", "asr"},
		{"whisper-large-v3", "asr"},
		{"funasr-streaming", "asr"},
		// tts
		{"cosyvoice2-0.5b", "tts"},
		{"vox-magic-7b", "tts"},
		{"edge-tts", "tts"},
		// imagegen
		{"zimage-v1", "imagegen"},
		{"flux-dev", "imagegen"},
		{"sd-image-variation", "imagegen"},
		// translation（mt/translate 优先于 qwen 等 chat 关键词）
		{"qwen2.5-mt-7b", "translation"},
		{"translate-en-zh", "translation"},
		{"中文翻译模型", "translation"},
		// 未识别
		{"unknown-model-x", ""},
	}
	for _, c := range cases {
		if got := ClassifyModelCapability(c.id); got != c.want {
			t.Errorf("ClassifyModelCapability(%q) = %q，期望 %q", c.id, got, c.want)
		}
	}
}

// TestHostPortOf 地址提取：默认回退与端口补全。
func TestHostPortOf(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "127.0.0.1:8080"},
		{"http://127.0.0.1:8080", "127.0.0.1:8080"},
		{"http://127.0.0.1:8080/v1", "127.0.0.1:8080"},
		{"http://localhost:9000", "localhost:9000"},
		{"http://localhost", "localhost:8080"}, // 无端口补默认
		{"https://10.0.0.5:9999", "10.0.0.5:9999"},
		{"不是合法URL", "127.0.0.1:8080"}, // 解析失败回退默认
	}
	for _, c := range cases {
		if got := hostPortOf(c.in); got != c.want {
			t.Errorf("hostPortOf(%q) = %q，期望 %q", c.in, got, c.want)
		}
	}
}
