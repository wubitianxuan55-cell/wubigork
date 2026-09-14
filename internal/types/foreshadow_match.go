package types

import "strings"

// ── 伏笔内容匹配（分析同步的回收兜底）────────────────────────
//
// 口径来源：docs/distill/01-foreshadow-spec.md §3.4（六策略加权评分）/
// §3.5（字符级 n-gram 相似度），对齐 MuMu foreshadow_service.py:1552-1691。
// 纯函数层，无 IO 无状态；与 foreshadow_urgency.go 同属共享契约层，
// 分析同步（internal/analysis）是当前唯一消费方，后续域不得另写第二份。

// resolveTitleSuffixes 回收动作标题的常见后缀（§3.4 第 0 步，MuMu :1584-1589）。
// 「铜匣之谜·回收」→ 剥成「铜匣之谜」再参与标题族比对。
var resolveTitleSuffixes = []string{"回收", "揭示", "解答", "兑现"}

// WordOverlap 字符级 n-gram 相似度（§3.5，MuMu :1657-1691）。
//
// 归一化（小写、去空白）后按 rune 切 2-gram 与 3-gram 做 Jaccard，
// 加权 o2*0.4 + o3*0.6（3-gram 更精确权重更高）。★ 必须按 rune 切分：
// 按 byte 切会让中文 n-gram 全部错位。
func WordOverlap(a, b string) float64 {
	norm := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\n", "")
		return s
	}
	ngrams := func(s string, n int) map[string]struct{} {
		r := []rune(norm(s))
		if len(r) == 0 {
			return nil
		}
		if len(r) < n {
			return map[string]struct{}{string(r): {}}
		}
		out := make(map[string]struct{}, len(r)-n+1)
		for i := 0; i+n <= len(r); i++ {
			out[string(r[i:i+n])] = struct{}{}
		}
		return out
	}
	jaccard := func(a, b map[string]struct{}) float64 {
		if len(a) == 0 || len(b) == 0 {
			return 0
		}
		inter := 0
		for k := range a {
			if _, ok := b[k]; ok {
				inter++
			}
		}
		union := len(a) + len(b) - inter
		if union == 0 {
			return 0
		}
		return float64(inter) / float64(union)
	}
	o2 := jaccard(ngrams(a, 2), ngrams(b, 2))
	o3 := jaccard(ngrams(a, 3), ngrams(b, 3))
	return o2*0.4 + o3*0.6
}

// StripResolveTitleSuffix 剥离回收标题后缀（只剥一次，顺序检查 §3.4 第 0 步）。
func StripResolveTitleSuffix(title string) string {
	for _, suffix := range resolveTitleSuffixes {
		if strings.HasSuffix(title, suffix) {
			return strings.TrimSuffix(title, suffix)
		}
	}
	return title
}

// MatchForeshadowByContent 内容匹配兜底（§3.4 六策略加权）。
//
// 在 planted 候选（应为存活可回收条目，按埋入章升序）中为一条 resolved
// 分析结果找归属。六策略：标题族（取最大）+ 关键词命中（取最大）+ 内容
// n-gram（取最大）+ 引用章号/分类/关联角色（累加）。采纳 = 严格大于当前
// 最优且 ≥ minSimilarity（默认 0.5）——同分取先出现者，即最早埋入的。
// 全部低于阈值返回 (nil, bestScore)。
func MatchForeshadowByContent(change ForeshadowHit, planted []Foreshadow, minSimilarity float64) (*Foreshadow, float64) {
	var best *Foreshadow
	bestScore := 0.0

	cleanTitle := StripResolveTitleSuffix(strings.TrimSpace(change.Title))
	rt := strings.TrimSpace(change.Title)

	for i := range planted {
		fs := &planted[i]
		score := 0.0
		fsTitle := strings.TrimSpace(fs.Title)

		// 策略1 标题族（取最大，不累加；MuMu :1604-1620）
		if rt != "" && fsTitle != "" {
			switch {
			case rt == fsTitle:
				score = 1.0
			case cleanTitle != "" && cleanTitle == fsTitle:
				score = 0.95
			case strings.Contains(rt, fsTitle) || strings.Contains(fsTitle, rt):
				score = maxFloat(score, 0.8)
			case cleanTitle != "" && (strings.Contains(cleanTitle, fsTitle) || strings.Contains(fsTitle, cleanTitle)):
				score = maxFloat(score, 0.75)
			default:
				score = maxFloat(score, WordOverlap(rt, fsTitle)*0.7)
			}
		}
		// 策略2 关键词命中（取最大；MuMu :1623-1625）
		if kw := change.Keyword.Text; kw != "" && fs.Description != "" && strings.Contains(fs.Description, kw) {
			score = maxFloat(score, 0.75)
		}
		// 策略3 内容 n-gram 相似（取最大；MuMu :1628-1630）
		if change.Content != "" && fs.Description != "" {
			score = maxFloat(score, WordOverlap(change.Content, fs.Description)*0.6)
		}
		// 策略4 引用章号一致（累加；MuMu :1633-1635）
		if change.ReferenceChapter > 0 && fs.PlantedIn != "" &&
			ChapterNumOf(fs.PlantedIn) == change.ReferenceChapter {
			score += 0.15
		}
		// 策略5 分类一致（累加；MuMu :1638-1640）
		if change.Category != "" && fs.Category == change.Category {
			score += 0.1
		}
		// 策略6 关联角色 Jaccard（累加；MuMu :1643-1645）
		if len(change.RelatedChars) > 0 && len(fs.RelatedCharacters) > 0 {
			inter, union := setInterUnion(change.RelatedChars, fs.RelatedCharacters)
			if union > 0 {
				score += float64(inter) / float64(union) * 0.1
			}
		}

		if score > bestScore && score >= minSimilarity {
			bestScore = score
			best = fs
		}
	}
	return best, bestScore
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// setInterUnion 两个字符串切片的交集/并集大小（各自先去重，防切片内
// 重复元素把交集算重；大小写敏感）。
func setInterUnion(a, b []string) (inter, union int) {
	setA := make(map[string]struct{}, len(a))
	for _, s := range a {
		setA[s] = struct{}{}
	}
	seen := make(map[string]struct{}, len(b))
	for _, s := range b {
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		if _, ok := setA[s]; ok {
			inter++
		}
	}
	return inter, len(setA) + len(seen) - inter
}
