package whisper

import "testing"

// TestExtractBasicTriples_EntitySubjectAndEmotion BASIC_PROFILE 主语实体化 +
// 情绪随三元组落图（审计 §C 两项欠账：主语硬编码「用户」/情绪活在图外）。
func TestExtractBasicTriples_EntitySubjectAndEmotion(t *testing.T) {
	f := &Fact{MemoryFact: MemoryFact{
		Subject: "生日", Subcategory: "BASIC_PROFILE", Domain: "user_profile",
		Summary: "我的生日是 5 月 20 日", Confidence: 0.9,
		EmotionalContext: &EmotionalContext{Intensity: 0.8, Valence: 0.6},
	}}
	tris := extractBasicTriples(f)
	if len(tris) != 1 {
		t.Fatalf("应提取 1 条三元组, got %d", len(tris))
	}
	tp := tris[0]
	if tp.Subject != "生日" {
		t.Errorf("主语应实体化为「生日」, got %q", tp.Subject)
	}
	if tp.Predicate != "属性" {
		t.Errorf("谓词应为「属性」, got %q", tp.Predicate)
	}
	if tp.EmotionLabel != "正面" || tp.EmotionalIntensity != 0.8 || tp.Valence != 0.6 {
		t.Errorf("情绪维度未携带: %+v", tp)
	}
}

// TestExtractBasicTriples_NegativeEmotion 负效价 → 负面标签；关系类主语保持「用户」。
func TestExtractBasicTriples_NegativeEmotion(t *testing.T) {
	f := &Fact{MemoryFact: MemoryFact{
		Subject: "深夜emo", Subcategory: "VULNERABILITIES", Summary: "最近很低落",
		EmotionalContext: &EmotionalContext{Intensity: 0.9, Valence: -0.5},
	}}
	tp := extractBasicTriples(f)[0]
	if tp.EmotionLabel != "负面" {
		t.Errorf("负效价应标负面, got %q", tp.EmotionLabel)
	}
	if tp.Subject != "用户" {
		t.Errorf("VULNERABILITIES 主语保持「用户」, got %q", tp.Subject)
	}
}

// TestKnowledgeGraph_AddTriplePreservesEmotion AddTriple 保留情绪字段。
func TestKnowledgeGraph_AddTriplePreservesEmotion(t *testing.T) {
	kg := NewKnowledgeGraph()
	got := kg.AddTriple(Triple{
		Subject: "用户", Predicate: "喜欢", Object: "咖啡",
		EmotionLabel: "正面", EmotionalIntensity: 0.7, Valence: 0.5,
	})
	if got.ID == "" {
		t.Fatal("AddTriple 应分配 ID")
	}
	all := kg.ListAll()
	if len(all) != 1 || all[0].EmotionLabel != "正面" || all[0].Valence != 0.5 {
		t.Fatalf("AddTriple 未保留情绪: %+v", all)
	}
}

// TestQuerySubgraph_CarriesEmotionOnEdges 子图边带情绪标签（前端按情绪着色）。
func TestQuerySubgraph_CarriesEmotionOnEdges(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddTriple(Triple{Subject: "用户", Predicate: "喜欢", Object: "咖啡", EmotionLabel: "正面"})
	kg.AddTriple(Triple{Subject: "咖啡", Predicate: "产于", Object: "云南", EmotionLabel: "中性"})
	sub := kg.QuerySubgraph("用户", 1)
	if len(sub.Edges) != 1 {
		t.Fatalf("hops=1 应 1 条边, got %d", len(sub.Edges))
	}
	if sub.Edges[0].EmotionLabel != "正面" {
		t.Errorf("边应带情绪标签, got %+v", sub.Edges[0])
	}
}
