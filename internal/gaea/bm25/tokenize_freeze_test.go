package bm25_test

// 行为冻结快照（审计 GA3-03/GA3-04/IN2-11）：钉死各分词/BM25 口径的**现状**
// 输出，统一实现落地后这些快照必须一字不改全绿——这是「统一不改变行为」的
// 证明面。口径对照表见 bm25.go 的 Options 文档注释。

import (
	"math"
	"strconv"
	"testing"

	"github.com/gaea/gaea/internal/gaea/bm25"
	"github.com/gaea/gaea/internal/gaea/search"
	"github.com/gaea/gaea/internal/gaea/textsim"
)

// freezeCorpus 覆盖各口径的全部分歧维度：两字 CJK 串（两字去重分歧点）、
// 停用词、大小写（search 口径不小写化）、下划线/连字符（是否分词字符）、
// 片假名/谚文（bm25 族视为 CJK、internal/memory 视为字母）、纯符号
// （search 视为字段、internal/memory 输出单 rune token、bm25 族丢弃）。
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

// fmtTokens 把 token 序列格式化为可比对的稳定字符串。
func fmtTokens(ts []string) string {
	out := ""
	for i, t := range ts {
		if i > 0 {
			out += "|"
		}
		out += t
	}
	return out
}

// TestBM25TokenizeFreeze 钉死 bm25 默认口径（成本库 Ranker/Cache 用）：
// 无停用词、两字串整串即二元组不重复输出、四脚本 CJK、下划线为词字符、
// 逐 rune 小写化。
func TestBM25TokenizeFreeze(t *testing.T) {
	want := []string{
		"你好",
		"你好世界|你好|好世|世界",
		"高频液压振动锤|高频|频液|液压|压振|振动|动锤",
		"的了一个是这个|的了|了一|一个|个是|是这|这个",
		"这个功能如何使用|这个|个功|功能|能如|如何|何使|使用",
		"hp300|高频振动锤|高频|频振|振动|动锤|台班",
		"hello|world|123",
		"test_case|var|name",
		"カタカナと한글|カタ|タカ|カナ|ナと|と한|한글",
		"",
		"",
		"混合|mixed|文本|text2026",
		"",
		"",
	}
	for i, text := range freezeCorpus {
		if got := fmtTokens(bm25.Tokenize(text)); got != want[i] {
			t.Errorf("Tokenize(%q) = %q, want %q", text, got, want[i])
		}
	}
}

// TestSearchTokenizeFreeze 钉死 search 口径（TF-IDF 知识库/工作区搜索）：
// 特定标点集分字段、整词 ≥2 rune、字段内相邻 bigram（含字段内符号）、
// 全局去重、**不小写化**。与 bm25 族是不同算法族，行为冻结前提无法并入
// 单一分词器（分隔集/大小写/去重策略均不同）——刻意保留，口径见对照表。
func TestSearchTokenizeFreeze(t *testing.T) {
	want := []string{
		"你好",
		"你好世界|你好|好世|世界",
		"高频液压振动锤|高频|频液|液压|压振|振动|动锤",
		"的了一个是这个|的了|了一|一个|个是|是这|这个",
		"这个功能如何使用|这个|个功|功能|能如|如何|何使|使用",
		"HP300|HP|P3|30|00|高频振动锤|高频|频振|振动|动锤|台班",
		"Hello|He|el|ll|lo|World|Wo|or|rl|ld|123|12|23",
		"test_case|te|es|st|t_|_c|ca|as|se|var-name|va|ar|r-|-n|na|am|me",
		"カタカナと한글|カタ|タカ|カナ|ナと|と한|한글",
		"",
		"～℃±|～℃|℃±",
		"混合mixed文本text2026|混合|合m|mi|ix|xe|ed|d文|文本|本t|te|ex|xt|t2|20|02|26",
		"",
		"",
	}
	for i, text := range freezeCorpus {
		if got := fmtTokens(search.Tokenize(text)); got != want[i] {
			t.Errorf("search.Tokenize(%q) = %q, want %q", text, got, want[i])
		}
	}
}

// TestTextsimSimilarityFreeze 钉死 textsim 集合 Dice 口径（查重/合并提示）：
// Han 串+二元组、字母数字词、小写化、集合去重后 Dice。textsim 与检索分词
// 不同族（集合语义、仅 Han、无下划线词字符），评估后刻意保留——此处快照
// 证明统一期内未动它。
func TestTextsimSimilarityFreeze(t *testing.T) {
	pairs := [][2]string{
		{"液压振动锤台班", "液压振动锤台班"},
		{"高频液压振动锤", "液压振动锤"},
		{"Hello World", "hello world"},
		{"", "anything"},
		{"ABC123", "abc123"},
	}
	want := []string{"1", "0.6666666666666666", "1", "0", "1"}
	for i, p := range pairs {
		got := strconv.FormatFloat(textsim.Similarity(p[0], p[1]), 'g', -1, 64)
		if got != want[i] {
			t.Errorf("Similarity(%q,%q) = %s, want %s", p[0], p[1], got, want[i])
		}
	}
}

// ── 统一后的新 API 面（GA3-03/GA3-04/IN2-11 单源消费方口径钉死）──────

// freezeGaeaStopwords 是 internal/gaea/memory cjkStopwords 的等价内联
// （该表包内私有）：两表漂移会被 gaea/memory 的 TestTokenizeFreezeSnapshot
// 捕获（其语料与本文件 freezeCorpus 同形）。
var freezeGaeaStopwords = map[string]bool{
	"的": true, "了": true, "是": true, "在": true, "和": true,
	"也": true, "就": true, "都": true, "而": true, "及": true,
	"与": true, "或": true, "一个": true, "这个": true, "那个": true,
	"什么": true, "怎么": true, "如何": true, "为什么": true,
}

// TestTokenizeOptsGaeaMemoryCaliber 钉死 gaea 记忆口径经统一分词器的输出：
// 停用词整 token 丢弃 + 两字串不去重。与 gaea/memory 包内
// TestTokenizeFreezeSnapshot 的 pin 逐条同形 = 委托路径字节等价的跨包证明。
func TestTokenizeOptsGaeaMemoryCaliber(t *testing.T) {
	want := []string{
		"你好|你好",
		"你好世界|你好|好世|世界",
		"高频液压振动锤|高频|频液|液压|压振|振动|动锤",
		"的了一个是这个|的了|了一|个是|是这",
		"这个功能如何使用|个功|功能|能如|何使|使用",
		"hp300|高频振动锤|高频|频振|振动|动锤|台班|台班",
		"hello|world|123",
		"test_case|var|name",
		"カタカナと한글|カタ|タカ|カナ|ナと|と한|한글",
		"",
		"",
		"混合|混合|mixed|文本|文本|text2026",
		"",
		"",
	}
	for i, text := range freezeCorpus {
		got := bm25.TokenizeOpts(text, bm25.Options{Stopwords: freezeGaeaStopwords})
		if fmtTokens(got) != want[i] {
			t.Errorf("TokenizeOpts(gaea caliber)(%q) = %q, want %q", text, fmtTokens(got), want[i])
		}
	}
}

// freezeRankDocs 参数化语料：tf 与文档长度均有梯度。
var freezeRankDocs = []bm25.Doc{
	{ID: 0, Text: "通用配件 台班 振动锤 桩基"},
	{ID: 1, Text: "HP300 高频液压振动锤 台班 300kW"},
	{ID: 2, Text: "P.O 42.5 水泥 吨"},
	{ID: 3, Text: "振动锤振动锤振动锤 高频振动锤组配件"},
}

// TestRankerParamsFreeze 钉死参数化排序：1.2=成本库/gaea 记忆口径（cost
// golden 的隐含前提），1.5=故事记忆口径。同一公式单源，仅参数不同。
func TestRankerParamsFreeze(t *testing.T) {
	cases := []struct {
		k1   string
		want string
	}{
		{"1.2", "1:3.183307989837867, 3:1.6434621400363514, 0:1.493796886478353"},
		{"1.5", "1:3.189645032949688, 3:1.6933537712304867, 0:1.503126638024713"},
	}
	for _, c := range cases {
		k1, _ := strconv.ParseFloat(c.k1, 64)
		r := bm25.NewRankerOpts(freezeRankDocs, bm25.DefaultTokens, bm25.Params{K1: k1, B: 0.75})
		got := ""
		for i, s := range r.Rank("液压振动锤 配件") {
			if i > 0 {
				got += ", "
			}
			got += strconv.Itoa(s.ID) + ":" + strconv.FormatFloat(s.Score, 'g', -1, 64)
		}
		if got != c.want {
			t.Errorf("Rank(k1=%s) = [%s], want [%s]", c.k1, got, c.want)
		}
	}
}

// TestTermScoreMatchesLegacyFormula 守护单源公式与收敛前字面表达式
// （internal/memory 原实现）逐位一致——任一侧改动即红。
func TestTermScoreMatchesLegacyFormula(t *testing.T) {
	want := []float64{
		5.172857766685987,
		2.4295043650549895,
		4.332953681716501,
	}
	cases := [][5]float64{
		{3, 20, 15, 3, 100},
		{1, 5, 12, 8, 50},
		{2, 7.5, 8.25, 1.5, 37},
	}
	for i, c := range cases {
		got := bm25.Params{K1: 1.5, B: 0.75}.TermScore(c[0], c[1], c[2], c[3], c[4])
		// 收敛前 internal/memory 的字面公式（k1=1.5, b=0.75）
		idf := math.Log(1 + (c[4]-c[3]+0.5)/(c[3]+0.5))
		numerator := c[0] * (1.5 + 1)
		denominator := c[0] + 1.5*(1-0.75+0.75*c[1]/c[2])
		if legacy := idf * numerator / denominator; got != legacy {
			t.Errorf("TermScore(%v) != legacy formula (%v) for %v", got, legacy, c)
		}
		if strconv.FormatFloat(got, 'g', -1, 64) != strconv.FormatFloat(want[i], 'g', -1, 64) {
			t.Errorf("TermScore pin %d drifted: %v want %v", i, got, want[i])
		}
	}
}

// TestRankerEmpty 钉死 Empty 语义（gaea 记忆空索引判定：无可命中词条）。
func TestRankerEmpty(t *testing.T) {
	if !(*bm25.Ranker)(nil).Empty() {
		t.Error("nil ranker should be Empty")
	}
	if !bm25.NewRanker(nil).Empty() {
		t.Error("no docs should be Empty")
	}
	// 纯标点/单字符文档：建索引但无可命中词条
	if !bm25.NewRanker([]bm25.Doc{{ID: 0, Text: "。！？ a b"}}).Empty() {
		t.Error("no indexable tokens should be Empty")
	}
	if bm25.NewRanker([]bm25.Doc{{ID: 0, Text: "振动锤"}}).Empty() {
		t.Error("indexed tokens should not be Empty")
	}
}
