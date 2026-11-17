package skill

import (
	"strings"
	"unicode"
)

// 词法打分（codex 蒸馏 T14，weighted_lexical_v1 同构）：整数权重梯度
// 256/128/64/24/16/6/4/1——名命中远大于描述命中，精确命中远大于相关词。
// 全整数、全确定，同输入永远同分同序。收尾 matched²平方加成奖励「覆盖多个
// 不同查询词」，一行代码抑制「单个高频词刷榜」。

// 词法权重表（照抄上游梯度，不自调）。
const (
	weightPhraseInName   = 256 // 查询整句出现在名里
	weightNameEqualsTerm = 128 // 名 == 查询词
	weightNameHasTerm    = 64  // 名的分词集合包含查询词
	weightNameRelated    = 24  // 名里含查询词子串（非整词）
	weightDescHasTerm    = 16  // 一句话描述包含查询词
	weightDescRelated    = 6   // 描述含查询词子串
	weightBodyHasTerm    = 4   // 正文包含查询词
	weightBodyRelated    = 1   // 正文含查询词子串
)

// 词法路硬上限（截断不报错，纪律见 ngram.go 头注释）。
const (
	lexicalMaxQueryBytes = 4 << 10
	lexicalMaxQueryTerms = 64
	lexicalMaxDocBytes   = 4 << 10
	lexicalMaxDocTerms   = 256
)

// tokenizeQuery 把查询切成词项：ASCII 按非字母数字分隔小写；CJK 连续段按
// 二字窗口切词（中文无空格，整段一词会让词法路失明，二字窗口与 ngram 路同思路）。
// 超限截断（4KiB / 64 词），truncated 上报。
func tokenizeQuery(q string) (terms []string, phrase string, truncated bool) {
	if len(q) > lexicalMaxQueryBytes {
		q = q[:lexicalMaxQueryBytes]
		truncated = true
	}
	phrase = strings.ToLower(strings.TrimSpace(q))
	seen := map[string]bool{}
	var cur []rune
	flush := func() {
		defer func() { cur = cur[:0] }()
		if len(cur) == 0 {
			return
		}
		runes := cur
		if unicode.Is(unicode.Han, runes[0]) {
			// CJK 段：二字窗口；单字段自成一词。
			if len(runes) == 1 {
				addTerm(&terms, seen, string(runes), &truncated)
				return
			}
			for i := 0; i+1 < len(runes); i++ {
				addTerm(&terms, seen, string(runes[i:i+2]), &truncated)
			}
			return
		}
		addTerm(&terms, seen, strings.ToLower(string(runes)), &truncated)
	}
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur = append(cur, unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return terms, phrase, truncated
}

// addTerm 去重追加词项；超 64 词截断。
func addTerm(terms *[]string, seen map[string]bool, t string, truncated *bool) {
	if *truncated || t == "" || seen[t] {
		return
	}
	if len(*terms) >= lexicalMaxQueryTerms {
		*truncated = true
		return
	}
	seen[t] = true
	*terms = append(*terms, t)
}

// docTokens 把文档字段切词（同查询规则；4KiB/256 词上限截断）。
func docTokens(s string) (map[string]bool, bool) {
	truncated := false
	if len(s) > lexicalMaxDocBytes {
		s = s[:lexicalMaxDocBytes]
		truncated = true
	}
	seen := map[string]bool{}
	count := 0
	var cur []rune
	flush := func() {
		if len(cur) == 0 {
			return
		}
		runes := cur
		cur = cur[:0]
		if count >= lexicalMaxDocTerms {
			truncated = true
			return
		}
		if unicode.Is(unicode.Han, runes[0]) {
			if len(runes) == 1 {
				seen[string(runes)] = true
				count++
				return
			}
			for i := 0; i+1 < len(runes); i++ {
				if count >= lexicalMaxDocTerms {
					truncated = true
					return
				}
				seen[string(runes[i:i+2])] = true
				count++
			}
			return
		}
		w := strings.ToLower(string(runes))
		seen[w] = true
		count++
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur = append(cur, unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return seen, truncated
}

// containsFold 是子串相关判定（小写归一后 Contains）。
func containsFold(hay, needle string) bool {
	return strings.Contains(strings.ToLower(hay), needle)
}

// scoreWeighted 对单个技能打词法分：返回分数与命中的不同查询词数。
func scoreWeighted(terms []string, phrase, name, desc, body string) (score uint32, matched int) {
	nameTokens, _ := docTokens(name)
	descTokens, _ := docTokens(desc)
	nameLower := strings.ToLower(name)
	matchedTerms := map[string]bool{}
	// 查询整句出现在名里：最强信号。
	if phrase != "" && strings.Contains(nameLower, phrase) {
		score += weightPhraseInName
	}
	for _, t := range terms {
		hit := false
		switch {
		case name == t || nameLower == t:
			score += weightNameEqualsTerm
			hit = true
		case nameTokens[t]:
			score += weightNameHasTerm
			hit = true
		case containsFold(name, t):
			score += weightNameRelated
			hit = true
		}
		if descTokens[t] {
			score += weightDescHasTerm
			hit = true
		} else if containsFold(desc, t) {
			score += weightDescRelated
			hit = true
		}
		if containsFold(body, t) {
			// 正文：整词与子串同权（正文长，细分意义小）。
			score += weightBodyHasTerm
			hit = true
		}
		if hit {
			matchedTerms[t] = true
		}
	}
	matched = len(matchedTerms)
	score += uint32(matched * matched) // 平方加成：覆盖多词 > 单词刷频
	return score, matched
}
