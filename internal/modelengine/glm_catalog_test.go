package modelengine

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// glmCatalogLegacyIDs v4.11.0 内嵌目录锚定清单（22 个）：B 刀 schema v2 后
// 仍是目录前 22 项（顺序不变），其后再追加官方新清单条目——防数据驱动化漂移。
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

// glmCatalogV2NewIDs B 刀按 docs.bigmodel.cn「模型概览」（2026-09-02 核实）
// 追加的官方条目（23 个，接在 legacy 清单之后）。
var glmCatalogV2NewIDs = []string{
	"glm-4.5-airx", "glm-4-flash-250414", "glm-4-flashx-250414",
	"glm-5v-turbo", "glm-4.6v-flash", "glm-4.1v-thinking-flashx",
	"glm-4.1v-thinking-flash", "glm-4v-flash", "autoglm-phone",
	"charglm-4", "emohaa", "codegeex-4", "embedding-2", "glm-ocr",
	"cogvideox-3", "vidu-q1", "vidu-2", "cogvideox-flash", "glm-tts-clone",
	"glm-4-voice", "glm-realtime", "glm-realtime-flash", "glm-realtime-air",
}

// TestGLMCatalogEmbeddedMatchesLegacy 内嵌 JSON（schema v2）锚定：legacy 22
// 项逐字保留且顺序不变，其后为官方新清单 22 项；kind 仍由 ClassifyModelKind
// 判定兜底。
func TestGLMCatalogEmbeddedMatchesLegacy(t *testing.T) {
	models := glmStaticModels()
	wantIDs := append(append([]string(nil), glmCatalogLegacyIDs...), glmCatalogV2NewIDs...)
	if len(models) != len(wantIDs) {
		t.Fatalf("内嵌目录数量 = %d, want %d", len(models), len(wantIDs))
	}
	byID := map[string]ModelInfo{}
	for i, m := range models {
		if m.ID != wantIDs[i] {
			t.Errorf("目录第 %d 项 = %q, want %q", i, m.ID, wantIDs[i])
		}
		if m.OwnedBy != "glm" {
			t.Errorf("%s OwnedBy = %q, want glm", m.ID, m.OwnedBy)
		}
		if m.Kind == "" {
			t.Errorf("%s Kind 为空，应由 ClassifyModelKind 兜底", m.ID)
		}
		byID[m.ID] = m
	}
	// kind 判型与 v1 一致（新增条目抽锚点）
	if byID["glm-5.3"].Kind != "llm" || byID["glm-image"].Kind != "image" ||
		byID["glm-tts"].Kind != "tts" || byID["embedding-3"].Kind != "embedding" {
		t.Errorf("kind 判型漂移: %+v", byID)
	}
	if byID["glm-ocr"].Kind != "ocr" || byID["glm-tts-clone"].Kind != "tts" ||
		byID["embedding-2"].Kind != "embedding" || byID["glm-4-voice"].Kind != "tts" {
		t.Errorf("新增条目 kind 兜底漂移: ocr=%q tts-clone=%q embedding-2=%q voice=%q",
			byID["glm-ocr"].Kind, byID["glm-tts-clone"].Kind, byID["embedding-2"].Kind, byID["glm-4-voice"].Kind)
	}
	// 无重复 ID
	if len(byID) != len(wantIDs) {
		t.Errorf("目录存在重复 ID: %d 项 vs %d 唯一", len(wantIDs), len(byID))
	}
}

// TestGLMCatalogEmbeddedV2Metadata schema v2 元数据锚定（官方清单 2026-09-02
// 核实）：上下文/最大输出、能力标记、免费档、核实国内价、coding 积分系数。
func TestGLMCatalogEmbeddedV2Metadata(t *testing.T) {
	byID := map[string]ModelInfo{}
	for _, m := range glmStaticModels() {
		byID[m.ID] = m
	}
	// 上下文/最大输出（官方「模型概览」分档）
	if got := byID["glm-5.3"]; got.ContextLength != 1000000 || got.MaxOutput != 128000 {
		t.Errorf("glm-5.3 ctx/out = %d/%d, want 1000000/128000", got.ContextLength, got.MaxOutput)
	}
	if got := byID["glm-5.2"]; got.ContextLength != 200000 || got.MaxOutput != 128000 {
		t.Errorf("glm-5.2 ctx/out = %d/%d, want 200000/128000", got.ContextLength, got.MaxOutput)
	}
	if got := byID["glm-4.5-air"]; got.ContextLength != 128000 || got.MaxOutput != 96000 {
		t.Errorf("glm-4.5-air ctx/out = %d/%d, want 128000/96000", got.ContextLength, got.MaxOutput)
	}
	if got := byID["glm-4-long"]; got.ContextLength != 1000000 || got.MaxOutput != 4000 {
		t.Errorf("glm-4-long ctx/out = %d/%d, want 1000000/4000", got.ContextLength, got.MaxOutput)
	}
	if got := byID["glm-4.1v-thinking-flashx"]; got.ContextLength != 64000 || got.MaxOutput != 16000 {
		t.Errorf("glm-4.1v-thinking-flashx ctx/out = %d/%d, want 64000/16000", got.ContextLength, got.MaxOutput)
	}
	// 能力标记（宁缺勿滥：官方未列不填）
	wantCaps := map[string][]string{
		"glm-5.3":       {"tools", "reasoning"},
		"glm-5.2":       {"tools", "reasoning"},
		"glm-5.3-flash": {"vision", "tools", "reasoning"},
		"glm-realtime":  {"vision", "tools", "search"},
		"glm-4.6v":      {"tools"},
		"glm-ocr":       {"json"},
	}
	for id, caps := range wantCaps {
		got := byID[id].Caps
		if len(got) != len(caps) {
			t.Errorf("%s caps = %v, want %v", id, got, caps)
			continue
		}
		for i := range caps {
			if got[i] != caps[i] {
				t.Errorf("%s caps = %v, want %v", id, got, caps)
				break
			}
		}
	}
	// 官方未列能力者不得携带 caps（宁缺勿滥）
	for _, id := range []string{"glm-5.1", "glm-4.6", "glm-5v-turbo", "autoglm-phone"} {
		if len(byID[id].Caps) != 0 {
			t.Errorf("%s 不应携带 caps, got %v", id, byID[id].Caps)
		}
	}
	// 官方免费档（8 个）
	freeSet := map[string]bool{
		"glm-4.7-flash": true, "glm-4.5-flash": true, "glm-4-flash-250414": true,
		"glm-4.6v-flash": true, "glm-4.1v-thinking-flash": true, "glm-4v-flash": true,
		"cogview-3-flash": true, "cogvideox-flash": true,
	}
	freeCount := 0
	for _, m := range glmStaticModels() {
		if m.Free {
			freeCount++
			if !freeSet[m.ID] {
				t.Errorf("%s 不应在免费档清单内却 free=true", m.ID)
			}
		}
	}
	if freeCount != len(freeSet) {
		t.Errorf("免费档数量 = %d, want %d", freeCount, len(freeSet))
	}
	// 官方核实国内价（CNY）
	if got := byID["glm-ocr"]; got.PriceIn != 0.2 || got.PriceOut != 0.2 || got.Currency != "CNY" {
		t.Errorf("glm-ocr 价 = %v/%v/%q, want 0.2/0.2 CNY", got.PriceIn, got.PriceOut, got.Currency)
	}
	if got := byID["embedding-3"]; got.PriceIn != 0.5 || got.PriceOut != 0 || got.Currency != "CNY" {
		t.Errorf("embedding-3 价 = %v/%v/%q, want 0.5/0 CNY", got.PriceIn, got.PriceOut, got.Currency)
	}
	if got := byID["glm-image"]; got.PriceIn != 0.1 || got.Currency != "CNY" || got.Unit != "call" {
		t.Errorf("glm-image 价 = %v/%q/%q, want 0.1 CNY call", got.PriceIn, got.Currency, got.Unit)
	}
	if got := byID["cogvideox-3"]; got.PriceIn != 1 || got.Currency != "CNY" || got.Unit != "call" {
		t.Errorf("cogvideox-3 价 = %v/%q/%q, want 1 CNY call", got.PriceIn, got.Currency, got.Unit)
	}
	if got := byID["glm-realtime-flash"]; got.PriceIn != 0.18 || got.Currency != "CNY" || got.Unit != "minute" {
		t.Errorf("glm-realtime-flash 价 = %v/%q/%q, want 0.18 CNY minute", got.PriceIn, got.Currency, got.Unit)
	}
	if got := byID["glm-realtime-air"]; got.PriceIn != 0.3 || got.Currency != "CNY" || got.Unit != "minute" {
		t.Errorf("glm-realtime-air 价 = %v/%q/%q, want 0.3 CNY minute", got.PriceIn, got.Currency, got.Unit)
	}
	// glm-5.3-flash 官方仅相对价：不填绝对价，price_note 记录口径
	if got := byID["glm-5.3-flash"]; got.PriceIn != 0 || got.PriceOut != 0 || got.PriceNote == "" {
		t.Errorf("glm-5.3-flash 应无绝对价且有 price_note, got %v/%v/%q", got.PriceIn, got.PriceOut, got.PriceNote)
	}
	// 其余付费模型官方绝对价未查到：不填 price（估算回退内置表）
	for _, id := range []string{"glm-5.3", "glm-5.2", "glm-5.1", "glm-5", "glm-4.6", "glm-4.6v", "glm-4.5-air"} {
		if got := byID[id]; got.PriceIn != 0 || got.PriceOut != 0 || got.Currency != "" || got.Unit != "" {
			t.Errorf("%s 不应携带目录价, got %v/%v/%q/%q", id, got.PriceIn, got.PriceOut, got.Currency, got.Unit)
		}
	}
	// coding 积分系数：仅 glm-5.3 / glm-5.3-flash 有（coding 端点仅支持这两个）
	if got := byID["glm-5.3"]; got.PointsIn != 6.9 || got.PointsCached != 1.7 || got.PointsOut != 24 || got.PointsPeak != 3 {
		t.Errorf("glm-5.3 积分系数 = %v/%v/%v/%v, want 6.9/1.7/24/3", got.PointsIn, got.PointsCached, got.PointsOut, got.PointsPeak)
	}
	if got := byID["glm-5.3-flash"]; got.PointsIn != 2.3 || got.PointsCached != 0.56 || got.PointsOut != 8 || got.PointsPeak != 1.2 {
		t.Errorf("glm-5.3-flash 积分系数 = %v/%v/%v/%v, want 2.3/0.56/8/1.2", got.PointsIn, got.PointsCached, got.PointsOut, got.PointsPeak)
	}
	for _, id := range []string{"glm-5.2", "glm-4.6", "glm-4.7"} {
		if got := byID[id]; got.PointsIn != 0 || got.PointsOut != 0 || got.PointsPeak != 0 {
			t.Errorf("%s 不应携带积分系数, got %v/%v/%v", id, got.PointsIn, got.PointsOut, got.PointsPeak)
		}
	}
}

// TestGLMCatalogOverrideAndReload 覆盖文件：同 ID 替换 + 新 ID 追加 +
// mtime 热重载 + 坏 JSON 静默回退内嵌。v1 裸数组覆盖文件仍兼容。
func TestGLMCatalogOverrideAndReload(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
	path := filepath.Join(t.TempDir(), "glm_catalog_override.json")
	setGLMCatalogPath(path)

	// 覆盖文件不存在：内嵌目录原样返回
	models := glmStaticModels()
	if len(models) != 45 {
		t.Fatalf("无覆盖文件时目录数量 = %d, want 45", len(models))
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
	if len(models) != 46 {
		t.Fatalf("覆盖后目录数量 = %d, want 46（45 + 追加 1）", len(models))
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
	if len(models) != 45 {
		t.Fatalf("热重载后目录数量 = %d, want 45", len(models))
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
	if got := glmStaticModels(); len(got) != 45 {
		t.Errorf("mtime 未变时目录数量 = %d, want 45", len(got))
	}

	// 坏 JSON：静默回退内嵌，不 panic、不体现覆盖内容
	t2 := t1.Add(time.Hour)
	write(`{"bad json,,,`, t2)
	models = glmStaticModels()
	if len(models) != 45 {
		t.Fatalf("坏 JSON 回退后目录数量 = %d, want 45", len(models))
	}
	for _, m := range models {
		if m.ID == "glm-5.3" && m.Kind != "llm" {
			t.Errorf("坏 JSON 回退后 glm-5.3 kind = %q, want 内嵌分类 llm", m.Kind)
		}
	}
}

// TestGLMCatalogOverrideV2MergeFields schema v2 对象覆盖 + merge 泛化语义：
// 同 ID 替换时字段显式给出才覆盖、否则保留内嵌值；新 ID 追加；free 条目
// 估算 {0,0,CNY}；未知字段忽略；目录价参与 estimatePrice。
func TestGLMCatalogOverrideV2MergeFields(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
	path := filepath.Join(t.TempDir(), "glm_catalog_override_v2.json")
	setGLMCatalogPath(path)

	doc := `{
	  "version": 3,
	  "updated": "2026-09-01",
	  "unknown_top_level": {"x": 1},
	  "models": [
	    {"id": "glm-4.6", "price_in": 9.9, "price_out": 19.9, "currency": "CNY", "unknown_entry_field": true},
	    {"id": "glm-5.3", "context_length": 777},
	    {"id": "glm-brand-new", "context_length": 4096, "caps": ["tools"], "free": true}
	  ]
	}`
	t0 := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, t0, t0); err != nil {
		t.Fatal(err)
	}

	models := glmStaticModels()
	if len(models) != 46 {
		t.Fatalf("v2 覆盖后目录数量 = %d, want 46（45 + 追加 1）", len(models))
	}
	byID := map[string]ModelInfo{}
	for _, m := range models {
		byID[m.ID] = m
	}
	// glm-4.6：价格显式给出→覆盖；上下文未给出→保留内嵌 200000
	if got := byID["glm-4.6"]; got.PriceIn != 9.9 || got.PriceOut != 19.9 || got.Currency != "CNY" {
		t.Errorf("glm-4.6 价应被覆盖, got %v/%v/%q", got.PriceIn, got.PriceOut, got.Currency)
	}
	if got := byID["glm-4.6"]; got.ContextLength != 200000 || got.Kind != "llm" {
		t.Errorf("未显式给出的字段应保留内嵌值, got ctx=%d kind=%q", got.ContextLength, got.Kind)
	}
	// glm-5.3：ctx 显式覆盖；caps/积分系数保留内嵌
	if got := byID["glm-5.3"]; got.ContextLength != 777 {
		t.Errorf("glm-5.3 ctx = %d, want 777", got.ContextLength)
	}
	if got := byID["glm-5.3"]; len(got.Caps) != 2 || got.PointsIn != 6.9 {
		t.Errorf("glm-5.3 caps/points 应保留内嵌值, got %v/%v", got.Caps, got.PointsIn)
	}
	// 新 ID：free 条目 + caps；kind 兜底
	got := byID["glm-brand-new"]
	if !got.Free || got.ContextLength != 4096 || len(got.Caps) != 1 || got.Caps[0] != "tools" || got.Kind != "llm" {
		t.Errorf("glm-brand-new = %+v", got)
	}
	// free 条目估算 {0,0,CNY}（目录层直接命中，不回退内置表）
	if p := estimatePrice("glm", "glm-brand-new"); p.InputPerM != 0 || p.OutputPerM != 0 || p.Currency != "CNY" {
		t.Errorf("free 条目 estimatePrice = %+v, want 0/0/CNY", p)
	}
	// 目录价参与估算
	if p := estimatePrice("glm", "glm-4.6"); p.InputPerM != 9.9 || p.OutputPerM != 19.9 || p.Currency != "CNY" {
		t.Errorf("覆盖价 estimatePrice = %+v, want 9.9/19.9/CNY", p)
	}
	// 删除覆盖文件：回退内嵌 45 条
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if models = glmStaticModels(); len(models) != 45 {
		t.Errorf("删除覆盖文件后目录数量 = %d, want 45", len(models))
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

// TestGLMPricingVerified GLM 定价抽查（迁移后数值不回归锁）：内置表条目
// （z.ai 官方 USD，核实 2026-08-31）经目录无价回退路径取值不变；官方核实
// 国内价（B 刀，2026-09-02）在目录层直接命中；免费档 0；官方页未列出者
// 不计价、glm-5-turbo 前缀置空不被 glm-5 误匹配。
func TestGLMPricingVerified(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
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
	if p := estimatePrice("glm", "glm-5.3-flash"); p.InputPerM != 0.15 || p.OutputPerM != 0.5 || p.Currency != "USD" {
		t.Errorf("glm-5.3-flash = %+v, want 0.15/0.5 USD", p)
	}
	if p := estimatePrice("glm", "glm-4.6v"); p.InputPerM != 0.3 || p.OutputPerM != 0.9 || p.Currency != "USD" {
		t.Errorf("glm-4.6v = %+v, want 0.3/0.9 USD", p)
	}
	if p := estimatePrice("glm", "glm-asr-2512"); p.InputPerM != 0.03 || p.OutputPerM != 0.03 || p.Currency != "USD" {
		t.Errorf("glm-asr-2512 = %+v, want 0.03/0.03 USD", p)
	}
	// 既有国内口径条目保持不动（目录无价回退内置表）
	if p := estimatePrice("glm", "glm-4.7"); p.InputPerM != 2 || p.OutputPerM != 8 || p.Currency != "CNY" {
		t.Errorf("glm-4.7（既有条目）= %+v, want 2/8 CNY", p)
	}
	// 免费档：目录 free 层直接命中，费用恒 0
	for _, id := range []string{"glm-4.7-flash", "glm-4.5-flash", "glm-4.6v-flash", "cogview-3-flash"} {
		if c, cur := estimatedCostFor("glm", id, 1e6, 1e6); c != 0 || cur == "" {
			t.Errorf("%s 免费档 = %v/%q, want 0/有币种", id, c, cur)
		}
	}
	// 官方核实国内价（目录层命中，B 刀新增计价）
	if p := estimatePrice("glm", "glm-ocr"); p.InputPerM != 0.2 || p.OutputPerM != 0.2 || p.Currency != "CNY" {
		t.Errorf("glm-ocr = %+v, want 0.2/0.2 CNY", p)
	}
	if p := estimatePrice("glm", "embedding-3"); p.InputPerM != 0.5 || p.OutputPerM != 0 || p.Currency != "CNY" {
		t.Errorf("embedding-3 = %+v, want 0.5/0 CNY", p)
	}
	// call/minute 单位价可从目录读出，但无法从 token 数推导：估算不计价
	if p := estimatePrice("glm", "glm-image"); p.InputPerM != 0.1 || p.Currency != "CNY" || p.Unit != "call" {
		t.Errorf("glm-image = %+v, want 0.1 CNY call", p)
	}
	for _, id := range []string{"glm-image", "cogvideox-3", "glm-realtime-flash", "glm-realtime-air"} {
		if c, cur := estimatedCostFor("glm", id, 1e6, 1e6); c != 0 || cur != "" {
			t.Errorf("%s（%q 单位）估算 = %v/%q, want 0/空", id, estimatePrice("glm", id).Unit, c, cur)
		}
	}
	// 官方页未列出 → 不计价（含 glm-5-turbo 前缀挡板）
	for _, id := range []string{"glm-5-turbo", "glm-4-long", "glm-tts", "rerank", "cogview-4"} {
		if p := estimatePrice("glm", id); p.Currency != "" {
			t.Errorf("%s 未核实不应计价, got %+v", id, p)
		}
	}
	// 前缀回退：目录前缀命中无价条目后继续回退内置表前缀规则
	if p := estimatePrice("glm", "glm-5.3x-turbo"); p.InputPerM != 1.4 || p.Currency != "USD" {
		t.Errorf("glm-5.3x-turbo 应按 glm-5.3 内置前缀计价, got %+v", p)
	}
}

// TestEstimateCostCNY_GLMValueLocks GLM 代表模型 EstimateCostCNY 锁值
// （汇率 7.2）：迁移后估算数值不得回归——glm-5.3/glm-5.3-flash/glm-4.6v
// 与迁移前一致；glm-ocr/embedding-3 为 B 刀官方核实国内价新增计价。
func TestEstimateCostCNY_GLMValueLocks(t *testing.T) {
	cases := []struct {
		model  string
		inTok  int64
		outTok int64
		want   float64 // CNY
	}{
		{"glm-5.3", 1_000_000, 0, 1.4 * 7.2},                // 10.08
		{"glm-5.3", 0, 1_000_000, 4.4 * 7.2},                // 31.68
		{"glm-5.3-flash", 1_000_000, 1_000_000, 0.65 * 7.2}, // (0.15+0.5)*7.2
		{"glm-4.6v", 1_000_000, 0, 0.3 * 7.2},               // 2.16
		{"glm-4.6", 500_000, 250_000, (0.3 + 0.55) * 7.2},
		{"glm-ocr", 1_000_000, 1_000_000, 0.4},     // CNY 直用
		{"embedding-3", 1_000_000, 0, 0.5},         // CNY 直用
		{"glm-4.7", 1_000_000, 0, 2},               // 既有国内条目
		{"glm-4.7-flash", 1_000_000, 1_000_000, 0}, // 免费档
		{"glm-image", 1_000_000, 0, 0},             // call 单位：估算不计价
	}
	for _, tc := range cases {
		if got := EstimateCostCNY("glm", tc.model, tc.inTok, tc.outTok, 7.2); math.Abs(got-tc.want) > 0.00001 {
			t.Errorf("EstimateCostCNY(glm, %s) = %.6f, want %.6f", tc.model, got, tc.want)
		}
	}
}

// TestGLMCodingAliasTable coding 端点别名表锚定（docs.bigmodel.cn
// 「GLM Coding Plan 套餐概览」，2026-09-02 复核一致）：表内容逐项锁定，
// 防止静默漂移；std 家族恒无别名。
func TestGLMCodingAliasTable(t *testing.T) {
	want := map[string]string{
		"glm-5.2":     "glm-5.3",
		"glm-5.1":     "glm-5.3",
		"glm-5-turbo": "glm-5.3-flash",
		"glm-4.7":     "glm-5.3-flash",
	}
	if len(glmCodingAlias) != len(want) {
		t.Fatalf("glmCodingAlias = %v, want 恰好 %d 项", glmCodingAlias, len(want))
	}
	for k, v := range want {
		if glmCodingAlias[k] != v {
			t.Errorf("glmCodingAlias[%q] = %q, want %q", k, glmCodingAlias[k], v)
		}
		if got := GlmAliasOf("coding", strings.ToUpper(k)); got != v {
			t.Errorf("GlmAliasOf(coding, %q 大小写不敏感) = %q, want %q", k, got, v)
		}
		if got := GlmAliasOf("std", k); got != "" {
			t.Errorf("GlmAliasOf(std, %q) = %q, want 空", k, got)
		}
	}
}

// TestModelStatsSummary_CatalogPassthrough ModelStatsSummary 透传目录版本与
// 来源（零新绑定）：默认内嵌目录 → catalog_version "2" +
// catalog_source "builtin v2 (2026-09-02)"。
func TestModelStatsSummary_CatalogPassthrough(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
	m := NewManager("", "")
	sum := m.GetModelCallStats()
	if sum.CatalogVersion != "2" {
		t.Errorf("CatalogVersion = %q, want 2", sum.CatalogVersion)
	}
	if sum.CatalogSource != "builtin v2 (2026-09-02)" {
		t.Errorf("CatalogSource = %q, want builtin v2 (2026-09-02)", sum.CatalogSource)
	}

	// 远程缓存生效 → source 变为 remote <version>
	dir := t.TempDir()
	remote := filepath.Join(dir, "glm_catalog_remote.json")
	if err := os.WriteFile(remote, []byte(`{"version":7,"updated":"2026-09-01","models":[{"id":"glm-remote-only"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	setGLMCatalogRemotePath(remote)
	sum = m.GetModelCallStats()
	if sum.CatalogVersion != "7" || sum.CatalogSource != "remote 7" {
		t.Errorf("远程生效后 = %q/%q, want 7/remote 7", sum.CatalogVersion, sum.CatalogSource)
	}

	// 覆盖文件优先级最高 → source 变为 override
	ov := filepath.Join(dir, "override.json")
	if err := os.WriteFile(ov, []byte(`{"version":9,"models":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	setGLMCatalogPath(ov)
	sum = m.GetModelCallStats()
	if sum.CatalogVersion != "9" || sum.CatalogSource != "override" {
		t.Errorf("覆盖生效后 = %q/%q, want 9/override", sum.CatalogVersion, sum.CatalogSource)
	}
}
