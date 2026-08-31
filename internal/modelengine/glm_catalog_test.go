package modelengine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// glmCatalogLegacyIDs v4.11.0 内嵌目录锚定清单（22 个，一个不增不减不改）：
// 与 engine.go 旧硬编码清单逐字一致，防数据驱动化漂移。
var glmCatalogLegacyIDs = []string{
	// 文本（旗舰在前，flash 为免费档）
	"glm-5.3", "glm-5.2", "glm-5.1", "glm-5", "glm-5-turbo",
	"glm-4.7", "glm-4.7-flashx", "glm-4.6", "glm-4.5-air", "glm-4-long",
	"glm-4.7-flash", "glm-4.5-flash",
	// 多模态 / 视觉理解
	"glm-5.3-flash", "glm-4.6v",
	// 图像生成（官方 images/generations 的 model 枚举）
	"glm-image", "cogview-4-250304", "cogview-4", "cogview-3-flash",
	// 语音 / 向量 / 重排
	"glm-tts", "glm-asr-2512", "embedding-3", "rerank",
}

// TestGLMCatalogEmbeddedMatchesLegacy 内嵌 JSON 必须与旧硬编码清单逐字一致
// （22 个模型 ID 一个不增不减不改，kind 仍由 ClassifyModelKind 判定）。
func TestGLMCatalogEmbeddedMatchesLegacy(t *testing.T) {
	models := glmStaticModels()
	if len(models) != len(glmCatalogLegacyIDs) {
		t.Fatalf("内嵌目录数量 = %d, want %d", len(models), len(glmCatalogLegacyIDs))
	}
	byID := map[string]ModelInfo{}
	for i, m := range models {
		if m.ID != glmCatalogLegacyIDs[i] {
			t.Errorf("目录第 %d 项 = %q, want %q（顺序也应与旧清单一致）", i, m.ID, glmCatalogLegacyIDs[i])
		}
		if m.OwnedBy != "glm" {
			t.Errorf("%s OwnedBy = %q, want glm", m.ID, m.OwnedBy)
		}
		if m.Kind == "" {
			t.Errorf("%s Kind 为空，应由 ClassifyModelKind 兜底", m.ID)
		}
		byID[m.ID] = m
	}
	// kind 判型与旧实现一致（抽锚点，全量断言在 TestGLMStaticModels）
	if byID["glm-5.3"].Kind != "llm" || byID["glm-image"].Kind != "image" ||
		byID["glm-tts"].Kind != "tts" || byID["embedding-3"].Kind != "embedding" {
		t.Errorf("kind 判型漂移: %+v", byID)
	}
}

// TestGLMCatalogOverrideAndReload 覆盖文件：同 ID 替换 + 新 ID 追加 +
// mtime 热重载 + 坏 JSON 静默回退内嵌。
func TestGLMCatalogOverrideAndReload(t *testing.T) {
	defer setGLMCatalogPath("")
	path := filepath.Join(t.TempDir(), "glm_catalog_override.json")
	setGLMCatalogPath(path)

	// 覆盖文件不存在：内嵌目录原样返回
	models := glmStaticModels()
	if len(models) != 22 {
		t.Fatalf("无覆盖文件时目录数量 = %d, want 22", len(models))
	}

	t0 := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	write := func(content string, mod time.Time) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatal(err)
		}
	}

	// 覆盖 v1：同 ID 替换（带 kind）+ 追加新 ID
	write(`[{"id":"glm-5.3","kind":"vision-x"},{"id":"glm-6-test"}]`, t0)
	models = glmStaticModels()
	if len(models) != 23 {
		t.Fatalf("覆盖后目录数量 = %d, want 23（22 + 追加 1）", len(models))
	}
	byID := map[string]ModelInfo{}
	for _, m := range models {
		byID[m.ID] = m
	}
	if byID["glm-5.3"].Kind != "vision-x" {
		t.Errorf("覆盖文件应替换同 ID 的 kind, got %q", byID["glm-5.3"].Kind)
	}
	if byID["glm-6-test"].Kind != "llm" {
		t.Errorf("追加新 ID 应由 ClassifyModelKind 兜底判型, got %q", byID["glm-6-test"].Kind)
	}
	if _, ok := byID["glm-tts"]; !ok {
		t.Error("未覆盖的条目不应丢失")
	}

	// 覆盖 v2（mtime 变化）：内容缩回，热重载生效
	t1 := t0.Add(time.Hour)
	write(`[{"id":"glm-5.3","kind":"llm"}]`, t1)
	models = glmStaticModels()
	if len(models) != 22 {
		t.Fatalf("热重载后目录数量 = %d, want 22", len(models))
	}
	for _, m := range models {
		if m.ID == "glm-6-test" {
			t.Error("热重载后不应残留旧覆盖条目")
		}
		if m.ID == "glm-5.3" && m.Kind != "llm" {
			t.Errorf("热重载后 glm-5.3 kind = %q, want llm", m.Kind)
		}
	}

	// mtime 未变：复用缓存（行为不变，此处只验证仍正确）
	if got := glmStaticModels(); len(got) != 22 {
		t.Errorf("mtime 未变时目录数量 = %d, want 22", len(got))
	}

	// 坏 JSON：静默回退内嵌，不 panic、不体现覆盖内容
	t2 := t1.Add(time.Hour)
	write(`{"bad json,,,`, t2)
	models = glmStaticModels()
	if len(models) != 22 {
		t.Fatalf("坏 JSON 回退后目录数量 = %d, want 22", len(models))
	}
	for _, m := range models {
		if m.ID == "glm-5.3" && m.Kind != "llm" {
			t.Errorf("坏 JSON 回退后 glm-5.3 kind = %q, want 内嵌分类 llm", m.Kind)
		}
	}
}

// TestGLMAliasAnnotation coding 端点旧名注记 alias_of；std 端点不注记。
// 依据 docs.bigmodel.cn「GLM Coding Plan 套餐概览」（2026-08-31）。
func TestGLMAliasAnnotation(t *testing.T) {
	if got := GlmAliasOf("coding", "GLM-5.2"); got != "glm-5.3" {
		t.Errorf(`GlmAliasOf("coding", "GLM-5.2") = %q, want glm-5.3`, got)
	}
	if got := GlmAliasOf("std", "glm-5.2"); got != "" {
		t.Errorf("std 家族不应有别名, got %q", got)
	}
	if got := GlmAliasOf("coding", "glm-5.3"); got != "" {
		t.Errorf("新模型名不应有别名, got %q", got)
	}

	m := NewManager("", "")
	// std 家族：全部不注记
	stdModels, err := m.fetchModels(context.Background(), &EngineConfig{ID: "glm", Type: EngineGLM, BaseURL: GLMBaseURLStd})
	if err != nil {
		t.Fatalf("fetchModels(std): %v", err)
	}
	for _, mo := range stdModels {
		if mo.AliasOf != "" {
			t.Errorf("std 家族 %s 不应注记 alias_of, got %q", mo.ID, mo.AliasOf)
		}
	}

	// coding 家族：旧名按套餐概览注记，其余不注记
	if err := m.SetGlmEndpoint("coding"); err != nil {
		t.Fatalf("SetGlmEndpoint(coding): %v", err)
	}
	codingModels, err := m.fetchModels(context.Background(), &EngineConfig{ID: "glm", Type: EngineGLM, BaseURL: GLMBaseURLCoding})
	if err != nil {
		t.Fatalf("fetchModels(coding): %v", err)
	}
	want := map[string]string{
		"glm-5.2":     "glm-5.3",
		"glm-5.1":     "glm-5.3",
		"glm-5-turbo": "glm-5.3-flash",
		"glm-4.7":     "glm-5.3-flash",
	}
	annotated := 0
	for _, mo := range codingModels {
		if w, ok := want[mo.ID]; ok {
			if mo.AliasOf != w {
				t.Errorf("coding 家族 %s alias_of = %q, want %q", mo.ID, mo.AliasOf, w)
			}
			annotated++
		} else if mo.AliasOf != "" {
			t.Errorf("coding 家族 %s 不应注记 alias_of, got %q", mo.ID, mo.AliasOf)
		}
	}
	if annotated != len(want) {
		t.Errorf("注记数量 = %d, want %d", annotated, len(want))
	}
}

// TestRecordCall_CodingPointsBilling coding 端点按套餐口径记账：费用恒 0、
// 不进 TotalCost，Token 照常计入且旧名归一到服务端实际模型桶；std 端点
// 照旧按 Token 估算（glm-5.3 计价 > 0）。
func TestRecordCall_CodingPointsBilling(t *testing.T) {
	m := NewManager("", "")
	if err := m.SetGlmEndpoint("coding"); err != nil {
		t.Fatalf("SetGlmEndpoint(coding): %v", err)
	}
	m.RecordCall(ModelCallUsage{
		EngineID: "glm", Model: "glm-5.2",
		InputTokens: 1000, OutputTokens: 500, DurationMs: 100, Success: true,
	})
	// glm-4.6 现已有定价，但 coding 口径下也必须保持 0（门控验证）
	m.RecordCall(ModelCallUsage{
		EngineID: "glm", Model: "glm-4.6",
		InputTokens: 100, OutputTokens: 200, DurationMs: 100, Success: true,
	})

	sum := m.GetModelCallStats()
	if sum.TotalCalls != 2 || sum.TotalTokens != 1800 {
		t.Errorf("coding 汇总 = %d 次/%d token, want 2/1800", sum.TotalCalls, sum.TotalTokens)
	}
	if sum.TotalCost != 0 {
		t.Errorf("coding 口径 TotalCost = %v, want 0", sum.TotalCost)
	}
	for _, pm := range sum.PerModel {
		if pm.BillingMode != BillingCodingPoints {
			t.Errorf("%s BillingMode = %q, want coding_points", pm.Model, pm.BillingMode)
		}
		if pm.EstimatedCost != 0 || pm.Currency != "" {
			t.Errorf("%s coding 费用 = %v/%q, want 0/空", pm.Model, pm.EstimatedCost, pm.Currency)
		}
	}
	byModel := map[string]ModelUsageStats{}
	for _, pm := range sum.PerModel {
		byModel[pm.Model] = pm
	}
	// glm-5.2 旧名归一到 glm-5.3 桶
	g53, ok := byModel["glm-5.3"]
	if !ok {
		t.Fatalf("glm-5.2 应归一到 glm-5.3 桶, PerModel = %+v", sum.PerModel)
	}
	if g53.TotalTokens != 1500 {
		t.Errorf("glm-5.3 桶 token = %d, want 1500", g53.TotalTokens)
	}
	if _, ok := byModel["glm-5.2"]; ok {
		t.Error("coding 口径下不应保留 glm-5.2 独立桶")
	}
	// 引擎聚合：coding 单列，Tokens 计入、费用 0
	eng, ok := sum.Engines["glm@coding"]
	if !ok {
		t.Fatalf("Engines 缺少 glm@coding 单列: %+v", sum.Engines)
	}
	if eng.Calls != 2 || eng.Tokens != 1800 || eng.EstimatedCostCNY != 0 {
		t.Errorf("glm@coding 小计 = %+v, want 2 次/1800 token/0 费用", eng)
	}
	if _, ok := sum.Engines["glm"]; ok {
		t.Error("coding 口径不应计入 std 引擎小计")
	}

	// std 家族照旧：glm-5.3 计价 > 0（z.ai 官方 $1.4/$4.4，核实 2026-08-31）
	m2 := NewManager("", "")
	m2.SetUsdCnyRate(7.2)
	m2.RecordCall(ModelCallUsage{
		EngineID: "glm", Model: "glm-5.3",
		InputTokens: 1_000_000, OutputTokens: 0, DurationMs: 100, Success: true,
	})
	sum2 := m2.GetModelCallStats()
	pm := sum2.PerModel[0]
	if pm.BillingMode != "" {
		t.Errorf("std 口径 BillingMode = %q, want 空", pm.BillingMode)
	}
	if pm.Currency != "USD" || pm.EstimatedCost < 1.39 || pm.EstimatedCost > 1.41 {
		t.Errorf("std glm-5.3 费用 = %v/%q, want ~1.4 USD", pm.EstimatedCost, pm.Currency)
	}
	if got := sum2.TotalCost; got < 10.07 || got > 10.09 { // 1.4 × 7.2
		t.Errorf("std TotalCost = %v, want ~10.08", got)
	}
	if eng := sum2.Engines["glm"]; eng.EstimatedCostCNY < 10.07 || eng.EstimatedCostCNY > 10.09 {
		t.Errorf("std glm 引擎小计 = %+v, want ~10.08 CNY", sum2.Engines["glm"])
	}
	if _, ok := sum2.Engines["glm@coding"]; ok {
		t.Error("std 口径不应出现 glm@coding 单列")
	}
}

// TestGLMPricingVerified GLM 定价表抽查（来源 docs.z.ai/guides/overview/pricing，
// 核实 2026-08-31）：旗舰计价、免费档 0、官方页未列出者不计价、
// glm-5-turbo 前缀置空不被 glm-5 误匹配。
func TestGLMPricingVerified(t *testing.T) {
	if c, cur := estimatedCostFor("glm", "glm-5.3", 1e6, 0); c < 1.39 || c > 1.41 || cur != "USD" {
		t.Errorf("glm-5.3 input = %v/%q, want ~1.4 USD", c, cur)
	}
	if c, cur := estimatedCostFor("glm", "glm-5.3", 0, 1e6); c < 4.39 || c > 4.41 || cur != "USD" {
		t.Errorf("glm-5.3 output = %v/%q, want ~4.4 USD", c, cur)
	}
	if p := estimatePrice("glm", "glm-5.2"); p.InputPerM != 1.4 || p.OutputPerM != 4.4 || p.Currency != "USD" {
		t.Errorf("glm-5.2（std 独立计价）= %+v, want 1.4/4.4 USD", p)
	}
	if p := estimatePrice("glm", "glm-4.6"); p.InputPerM != 0.6 || p.OutputPerM != 2.2 || p.Currency != "USD" {
		t.Errorf("glm-4.6 = %+v, want 0.6/2.2 USD", p)
	}
	// 既有国内口径条目保持不动
	if p := estimatePrice("glm", "glm-4.7"); p.InputPerM != 2 || p.OutputPerM != 8 || p.Currency != "CNY" {
		t.Errorf("glm-4.7（既有条目）= %+v, want 2/8 CNY", p)
	}
	// 免费档：费用恒 0
	for _, id := range []string{"glm-4.7-flash", "glm-4.5-flash", "glm-4.6v-flash"} {
		if c, cur := estimatedCostFor("glm", id, 1e6, 1e6); c != 0 || cur == "" {
			t.Errorf("%s 免费档 = %v/%q, want 0/有币种", id, c, cur)
		}
	}
	// 官方页未列出 → 不计价（含 glm-5-turbo 前缀挡板）
	for _, id := range []string{"glm-5-turbo", "glm-4-long", "glm-tts", "embedding-3", "rerank", "cogview-4", "glm-image"} {
		if p := estimatePrice("glm", id); p.Currency != "" {
			t.Errorf("%s 未核实不应计价, got %+v", id, p)
		}
	}
}
