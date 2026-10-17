package ai

// Tianxuan（gufo，Strix Halo 垂直线）思考通道与预算守护回归。
//
// 背景（2026-10-17 接入后补）：天璇引擎与 strata 是同族「思考原生」本地引擎——
// Qwen3.8/3.6 系模板缺省开思考，reasoning_content 与正文共享 max_tokens，
// chat_template_kwargs 是唯一有效开关（顶层 enable_thinking 真机实测无效）。
// 接入时 prepareStreamRequest 的装配分支写对了，但 ChatStream/chatOnce 漏斗上的
// applyThinkingBudgetGuard 只列了 strata——手装 ChatRequest 的短预算调用点
// （copilot 续写 256、章节护栏、wx_agent、gaea provider bridge）在 tianxuan 上
// 会把预算烧在推理里，正文被拦腰截断。
//
// 断言一律写死 4096 字面量（不引用 minThinkingBudget）：常量被改小时本文件必须
// 变红，否则守护测试与实现同源、失去证明力（照 client_strata_test.go 配方）。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// newTianxuanTestClient 构造 engineMgr 就绪的 Client：tianxuan（与 herdsman
// 对照）都指到同一个捕获服务器，返回「最近一次请求体」的读取闭包。
func newTianxuanTestClient(t *testing.T) (*Client, func() map[string]any) {
	t.Helper()
	var mu sync.Mutex
	got := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = map[string]any{}
		_ = json.Unmarshal(body, &got)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"好\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)

	mgr := modelengine.NewManager("", "")
	for _, id := range []string{"tianxuan", "herdsman"} {
		if err := mgr.SaveEngine(modelengine.EngineConfig{ID: id, BaseURL: srv.URL + "/v1", Enabled: true}); err != nil {
			t.Fatalf("SaveEngine(%s): %v", id, err)
		}
	}

	c, _ := newTestClient(t, http.NotFoundHandler())
	c.SetEngineManager(mgr)
	return c, func() map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return got
	}
}

// TestTianxuan_ThinkingDefaultsOnViaCTKOnly 缺省即开思考，且只走 ctk。
func TestTianxuan_ThinkingDefaultsOnViaCTKOnly(t *testing.T) {
	c, _ := newTianxuanTestClient(t)

	req := c.prepareStreamRequest("qwen3.8-27b", nil, ChatSimpleOptions{EngineID: "tianxuan"})
	if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); !ok || !v {
		t.Errorf("tianxuan 缺省应开思考（chat_template_kwargs.enable_thinking=true），got %v", req.ChatTemplateKwargs)
	}
	if req.EnableThinking != nil {
		t.Errorf("tianxuan 不应发顶层 enable_thinking（实测无效），got %v", *req.EnableThinking)
	}
}

// TestTianxuan_DisableThinkingExplicit 显式关思考：ctk false，且不抬预算。
func TestTianxuan_DisableThinkingExplicit(t *testing.T) {
	c, _ := newTianxuanTestClient(t)

	req := c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "tianxuan", MaxTokens: 160, DisableThinking: true})
	if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); !ok || v {
		t.Errorf("显式关思考应传 ctk false，got %v", req.ChatTemplateKwargs)
	}
	if req.MaxTokens != 160 {
		t.Errorf("关思考后不应守护预算，got %d", req.MaxTokens)
	}
}

// TestTianxuan_BudgetGuardOnRawChatStream 漏斗层兜底：手装 ChatRequest 直接调
// ChatStream 的调用方绕过 prepareStreamRequest，也必须在 tianxuan 上被守护。
func TestTianxuan_BudgetGuardOnRawChatStream(t *testing.T) {
	cases := []struct {
		name    string
		engine  string
		maxTok  int
		ctkOff  bool
		wantMax int // 线上 max_tokens；0 = 期望该字段不出现
	}{
		{"copilot 256 被守护", "tianxuan", 256, false, 4096},
		{"意图 64 被守护", "tianxuan", 64, false, 4096},
		{"章节护栏 2048 被守护", "tianxuan", 2048, false, 4096},
		{"显式关思考则不守护（尊重调用方上限）", "tianxuan", 256, true, 256},
		{"显式大预算不被削", "tianxuan", 8192, false, 8192},
		{"刻意不发上限则保持不发", "tianxuan", 0, false, 0},
		{"非思考原生引擎（herdsman）不受影响", "herdsman", 256, false, 256},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, got := newTianxuanTestClient(t)
			req := &ChatRequest{
				EngineID:    tc.engine,
				Messages:    []ChatMessage{{Role: "user", Content: "hi"}},
				MaxTokens:   tc.maxTok,
				Temperature: 0.7,
			}
			if tc.ctkOff {
				req.ChatTemplateKwargs = map[string]any{"enable_thinking": false}
			}

			chunks, err := c.ChatStream(context.Background(), req)
			if err != nil {
				t.Fatalf("ChatStream: %v", err)
			}
			for ch := range chunks { // 排空：解析协程负责 close
				if ch.Error != "" {
					t.Fatalf("stream error: %s", ch.Error)
				}
			}

			body := got()
			if len(body) == 0 {
				t.Fatal("捕获服务器未收到请求体")
			}
			if tc.wantMax == 0 {
				if v, present := body["max_tokens"]; present {
					t.Errorf("调用方刻意不发上限时不应凭空造一个，got %v", v)
				}
				return
			}
			if v, ok := body["max_tokens"].(float64); !ok || int(v) != tc.wantMax {
				t.Errorf("线上 max_tokens = %v, want %d", body["max_tokens"], tc.wantMax)
			}
		})
	}
}

// TestTianxuan_WireBody 端到端：ChatStreamChunks 全链走一遍，钉住最终 JSON。
func TestTianxuan_WireBody(t *testing.T) {
	c, got := newTianxuanTestClient(t)

	chunks, cancel, err := c.ChatStreamChunks(context.Background(), "", "sys", "hi",
		ChatSimpleOptions{EngineID: "tianxuan", MaxTokens: 160})
	if err != nil {
		t.Fatalf("ChatStreamChunks: %v", err)
	}
	drainChunks(t, chunks, cancel)

	body := got()
	if len(body) == 0 {
		t.Fatal("捕获服务器未收到请求体")
	}
	if v, ok := body["max_tokens"].(float64); !ok || int(v) != 4096 {
		t.Errorf("线上 max_tokens = %v, want 4096（160 预算应被守护）", body["max_tokens"])
	}
	if _, present := body["enable_thinking"]; present {
		t.Errorf("线上不应出现顶层 enable_thinking，body=%v", body["enable_thinking"])
	}
	ctk, ok := body["chat_template_kwargs"].(map[string]any)
	if !ok {
		t.Fatalf("线上应带 chat_template_kwargs，got %v", body["chat_template_kwargs"])
	}
	if v, _ := ctk["enable_thinking"].(bool); !v {
		t.Errorf("线上 ctk.enable_thinking = %v, want true（缺省开思考）", ctk["enable_thinking"])
	}
}
