package memory

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// ── 项目本体注入（阶段六 6.2 首刀）────────────────────────────────
//
// BuildProjectBrief 把记忆投影成「项目事实表」：口径/模板/命名规范/历史
// 决策，每行带稳定引用键 [MEM:<name>]（模型采纳时句末引用 → 触达与徽标
// 渲染同源）与决策来源归因（沉淀会话/轮次）。注入点=work 空间会话装配
// （缓存稳定前缀，跨会话项目连续性），可关（config project_brief）。
//
// 纯函数：同一输入（记忆 + 时钟值 + 预算）输出逐字节一致。

// projectBriefBudgetRunes 是注入块的默认 rune 预算（与画像注入 profileBudget
// 同量级；项目事实贵在准不在多）。
const projectBriefBudgetRunes = 600

// sourceAttribution 把沉淀来源（session-20260810-xxx.jsonl / turn 3）归一成
// 决策引用文案；两者皆空返回空串（存量行无来源，诚实不造数）。
func sourceAttribution(m Memory) string {
	session := strings.TrimSpace(m.SourceSession)
	session = strings.TrimSuffix(session, ".jsonl")
	turn := strings.TrimSpace(m.SourceMessage)
	switch {
	case session != "" && turn != "":
		return fmt.Sprintf("依据 %s 会话 · %s", session, turn)
	case session != "":
		return fmt.Sprintf("依据 %s 会话", session)
	case turn != "":
		return fmt.Sprintf("依据 %s", turn)
	default:
		return ""
	}
}

// briefItem 是项目事实表的一个候选（排序用）。
type briefItem struct {
	m     Memory
	score float64
}

// BuildProjectBrief 渲染项目本体注入块。选择口径：固化全收（用户明示保留的
// 决策/规范），再补 project/feedback 型按衰减评分降序；预算内按行完整收录
// （截行不截半句）。无候选或全超预算返回 ""（调用方零注入，前缀不变）。
func BuildProjectBrief(mems []Memory, now time.Time, budgetRunes int) string {
	if budgetRunes <= 0 {
		budgetRunes = projectBriefBudgetRunes
	}
	var pinned, others []briefItem
	for _, m := range mems {
		if m.Pinned {
			pinned = append(pinned, briefItem{m, DecayScore(m, now)})
			continue
		}
		if m.Type == TypeProject || m.Type == TypeFeedback {
			others = append(others, briefItem{m, DecayScore(m, now)})
		}
	}
	// 组内确定性排序：评分降序 → 名称升序兜底。
	sort.SliceStable(pinned, func(i, j int) bool {
		if pinned[i].score != pinned[j].score {
			return pinned[i].score > pinned[j].score
		}
		return pinned[i].m.Name < pinned[j].m.Name
	})
	sort.SliceStable(others, func(i, j int) bool {
		if others[i].score != others[j].score {
			return others[i].score > others[j].score
		}
		return others[i].m.Name < others[j].m.Name
	})

	var b strings.Builder
	b.WriteString("## Project fact sheet（项目本体 · 依据历史会话决策）\n")
	b.WriteString("以下口径/规范/决策来自既往会话沉淀，遵循并在采纳处句末标注引用键：\n")
	written := 0
	used := map[string]bool{}
	appendLine := func(it briefItem) bool {
		desc := strings.TrimSpace(oneLine(it.m.Description))
		if desc == "" {
			desc = strings.TrimSpace(displayTitle(it.m.Title, it.m.Name))
		}
		if desc == "" {
			return false
		}
		attr := sourceAttribution(it.m)
		tag := ""
		if it.m.Pinned {
			tag = "固化 · "
		}
		if attr != "" {
			attr = "（" + tag + attr + "）"
		} else if tag != "" {
			attr = "（" + strings.TrimRight(tag, " ·") + "）"
		}
		line := fmt.Sprintf("- %s %s%s\n", "[MEM:"+it.m.Name+"]", desc, attr)
		n := utf8.RuneCountInString(line)
		if written+n > budgetRunes {
			return false
		}
		b.WriteString(line)
		written += n
		used[it.m.Name] = true
		return true
	}
	for _, it := range pinned {
		appendLine(it)
	}
	for _, it := range others {
		if used[it.m.Name] {
			continue
		}
		appendLine(it)
	}
	if written == 0 {
		return ""
	}
	return strings.TrimRight(b.String(), "\n")
}
