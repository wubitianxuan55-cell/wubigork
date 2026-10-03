package modelengine

// modelprice.go — 模型计价域（批 25 IN2-12 文件拆分，自 stats.go 原位搬移，
// 零逻辑改动）：内置定价表 + 模型 ID 归一化 + estimatePrice/estimatedCostFor/
// EstimateCostCNY 估算口径 + 默认汇率。目录价（glmCatalogPrice/
// engineCatalogPrice）与用户自定义价（userEnginePrice）见 catalog_models.go /
// user_price.go；统计记录器见 stats.go。

import (
	"math"
	"regexp"
	"strings"
)

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
// 价目 v1：自定义引擎用户价目最高优先（userEnginePrice 注册表，Manager 随
// 引擎配置加载/保存重建——用户对中转站实际计费的声明，理应压过按模型名的
// 目录/内置前缀猜测；本地引擎 ollama/herdsman 恒不计价，不消费用户价，
// 见 user_price.go）。
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
	if p, ok := userEnginePrice(engineID); ok {
		return p
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
