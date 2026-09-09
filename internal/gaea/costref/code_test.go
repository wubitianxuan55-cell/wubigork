package costref

// 明细→条目编码优先匹配测试（v4.178.0）：带码明细同码精确锚定（含归一化），
// 无码回退 EntryName/标题，matchedBy 溯源三种形态。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costproject"
)

func TestMatchEntryCodePriority(t *testing.T) {
	byName := map[string]cost.Summary{
		"dig": {Name: "dig", Title: "人工挖沟槽土方", Code: "A1-12", Price: 40},
	}
	byCode := map[string]cost.Summary{
		"A1-13": {Name: "rock", Title: "人工挖沟槽松石", Code: "A1-13", Price: 90},
	}
	byTitle := map[string]cost.Summary{
		"人工挖沟槽土方": {Name: "dig", Title: "人工挖沟槽土方", Code: "A1-12", Price: 40},
	}

	// 编码命中优先于标题/名称：同码不同标题仍精确锚定。
	e, matchedBy, ok := matchEntry(costproject.Item{Title: "挖沟槽（现场写法）", Code: "ａ１－１３"}, byName, byCode, byTitle)
	if !ok || matchedBy != "code" || e.Name != "rock" {
		t.Fatalf("code match = %+v %q %v", e, matchedBy, ok)
	}
	// 带码未命中不落标题兜底（同标题不同编码=不同子目，宁新增勿误配）。
	if _, _, ok := matchEntry(costproject.Item{Title: "人工挖沟槽土方", Code: "B9-9"}, byName, byCode, byTitle); ok {
		t.Fatal("带码未命中不应回退标题匹配")
	}
	// 无码：EntryName 精确。
	e, matchedBy, ok = matchEntry(costproject.Item{EntryName: "dig", Title: "别的"}, byName, byCode, byTitle)
	if !ok || matchedBy != "entry_name" || e.Name != "dig" {
		t.Fatalf("entry_name match = %+v %q %v", e, matchedBy, ok)
	}
	// 无码：标题归一化兜底。
	_, matchedBy, ok = matchEntry(costproject.Item{Title: "人工挖沟槽土方（一类土）"}, byName, byCode, byTitle)
	if !ok || matchedBy != "title" {
		t.Fatalf("title match = %q %v", matchedBy, ok)
	}
}
