package memory

// ── 晨报（做梦 2.0 主动预取 MVP：纯本地晨报）─────────────────────────
// BuildMorningBrief 是纯函数：零 LLM、零 IO、确定性输出。输入当前空间的
// 记忆列表 + 近 24h dream 沉淀计数，输出前端「今日晨报」卡片的全部数据。
//
// 输出纪律：
//   - Items：top5，按 max(UpdatedAt, LastUsedAt) 降序；Type∈{user,project}
//     的事实优先，不足用其余补位（优先级不破坏各自的时序相对序）；
//   - Rules：Kind∈{procedural,rule} 按同一排序取 ≤3 条描述；
//   - 空输入 → 空结构（Items/Rules 为非 nil 空数组，前端 JSON.parse 后
//     直接可得 []，不会被 null 炸穿渲染）；
//   - 总预算 ~600 rune 精神：Description 截断到 120 rune 即可，不做精确
//     计量（5 × 120 = 600，正好在预算内）。

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// BriefItem 是晨报里的一条记忆（前端渲染名称 + 描述摘要）。
type BriefItem struct {
	Name        string `json:"name"`
	Description string `json:"description"` // ≤120 rune，UTF-8 边界截断
	Kind        string `json:"kind,omitempty"`
	Category    string `json:"category,omitempty"`
	UpdatedAt   int64  `json:"updatedAt"`            // Unix 毫秒
	LastUsedAt  int64  `json:"lastUsedAt,omitempty"` // Unix 毫秒，零值省略
}

// Brief 是「今日晨报」完整数据（JSON 串绑定，前端 JSON.parse 后渲染）。
type Brief struct {
	Items       []BriefItem `json:"items"` // top5
	Rules       []string    `json:"rules"` // ≤3 条
	Dreamed24h  int         `json:"dreamed24h"`
	GeneratedAt int64       `json:"generatedAt"` // Unix 毫秒
}

const (
	morningItemBudget = 5   // Items top5
	morningRuleBudget = 3   // Rules ≤3
	morningDescRunes  = 120 // Description 截断预算（rune）
)

// morningRankTime 返回记忆的排序键时间：max(UpdatedAt, LastUsedAt)。
func morningRankTime(m Memory) time.Time {
	if m.LastUsedAt.After(m.UpdatedAt) {
		return m.LastUsedAt
	}
	return m.UpdatedAt
}

// truncateMorningDesc 按 rune 边界截断描述（不切开多字节 UTF-8 字符）：
// 超预算时取前 119 rune 追加 "…"（共 120 rune，对齐项目既有 "…" 截断惯例）。
func truncateMorningDesc(s string) string {
	s = strings.TrimSpace(oneLine(s))
	if utf8.RuneCountInString(s) <= morningDescRunes {
		return s
	}
	rs := []rune(s)
	return string(rs[:morningDescRunes-1]) + "\u2026"
}

// morningRecencySort 按 max(UpdatedAt,LastUsedAt) 降序稳定排序，同刻按
// Name 升序兜底（与 ProfileBlock 排序口径一致，保证输出确定、可测）。
func morningRecencySort(mems []Memory) {
	sort.SliceStable(mems, func(i, j int) bool {
		ti, tj := morningRankTime(mems[i]), morningRankTime(mems[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return mems[i].Name < mems[j].Name
	})
}

// morningPreferred 判断事实是否优先进入晨报条目（Type=user/project 优先，
// 对齐设计「kind∈{user,project} 优先」——Memory 的类型轴为 Type，认知轴
// Kind 无 user/project 取值，故按 Type 判定）。
func morningPreferred(m Memory) bool {
	return m.Type == TypeUser || m.Type == TypeProject
}

// BuildMorningBrief 生成确定性晨报。mems 为当前空间的活跃记忆（调用方
// 已按空间取数，本函数不感知空间）；dreamed24h 为近 24h 沉淀计数（统计
// 失败时调用方传 0，不阻断晨报生成）；now 仅用于 GeneratedAt 落毫秒。
func BuildMorningBrief(mems []Memory, dreamed24h int, now time.Time) Brief {
	b := Brief{
		Items:       []BriefItem{},
		Rules:       []string{},
		Dreamed24h:  dreamed24h,
		GeneratedAt: now.UnixMilli(),
	}
	if len(mems) == 0 {
		return b
	}

	// 1) 全量按时序排序（排序键=max(UpdatedAt,LastUsedAt) 降序）。
	sorted := make([]Memory, len(mems))
	copy(sorted, mems)
	morningRecencySort(sorted)

	// 2) Items：user/project 优先（保持各自时序相对序），不足用其余补位。
	preferred := make([]Memory, 0, len(sorted))
	rest := make([]Memory, 0, len(sorted))
	for _, m := range sorted {
		if morningPreferred(m) {
			preferred = append(preferred, m)
		} else {
			rest = append(rest, m)
		}
	}
	for _, m := range append(preferred, rest...) {
		if len(b.Items) >= morningItemBudget {
			break
		}
		item := BriefItem{
			Name:        m.Name,
			Description: truncateMorningDesc(m.Description),
			Kind:        string(m.Kind),
			Category:    string(m.Type),
			UpdatedAt:   m.UpdatedAt.UnixMilli(),
		}
		if !m.LastUsedAt.IsZero() {
			item.LastUsedAt = m.LastUsedAt.UnixMilli()
		}
		b.Items = append(b.Items, item)
	}

	// 3) Rules：Kind∈{procedural,rule} 按同一排序取 ≤3 条描述（无则空数组）。
	for _, m := range sorted {
		if len(b.Rules) >= morningRuleBudget {
			break
		}
		if m.Kind != KindProcedural && m.Kind != Kind("rule") {
			continue
		}
		if desc := truncateMorningDesc(m.Description); desc != "" {
			b.Rules = append(b.Rules, desc)
		}
	}
	return b
}
