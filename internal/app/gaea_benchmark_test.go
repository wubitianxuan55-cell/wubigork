package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// TestReadBenchmarkRuns 解析 runs.json 明细（HERDSMAN_DATA_DIR 注入）。
func TestReadBenchmarkRuns(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERDSMAN_DATA_DIR", home)
	dir := filepath.Join(home, "model_benchmark")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	runs := []BenchmarkRunDetail{
		{
			ID: "run-1", Status: "succeeded", CreatedAt: "2026-08-11T17:50:51+08:00",
			Config: BenchmarkRequest{ModelNames: []string{"A"}, ContextSizes: []int{4096},
				Request: BenchmarkPromptRequest{UserPrompt: "hi", MaxTokens: 128}},
			Summary: BenchmarkSummary{TotalCases: 1, Succeeded: 1, AvgTPS: 65.1},
			Cases: []BenchmarkCase{
				{ModelName: "A", VariantID: "standard", ContextSize: 4096, Status: "succeeded",
					DurationMS: 304, TTFTMSAvg: 29.5, InputTokens: 21, OutputTokens: 4, TotalTokens: 25},
			},
		},
	}
	data, _ := json.Marshal(runs)
	if err := os.WriteFile(filepath.Join(dir, "runs.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := readBenchmarkRuns()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "run-1" || got[0].Cases[0].TTFTMSAvg != 29.5 {
		t.Fatalf("readBenchmarkRuns = %+v", got)
	}
}

// TestGaeaBenchmarkListAndStart HTTP 端到端（注入 httptest server）。
func TestGaeaBenchmarkListAndStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/benchmarks":
			_ = json.NewEncoder(w).Encode(benchmarkListResp{Data: []BenchmarkRunSummary{
				{ID: "run-9", Status: "succeeded", ModelNames: []string{"A", "B"},
					Summary: BenchmarkSummary{TotalCases: 2, Succeeded: 2, AvgTPS: 70}},
			}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/benchmarks":
			var req BenchmarkRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if len(req.ModelNames) == 0 || req.Request.UserPrompt == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(benchmarkStartResp{ID: "run-new"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldBase, oldClient := herdsmanBenchBaseURL, herdsmanBenchHTTP
	herdsmanBenchBaseURL = srv.URL
	defer func() { herdsmanBenchBaseURL, herdsmanBenchHTTP = oldBase, oldClient }()

	a := &App{}
	list, err := a.GaeaBenchmarkList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "run-9" || list[0].Summary.AvgTPS != 70 {
		t.Fatalf("list = %+v", list)
	}

	id, err := a.GaeaBenchmarkStart(BenchmarkRequest{
		ModelNames: []string{"A"}, ContextSizes: []int{4096},
		Request: BenchmarkPromptRequest{UserPrompt: "hello", MaxTokens: 128},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "run-new" {
		t.Fatalf("start id = %q", id)
	}

	// 校验默认值补齐：空提示词应报错（不发起请求）。
	if _, err := a.GaeaBenchmarkStart(BenchmarkRequest{ModelNames: []string{"A"}}); err == nil {
		t.Fatal("空提示词应报错")
	}
}

// TestGaeaBenchmarkExport 导出 Markdown 报告。
func TestGaeaBenchmarkExport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERDSMAN_DATA_DIR", home)
	dir := filepath.Join(home, "model_benchmark")
	_ = os.MkdirAll(dir, 0755)
	runs := []BenchmarkRunDetail{{
		ID: "run-x", Status: "succeeded", CreatedAt: "2026-08-11T17:50:51+08:00",
		Config: BenchmarkRequest{ModelNames: []string{"A", "B"}, ContextSizes: []int{4096},
			Request: BenchmarkPromptRequest{UserPrompt: "bench prompt", MaxTokens: 128}},
		Summary: BenchmarkSummary{TotalCases: 2, Succeeded: 2, AvgTTFTMs: 29.5, AvgTPS: 65},
		Cases: []BenchmarkCase{
			{ModelName: "A", VariantID: "standard", ContextSize: 4096, Status: "succeeded",
				DurationMS: 300, TTFTMSAvg: 28, TotalTokens: 25},
			{ModelName: "B", VariantID: "standard", ContextSize: 4096, Status: "succeeded",
				DurationMS: 400, TTFTMSAvg: 31, TotalTokens: 30},
		},
	}}
	data, _ := json.Marshal(runs)
	_ = os.WriteFile(filepath.Join(dir, "runs.json"), data, 0644)

	outDir := t.TempDir()
	a := &App{}
	path, err := a.GaeaBenchmarkExport("run-x", outDir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(path), "herdsman-benchmark-") {
		t.Fatalf("导出文件名异常: %s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	md := string(content)
	for _, want := range []string{"# Herdsman 受控测评报告", "run-x", "bench prompt", "65.0", "| A |", "| B |"} {
		if !strings.Contains(md, want) {
			t.Fatalf("报告缺少 %q:\n%s", want, md)
		}
	}
}

// TestGaeaBenchmarkDetail_NotFound 不存在的运行返回明确错误。
func TestGaeaBenchmarkDetail_NotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERDSMAN_DATA_DIR", home)
	_ = os.MkdirAll(filepath.Join(home, "model_benchmark"), 0755)
	_ = os.WriteFile(filepath.Join(home, "model_benchmark", "runs.json"), []byte(`[]`), 0644)
	a := &App{}
	if _, err := a.GaeaBenchmarkDetail("nope"); err == nil {
		t.Fatal("应报未找到")
	}
}

// richCase 构造带 D3-4 富字段的用例。
func richCase(model string, ctx int, ttft, tps, secondTTFT, speedup float64) BenchmarkCase {
	return BenchmarkCase{
		ModelName: model, VariantID: "standard", ContextSize: ctx, Status: "succeeded",
		DurationMS: 800, TTFTMSAvg: ttft, TTFTMSP95: ttft + 10,
		InputTokens: 100, OutputTokens: 80, TotalTokens: 180, CachedTokens: 40,
		SecondDurationMS: 700, SecondTTFTMSAvg: secondTTFT,
		PromptTokensTPS: 100, OutputTokensTPS: tps,
		PrefillSpeedupRatio: speedup, PrefillMSSaved: 5.5,
		EffectiveLaunchParams: map[string]any{
			"gpu_layers": float64(99), "no_kv_offload": false, "context_size": float64(ctx),
			"batch_size": float64(2048), "ubatch_size": float64(512), "cache_type_k": "f16",
		},
	}
}

// TestRenderBenchmarkAnalysis D3-4 专项段落：每模型对比/长上下文/缓存复用/显存参数。
func TestRenderBenchmarkAnalysis(t *testing.T) {
	run := BenchmarkRunDetail{
		ID: "run-a", Status: "succeeded",
		Config: BenchmarkRequest{
			ModelNames: []string{"A", "B"}, ContextSizes: []int{4096, 8192},
			Concurrency: 2, RepeatCount: 1, WarmupCount: 1, CacheReuseMode: "same_prompt_second",
			Request: BenchmarkPromptRequest{UserPrompt: "p"},
		},
		Cases: []BenchmarkCase{
			richCase("A", 4096, 30, 60, 20, 1.5),
			richCase("A", 8192, 55, 55, 35, 1.6),
			richCase("B", 4096, 25, 70, 18, 1.4),
		},
	}
	md := renderBenchmarkReport(run)
	for _, want := range []string{
		"## 每模型对比",
		"## 长上下文专项（TTFT vs 上下文长度）",
		"## 缓存复用专项（同提示词二次请求）",
		"## 显存相关启动参数（effective_launch_params）",
		"## 并发专项",
		"gpu_layers=99",
		"| A | 2 | 42 | 57.5 |", // (30+55)/2=42.5→42（%.0f 四舍五入为 42? 42.5 → "42"? 实际 %.0f 42.5 → 42? Go 四舍六入五成双 → 42）；TPS (60+55)/2=57.5
		"| B | 1 | 25 | 70.0 |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("报告缺少 %q\n---\n%s", want, md)
		}
	}
}

// ── T7-2 可见性收口：参数钳制 / engineMgr 基地址 / 原子导出 ──

// TestGaeaBenchmarkStart_ClampsParams 并发 >4 被钳到 4、重复 >20 被钳到 20
// （服务端收到的请求体即为钳制后的值）。
func TestGaeaBenchmarkStart_ClampsParams(t *testing.T) {
	var got BenchmarkRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
			w.WriteHeader(405)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
			w.WriteHeader(400)
			return
		}
		_ = json.NewEncoder(w).Encode(benchmarkStartResp{ID: "run-clamped"})
	}))
	defer srv.Close()

	oldBase := herdsmanBenchBaseURL
	herdsmanBenchBaseURL = srv.URL
	defer func() { herdsmanBenchBaseURL = oldBase }()

	a := &App{}
	id, err := a.GaeaBenchmarkStart(BenchmarkRequest{
		ModelNames:  []string{"A"},
		Concurrency: 9,
		RepeatCount: 99,
		Request:     BenchmarkPromptRequest{UserPrompt: "hi", MaxTokens: 128},
	})
	if err != nil {
		t.Fatalf("GaeaBenchmarkStart: %v", err)
	}
	if id != "run-clamped" {
		t.Errorf("id = %q", id)
	}
	if got.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4（>4 钳制）", got.Concurrency)
	}
	if got.RepeatCount != 20 {
		t.Errorf("RepeatCount = %d, want 20（>20 钳制）", got.RepeatCount)
	}
}

// TestGaeaBenchmarkStart_ClampsNoOverreach 未超限参数不被误钳。
func TestGaeaBenchmarkStart_ClampsNoOverreach(t *testing.T) {
	var got BenchmarkRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(benchmarkStartResp{ID: "run-ok"})
	}))
	defer srv.Close()
	oldBase := herdsmanBenchBaseURL
	herdsmanBenchBaseURL = srv.URL
	defer func() { herdsmanBenchBaseURL = oldBase }()

	if _, err := (&App{}).GaeaBenchmarkStart(BenchmarkRequest{
		ModelNames: []string{"A"}, Concurrency: 3, RepeatCount: 5,
		Request: BenchmarkPromptRequest{UserPrompt: "hi", MaxTokens: 128},
	}); err != nil {
		t.Fatal(err)
	}
	if got.Concurrency != 3 || got.RepeatCount != 5 {
		t.Errorf("未超限参数不应被钳制: %d/%d", got.Concurrency, got.RepeatCount)
	}
}

// TestGaeaBenchmarkStart_BaseURLFromEngineMgr 基地址取 engineMgr 的 herdsman
// BaseURL（带 /v1 后缀被规整为根路径），而非硬编码 127.0.0.1:8080。
func TestGaeaBenchmarkStart_BaseURLFromEngineMgr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/benchmarks" {
			t.Errorf("path = %q, want /api/benchmarks（不应带 /v1）", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		_ = json.NewEncoder(w).Encode(benchmarkStartResp{ID: "run-eng"})
	}))
	defer srv.Close()

	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "herdsman", Enabled: true, BaseURL: srv.URL + "/v1"}); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{engineMgr: mgr}}

	// 不覆盖 herdsmanBenchBaseURL：若基地址仍取硬编码值，请求会打到 127.0.0.1:8080 失败。
	id, err := a.GaeaBenchmarkStart(BenchmarkRequest{
		ModelNames: []string{"A"},
		Request:    BenchmarkPromptRequest{UserPrompt: "hi", MaxTokens: 128},
	})
	if err != nil {
		t.Fatalf("GaeaBenchmarkStart: %v", err)
	}
	if id != "run-eng" {
		t.Errorf("id = %q", id)
	}
}

// TestGaeaBenchmarkList_BaseURLFromEngineMgr 列表接口同样取 engineMgr 基地址。
func TestGaeaBenchmarkList_BaseURLFromEngineMgr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/benchmarks" {
			t.Errorf("path = %q", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		_ = json.NewEncoder(w).Encode(benchmarkListResp{Data: []BenchmarkRunSummary{{ID: "r1", Status: "succeeded"}}})
	}))
	defer srv.Close()
	mgr := modelengine.NewManager("", "")
	_ = mgr.SaveEngine(modelengine.EngineConfig{ID: "herdsman", Enabled: true, BaseURL: srv.URL})
	a := &App{core: &core{engineMgr: mgr}}
	list, err := a.GaeaBenchmarkList()
	if err != nil {
		t.Fatalf("GaeaBenchmarkList: %v", err)
	}
	if len(list) != 1 || list[0].ID != "r1" {
		t.Fatalf("list = %+v", list)
	}
}

// TestGaeaBenchmarkExport_AtomicNoTempLeft 原子导出：成功后目录只留下最终报告，
// 无 .tmp 半截文件。
func TestGaeaBenchmarkExport_AtomicNoTempLeft(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERDSMAN_DATA_DIR", home)
	dir := filepath.Join(home, "model_benchmark")
	_ = os.MkdirAll(dir, 0755)
	runs := []BenchmarkRunDetail{{
		ID: "run-atomic", Status: "succeeded", CreatedAt: "2026-08-11T17:50:51+08:00",
		Config: BenchmarkRequest{ModelNames: []string{"A"}, Request: BenchmarkPromptRequest{UserPrompt: "p", MaxTokens: 128}},
	}}
	data, _ := json.Marshal(runs)
	_ = os.WriteFile(filepath.Join(dir, "runs.json"), data, 0644)

	outDir := t.TempDir()
	path, err := (&App{}).GaeaBenchmarkExport("run-atomic", outDir)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("导出后目录应只有最终报告, got %d 个文件: %v", len(entries), entries)
	}
	if entries[0].Name() != filepath.Base(path) {
		t.Errorf("残留文件异常: %s", entries[0].Name())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("报告不存在: %v", err)
	}
}
