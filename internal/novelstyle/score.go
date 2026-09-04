package novelstyle

import (
	"math"
	"strings"
	"unicode"
)

// ── AI 味检测（确定性，不调 LLM） ─────────────────────────────────────────

// TasteIssue 一个被标记的 AI 味问题（span 定位到字偏移，rune 下标）。
type TasteIssue struct {
	Start      int    `json:"start"`      // 在原文的 [Start,End) rune 偏移
	End        int    `json:"end"`        // 在原文的 [Start,End) rune 偏移
	Reason     string `json:"reason"`     // 命中规则
	Severity   string `json:"severity"`   // low/medium/high/blocker
	Suggestion string `json:"suggestion"` // 修复建议
}

// TasteScore AI 味评分。
type TasteScore struct {
	Score  int          `json:"score"`  // 0-100，越高越像 AI
	Issues []TasteIssue `json:"issues"` // 所有命中 span
}

// ── 阈值常量（源自 anti-ai-polish，须在自采语料重标） ──
const (
	maxMetaphorPerPara    = 1    // 每段最多 1 个比喻标记
	maxConnectiveRun      = 1    // 句首连接词连续最多 1 句
	maxFourCharPerClause  = 1    // 一个分句最多 1 个四字格
	maxEllipsisPerPara    = 1    // 每段省略号 <=1
	maxEllipsisPerChapter = 5    // 每章省略号 <=5
	maxExclamPerPara      = 1    // 每段感叹号 <=1
	maxExclamPerChapter   = 8    // 每章感叹号 <=8
	sentenceSdTooUniform  = 6.0  // 句长 std 低于此视为太均匀
	adjAdvDensityMax      = 30.0 // 形容词/副词密度上限（次/1000字）
	fourCharPer500Max     = 3    // 四字成语 <=3/500字
)

// ── 严重度权重（合成 0-100 分） ──
const (
	weightLow     = 4
	weightMedium  = 9
	weightHigh    = 16
	weightBlocker = 30
)

func severityToWeight(sev string) int {
	switch sev {
	case "blocker":
		return weightBlocker
	case "high":
		return weightHigh
	case "medium":
		return weightMedium
	default:
		return weightLow
	}
}

// ScoreText 对一段文本打 AI 味分（0-100，越高越像 AI）+ 定位所有 span。
// fp 提供参考指纹（用于句长方差等相对判断）；fp 为 nil 时退化为 ScoreTextNoRef。
func ScoreText(text string, fp *Fingerprint) (*TasteScore, error) {
	if strings.TrimSpace(text) == "" {
		return &TasteScore{Score: 0}, nil
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return &TasteScore{Score: 0, Issues: nil}, nil
	}
	nonSpace := countNonSpaceRunes(runes)
	if nonSpace == 0 {
		return &TasteScore{Score: 0, Issues: nil}, nil
	}

	var issues []TasteIssue

	// 规则 1：无缘无故的修辞（一段 >1 个比喻）
	issues = append(issues, ruleUnmotivatedMetaphor(text, runes)...)

	// 规则 2：语域不一致
	issues = append(issues, ruleRegisterBreak(text, runes)...)

	// 规则 3：show-don't-tell 情绪直述
	issues = append(issues, ruleEmotionDirect(text, runes)...)

	// 规则 4：连接词密集
	issues = append(issues, ruleConnectiveDense(text, runes)...)

	// 规则 5：四字格连用
	issues = append(issues, ruleFourCharConsecutive(text, runes)...)

	// 规则 6：句长方差过小
	issues = append(issues, ruleSentenceUniform(text, runes, fp)...)

	// 规则 7：形容词/副词密度
	issues = append(issues, ruleAdjAdvDensity(text, runes)...)

	// 规则 8：标点滥用
	issues = append(issues, rulePunctuation(text, runes)...)

	// 规则 9：AI 高频词黑名单
	issues = append(issues, ruleAIBlacklist(text, runes)...)

	score := 0
	for _, iss := range issues {
		score += severityToWeight(iss.Severity)
	}
	if score > 100 {
		score = 100
	}
	return &TasteScore{Score: score, Issues: issues}, nil
}

// ScoreTextNoRef 无参考指纹时用通用阈值打分。
func ScoreTextNoRef(text string) (*TasteScore, error) {
	return ScoreText(text, nil)
}

// ── 各规则实现 ──────────────────────────────────────────────

// ruleUnmotivatedMetaphor 规则 1：一段超过 1 个比喻标记。
func ruleUnmotivatedMetaphor(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	for _, pr := range paragraphRanges(runes) {
		paraText := string(runes[pr[0]:pr[1]])
		n := countMetaphorMarks(paraText)
		if n > maxMetaphorPerPara {
			issues = append(issues, TasteIssue{
				Start: pr[0], End: pr[1],
				Reason:     "无缘无故的修辞：一段超过 1 个比喻（删「为好看」的排比叠喻）",
				Severity:   "medium",
				Suggestion: "同一段只保留一个最贴切的比喻，删去为华丽而堆砌的排比叠喻。",
			})
		}
	}
	return issues
}

// ruleRegisterBreak 规则 2：通俗文中出现违禁语域词且该段语气平淡。
func ruleRegisterBreak(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	for _, pr := range paragraphRanges(runes) {
		paraText := string(runes[pr[0]:pr[1]])
		// 语气平淡 = 无比喻标记且无四字格/文学化表达
		plainTone := countMetaphorMarks(paraText) == 0 && countFourChar(paraText) == 0
		if !plainTone {
			continue
		}
		for _, w := range registerBreakWords {
			for _, rng := range findRuneRanges(paraText, w) {
				issues = append(issues, TasteIssue{
					Start: pr[0] + rng[0], End: pr[0] + rng[1],
					Reason:     "语域不一致：通俗文里出现古诗典/学术/西方哲学/网络梗词",
					Severity:   "medium",
					Suggestion: "改用与正文一致的通俗口吻。",
				})
			}
		}
	}
	return issues
}

// ruleEmotionDirect 规则 3：抽象情绪直述。
func ruleEmotionDirect(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	for _, w := range emotionDirectWords {
		for _, rng := range findRuneRanges(text, w) {
			issues = append(issues, TasteIssue{
				Start: rng[0], End: rng[1],
				Reason:     "show-don't-tell：抽象情绪直述",
				Severity:   "medium",
				Suggestion: "换成身体反应来外化情绪（握拳、指节发白、眼泪唰地掉），避免「内心充满了/感到很」式直接陈述。",
			})
		}
	}
	return issues
}

// ruleConnectiveDense 规则 4：句首连接词连续 >=2 句。
func ruleConnectiveDense(text string, runes []rune) []TasteIssue {
	sents := sentenceRanges(runes)
	var issues []TasteIssue
	run := 0
	for i, se := range sents {
		sentText := strings.TrimSpace(string(runes[se[0]:se[1]]))
		if isConnectiveStart(sentText) {
			run++
			if run >= maxConnectiveRun+1 {
				issues = append(issues, TasteIssue{
					Start: sents[i-1][0], End: se[1],
					Reason:     "连接词密集：句首连接词连续 >=2 句",
					Severity:   "medium",
					Suggestion: "拆开或删去部分句首连接词，让句间衔接更自然、隐含。",
				})
			}
		} else {
			run = 0
		}
	}
	return issues
}

// ruleFourCharConsecutive 规则 5：一个分句内四字格连续 >=2 个。
func ruleFourCharConsecutive(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	for _, pr := range paragraphRanges(runes) {
		for _, cr := range clauseRanges(runes, pr[0], pr[1]) {
			clsText := string(runes[cr[0]:cr[1]])
			cnt := countFourChar(clsText)
			if cnt > maxFourCharPerClause {
				issues = append(issues, TasteIssue{
					Start: cr[0], End: cr[1],
					Reason:     "四字成语/四字格连用：一个分句内 >=2 个四字格",
					Severity:   "medium",
					Suggestion: "削减四字格用量（anti-ai-polish：四字成语 <=3/500字），改为口语短句。",
				})
			}
		}
	}
	return issues
}

// ruleSentenceUniform 规则 6：句长方差过小（太均匀）。
func ruleSentenceUniform(text string, runes []rune, fp *Fingerprint) []TasteIssue {
	se := sentenceRanges(runes)
	if len(se) < 2 {
		return nil
	}
	vals := make([]float64, 0, len(se))
	for _, r := range se {
		vals = append(vals, float64(r[1]-r[0]))
	}
	std := stddev(vals)
	threshold := sentenceSdTooUniform
	if fp != nil && fp.SentenceLen.Sd > 0 {
		threshold = math.Min(sentenceSdTooUniform, fp.SentenceLen.Sd*0.6)
	}
	if std > 0 && std < threshold {
		return []TasteIssue{{
			Start: 0, End: len(runes),
			Reason:     "句长方差过小：全篇句长过于均匀，缺少节奏变化",
			Severity:   "medium",
			Suggestion: "长短句交错，加入少量短句与长句制造节奏。",
		}}
	}
	return nil
}

// ruleAdjAdvDensity 规则 7：形容词/副词密度超阈值。
func ruleAdjAdvDensity(text string, runes []rune) []TasteIssue {
	nonSpace := float64(countNonSpaceRunes(runes))
	if nonSpace <= 0 {
		return nil
	}
	density := float64(countWords(text, adjAdvWords)) * 1000 / nonSpace
	if density > adjAdvDensityMax {
		return []TasteIssue{{
			Start: 0, End: len(runes),
			Reason:     "形容词/副词密度过高：修饰语堆砌",
			Severity:   "medium",
			Suggestion: "删减冗余的形容词/副词，让名词与动词自己说话。",
		}}
	}
	return nil
}

// rulePunctuation 规则 8：省略号/感叹号超阈值。
func rulePunctuation(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	// 整章阈值
	if countEllipsisIn(text) > maxEllipsisPerChapter {
		issues = append(issues, TasteIssue{
			Start: 0, End: len(runes),
			Reason:     "标点滥用：省略号超阈值（<=5/章）",
			Severity:   "low",
			Suggestion: "减少省略号，用句子收束替代无谓的留白。",
		})
	}
	if countExclamIn(runes) > maxExclamPerChapter {
		issues = append(issues, TasteIssue{
			Start: 0, End: len(runes),
			Reason:     "标点滥用：感叹号超阈值（<=8/章）",
			Severity:   "low",
			Suggestion: "减少感叹号，用叙述本身传达情绪。",
		})
	}
	// 段落阈值
	for _, pr := range paragraphRanges(runes) {
		paraText := string(runes[pr[0]:pr[1]])
		if countEllipsisIn(paraText) > maxEllipsisPerPara {
			issues = append(issues, TasteIssue{
				Start: pr[0], End: pr[1],
				Reason:     "标点滥用：省略号超阈值（<=1/段）",
				Severity:   "low",
				Suggestion: "删除冗余省略号。",
			})
		}
		if countExclamIn(runes[pr[0]:pr[1]]) > maxExclamPerPara {
			issues = append(issues, TasteIssue{
				Start: pr[0], End: pr[1],
				Reason:     "标点滥用：感叹号超阈值（<=1/段）",
				Severity:   "low",
				Suggestion: "删除冗余感叹号。",
			})
		}
	}
	return issues
}

// ruleAIBlacklist 规则 9：AI 高频词黑名单。
func ruleAIBlacklist(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	for _, w := range aiBlacklist {
		for _, rng := range findRuneRanges(text, w) {
			issues = append(issues, TasteIssue{
				Start: rng[0], End: rng[1],
				Reason:     "AI 高频词黑名单命中",
				Severity:   "high",
				Suggestion: "替换为更自然、具体的表达，避免公式化词汇。",
			})
		}
	}
	return issues
}

// ── 文本结构助手 ────────────────────────────────────────────

// findRuneRanges 返回 text 中所有 sub 出现的 rune 下标区间（允许重叠）。
func findRuneRanges(text, sub string) [][2]int {
	var out [][2]int
	rs := []rune(text)
	sr := []rune(sub)
	n := len(rs)
	m := len(sr)
	if m == 0 || m > n {
		return out
	}
	for i := 0; i+m <= n; i++ {
		match := true
		for j := 0; j < m; j++ {
			if rs[i+j] != sr[j] {
				match = false
				break
			}
		}
		if match {
			out = append(out, [2]int{i, i + m})
		}
	}
	return out
}

// countMetaphorMarks 统计段落内比喻标记次数（标记词 + 「像…一样」模式）。
func countMetaphorMarks(para string) int {
	c := 0
	for _, w := range metaphorMarkers {
		c += len(findRuneRanges(para, w))
	}
	rs := []rune(para)
	for i, r := range rs {
		if r != '像' {
			continue
		}
		limit := i + 8
		if limit > len(rs) {
			limit = len(rs)
		}
		// 「像…一样」：查找「一样」二字出现在 '像' 之后的窗口内
		for j := i + 1; j+1 < limit; j++ {
			if rs[j] == '一' && rs[j+1] == '样' {
				c++
				break
			}
		}
	}
	return c
}

// countFourChar 统计文本内四字格（成语）出现次数（非重叠）。
func countFourChar(text string) int {
	return countWords(text, fourCharSet)
}

// countEllipsisIn 统计省略号出现次数。
func countEllipsisIn(text string) int {
	return countPunct(text, []string{"……", "..."})
}

// countExclamIn 统计感叹号个数。
func countExclamIn(runes []rune) int {
	return countRune(runes, '！') + countRune(runes, '!')
}

// isConnectiveStart 判断句子开头是否为连接词。
func isConnectiveStart(sentence string) bool {
	s := strings.TrimLeftFunc(sentence, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	for _, w := range connectives {
		if strings.HasPrefix(s, w) {
			return true
		}
	}
	return false
}

// clauseRanges 将 [start,end) 内的 rune 按分句标点拆成子区间（绝对 rune 下标）。
func clauseRanges(runes []rune, start, end int) [][2]int {
	var out [][2]int
	segStart := start
	for i := start; i < end; i++ {
		if isClauseBoundary(runes[i]) {
			if i > segStart {
				out = append(out, [2]int{segStart, i})
			}
			segStart = i + 1
		}
	}
	if end > segStart {
		out = append(out, [2]int{segStart, end})
	}
	return out
}

// isClauseBoundary 判断 rune 是否为分句边界标点/空白。
func isClauseBoundary(r rune) bool {
	switch r {
	case '，', '；', '、', '。', '！', '？', '…', ',', ';', '.', '!', '?':
		return true
	}
	return unicode.IsSpace(r)
}
