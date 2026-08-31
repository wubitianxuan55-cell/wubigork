package memory

// ── 晨报预载块（v4.16 刀④：高频工作记忆预装配进 agent 上下文）──────────────
// BuildMorningPreloadBlock 把高频工作记忆确定性聚合为一段「晨报预载」注入块：
// 会话装配（系统提示词记忆索引处）时预装配进 agent 上下文。零 LLM、零 IO、
// 预算受限、work 空间只读——与 v4.14 BuildMorningBrief（首页晨报卡片）同源
// 排序口径（max(UpdatedAt,LastUsedAt) 降序），但输出是注入文本块而非视图结构；
// 装配点按空间取数（InSpace 视图），本函数不感知空间。

import (
	"strings"
	"time"
	"unicode/utf8"
)

// DefaultMorningPreloadBudget 是晨报预载块默认总预算（rune）。对齐画像注入
// 预算 600 的精神（profileBudget，recall.go），保证预载块不挤占 ProfileBlock/
// RecallBlock 的既有注入额度。
const DefaultMorningPreloadBudget = 600

// morningPreloadHeader 是预载块头（无注入行时用于早退判断）。
const morningPreloadHeader = "【工作记忆晨报】"

// BuildMorningPreloadBlock 生成确定性晨报预载块。mems 为当前空间的活跃记忆
// （调用方已按空间取数，本函数不感知空间）；按 max(UpdatedAt, LastUsedAt)
// 降序（同刻 Name 升序，复用 BuildMorningBrief 排序口径）取高频条目，渲染为：
//
//	【工作记忆晨报】
//	- 名称：摘要（≤120 rune，UTF-8 边界截断）
//	...
//
// 块总长度 ≤ maxRunes（rune 计数；≤0 时用 DefaultMorningPreloadBudget）：
// 整行放不下时按 UTF-8 字符边界截断到剩余预算后停止（不切开多字节字符）；
// 预算不足容纳块头时返回空串。空输入或无可渲染条目（名称与内容全空）返回
// 空串。now 用于锚定确定性输出（预留；当前渲染不依赖时间，仅约束签名供
// 测试注入固定时刻）。
func BuildMorningPreloadBlock(mems []Memory, now time.Time, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = DefaultMorningPreloadBudget
	}
	sorted := make([]Memory, len(mems))
	copy(sorted, mems)
	morningRecencySort(sorted)

	var b strings.Builder
	b.WriteString(morningPreloadHeader + "\n")
	written := utf8.RuneCountInString(morningPreloadHeader + "\n")
	if written > maxRunes {
		return "" // 预算连块头都放不下：诚实返回空串，不注入残缺块
	}
	for _, m := range sorted {
		line := formatMorningPreloadLine(m)
		if line == "" {
			continue
		}
		lineRunes := utf8.RuneCountInString(line)
		if written+lineRunes > maxRunes {
			// 预算不够整行：按 UTF-8 边界截断到剩余预算，追加后停止。
			if remain := maxRunes - written; remain > 0 {
				rs := []rune(line)
				if remain < len(rs) {
					b.WriteString(string(rs[:remain]))
				} else {
					b.WriteString(line)
				}
			}
			break
		}
		b.WriteString(line)
		written += lineRunes
	}
	block := strings.TrimSpace(b.String())
	if block == morningPreloadHeader {
		return "" // 只有块头没有注入行：与空记忆同语义，不注入
	}
	return block
}

// formatMorningPreloadLine 渲染一条预载行：「- 名称：摘要」。名称取 Name
// （空则 Title），摘要取 Description（空则 Body），单行化并截断到 120 rune
// （复用晨报条目描述预算，对齐「5 × 120 = 600」精神）。名称与内容全空返回
// ""（跳过该条）。
func formatMorningPreloadLine(m Memory) string {
	name := strings.TrimSpace(m.Name)
	if name == "" {
		name = strings.TrimSpace(m.Title)
	}
	desc := strings.TrimSpace(m.Description)
	if desc == "" {
		desc = strings.TrimSpace(m.Body)
	}
	if name == "" || desc == "" {
		return ""
	}
	return "- " + name + "：" + truncateMorningDesc(desc) + "\n"
}
