package ai

// IN2-14 思考预算守护回归：modelhub 与 herdsman/ollama 两分支共用
// clampThinkingBudget（原两处逐字复制、magic 4096 各写一遍），预算下限收口为
// minThinkingBudget，流式默认 max_tokens 与其同源。直接驱动 prepareStreamRequest
// 逐字段断言（守护逻辑不经过网络）。
//
// 断言一律写死 4096 字面量而非引用 minThinkingBudget：把常量改成 2048 时
// 本文件必须变红（反向证据锚点），否则守护测试与实现同源、失去证明力。

import (
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// TestClampThinkingBudget_HerdsmanOllamaBranch herdsman/ollama 分支：开思考
// 显式小预算抬到 4096，同时置顶层 enable_thinking 与 chat_template_kwargs。
func TestClampThinkingBudget_HerdsmanOllamaBranch(t *testing.T) {
	c := &Client{} // engineMgr nil → 非 modelhub，走 herdsman/ollama 分支
	for _, engine := range []string{"herdsman", "ollama"} {
		req := c.prepareStreamRequest("qwen3-8b", nil, ChatSimpleOptions{EngineID: engine, MaxTokens: 1024, EnableThinking: true})
		if req.MaxTokens != 4096 {
			t.Errorf("%s 开思考显式小预算应抬到 4096，got %d", engine, req.MaxTokens)
		}
		if req.EnableThinking == nil || !*req.EnableThinking {
			t.Errorf("%s 应置顶层 enable_thinking=true，got %v", engine, req.EnableThinking)
		}
		if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); !ok || !v {
			t.Errorf("%s 应带 chat_template_kwargs.enable_thinking=true，got %v", engine, req.ChatTemplateKwargs)
		}
	}
}

// TestClampThinkingBudget_GuardDoesNotFire 守护只在「开思考 + 显式小预算」时生效。
func TestClampThinkingBudget_GuardDoesNotFire(t *testing.T) {
	c := &Client{}

	// 显式大预算不动
	req := c.prepareStreamRequest("qwen3-8b", nil, ChatSimpleOptions{EngineID: "herdsman", MaxTokens: 8192, EnableThinking: true})
	if req.MaxTokens != 8192 {
		t.Errorf("显式 8192 不应被改写，got %d", req.MaxTokens)
	}

	// 未开思考不动（守护只在开思考时生效）
	req = c.prepareStreamRequest("qwen3-8b", nil, ChatSimpleOptions{EngineID: "herdsman", MaxTokens: 1024})
	if req.MaxTokens != 1024 {
		t.Errorf("未开思考不应改写预算，got %d", req.MaxTokens)
	}
	if req.EnableThinking != nil {
		t.Errorf("未开思考不应置顶层 enable_thinking，got %v", *req.EnableThinking)
	}
	if req.ChatTemplateKwargs != nil {
		t.Errorf("未开思考不应带 chat_template_kwargs，got %v", req.ChatTemplateKwargs)
	}

	// 非 herdsman/ollama 引擎不开思考也不抬预算
	req = c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "custom-1", MaxTokens: 1024, EnableThinking: true})
	if req.MaxTokens != 1024 {
		t.Errorf("非 herdsman/ollama 引擎不应抬预算，got %d", req.MaxTokens)
	}
}

// TestClampThinkingBudget_ModelHubBranch modelhub 分支：只走
// chat_template_kwargs，不开顶层 enable_thinking；开思考小预算同样抬到 4096。
func TestClampThinkingBudget_ModelHubBranch(t *testing.T) {
	c, _ := newModelHubTestClient(t, "modelhub")

	req := c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "modelhub", MaxTokens: 1024, EnableThinking: true})
	if req.MaxTokens != 4096 {
		t.Errorf("modelhub 开思考显式小预算应抬到 4096，got %d", req.MaxTokens)
	}
	if v, ok := req.ChatTemplateKwargs["enable_thinking"].(bool); !ok || !v {
		t.Errorf("modelhub 开思考应传 chat_template_kwargs.enable_thinking=true，got %v", req.ChatTemplateKwargs)
	}
	if req.EnableThinking != nil {
		t.Errorf("modelhub 不应发顶层 enable_thinking，got %v", *req.EnableThinking)
	}

	// 不开思考：预算不动，kwargs 显式 false
	req = c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "modelhub", MaxTokens: 1024})
	if req.MaxTokens != 1024 {
		t.Errorf("modelhub 未开思考不应改写预算，got %d", req.MaxTokens)
	}
	if v, _ := req.ChatTemplateKwargs["enable_thinking"].(bool); v {
		t.Errorf("modelhub 未开思考应显式 enable_thinking=false，got %v", req.ChatTemplateKwargs)
	}
}

// TestStreamDefaultMaxTokensIsThinkingBudgetFloor 默认 max_tokens 与思考预算
// 下限同源（IN2-14：原 4096 两处/三处硬编码同值不同源）。
func TestStreamDefaultMaxTokensIsThinkingBudgetFloor(t *testing.T) {
	if minThinkingBudget != 4096 {
		t.Fatalf("minThinkingBudget 应为 4096，got %d", minThinkingBudget)
	}
	c := &Client{}
	for _, engine := range []string{"herdsman", "ollama"} {
		req := c.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: engine})
		if req.MaxTokens != 4096 {
			t.Errorf("%s 未配置 max_tokens 默认应为 4096，got %d", engine, req.MaxTokens)
		}
	}

	// modelhub 分支同样走同一默认（engineMgr 判定型）
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "modelhub", BaseURL: "http://127.0.0.1:1/v1", Enabled: true}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	c2 := &Client{}
	c2.SetEngineManager(mgr)
	req := c2.prepareStreamRequest("m", nil, ChatSimpleOptions{EngineID: "modelhub"})
	if req.MaxTokens != 4096 {
		t.Errorf("modelhub 未配置 max_tokens 默认应为 4096，got %d", req.MaxTokens)
	}
}
