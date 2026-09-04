package novelstyle

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/util"
)

// ── 文风指纹（确定性计算） ─────────────────────────────────────────
// 本文件不调用任何 LLM，所有统计均为可从样本文本确定的纯函数。

// Fingerprint 文风指纹（确定性计算）。
type Fingerprint struct {
	FunctionWordVec   map[string]float64 `json:"function_word_vec"`  // 函数词 z 值
	SentenceLen       LenDist            `json:"sentence_len"`       // 句长分布
	ParaLen           LenDist            `json:"para_len"`           // 段长分布
	TTR1000           float64            `json:"ttr1000"`            // 1000 字滑动窗 type-token ratio
	TTRSd             float64            `json:"ttrsd"`              // 滑动窗 TTR 标准差（越小越单调）
	HapaxRatio        float64            `json:"hapax_ratio"`        // 仅现一次词占比
	LexicalEntropy    float64            `json:"lexical_entropy"`    // 词频熵
	DialogRatio       float64            `json:"dialog_ratio"`       // 对话占比
	FourCharRatio     float64            `json:"four_char_ratio"`    // 四字格密度（次/1000字）
	ConnectiveDensity float64            `json:"connective_density"` // 连接词密度（次/1000字）
	AdjAdvDensity     float64            `json:"adjadv_density"`     // 形容词/副词密度（次/1000字）
	Punctuation       Punctuation        `json:"punctuation"`
	TopBigrams        []string           `json:"top_bigrams"`
	TopTrigrams       []string           `json:"top_trigrams"`
	AuthorSignWords   []string           `json:"author_sign_words"` // 作者签名词
}

// LenDist 长度分布统计。
type LenDist struct {
	Mean          float64 `json:"mean"`
	Sd            float64 `json:"sd"`
	P10           float64 `json:"p10"`
	P90           float64 `json:"p90"`
	LongTailRatio float64 `json:"long_tail_ratio"`
}

// Punctuation 标点统计。
type Punctuation struct {
	Ellipsis            float64 `json:"ellipsis"`              // 省略号个数
	Exclam              float64 `json:"exclam"`                // 感叹号个数
	Dash                float64 `json:"dash"`                  // 破折号个数
	CommaPerSentence    float64 `json:"comma_per_sentence"`    // 每句逗号数
	QuestionPerSentence float64 `json:"question_per_sentence"` // 每句问号数
}

// 句长/段长常量（占位：须在自采语料重标）。
const (
	sentenceLongTailMin = 40.0  // 句长≥40 字记为长尾
	paraLongTailMin     = 120.0 // 段长≥120 字记为长尾
	ttrWindowSize       = 1000  // TTR 滑动窗宽（字）
)

// g 用于占位，避免未使用变量告警（保留 util 对齐）。
var _ = slog.Warn

// ComputeFingerprint 对一组中文样本计算确定性文风指纹。
func ComputeFingerprint(samples []string) (*Fingerprint, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("novelstyle: 无样本可分析")
	}
	var sb strings.Builder
	empty := true
	for _, s := range samples {
		if strings.TrimSpace(s) == "" {
			slog.Warn("novelstyle: 忽略空样本")
			continue
		}
		empty = false
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	if empty {
		return nil, fmt.Errorf("novelstyle: 样本均为空")
	}
	text := sb.String()
	runes := []rune(text)
	nonSpace := countNonSpaceRunes(runes)
	if nonSpace == 0 {
		return nil, fmt.Errorf("novelstyle: 样本无有效字符")
	}

	fp := &Fingerprint{
		FunctionWordVec: computeFunctionWordZ(text, float64(nonSpace)),
	}

	// 句长 / 段长分布
	fp.SentenceLen = lenDistFor(sentenceRanges(runes), sentenceLongTailMin)
	fp.ParaLen = lenDistFor(paragraphRanges(runes), paraLongTailMin)

	// 词汇丰富度（滑动窗 TTR）
	fp.TTR1000, fp.TTRSd = slidingTTR(runes, ttrWindowSize)

	// 词频统计
	toks := tokenize(text)
	fp.HapaxRatio, fp.LexicalEntropy = tokenStats(toks)

	// 对话占比
	fp.DialogRatio = dialogRatio(runes)

	// 密度统计（次/1000字）
	fp.FourCharRatio = float64(countWords(text, fourCharSet)) * 1000 / float64(nonSpace)
	fp.ConnectiveDensity = float64(countWords(text, connectives)) * 1000 / float64(nonSpace)
	fp.AdjAdvDensity = float64(countWords(text, adjAdvWords)) * 1000 / float64(nonSpace)

	// 标点
	fp.Punctuation = punctuationStats(text, runes)

	// n-gram
	fp.TopBigrams = topNGrams(runes, 2, 10)
	fp.TopTrigrams = topNGrams(runes, 3, 10)

	// 作者签名词
	fp.AuthorSignWords = authorSignWords(toks)

	return fp, nil
}

// countNonSpaceRunes 统计非空白 rune 数。
func countNonSpaceRunes(rs []rune) int {
	c := 0
	for _, r := range rs {
		if !unicode.IsSpace(r) {
			c++
		}
	}
	return c
}

// computeFunctionWordZ 计算函数词 z 值。
// 说明：单文本无外部参考语料，故 z 值在「函数词频率向量」内部标准化
// （均值=0，标准差=1），刻画函数词使用形状；Delta 比较两个指纹该形状差异。
func computeFunctionWordZ(text string, totalChars float64) map[string]float64 {
	freq := make(map[string]float64, len(functionWords))
	counts := map[string]int{}
	for _, t := range tokenize(text) {
		if _, ok := functionWordsSet[t]; ok {
			counts[t]++
		}
	}
	for _, w := range functionWords {
		if totalChars <= 0 {
			freq[w] = 0
			continue
		}
		freq[w] = float64(counts[w]) * 1000 / totalChars
	}
	return zscoreMap(freq)
}

// zscoreMap 对 map 中的数值向量做 z 标准化（mean=0,std=1）。std 过小则全返回 0。
func zscoreMap(m map[string]float64) map[string]float64 {
	if len(m) == 0 {
		return map[string]float64{}
	}
	sum, sumSq := 0.0, 0.0
	for _, v := range m {
		sum += v
		sumSq += v * v
	}
	n := float64(len(m))
	mean := sum / n
	var std float64
	v := sumSq/n - mean*mean
	if v > 0 {
		std = math.Sqrt(v)
	}
	if std < 1e-9 {
		out := make(map[string]float64, len(m))
		for k := range m {
			out[k] = 0
		}
		return out
	}
	out := make(map[string]float64, len(m))
	for k, x := range m {
		out[k] = (x - mean) / std
	}
	return out
}

// lenDistFor 依据 rune 区间列表计算长度分布。
func lenDistFor(ranges [][2]int, longTailMin float64) LenDist {
	vals := make([]float64, 0, len(ranges))
	for _, r := range ranges {
		vals = append(vals, float64(r[1]-r[0]))
	}
	if len(vals) == 0 {
		return LenDist{}
	}
	sort.Float64s(vals)
	ld := LenDist{
		Mean: mean(vals),
		Sd:   stddev(vals),
		P10:  percentile(vals, 0.10),
		P90:  percentile(vals, 0.90),
	}
	if longTailMin > 0 {
		c := 0
		for _, v := range vals {
			if v >= longTailMin {
				c++
			}
		}
		ld.LongTailRatio = float64(c) / float64(len(vals))
	}
	return ld
}

// sentenceRanges 按 。！？…（含 ASCII .!?）切分句子，返回 rune 区间。
func sentenceRanges(runes []rune) [][2]int {
	var out [][2]int
	start := 0
	for i, r := range runes {
		if isSentenceEnd(r) {
			if i > start {
				out = append(out, [2]int{start, i})
			}
			start = i + 1
		}
	}
	if start < len(runes) {
		out = append(out, [2]int{start, len(runes)})
	}
	return out
}

// paragraphRanges 按换行切分段落，返回非空段落的 rune 区间。
func paragraphRanges(runes []rune) [][2]int {
	var out [][2]int
	start := 0
	for i, r := range runes {
		if r == '\n' {
			if i > start && !isAllSpace(runes[start:i]) {
				out = append(out, [2]int{start, i})
			}
			start = i + 1
		}
	}
	if start < len(runes) && !isAllSpace(runes[start:]) {
		out = append(out, [2]int{start, len(runes)})
	}
	return out
}

// isAllSpace 判断 rune 区间是否全为空白。
func isAllSpace(rs []rune) bool {
	for _, r := range rs {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// isSentenceEnd 判断 rune 是否为句子结束标点。
func isSentenceEnd(r rune) bool {
	switch r {
	case '。', '！', '？', '…', '.', '!', '?':
		return true
	}
	return false
}

// slidingTTR 计算 1000 字滑动窗的 TTR 均值与标准差。
func slidingTTR(runes []rune, winSize int) (avg, sd float64) {
	n := len(runes)
	if n == 0 {
		return 0, 0
	}
	step := util.Max(1, winSize/2)
	var ttrs []float64
	if n <= winSize {
		ttrs = append(ttrs, ttrOf(tokenize(string(runes))))
	} else {
		for start := 0; start+winSize <= n; start += step {
			ttrs = append(ttrs, ttrOf(tokenize(string(runes[start:start+winSize]))))
		}
		// 余下尾部窗
		if (n-winSize)%step != 0 {
			ttrs = append(ttrs, ttrOf(tokenize(string(runes[n-winSize:]))))
		}
	}
	if len(ttrs) == 0 {
		return 0, 0
	}
	return mean(ttrs), stddev(ttrs)
}

// ttrOf 词序列的 type-token ratio。
func ttrOf(toks []string) float64 {
	if len(toks) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(toks))
	for _, t := range toks {
		set[t] = struct{}{}
	}
	return float64(len(set)) / float64(len(toks))
}

// tokenStats 词序列的 hapax 占比（仅现一次词占比）与词频熵（bit）。
func tokenStats(toks []string) (hapax, entropy float64) {
	if len(toks) == 0 {
		return 0, 0
	}
	counts := map[string]int{}
	for _, t := range toks {
		counts[t]++
	}
	n := float64(len(toks))
	hap := 0
	for _, c := range counts {
		if c == 1 {
			hap++
		}
	}
	hapax = float64(hap) / n
	entropy = 0
	for _, c := range counts {
		p := float64(c) / n
		entropy -= p * math.Log2(p)
	}
	return hapax, entropy
}

// dialogRatio 计算含引号直接引语字符占比。
func dialogRatio(runes []rune) float64 {
	nonSpace := countNonSpaceRunes(runes)
	if nonSpace == 0 {
		return 0
	}
	inQuote := false
	dialogChars := 0
	for _, r := range runes {
		switch r {
		case '「', '『', '“', '‘':
			if !inQuote {
				inQuote = true
			}
		case '」', '』', '”', '’':
			if inQuote {
				inQuote = false
			}
		default:
			if inQuote && !unicode.IsSpace(r) {
				dialogChars++
			}
		}
	}
	return float64(dialogChars) / float64(nonSpace)
}

// countWords 统计多个目标词在文本中的非重叠出现次数之和。
func countWords(text string, words []string) int {
	total := 0
	for _, w := range words {
		if w == "" {
			continue
		}
		total += strings.Count(text, w)
	}
	return total
}

// punctuationStats 标点统计。
func punctuationStats(text string, runes []rune) Punctuation {
	sentences := sentenceRanges(runes)
	sentCount := float64(len(sentences))
	if sentCount == 0 {
		sentCount = 1
	}
	comma := 0
	question := 0
	for _, r := range runes {
		switch r {
		case '，', ',':
			comma++
		case '？', '?':
			question++
		}
	}
	return Punctuation{
		Ellipsis:            float64(countPunct(text, []string{"……", "..."})),
		Exclam:              float64(countRune(runes, '！') + countRune(runes, '!')),
		Dash:                float64(countPunct(text, []string{"——", "—", "--"})),
		CommaPerSentence:    float64(comma) / sentCount,
		QuestionPerSentence: float64(question) / sentCount,
	}
}

// countPunct 统计标点序列出现次数（非重叠，用于多字标点）。
func countPunct(text string, marks []string) int {
	total := 0
	for _, m := range marks {
		if m == "" {
			continue
		}
		total += strings.Count(text, m)
	}
	return total
}

// countRune 统计单个 rune 出现次数。
func countRune(runes []rune, target rune) int {
	c := 0
	for _, r := range runes {
		if r == target {
			c++
		}
	}
	return c
}

// topNGrams 计算出现次数最多的连续 n 文法（跳过含空白窗）。
func topNGrams(runes []rune, n, limit int) []string {
	counts := map[string]int{}
	for i := 0; i+n <= len(runes); i++ {
		hasSpace := false
		for j := 0; j < n; j++ {
			if unicode.IsSpace(runes[i+j]) {
				hasSpace = true
				break
			}
		}
		if hasSpace {
			continue
		}
		counts[string(runes[i:i+n])]++
	}
	type kv struct {
		s string
		c int
	}
	var arr []kv
	for s, c := range counts {
		arr = append(arr, kv{s, c})
	}
	sort.Slice(arr, func(a, b int) bool {
		if arr[a].c != arr[b].c {
			return arr[a].c > arr[b].c
		}
		return arr[a].s < arr[b].s
	})
	if limit > len(arr) {
		limit = len(arr)
	}
	out := make([]string, 0, limit)
	for _, e := range arr[:limit] {
		out = append(out, e.s)
	}
	return out
}

// authorSignWords 作者签名词：高频、非停用词、长度≥2 的词。
func authorSignWords(toks []string) []string {
	counts := map[string]int{}
	for _, t := range toks {
		if len([]rune(t)) < 2 {
			continue
		}
		if _, ok := stopwordSet[t]; ok {
			continue
		}
		if _, ok := functionWordsSet[t]; ok {
			continue
		}
		counts[t]++
	}
	type kv struct {
		s string
		c int
	}
	var arr []kv
	for s, c := range counts {
		if c >= 2 {
			arr = append(arr, kv{s, c})
		}
	}
	sort.Slice(arr, func(a, b int) bool {
		if arr[a].c != arr[b].c {
			return arr[a].c > arr[b].c
		}
		return arr[a].s < arr[b].s
	})
	limit := 10
	if len(arr) < limit {
		limit = len(arr)
	}
	out := make([]string, 0, limit)
	for _, e := range arr[:limit] {
		out = append(out, e.s)
	}
	return out
}

// ── 数值助手 ────────────────────────────────────────────────

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

func stddev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	m := mean(vals)
	s := 0.0
	for _, v := range vals {
		d := v - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(vals)))
}

// percentile 取已排序切片的分位数（最近索引）。
func percentile(sorted []float64, q float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	idx := int(math.Round(q * float64(n-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}

// ── Burrows Delta ─────────────────────────────────────────

// Delta Burrows Delta：越小越像参考风格。
// Delta = (1/n)*Σ|z(observed) − z(reference)|。
func Delta(observed, reference *Fingerprint) float64 {
	if observed == nil || reference == nil {
		return 0
	}
	keys := map[string]struct{}{}
	for k := range observed.FunctionWordVec {
		keys[k] = struct{}{}
	}
	for k := range reference.FunctionWordVec {
		keys[k] = struct{}{}
	}
	if len(keys) == 0 {
		return 0
	}
	sum := 0.0
	for k := range keys {
		sum += math.Abs(observed.FunctionWordVec[k] - reference.FunctionWordVec[k])
	}
	return sum / float64(len(keys))
}

// ── 序列化 ────────────────────────────────────────────────

// ToJSON 序列化指纹为 JSON。
func (f *Fingerprint) ToJSON() ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("novelstyle: 空指纹")
	}
	return json.Marshal(f)
}

// LoadFingerprint 从 JSON 字节加载指纹。
func LoadFingerprint(jsonBytes []byte) (*Fingerprint, error) {
	if len(jsonBytes) == 0 {
		return nil, fmt.Errorf("novelstyle: JSON 为空")
	}
	var fp Fingerprint
	if err := json.Unmarshal(jsonBytes, &fp); err != nil {
		return nil, fmt.Errorf("novelstyle: 解析指纹失败: %s", util.Truncate(err.Error(), 120))
	}
	return &fp, nil
}
