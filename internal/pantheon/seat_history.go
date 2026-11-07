package pantheon

import (
	"fmt"
	"strings"
)

// buildSeatHistory 从贡献记录确定性重建某席位视角的往轮会议记录（2026-11-07
// A5③ 官方共享上下文）。不维护可变台账：跳过/重试/恢复/重启之后按同一函数
// 重建，永远与存档一致——台账会腐化，存档不会。
//
// 重建按贡献存档序（=完成序）严格回放：他人发言与用户插话累积进 user 侧，
// 本席自己的下一次发言触发一次 user+assistant 配对——因此「某条发言出现在
// 本席哪条消息之前」与现场完全一致，没有轮内时序错位。本席缺席（失败/跳过）
// 期间的他人发言顺延累积，与本席下一次发言配对；本席自己的连续条目（讨论+
// 主席小结）合并为一条 assistant 消息。全文保真是这条链的存在意义（「往轮
// 逐字保留」），宿主侧不做任何截断；长场超窗交给引擎侧会话自身的预算机制
// （compact）兜底。
//
// brief（议事简报）并入第一条 user 消息：历史必须自带议题上下文，否则重建
// 出的会话只有过程没有命题。
//
// beforeRound 是信息可见性边界：历史只含 Round < beforeRound 的贡献。讨论与
// 主席小结传当前轮（同轮可见性由提示词的共享段负责——独立开场第 1 轮的
// 「互不可见」纪律因此不被历史泄漏破坏）；汇总传 0 = 无界（全场都是历史）。
func buildSeatHistory(session *Session, agentID string, brief string, beforeRound int) []SeatMessage {
	var out []SeatMessage
	var pendingUser []string
	var pendingAssistant []string
	briefPlaced := false
	flushPair := func() {
		if len(pendingAssistant) == 0 {
			return // 只在有本席发言时配对；纯他人内容继续累积等待。
		}
		body := strings.Join(pendingUser, "\n\n")
		if !briefPlaced && brief != "" {
			body = strings.TrimSpace(brief + "\n\n" + body)
			briefPlaced = true
		}
		out = append(out, SeatMessage{Role: "user", Content: "—— 会议记录（新增发言与插话）——\n\n" + body})
		out = append(out, SeatMessage{Role: "assistant", Content: strings.Join(pendingAssistant, "\n\n")})
		pendingUser = nil
		pendingAssistant = nil
	}
	for _, c := range session.Contributions {
		content := strings.TrimSpace(c.Content)
		if content == "" || c.Phase == PhaseSynthesis {
			// 汇总是终点产物，不再有后续席位调用消费它；跳过防旧存档
			// Round=0 扭曲时序。
			continue
		}
		if beforeRound > 0 && c.Round >= beforeRound {
			// 同轮可见性归提示词共享段管（含独立开场「互不可见」纪律），
			// 历史只承载严格往轮。
			continue
		}
		if c.AgentID == agentID {
			if c.Phase == PhaseModeration {
				pendingAssistant = append(pendingAssistant, "（你的第 "+fmt.Sprint(c.Round)+" 轮主席小结）\n"+content)
				continue
			}
			pendingAssistant = append(pendingAssistant, content)
			continue
		}
		// 遇到他人发言前，先把本席已累积的发言配对落位——保证「他人这条
		// 发言在本席哪条消息之后」与现场一致。
		flushPair()
		pendingUser = append(pendingUser, seatRecordPart(c))
	}
	flushPair()
	if len(pendingUser) > 0 {
		// 尾部纯他人内容（本席最后缺席的收尾轮次）：以纯 user 消息保留。
		body := strings.Join(pendingUser, "\n\n")
		if !briefPlaced && brief != "" {
			body = strings.TrimSpace(brief + "\n\n" + body)
		}
		out = append(out, SeatMessage{Role: "user", Content: "—— 会议记录（新增发言与插话）——\n\n" + body})
	}
	return out
}

// fullWindowBudgetRunes 全量窗口预算（rune）：模型席位的往轮全文内联上限。
// 超出时从最旧开始整体舍弃（带省略标注），最新的发言永远保真——clampPrompt
// 的 60000 总预算还要装简报、指令与当轮，往轮记录只分到一部分。
const fullWindowBudgetRunes = 36000

// fullRecordBlock 全量窗口：全部贡献按存档序逐字内联（含本席自己的发言），
// 预算超限时舍最旧。A5③ 实弹回退后的模型席位记忆通道（Seed 已证伪：
// 模型带着 Seed 历史仍跑 12 步全工具循环，全量窗口的显式文本更可预期）。
func fullRecordBlock(contributions []Contribution, agentID string, budgetRunes int) string {
	items := make([]Contribution, 0, len(contributions))
	for _, c := range contributions {
		if strings.TrimSpace(c.Content) == "" || c.Phase == PhaseSynthesis {
			continue
		}
		items = append(items, c)
	}
	render := func(c Contribution) string {
		if agentID != "" && c.AgentID == agentID && c.Phase != PhaseModeration {
			return "（你此前的发言 · 第 " + fmt.Sprint(c.Round) + " 轮）\n" + strings.TrimSpace(c.Content)
		}
		return seatRecordPart(c)
	}
	kept := make([]Contribution, 0, len(items))
	used := 0
	dropped := 0
	for i := len(items) - 1; i >= 0; i-- {
		size := len([]rune(render(items[i])))
		if used+size > budgetRunes && len(kept) > 0 {
			dropped = i + 1
			break
		}
		used += size
		kept = append([]Contribution{items[i]}, kept...)
	}
	var b strings.Builder
	if dropped > 0 {
		fmt.Fprintf(&b, "（更早的 %d 条发言因窗口预算省略，全文见归档文件）\n\n", dropped)
	}
	for _, c := range kept {
		b.WriteString(render(c))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

// seatRecordPart 一条他人贡献在席位会议记录里的呈现（A5③ 历史重建与 A5①
// 增量投递共用）：轮次+说话者标注+全文——两路消费方对「同一条发言长什么样」
// 必须看到同一形状，否则跨形态记忆出现两套真相。
func seatRecordPart(c Contribution) string {
	label := "【" + c.AgentName + "】"
	switch {
	case c.Phase == PhaseUserAnswer:
		label = "【用户回答】"
	case c.Phase == PhaseUserNote || c.AgentID == "user":
		label = "【用户插话】"
	case c.Phase == PhaseModeration:
		label = "【主席轮间小结 · " + c.AgentName + "】"
	}
	return "（第 " + fmt.Sprint(c.Round) + " 轮）" + label + "\n" + strings.TrimSpace(c.Content)
}

// deltaContributionItems 返回 agentID 最后一次发言之后的全部条目（存档序），
// 即 A5① 增量投递的载荷：续接席位的引擎侧会话已持有此前全部交互，提示词
// 只需要「自你上次发言以来的新增」。synthesis 条目是终点产物，永不入载荷。
func deltaContributionItems(contributions []Contribution, agentID string) []Contribution {
	last := -1
	for i, c := range contributions {
		if c.AgentID == agentID && c.Phase != PhaseSynthesis {
			last = i
		}
	}
	items := make([]Contribution, 0, len(contributions))
	for _, c := range contributions[last+1:] {
		if strings.TrimSpace(c.Content) == "" || c.Phase == PhaseSynthesis {
			continue
		}
		items = append(items, c)
	}
	return items
}
