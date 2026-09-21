package memory

// ── activation 二维：固化正文随会话快照（v4.378，Reasonix memory/activation.go
// 留池蒸馏）────────────────────────────────────────────────────────────────
//
// 上游把事实如何触达模型拆成正交两维：scope 管「在哪可用」，activation 管
// 「怎么触达」——pinned=正文随会话上下文快照装配，relevant=仅索引+检索可达。
// gaea 侧 Pinned 固化态此前只有排序加权与生命周期豁免（5.3），「正文随快照」
// 这一维缺席；本刀补齐：固化事实正文预算化注入缓存稳定前缀，用户在记忆面板
// 的「固化」动作从此有功能后果（standing context），不再只是排序信号。
//
// 与晨报预载（BuildMorningPreloadBlock）的分工：晨报=高频近用的「名称：摘要」
// 行（recency 序，触碰即翻新）；固化块=pinned 全文的整行注入。排序刻意取
// Name 升序而非 recency——本块 ride 缓存稳定前缀，touch 不能翻动它（否则每次
// 检索触碰都会打碎前缀缓存）；只有固化集合本身变化才换前缀，新会话生效。

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// DefaultPinnedFactsBudget 是固化正文块默认总预算（rune）。
const DefaultPinnedFactsBudget = 1200

// pinnedFactsBodyCap 是单条固化正文的 rune 上限：超长正文截断留标（对齐
// 晨报「单行描述预算」精神——块预算应服务多条固化事实而非被单条吃穿）。
const pinnedFactsBodyCap = 400

// pinnedFactsHeader 是固化块头（无注入行时用于早退判断）。
const pinnedFactsHeader = "【固化记忆】"

// pinnedFactsTruncMark 是单条正文被截断时的尾标。
const pinnedFactsTruncMark = "……"

// BuildPinnedFactsBlock 生成固化事实正文注入块。mems 为当前空间的活跃记忆
// （调用方已按空间取数，本函数不感知空间）；仅取 Pinned 条目，按 Name 升序
// （前缀稳定：touch/触达不翻动块内容），渲染为：
//
//	【固化记忆】
//	- 名称：正文（≤400 rune，UTF-8 边界截断加省略标）
//	...
//
// 块总长度 ≤ maxRunes（rune 计数；≤0 时用 DefaultPinnedFactsBudget）：整行
// 放不下即停止（不切行，预算诚实）；预算不足容纳块头或无条目可渲染返回
// 空串（零注入，前缀逐字节不变）。
func BuildPinnedFactsBlock(mems []Memory, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = DefaultPinnedFactsBudget
	}

	// 只留有正文可装配的固化条，Name 升序（Name 空取 Title——与索引展示
	// 同一兜底）。
	pinned := make([]Memory, 0, len(mems))
	for _, m := range mems {
		if !m.Pinned {
			continue
		}
		if strings.TrimSpace(m.Body) == "" {
			continue // 正文空：无「随快照」内容，索引行已覆盖
		}
		pinned = append(pinned, m)
	}
	sort.Slice(pinned, func(i, j int) bool {
		return pinnedFactName(pinned[i]) < pinnedFactName(pinned[j])
	})

	written := utf8.RuneCountInString(pinnedFactsHeader + "\n")
	if written > maxRunes {
		return "" // 预算连块头都放不下：诚实返回空串
	}
	var b strings.Builder
	b.WriteString(pinnedFactsHeader + "\n")
	emitted := false
	for _, m := range pinned {
		line := formatPinnedFactLine(m)
		if line == "" {
			continue
		}
		n := utf8.RuneCountInString(line)
		if written+n > maxRunes {
			break // 预算不够整行：停止（不切行）
		}
		b.WriteString(line)
		written += n
		emitted = true
	}
	if !emitted {
		return "" // 只有块头没有注入行：与空记忆同语义，不注入
	}
	return b.String()
}

// formatPinnedFactLine 渲染一条固化行：「- 名称：正文」。正文单行化后按
// pinnedFactsBodyCap 截断（UTF-8 边界，加省略标）；放不下返回 ""（跳过）。
func formatPinnedFactLine(m Memory) string {
	name := pinnedFactName(m)
	if name == "" {
		return ""
	}
	body := strings.Join(strings.Fields(m.Body), " ") // 单行化：压掉换行与连续空白
	body = truncatePinnedBody(body)
	if body == "" {
		return ""
	}
	return "- " + name + "：" + body + "\n"
}

// pinnedFactName 取条目展示名（Name 优先，Title 兜底）。
func pinnedFactName(m Memory) string {
	if n := strings.TrimSpace(m.Name); n != "" {
		return n
	}
	return strings.TrimSpace(m.Title)
}

// truncatePinnedBody 按预算截断正文（rune 计数，UTF-8 边界安全），截断时加
// 省略标。
func truncatePinnedBody(body string) string {
	rs := []rune(body)
	if len(rs) <= pinnedFactsBodyCap {
		return body
	}
	return strings.TrimSpace(string(rs[:pinnedFactsBodyCap-utf8.RuneCountInString(pinnedFactsTruncMark)])) + pinnedFactsTruncMark
}
