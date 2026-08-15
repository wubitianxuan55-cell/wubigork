package search

import "testing"

// TestSearchCapsAtMaxResults 结果上限：调用方即使传 topK=全部文档数，结果也
// 不会超过 MaxResults（防全量返回）。
func TestSearchCapsAtMaxResults(t *testing.T) {
	docs := make([]Doc, 0, 150)
	for i := 0; i < 150; i++ {
		docs = append(docs, Doc{ID: "d" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Text: "关键词甲 乙"})
	}
	idx := NewTfidfIndex()
	idx.Build(docs)

	res := idx.Search("关键词", len(docs), 0.05)
	if len(res) != MaxResults {
		t.Fatalf("应截断到 MaxResults=%d，实际 %d", MaxResults, len(res))
	}
}

// TestSearchTopKNonPositive 非正 topK 不 panic 且返回空。
func TestSearchTopKNonPositive(t *testing.T) {
	idx := NewTfidfIndex()
	idx.Build([]Doc{{ID: "a", Text: "关键词 内容"}})

	for _, k := range []int{0, -1, -100} {
		if res := idx.Search("关键词", k, 0.05); res != nil {
			t.Fatalf("topK=%d 应返回 nil，实际 %+v", k, res)
		}
	}
}

// TestSearchHonorsSmallTopK 未超上限时按请求的 topK 截断。
func TestSearchHonorsSmallTopK(t *testing.T) {
	docs := make([]Doc, 0, 10)
	for i := 0; i < 10; i++ {
		docs = append(docs, Doc{ID: "d" + string(rune('a'+i)), Text: "关键词 甲 乙"})
	}
	idx := NewTfidfIndex()
	idx.Build(docs)

	if res := idx.Search("关键词", 3, 0.05); len(res) != 3 {
		t.Fatalf("topK=3 应返回 3 条，实际 %d", len(res))
	}
}
