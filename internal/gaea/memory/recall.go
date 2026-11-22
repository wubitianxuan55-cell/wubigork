package memory

import (
	"sort"
	"strconv"
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
// 每行带稳定引用键 [MEM:<name>]（citations.go）：模型采纳时在句末标注同名
// 引用，回合结束解析回传并 Touch——引用可追溯闭环的注入侧。
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
	b.WriteString(recallHeader + "\n")
	written := 0
	for _, m := range ranked {
		// 失效条目不注入（阶段九 9.3）：invalid_at 非空=已被新事实推翻——注入
		// 面永远排除（失效的语义就是不注入），原文仍可面板搜索/恢复（裁决入口）。
		if !m.InvalidAt.IsZero() {
			continue
		}
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
	if block == recallHeader {
		return ""
	}
	return block
}

// RecallBlockFused 是**聊天席（play 空间）专属**的四臂轻量融合注入（Nebula
// Memory 蒸馏接入，用户拍板 2026-11-22：聊天接、办公/文学保缓存命中率不接）。
//
// 候选集与 RecallBlock 同口径（procedural 常驻 + episodic 触发 + 关键词命中，
// 失效条目排除）+ 图谱臂发现的互引邻居（想起 A 带出 B——chat 席独有的发现
// 面）；排序换 FuseRecallArms：词法 recallScore 归一 + 图谱 1 跳扩散 + 置顶
// +0.10 + 稳定性衰减（引用多衰减慢）× 休眠惩罚。预算与渲染面与 RecallBlock
// 完全一致。语义向量臂不进热路径（每轮嵌查询=百毫秒级延迟，聊天席不可接受）。
// 数据源 Store.CiteCounts/GraphNeighbors 各一条聚合 SQL（毫秒级）。
func (s *Set) RecallBlockFused(text string, budget int) string {
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
	byName := make(map[string]Memory, len(ms))
	for _, m := range ms {
		byName[m.Name] = m
	}

	// 词法臂：recallScore 归一（同一打分公式，办公席不变）
	scored := make([]rankedMemory, 0, len(ms))
	base := make(map[string]bool, len(ms))
	maxLex := 0.0
	for _, m := range ms {
		if !m.InvalidAt.IsZero() {
			continue
		}
		if m.Kind != KindProcedural && !episodicTriggered(text, m) && !keywordHit(text, m) {
			continue
		}
		sc := recallScore(text, m, now)
		if sc > maxLex {
			maxLex = sc
		}
		scored = append(scored, rankedMemory{m: m, score: sc})
		base[m.Name] = true
	}
	lex := make(map[string]float64, len(scored))
	if maxLex > 0 {
		for _, r := range scored {
			lex[r.m.Name] = r.score / maxLex
		}
	}
	// 图谱臂：词法 top 种子沿互引邻接扩散（邻居即使零词面命中也可入选——
	// 发现面是聊天席独有的增量）。
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].m.Name < scored[j].m.Name
	})
	seeds := make([]RecallArmHit, 0, RecallSeedCount)
	for _, r := range scored {
		if len(seeds) >= RecallSeedCount {
			break
		}
		seeds = append(seeds, RecallArmHit{Name: r.m.Name, Score: lex[r.m.Name]})
	}
	graphHits := ExpandGraphSeeds(seeds, s.Store.GraphNeighbors())

	// 融合输入：基础候选 ∪ 图谱发现（发现项 zero 词法臂）。
	citeCounts := s.Store.CiteCounts()
	inputs := make([]RecallFuseInput, 0, len(scored)+len(graphHits))
	for _, r := range scored {
		inputs = append(inputs, RecallFuseInput{
			Name:    r.m.Name,
			Lexical: lex[r.m.Name],
			Graph:   graphHits[r.m.Name],
			Pinned:  r.m.Pinned,
			Dormant: LifecycleOf(r.m, now, 0) == LifecycleDecaying,
			Decay:   RecallDecayFactor(r.m, citeCounts[r.m.Name], now),
		})
	}
	for name, g := range graphHits {
		if base[name] {
			continue
		}
		m, ok := byName[name]
		if !ok || !m.InvalidAt.IsZero() {
			continue
		}
		inputs = append(inputs, RecallFuseInput{
			Name:    name,
			Graph:   g,
			Pinned:  m.Pinned,
			Dormant: LifecycleOf(m, now, 0) == LifecycleDecaying,
			Decay:   RecallDecayFactor(m, citeCounts[name], now),
		})
	}
	fused := FuseRecallArms(inputs)

	var b strings.Builder
	b.WriteString(recallHeader + "\n")
	written := 0
	for _, h := range fused {
		m, ok := byName[h.Name]
		if !ok {
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
	if block == recallHeader {
		return ""
	}
	return block
}

// recallHeader 是注入块头（无注入行时用于早退判断）。
const recallHeader = "## 记忆上下文（按相关度/近期自动注入；采纳某条时在句末标注其 [MEM:引用键]，记忆可能过时，以工作区文件为准）"

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
// 其他附一句话摘要；行尾带引用键 [MEM:<name>]（模型照抄标注，前端渲染成
// 可点击溯源徽标），陈旧记忆（90 天未修订）附时效提示。
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
	var line string
	if m.Kind == KindProcedural && strings.TrimSpace(m.Body) != "" {
		body := strings.TrimSpace(m.Body)
		if r := []rune(body); len(r) > 200 {
			body = string(r[:200]) + "…"
		}
		line = "- [" + fmtTags + "] " + label + "：" + oneLine(body)
	} else {
		desc := oneLine(m.Description)
		if desc == "" {
			desc = label
		}
		line = "- [" + fmtTags + "] " + label + "：" + desc
	}
	if staleDays(m, time.Now()) > 0 {
		line += "（已 " + strconv.Itoa(staleDays(m, time.Now())) + " 天未修订，注意时效）"
	}
	line += " [MEM:" + m.Name + "]"
	b.WriteString(line + "\n")
	return b.String()
}

// staleCitedDays 是「陈旧记忆」判定阈值：距最近修订超过该天数视为可能过时。
const staleCitedDays = 90

// staleDays 返回记忆距最近修订的天数；不足阈值（含零值未知时间）返回 0 表示
// 不提示时效。
func staleDays(m Memory, now time.Time) int {
	if m.UpdatedAt.IsZero() {
		return 0
	}
	days := int(now.Sub(m.UpdatedAt).Hours() / 24)
	if days < staleCitedDays {
		return 0
	}
	return days
}
