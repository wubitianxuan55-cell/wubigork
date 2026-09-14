package app

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/util"
)

// ── 章节生成的角色状态机回灌区段（t5 §7.4-2/§7.4-3）────────────
//
// create-chapter 模板 P1 槽位 character_states 的内容来源。P0 characters 区段
// 只有人设与关系类型摘要（buildCharacterSummary/buildRelationDigest），本区段
// 补状态机增量：心理/处境、存活水位、职业、组织归属、亲密度数值。
// MuMu 有两条回灌通道且格式不一致（一行式 vs 多行式，05 §3.9 历史债）——gaea
// 统一多行式（chapter_context_service.py:31-45 口径）。
//
// 降噪规则（照搬 MuMu chapters.py:794/821）：intimacy==50 与 loyalty==50 的
// 默认值不注入。存量项目角色无任何 v2 状态字段时整角色跳过、全员无则返回
// 空串——BuildUserPrompt 不渲染空 context，存量项目零噪声。

const (
	charStatesStateLen = 150  // 心理/处境截断（MuMu chapter_context_service :40-45）
	charStatesRelLen   = 40   // 单条关系描述截断
	charStatesRelMax   = 5    // 关系网络条数上限（MuMu :807）
	charStatesOrgMax   = 3    // 组织归属条数上限（MuMu :791）
	charStatesBudget   = 1500 // 区段整体预算（rune）
)

// buildCharacterStatesSection 生成 character_states 区段文本；无状态机数据时
// 返回空串（容错：读档失败同样空串，绝不中断生成主链路）。
func (a *writingState) buildCharacterStatesSection(pm *project.Manager) string {
	if pm == nil {
		return ""
	}
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil || len(cf.Characters) == 0 {
		return ""
	}

	nameByID := make(map[string]string, len(cf.Characters))
	for _, ch := range cf.Characters {
		nameByID[ch.ID] = ch.Name
	}
	// 组织归属索引：角色 ID → [(组织名, 职位, 忠诚度)]（仅 active，缺省语义 active）
	orgByChar := make(map[string][]string)
	for _, org := range cf.Organizations {
		for _, m := range org.MemberList {
			if m.CharacterID == "" || (m.Status != "" && m.Status != "active") {
				continue
			}
			item := org.Name
			if p := strings.TrimSpace(m.Position); p != "" {
				item += "（" + p + "）"
			}
			// 降噪：默认忠诚度 50 不注入（0 视为未记录，同样不注入）
			if m.Loyalty != 50 && m.Loyalty != 0 {
				item += fmt.Sprintf("[忠诚度:%d]", m.Loyalty)
			}
			orgByChar[m.CharacterID] = append(orgByChar[m.CharacterID], item)
		}
	}
	// 关系索引：角色 ID → 关系条目（双向；past 已被存活级联结束，不注入）
	relByChar := make(map[string][]string)
	for _, r := range cf.Relationships {
		if r.Status == "past" {
			continue
		}
		for _, id := range [2]string{r.FromID, r.ToID} {
			other := r.ToID
			if id == r.ToID {
				other = r.FromID
			}
			otherName := nameByID[other]
			if id == "" || otherName == "" {
				continue
			}
			item := "与" + otherName
			if d := strings.TrimSpace(r.Description); d != "" {
				item += "：" + util.Truncate(d, charStatesRelLen)
			}
			// 降噪：默认亲密度 50 不注入
			if r.Intimacy != 50 {
				item += fmt.Sprintf("[%d]", r.Intimacy)
			}
			relByChar[id] = append(relByChar[id], item)
		}
	}

	var lines []string
	for _, ch := range cf.Characters {
		// 存量角色零 v2 字段 → 整体跳过（区段只承载状态机增量）
		hasV2 := ch.CurrentState != "" || ch.CareerMainLabel() != "" ||
			len(ch.SubCareers) > 0 || len(orgByChar[ch.ID]) > 0 ||
			ch.StatusChangedChapter > 0 || ch.StateUpdatedChapter > 0
		if !hasV2 {
			continue
		}

		head := "【" + ch.Name + "】"
		if emoji := survivalEmoji(ch.Status); emoji != "" {
			head += "(" + emoji + ")"
		}
		lines = append(lines, head)

		if st := strings.TrimSpace(ch.CurrentState); st != "" {
			lines = append(lines, "  当前状态: "+util.Truncate(st, charStatesStateLen))
		} else if ch.Status != "" && ch.Status != "Alive" {
			lines = append(lines, "  当前状态: "+survivalText(ch.Status))
		}
		if lb := ch.CareerMainLabel(); lb != "" {
			lines = append(lines, "  主职业: "+lb)
		}
		if subs := ch.CareerSubLabels(); len(subs) > 0 {
			lines = append(lines, "  副职业: "+strings.Join(subs, "、"))
		}
		if orgs := orgByChar[ch.ID]; len(orgs) > 0 {
			if len(orgs) > charStatesOrgMax {
				orgs = orgs[:charStatesOrgMax]
			}
			lines = append(lines, "  所属组织: "+strings.Join(orgs, "、"))
		}
		if rels := relByChar[ch.ID]; len(rels) > 0 {
			sort.Strings(rels) // 确定性（双向展开后同条关系会出现两次，去重前排序稳定）
			seen := make(map[string]bool, len(rels))
			dedup := make([]string, 0, len(rels))
			for _, r := range rels {
				if !seen[r] {
					seen[r] = true
					dedup = append(dedup, r)
				}
			}
			if len(dedup) > charStatesRelMax {
				dedup = dedup[:charStatesRelMax]
			}
			lines = append(lines, "  关系网络: "+strings.Join(dedup, "；"))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	out := truncateBudget(strings.Join(lines, "\n"), charStatesBudget)
	slog.Info("角色状态回灌区段", "lines", len(lines), "budgetRunes", len([]rune(out)))
	return out
}

// survivalEmoji 存活状态 emoji 标记（MuMu :718-724 口径映射 gaea 值域；
// Alive/Transformed 不标——活着与蜕变都是可出场状态，标注反而成噪声）。
func survivalEmoji(status string) string {
	switch status {
	case "Dead":
		return "💀已死亡"
	case "Missing":
		return "❓已失踪"
	case "Retired":
		return "📤已退场"
	}
	return ""
}

// survivalText 非 Alive 存活状态的中文文本（无心理状态文本时的兜底行）。
func survivalText(status string) string {
	switch status {
	case "Dead":
		return "已死亡（后续章节不得再出场）"
	case "Missing":
		return "已失踪（除非剧情明确回归，不得出场）"
	case "Retired":
		return "已退场（除非剧情明确回归，不得出场）"
	case "Transformed":
		return "已蜕变"
	}
	return ""
}
