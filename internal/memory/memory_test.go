package memory

import (
	"strconv"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/util"
)

func TestTokenize_Chinese(t *testing.T) {
	tokens := tokenize("青云宗坐落于苍山")
	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d: %v", len(tokens), tokens)
	}
	// 应该有 2-gram: "青云", "云宗", "坐落", "落于", "于苍", "苍山"
	hasBigram := false
	for _, tok := range tokens {
		if tok == "青云" || tok == "苍山" {
			hasBigram = true
		}
	}
	if !hasBigram {
		t.Fatalf("expected Chinese bigrams, got: %v", tokens)
	}
}

func TestTokenize_Mixed(t *testing.T) {
	tokens := tokenize("Elara walked into 青云宗")
	if len(tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d: %v", len(tokens), tokens)
	}
	hasEnglish := false
	hasChinese := false
	for _, tok := range tokens {
		if tok == "elara" || tok == "walked" {
			hasEnglish = true
		}
		if strings.ContainsAny(tok, "青云宗") || tok == "云宗" {
			hasChinese = true
		}
	}
	if !hasEnglish || !hasChinese {
		t.Fatalf("expected mixed tokens, got: %v", tokens)
	}
}

func TestBM25_AddSearch(t *testing.T) {
	idx := NewIndex()

	idx.Add(Memory{ID: "ch1", ChapterNum: 1, Text: "Elara enters the 青云宗 for the first time"})
	idx.Add(Memory{ID: "ch2", ChapterNum: 2, Text: "青云宗 holds a grand ceremony"})
	idx.Add(Memory{ID: "ch3", ChapterNum: 3, Text: "Kael trains in the mountains"})

	// 搜索 "青云宗"
	results := idx.Search("青云宗", 3)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results for 青云宗, got %d", len(results))
	}
	if results[0].ChapterNum != 2 && results[0].ChapterNum != 1 {
		t.Fatalf("highest scoring should be ch1 or ch2, got ch%d", results[0].ChapterNum)
	}

	// 搜索不存在的词
	results2 := idx.Search("zzzznonexistent", 3)
	if len(results2) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results2))
	}
}

func TestBM25_EmptyIndex(t *testing.T) {
	idx := NewIndex()
	results := idx.Search("anything", 5)
	if len(results) != 0 {
		t.Fatalf("expected 0 results from empty index, got %d", len(results))
	}
}

func TestEstimateTokens(t *testing.T) {
	chinese := "这是一段中文测试文本"
	tokens := util.EstimateTokens(chinese)
	if tokens < 5 || tokens > 20 {
		t.Fatalf("unexpected token estimate for Chinese: %d", tokens)
	}

	english := "This is a test sentence for token estimation"
	tokens2 := util.EstimateTokens(english)
	if tokens2 < 5 || tokens2 > 25 {
		t.Fatalf("unexpected token estimate for English: %d", tokens2)
	}
}

// ── 行为冻结快照（审计 GA3-04/IN2-11）─────────────────────────────

// freezeCorpus 覆盖各口径分歧维度（与 internal/gaea/bm25 冻结语料同形，
// 逐字节同源拷贝）。
var freezeCorpus = []string{
	"你好",
	"你好世界",
	"高频液压振动锤",
	"的了一个是这个",
	"这个功能如何使用",
	"HP300 高频振动锤 台班",
	"Hello World 123",
	"test_case var-name",
	"カタカナと한글",
	"，。！？、；：（）【】.,!?-",
	"～℃±",
	"混合mixed文本text2026",
	"",
	"   ",
}

// fmtTokensFreeze 把 token 序列格式化为可比对的稳定字符串。
func fmtTokensFreeze(ts []string) string {
	out := ""
	for i, t := range ts {
		if i > 0 {
			out += "|"
		}
		out += t
	}
	return out
}

// freezeDocs 检索快照语料：多查询词、df/tf/文档长度均有梯度（含罕见词
// 天阶 vs 高频词祖符的 idf 对比）。
var freezeDocs = []Memory{
	{ID: "第1章", Text: "主角林动在青阳镇获得神秘石符，修炼通背拳突破。", Category: "event"},
	{ID: "第2章", Text: "林动离开青阳镇前往炎城，路遇异兽袭击受伤。", Category: "event"},
	{ID: "第3章", Text: "炎城林家大比，林动以通背拳击败林宏成名。", Category: "event"},
	{ID: "第4章", Text: "天阶功法现世，林动机缘之下初窥天阶之门。", Category: "plot"},
	{ID: "第5章", Text: "林动手持祖符觉醒血脉，斩杀强敌。", Category: "event"},
	{ID: "第6章", Text: "小貂苏醒后大闹炎城，吞噬祖符夜闯林家宝库。", Category: "event"},
	{ID: "第7章", Text: "祖符之力大涨，林动晋阶淬体九重巅峰。", Category: "event"},
	{ID: "第8章", Text: "林动以祖符之力镇压叛乱，名震炎城。", Category: "event"},
	{ID: "第9章", Text: "天阶秘宝出世，引得各方势力争夺。林动凭借天阶造化与天阶传承连败群雄，天阶之威名震炎城，祖符亦随之大放光彩。", Category: "plot"},
	{ID: "第10章", Text: "林动得祖符，祖符认主，祖符之内更藏天阶功法。", Category: "plot"},
}

func buildFreezeIndex() *Index {
	idx := NewIndex()
	for _, m := range freezeDocs {
		idx.Add(m)
	}
	return idx
}

func formatResults(ms []Memory) string {
	out := ""
	for i, m := range ms {
		if i > 0 {
			out += ", "
		}
		out += m.ID + ":" + strconv.FormatFloat(m.Score, 'g', -1, 64)
	}
	return out
}

// TestTokenizeFreezeSnapshot 钉死本包分词口径：字母数字连排（CJK 属字母，
// 连串为**整串单 token**）+ 非 space/punct 的单 rune token + 全文 Han 相邻
// 二元组后置追加。与 bm25/gaea-memory 的「CJK 串拆整串+行内 bigram」是
// 不同算法族——行为冻结前提无法并入，刻意保留（口径对照表见 bm25.Options）。
func TestTokenizeFreezeSnapshot(t *testing.T) {
	want := []string{
		"你好|你好",
		"你好世界|你好|好世|世界",
		"高频液压振动锤|高频|频液|液压|压振|振动|动锤",
		"的了一个是这个|的了|了一|一个|个是|是这|这个",
		"这个功能如何使用|这个|个功|功能|能如|如何|何使|使用",
		"hp300|高频振动锤|台班|高频|频振|振动|动锤|台班",
		"hello|world|123",
		"test|case|var|name",
		"カタカナと한글",
		"",
		"～|℃|±",
		"混合mixed文本text2026|混合|文本",
		"",
		"",
	}
	for i, text := range freezeCorpus {
		if got := fmtTokensFreeze(tokenize(text)); got != want[i] {
			t.Errorf("tokenize(%q) = %q, want %q", text, got, want[i])
		}
	}
}

// TestIndexRetrievalFreezeSnapshot 钉死检索结果（顺序+2 位舍入分值）：
// k1=1.5/b=0.75 是本包真实参数口径（成本库/gaea 记忆为 1.2）。注：固定 b 下
// k1 不改变两两排序（g(a,n)=g(b,n') ⟺ a·n'=b·n，与 k1 无关），只改变分值
// ——故 k1 透传错误的反向证据落在分值快照上。
func TestIndexRetrievalFreezeSnapshot(t *testing.T) {
	idx := buildFreezeIndex()
	type qc struct{ query, want string }
	cases := []qc{
		{"林动 秘境", "第5章:0.34, 第8章:0.33, 第7章:0.33, 第3章:0.31"},
		{"通背拳", "第3章:3.15, 第1章:2.94"},
		{"青阳镇", "第2章:3.08, 第1章:2.94"},
		{"异兽袭击", "第2章:6.21"},
		{"天阶 祖符", "第10章:4.17, 第9章:3.94, 第4章:3.41, 第5章:1.23"},
	}
	for _, c := range cases {
		if got := formatResults(idx.Search(c.query, 4)); got != c.want {
			t.Errorf("Search(%q, 4) = [%s], want [%s]", c.query, got, c.want)
		}
	}
}
