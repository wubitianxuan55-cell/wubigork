package search

import (
	"strings"
	"testing"
)

func TestTokenizeBigram(t *testing.T) {
	toks := Tokenize("土壤修复方案编制")
	joined := strings.Join(toks, "|")
	for _, want := range []string{"土壤修复方案编制", "土壤", "壤修", "修复", "复方", "方案", "案编", "编制"} {
		if !strings.Contains(joined, want) {
			t.Errorf("tokenize missing %q: %s", want, joined)
		}
	}
}

func TestTfidfSearchRanksRelevant(t *testing.T) {
	idx := NewTfidfIndex()
	idx.Build([]Doc{
		{ID: "soil", Text: "土壤污染状况调查 风险评估 修复方案 成本测算"},
		{ID: "bid", Text: "投标文件编制 商务标 技术标 报价清单"},
		{ID: "voice", Text: "语音识别 语音合成 情感交互 陪伴对话"},
	})

	// 土壤相关查询 → soil 排第一
	res := idx.Search("土壤修复方案", 3, 0.05)
	if len(res) == 0 {
		t.Fatal("no results")
	}
	if res[0].ID != "soil" {
		t.Errorf("top result = %s, want soil (got %+v)", res[0].ID, res)
	}

	// 投标查询 → bid 排第一
	res2 := idx.Search("投标文件 报价", 3, 0.05)
	if len(res2) == 0 || res2[0].ID != "bid" {
		t.Errorf("bid query top = %+v", res2)
	}

	// 不相关查询 → 分数低或为空
	res3 := idx.Search("美食菜谱", 3, 0.2)
	for _, r := range res3 {
		if r.Score >= 0.5 {
			t.Errorf("unrelated query scored high: %+v", r)
		}
	}
}

func TestCosine(t *testing.T) {
	a := Vector{"x": 1, "y": 0}
	b := Vector{"x": 1, "y": 0}
	c := Vector{"x": 0, "y": 1}
	if got := Cosine(a, b); got != 1.0 {
		t.Errorf("identical vectors cosine = %v", got)
	}
	if got := Cosine(a, c); got != 0.0 {
		t.Errorf("orthogonal vectors cosine = %v", got)
	}
}
