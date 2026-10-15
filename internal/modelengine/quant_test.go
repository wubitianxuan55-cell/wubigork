package modelengine

// 阶段十 10.1 量化红线测试：档位解析（llama.cpp 命名法全覆盖）+ 工具调用
// 低比特警告（出口判据「低档警告有触发用例」）。

import "testing"

func TestParseQuantLevel(t *testing.T) {
	cases := []struct {
		model string
		level int
		label string
		known bool
	}{
		{"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P", 4, "Q4_K_P", true},
		{"qwen2.5-7b-instruct-q5_k_m", 5, "Q5_K_M", true}, // 小写
		{"Qwen3.8-Flash-Next-GSQ-RCO-IQ2_XS-00001-of-00002", 2, "IQ2_XS", true},
		{"meta-llama-3.1-8b-IQ1_S", 1, "IQ1_S", true},
		{"Qwen3-8B-Q8_0", 8, "Q8_0", true},
		{"model-f16", 16, "F16", true},
		{"Qwen3-8B-BF16", 16, "BF16", true},
		{"Q2_K", 2, "Q2_K", true},
		{"IQ3_XXS", 3, "IQ3_XXS", true},
		{"grok-4.20", 0, "", false},         // 云端模型无量化标记
		{"deepseek-v4-flash", 0, "", false}, // 同上
		{"Qwen3.6-35B-A3B-MTP", 0, "", false},
	}
	for _, tc := range cases {
		q := ParseQuantLevel(tc.model)
		if q.Known != tc.known || (tc.known && (q.Level != tc.level || q.Label != tc.label)) {
			t.Fatalf("ParseQuantLevel(%q) = %+v, want level=%d label=%q known=%v", tc.model, q, tc.level, tc.label, tc.known)
		}
	}
}

func TestParseQuantLevel_MixedTakesLowest(t *testing.T) {
	// 混合标注取最低档（保守：按最差档警告）
	q := ParseQuantLevel("model-q8_0-part-iq2_xs")
	if q.Level != 2 || q.Label != "IQ2_XS" {
		t.Fatalf("混合应取最低档 2: %+v", q)
	}
}

func TestToolCallQuantWarning(t *testing.T) {
	// 低比特本地引擎 → 警告
	w := ToolCallQuantWarning(EngineOllama, "qwen3-8b-iq2_xs")
	if w == "" || !contains(w, "IQ2_XS") || !contains(w, "Q4") {
		t.Fatalf("IQ2 本地应警告: %q", w)
	}
	// Q4 达标下限 → 不警告
	if w := ToolCallQuantWarning(EngineOllama, "model-Q4_K_P"); w != "" {
		t.Fatalf("Q4 达标不应警告: %q", w)
	}
	// 云端低比特名 → 不警告（云端不经本地量化红线）
	if w := ToolCallQuantWarning(EngineXAI, "grok-iq2_xs"); w != "" {
		t.Fatalf("云端不应警告: %q", w)
	}
	// 无标记 → 不警告（宁漏勿误）
	if w := ToolCallQuantWarning(EngineHerdsman, "Qwen3.6-35B-A3B-MTP"); w != "" {
		t.Fatalf("无标记不应警告: %q", w)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// 10.2 适用面标注：IQ2_XS（strata 现有写作包）标注「写作可用、知识/agent 不
// 适用」（规划判据逐字）；Q4 甜点；无标记空串。
func TestQuantSuitability(t *testing.T) {
	cases := []struct {
		model string
		want  string
	}{
		{"Qwen3.8-Flash-Next-GSQ-RCO-IQ2_XS", "写作可用（低量化写作有实证）；知识/agent 不适用（建议 ≥Q4）"},
		{"model-Q4_K_M", "写作/知识/agent 均适用（Q4 甜点档）；复杂 agent 编排建议 Q5+"},
		{"model-Q8_0", "全档位适用（Q8+/全精度无损）"},
		{"model-IQ3_M", "写作可用；知识问答开始失真、agent 工具调用不适用（建议 ≥Q4）"},
		{"grok-4.20", ""},
	}
	for _, tc := range cases {
		if got := QuantSuitability(tc.model); got != tc.want {
			t.Fatalf("QuantSuitability(%q): got %q want %q", tc.model, got, tc.want)
		}
	}
}
