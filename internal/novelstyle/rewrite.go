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

// aiReplacements 中文网文常见 AI 高频词 → 平实替代表（确定性去 AI 味第一梯队）。
// 原则：不改变情节与语义，只把「AI 腔」词汇换成更口语化/更平实的写法。
// 这些词与 segment.go 的 aiBlacklist 同源，替换后应显著降低 ScoreText 的 AI 味分。
// 注意：等价映射难免有语境误差，作者可在结果上再手改；本表是保守的「降 AI 味」基线。
var aiReplacements = map[string]string{
	"眸光流转": "目光流转",
	"眼睑":   "眼皮",
	"眼帘":   "眼睛",
	"轻叹":   "叹息",
	"眸光":   "目光",
	"眸色":   "眼色",
	"凤眸":   "凤眼",
	"微微上扬": "微扬",
	"嘴角勾起": "嘴角弯起",
	"缓缓":   "慢慢",
	"不由":   "禁不住",
	"旋即":   "随即",
	"须臾":   "片刻",
	"定睛":   "凝神",
	"精光一闪": "眼神一闪",
	"仿佛":   "好像",
	"唇角":   "嘴角",
	"勾唇":   "弯唇",
	"略微":   "稍稍",
	"颔首":   "点头",
	"骤然":   "突然",
	"霍地":   "猛地",
	"悄然":   "悄悄",
	"淡然":   "平静",
	"眸底":   "眼底",
	"沉声道":  "低声说",
	"微微一怔": "愣了一下",
}

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
	words := make([]string, 0, len(aiReplacements))
	for w := range aiReplacements {
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool { return len([]rune(words[i])) > len([]rune(words[j])) })

	out := text
	for _, w := range words {
		after := aiReplacements[w]
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
