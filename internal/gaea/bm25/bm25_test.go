package bm25

import (
	"reflect"
	"testing"
)

func TestTokenizeCJK(t *testing.T) {
	got := Tokenize("HP300 高频液压振动锤 台班")
	want := []string{"hp300", "高频液压振动锤", "高频", "频液", "液压", "压振", "振动", "动锤", "台班"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tokenize = %v, want %v", got, want)
	}
}

func TestRankerOrdersByRelevance(t *testing.T) {
	docs := []Doc{
		{ID: 0, Text: "通用配件 台班 振动锤 桩基"},          // 命中 2 个查询二元组
		{ID: 1, Text: "HP300 高频液压振动锤 台班 300kW"},   // 命中 4 个查询二元组
		{ID: 2, Text: "P.O 42.5 水泥 吨"},                 // 不命中
	}
	r := NewRanker(docs)
	got := r.Rank("液压振动锤")
	if len(got) == 0 {
		t.Fatal("expected ranked results")
	}
	if got[0].ID != 1 {
		t.Errorf("top result = %d, want 1（标题命中更多二元组）: %+v", got[0].ID, got)
	}
	if got[1].ID != 0 {
		t.Errorf("second result = %d, want 0: %+v", got[1].ID, got)
	}
}

func TestRankerNoMatch(t *testing.T) {
	r := NewRanker([]Doc{{ID: 0, Text: "水泥 吨 480 元"}})
	if got := r.Rank("液压振动锤"); len(got) != 0 {
		t.Errorf("expected no match, got %+v", got)
	}
}

func TestRankerCostDocs(t *testing.T) {
	docs := []Doc{
		{ID: 0, Text: "hp300 HP300 高频液压振动锤 台班"},
		{ID: 1, Text: "vib2 液压振动锤 液压振动锤配件 件"},
	}
	got := NewRanker(docs).Rank("液压振动锤")
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %+v", got)
	}
	if got[0].ID != 1 {
		t.Errorf("top = %d, want 1（标题命中两次）: %+v", got[0].ID, got)
	}
}
