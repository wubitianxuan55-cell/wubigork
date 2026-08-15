package memory

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/util"
)

// ── 语义记忆（BM25 检索，零外部依赖）─────────────────────────

// Memory 一条可检索的故事记忆
type Memory struct {
	ID         string  `json:"id"`
	ChapterNum int     `json:"chapter_num"`
	Text       string  `json:"text"`     // 记忆内容
	Category   string  `json:"category"` // event / character / world / plot
	Tokens     int     `json:"tokens"`   // 估算 token 数
	Score      float64 `json:"score"`    // BM25 相关度分数（检索时填充）
}

// Index BM25 索引
type Index struct {
	memories   []Memory
	docFreq    map[string]int // 词 → 包含该词的文档数
	docLengths []int          // 每个文档的长度
	avgDocLen  float64
	k1         float64 // BM25 参数
	b          float64
}

// NewIndex 创建 BM25 索引
func NewIndex() *Index {
	return &Index{
		docFreq: make(map[string]int),
		k1:      1.5,
		b:       0.75,
	}
}

// tokenize 中文友好分词：按 Unicode 类别切分
func tokenize(text string) []string {
	var tokens []string
	var current []rune

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = nil
			}
			if !unicode.IsSpace(r) && !unicode.IsPunct(r) {
				tokens = append(tokens, strings.ToLower(string(r)))
			}
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}

	// 对中文做 2-gram 增强（捕获双字词组）
	runes := []rune(strings.ToLower(text))
	for i := 0; i < len(runes)-1; i++ {
		if unicode.Is(unicode.Han, runes[i]) && unicode.Is(unicode.Han, runes[i+1]) {
			tokens = append(tokens, string(runes[i:i+2]))
		}
	}

	return tokens
}

// BuildFromProject 从项目构建 BM25 索引（T7-3「章节断档即停」修复）：
// 改用 project.ReadAllChapterSummaries 单次目录扫描读取全部章节摘要，替代
// 逐个文件探测——中间缺章（如 1、2、4）不再因缺章文件提前终止索引，断档后的
// 章节摘要也能进入记忆库。章节号由文件名 NNN 前缀解析（与摘要排序一致）。
func BuildFromProject(pm *project.Manager) (*Index, error) {
	idx := NewIndex()

	summaries, err := pm.ReadAllChapterSummaries()
	if err != nil {
		// chapters 目录不存在/不可读：按空项目处理（保持旧行为：空索引不报错）。
		return idx, nil
	}
	nums := summaryChapterNums(pm) // 文件名章节号（与 ReadAllChapterSummaries 排序一致）

	for i, s := range summaries {
		text := s.Summary
		if text == "" {
			continue
		}
		num := 0
		if i < len(nums) {
			num = nums[i]
		}
		mem := Memory{
			ID:         s.Title,
			ChapterNum: num,
			Text:       text,
			Category:   "event",
			Tokens:     util.EstimateTokens(text),
		}
		idx.Add(mem)
	}

	return idx, nil
}

// summaryChapterNums 单次目录扫描解析各摘要文件的章节号（文件名开头的 NNN
// 数字；分支摘要 NNNx 取主章节号），排序规则与 ReadAllChapterSummaries 一致
// （字典序），保证按序配对。目录不可读返回 nil（调用方降级为 0）。
func summaryChapterNums(pm *project.Manager) []int {
	entries, err := os.ReadDir(filepath.Join(pm.Dir, "chapters"))
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "-summary.json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	nums := make([]int, 0, len(names))
	for _, n := range names {
		nums = append(nums, leadingNumber(n))
	}
	return nums
}

// leadingNumber 解析字符串开头的连续数字（无数字前缀返回 0）。
func leadingNumber(s string) int {
	num := 0
	started := false
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		started = true
		num = num*10 + int(r-'0')
	}
	if !started {
		return 0
	}
	return num
}

// Add 添加一条记忆到索引
func (idx *Index) Add(m Memory) {
	tokens := tokenize(m.Text)
	seen := make(map[string]bool)

	for _, t := range tokens {
		if !seen[t] {
			idx.docFreq[t]++
			seen[t] = true
		}
	}

	idx.memories = append(idx.memories, m)
	idx.docLengths = append(idx.docLengths, len(tokens))

	total := 0
	for _, l := range idx.docLengths {
		total += l
	}
	if len(idx.docLengths) > 0 {
		idx.avgDocLen = float64(total) / float64(len(idx.docLengths))
	}
}

// Search 检索最相关的 K 条记忆
func (idx *Index) Search(query string, k int) []Memory {
	if len(idx.memories) == 0 {
		return nil
	}

	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	N := float64(len(idx.memories))

	type scored struct {
		mem   Memory
		score float64
	}

	var results []scored

	for i, mem := range idx.memories {
		docTokens := tokenize(mem.Text)
		tf := make(map[string]int)
		for _, t := range docTokens {
			tf[t]++
		}

		score := 0.0
		for _, qt := range queryTokens {
			f := float64(tf[qt])
			if f == 0 {
				continue
			}
			df := float64(idx.docFreq[qt])
			if df == 0 {
				df = 0.5 // 平滑
			}

			// BM25 公式
			idf := math.Log(1 + (N-df+0.5)/(df+0.5))
			numerator := f * (idx.k1 + 1)
			denominator := f + idx.k1*(1-idx.b+idx.b*float64(idx.docLengths[i])/idx.avgDocLen)
			score += idf * numerator / denominator
		}

		if score > 0 {
			results = append(results, scored{mem: mem, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if k > len(results) {
		k = len(results)
	}

	var topK []Memory
	for i := 0; i < k; i++ {
		m := results[i].mem
		m.Score = math.Round(results[i].score*100) / 100
		topK = append(topK, m)
	}

	return topK
}

// InjectIntoContext 将相关记忆注入到 AI 上下文字符串末尾
// 返回注入后的上下文和注入的记忆列表
func (idx *Index) InjectIntoContext(currentContext string, maxMemories int, maxTokens int) (string, []Memory) {
	memories := idx.Search(currentContext, maxMemories)
	if len(memories) == 0 {
		return currentContext, nil
	}

	var parts []string
	var injected []Memory
	tokenUsed := 0

	parts = append(parts, currentContext)
	parts = append(parts, "\n\n## 相关历史记忆（按相关度排序）")

	for _, m := range memories {
		if tokenUsed+m.Tokens > maxTokens {
			break
		}
		parts = append(parts, formatMemory(m))
		tokenUsed += m.Tokens
		injected = append(injected, m)
	}

	return strings.Join(parts, "\n"), injected
}

func formatMemory(m Memory) string {
	return strings.Join([]string{
		"---",
		"第" + strings.TrimPrefix(m.ID, "第 ") + "",
		"相关度: " + formatScore(m.Score),
		m.Text,
	}, "\n")
}

func formatScore(s float64) string {
	if s >= 5 {
		return "⭐⭐⭐ 高"
	} else if s >= 3 {
		return "⭐⭐ 中"
	}
	return "⭐ 低"
}
