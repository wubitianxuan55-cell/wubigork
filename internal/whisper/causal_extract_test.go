package whisper

import "testing"

// TestExtractCausalTriples_YinweiSuoyi 因为…所以… → {因, 导致, 果}。
func TestExtractCausalTriples_YinweiSuoyi(t *testing.T) {
	f := &Fact{MemoryFact: MemoryFact{
		Summary: "因为加班太多，所以最近睡不好", Confidence: 0.9,
	}}
	tris := extractCausalTriples(f)
	if len(tris) != 1 {
		t.Fatalf("应提取 1 条因果三元组, got %d: %+v", len(tris), tris)
	}
	tp := tris[0]
	if tp.Subject != "加班太多" || tp.Predicate != "导致" || tp.Object != "最近睡不好" {
		t.Errorf("因果三元组不符: %+v", tp)
	}
}

// TestExtractCausalTriples_DirectAndConjunctionStrip X导致Y + 剥离「因为」前缀。
func TestExtractCausalTriples_DirectAndConjunctionStrip(t *testing.T) {
	f := &Fact{MemoryFact: MemoryFact{
		Summary: "因为加班导致睡不好", Confidence: 0.9,
	}}
	tris := extractCausalTriples(f)
	if len(tris) != 1 {
		t.Fatalf("应提取 1 条, got %d: %+v", len(tris), tris)
	}
	tp := tris[0]
	if tp.Subject != "加班" || tp.Object != "睡不好" {
		t.Errorf("因侧应剥离「因为」: %+v", tp)
	}
}

// TestExtractCausalTriples_RangWo 让我/使我 模式。
func TestExtractCausalTriples_RangWo(t *testing.T) {
	f := &Fact{MemoryFact: MemoryFact{
		Summary: "项目让我压力很大", Confidence: 0.8,
	}}
	tris := extractCausalTriples(f)
	if len(tris) != 1 {
		t.Fatalf("应提取 1 条, got %d", len(tris))
	}
	if tris[0].Subject != "项目" || tris[0].Object != "压力很大" {
		t.Errorf("因果三元组不符: %+v", tris[0])
	}
}

// TestExtractCausalTriples_NoPattern 无因果模式 → 空。
func TestExtractCausalTriples_NoPattern(t *testing.T) {
	f := &Fact{MemoryFact: MemoryFact{Summary: "今天天气不错，去公园走了走"}}
	if got := extractCausalTriples(f); len(got) != 0 {
		t.Fatalf("无因果模式应返回空, got %+v", got)
	}
}

// TestExtractCausalTriples_EmotionCarried 因果三元组同样携带情绪维度。
func TestExtractCausalTriples_EmotionCarried(t *testing.T) {
	f := &Fact{MemoryFact: MemoryFact{
		Summary:          "加班导致很累",
		EmotionalContext: &EmotionalContext{Intensity: 0.8, Valence: -0.4},
	}}
	tp := extractCausalTriples(f)[0]
	if tp.EmotionLabel != "负面" || tp.Valence != -0.4 {
		t.Errorf("因果三元组情绪未携带: %+v", tp)
	}
}

// TestExtractTriples_IncludesCausal 摄入链路：事实含因果模式 → KG 出现「导致」边。
func TestExtractTriples_IncludesCausal(t *testing.T) {
	fs := NewFactStore()
	f := fs.Add(MemoryFact{
		ID: "fCausal", Subject: "工作", Subcategory: "CAREER", Domain: "pursuits",
		Summary:    "因为项目赶进度，所以最近很焦虑",
		Confidence: 0.9, SourceSessionID: "s1", SourceTurnIndex: 1,
		EmotionalContext: &EmotionalContext{Intensity: 0.7, Valence: -0.3},
	})
	kg := NewKnowledgeGraph()
	p := NewMemoryIngestPipeline(nil)
	p.extractTriples(fs, "s1", 1, kg)

	all := kg.ListAll()
	found := false
	for _, tp := range all {
		if tp.Predicate == "导致" && tp.Subject == "项目赶进度" && tp.Object == "最近很焦虑" {
			found = true
			if tp.EmotionLabel != "负面" {
				t.Errorf("因果边情绪应为负面: %+v", tp)
			}
		}
	}
	if !found {
		t.Errorf("KG 应含因果边, got %+v", all)
	}
	_ = f
}
