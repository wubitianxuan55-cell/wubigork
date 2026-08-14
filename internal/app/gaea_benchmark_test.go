package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
