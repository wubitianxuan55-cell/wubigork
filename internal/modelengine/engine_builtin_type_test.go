package modelengine

// 内置引擎类型查证口的对账与消费回归。
//
// 背景：strata/modelhub 加入「本地引擎」族时，代码里有几处按引擎名硬编码的名单
// 漏改（estimatePrice 的 `case "ollama", "herdsman"`、app 的 isLocalEngine
// `case "herdsman", "ollama", "cosyvoice"`）。本文件钉住单一真相口
// BuiltinEngineTypeOf 与预置种子的对账、以及本地引擎不计价的消费口径。

import "testing"

// TestBuiltinEngineTypeOf_MatchesSeeds 对账：名单必须与 NewManager 的预置种子
// 逐条一致——新增内置引擎时两处同改，漏改即红（这是「名单漏项」这一类回归的
// 唯一防线：其余消费点都改走本函数，不再各自维护名单）。
func TestBuiltinEngineTypeOf_MatchesSeeds(t *testing.T) {
	m := NewManager("", "")

	// 正向：每个预置种子都能被名单还原出同一个 Type。
	for id, eng := range m.engines {
		if got := BuiltinEngineTypeOf(id); got != eng.Type {
			t.Errorf("BuiltinEngineTypeOf(%q) = %q, want %q（与预置种子不一致）", id, got, eng.Type)
		}
	}

	// 反向：名单不得虚报 NewManager 未种子的 ID。
	for _, id := range []string{
		"xai", "ollama", "herdsman", "deepseek", "glm", "cosyvoice",
		"modelhub", "strata", "opencode-go", "opencode-zen",
	} {
		if _, ok := m.engines[id]; !ok {
			t.Errorf("名单含 %q，但 NewManager 未种子该引擎", id)
		}
		if BuiltinEngineTypeOf(id) == "" {
			t.Errorf("BuiltinEngineTypeOf(%q) 不应为空串", id)
		}
	}

	// 未知 / 自定义（custom-* 不是内置项）：空串——其 Type 恒为 EngineCustom，
	// IsLocal() 亦为假，口径一致。
	for _, id := range []string{"", "custom-abc", "custom-", "nope", "STRATA"} {
		if got := BuiltinEngineTypeOf(id); got != "" {
			t.Errorf("BuiltinEngineTypeOf(%q) = %q, want 空串", id, got)
		}
	}
}

// TestBuiltinEngineTypeOf_LocalFamily 本地引擎族必须齐五：漏一个就会在上述
// 漏项类回归里被误判成云端（离线门控、不计价、加载预警三处同时错）。
func TestBuiltinEngineTypeOf_LocalFamily(t *testing.T) {
	local := map[string]bool{
		"ollama": true, "herdsman": true, "cosyvoice": true,
		"modelhub": true, "strata": true,
	}
	for id, want := range local {
		if got := BuiltinEngineTypeOf(id).IsLocal(); got != want {
			t.Errorf("BuiltinEngineTypeOf(%q).IsLocal() = %v, want %v", id, got, want)
		}
	}
	// 云端族反向断言
	for _, id := range []string{"xai", "deepseek", "glm", "opencode-go", "opencode-zen"} {
		if BuiltinEngineTypeOf(id).IsLocal() {
			t.Errorf("%s 不是本地引擎，IsLocal() 不应为真", id)
		}
	}
}

// TestEstimatePrice_LocalFamilyFree 本地引擎族恒不计价。用「必然命中内置前缀
// 表」的模型名（gpt-5）作探针，排除「没命中所以为 0」的假绿；末尾用非本地引擎
// 做反向对照，证明该探针确实会命中。
func TestEstimatePrice_LocalFamilyFree(t *testing.T) {
	for _, id := range []string{"ollama", "herdsman", "cosyvoice", "modelhub", "strata"} {
		if p := estimatePrice(id, "gpt-5"); p.Currency != "" || p.InputPerM != 0 || p.OutputPerM != 0 {
			t.Errorf("本地引擎 %s 不应计价，got %+v", id, p)
		}
		if c, cur := estimatedCostFor(id, "gpt-5", 1_000_000, 1_000_000); c != 0 || cur != "" {
			t.Errorf("本地引擎 %s 不应产生费用，got %v/%q", id, c, cur)
		}
	}

	// 反向对照：同一模型名在非本地引擎上必须命中（否则上面是假绿）。
	if p := estimatePrice("custom-probe", "gpt-5"); p.Currency == "" {
		t.Fatal("对照失败：gpt-5 在非本地引擎上应命中内置价表，测试失去证明力")
	}
}
