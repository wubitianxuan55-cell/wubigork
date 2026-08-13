package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

const statsFixture = `{"request_id":"a1","model":"bge-m3","model_type":"embedding","runtime":"llama.cpp","status":"succeeded","duration_ms":120,"input_tokens":10,"output_tokens":5,"ttft_ms":40,"prompt_tps":200,"predicted_tps":100,"started_at":"2026-08-13T10:00:00+08:00","ended_at":"2026-08-13T10:00:00.120+08:00"}
{"request_id":"a2","model":"bge-m3","model_type":"embedding","runtime":"llama.cpp","status":"failed","duration_ms":60,"input_tokens":4,"output_tokens":0,"started_at":"2026-08-13T10:01:00+08:00","ended_at":"2026-08-13T10:01:00.060+08:00"}
{"request_id":"a3","model":"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2","model_type":"text-generation","runtime":"llama.cpp","status":"succeeded","duration_ms":7419,"input_tokens":66,"output_tokens":464,"ttft_ms":308,"prompt_tps":301,"predicted_tps":65,"started_at":"2026-08-13T09:00:00+08:00","ended_at":"2026-08-13T09:00:07+08:00"}
`

func TestParseHerdsmanModelStats(t *testing.T) {
	stats, err := parseHerdsmanModelStats([]byte(statsFixture))
	if err != nil {
		t.Fatalf("parseHerdsmanModelStats: %v", err)
	}
	if stats.Total != 2 || len(stats.PerModel) != 2 {
		t.Fatalf("Total = %d, PerModel = %d", stats.Total, len(stats.PerModel))
	}
	// 按调用次数降序：bge-m3(2) 在前。
	if stats.PerModel[0].Model != "bge-m3" {
		t.Fatalf("排序错误: %+v", stats.PerModel[0])
	}
	b := stats.PerModel[0]
	if b.Calls != 2 || b.Succeeded != 1 || b.Failed != 1 {
		t.Errorf("bge-m3 计数异常: %+v", b)
	}
	if b.InputTokens != 14 || b.OutputTokens != 5 || b.TotalDurationMs != 180 {
		t.Errorf("bge-m3 token/耗时异常: %+v", b)
	}
	if b.AvgDurationMs != 90 {
		t.Errorf("AvgDurationMs = %d, want 90", b.AvgDurationMs)
	}
	if b.AvgTTFTMs != 40 || b.AvgPromptTPS != 200 || b.AvgPredictedTPS != 100 {
		t.Errorf("bge-m3 平均性能异常: %+v", b)
	}
	if stats.Since != "2026-08-13T09:00:00+08:00" {
		t.Errorf("Since = %q", stats.Since)
	}
	// 非法行跳过不报错。
	stats, err = parseHerdsmanModelStats([]byte("not-json\n"))
	if err != nil || stats.Total != 0 {
		t.Fatalf("非法行应跳过: stats=%+v err=%v", stats, err)
	}
}

func TestHerdsmanModelStats_FromDisk(t *testing.T) {
	dir := t.TempDir()
	statsDir := filepath.Join(dir, "model_stats")
	if err := os.MkdirAll(statsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(statsDir, "events.jsonl"), []byte(statsFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDSMAN_DATA_DIR", dir)
	a := &App{}
	stats, err := a.HerdsmanModelStats()
	if err != nil || stats.Total != 2 {
		t.Fatalf("HerdsmanModelStats: %+v err=%v", stats, err)
	}
	// 文件缺失：返回错误与 Error 字段。
	t.Setenv("HERDSMAN_DATA_DIR", t.TempDir())
	stats, err = a.HerdsmanModelStats()
	if err == nil || !strings.Contains(stats.Error, "no such file") && !strings.Contains(stats.Error, "cannot find") {
		t.Fatalf("缺失应报错: stats=%+v err=%v", stats, err)
	}
}

func TestResolveHerdsmanSearchModel(t *testing.T) {
	mgr := modelengine.NewManager("", "")
	_ = mgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", BaseURL: "http://127.0.0.1:8080", Enabled: true,
		Models: []modelengine.ModelInfo{
			{ID: "bge-m3"},
			{ID: "qwen3-embedding-4b"},
			{ID: "bge-reranker-v2-m3"},
			{ID: "qwen3-reranker-4b"},
		},
	})
	a := &App{core: &core{engineMgr: mgr}}
	e := a.localSearchEmbedder()
	if e == nil || e.Model != "qwen3-embedding-4b" {
		t.Fatalf("embedding 应优先 qwen3-embedding-4b, got %+v", e)
	}
	r := a.localSearchReranker()
	if r == nil || r.Model != "qwen3-reranker-4b" {
		t.Fatalf("rerank 应优先 qwen3-reranker-4b, got %+v", r)
	}
	// 环境变量显式覆盖。
	t.Setenv("HERDSMAN_EMBED_MODEL", "bge-m3")
	e = a.localSearchEmbedder()
	if e == nil || e.Model != "bge-m3" {
		t.Fatalf("环境变量应覆盖, got %+v", e)
	}
	t.Setenv("HERDSMAN_EMBED_MODEL", "")
	// 仅 bge 时回退。
	mgr2 := modelengine.NewManager("", "")
	_ = mgr2.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", BaseURL: "http://127.0.0.1:8080", Enabled: true,
		Models: []modelengine.ModelInfo{{ID: "bge-m3"}, {ID: "bge-reranker-v2-m3"}},
	})
	a2 := &App{core: &core{engineMgr: mgr2}}
	if e := a2.localSearchEmbedder(); e == nil || e.Model != "bge-m3" {
		t.Fatalf("应回退 bge-m3, got %+v", e)
	}
	if r := a2.localSearchReranker(); r == nil || r.Model != "bge-reranker-v2-m3" {
		t.Fatalf("应回退 bge-reranker-v2-m3, got %+v", r)
	}
	// 引擎停用 → nil。
	mgrOff := modelengine.NewManager("", "")
	_ = mgrOff.SaveEngine(modelengine.EngineConfig{ID: "herdsman", Enabled: false})
	if e := (&App{core: &core{engineMgr: mgrOff}}).localSearchEmbedder(); e != nil {
		t.Fatalf("停用引擎应 nil, got %+v", e)
	}
}
