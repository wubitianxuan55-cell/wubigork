package modelengine

import (
	"encoding/json"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"log/slog"
	"math"
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
	// KV 缓存拆分（命中缓存的 prompt token / 未命中缓存的 prompt token）。
	CacheHitTokens  int64  `json:"cache_hit_tokens,omitempty"`
	CacheMissTokens int64  `json:"cache_miss_tokens,omitempty"`
	DurationMs      int64  `json:"duration_ms"`
	Success         bool   `json:"success"`
	ErrorMessage    string `json:"error_message,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
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
	CacheHitTokens  int64   `json:"cache_hit_tokens,omitempty"`  // KV 缓存命中 prompt token
	CacheMissTokens int64   `json:"cache_miss_tokens,omitempty"` // KV 缓存未命中 prompt token
	TotalDurationMs int64   `json:"total_duration_ms"`
	EstimatedCost   float64 `json:"estimated_cost,omitempty"` // 估算费用（按内置定价表从 Token 推导）
	Currency        string  `json:"currency,omitempty"`       // "CNY" | "USD"；空表示本地/未知模型
	// BillingMode 计费口径："coding_points"=GLM 编码套餐额度内调用（费用恒 0、
	// 不折入 TotalCost，Token 照常计入）；空=按量计费（按定价表估算）。
	BillingMode  string `json:"billing_mode,omitempty"`
	LastError    string `json:"last_error,omitempty"`
	LastCalledAt string `json:"last_called_at,omitempty"`
}

// ModelStatsSummary 返回给前端的统计汇总。
type ModelStatsSummary struct {
	TotalCalls      int64             `json:"total_calls"`
	SuccessCalls    int64             `json:"success_calls"`
	FailCalls       int64             `json:"fail_calls"`
	TotalTokens     int64             `json:"total_tokens"`
	InputTokens     int64             `json:"input_tokens"`
	OutputTokens    int64             `json:"output_tokens"`
	CacheHitTokens  int64             `json:"cache_hit_tokens,omitempty"`  // KV 缓存命中 prompt token（全局汇总）
	CacheMissTokens int64             `json:"cache_miss_tokens,omitempty"` // KV 缓存未命中 prompt token（全局汇总）
	TotalDurationMs int64             `json:"total_duration_ms"`
	AvgDurationMs   int64             `json:"avg_duration_ms"`
	TotalCost       float64           `json:"total_cost"` // 估算费用总额（统一折算为人民币）
	Trend           []TrendPoint      `json:"trend"`      // 按小时的趋势序列（升序）
	PerModel        []ModelUsageStats `json:"per_model"`
	// Engines 按引擎聚合小计（加法字段，旧 JSON 兼容）。编码套餐口径的调用
	// 以 "<engine>@coding" 单列：Tokens/Calls 计入、费用 0（套餐额度内）。
	Engines  map[string]EngineSubtotal `json:"engines,omitempty"`
	Since    string                    `json:"since,omitempty"`
	UsdToCny float64                   `json:"usd_to_cny"` // 展示用美元→人民币汇率（单一来源，前端不再硬编码）
	// CatalogVersion/CatalogSource GLM 目录当前生效 schema 版本与来源
	// （B 刀，GetModelCallStats 组装处填 glmCatalog 当前生效源，零新绑定）：
	// version 如 "2"；source 如 "builtin v2 (2026-09-02)" / "override" /
	// "remote 2"（生效优先级 覆盖 > 远程 > 内嵌，见 glm_catalog.go）。
	CatalogVersion string `json:"catalog_version,omitempty"`
	CatalogSource  string `json:"catalog_source,omitempty"`
}

// EngineSubtotal 按引擎聚合的小计（ModelStatsSummary.Engines 的值）。
type EngineSubtotal struct {
	Tokens           int64   `json:"tokens"`
	Calls            int64   `json:"calls"`
	EstimatedCostCNY float64 `json:"estimated_cost_cny"`
}

// BillingCodingPoints 编码套餐计费口径（ModelUsageStats.BillingMode 取值）：
// GLM coding 端点的调用消耗套餐积分/额度，不再按 Token 计价。
const BillingCodingPoints = "coding_points"

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
	// Unit 计价单位：空=每百万 tokens（可从 token 数估算）；"call"=每次；
	// "minute"=每分钟（GLM 目录价，无法从 token 数推导，估算不计价）。
	Unit string
}

// modelPricing 内置定价表：按归一化模型名前缀匹配，长前缀在前。
var modelPricing = []struct {
	prefix string
	price  modelPrice
}{
	{"deepseek-v4-flash", modelPrice{1, 2, "CNY", ""}},
	{"deepseek-v4-pro", modelPrice{12, 24, "CNY", ""}},
	{"deepseek-chat", modelPrice{1, 2, "CNY", ""}}, // 官方映射到 v4-flash
	{"deepseek-reasoner", modelPrice{1, 2, "CNY", ""}},
	{"grok-4.20", modelPrice{2, 6, "USD", ""}},
	{"grok-4", modelPrice{3, 15, "USD", ""}},
	{"grok-3", modelPrice{3, 15, "USD", ""}},
	{"grok-2", modelPrice{2, 10, "USD", ""}},
	{"gpt-5.5", modelPrice{5, 30, "USD", ""}},
	{"gpt-5.3-codex", modelPrice{1.75, 14, "USD", ""}},
	{"gpt-5.2-codex", modelPrice{1.75, 14, "USD", ""}},
	{"gpt-5.1", modelPrice{1.25, 10, "USD", ""}},
	{"gpt-5", modelPrice{1.25, 10, "USD", ""}},
	{"claude-opus-4-8", modelPrice{5, 25, "USD", ""}},
	{"claude-opus-4-5", modelPrice{5, 25, "USD", ""}},
	{"claude-opus-4", modelPrice{15, 75, "USD", ""}},
	{"claude-sonnet-4-5", modelPrice{3, 15, "USD", ""}},
	{"claude-sonnet-4", modelPrice{3, 15, "USD", ""}},
	{"claude-haiku-4-5", modelPrice{1, 5, "USD", ""}},
	{"claude-3-5-sonnet", modelPrice{3, 15, "USD", ""}},
	{"claude-3-5-haiku", modelPrice{0.8, 4, "USD", ""}},
	{"gemini-3-pro", modelPrice{2, 12, "USD", ""}},
	{"gemini-3-flash", modelPrice{0.5, 3, "USD", ""}},
	{"gemini-2.5-pro", modelPrice{1.25, 10, "USD", ""}},
	{"gemini-2.5-flash", modelPrice{0.3, 2.5, "USD", ""}},
	{"kimi-k2", modelPrice{4, 16, "CNY", ""}},
	// GLM（智谱）定价：来源 https://docs.z.ai/guides/overview/pricing（官方
	// 国际站，USD 计价），核实日期 2026-08-31。国内 bigmodel.cn 定价页为
	// JS 渲染、无静态数据可抓，故本表采用 z.ai 官方 USD 价，折人民币走
	// usd_cny_rate（见 usdToCNYRate）。免费档填 0（费用恒 0）；官方页未
	// 列出的模型不进表（不计价）——glm-5-turbo 显式置空前缀，挡住下条
	// "glm-5" 的前缀匹配，其余未列出者（glm-4-long/glm-tts/embedding-3/
	// rerank/cogview-*/glm-image）无前缀冲突、天然不计价。长前缀在前。
	// B 刀备注：estimatePrice 的 GLM 分支先查目录（glmCatalogPrice，官方
	// 核实国内价的 glm-ocr/embedding-3 等在目录层命中），目录无价才落到
	// 本表——下列 GLM 条目数值迁移后不变（测试逐条锁定）。
	{"glm-5.3-flash", modelPrice{0.15, 0.5, "USD", ""}}, // 列表价；官方另有 50% 高峰外限时优惠（至 2026-09-09）
	{"glm-5.3", modelPrice{1.4, 4.4, "USD", ""}},
	{"glm-5.2", modelPrice{1.4, 4.4, "USD", ""}},
	{"glm-5.1", modelPrice{1.4, 4.4, "USD", ""}},
	{"glm-5-turbo", modelPrice{0, 0, "", ""}}, // 官方定价页未列出：置空挡住 glm-5 前缀
	{"glm-5", modelPrice{1, 3.2, "USD", ""}},
	{"glm-4.7-flashx", modelPrice{0.07, 0.4, "USD", ""}},
	{"glm-4.7-flash", modelPrice{0, 0, "CNY", ""}},  // 官方免费档
	{"glm-4.6v-flash", modelPrice{0, 0, "CNY", ""}}, // 官方免费档
	{"glm-4.6v", modelPrice{0.3, 0.9, "USD", ""}},
	{"glm-4.6", modelPrice{0.6, 2.2, "USD", ""}},
	{"glm-4.5-flash", modelPrice{0, 0, "CNY", ""}}, // 官方免费档
	{"glm-4.5-air", modelPrice{0.2, 1.1, "USD", ""}},
	{"glm-asr-2512", modelPrice{0.03, 0.03, "USD", ""}}, // 官方 $0.03/MTok（语音识别）
	{"glm-4.7", modelPrice{2, 8, "CNY", ""}},            // 既有条目（国内口径预设），保持不动
	{"doubao-seed-code", modelPrice{1.2, 8, "CNY", ""}},
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
// GLM 引擎先查 GLM 目录（normalizeModelID 归一后精确匹配→最长前缀匹配；
// 条目 free=true→{0,0,"CNY"}；带价→用目录价含 unit 判断；目录无价→回退
// 内置表，现状不变——glm-asr-2512/glm-4.7 等内置条目价格由此保持）。
// deepseek/xai/opencode-zen（C 刀目录通用化）先查通用目录（engineCatalogPrice，
// 同归一化与最长前缀口径），目录无价回退内置表；其余引擎（opencode-go/
// custom 等不在通用目录内）行为不变——opencode-go 订阅制无按量售价，刻意
// 不进目录（见 catalog_models.go 拍板决策）。
func estimatePrice(engineID, model string) modelPrice {
	switch engineID {
	case "ollama", "herdsman":
		return modelPrice{}
	}
	n := normalizeModelID(model)
	if engineID == string(EngineGLM) {
		if p, ok := glmCatalogPrice(n); ok {
			return p
		}
	}
	if p, ok := engineCatalogPrice(engineID, n); ok {
		return p
	}
	for _, e := range modelPricing {
		if strings.HasPrefix(n, e.prefix) {
			return e.price
		}
	}
	return modelPrice{}
}

// estimatedCostFor 按定价估算累计费用（每百万 Token 计价）。
// call/minute 计价单位（GLM 目录价）无法从 token 数推导：与未知模型同口径
// 不计价（glm-image/cogvideox-3 等单次计费模型估算值与迁移前一致恒 0）。
func estimatedCostFor(engineID, model string, inputTokens, outputTokens int64) (cost float64, currency string) {
	p := estimatePrice(engineID, model)
	if p.Currency == "" || p.Unit != "" {
		return 0, ""
	}
	cost = p.InputPerM*float64(inputTokens)/1e6 + p.OutputPerM*float64(outputTokens)/1e6
	return cost, p.Currency
}

// EstimateCostCNY 估算单次调用的费用并统一折算为人民币（v4.15 聊天
// answered_by 回显用）。口径与 estimatedCostFor 完全一致：本地引擎
// （ollama/herdsman）与未知模型恒 0；USD 计价按 usdCny 汇率折算 CNY，
// CNY 计价直用。汇率守卫与 statsRecorder.usdToCNYRate 一致：usdCny 非法
// （<=0 / NaN / Inf）时回退默认 7.2，绝不用 0 汇率把 USD 费用抹成 0。
func EstimateCostCNY(engineID, model string, inTok, outTok int64, usdCny float64) float64 {
	cost, currency := estimatedCostFor(engineID, model, inTok, outTok)
	if cost == 0 || currency == "" {
		return 0
	}
	if currency == "USD" {
		rate := usdCny
		if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
			rate = defaultUsdCnyRate
		}
		return cost * rate
	}
	return cost
}

// defaultUsdCnyRate 美元→人民币汇率默认值（未注入配置时使用，7.2）。
// 该值经 config.KeyUsdCnyRate（~/.gaea_config.json）由 app 层注入：
// 启动时 cfg.UsdCnyRate → Manager.SetUsdCnyRate，运行时由
// GaeaSetUsdCnyRate 更新。statsRecorder 持有内存副本，避免每次
// 计算都读配置文件（config.Load 含磁盘 IO，逐调用读取不划算）。
const defaultUsdCnyRate = 7.2

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

	// usdCnyRate 美元→人民币汇率（费用估算折算用；默认 7.2，
	// 由 app 层按 ~/.gaea_config.json 的 usd_cny_rate 注入/更新）。
	usdCnyRate float64
}

// 统计落盘节流：高频调用时避免每条记录都全量写盘。
const (
	statsSaveInterval = 3 * time.Second
	statsSaveBatch    = 25
)

func newStatsRecorder() *statsRecorder {
	return &statsRecorder{
		models:     make(map[string]*ModelUsageStats),
		trends:     make(map[string]*TrendPoint),
		usdCnyRate: defaultUsdCnyRate,
	}
}

// usdToCNYRate 返回当前美元→人民币汇率；持有值非法（<=0 / NaN / Inf）时
// 回退默认 7.2——防御性兜底：任何路径都不会用 0 汇率把 USD 费用抹成 0。
// 需持 r.mu 调用（或调用方保证无并发写）。
func (r *statsRecorder) usdToCNYRate() float64 {
	if rate := r.usdCnyRate; rate > 0 && !math.IsNaN(rate) && !math.IsInf(rate, 0) {
		return rate
	}
	return defaultUsdCnyRate
}

// setUsdCnyRate 更新汇率（由 app 层按配置注入；rate <= 0 / NaN / Inf 时
// 回退默认，与 usdToCNYRate 的守卫一致）。
func (r *statsRecorder) setUsdCnyRate(rate float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rate > 0 && !math.IsNaN(rate) && !math.IsInf(rate, 0) {
		r.usdCnyRate = rate
	} else {
		r.usdCnyRate = defaultUsdCnyRate
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

// record 记录一次调用并落盘。billing 为该次调用的计费口径（空=按量计费）；
// coding_points 口径下费用恒 0、不折入 TotalCost，Token 照常累计。
func (r *statsRecorder) record(u ModelCallUsage, billing string) {
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
	// 计费口径随桶记录（桶级字段）：同一（引擎, 模型）桶内 std/coding 混合的
	// 窗口（用户切换端点家族）以最近一次调用为准，见 summary 的费用门控。
	st.BillingMode = billing
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
	st.CacheHitTokens += u.CacheHitTokens
	st.CacheMissTokens += u.CacheMissTokens
	if u.FinishedAt != "" {
		st.LastCalledAt = u.FinishedAt
	}
	cost, cur := 0.0, ""
	if billing != BillingCodingPoints {
		cost, cur = estimatedCostFor(u.EngineID, u.Model, u.InputTokens, u.OutputTokens)
	}
	costCNY := cost
	if cur == "USD" {
		costCNY *= r.usdToCNYRate()
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
	sum.UsdToCny = r.usdToCNYRate()
	for _, st := range r.models {
		sum.TotalCalls += st.CallCount
		sum.SuccessCalls += st.SuccessCount
		sum.FailCalls += st.FailCount
		sum.TotalTokens += st.TotalTokens
		sum.InputTokens += st.InputTokens
		sum.OutputTokens += st.OutputTokens
		sum.CacheHitTokens += st.CacheHitTokens
		sum.CacheMissTokens += st.CacheMissTokens
		sum.TotalDurationMs += st.TotalDurationMs
		cp := *st
		// 编码套餐口径的桶不计价（费用恒 0）；std 家族照旧按 Token 重算。
		cost, cur := 0.0, ""
		if st.BillingMode != BillingCodingPoints {
			cost, cur = estimatedCostFor(st.EngineID, st.Model, st.InputTokens, st.OutputTokens)
		}
		cp.EstimatedCost = cost
		cp.Currency = cur
		costCNY := 0.0
		if cost > 0 {
			if cur == "USD" {
				costCNY = cost * r.usdToCNYRate()
			} else {
				costCNY = cost
			}
			sum.TotalCost += costCNY
		}
		// 按引擎聚合；编码套餐口径以 "<engine>@coding" 单列（Tokens 计入、费用 0）。
		engKey := st.EngineID
		if st.BillingMode == BillingCodingPoints {
			engKey = st.EngineID + "@coding"
		}
		if sum.Engines == nil {
			sum.Engines = make(map[string]EngineSubtotal, 4)
		}
		eng := sum.Engines[engKey]
		eng.Tokens += st.TotalTokens
		eng.Calls += st.CallCount
		eng.EstimatedCostCNY += costCNY
		sum.Engines[engKey] = eng
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
// GLM 引擎按当前端点家族判定计费口径：coding 端点（baseURL 含 /api/coding/）
// 走编码套餐口径（coding_points，不计价），并把旧模型名归一到服务端实际
// 模型（GlmAliasOf，让 glm-5.2 的 token 落到 glm-5.3 价格桶）；std 家族
// 照旧按 Token 估算、旧名独立计价。
func (m *Manager) RecordCall(u ModelCallUsage) {
	billing := ""
	if strings.TrimSpace(u.EngineID) == string(EngineGLM) {
		billing = m.glmCallBilling()
		if billing == BillingCodingPoints {
			if a := GlmAliasOf("coding", u.Model); a != "" {
				u.Model = a // 记账归一：coding 家族旧名按服务端实际模型落桶
			}
		}
	}
	m.stats().record(u, billing)
}

// glmCallBilling 判定当前 GLM 引擎的计费口径：编码套餐端点（baseURL 含
// /api/coding/，LoadState 已保证地址只会是两个官方常量之一）返回
// coding_points；标准端点返回空（按量计费）。
func (m *Manager) glmCallBilling() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	eng, ok := m.engines["glm"]
	if !ok {
		return ""
	}
	if strings.Contains(eng.BaseURL, "/api/coding/") {
		return BillingCodingPoints
	}
	return ""
}

// GetModelCallStats 获取模型调用统计汇总（附 GLM 目录当前生效版本/来源）。
func (m *Manager) GetModelCallStats() ModelStatsSummary {
	sum := m.stats().summary()
	sum.CatalogVersion, sum.CatalogSource = glmCatalogInfo()
	return sum
}

// ResetModelCallStats 清空模型调用统计。
func (m *Manager) ResetModelCallStats() {
	m.stats().reset()
}

// SetStatsPath 设置统计持久化路径（与引擎状态文件同目录，独立文件）。
func (m *Manager) SetStatsPath(path string) {
	m.stats().setStatsPath(path)
}

// UsdCnyRate 返回当前美元→人民币汇率（费用估算折算用，默认 7.2）。
// 未记录过任何调用时也返回配置值（懒创建统计器时已注入）。
func (m *Manager) UsdCnyRate() float64 {
	return m.stats().usdToCNYRate()
}

// SetUsdCnyRate 设置美元→人民币汇率（由 app 层按 ~/.gaea_config.json
// 的 usd_cny_rate 注入；非法值回退默认 7.2）。之后 record/summary 折算
// 立即按新汇率生效。
func (m *Manager) SetUsdCnyRate(rate float64) {
	m.stats().setUsdCnyRate(rate)
}

// StatsPathFor 由引擎状态文件路径推导统计文件路径。
func StatsPathFor(statePath string) string {
	if statePath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(statePath), "model_stats.json")
}
