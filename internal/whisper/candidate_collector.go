// Package whisper — candidate_collector.go
// 100% 对齐 ackem extensions/dispatch/candidateCollector.ts
// 候选收集器：关键词精确匹配 + 语义 token/bigram 评分

package whisper

import (
	"sort"
	"strings"
	"time"
)

const (
	candidateMaxCount      = 5
	candidateSemanticMin   = 0.08
	candidateCooldownMin   = 10 // 分钟
)

// ─── 过滤 ──────────────────────────────────────────────────────

// filterEligibleCandidates 过滤符合条件的候选
func filterEligibleCandidates(candidates []DispatchCandidate, now time.Time) []DispatchCandidate {
	var eligible []DispatchCandidate
	for _, c := range candidates {
		if len(c.Keywords) == 0 {
			continue
		}
		eligible = append(eligible, c)
	}
	if len(eligible) > 20 {
		eligible = eligible[:20]
	}
	return eligible
}

// collectDispatchCandidates 关键词精确匹配收集
func collectDispatchCandidates(msg string, candidates []DispatchCandidate, now time.Time) []DispatchCandidate {
	eligible := filterEligibleCandidates(candidates, now)
	normalized := strings.ToLower(msg)

	var hits []DispatchCandidate
	for _, c := range eligible {
		for _, kw := range c.Keywords {
			if strings.Contains(normalized, strings.ToLower(kw)) {
				hits = append(hits, c)
				break
			}
		}
		if len(hits) >= candidateMaxCount {
			break
		}
	}
	return hits
}

// ─── 语义评分 ──────────────────────────────────────────────────

// tokenize 中文分词（bag-of-words）
func tokenize(text string) map[string]bool {
	tokens := make(map[string]bool)
	lower := strings.ToLower(text)

	// 按标点和空白分割
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return r == '，' || r == '。' || r == '！' || r == '？' || r == '、' ||
			r == '；' || r == '：' || r == ' ' || r == '\t' || r == '\n' ||
			r == ',' || r == '.' || r == '!' || r == '?' || r == ';' || r == ':'
	})
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if len([]rune(f)) >= 2 {
			tokens[f] = true
		}
	}

	// 同时做 bigram
	clean := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' {
			return -1
		}
		return r
	}, lower)
	runes := []rune(clean)
	for i := 0; i < len(runes)-1; i++ {
		tokens[string(runes[i:i+2])] = true
	}

	return tokens
}

// overlapRatio token 重叠率
func overlapRatio(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	hit := 0
	for t := range a {
		if b[t] {
			hit++
		}
	}
	return float64(hit) / float64(len(a))
}

// scoreCandidateEntry 对候选条目语义评分
func scoreCandidateEntry(msg string, c DispatchCandidate) float64 {
	msgTokens := tokenize(msg)

	// 构建语料：name + summary + keywords + scenarios + habits
	var corpusParts []string
	corpusParts = append(corpusParts, c.Name)
	corpusParts = append(corpusParts, c.Summary)
	corpusParts = append(corpusParts, c.Keywords...)
	corpusParts = append(corpusParts, c.Scenarios...)
	corpusParts = append(corpusParts, c.Habits...)
	corpus := strings.Join(corpusParts, " ")

	corpusTokens := tokenize(corpus)
	score := overlapRatio(msgTokens, corpusTokens)

	// 关键词精确包含提权
	for _, kw := range c.Keywords {
		if strings.Contains(msg, kw) {
			if score < 0.5 {
				score = 0.5
			}
			break
		}
	}
	return score
}

// collectSemanticDispatchCandidates 语义评分收集
func collectSemanticDispatchCandidates(msg string, candidates []DispatchCandidate, now time.Time) []DispatchCandidate {
	eligible := filterEligibleCandidates(candidates, now)

	type entry struct {
		candidate DispatchCandidate
		score     float64
	}
	var entries []entry
	for _, c := range eligible {
		s := scoreCandidateEntry(msg, c)
		if s >= candidateSemanticMin {
			entries = append(entries, entry{candidate: c, score: s})
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].score > entries[j].score })

	var hits []DispatchCandidate
	for _, e := range entries {
		e.candidate.Score = e.score
		hits = append(hits, e.candidate)
		if len(hits) >= candidateMaxCount {
			break
		}
	}
	return hits
}

// ─── 合并 ──────────────────────────────────────────────────────

// mergeDispatchCandidates 合并关键词+语义候选（去重）
func mergeDispatchCandidates(keywordHits, semanticHits []DispatchCandidate) []DispatchCandidate {
	seen := make(map[string]bool)
	var merged []DispatchCandidate
	for _, c := range keywordHits {
		if !seen[c.ID] {
			seen[c.ID] = true
			merged = append(merged, c)
		}
	}
	for _, c := range semanticHits {
		if !seen[c.ID] {
			seen[c.ID] = true
			merged = append(merged, c)
		}
	}
	if len(merged) > candidateMaxCount {
		merged = merged[:candidateMaxCount]
	}
	return merged
}
