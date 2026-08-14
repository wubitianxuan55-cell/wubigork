package whisper

import (
	"testing"
)

// ─── T6-5.1: memory_consolidator.go ─────────────────────────────

// seedDistinctRawFacts 灌入 n 条 subcategory 互不相同的事实（避免 Jaccard 去重合并）。
func seedDistinctRawFacts(fs *FactStore, n int) []string {
	var ids []string
	for i := 0; i < n; i++ {
		f := fs.Add(MemoryFact{
			Domain: "user_profile", Subcategory: "SUB_" + string(rune('A'+i)), Subject: "用户",
			Summary: "用户的事实内容" + string(rune('0'+i%10)), Weight: 1, Confidence: 0.8,
		})
		ids = append(ids, f.ID)
	}
	return ids
}

func TestConsolidate_TooFewFactsReturnsZero(t *testing.T) {
	fs := NewFactStore()
	seedDistinctRawFacts(fs, consolidationMinFacts-1)
	mc := NewMemoryConsolidator()
	if got := mc.Consolidate(fs, llmStub{reply: "{}"}, nil, "s1", 1); got != 0 {
		t.Errorf("事实不足时应返回 0, got %d", got)
	}
	if fs.Count() != consolidationMinFacts-1 {
		t.Errorf("不足时不应新增事实, count = %d", fs.Count())
	}
}

func TestConsolidate_SuccessAddsInsights(t *testing.T) {
	fs := NewFactStore()
	seedDistinctRawFacts(fs, 10)
	mc := NewMemoryConsolidator()
	raw := "{\"insights\":[{\"subcategory\":\"VALUES_BELIEFS\",\"subject\":\"坚持\",\"summary\":\"用户重视坚持\",\"triggers\":[\"坚持\"]},{\"subcategory\":\"MOOD\",\"subject\":\"情绪\",\"summary\":\"用户近期情绪平稳\",\"triggers\":[\"情绪\"]},{\"subcategory\":\"LIFESTYLE\",\"subject\":\"作息\",\"summary\":\"用户习惯早睡\",\"triggers\":[\"早睡\"]}],\"associations\":[{\"fact_a_idx\":1,\"fact_b_idx\":2,\"type\":\"thematic\",\"strength\":\"medium\"}]}"

	added := mc.Consolidate(fs, llmStub{reply: raw}, nil, "s1", 7)
	if added != 3 {
		t.Fatalf("应整合 3 条洞察, got %d", added)
	}
	if fs.Count() != 13 {
		t.Errorf("整合后应有 13 条事实, got %d", fs.Count())
	}
	var consolidated int
	for _, f := range fs.ListAll() {
		if f.FactLayer == "consolidated" {
			consolidated++
			if f.SourceSessionID != "s1" || f.SourceTurnIndex != 7 {
				t.Errorf("洞察应带来源会话/轮次: %+v", f.MemoryFact)
			}
			if len(f.DerivedFrom) == 0 {
				t.Error("洞察应带 DerivedFrom 溯源")
			}
		}
	}
	if consolidated != 3 {
		t.Errorf("consolidated 层应有 3 条, got %d", consolidated)
	}
}

func TestConsolidate_InsightsSkippedAndCapped(t *testing.T) {
	fs := NewFactStore()
	seedDistinctRawFacts(fs, 10)
	mc := NewMemoryConsolidator()
	raw := "{\"insights\":[{\"subcategory\":\"\",\"subject\":\"bad\",\"summary\":\"缺子类\"},{\"subcategory\":\"MOOD\",\"subject\":\"a\",\"summary\":\"洞察1\"},{\"subcategory\":\"MOOD\",\"subject\":\"b\",\"summary\":\"洞察2\"},{\"subcategory\":\"MOOD\",\"subject\":\"c\",\"summary\":\"洞察3\"},{\"subcategory\":\"MOOD\",\"subject\":\"d\",\"summary\":\"洞察4\"},{\"subcategory\":\"MOOD\",\"subject\":\"e\",\"summary\":\"洞察5\"}],\"associations\":[]}"

	added := mc.Consolidate(fs, llmStub{reply: raw}, nil, "s1", 1)
	if added != 5 {
		t.Errorf("应整合 5 条（跳过无效+触顶）: got %d", added)
	}
}

func TestConsolidate_NilEmotionContextSafe(t *testing.T) {
	fs := NewFactStore()
	seedDistinctRawFacts(fs, 10)
	mc := NewMemoryConsolidator()
	raw := "{\"insights\":[{\"subcategory\":\"MOOD\",\"subject\":\"x\",\"summary\":\"洞察\"}],\"associations\":[]}"

	added := mc.Consolidate(fs, llmStub{reply: raw}, nil, "s1", 1)
	if added != 1 {
		t.Errorf("nil 情感上下文不应 panic, got %d", added)
	}
}

func TestConsolidate_FallbackJSONExtraction(t *testing.T) {
	fs := NewFactStore()
	seedDistinctRawFacts(fs, 10)
	mc := NewMemoryConsolidator()
	inner := "{\"insights\":[{\"subcategory\":\"GOALS\",\"subject\":\"目标\",\"summary\":\"用户有明确目标\"}],\"associations\":[]}"

	raw := "思考过程...结果是：" + inner + " ...结束"
	added := mc.Consolidate(fs, llmStub{reply: raw}, nil, "s1", 2)
	if added != 1 {
		t.Errorf("兜底 JSON 提取应整合 1 条, got %d", added)
	}
}

func TestConsolidate_InvalidAssociationsSkipped(t *testing.T) {
	fs := NewFactStore()
	seedDistinctRawFacts(fs, 10)
	mc := NewMemoryConsolidator()
	// 关联：idx 越界(0 和 99)、type 为空、strength 非法 → 全部被跳过（不 panic）
	raw := "{\"insights\":[],\"associations\":[{\"fact_a_idx\":0,\"fact_b_idx\":1,\"type\":\"thematic\",\"strength\":\"medium\"},{\"fact_a_idx\":1,\"fact_b_idx\":99,\"type\":\"x\",\"strength\":\"strong\"},{\"fact_a_idx\":1,\"fact_b_idx\":2,\"type\":\"\",\"strength\":\"medium\"},{\"fact_a_idx\":1,\"fact_b_idx\":2,\"type\":\"thematic\",\"strength\":\"bogus\"}]}"
	if got := mc.Consolidate(fs, llmStub{reply: raw}, nil, "s1", 1); got != 0 {
		t.Errorf("无有效洞察应返回 0, got %d", got)
	}
}

func TestSubcategoryToDomain(t *testing.T) {
	cases := map[string]string{
		"BASIC_PROFILE": "IDENTITY", "LIFE_STORY": "IDENTITY",
		"OUR_BOND": "SOCIAL", "FAMILY": "SOCIAL",
		"ROUTINES": "DAILY_LIFE", "HEALTH": "DAILY_LIFE",
		"CAREER": "PURSUITS", "PROJECTS": "PURSUITS",
		"MOOD": "INNER_WORLD", "TASTES": "INNER_WORLD",
		"NOW": "TEMPORAL", "PLANS": "TEMPORAL",
	}
	for sub, want := range cases {
		if got := subcategoryToDomain(sub); got != want {
			t.Errorf("subcategoryToDomain(%s) = %q, want %q", sub, got, want)
		}
	}
	if got := subcategoryToDomain("SOMETHING_NEW"); got != "INNER_WORLD" {
		t.Errorf("未知子类应回退 INNER_WORLD, got %q", got)
	}
}

func TestAssocStrength(t *testing.T) {
	cases := map[string]float64{"strong": 0.8, "medium": 0.5, "weak": 0.2, "whatever": 0}
	for s, want := range cases {
		if got := assocStrength(s); got != want {
			t.Errorf("assocStrength(%s) = %v, want %v", s, got, want)
		}
	}
}
