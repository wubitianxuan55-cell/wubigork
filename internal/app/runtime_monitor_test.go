package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// ── /metrics 代理（fetchEngineMetrics）───────────────────────────

// TestFetchEngineMetrics_Passthrough 200 + JSON 原样透传为 map。
func TestFetchEngineMetrics_Passthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if r := recover(); r != nil {
			t.Errorf("handler panic: %v", r)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"live":{"state":"generating","tok_s":65.2},"hardware":{"cpu":31}}`))
	}))
	defer srv.Close()
	m := fetchEngineMetrics(srv.URL)
	if m == nil {
		t.Fatal("200 + 合法 JSON 应透传非 nil")
	}
	live, _ := m["live"].(map[string]interface{})
	if live["state"] != "generating" {
		t.Errorf("live.state = %v, want generating", live["state"])
	}
	if hw, _ := m["hardware"].(map[string]interface{}); hw["cpu"] != float64(31) {
		t.Errorf("hardware.cpu = %v, want 31", hw["cpu"])
	}
	// 可再序列化（前端透传面）
	if b, err := json.Marshal(m); err != nil || len(b) == 0 {
		t.Errorf("透传 map 应可再序列化: %v", err)
	}
}

// TestFetchEngineMetrics_DownOrBadStatus 停机/非 200/坏 JSON → nil（「引擎未运行」口径）。
func TestFetchEngineMetrics_DownOrBadStatus(t *testing.T) {
	dead := httptest.NewServer(http.NewServeMux())
	dead.Close()
	if got := fetchEngineMetrics(dead.URL); got != nil {
		t.Error("停机引擎应返回 nil")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if got := fetchEngineMetrics(srv.URL); got != nil {
		t.Error("HTTP 500 应返回 nil")
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer bad.Close()
	if got := fetchEngineMetrics(bad.URL); got != nil {
		t.Error("坏 JSON 应返回 nil")
	}
}

// TestRuntimeMonitor_Aggregation 快照聚合形状：引擎表 + /metrics 代理 +
// 最近调用 + 累计标量。用 httptest 充当天璇 /metrics；strata 停机 → null。
func TestRuntimeMonitor_Aggregation(t *testing.T) {
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"live":{"state":"idle"},"hardware":{"cpu":31,"ram_used":45.2}}`))
	}))
	defer metrics.Close()

	a := &App{core: &core{
		engineMgr: modelengine.NewManager("", ""),
	}}
	// 天璇指到假 /metrics 且启用（strata 留默认停用分支）
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "tianxuan", Enabled: true, BaseURL: metrics.URL}); err != nil {
		t.Fatal(err)
	}
	// 一笔记账（recent_calls/totals 出数）
	a.engineMgr.RecordCall(modelengine.ModelCallUsage{EngineID: "tianxuan", Model: "flash", InputTokens: 100, OutputTokens: 20, DurationMs: 1500, Success: true})

	snap := a.RuntimeMonitor()

	engines, _ := snap["engines"].([]map[string]interface{})
	if len(engines) == 0 {
		t.Fatal("engines 表为空")
	}
	tx, _ := snap["tianxuan"].(map[string]interface{})
	if tx == nil {
		t.Fatal("天璇启用且地址可达，/metrics 代理应为非 nil")
	}
	live, _ := tx["live"].(map[string]interface{})
	if live["state"] != "idle" {
		t.Errorf("live.state = %v, want idle", live["state"])
	}
	if _, ok := snap["strata"].(map[string]interface{}); ok {
		t.Error("strata 未启用应为 nil（引擎未运行口径）")
	}
	recent, _ := snap["recent_calls"].([]modelengine.ModelCallUsage)
	if len(recent) != 1 || recent[0].Model != "flash" {
		t.Errorf("recent_calls = %+v, want 1 条 flash", recent)
	}
	totals, _ := snap["totals"].(map[string]interface{})
	if totals["total_calls"] != int64(1) {
		t.Errorf("totals.total_calls = %v, want 1", totals["total_calls"])
	}
}

// TestRuntimeMonitor_ZeroAppSafe 零值 App（无 core）不 panic，字段空壳。
func TestRuntimeMonitor_ZeroAppSafe(t *testing.T) {
	a := &App{}
	snap := a.RuntimeMonitor()
	if snap == nil {
		t.Fatal("零值 App 应返回空壳快照而非 nil")
	}
	if _, ok := snap["engines"].([]map[string]interface{}); !ok {
		t.Error("engines 应为空切片")
	}
	if v, ok := snap["strata"].(map[string]interface{}); ok && v != nil {
		t.Error("无 engineMgr 时 strata 应为 nil")
	}
}

// TestRuntimeMonitor_MetricsDownIsNil 引擎启用但 /metrics 拉失败 → nil（不拖慢快照）。
func TestRuntimeMonitor_MetricsDownIsNil(t *testing.T) {
	dead := httptest.NewServer(http.NewServeMux())
	dead.Close()
	a := &App{core: &core{engineMgr: modelengine.NewManager("", "")}}
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "tianxuan", Enabled: true, BaseURL: dead.URL}); err != nil {
		t.Fatal(err)
	}
	snap := a.RuntimeMonitor()
	if v, _ := snap["tianxuan"].(map[string]interface{}); v != nil {
		t.Error("引擎停机（拒连）时代理应为 nil")
	}
}
