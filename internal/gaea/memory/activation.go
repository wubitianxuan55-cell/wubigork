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
// 与记忆索引（renderIndex）的分工：索引=全部活跃记忆的「名称 + 摘要」一行
// 行（按名称序）；固化块=pinned 全文的整行注入。排序刻意取
// Name 升序而非 recency——本块 ride 缓存稳定前缀，touch 不能翻动它（否则每次
// 检索触碰都会打碎前缀缓存）；只有固化集合本身变化才换前缀，新会话生效。
//
// 2026-11-28 众神殿记忆注入论证收口（pan_1795821263932）：固化块此前装不下
// 就静默 break，「固化=必然在场」承诺失守且无提示；本刀让块报告实际发射面
// （Emitted/Truncated/Omitted）并追加块级溢出提示，召回侧只对「已发射且正文
// 未截断」的条目去重——不能按 m.Pinned 全量判断，被预算挤出的 pinned 若同时
// 被召回跳过就两处皆失。

import (
	"sort"
	"strconv"
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

// pinnedOverflowMaxNames 是溢出提示行最多点名的条目数（提示自身必须计入
// 预算，点名数封顶防提示被长名单撑爆预算）。
const pinnedOverflowMaxNames = 3

// PinnedFactsResult 是固化块的注入产物：块文本与实际发射面。Emitted=正文已
// 写入块内的条目名（Name 稳定序）；Truncated ⊆ Emitted=正文被
// pinnedFactsBodyCap 截断的；Omitted=因块预算未入块的（溢出提示点名）。
// 召回去重的唯一合法依据是「Emitted − Truncated」，见 PinnedFactsState。
type PinnedFactsResult struct {
	Block     string
	Emitted   []string
	Truncated []string
	Omitted   []string
}

// PinnedFactsState 是装配点留给会话的固化块发射面快照：boot 装配写入 Set，
// 召回侧只读。去重依据必须跟「实际注入的前缀」走——会话中途重载只换读端
// 快照（索引/检索/召回的 List），缓存前缀仍是 boot 装配的那一块，换入方须
// 携带本状态而不是按新 List 重算（否则会话中途新固化的条目会被误判为「已
// 在前缀」而失去召回正文）。
type PinnedFactsState struct {
	Emitted   []string
	Truncated []string
	Omitted   []string
}

// SetPinnedFactsState 写入装配点测得的固化块发射面（boot/sysprompt 调用）。
func (s *Set) SetPinnedFactsState(st PinnedFactsState) {
	if s == nil {
		return
	}
	s.pinnedFacts = st
}

// PinnedFactsSnapshot 读回固化块发射面快照（会话中途换 Set 时携带用）。
func (s *Set) PinnedFactsSnapshot() PinnedFactsState {
	if s == nil {
		return PinnedFactsState{}
	}
	return s.pinnedFacts
}

// pinnedRecallExcluded 返回召回去重名单：已发射且正文未截断（Emitted −
// Truncated）。截断条目保留在召回面（召回行还带描述与引用键，可补足上下文）；
// 未发射条目更不能排除——否则与固化块溢出叠加成两处皆失。
func (s *Set) pinnedRecallExcluded() map[string]bool {
	excluded := map[string]bool{}
	if s == nil {
		return excluded
	}
	cut := make(map[string]bool, len(s.pinnedFacts.Truncated))
	for _, n := range s.pinnedFacts.Truncated {
		cut[n] = true
	}
	for _, n := range s.pinnedFacts.Emitted {
		if !cut[n] {
			excluded[n] = true
		}
	}
	return excluded
}

// BuildPinnedFacts 生成固化事实正文注入块及其发射面。mems 为当前空间的活跃
// 记忆（调用方已按空间取数，本函数不感知空间）；仅取 Pinned 且有正文的条目，
// 按 Name 升序（前缀稳定：touch/触达不翻动块内容），渲染为：
//
//	【固化记忆】
//	- 名称：正文（≤400 rune，UTF-8 边界截断加省略标）
//	...
//	（另有 N 条固化记忆超出本块预算未入块：…；正文可用 memory_get 按名读取）
//
// 块总长度 ≤ maxRunes（rune 计数；≤0 时用 DefaultPinnedFactsBudget）：整行
// 放不下即不计入（不切行，预算诚实），溢出行进 Omitted 并在块尾追加预算内
// 的溢出提示；提示放不下时从已发射尾部逐条挪回 Omitted（候选有序，循环必然
// 终止），最坏=纯提示块。预算连块头都放不下、或无条目可渲染时 Block 为空串
// （零注入，前缀逐字节不变），发射面字段仍如实带回。
func BuildPinnedFacts(mems []Memory, maxRunes int) PinnedFactsResult {
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
			continue // 正文空：无「随快照」内容，索引行已覆盖（召回仍可达）
		}
		pinned = append(pinned, m)
	}
	sort.Slice(pinned, func(i, j int) bool {
		return pinnedFactName(pinned[i]) < pinnedFactName(pinned[j])
	})

	// 预渲染候选行并记录正文是否被单条预算截断（截断=正文未完整在场，该条
	// 不参与召回去重）。
	type pinnedCand struct {
		name string
		line string
		cut  bool
	}
	cands := make([]pinnedCand, 0, len(pinned))
	for _, m := range pinned {
		name := pinnedFactName(m)
		if name == "" {
			continue
		}
		body := strings.Join(strings.Fields(m.Body), " ") // 单行化：压掉换行与连续空白
		cut := utf8.RuneCountInString(body) > pinnedFactsBodyCap
		line := "- " + name + "：" + truncatePinnedBody(body) + "\n"
		cands = append(cands, pinnedCand{name: name, line: line, cut: cut})
	}

	headerRunes := utf8.RuneCountInString(pinnedFactsHeader + "\n")
	if headerRunes > maxRunes {
		return PinnedFactsResult{} // 预算连块头都放不下：诚实零注入
	}

	var b strings.Builder
	b.WriteString(pinnedFactsHeader + "\n")
	written := headerRunes
	var emitted, truncated, omitted []string
	var lines []pinnedCand
	for _, c := range cands {
		n := utf8.RuneCountInString(c.line)
		if written+n > maxRunes {
			omitted = append(omitted, c.name)
			continue // 预算不够整行：不切行，记溢出（不 break——更短的后续行仍可入块）
		}
		b.WriteString(c.line)
		written += n
		lines = append(lines, c)
		emitted = append(emitted, c.name)
		if c.cut {
			truncated = append(truncated, c.name)
		}
	}

	// 块级溢出提示计入预算（众神殿 action-2：提示不能成为新的静默截断）。
	// 先在当前余量内降级点名数取放得下的形态；所有形态都放不下才从已发射
	// 尾部逐条挪回 Omitted（同步修正 Emitted/Truncated 面）给提示腾位——内容
	// 优先于提示点名数。挪空仍放不下则放弃提示（预算病态，块保持已发射部分；
	// Omitted 面仍如实带回给召回侧，未发射条目不会两处皆失）。
	noticeWritten := false
	if len(omitted) > 0 {
		for {
			if notice, ok := pinnedOverflowNoticeFit(omitted, maxRunes-written); ok {
				b.WriteString(notice)
				noticeWritten = true
				break
			}
			if len(lines) == 0 {
				break // 挪空仍放不下：放弃提示
			}
			last := lines[len(lines)-1]
			lines = lines[:len(lines)-1]
			written -= utf8.RuneCountInString(last.line)
			emitted = emitted[:len(emitted)-1]
			if last.cut {
				truncated = truncated[:len(truncated)-1]
			}
			omitted = append([]string{last.name}, omitted...) // 尾部条目=Omitted 头的前身，升序保持
		}
	}

	res := PinnedFactsResult{Emitted: emitted, Truncated: truncated, Omitted: omitted}
	if len(emitted) == 0 && !noticeWritten {
		return res // 无条目或只有块头（提示没挤进预算）：零注入，发射面诚实带回
	}
	res.Block = b.String()
	return res
}

// pinnedOverflowNotice 渲染块级溢出提示行：点名 omitted 前 shown 个名称
// （Name 升序），超出的以「等 N 条」收口。
func pinnedOverflowNotice(omitted []string, shown int) string {
	if shown > len(omitted) {
		shown = len(omitted)
	}
	var b strings.Builder
	b.WriteString("（另有 ")
	b.WriteString(strconv.Itoa(len(omitted)))
	b.WriteString(" 条固化记忆超出本块预算未入块")
	if shown > 0 {
		b.WriteString("：")
		b.WriteString(strings.Join(omitted[:shown], "、"))
		if shown < len(omitted) {
			b.WriteString(" 等")
		}
	}
	b.WriteString("；正文可用 memory_get 按名读取）\n")
	return b.String()
}

// pinnedOverflowNoticeFit 在 budgetRunes 内取放得下的最长点名形态（逐级降
// 点名数，0 个名称的最短形态都不行则返回 false）。
func pinnedOverflowNoticeFit(omitted []string, budgetRunes int) (string, bool) {
	maxShown := len(omitted)
	if maxShown > pinnedOverflowMaxNames {
		maxShown = pinnedOverflowMaxNames
	}
	for shown := maxShown; shown >= 0; shown-- {
		line := pinnedOverflowNotice(omitted, shown)
		if utf8.RuneCountInString(line) <= budgetRunes {
			return line, true
		}
	}
	return "", false
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
