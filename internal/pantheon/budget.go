package pantheon

import (
	"fmt"
	"strings"
	"time"
)

// B4 整场预算阶梯（2026-11-07，蒸馏自 Agorai orchestrator 按消耗分档）。
//
// 预算单位是议事墙钟（分钟），不是 token：宿主的 token 账目只覆盖自报
// Usage 的席位（DSH/ZCode），codex 只报会话 ID、模型席不回 Usage——按
// token 设阶梯会系统性少算半数席位；墙钟对所有席位同样可测，也正是实弹
// 痛点本身（4 席 4 轮 83 分钟、140 分钟被取消）。
//
// 阶梯两档，只在宿主确定性检查点生效：
//   - ≥80%：历史降档——往轮全文形态（seeded/全量窗口）退回梯度分发；
//     delta 已是最低投递形态，不动。
//   - ≥90%：轮间收口——本轮已收声，不再追加轮次，跳过轮间小结直接汇总。
//     余下 10% 留给汇总：汇总是交付物，允许越过预算线完成。
//
// 不抄上游「减席」档：缺席≠弃权（前序档 C3），砍席会造出替缺席者裁决的
// 结局。预算为 0 = 不限，行为与预算落地前完全一致。
const (
	budgetHistoryTier = 80
	budgetRoundTier   = 90
)

func timeBudget(req StartRequest) time.Duration {
	if req.TimeBudgetMinutes <= 0 {
		return 0
	}
	return time.Duration(req.TimeBudgetMinutes) * time.Minute
}

// budgetPercent 返回已用时占预算的百分比；无预算返回 0。可以超过 100。
func budgetPercent(elapsed, budget time.Duration) int {
	if budget <= 0 || elapsed <= 0 {
		return 0
	}
	return int(elapsed * 100 / budget)
}

// budgetTier 返回已触达的最高阶梯档（0 / budgetHistoryTier / budgetRoundTier）。
func budgetTier(elapsed, budget time.Duration) int {
	switch percent := budgetPercent(elapsed, budget); {
	case percent >= budgetRoundTier:
		return budgetRoundTier
	case percent >= budgetHistoryTier:
		return budgetHistoryTier
	default:
		return 0
	}
}

// budgetDowngradeMode 是第一档的唯一动作：高水位时把往轮全文形态退回梯度。
// 其余形态原样返回——梯度本就是基线，delta 比梯度更省，不能「降」成梯度。
func budgetDowngradeMode(mode seatHistoryMode, tier int) seatHistoryMode {
	if tier < budgetHistoryTier {
		return mode
	}
	if mode == historySeeded || mode == historyFullWindow {
		return historyGradient
	}
	return mode
}

func formatBudgetMinutes(d time.Duration) string {
	return fmt.Sprintf("%.1f", d.Minutes())
}

// budgetStopText 是收口事件与警示共用的一句人话：用时、预算、百分比、动作。
func budgetStopText(round int, elapsed, budget time.Duration) string {
	return fmt.Sprintf(
		"第 %d 轮后整场议事已用时 %s / %s 分钟（%d%%），触发预算收口：停止追加轮次，跳过轮间小结，直接进入协作汇总。",
		round, formatBudgetMinutes(elapsed), formatBudgetMinutes(budget), budgetPercent(elapsed, budget),
	)
}

// sessionBudgetElapsed 是阶梯的唯一时钟：开议起的墙钟。恢复/重启后继续
// 累计——预算约束的是这场议事的总时长，不是单个进程的寿命。
func (m *Manager) sessionBudgetElapsed(session *Session) time.Duration {
	if m.budgetElapsed != nil {
		return m.budgetElapsed(session)
	}
	if session.CreatedAt <= 0 {
		return 0
	}
	return time.Since(time.UnixMilli(session.CreatedAt))
}

// sessionBudgetTier 是 runSeat 降档检查点的入口；无预算恒为 0，零开销。
func (m *Manager) sessionBudgetTier(session *Session) int {
	budget := timeBudget(session.Request)
	if budget <= 0 {
		return 0
	}
	return budgetTier(m.sessionBudgetElapsed(session), budget)
}

// budgetStopsAfterRound 是轮间收口检查点：还有下一轮且已达收口档时，落
// 事件与警示并返回 true。自然末轮不评估——避免对本就要结束的会议报警。
func (m *Manager) budgetStopsAfterRound(session *Session, round int) bool {
	budget := timeBudget(session.Request)
	if budget <= 0 || round >= session.Request.Rounds {
		return false
	}
	elapsed := m.sessionBudgetElapsed(session)
	if budgetTier(elapsed, budget) < budgetRoundTier {
		return false
	}
	text := budgetStopText(round, elapsed, budget)
	m.mu.Lock()
	event := m.appendEventLocked(session, Event{
		Phase: PhaseDiscussion,
		Round: round,
		Kind:  EventBudgetStop,
		Text:  text,
		At:    time.Now().UnixMilli(),
	})
	m.mu.Unlock()
	m.emitSafe(event)
	m.addWarning(session, text)
	return true
}

// budgetStopRound 从事件流回读收口轮次（0 = 未收口）。事件是续跑与导出的
// 单一事实源，不另设会话字段。
func budgetStopRound(session *Session) int {
	round := 0
	for _, event := range session.Events {
		if event.Kind == EventBudgetStop && event.Round > round {
			round = event.Round
		}
	}
	return round
}

// writeBudgetSignals 让预算消耗可审计：有预算的会话无论何种协议都写这一节。
func writeBudgetSignals(b *strings.Builder, session *Session) {
	budget := timeBudget(session.Request)
	if budget <= 0 {
		return
	}
	b.WriteString("## 预算\n\n")
	fmt.Fprintf(b, "- 整场议事时间预算：%d 分钟（≥%d%% 历史降档；≥%d%% 轮间收口）\n",
		session.Request.TimeBudgetMinutes, budgetHistoryTier, budgetRoundTier)
	if session.CreatedAt > 0 && session.UpdatedAt >= session.CreatedAt {
		elapsed := time.Duration(session.UpdatedAt-session.CreatedAt) * time.Millisecond
		fmt.Fprintf(b, "- 截至最后更新已用时：%s 分钟（%d%%）\n", formatBudgetMinutes(elapsed), budgetPercent(elapsed, budget))
	}
	if round := budgetStopRound(session); round > 0 {
		fmt.Fprintf(b, "- 收口：第 %d 轮后触发预算收口，未追加后续轮次\n", round)
	} else {
		b.WriteString("- 收口：未触发\n")
	}
	b.WriteString("\n")
}
