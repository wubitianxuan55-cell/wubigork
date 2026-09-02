package modelengine

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ── 目录锚定（官方核实 2026-09-02，schema v1）──────────────────

// TestModelCatalogEmbeddedV1 内嵌 model_catalog.json schema v1 锚定：
// 顶层 version/updated/engines 三键、引擎集合恰好 deepseek/xai/opencode-zen、
// 条目数与代表字段不漂移。
func TestModelCatalogEmbeddedV1(t *testing.T) {
	doc, err := parseModelCatalog(modelCatalogJSON)
	if err != nil {
		t.Fatalf("内嵌通用目录解析失败: %v", err)
	}
	if doc.Version != 1 || doc.Updated != "2026-09-02" {
		t.Errorf("version/updated = %d/%q, want 1/2026-09-02", doc.Version, doc.Updated)
	}
	wantEngines := map[string]int{"deepseek": 3, "xai": 7, "opencode-zen": 15}
	if len(doc.Engines) != len(wantEngines) {
		t.Fatalf("engines 集合 = %v, want 恰好 %v", keysOf(doc.Engines), wantEngines)
	}
	for eng, want := range wantEngines {
		if got := len(doc.Engines[eng]); got != want {
			t.Errorf("%s 条目数 = %d, want %d", eng, got, want)
		}
	}
	// 快照形态：数量一致、OwnedBy/Kind 填充、无重复归一化 ID
	for eng, want := range wantEngines {
		models, ok := engineCatalogInfo(eng)
		if !ok || len(models) != want {
			t.Fatalf("engineCatalogInfo(%q) = %d/%v, want %d 条", eng, len(models), ok, want)
		}
		seen := map[string]bool{}
		for _, m := range models {
			if m.OwnedBy != eng || m.Kind == "" {
				t.Errorf("%s: %s OwnedBy/Kind = %q/%q, want %s/非空", eng, m.ID, m.OwnedBy, m.Kind, eng)
			}
			if seen[m.ID] {
				t.Errorf("%s 存在重复 ID %q", eng, m.ID)
			}
			seen[m.ID] = true
		}
	}
	// 目录外引擎（opencode-go 订阅制拍板不进目录 / custom / glm 走专属目录）
	for _, eng := range []string{"opencode-go", "custom-x", "glm", "ollama"} {
		if _, ok := engineCatalogInfo(eng); ok {
			t.Errorf("engineCatalogInfo(%q) 不应命中", eng)
		}
	}
}

// TestModelCatalogDeepseekEntries deepseek 条目代表字段锚定（api-docs.
// deepseek.com/quick_start/pricing，2026-09-02；峰谷双价取峰值）。
func TestModelCatalogDeepseekEntries(t *testing.T) {
	models, ok := engineCatalogInfo("deepseek")
	if !ok {
		t.Fatal("deepseek 目录缺失")
	}
	byID := map[string]ModelInfo{}
	for _, m := range models {
		byID[m.ID] = m
	}
	if got := byID["deepseek-v4-flash"]; got.ContextLength != 1000000 || got.MaxOutput != 384000 ||
		got.PriceIn != 0.44 || got.PriceOut != 1.32 || got.Currency != "USD" ||
		len(got.Caps) != 3 || got.PriceNote == "" {
		t.Errorf("deepseek-v4-flash = %+v", got)
	}
	if got := byID["deepseek-v4-pro"]; got.ContextLength != 1000000 || got.MaxOutput != 384000 ||
		got.PriceIn != 1.32 || got.PriceOut != 3.96 || got.Currency != "USD" ||
		!strings.Contains(got.PriceNote, "思考力度") {
		t.Errorf("deepseek-v4-pro = %+v", got)
	}
	if got := byID["deepseek-v4-flash-vision-exp"]; got.PriceIn != 0.44 || got.PriceOut != 1.32 ||
		len(got.Caps) != 4 || !strings.Contains(got.PriceNote, "不支持 FIM") {
		t.Errorf("deepseek-v4-flash-vision-exp = %+v", got)
	}
	// 官方 2026-07-24 停用的旧名不加条目（动态列表不再返回；旧名走内置表兜底）
	for _, id := range []string{"deepseek-chat", "deepseek-reasoner"} {
		if _, ok := byID[id]; ok {
			t.Errorf("%s 官方已停用，不应有目录条目", id)
		}
	}
}

// TestModelCatalogXAIEntries xAI 条目锚定（docs.x.ai/developers/models，
// 2026-09-02；双档取第一档；官方无能力矩阵 → caps 一律不填）。
func TestModelCatalogXAIEntries(t *testing.T) {
	models, ok := engineCatalogInfo("xai")
	if !ok {
		t.Fatal("xai 目录缺失")
	}
	byID := map[string]ModelInfo{}
	for _, m := range models {
		byID[m.ID] = m
	}
	wantCtx := map[string]int{
		"grok-4.6":                     500000,
		"grok-4.5":                     500000,
		"grok-4.3":                     1000000,
		"grok-4.20-0309-reasoning":     1000000,
		"grok-4.20-0309-non-reasoning": 1000000,
		"grok-4.20-multi-agent-0309":   1000000,
		"grok-build-0.1":               256000,
	}
	for id, ctx := range wantCtx {
		got, ok := byID[id]
		if !ok {
			t.Errorf("xai 目录缺 %q", id)
			continue
		}
		if got.ContextLength != ctx || got.MaxOutput != 0 {
			t.Errorf("%s ctx/out = %d/%d, want %d/0（官方未列不填）", id, got.ContextLength, got.MaxOutput, ctx)
		}
		if len(got.Caps) != 0 {
			t.Errorf("%s 不应携带 caps（官方无能力矩阵）, got %v", id, got.Caps)
		}
		if !strings.Contains(got.PriceNote, "2 倍计费") {
			t.Errorf("%s price_note 应标注长上下文档加价, got %q", id, got.PriceNote)
		}
	}
	if got := byID["grok-4.6"]; got.PriceIn != 2 || got.PriceOut != 6 || got.Currency != "USD" {
		t.Errorf("grok-4.6 价 = %v/%v/%q, want 2/6 USD", got.PriceIn, got.PriceOut, got.Currency)
	}
	if got := byID["grok-build-0.1"]; got.PriceIn != 1 || got.PriceOut != 2 {
		t.Errorf("grok-build-0.1 价 = %v/%v, want 1/2 USD", got.PriceIn, got.PriceOut)
	}
	// 官方已不列的旧系不加条目（内置表兜底）
	for _, id := range []string{"grok-4", "grok-4-fast", "grok-3", "grok-2"} {
		if _, ok := byID[id]; ok {
			t.Errorf("%s 官方已不列，不应有目录条目", id)
		}
	}
}

// TestModelCatalogZenEntries opencode-zen 条目锚定（opencode.ai/docs/zen，
// 2026-09-02；条目裸名；官方仅给计费断点 → context_length 不填；caps 不填）。
func TestModelCatalogZenEntries(t *testing.T) {
	models, ok := engineCatalogInfo("opencode-zen")
	if !ok {
		t.Fatal("opencode-zen 目录缺失")
	}
	byID := map[string]ModelInfo{}
	for _, m := range models {
		byID[m.ID] = m
		if m.ContextLength != 0 || len(m.Caps) != 0 {
			t.Errorf("%s ctx/caps = %d/%v, want 0/空（官方未给不填）", m.ID, m.ContextLength, m.Caps)
		}
		if m.Currency != "USD" && !m.Free {
			t.Errorf("%s currency = %q, want USD（免费条目除外）", m.ID, m.Currency)
		}
	}
	wantPrice := map[string][2]float64{
		"gpt-5.6-sol":      {2, 10},
		"gpt-5.5":          {5, 30},
		"gpt-5.6-luna":     {0.2, 1.2},
		"claude-opus-5":    {5, 25},
		"claude-sonnet-5":  {2, 10},
		"claude-haiku-4.5": {1, 5},
		"gemini-3.1-pro":   {2, 12},
		"grok-4.6":         {2, 6},
		"deepseek-v4-pro":  {0.66, 1.98},
		"glm-5.2":          {1.4, 4.4},
		"kimi-k3":          {3, 15},
		"qwen3.7-max":      {2.5, 7.5},
	}
	for id, price := range wantPrice {
		got, ok := byID[id]
		if !ok {
			t.Errorf("zen 目录缺 %q", id)
			continue
		}
		if got.PriceIn != price[0] || got.PriceOut != price[1] {
			t.Errorf("%s 价 = %v/%v, want %v/%v USD", id, got.PriceIn, got.PriceOut, price[0], price[1])
		}
	}
	if got := byID["gpt-5.6-sol"]; !strings.Contains(got.PriceNote, ">272K") || !strings.Contains(got.PriceNote, "2026-09-18") {
		t.Errorf("gpt-5.6-sol price_note = %q, want 含分档断点与促销截止", got.PriceNote)
	}
	if got := byID["gemini-3.1-pro"]; !strings.Contains(got.PriceNote, ">200K") {
		t.Errorf("gemini-3.1-pro price_note = %q, want 含 >200K 断点", got.PriceNote)
	}
	// 免费系（官方文档核实 ID）：free=true、无绝对价
	for _, id := range []string{"big-pickle", "mimo-v2.5-free", "nemotron-3-ultra-free"} {
		got, ok := byID[id]
		if !ok {
			t.Errorf("zen 目录缺免费条目 %q", id)
			continue
		}
		if !got.Free || got.PriceIn != 0 || got.PriceOut != 0 {
			t.Errorf("%s = free %v 价 %v/%v, want true/0/0", id, got.Free, got.PriceIn, got.PriceOut)
		}
	}
	freeCount := 0
	for _, m := range models {
		if m.Free {
			freeCount++
		}
	}
	if freeCount != 3 {
		t.Errorf("免费条目数 = %d, want 3", freeCount)
	}
}

// ── 坏 JSON / 版本不符：静默回退（无目录 = 估算走内置表）─────────

func TestModelCatalogBadJSONFallsBack(t *testing.T) {
	if _, err := parseModelCatalog([]byte(`{"bad json,,,`)); err == nil {
		t.Fatal("坏 JSON 应返回解析错误")
	}
	bad := newModelCatalog([]byte(`{"bad json,,,`))
	if bad.ok {
		t.Fatal("坏 JSON 目录应处于无目录态")
	}
	if _, ok := bad.info("deepseek"); ok {
		t.Error("无目录态 info 不应命中")
	}
	if _, ok := bad.price("deepseek", "deepseek-v4-flash"); ok {
		t.Error("无目录态 price 不应命中")
	}
	models := []ModelInfo{{ID: "deepseek-v4-flash"}, {ID: "grok-4.6"}}
	if got := bad.enrich("deepseek", models); !reflect.DeepEqual(got, models) {
		t.Errorf("无目录态 enrich 应原样返回, got %+v", got)
	}

	// 版本不符（模拟未来 schema 升级被旧解析器读到的防线）：同样无目录态
	v2 := newModelCatalog([]byte(`{"version":2,"updated":"2027-01-01","engines":{"deepseek":[{"id":"deepseek-v4-flash","price_in":9.9}]}}`))
	if v2.ok {
		t.Fatal("version != 1 应拒收")
	}
	if _, ok := v2.price("deepseek", "deepseek-v4-flash"); ok {
		t.Error("版本不符时 price 不应命中")
	}

	// 端到端：目录不可用 = 估算逐字节走内置表（临时替换进程级单例）
	old := genericCatalog
	genericCatalog = newModelCatalog([]byte(`broken`))
	defer func() { genericCatalog = old }()
	if p := estimatePrice("deepseek", "deepseek-v4-flash"); p.InputPerM != 1 || p.OutputPerM != 2 || p.Currency != "CNY" {
		t.Errorf("无目录时估算应回退内置表, got %+v", p)
	}
}

// ── 查价口径：精确 → 最长前缀 → 无价回退内置表 ─────────────────

func TestEngineCatalogPriceTiers(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
	// 精确命中
	if p, ok := engineCatalogPrice("deepseek", "deepseek-v4-flash"); !ok || p.InputPerM != 0.44 || p.OutputPerM != 1.32 || p.Currency != "USD" {
		t.Errorf("精确命中 = %+v/%v, want 0.44/1.32 USD", p, ok)
	}
	// 最长前缀命中（目录 ID 是归一化模型的前缀，取最长者）
	if p, ok := engineCatalogPrice("xai", "grok-4.6-fast"); !ok || p.InputPerM != 2 || p.OutputPerM != 6 {
		t.Errorf("前缀命中 = %+v/%v, want 2/6 USD", p, ok)
	}
	if p, ok := engineCatalogPrice("deepseek", "deepseek-v4-flash-vision"); !ok || p.InputPerM != 0.44 || p.OutputPerM != 1.32 {
		// "deepseek-v4-flash-vision-exp" 非其前缀，应落到更短的 "deepseek-v4-flash"
		t.Errorf("最长前缀 = %+v/%v, want 0.44/1.32 USD", p, ok)
	}
	if p, ok := engineCatalogPrice("deepseek", "deepseek-v4-pro-high"); !ok || p.InputPerM != 1.32 {
		t.Errorf("前缀命中 v4-pro = %+v/%v, want 1.32 USD", p, ok)
	}
	// 免费条目：{0,0,"CNY"} 口径（与 GLM 免费档一致），费用恒 0 但有币种
	if p, ok := engineCatalogPrice("opencode-zen", "big-pickle"); !ok || p.InputPerM != 0 || p.OutputPerM != 0 || p.Currency != "CNY" {
		t.Errorf("免费条目 = %+v/%v, want 0/0/CNY", p, ok)
	}
	if c, cur := estimatedCostFor("opencode-zen", "mimo-v2.5-free", 1e6, 1e6); c != 0 || cur != "CNY" {
		t.Errorf("免费条目估算 = %v/%q, want 0/CNY", c, cur)
	}
	if got := EstimateCostCNY("opencode-zen", "nemotron-3-ultra-free", 1e6, 1e6, 7.2); got != 0 {
		t.Errorf("免费条目 EstimateCostCNY = %v, want 0", got)
	}
	// 无价条目/无匹配：回退内置表
	if _, ok := engineCatalogPrice("xai", "grok-4"); ok {
		t.Error("grok-4 不应命中目录（官方已不列）")
	}
	if p := estimatePrice("xai", "grok-4"); p.InputPerM != 3 || p.OutputPerM != 15 || p.Currency != "USD" {
		t.Errorf("grok-4 应走内置表 {3,15,USD}, got %+v", p)
	}
	if p := estimatePrice("xai", "grok-4.20"); p.InputPerM != 2 || p.OutputPerM != 6 || p.Currency != "USD" {
		t.Errorf("grok-4.20 应走内置表 {2,6,USD}（目录 0309 系条目非其前缀）, got %+v", p)
	}
	// 目录外引擎逐字节不变：opencode-go/custom/glm 专属链路
	if p := estimatePrice("opencode-go", "deepseek-v4-pro"); p.InputPerM != 12 || p.OutputPerM != 24 || p.Currency != "CNY" {
		t.Errorf("opencode-go（订阅制不进目录）应走内置表, got %+v", p)
	}
	if p := estimatePrice("custom-abc", "claude-opus-4-8"); p.InputPerM != 5 || p.OutputPerM != 25 || p.Currency != "USD" {
		t.Errorf("custom 引擎应走内置表, got %+v", p)
	}
	// 新增计价（内置表此前无此模型）：grok-build-0.1 / kimi-k3 / qwen3.7-max
	if p, ok := engineCatalogPrice("xai", "grok-build-0.1"); !ok || p.InputPerM != 1 || p.OutputPerM != 2 {
		t.Errorf("grok-build-0.1 = %+v/%v, want 1/2 USD", p, ok)
	}
	if p, ok := engineCatalogPrice("opencode-zen", "kimi-k3"); !ok || p.InputPerM != 3 || p.OutputPerM != 15 {
		t.Errorf("kimi-k3 = %+v/%v, want 3/15 USD", p, ok)
	}
}

// ── 估算切换锁值（EstimateCostCNY，汇率 7.2）────────────────────

// TestEstimateCostCNY_CatalogSwitchLocks C 刀目录通用化后的估算锁值：
// 目录引擎（deepseek/xai/opencode-zen）目录价优先；内置表兜底模型逐位不变
// （回归锁）；本地引擎恒 0；GLM 锁值不变。
func TestEstimateCostCNY_CatalogSwitchLocks(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
	cases := []struct {
		name   string
		engine string
		model  string
		inTok  int64
		outTok int64
		want   float64 // CNY
	}{
		// 目录价（1M in + 1M out，USD × 7.2）
		{"deepseek-v4-flash 目录价", "deepseek", "deepseek-v4-flash", 1e6, 1e6, (0.44 + 1.32) * 7.2}, // 12.672
		{"deepseek-v4-pro 目录价", "deepseek", "deepseek-v4-pro", 1e6, 1e6, (1.32 + 3.96) * 7.2},     // 38.016
		{"deepseek-v4-flash-vision-exp 同 flash 价", "deepseek", "deepseek-v4-flash-vision-exp", 1e6, 1e6, (0.44 + 1.32) * 7.2},
		{"grok-4.6 目录价", "xai", "grok-4.6", 1e6, 1e6, (2 + 6) * 7.2},                        // 57.6
		{"grok-4.5 目录价", "xai", "grok-4.5", 1e6, 1e6, (2 + 6) * 7.2},                        // 57.6
		{"grok-4.3 目录价", "xai", "grok-4.3", 1e6, 1e6, (1.25 + 2.5) * 7.2},                   // 27
		{"grok-0309 系目录价", "xai", "grok-4.20-0309-reasoning", 1e6, 1e6, (1.25 + 2.5) * 7.2}, // 27
		{"gpt-5.6-sol 目录价", "opencode-zen", "gpt-5.6-sol", 1e6, 1e6, (2 + 10) * 7.2},        // 86.4
		{"gpt-5.6-sol opencode/ 前缀归一命中", "opencode-zen", "opencode/gpt-5.6-sol", 1e6, 1e6, (2 + 10) * 7.2},
		{"claude-opus-5 目录价", "opencode-zen", "claude-opus-5", 1e6, 1e6, (5 + 25) * 7.2},              // 216
		{"zen deepseek-v4-pro 目录价", "opencode-zen", "deepseek-v4-pro", 1e6, 1e6, (0.66 + 1.98) * 7.2}, // 19.008
		{"zen glm-5.2 目录价（与内置表同值）", "opencode-zen", "glm-5.2", 1e6, 1e6, (1.4 + 4.4) * 7.2},           // 41.76
		// 免费条目
		{"zen 免费系", "opencode-zen", "big-pickle", 1e6, 1e6, 0},
		// 内置表兜底回归锁（值与迁移前逐位一致）
		{"deepseek-chat 内置表", "deepseek", "deepseek-chat", 1e6, 1e6, 3},                      // (1+2) CNY 直用
		{"grok-4.20 内置表", "xai", "grok-4.20", 1e6, 1e6, (2 + 6) * 7.2},                       // 57.6
		{"grok-4 内置表", "xai", "grok-4", 1e6, 1e6, (3 + 15) * 7.2},                            // 129.6
		{"claude-opus-4-8 内置表", "opencode-go", "claude-opus-4-8", 1e6, 1e6, (5 + 25) * 7.2},  // 216
		{"gpt-5.5 内置表（opencode-go 无目录）", "opencode-go", "gpt-5.5", 1e6, 1e6, (5 + 30) * 7.2}, // 252
		{"gpt-5.5 zen 目录价与内置表同值", "opencode-zen", "gpt-5.5", 1e6, 1e6, (5 + 30) * 7.2},       // 252
		{"gemini-3-pro 内置表", "custom-x", "gemini-3-pro", 1e6, 1e6, (2 + 12) * 7.2},           // 100.8
		{"kimi-k2 内置表", "custom-x", "kimi-k2", 1e6, 1e6, 20},                                 // (4+16) CNY 直用
		{"zen deepseek-v4-flash 不在目录走内置表", "opencode-zen", "deepseek-v4-flash", 1e6, 1e6, 3}, // (1+2) CNY 直用
		// 本地引擎恒 0
		{"ollama 恒 0", "ollama", "qwen3", 1e6, 1e6, 0},
		{"herdsman 恒 0", "herdsman", "qwen3", 1e6, 1e6, 0},
		// GLM 锁值不变
		{"glm-5.3 锁值", "glm", "glm-5.3", 1e6, 0, 1.4 * 7.2}, // 10.08
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EstimateCostCNY(tc.engine, tc.model, tc.inTok, tc.outTok, 7.2); math.Abs(got-tc.want) > 0.00001 {
				t.Errorf("EstimateCostCNY(%q, %q, %d, %d) = %.6f, want %.6f",
					tc.engine, tc.model, tc.inTok, tc.outTok, got, tc.want)
			}
		})
	}
}

// ── enrich：动态列表按目录补充元数据 ────────────────────────────

func TestEnrichCatalogMeta(t *testing.T) {
	models := []ModelInfo{
		{ID: "opencode/gpt-5.6-sol", OwnedBy: "opencode"}, // 前缀归一命中裸名
		{ID: "grok-4.6-2026-01-09", OwnedBy: "xai"},       // 日期后缀变体
		// 引擎返回已有值：只填空字段，绝不覆盖
		{ID: "glm-5.2", OwnedBy: "opencode", ContextLength: 123, PriceIn: 9.9, Currency: "CNY", Caps: []string{"x"}, PriceNote: "own"},
		{ID: "totally-unknown-model", OwnedBy: "opencode"}, // 目录外原样
	}
	got := enrichCatalogMeta("opencode-zen", models)
	if !reflect.DeepEqual(got, models) {
		t.Fatalf("enrich 应原地修改并返回同一切片")
	}
	sol := models[0]
	if sol.PriceIn != 2 || sol.PriceOut != 10 || sol.Currency != "USD" || sol.PriceNote == "" {
		t.Errorf("gpt-5.6-sol 补充 = %+v", sol)
	}
	if sol.ContextLength != 0 || len(sol.Caps) != 0 || sol.Kind != "" {
		t.Errorf("gpt-5.6-sol 不应被填 ctx/caps/kind（目录未给）, got %+v", sol)
	}
	grok := models[1]
	if grok.PriceIn != 2 || grok.PriceOut != 6 || grok.Currency != "USD" || !strings.Contains(grok.PriceNote, ">200K") {
		t.Errorf("日期后缀变体应命中 grok-4.6, got %+v", grok)
	}
	glm := models[2]
	if glm.ContextLength != 123 || glm.PriceIn != 9.9 || glm.Currency != "CNY" ||
		!reflect.DeepEqual(glm.Caps, []string{"x"}) || glm.PriceNote != "own" {
		t.Errorf("既有值不得被覆盖, got %+v", glm)
	}
	if glm.PriceOut != 4.4 {
		t.Errorf("空字段应被补齐 PriceOut=4.4, got %v", glm.PriceOut)
	}
	if models[3].PriceIn != 0 || models[3].Currency != "" || models[3].ContextLength != 0 {
		t.Errorf("目录外模型应原样, got %+v", models[3])
	}

	// 归一化口径：大写/冒号/[1m] 变体命中
	for _, raw := range []string{"DEEPSEEK-V4-PRO[1m]", "deepseek-v4-pro:latest", "opencode/deepseek-v4-pro"} {
		got := enrichCatalogMeta("opencode-zen", []ModelInfo{{ID: raw}})
		if got[0].PriceIn != 0.66 || got[0].PriceOut != 1.98 || got[0].Currency != "USD" {
			t.Errorf("%q enrich = %+v, want 0.66/1.98 USD", raw, got[0])
		}
	}

	// xai：补 ctx/价/note，caps 不填；grok-tts 目录外原样
	xai := enrichCatalogMeta("xai", []ModelInfo{{ID: "grok-4.6"}, {ID: "grok-tts", Kind: "tts"}})
	if xai[0].ContextLength != 500000 || xai[0].PriceIn != 2 || xai[0].PriceOut != 6 || len(xai[0].Caps) != 0 {
		t.Errorf("xai grok-4.6 enrich = %+v", xai[0])
	}
	if xai[1].Kind != "tts" || xai[1].PriceIn != 0 || xai[1].ContextLength != 0 {
		t.Errorf("grok-tts 应原样, got %+v", xai[1])
	}

	// deepseek：caps 补齐
	ds := enrichCatalogMeta("deepseek", []ModelInfo{{ID: "deepseek-v4-pro"}})
	if ds[0].ContextLength != 1000000 || ds[0].MaxOutput != 384000 ||
		!reflect.DeepEqual(ds[0].Caps, []string{"reasoning", "tools", "json"}) {
		t.Errorf("deepseek-v4-pro enrich = %+v", ds[0])
	}

	// 目录外引擎无效果（opencode-go 订阅制拍板不进目录；custom 同理）
	unchanged := []ModelInfo{{ID: "deepseek-v4-pro", PriceIn: 1}, {ID: "grok-4.6"}}
	for _, eng := range []string{"opencode-go", "custom-x", "glm", "ollama"} {
		got := enrichCatalogMeta(eng, append([]ModelInfo(nil), unchanged...))
		if !reflect.DeepEqual(got, unchanged) {
			t.Errorf("enrichCatalogMeta(%q) 应无效果, got %+v", eng, got)
		}
	}
}

// ── fetchModels 接线：enrich 进动态列表并随 saveState 持久化 ─────

func TestFetchModels_EnrichesFromCatalogAndPersists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "deepseek-v4-flash", "owned_by": "deepseek"},
				{"id": "totally-unknown-model", "owned_by": "deepseek"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	statePath := filepath.Join(t.TempDir(), "engines.json")
	m := NewManager("", "")
	if err := m.LoadState(statePath); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if err := m.SaveEngine(EngineConfig{ID: "deepseek", BaseURL: srv.URL, Enabled: true}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	models, err := m.RefreshModels(context.Background(), "deepseek")
	if err != nil {
		t.Fatalf("RefreshModels: %v", err)
	}
	byID := map[string]ModelInfo{}
	for _, mo := range models {
		byID[mo.ID] = mo
	}
	flash := byID["deepseek-v4-flash"]
	if flash.PriceIn != 0.44 || flash.PriceOut != 1.32 || flash.Currency != "USD" ||
		flash.ContextLength != 1000000 || flash.MaxOutput != 384000 || len(flash.Caps) != 3 {
		t.Errorf("刷新后 deepseek-v4-flash 元数据 = %+v", flash)
	}
	if byID["totally-unknown-model"].PriceIn != 0 || byID["totally-unknown-model"].Currency != "" {
		t.Errorf("目录外模型不应被改写: %+v", byID["totally-unknown-model"])
	}
	// 落盘 JSON 含新元数据字段（saveState 自动带出）
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Engines map[string]EngineConfig `json:"engines"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	var flashSaved *ModelInfo
	for i := range state.Engines["deepseek"].Models {
		if state.Engines["deepseek"].Models[i].ID == "deepseek-v4-flash" {
			flashSaved = &state.Engines["deepseek"].Models[i]
		}
	}
	if flashSaved == nil || flashSaved.PriceIn != 0.44 || flashSaved.ContextLength != 1000000 {
		t.Errorf("落盘 JSON 应含目录元数据字段: %s", data)
	}

	// GLM 静态目录分支不走 enrich 公共出口：GLM 目录语义原样（无双重 enrich）
	glmModels, err := m.fetchModels(context.Background(), &EngineConfig{ID: "glm", Type: EngineGLM, BaseURL: GLMBaseURLStd})
	if err != nil {
		t.Fatalf("fetchModels(glm): %v", err)
	}
	glmByID := map[string]ModelInfo{}
	for _, mo := range glmModels {
		glmByID[mo.ID] = mo
	}
	if glmByID["glm-5.3"].PriceIn != 0 || glmByID["glm-5.3"].Currency != "" {
		t.Errorf("GLM 静态目录不应被通用 enrich 改写: %+v", glmByID["glm-5.3"])
	}
	if glmByID["glm-ocr"].PriceIn != 0.2 || glmByID["glm-ocr"].Currency != "CNY" {
		t.Errorf("GLM 目录自有价格应保持, got %+v", glmByID["glm-ocr"])
	}
}

// keysOf map 键列表（测试辅助）。
func keysOf(m map[string][]glmCatalogEntry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
