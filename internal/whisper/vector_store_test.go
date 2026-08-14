package whisper

import (
	"testing"
)

// ─── T6-5.1: vector_store.go ────────────────────────────────────

func TestVectorStore_Tokenize(t *testing.T) {
	vs := NewVectorStore()
	f := &Fact{MemoryFact: MemoryFact{Subject: "用户", Summary: "喜欢吃辣"}}
	tokens := vs.tokenize(f)
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	for _, want := range []string{"吃辣", "喜欢", "喜欢吃辣"} {
		if !found[want] {
			t.Errorf("tokenize 应包含 %q, got %v", want, tokens)
		}
	}
}

func TestVectorStore_TokenizeQuery(t *testing.T) {
	vs := NewVectorStore()
	tokens := vs.tokenizeQuery("吃辣")
	if len(tokens) == 0 {
		t.Fatal("查询分词不应为空")
	}
	has := false
	for _, tok := range tokens {
		if tok == "吃辣" {
			has = true
		}
	}
	if !has {
		t.Errorf("查询应含「吃辣」bigram, got %v", tokens)
	}
	if got := vs.tokenizeQuery(""); len(got) != 0 {
		t.Errorf("空查询应无 token, got %v", got)
	}
}

func TestVectorStore_BuildAndSearch(t *testing.T) {
	fs := NewFactStore()
	f1 := fs.Add(MemoryFact{Subject: "食物", Summary: "用户喜欢吃辣", Weight: 1})
	f2 := fs.Add(MemoryFact{Subject: "娱乐", Summary: "用户喜欢看电影", Weight: 1})

	vs := NewVectorStore()
	vs.Build(fs.ListActive())

	results := vs.Search("吃辣", 5)
	if len(results) == 0 || results[0].FactID != f1.ID {
		t.Fatalf("搜索「吃辣」应命中 f1, got %v", results)
	}
	results2 := vs.Search("电影", 5)
	if len(results2) == 0 || results2[0].FactID != f2.ID {
		t.Fatalf("搜索「电影」应命中 f2, got %v", results2)
	}
}

func TestVectorStore_SearchNoMatchReturnsNil(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Subject: "食物", Summary: "用户喜欢吃辣", Weight: 1})
	vs := NewVectorStore()
	vs.Build(fs.ListActive())
	if got := vs.Search("不存在词", 5); got != nil {
		t.Errorf("无命中应返回 nil, got %v", got)
	}
}

func TestVectorStore_BuildIdempotentAndRebuild(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Subject: "食物", Summary: "用户喜欢吃辣", Weight: 1})
	vs := NewVectorStore()
	vs.Build(fs.ListActive())
	first := len(vs.Search("吃辣", 5))
	// 同批事实重复 Build：hash 命中，不应重建
	vs.Build(fs.ListActive())
	if len(vs.Search("吃辣", 5)) != first {
		t.Error("同批事实重复 Build 后结果应一致")
	}
	// 新增事实后重建
	fs.Add(MemoryFact{Subject: "娱乐", Summary: "用户喜欢看电影", Weight: 1})
	vs.Build(fs.ListActive())
	if got := vs.Search("电影", 5); len(got) == 0 {
		t.Error("重建后应能搜到新增事实")
	}
}

func TestVectorStore_SearchTopKTruncation(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Subject: "食物", Summary: "用户喜欢吃辣", Weight: 1})
	fs.Add(MemoryFact{Subject: "饮料", Summary: "用户很爱吃辣", Weight: 1})
	vs := NewVectorStore()
	vs.Build(fs.ListActive())
	if got := vs.Search("吃辣", 1); len(got) != 1 {
		t.Errorf("topK=1 应截断到 1 条, got %d", len(got))
	}
	// topK<=0 走默认 5
	if got := vs.Search("吃辣", 0); len(got) != 2 {
		t.Errorf("topK=0 应返回全部 2 条, got %d", len(got))
	}
}

func TestVectorStore_BuildDenseAndSearchDense(t *testing.T) {
	facts := []*Fact{
		{MemoryFact: MemoryFact{ID: "a", Status: "active"}, Active: true},
		{MemoryFact: MemoryFact{ID: "b", Status: "active"}, Active: true},
	}
	vs := NewVectorStore()
	vs.BuildDense(facts, [][]float64{{1, 0, 0}, {0, 1, 0}})
	results := vs.SearchDense([]float64{1, 0, 0}, 5)
	if len(results) != 1 || results[0].FactID != "a" {
		t.Fatalf("稠密搜索应命中 a, got %v", results)
	}
	// 正交查询无命中
	if got := vs.SearchDense([]float64{0, 0, 1}, 5); len(got) != 0 {
		t.Errorf("正交查询应无命中, got %v", got)
	}
	// 零向量 → nil
	if got := vs.SearchDense([]float64{0, 0, 0}, 5); got != nil {
		t.Errorf("零查询向量应返回 nil, got %v", got)
	}
}

func TestVectorStore_EmptyTokenFactNoPanic(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Subject: "食物", Summary: "用户喜欢吃辣", Weight: 1})
	fs.Add(MemoryFact{Subject: "", Summary: "", Weight: 1}) // 无 token
	vs := NewVectorStore()
	vs.Build(fs.ListActive()) // 不应 panic
	if got := vs.Search("吃辣", 5); len(got) != 1 {
		t.Errorf("空 token 事实不应影响搜索, got %v", got)
	}
}

func TestVectorStore_SearchDenseLengthMismatch(t *testing.T) {
	facts := []*Fact{
		{MemoryFact: MemoryFact{ID: "a", Status: "active"}, Active: true},
	}
	vs := NewVectorStore()
	vs.BuildDense(facts, [][]float64{{1, 0}})
	// 查询维度长于向量维度 → 只累加重叠部分，不 panic
	results := vs.SearchDense([]float64{1, 0, 5}, 5)
	if len(results) != 1 || results[0].FactID != "a" {
		t.Errorf("维度不匹配应安全计算, got %v", results)
	}
}

func TestVectorStore_BuildDenseSkipsInactive(t *testing.T) {
	facts := []*Fact{
		{MemoryFact: MemoryFact{ID: "a", Status: "active"}, Active: true},
		{MemoryFact: MemoryFact{ID: "b", Status: "retired"}, Active: false},
	}
	vs := NewVectorStore()
	vs.BuildDense(facts, [][]float64{{1, 0}, {0, 1}})
	results := vs.SearchDense([]float64{0, 1}, 5)
	if len(results) != 0 {
		t.Errorf("非激活事实不应被索引, got %v", results)
	}
}
