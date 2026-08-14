package app

// Herdsman 受控测评（D3-3 测评产品化）：复用 herdsman /api/benchmarks 异步
// 测评（多模型 × 变体 × 上下文长度，TTFT/TPS/token 逐 case 采样），模型中心
// 提供「一键受控测评」：发起任务 / 查看运行列表与明细 / 导出 Markdown 报告。
// 数据契约：GET /api/benchmarks（列表，含 summary）；POST /api/benchmarks
// （发起，body 见 BenchmarkRequest）；明细从 herdsman 数据目录
// model_benchmark/runs.json 读取（含逐 case 的 ttft/tps/token）。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BenchmarkRunSummary 测评运行摘要（GET /api/benchmarks 元素）。
type BenchmarkRunSummary struct {
	ID          string           `json:"id"`
	CreatedAt   string           `json:"created_at"`
	FinishedAt  string           `json:"finished_at,omitempty"`
	Status      string           `json:"status"` // succeeded / failed / running / canceled
	ModelNames  []string         `json:"model_names"`
	Variants    []string         `json:"variants"`
	ContextSizes []int           `json:"context_sizes"`
	Summary     BenchmarkSummary `json:"summary,omitempty"`
}

// BenchmarkSummary 运行级汇总指标。
type BenchmarkSummary struct {
	TotalCases   int     `json:"total_cases"`
	Succeeded    int     `json:"succeeded"`
	Failed       int     `json:"failed"`
	Canceled     int     `json:"canceled"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	AvgTTFTMs    float64 `json:"avg_ttft_ms"`
	AvgTPS       float64 `json:"avg_tps"`
}

// BenchmarkRequest 发起受控测评的配置（对齐 runs.json config 契约）。
type BenchmarkRequest struct {
	ModelNames    []string `json:"model_names"`
	Variants      []string `json:"variants"`
	ContextSizes  []int    `json:"context_sizes"`
	CacheReuseMode string  `json:"cache_reuse_mode"` // "same_prompt_second"
	WarmupCount   int      `json:"warmup_count"`
	RepeatCount   int      `json:"repeat_count"`
	Concurrency   int      `json:"concurrency"`
	Request       BenchmarkPromptRequest `json:"request"`
}

// BenchmarkPromptRequest 单次请求参数。
type BenchmarkPromptRequest struct {
	UserPrompt      string  `json:"user_prompt"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"top_p"`
	TopK            int     `json:"top_k"`
	RepeatPenalty   float64 `json:"repeat_penalty"`
	MaxTokens       int     `json:"max_tokens"`
	Stream          bool    `json:"stream"`
	TimeoutSeconds  int     `json:"timeout_seconds"`
}

// BenchmarkCase 逐 case 明细（来自 runs.json）。
type BenchmarkCase struct {
	ModelName     string  `json:"model_name"`
	VariantID     string  `json:"variant_id"`
	ContextSize   int     `json:"context_size"`
	Status        string  `json:"status"`
	StartedAt     string  `json:"started_at,omitempty"`
	EndedAt       string  `json:"ended_at,omitempty"`
	DurationMS    int64   `json:"duration_ms"`
	TTFTMSAvg     float64 `json:"ttft_ms_avg"`
	TTFTMSP95     float64 `json:"ttft_ms_p95"`
	InputTokens   int64   `json:"input_tokens"`
	OutputTokens  int64   `json:"output_tokens"`
	TotalTokens   int64   `json:"total_tokens"`
	CachedTokens  int64   `json:"cached_tokens"`
	SecondDurationMS int64 `json:"second_duration_ms"`
	SecondTTFTMSAvg  float64 `json:"second_ttft_ms_avg"`
	Error          string  `json:"error,omitempty"`
}

// BenchmarkRunDetail 运行完整明细（含 config 与逐 case）。
type BenchmarkRunDetail struct {
	ID        string          `json:"id"`
	CreatedAt string          `json:"created_at"`
	FinishedAt string         `json:"finished_at,omitempty"`
	Status    string          `json:"status"`
	Config    BenchmarkRequest `json:"config"`
	Summary   BenchmarkSummary `json:"summary"`
	Cases     []BenchmarkCase `json:"cases"`
}

// benchmarkListResp GET /api/benchmarks 响应。
type benchmarkListResp struct {
	Data []BenchmarkRunSummary `json:"data"`
}

// benchmarkStartResp POST /api/benchmarks 响应。
type benchmarkStartResp struct {
	ID string `json:"id"`
}

// herdsmanBenchHTTP 可注入的 HTTP 客户端（测试替身）。
var herdsmanBenchHTTP = &http.Client{Timeout: 30 * time.Second}

// herdsmanBenchBaseURL 测评 API 基地址（测试可注入）。
var herdsmanBenchBaseURL = "http://127.0.0.1:8080"

// GaeaBenchmarkList 返回 Herdsman 受控测评运行列表（按创建时间倒序）。
func (a *App) GaeaBenchmarkList() ([]BenchmarkRunSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, herdsmanBenchBaseURL+"/api/benchmarks", nil)
	if err != nil {
		return nil, err
	}
	resp, err := herdsmanBenchHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 Herdsman 测评接口失败（%s 未运行？）: %w", herdsmanBenchBaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("测评列表接口返回 %d", resp.StatusCode)
	}
	var out benchmarkListResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("解析测评列表失败: %w", err)
	}
	// 倒序：最近的在前面
	sort.Slice(out.Data, func(i, j int) bool { return out.Data[i].CreatedAt > out.Data[j].CreatedAt })
	return out.Data, nil
}

// GaeaBenchmarkStart 发起一次受控测评，返回运行 ID。
func (a *App) GaeaBenchmarkStart(req BenchmarkRequest) (string, error) {
	if len(req.ModelNames) == 0 {
		return "", errors.New("至少选择一个模型")
	}
	if strings.TrimSpace(req.Request.UserPrompt) == "" {
		return "", errors.New("测评提示词不能为空")
	}
	if len(req.Variants) == 0 {
		req.Variants = []string{"standard"}
	}
	if len(req.ContextSizes) == 0 {
		req.ContextSizes = []int{4096}
	}
	if req.CacheReuseMode == "" {
		req.CacheReuseMode = "same_prompt_second"
	}
	if req.WarmupCount <= 0 {
		req.WarmupCount = 1
	}
	if req.RepeatCount <= 0 {
		req.RepeatCount = 1
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 1
	}
	if req.Request.MaxTokens <= 0 {
		req.Request.MaxTokens = 512
	}
	if req.Request.TimeoutSeconds <= 0 {
		req.Request.TimeoutSeconds = 1800
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, herdsmanBenchBaseURL+"/api/benchmarks", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	hr.Header.Set("Content-Type", "application/json")
	resp, err := herdsmanBenchHTTP.Do(hr)
	if err != nil {
		return "", fmt.Errorf("连接 Herdsman 测评接口失败（%s 未运行？）: %w", herdsmanBenchBaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("发起测评失败（%d）: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out benchmarkStartResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("解析发起结果失败: %w", err)
	}
	if out.ID == "" {
		return "", errors.New("Herdsman 未返回测评运行 ID")
	}
	return out.ID, nil
}

// GaeaBenchmarkDetail 返回指定运行的完整明细（从 runs.json 读取逐 case 数据）。
func (a *App) GaeaBenchmarkDetail(id string) (BenchmarkRunDetail, error) {
	runs, err := readBenchmarkRuns()
	if err != nil {
		return BenchmarkRunDetail{}, err
	}
	for _, r := range runs {
		if r.ID == id {
			return r, nil
		}
	}
	return BenchmarkRunDetail{}, fmt.Errorf("未找到测评运行 %s（runs.json 可能尚未刷新）", id)
}

// GaeaBenchmarkExport 把指定测评运行导出为 Markdown 报告，返回文件路径。
func (a *App) GaeaBenchmarkExport(id, dir string) (string, error) {
	run, err := a.GaeaBenchmarkDetail(id)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("导出目录不能为空")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	md := renderBenchmarkReport(run)
	// 文件名带时间与状态，避免覆盖。
	ts := strings.ReplaceAll(strings.ReplaceAll(run.CreatedAt, ":", ""), "T", "_")
	ts = strings.Split(ts, ".")[0]
	shortID := id
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	path := filepath.Join(dir, fmt.Sprintf("herdsman-benchmark-%s-%s.md", ts, shortID))
	if err := os.WriteFile(path, []byte(md), 0644); err != nil {
		return "", fmt.Errorf("写入报告失败: %w", err)
	}
	return path, nil
}

// readBenchmarkRuns 从 herdsman 数据目录读取全部运行明细。
func readBenchmarkRuns() ([]BenchmarkRunDetail, error) {
	dataDir := herdsmanDataDir()
	if dataDir == "" {
		return nil, errors.New("无法定位 Herdsman 数据目录")
	}
	data, err := os.ReadFile(filepath.Join(dataDir, "model_benchmark", "runs.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("尚无测评记录（model_benchmark/runs.json 不存在）")
		}
		return nil, fmt.Errorf("读取测评记录失败: %w", err)
	}
	var runs []BenchmarkRunDetail
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, fmt.Errorf("解析测评记录失败: %w", err)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt > runs[j].CreatedAt })
	return runs, nil
}

// renderBenchmarkReport 生成中文 Markdown 测评报告。
func renderBenchmarkReport(run BenchmarkRunDetail) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Herdsman 受控测评报告\n\n")
	fmt.Fprintf(&b, "- 运行 ID：`%s`\n- 状态：%s\n", run.ID, run.Status)
	if run.CreatedAt != "" {
		fmt.Fprintf(&b, "- 发起时间：%s\n", run.CreatedAt)
	}
	if run.FinishedAt != "" {
		fmt.Fprintf(&b, "- 完成时间：%s\n", run.FinishedAt)
	}
	fmt.Fprintf(&b, "- 模型：%s\n", strings.Join(run.Config.ModelNames, "、"))
	if len(run.Config.Variants) > 0 {
		fmt.Fprintf(&b, "- 变体：%s\n", strings.Join(run.Config.Variants, "、"))
	}
	if len(run.Config.ContextSizes) > 0 {
		var cs []string
		for _, n := range run.Config.ContextSizes {
			cs = append(cs, fmt.Sprintf("%d", n))
		}
		fmt.Fprintf(&b, "- 上下文长度：%s\n", strings.Join(cs, "、"))
	}
	fmt.Fprintf(&b, "- 并发：%d · 预热 %d · 重复 %d · cache 模式：%s\n",
		run.Config.Concurrency, run.Config.WarmupCount, run.Config.RepeatCount, run.Config.CacheReuseMode)
	fmt.Fprintf(&b, "- 提示词：%s\n\n", run.Config.Request.UserPrompt)

	b.WriteString("## 汇总\n\n")
	s := run.Summary
	fmt.Fprintf(&b, "| 用例 | 成功 | 失败 | 取消 | 平均耗时 | 平均 TTFT | 平均 TPS |\n")
	fmt.Fprintf(&b, "|--:|--:|--:|--:|--:|--:|--:|\n")
	fmt.Fprintf(&b, "| %d | %d | %d | %d | %.0f ms | %.1f ms | %.1f |\n",
		s.TotalCases, s.Succeeded, s.Failed, s.Canceled, s.AvgDurationMs, s.AvgTTFTMs, s.AvgTPS)

	b.WriteString("\n## 逐用例明细\n\n")
	b.WriteString("| 模型 | 变体 | 上下文 | 状态 | 耗时(ms) | TTFT avg/p95(ms) | 入/出/总 token | 缓存 token | 二次耗时 |\n")
	b.WriteString("|---|---|---|--:|--:|--:|--:|--:|--:|\n")
	for _, c := range run.Cases {
		fmt.Fprintf(&b, "| %s | %s | %d | %s | %d | %.0f / %.0f | %d / %d / %d | %d | %d |\n",
			c.ModelName, c.VariantID, c.ContextSize, c.Status,
			c.DurationMS, c.TTFTMSAvg, c.TTFTMSP95,
			c.InputTokens, c.OutputTokens, c.TotalTokens, c.CachedTokens, c.SecondDurationMS)
	}
	if len(run.Cases) == 0 {
		b.WriteString("（无逐用例数据；运行可能尚未完成或已清理）\n")
	}
	b.WriteString("\n---\n")
	b.WriteString("*由 gaea 模型中心「受控测评」导出。*\n")
	return b.String()
}
