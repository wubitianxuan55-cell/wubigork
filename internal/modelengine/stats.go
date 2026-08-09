package modelengine

import (
	"encoding/json"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── 模型调用统计 ─────────────────────────────────────────────

// ModelCallUsage 单次调用的用量快照（由 ai.Client 在请求结束后上报）。
type ModelCallUsage struct {
	EngineID     string `json:"engine_id"`
	Model        string `json:"model"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	DurationMs   int64  `json:"duration_ms"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
}

// ModelUsageStats 单个（引擎, 模型）维度的累计统计。
type ModelUsageStats struct {
	EngineID        string  `json:"engine_id"`
	Model           string  `json:"model"`
	CallCount       int64   `json:"call_count"`
	SuccessCount    int64   `json:"success_count"`
	FailCount       int64   `json:"fail_count"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalDurationMs int64   `json:"total_duration_ms"`
	EstimatedCost   float64 `json:"estimated_cost,omitempty"` // 估算费用（按内置定价表从 Token 推导）
	Currency        string  `json:"currency,omitempty"`       // "CNY" | "USD"；空表示本地/未知模型
	LastError       string  `json:"last_error,omitempty"`
	LastCalledAt    string  `json:"last_called_at,omitempty"`
}

// ModelStatsSummary 返回给前端的统计汇总。
type ModelStatsSummary struct {
	TotalCalls      int64             `json:"total_calls"`
	SuccessCalls    int64             `json:"success_calls"`
	FailCalls       int64             `json:"fail_calls"`
	TotalTokens     int64             `json:"total_tokens"`
	InputTokens     int64             `json:"input_tokens"`
	OutputTokens    int64             `json:"output_tokens"`
	TotalDurationMs int64             `json:"total_duration_ms"`
	AvgDurationMs   int64             `json:"avg_duration_ms"`
	TotalCost       float64           `json:"total_cost"` // 估算费用总额（统一折算为人民币）
	Trend           []TrendPoint      `json:"trend"`      // 按小时的趋势序列（升序）
	PerModel        []ModelUsageStats `json:"per_model"`
	Since           string            `json:"since,omitempty"`
}

// TrendPoint 单个小时桶的用量趋势。
type TrendPoint struct {
	Time         string  `json:"time"` // 小时桶 "2006-01-02T15:00"
	Calls        int64   `json:"calls"`
	SuccessCalls int64   `json:"success_calls"`
	FailCalls    int64   `json:"fail_calls"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	Cost         float64 `json:"cost"` // 估算费用（统一折算为人民币）
}

// ── 费用估算（参考 CCSwitch 预设的官方定价） ────────────────────

// modelPrice 每百万 Token 的官方定价。
type modelPrice struct {
	InputPerM  float64
	OutputPerM float64
	Currency   string // "CNY" | "USD"
}

// modelPricing 内置定价表：按归一化模型名前缀匹配，长前缀在前。
var modelPricing = []struct {
	prefix string
	price  modelPrice
}{
	{"deepseek-v4-flash", modelPrice{1, 2, "CNY"}},
	{"deepseek-v4-pro", modelPrice{12, 24, "CNY"}},
	{"deepseek-chat", modelPrice{1, 2, "CNY"}}, // 官方映射到 v4-flash
	{"deepseek-reasoner", modelPrice{1, 2, "CNY"}},
	{"grok-4.20", modelPrice{2, 6, "USD"}},
	{"grok-4", modelPrice{3, 15, "USD"}},
	{"grok-3", modelPrice{3, 15, "USD"}},
	{"grok-2", modelPrice{2, 10, "USD"}},
	{"gpt-5.5", modelPrice{5, 30, "USD"}},
	{"gpt-5.3-codex", modelPrice{1.75, 14, "USD"}},
	{"gpt-5.2-codex", modelPrice{1.75, 14, "USD"}},
	{"gpt-5.1", modelPrice{1.25, 10, "USD"}},
	{"gpt-5", modelPrice{1.25, 10, "USD"}},
	{"claude-opus-4-8", modelPrice{5, 25, "USD"}},
	{"claude-opus-4-5", modelPrice{5, 25, "USD"}},
	{"claude-opus-4", modelPrice{15, 75, "USD"}},
	{"claude-sonnet-4-5", modelPrice{3, 15, "USD"}},
	{"claude-sonnet-4", modelPrice{3, 15, "USD"}},
	{"claude-haiku-4-5", modelPrice{1, 5, "USD"}},
	{"claude-3-5-sonnet", modelPrice{3, 15, "USD"}},
	{"claude-3-5-haiku", modelPrice{0.8, 4, "USD"}},
	{"gemini-3-pro", modelPrice{2, 12, "USD"}},
	{"gemini-3-flash", modelPrice{0.5, 3, "USD"}},
	{"gemini-2.5-pro", modelPrice{1.25, 10, "USD"}},
	{"gemini-2.5-flash", modelPrice{0.3, 2.5, "USD"}},
	{"kimi-k2", modelPrice{4, 16, "CNY"}},
	{"glm-4.7", modelPrice{2, 8, "CNY"}},
	{"doubao-seed-code", modelPrice{1.2, 8, "CNY"}},
}

var (
	reDateSuffix = regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$`)
	reDate8      = regexp.MustCompile(`-\d{8}$`)
)

// normalizeModelID 归一化模型 ID（对齐 CCSwitch 的匹配规则：去供应商前缀、
// 去 : 后缀、@ 换 -、去版本/日期后缀）。
func normalizeModelID(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.Index(s, ":"); i >= 0 {
		s = s[:i]
	}
	s = strings.ReplaceAll(s, "@", "-")
	s = strings.TrimSuffix(s, "[1m]")
	s = reDateSuffix.ReplaceAllString(s, "")
	s = reDate8.ReplaceAllString(s, "")
	return s
}

// estimatePrice 返回模型定价；本地引擎与未知模型返回空定价。
func estimatePrice(engineID, model string) modelPrice {
	switch engineID {
	case "ollama", "herdsman":
		return modelPrice{}
	}
	n := normalizeModelID(model)
	for _, e := range modelPricing {
		if strings.HasPrefix(n, e.prefix) {
			return e.price
		}
	}
	return modelPrice{}
}

// estimatedCostFor 按定价估算累计费用（每百万 Token 计价）。
func estimatedCostFor(engineID, model string, inputTokens, outputTokens int64) (cost float64, currency string) {
	p := estimatePrice(engineID, model)
	if p.Currency == "" {
		return 0, ""
	}
	cost = p.InputPerM*float64(inputTokens)/1e6 + p.OutputPerM*float64(outputTokens)/1e6
	return cost, p.Currency
}

// usdToCny 汇总口径统一折算为人民币（估算用固定汇率）。
const usdToCny = 7.2

// statsFile 磁盘统计文件结构。
type statsFile struct {
	Version int                        `json:"version"`
	Since   string                     `json:"since,omitempty"`
	Models  map[string]ModelUsageStats `json:"models"`           // key: engineID + "|" + model
	Trends  map[string]TrendPoint      `json:"trends,omitempty"` // key: 小时桶 "2006-01-02T15:00"
}

// statsRecorder 模型调用统计（线程安全 + JSON 持久化）。
type statsRecorder struct {
	mu       sync.Mutex
	path     string
	since    string
	models   map[string]*ModelUsageStats
	trends   map[string]*TrendPoint
	loaded   bool
	lastSave time.Time // 上次落盘时间（节流用）
	pending  int       // 自上次落盘以来累计的调用数（节流用）
}

// 统计落盘节流：高频调用时避免每条记录都全量写盘。
const (
	statsSaveInterval = 3 * time.Second
	statsSaveBatch    = 25
)

func newStatsRecorder() *statsRecorder {
	return &statsRecorder{
		models: make(map[string]*ModelUsageStats),
		trends: make(map[string]*TrendPoint),
	}
}

// trendMaxHours 趋势保留时长（30 天，按小时桶）。
const trendMaxHours = 30 * 24

// trendBucket 返回某个时间的小时桶 key。
func trendBucket(t time.Time) string {
	return t.Truncate(time.Hour).Format("2006-01-02T15:00")
}

// key 返回统计维度 key（引擎 + 模型，模型可能为空 → 归入引擎总计）。
func statsKey(engineID, model string) string {
	engineID = strings.TrimSpace(engineID)
	model = strings.TrimSpace(model)
	if engineID == "" {
		engineID = "unknown"
	}
	if model == "" {
		model = "(默认)"
	}
	return engineID + "|" + model
}

// load 从磁盘加载统计（懒加载：首次调用时执行）。
func (r *statsRecorder) load() {
	if r.loaded {
		return
	}
	r.loaded = true
	if r.path == "" {
		return
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("模型统计文件读取失败", "path", r.path, "error", err)
		}
		return
	}
	var f statsFile
	if err := json.Unmarshal(data, &f); err != nil {
		slog.Warn("模型统计文件解析失败", "path", r.path, "error", err)
		return
	}
	r.since = f.Since
	for k, v := range f.Models {
		vv := v
		r.models[k] = &vv
	}
	for k, v := range f.Trends {
		vv := v
		r.trends[k] = &vv
	}
	r.pruneTrendsLocked()
}

// save 将当前统计快照写回磁盘（无路径时跳过）。
func (r *statsRecorder) save() {
	if r.path == "" {
		return
	}
	r.mu.Lock()
	f := statsFile{
		Version: 2,
		Since:   r.since,
		Models:  make(map[string]ModelUsageStats, len(r.models)),
		Trends:  make(map[string]TrendPoint, len(r.trends)),
	}
	for k, v := range r.models {
		f.Models[k] = *v
	}
	for k, v := range r.trends {
		f.Trends[k] = *v
	}
	r.lastSave = time.Now()
	r.pending = 0
	r.mu.Unlock()

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		slog.Warn("序列化模型统计失败", "error", err)
		return
	}
	if err := fileutil.AtomicWrite(r.path, data, 0644); err != nil {
		slog.Warn("保存模型统计失败", "path", r.path, "error", err)
	}
}

// record 记录一次调用并落盘。
func (r *statsRecorder) record(u ModelCallUsage) {
	r.mu.Lock()
	r.load()
	if r.since == "" {
		r.since = time.Now().Format("2006-01-02 15:04:05")
	}
	key := statsKey(u.EngineID, u.Model)
	st, ok := r.models[key]
	if !ok {
		st = &ModelUsageStats{
			EngineID: strings.TrimSpace(u.EngineID),
			Model:    strings.TrimSpace(u.Model),
		}
		if st.EngineID == "" {
			st.EngineID = "unknown"
		}
		if st.Model == "" {
			st.Model = "(默认)"
		}
		r.models[key] = st
	}
	st.CallCount++
	st.TotalDurationMs += u.DurationMs
	if u.Success {
		st.SuccessCount++
	} else {
		st.FailCount++
		if u.ErrorMessage != "" {
			st.LastError = u.ErrorMessage
		}
	}
	st.InputTokens += u.InputTokens
	st.OutputTokens += u.OutputTokens
	st.TotalTokens += u.InputTokens + u.OutputTokens
	if u.FinishedAt != "" {
		st.LastCalledAt = u.FinishedAt
	}
	cost, cur := estimatedCostFor(u.EngineID, u.Model, u.InputTokens, u.OutputTokens)
	costCNY := cost
	if cur == "USD" {
		costCNY *= usdToCny
	}
	bucket := trendBucket(time.Now())
	tp, ok := r.trends[bucket]
	if !ok {
		tp = &TrendPoint{Time: bucket}
		r.trends[bucket] = tp
	}
	tp.Calls++
	if u.Success {
		tp.SuccessCalls++
	} else {
		tp.FailCalls++
	}
	tp.InputTokens += u.InputTokens
	tp.OutputTokens += u.OutputTokens
	tp.TotalTokens += u.InputTokens + u.OutputTokens
	tp.Cost += costCNY
	r.pruneTrendsLocked()
	r.pending++
	shouldSave := r.lastSave.IsZero() || time.Since(r.lastSave) >= statsSaveInterval || r.pending >= statsSaveBatch
	r.mu.Unlock()

	if shouldSave {
		r.save()
	}
}

// summary 汇总全部统计（按引擎、调用次数降序）。
func (r *statsRecorder) summary() ModelStatsSummary {
	r.mu.Lock()
	r.load()
	defer r.mu.Unlock()

	var sum ModelStatsSummary
	sum.Since = r.since
	for _, st := range r.models {
		sum.TotalCalls += st.CallCount
		sum.SuccessCalls += st.SuccessCount
		sum.FailCalls += st.FailCount
		sum.TotalTokens += st.TotalTokens
		sum.InputTokens += st.InputTokens
		sum.OutputTokens += st.OutputTokens
		sum.TotalDurationMs += st.TotalDurationMs
		cp := *st
		cost, cur := estimatedCostFor(st.EngineID, st.Model, st.InputTokens, st.OutputTokens)
		cp.EstimatedCost = cost
		cp.Currency = cur
		if cost > 0 {
			if cur == "USD" {
				sum.TotalCost += cost * usdToCny
			} else {
				sum.TotalCost += cost
			}
		}
		sum.PerModel = append(sum.PerModel, cp)
	}
	if sum.TotalCalls > 0 {
		sum.AvgDurationMs = sum.TotalDurationMs / sum.TotalCalls
	}
	for _, tp := range r.trends {
		sum.Trend = append(sum.Trend, *tp)
	}
	sort.Slice(sum.Trend, func(i, j int) bool {
		return sum.Trend[i].Time < sum.Trend[j].Time
	})
	sort.Slice(sum.PerModel, func(i, j int) bool {
		if sum.PerModel[i].EngineID != sum.PerModel[j].EngineID {
			return sum.PerModel[i].EngineID < sum.PerModel[j].EngineID
		}
		return sum.PerModel[i].CallCount > sum.PerModel[j].CallCount
	})
	return sum
}

// pruneTrendsLocked 清理超过保留期的小时桶（需持锁调用）。
func (r *statsRecorder) pruneTrendsLocked() {
	cutoff := trendBucket(time.Now().Add(-trendMaxHours * time.Hour))
	for k := range r.trends {
		if k < cutoff {
			delete(r.trends, k)
		}
	}
}

// reset 清空统计并落盘。
func (r *statsRecorder) reset() {
	r.mu.Lock()
	r.models = make(map[string]*ModelUsageStats)
	r.trends = make(map[string]*TrendPoint)
	r.since = ""
	r.loaded = true
	r.mu.Unlock()
	r.save()
}

// setStatsPath 设置统计文件路径（LoadState 后由 app 层调用）。
func (r *statsRecorder) setStatsPath(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.path = path
	r.loaded = false // 路径变化后重新懒加载
}

// ── Manager 集成 ────────────────────────────────────────────

func (m *Manager) stats() *statsRecorder {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()
	if m.statsRec == nil {
		m.statsRec = newStatsRecorder()
	}
	return m.statsRec
}

// RecordCall 记录一次模型调用（由 ai.Client 调用，线程安全）。
func (m *Manager) RecordCall(u ModelCallUsage) {
	m.stats().record(u)
}

// GetModelCallStats 获取模型调用统计汇总。
func (m *Manager) GetModelCallStats() ModelStatsSummary {
	return m.stats().summary()
}

// ResetModelCallStats 清空模型调用统计。
func (m *Manager) ResetModelCallStats() {
	m.stats().reset()
}

// SetStatsPath 设置统计持久化路径（与引擎状态文件同目录，独立文件）。
func (m *Manager) SetStatsPath(path string) {
	m.stats().setStatsPath(path)
}

// StatsPathFor 由引擎状态文件路径推导统计文件路径。
func StatsPathFor(statePath string) string {
	if statePath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(statePath), "model_stats.json")
}
