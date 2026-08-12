package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// routineLLMTestEnv 构造本地 OpenAI 兼容 mock + 引擎管理器 + App。
// 返回的请求通道让测试断言实际发往 routine 模型的 prompt 与目标。
func routineLLMTestEnv(t *testing.T) (*App, chan []byte, *httptest.Server) {
	t.Helper()
	gotRequests := make(chan []byte, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotRequests <- body
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"routine ok"},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`)
	}))
	t.Cleanup(srv.Close)

	cfg := &config.Config{}
	engMgr := modelengine.NewManager("", "")
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "qwen3-8b",
		Models: []modelengine.ModelInfo{{ID: "qwen3-8b"}},
	}); err != nil {
		t.Fatalf("SaveEngine herdsman: %v", err)
	}
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "opencode-zen", Name: "OpenCode Zen", Type: modelengine.EngineOpencodeZen,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "zen-1",
		Models: []modelengine.ModelInfo{{ID: "zen-1"}},
	}); err != nil {
		t.Fatalf("SaveEngine opencode-zen: %v", err)
	}

	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)
	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	return a, gotRequests, srv
}

func requestBody(t *testing.T, ch chan []byte) map[string]interface{} {
	t.Helper()
	select {
	case b := <-ch:
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("解析请求体失败: %v", err)
		}
		return m
	default:
		t.Fatal("未收到 routine_llm 请求")
		return nil
	}
}

// TestRoutineLLM_DefaultHerdsman 未绑定时回退本地 herdsman 引擎。
func TestRoutineLLM_DefaultHerdsman(t *testing.T) {
	a, got, _ := routineLLMTestEnv(t)
	tool := routineLLMTool{a: a}

	reply, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"把下面内容压缩成 3 条要点：A/B/C"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if reply != "routine ok" {
		t.Errorf("reply = %q, want routine ok", reply)
	}
	body := requestBody(t, got)
	if body["model"] != "qwen3-8b" {
		t.Errorf("model = %v, want qwen3-8b（herdsman 默认模型）", body["model"])
	}
	msgs := body["messages"].([]interface{})
	last := msgs[len(msgs)-1].(map[string]interface{})
	if !strings.Contains(last["content"].(string), "A/B/C") {
		t.Errorf("prompt 未透传: %v", last["content"])
	}
}

// TestRoutineLLM_DefaultPicksRunningModel 引擎默认模型指向已停止模型时，
// 优先挑第一个 running 的 LLM，避免把不可用模型发给本地服务。
func TestRoutineLLM_DefaultPicksRunningModel(t *testing.T) {
	a, got, srv := routineLLMTestEnv(t)
	// 引擎默认模型 = 已停止模型；列表里有一个 running 模型
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "stopped-model",
		Models: []modelengine.ModelInfo{
			{ID: "stopped-model", Status: "stopped"},
			{ID: "running-model", Status: "running"},
		},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	tool := routineLLMTool{a: a}

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"摘要"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	body := requestBody(t, got)
	if body["model"] != "running-model" {
		t.Errorf("model = %v, want running-model（自动选 running LLM）", body["model"])
	}
}

// TestRoutineLLM_FeatureBinding 模型中心「常规办公」绑定优先于默认。
func TestRoutineLLM_FeatureBinding(t *testing.T) {
	a, got, _ := routineLLMTestEnv(t)
	a.cfg.FuncRoutineEnabled = true
	a.cfg.FuncRoutineEngine = "opencode-zen"
	a.cfg.FuncRoutineModel = "zen-1"
	tool := routineLLMTool{a: a}

	reply, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"提取 JSON"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if reply != "routine ok" {
		t.Errorf("reply = %q, want routine ok", reply)
	}
	body := requestBody(t, got)
	if body["model"] != "zen-1" {
		t.Errorf("model = %v, want zen-1（常规办公绑定）", body["model"])
	}
}

// TestRoutineLLM_ExplicitOverride 显式 engine/model 参数优先级最高。
func TestRoutineLLM_ExplicitOverride(t *testing.T) {
	a, got, _ := routineLLMTestEnv(t)
	a.cfg.FuncRoutineEnabled = true
	a.cfg.FuncRoutineEngine = "opencode-zen"
	a.cfg.FuncRoutineModel = "zen-1"
	tool := routineLLMTool{a: a}

	reply, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"改写","engine":"herdsman","model":"qwen3-8b"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if reply != "routine ok" {
		t.Errorf("reply = %q, want routine ok", reply)
	}
	body := requestBody(t, got)
	if body["model"] != "qwen3-8b" {
		t.Errorf("model = %v, want qwen3-8b（显式覆盖）", body["model"])
	}
}

// TestRoutineLLM_Errors 非法引擎/空 prompt 返回可读错误。
func TestRoutineLLM_Errors(t *testing.T) {
	a, _, _ := routineLLMTestEnv(t)
	tool := routineLLMTool{a: a}

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":""}`)); err == nil {
		t.Error("空 prompt 应报错")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"x","engine":"nope"}`)); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Errorf("非法引擎应报错，got %v", err)
	}
}
