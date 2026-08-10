package memory

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// 注入预算（对标 Supermemory 的压缩注入：~500-800 token）：
// ProfileBlock 画像注入与逐轮 RecallBlock 都控制在预算内。
const (
	profileBudget = 600 // 画像注入上限（rune）
	recallBudget  = 800 // 逐轮记忆上下文注入上限（rune）
)

// rankedMemory 是一条带排序分值的记忆。
type rankedMemory struct {
	m     Memory
	score float64
}

// recallScore 计算一条记忆的「关键词 + 时间 + 高频」轻量排序分：
//   - 关键词重叠：查询 token 与 名称/标题/描述/正文 token 的交集越多越高；
//   - 近期：updated_at 越新越高（每 7 天衰减 0.1）；
//   - 高频：last_used_at 越近越高（最近 30 天内使用过 +0.25，7 天内 +0.4）；
//   - 规则优先：procedural 常驻 +0.5（保证方法论不因预算被挤掉）。
func recallScore(text string, m Memory, now time.Time) float64 {
	queryTokens := tokenize(strings.ToLower(text))
	docTokens := tokenize(strings.ToLower(m.Name + " " + m.Title + " " + m.Description + " " + m.Body))
	seen := map[string]bool{}
	overlap := 0
	for _, t := range queryTokens {
		if len(t) < 2 || seen[t] {
			continue
		}
		seen[t] = true
		for _, dt := range docTokens {
			if dt == t {
				overlap++
				break
			}
		}
	}
	score := float64(overlap)

	if !m.UpdatedAt.IsZero() {
		days := now.Sub(m.UpdatedAt).Hours() / 24
		if days < 0 {
			days = 0
		}
		score += 0.5 - 0.1*min(days/7, 5)
	}
	if !m.LastUsedAt.IsZero() {
		days := now.Sub(m.LastUsedAt).Hours() / 24
		switch {
		case days <= 7:
			score += 0.4
		case days <= 30:
			score += 0.25
		}
	}
	if m.Kind == KindProcedural {
		score += 0.5
	}
	return score
}

// rankMemories 按分值倒序排列记忆（稳定排序，同分保持原名序）。
func rankMemories(text string, ms []Memory, now time.Time) []Memory {
	ranked := make([]rankedMemory, 0, len(ms))
	for _, m := range ms {
		ranked = append(ranked, rankedMemory{m: m, score: recallScore(text, m, now)})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].m.Name < ranked[j].m.Name
	})
	out := make([]Memory, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.m)
	}
	return out
}

// RecallBlock 返回逐轮注入的「记忆上下文」：procedural 常驻 + 触发命中的
// episodic + 相关 user/semantic 事实，按「关键词 + 时间 + 高频」排序并压缩到
// budget（默认 800 rune，对标轻量压缩注入）。无记忆或全部不相关时返回 ""。
func (s *Set) RecallBlock(text string, budget int) string {
	if s == nil {
		return ""
	}
	ms := s.Store.List()
	if len(ms) == 0 {
		return ""
	}
	if budget <= 0 {
		budget = recallBudget
	}
	now := time.Now()
	ranked := rankMemories(text, ms, now)

	var b strings.Builder
	b.WriteString("## 记忆上下文（按相关度/近期自动注入）\n")
	written := 0
	for _, m := range ranked {
		// 非 procedural 且无关键词命中的事实不逐轮注入（靠系统前缀索引兜底）
		if m.Kind != KindProcedural && !episodicTriggered(text, m) && !keywordHit(text, m) {
			continue
		}
		line := formatRecallLine(m)
		lineRunes := utf8.RuneCountInString(line)
		if written+lineRunes > budget {
			continue
		}
		b.WriteString(line)
		written += lineRunes
	}
	block := strings.TrimSpace(b.String())
	if block == "## 记忆上下文（按相关度/近期自动注入）" {
		return ""
	}
	return block
}

// episodicTriggered 判断 episodic 记忆的触发标签是否命中输入（复用
// EpisodicMatches 的标签判定）。
func episodicTriggered(text string, m Memory) bool {
	if m.Kind != KindEpisodic || len(m.Tags) == 0 {
		return false
	}
	lower := strings.ToLower(text)
	for _, tag := range m.Tags {
		if strings.Contains(lower, strings.ToLower(tag)) {
			return true
		}
	}
	return false
}

// keywordHit 判断文本 token 是否与记忆描述/标题重叠（非 procedural 的
// 相关事实也注入，保证「问什么带什么」）。
func keywordHit(text string, m Memory) bool {
	queryTokens := tokenize(strings.ToLower(text))
	docTokens := tokenize(strings.ToLower(m.Title + " " + m.Description + " " + m.Body))
	for _, qt := range queryTokens {
		if len(qt) < 2 {
			continue
		}
		for _, dt := range docTokens {
			if dt == qt {
				return true
			}
		}
	}
	return false
}

// formatRecallLine 渲染一条注入行：procedural 附正文（供模型直接遵循），
// 其他附一句话摘要。
func formatRecallLine(m Memory) string {
	var b strings.Builder
	fmtTags := ""
	if m.Kind == KindEpisodic {
		fmtTags = "episodic"
	} else if m.Kind == KindProcedural {
		fmtTags = "procedural"
	}
	if fmtTags == "" {
		fmtTags = string(m.Type)
	}
	label := displayTitle(m.Title, m.Name)
	if m.Kind == KindProcedural && strings.TrimSpace(m.Body) != "" {
		body := strings.TrimSpace(m.Body)
		if r := []rune(body); len(r) > 200 {
			body = string(r[:200]) + "…"
		}
		b.WriteString("- [" + fmtTags + "] " + label + "：" + oneLine(body) + "\n")
		return b.String()
	}
	desc := oneLine(m.Description)
	if desc == "" {
		desc = label
	}
	b.WriteString("- [" + fmtTags + "] " + label + "：" + desc + "\n")
	return b.String()
}
