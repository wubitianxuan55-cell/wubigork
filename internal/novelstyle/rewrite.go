package novelstyle

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// RewriteChange 一次定点改写（AI 黑名单词 → 平实替代表）。
type RewriteChange struct {
	Word   string `json:"word"`
	Before string `json:"before"`
	After  string `json:"after"`
	Count  int    `json:"count"`
}

// RewriteReport 一次去 AI 味改写的结果。
type RewriteReport struct {
	BeforeScore int             `json:"before_score"`
	AfterScore  int             `json:"after_score"`
	Changes     []RewriteChange `json:"changes"`
	PunctFixed  int             `json:"punct_fixed"`
}

// aiReplacements 词表已外置为数据资产（v4.225 规范知识出内核）：默认表在
// words.json（go:embed），可被 .gaea/skills/novel-deslop/words.json 整体替换
// （LoadWordsFile），运行时经 currentWords().Replacements 读取。
// 原则不变：不改变情节与语义，只把「AI 腔」词汇换成更口语化/更平实的写法；
// 等价映射难免有语境误差，作者可在结果上再手改。

// punctOverRE 连串省略号 / 感叹号（used to collapse overloaded runs）。
var punctOverRE = regexp.MustCompile(`…{2,}|\.{6,}|！{2,}|!{2,}`)

// DeSlopRewrite 确定性定点去 AI 味重写：
//   - 把文本中出现的 AI 黑名单词替换为平实替代表（只动这些词，不碰其它内容）；
//   - 归一过度标点（连串省略号/感叹号 → 单个）；
//   - 返回改写后文本 + 改写报告；全文替换是确定性的、无 LLM、无网络。
//
// score 可为 nil（此时内部用 ScoreTextNoRef 先算一次 before）。返回 after 文本
// 由调用方决定是否落盘——本函数不修改传入内容，纯函数。
func DeSlopRewrite(text string, score *TasteScore) (string, *RewriteReport, error) {
	before := score
	if before == nil {
		b, err := ScoreTextNoRef(text)
		if err != nil {
			return "", nil, fmt.Errorf("novelstyle: 打分失败: %w", err)
		}
		before = b
	}
	report := &RewriteReport{BeforeScore: before.Score}

	// 1. 词表替换（全局，命中即换；按词长降序避免"眼帘"先于"眸光流转"截断）。
	// v4.225 词表外置：读词表快照（默认 words.json 内置，可被覆盖文件替换）。
	repl := currentWords().Replacements
	words := make([]string, 0, len(repl))
	for w := range repl {
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool { return len([]rune(words[i])) > len([]rune(words[j])) })

	out := text
	for _, w := range words {
		after := repl[w]
		if after == "" || !strings.Contains(out, w) {
			continue
		}
		beforeText := out
		out = strings.ReplaceAll(out, w, after)
		cnt := strings.Count(beforeText, w)
		if cnt > 0 {
			report.Changes = append(report.Changes, RewriteChange{Word: w, Before: w, After: after, Count: cnt})
		}
	}

	// 2. 标点归一（连串省略号/感叹号 → 单个，控制密度）。
	beforePunct := out
	out = punctOverRE.ReplaceAllStringFunc(out, func(m string) string {
		if strings.Contains(m, "…") || strings.Contains(m, ".") {
			return "……"
		}
		return "！"
	})
	if beforePunct != out {
		report.PunctFixed = 1
	}

	// 3. 复测 after 分数。
	if after, err := ScoreTextNoRef(out); err == nil {
		report.AfterScore = after.Score
	} else {
		report.AfterScore = before.Score
	}

	return out, report, nil
}
