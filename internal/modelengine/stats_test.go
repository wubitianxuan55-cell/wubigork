package modelengine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRecordCall_Aggregates 验证调用统计的聚合逻辑。
func TestRecordCall_Aggregates(t *testing.T) {
	m := NewManager("", "")
	m.RecordCall(ModelCallUsage{
		EngineID: "deepseek", Model: "deepseek-v4-pro",
		InputTokens: 100, OutputTokens: 50, DurationMs: 1200, Success: true,
	})
	m.RecordCall(ModelCallUsage{
		EngineID: "deepseek", Model: "deepseek-v4-pro",
		InputTokens: 200, OutputTokens: 80, DurationMs: 900, Success: true,
	})
	m.RecordCall(ModelCallUsage{
		EngineID: "deepseek", Model: "deepseek-v4-pro",
		InputTokens: 0, OutputTokens: 0, DurationMs: 100, Success: false, ErrorMessage: "HTTP 500",
	})
	m.RecordCall(ModelCallUsage{
		EngineID: "xai", Model: "grok-4.20",
		InputTokens: 10, OutputTokens: 5, DurationMs: 300, Success: true,
	})

	sum := m.GetModelCallStats()
	if sum.TotalCalls != 4 {
		t.Errorf("TotalCalls = %d, want 4", sum.TotalCalls)
	}
	if sum.SuccessCalls != 3 || sum.FailCalls != 1 {
		t.Errorf("Success/Fail = %d/%d, want 3/1", sum.SuccessCalls, sum.FailCalls)
	}
	if sum.InputTokens != 310 || sum.OutputTokens != 135 || sum.TotalTokens != 445 {
		t.Errorf("tokens = %d/%d/%d, want 310/135/445", sum.InputTokens, sum.OutputTokens, sum.TotalTokens)
	}
	if sum.AvgDurationMs != 625 { // (1200+900+100+300)/4
		t.Errorf("AvgDurationMs = %d, want 625", sum.AvgDurationMs)
	}
	if len(sum.PerModel) != 2 {
		t.Fatalf("PerModel 数量 = %d, want 2", len(sum.PerModel))
	}
	ds := sum.PerModel[0]
	if ds.EngineID != "deepseek" || ds.CallCount != 3 || ds.FailCount != 1 || ds.LastError != "HTTP 500" {
		t.Errorf("deepseek 统计 = %+v", ds)
	}
	// deepseek-v4-pro: 12*(300)/1e6 + 24*(130)/1e6 = 0.00672
	if got := ds.EstimatedCost; got < 0.00671 || got > 0.00673 {
		t.Errorf("deepseek EstimatedCost = %v, want ~0.00672", got)
	}
	if ds.Currency != "CNY" {
		t.Errorf("deepseek Currency = %q, want CNY", ds.Currency)
	}
	grok := sum.PerModel[1]
	// grok-4.20: 2*(10)/1e6 + 6*(5)/1e6 = 0.00005 USD
	if got := grok.EstimatedCost; got < 0.000049 || got > 0.000051 {
		t.Errorf("grok EstimatedCost = %v, want ~0.00005", got)
	}
	if grok.Currency != "USD" {
		t.Errorf("grok Currency = %q, want USD", grok.Currency)
	}
	// 汇总折算人民币: 0.00672 + 0.00005*7.2 = 0.00708
	if got := sum.TotalCost; got < 0.00707 || got > 0.00709 {
		t.Errorf("TotalCost = %v, want ~0.00708", got)
	}
	if len(sum.Trend) != 1 {
		t.Fatalf("Trend 桶数量 = %d, want 1", len(sum.Trend))
	}
	tp := sum.Trend[0]
	if tp.Calls != 4 || tp.SuccessCalls != 3 || tp.FailCalls != 1 {
		t.Errorf("Trend 调用 = %d/%d/%d, want 4/3/1", tp.Calls, tp.SuccessCalls, tp.FailCalls)
	}
	if tp.InputTokens != 310 || tp.OutputTokens != 135 || tp.TotalTokens != 445 {
		t.Errorf("Trend Token = %d/%d/%d, want 310/135/445", tp.InputTokens, tp.OutputTokens, tp.TotalTokens)
	}
	if got := tp.Cost; got < 0.00707 || got > 0.00709 {
		t.Errorf("Trend Cost = %v, want ~0.00708", got)
	}
}

// TestNormalizeModelID 验证模型 ID 归一化（对齐 CCSwitch 规则）。
func TestNormalizeModelID(t *testing.T) {
	cases := map[string]string{
		"stepfun-ai/step-3.5-flash":      "step-3.5-flash",
		"moonshotai/kimi-k2-0905:exa":    "kimi-k2-0905",
		"gpt-5.2-codex@low":              "gpt-5.2-codex-low",
		"OpenAI/GPT-5.5-2026-05-14":      "gpt-5.5",
		"anthropic/claude-opus-4.8":      "claude-opus-4.8",
		"anthropic/claude-opus-4-8-v1:0": "claude-opus-4-8-v1",
		"DeepSeek-V4-Pro[1m]":            "deepseek-v4-pro",
	}
	for in, want := range cases {
		if got := normalizeModelID(in); got != want {
			t.Errorf("normalizeModelID(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestEstimateCost_LocalAndUnknown 本地/未知模型不计费。
func TestEstimateCost_LocalAndUnknown(t *testing.T) {
	if c, cur := estimatedCostFor("ollama", "qwen2.5-coder:14b", 1000, 500); c != 0 || cur != "" {
		t.Errorf("ollama 应免费, got %v %q", c, cur)
	}
	if c, cur := estimatedCostFor("xai", "some-future-model-x", 1000, 500); c != 0 || cur != "" {
		t.Errorf("未知模型应不计费, got %v %q", c, cur)
	}
	// OpenCode 前缀模型也应命中 DeepSeek 定价
	if c, cur := estimatedCostFor("opencode-zen", "opencode/deepseek-v4-pro", 1e6, 0); c != 12 || cur != "CNY" {
		t.Errorf("opencode deepseek-v4-pro = %v %q, want 12 CNY", c, cur)
	}
}

// TestRecordCall_EmptyModel 空模型归入引擎维度。
func TestRecordCall_EmptyModel(t *testing.T) {
	m := NewManager("", "")
	m.RecordCall(ModelCallUsage{EngineID: "ollama", Model: "", InputTokens: 7, Success: true})
	sum := m.GetModelCallStats()
	if len(sum.PerModel) != 1 {
		t.Fatalf("PerModel 数量 = %d, want 1", len(sum.PerModel))
	}
	if sum.PerModel[0].Model != "(默认)" {
		t.Errorf("Model = %q, want (默认)", sum.PerModel[0].Model)
	}
}

// TestStats_PersistAcrossRestart 统计随独立文件持久化。
func TestStats_PersistAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	statsPath := filepath.Join(dir, "model_stats.json")

	m := NewManager("", "")
	m.SetStatsPath(statsPath)
	m.RecordCall(ModelCallUsage{
		EngineID: "opencode-zen", Model: "deepseek-v4-flash",
		InputTokens: 33, OutputTokens: 12, DurationMs: 500, Success: true,
	})

	// 重新创建 Manager 并从磁盘恢复
	m2 := NewManager("", "")
	m2.SetStatsPath(statsPath)
	sum := m2.GetModelCallStats()
	if sum.TotalCalls != 1 || sum.TotalTokens != 45 {
		t.Errorf("恢复后 TotalCalls/TotalTokens = %d/%d, want 1/45", sum.TotalCalls, sum.TotalTokens)
	}
	if len(sum.PerModel) != 1 || sum.PerModel[0].Model != "deepseek-v4-flash" {
		t.Errorf("恢复后 PerModel = %+v", sum.PerModel)
	}
	if len(sum.Trend) != 1 || sum.Trend[0].Calls != 1 || sum.Trend[0].TotalTokens != 45 {
		t.Errorf("恢复后 Trend = %+v, want 1 个桶 1 次调用 45 Token", sum.Trend)
	}
	if got := sum.Trend[0].Cost; got < 0.000056 || got > 0.000058 {
		t.Errorf("恢复后 Trend Cost = %v, want ~0.000057", got)
	}
}

// TestStats_Reset 清空统计并落盘。
func TestStats_Reset(t *testing.T) {
	dir := t.TempDir()
	statsPath := filepath.Join(dir, "model_stats.json")

	m := NewManager("", "")
	m.SetStatsPath(statsPath)
	m.RecordCall(ModelCallUsage{EngineID: "xai", Model: "grok-4.20", InputTokens: 5, Success: true})
	if m.GetModelCallStats().TotalCalls != 1 {
		t.Fatal("记录后 TotalCalls 应为 1")
	}
	m.ResetModelCallStats()
	if sum := m.GetModelCallStats(); sum.TotalCalls != 0 || len(sum.PerModel) != 0 {
		t.Fatalf("重置后 = %+v, want 空", sum)
	}
	if sum := m.GetModelCallStats(); len(sum.Trend) != 0 {
		t.Errorf("重置后 Trend = %+v, want 空", sum.Trend)
	}
	// 磁盘也应已清空
	m2 := NewManager("", "")
	m2.SetStatsPath(statsPath)
	if sum := m2.GetModelCallStats(); sum.TotalCalls != 0 {
		t.Errorf("磁盘恢复后 TotalCalls = %d, want 0", sum.TotalCalls)
	}
	if data, err := os.ReadFile(statsPath); err != nil {
		t.Fatalf("读取统计文件失败: %v", err)
	} else if len(data) == 0 {
		t.Error("统计文件不应为空")
	}
}

// TestStats_TrendPrune 超过保留期的小时桶会被清理。
func TestStats_TrendPrune(t *testing.T) {
	m := NewManager("", "")
	rec := m.stats()
	rec.mu.Lock()
	rec.trends["2020-01-01T00:00"] = &TrendPoint{Time: "2020-01-01T00:00", Calls: 99}
	rec.mu.Unlock()

	m.RecordCall(ModelCallUsage{EngineID: "deepseek", Model: "deepseek-v4-flash", InputTokens: 1, Success: true})
	sum := m.GetModelCallStats()
	for _, tp := range sum.Trend {
		if tp.Time == "2020-01-01T00:00" {
			t.Fatal("过期趋势桶未被清理")
		}
	}
	if len(sum.Trend) != 1 {
		t.Errorf("Trend 桶数量 = %d, want 1", len(sum.Trend))
	}
}

// TestStats_TrendAscending 趋势桶按时间升序返回。
func TestStats_TrendAscending(t *testing.T) {
	m := NewManager("", "")
	rec := m.stats()
	rec.mu.Lock()
	rec.trends["2026-08-08T02:00"] = &TrendPoint{Time: "2026-08-08T02:00", Calls: 1}
	rec.trends["2026-08-07T10:00"] = &TrendPoint{Time: "2026-08-07T10:00", Calls: 2}
	rec.trends["2026-08-08T01:00"] = &TrendPoint{Time: "2026-08-08T01:00", Calls: 3}
	rec.mu.Unlock()

	sum := m.GetModelCallStats()
	if len(sum.Trend) != 3 {
		t.Fatalf("Trend 桶数量 = %d, want 3", len(sum.Trend))
	}
	for i := 1; i < len(sum.Trend); i++ {
		if sum.Trend[i-1].Time >= sum.Trend[i].Time {
			t.Errorf("趋势桶未升序: %v", sum.Trend)
		}
	}
}

// TestStats_NoPathNoPanic 未设置路径时统计不落盘、不报错。
func TestStats_NoPathNoPanic(t *testing.T) {
	m := NewManager("", "")
	m.RecordCall(ModelCallUsage{EngineID: "xai", Model: "grok-4.20", Success: true})
	sum := m.GetModelCallStats()
	if sum.TotalCalls != 1 {
		t.Errorf("TotalCalls = %d, want 1", sum.TotalCalls)
	}
	m.ResetModelCallStats()
}
