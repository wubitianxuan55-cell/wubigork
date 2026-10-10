package ai

// Strata 思考通道与预算守护回归（2026-10-10 真机实测驱动）。
//
// 背景：Strata 跑 Qwen3.8-Flash-Next 125B MoE，是「思考原生」引擎——服务端默认
// 开思考，reasoning_content 与正文共享同一份 max_tokens。接入时只加了引擎种子，
// 请求装配点（prepareStreamRequest）没接，于是既不发思考开关、也不守护预算：
// 短预算调用点（意图 64 / 幽灵建议 160 / 章节阶段 700 / 人格护栏 256·2048）
// 的推理会把预算烧光，正文被拦腰截断，表现为「输出不稳定」。
//
// 真机对照（同一 160 token 续写请求）：
//   思考默认开、无守护 → finish_reason=length，正文只剩 8 字「晨雾尚未散尽，她」
//   ctk 关思考         → finish_reason=stop，完整 45 字
// 通道真机逐项对照：顶层 enable_thinking 传 false 仍产出 reasoning_content，
// 只有 chat_template_kwargs 有效——故本引擎只走 ctk、不发顶层字段。
//
// 断言一律写死 4096 字面量（不引用 minThinkingBudget）：常量被改成 2048 时本文件
// 必须变红，否则守护测试与实现同源、失去证明力（照 client_thinking_budget_test.go 配方）。

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

// newStrataTestClient 构造 engineMgr 就绪的 Client：strata（与 herdsman 对照）
// 都指到同一个捕获服务器，返回「最近一次请求体」的读取闭包（解码后的 map）。
func newStrataTestClient(t *testing.T) (*Client, func() map[string]any) {
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
	for _, id := range []string{"strata", "herdsman"} {
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

// TestStrata_ThinkingDefaultsOnViaCTKOnly strata 缺省即开思考，且只走
// chat_template_kwargs——顶层 enable_thinking 实测无效，发了会制造「以为关了
// 其实没关」的假象。
func TestStrata_ThinkingDefaultsOnViaCTKOnly(t *testing.T) {
	c, _ := newStrataTestClient(t)

	req := c.prepareStreamRequest("qwen3.8-flash-next-iq2_xs", nil, ChatSimpleOptions{EngineID: "strata"})
	if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); !ok || !v {
		t.Errorf("strata 缺省应开思考（chat_template_kwargs.enable_thinking=true），got %v", req.ChatTemplateKwargs)
	}
	if req.EnableThinking != nil {
		t.Errorf("strata 不应发顶层 enable_thinking（真机实测无效），got %v", *req.EnableThinking)
	}

	// 显式要求开（EnableThinking=true）与缺省同结果：都是 ctk true、顶层不发。
	req = c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "strata", EnableThinking: true})
	if v, _ := req.ChatTemplateKwargs["enable_thinking"].(bool); !v {
		t.Errorf("strata 显式开思考应传 ctk true，got %v", req.ChatTemplateKwargs)
	}
	if req.EnableThinking != nil {
		t.Errorf("strata 显式开思考也不应发顶层 enable_thinking，got %v", *req.EnableThinking)
	}
}

// TestStrata_BudgetGuard 思考预算守护：开思考时小预算一律抬到 4096。
// 这是「正文被推理挤断」的根修，覆盖真机上会截断的短预算档位。
func TestStrata_BudgetGuard(t *testing.T) {
	c, _ := newStrataTestClient(t)

	for _, in := range []int{64, 160, 256, 700, 1024, 2048, 4095} {
		req := c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "strata", MaxTokens: in})
		if req.MaxTokens != 4096 {
			t.Errorf("strata 开思考、预算 %d 应被守护抬到 4096，got %d", in, req.MaxTokens)
		}
	}

	// 显式大预算不动（守护只抬下限，不削上限）
	req := c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "strata", MaxTokens: 8192})
	if req.MaxTokens != 8192 {
		t.Errorf("显式 8192 不应被改写，got %d", req.MaxTokens)
	}

	// 未配置预算：走流式默认（与守护下限同源）
	req = c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "strata"})
	if req.MaxTokens != 4096 {
		t.Errorf("strata 未配置 max_tokens 默认应为 4096，got %d", req.MaxTokens)
	}
}

// TestStrata_DisableThinkingExplicit 显式关思考：ctk 传 false，且不再抬预算
// （推理不再抢预算，正文不再需要额外余量）。
func TestStrata_DisableThinkingExplicit(t *testing.T) {
	c, _ := newStrataTestClient(t)

	req := c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "strata", MaxTokens: 160, DisableThinking: true})
	if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); !ok || v {
		t.Errorf("显式关思考应传 chat_template_kwargs.enable_thinking=false，got %v", req.ChatTemplateKwargs)
	}
	if req.MaxTokens != 160 {
		t.Errorf("关思考后不应守护预算（推理不再抢预算），got %d", req.MaxTokens)
	}
}

// TestStrata_DisableThinkingDoesNotLeak 三态「关」侧不外溢：只对 strata 生效，
// herdsman/ollama 分支的开关语义不受影响（它们缺省关、只认 EnableThinking）。
func TestStrata_DisableThinkingDoesNotLeak(t *testing.T) {
	c := &Client{} // engineMgr nil → 非 modelhub/strata，走 herdsman/ollama 分支判定

	for _, engine := range []string{"herdsman", "ollama"} {
		req := c.prepareStreamRequest("qwen3-8b", nil, ChatSimpleOptions{EngineID: engine, MaxTokens: 1024, DisableThinking: true})
		if req.EnableThinking != nil {
			t.Errorf("%s 置 DisableThinking 不应反而开启顶层 enable_thinking，got %v", engine, *req.EnableThinking)
		}
		if req.ChatTemplateKwargs != nil {
			t.Errorf("%s 置 DisableThinking 不应携带 chat_template_kwargs，got %v", engine, req.ChatTemplateKwargs)
		}
		if req.MaxTokens != 1024 {
			t.Errorf("%s 置 DisableThinking 不应守护预算（未开思考），got %d", engine, req.MaxTokens)
		}
	}

	// 非本地引擎同样不受影响
	req := c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "custom-1", MaxTokens: 1024, DisableThinking: true})
	if req.ChatTemplateKwargs != nil || req.EnableThinking != nil {
		t.Errorf("云端/自定义引擎不应因 DisableThinking 变化，got ctk=%v top=%v", req.ChatTemplateKwargs, req.EnableThinking)
	}
}

// TestStrata_WireBody 端到端：真实流式请求体逐字段核对（装配正确不等于上线
// 正确——经 ChatStreamChunks 全链走一遍，钉住最终发出的 JSON）。
func TestStrata_WireBody(t *testing.T) {
	c, got := newStrataTestClient(t)

	chunks, cancel, err := c.ChatStreamChunks(context.Background(), "", "sys", "hi",
		ChatSimpleOptions{EngineID: "strata", MaxTokens: 160})
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
		t.Errorf("线上不应出现顶层 enable_thinking（strata 实测无效字段），body=%v", body["enable_thinking"])
	}
	ctk, ok := body["chat_template_kwargs"].(map[string]any)
	if !ok {
		t.Fatalf("线上应带 chat_template_kwargs，got %v", body["chat_template_kwargs"])
	}
	if v, _ := ctk["enable_thinking"].(bool); !v {
		t.Errorf("线上 chat_template_kwargs.enable_thinking = %v, want true（缺省开思考）", ctk["enable_thinking"])
	}
	if v, ok := body["temperature"].(float64); !ok || v != 0.7 {
		t.Errorf("线上 temperature = %v, want 0.7（strata 不参与 modelhub 采样让位）", body["temperature"])
	}
}

// TestStrata_BudgetGuardOnRawChatStream 漏斗层兜底：手装 ChatRequest 直接调
// ChatStream 的调用方绕过 prepareStreamRequest，同样必须被守护——真实调用点
// 都是短预算高风险形态：copilot CmdK 续写（256）、章节直连 applyChapterGuardrails
// （play 护栏的 max_output_tokens）、wx_agent（4096）、gaea provider bridge。
// 漏掉这条漏斗，小说正文与续写建议会继续被推理挤断。
func TestStrata_BudgetGuardOnRawChatStream(t *testing.T) {
	cases := []struct {
		name    string
		engine  string
		maxTok  int
		ctkOff  bool
		wantMax int // 线上 max_tokens；0 = 期望该字段不出现
	}{
		{"copilot 256 被守护", "strata", 256, false, 4096},
		{"章节护栏 1024 被守护", "strata", 1024, false, 4096},
		{"章节护栏 2048 被守护", "strata", 2048, false, 4096},
		{"显式关思考则不守护（尊重调用方上限）", "strata", 256, true, 256},
		{"显式大预算不被削", "strata", 8192, false, 8192},
		{"刻意不发上限则保持不发", "strata", 0, false, 0},
		{"非思考原生引擎（herdsman）不受影响", "herdsman", 256, false, 256},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, got := newStrataTestClient(t)
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

// TestStrata_BudgetGuardOnRawChat 非流式同漏斗兜底（Chat → chatOnce）。
func TestStrata_BudgetGuardOnRawChat(t *testing.T) {
	c, got := newStrataTestClient(t)
	req := &ChatRequest{
		EngineID:    "strata",
		Messages:    []ChatMessage{{Role: "user", Content: "hi"}},
		MaxTokens:   256, // copilot/章节同档位手装形态
		Temperature: 0.7,
	}
	// 捕获服务器只回 SSE；非流式解析会失败，但请求体已捕获，正是本测试要看的。
	_, _ = c.Chat(context.Background(), req)

	if v, ok := got()["max_tokens"].(float64); !ok || int(v) != 4096 {
		t.Errorf("非流式线上 max_tokens = %v, want 4096（Chat 漏斗同样需守护）", got()["max_tokens"])
	}
}
