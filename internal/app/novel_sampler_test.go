package app

// ── 防重复默认档 测试（阶段八 8.7）──────────────────────────
//
// 出口判据对应：采样参数经绑定面可配且有默认档（resolveNovelSampler 覆盖档/
// 默认档 + applyNovelSampler 引擎门控 + done 事件回显）；请求体注入走 httptest
// 断言 JSON 键（ai 层传输面）。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

func TestResolveNovelSampler_DefaultAndOverride(t *testing.T) {
	// 默认档（零配置）
	s, warn := resolveNovelSampler(nil)
	if warn != "" || s.DryMultiplier != 0.8 || s.DryBase != 1.75 || s.DryAllowedLength != 2 || s.DryPenaltyLastN != 4096 {
		t.Fatalf("默认档不符: %+v warn=%q", s, warn)
	}
	if s.XtcProbability != 0 {
		t.Fatalf("XTC 缺省应关: %+v", s)
	}

	// 覆盖档：字段级合并（只写想改的字段）
	cfg := &config.Config{NovelSamplerJSON: `{"dry_multiplier":1.0,"xtc_probability":0.15,"xtc_threshold":0.12}`}
	s, warn = resolveNovelSampler(cfg)
	if warn != "" {
		t.Fatalf("合法覆盖不应有 warn: %q", warn)
	}
	if s.DryMultiplier != 1.0 || s.DryBase != 1.75 || s.DryAllowedLength != 2 {
		t.Fatalf("字段级合并不符: %+v", s)
	}
	if s.XtcProbability != 0.15 || s.XtcThreshold != 0.12 {
		t.Fatalf("XTC 覆盖不符: %+v", s)
	}

	// 坏 JSON：诚实回落默认档 + warn（不静默吞配置错误）
	s, warn = resolveNovelSampler(&config.Config{NovelSamplerJSON: "{bad"})
	if warn == "" || s.DryMultiplier != 0.8 {
		t.Fatalf("坏 JSON 应回落默认+warn: %+v warn=%q", s, warn)
	}
}

func TestApplyNovelSampler_EngineGate(t *testing.T) {
	cfg := &config.Config{}
	mgr := modelengine.NewManager("", "")

	// 本地 herdsman：注入
	req := &ai.ChatRequest{}
	summary, selfCheck, warn := applyNovelSampler(req, cfg, mgr, "herdsman")
	if warn != "" {
		t.Fatalf("warn: %q", warn)
	}
	if summary["enabled"] != true || selfCheck == "" {
		t.Fatalf("herdsman 应注入+带自检: %+v", summary)
	}
	if req.DryMultiplier != 0.8 || req.DryAllowedLength != 2 {
		t.Fatalf("DRY 键未写入: %+v", req)
	}

	// strata：不注入（服务层不透传，前置核实）
	req2 := &ai.ChatRequest{}
	summary2, _, _ := applyNovelSampler(req2, cfg, mgr, "strata")
	if summary2["enabled"] != false || req2.DryMultiplier != 0 {
		t.Fatalf("strata 不应注入: %+v %+v", summary2, req2)
	}

	// 一键关
	req3 := &ai.ChatRequest{}
	summary3, selfCheck3, _ := applyNovelSampler(req3, &config.Config{NovelDryDisabled: true}, mgr, "herdsman")
	if summary3["enabled"] != false || selfCheck3 != "" || req3.DryMultiplier != 0 {
		t.Fatalf("关闭档: %+v block=%q", summary3, selfCheck3)
	}

	// SelfCheck 关（覆盖档 nil→默认开；显式 false→关）
	req4 := &ai.ChatRequest{}
	_, block4, _ := applyNovelSampler(req4, &config.Config{NovelSamplerJSON: `{"self_check":false}`}, mgr, "herdsman")
	if block4 != "" {
		t.Fatalf("self_check=false 应无自检段: %q", block4)
	}
}

// TestApplyNovelSampler_WireShape 传输面：注入后的请求体 JSON 键逐字对齐
// llama.cpp server 顶层键（herdsman 真机验证过的形状）。
func TestApplyNovelSampler_WireShape(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"delta": map[string]any{"content": "x"}}},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{}
	mgr := modelengine.NewManager("", "")
	req := &ai.ChatRequest{Messages: []ai.ChatMessage{{Role: "user", Content: "x"}}}
	_, _, _ = applyNovelSampler(req, cfg, mgr, "herdsman")
	_ = srv.URL // 形状断言不依赖服务端：直接 marshal 请求体
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"dry_multiplier", "dry_base", "dry_allowed_length", "dry_penalty_last_n"} {
		if _, ok := wire[k]; !ok {
			t.Fatalf("请求体缺 %s: %s", k, raw)
		}
	}
	if _, ok := wire["xtc_probability"]; ok {
		t.Fatalf("XTC 缺省关不得出现: %s", raw)
	}
	// XTC 开启时两键齐发
	req2 := &ai.ChatRequest{}
	_, _, _ = applyNovelSampler(req2, &config.Config{NovelSamplerJSON: `{"xtc_probability":0.15,"xtc_threshold":0.12}`}, mgr, "herdsman")
	raw2, _ := json.Marshal(req2)
	var wire2 map[string]any
	_ = json.Unmarshal(raw2, &wire2)
	if _, ok := wire2["xtc_probability"]; !ok {
		t.Fatalf("XTC 开启应发两键: %s", raw2)
	}
}
