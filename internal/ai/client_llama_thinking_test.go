package ai

// client_llama_thinking_test.go — 自管 llama 的思考通道测试（2026-11-11 实测
// 口径的契约钉）：服务级 --reasoning off 下唯一可覆盖通道是
// chat_template_kwargs.enable_thinking=true（顶层字段无效）；关思考零字段
// （缺省路径不变）；开思考必同批预算守护。

import (
	"testing"
)

func TestLlama_ThinkingExplicitOnViaCTKOnly(t *testing.T) {
	c, _ := newStrataTestClient(t)

	req := c.prepareStreamRequest("qwen35", nil, ChatSimpleOptions{EngineID: "llama", EnableThinking: true})
	if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); !ok || !v {
		t.Errorf("llama 显式开思考应传 ctk enable_thinking=true，got %v", req.ChatTemplateKwargs)
	}
	if req.EnableThinking != nil {
		t.Errorf("llama 不应发顶层 enable_thinking（b12110 实测无效），got %v", *req.EnableThinking)
	}
	if req.MaxTokens < minThinkingBudget {
		t.Errorf("llama 开思考预算应被守护抬到 %d，got %d", minThinkingBudget, req.MaxTokens)
	}

	// 显式关压过显式开（UI 互斥传参，但 API 语义上关优先——不发任何字段）
	req = c.prepareStreamRequest("qwen35", nil, ChatSimpleOptions{EngineID: "llama", EnableThinking: true, DisableThinking: true})
	if _, ok := req.ChatTemplateKwargs["enable_thinking"]; ok {
		t.Errorf("llama DisableThinking 优先：不应发 ctk，got %v", req.ChatTemplateKwargs)
	}
}

func TestLlama_ThinkingDefaultOff_KeepsBudget(t *testing.T) {
	c, _ := newStrataTestClient(t)

	// 未表态：不发思考字段（服务级 --reasoning off 兜底），预算原样保留
	//（不享受守护——llama 缺省无思考，小预算是调用方自己的选择）。
	req := c.prepareStreamRequest("qwen35", nil, ChatSimpleOptions{EngineID: "llama", MaxTokens: 300})
	if _, ok := req.ChatTemplateKwargs["enable_thinking"]; ok {
		t.Errorf("llama 缺省不应发思考字段，got %v", req.ChatTemplateKwargs)
	}
	if req.MaxTokens != 300 {
		t.Errorf("llama 缺省预算应原样 300，got %d", req.MaxTokens)
	}

	// 显式关：同样零字段
	req = c.prepareStreamRequest("qwen35", nil, ChatSimpleOptions{EngineID: "llama", DisableThinking: true, MaxTokens: 160})
	if _, ok := req.ChatTemplateKwargs["enable_thinking"]; ok {
		t.Errorf("llama 显式关不应发思考字段，got %v", req.ChatTemplateKwargs)
	}
	if req.MaxTokens != 160 {
		t.Errorf("llama 关思考预算应原样 160，got %d", req.MaxTokens)
	}
}
