package pantheon

import (
	"encoding/json"
	"fmt"
	"strings"
)

// A1 主席带外控制信封（2026-11-07，蒸馏自 OpenMausBot group-goal-run.ts 的
// goal envelope）：主席轮间小结在人类可见文本之外，行末可附一个私有信封，
// 对单个席位下发下一轮定向指令。解析纪律与上游同款：
//   - 只认最后一个完整信封（防引用示例劫持——模型复述示例后再发真信封时，
//     后者胜出；只复述示例不发真信封属提示词纪律问题，与上游取舍一致）；
//   - 可见文本剥除所有完整信封与未闭合残片后才入档/上屏；
//   - 解析失败降级为普通小结并留痕——「不为结构化字段牺牲有效发言」
//     （V2 路线图不抄 #9）。
const (
	chairEnvelopeOpen  = "<pantheon-chair>"
	chairEnvelopeClose = "</pantheon-chair>"
	// chairAssignmentLimit 信封派工条数上限：派工是收口手段不是新议程。
	chairAssignmentLimit = 8
	// chairInstructionRunes 单条指令长度上限。
	chairInstructionRunes = 200
)

// ChairAssignment 是信封里对单个席位的定向派工。
type ChairAssignment struct {
	AgentID     string `json:"agentId"`
	Instruction string `json:"instruction"`
}

// ChairDecision 是主席的终局信号（第二次蒸馏 E3，蒸馏自 group-goal-run.ts
// 的 goal envelope status）。它只记录主席说了什么；能不能停由宿主按既有
// 共识/分歧/缺口条件裁决——completed 不越权。
type ChairDecision struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

const (
	chairDecisionContinue   = "continue"
	chairDecisionCompleted  = "completed"
	chairDecisionNeedsInput = "needs-input"
	chairDecisionBlocked    = "blocked"
)

type chairEnvelopePayload struct {
	Decision    string            `json:"decision,omitempty"`
	Detail      string            `json:"detail,omitempty"`
	Assignments []ChairAssignment `json:"assignments"`
}

// parseChairEnvelope 解析主席输出：返回合法派工、剥除信封后的可见文本、
// 以及「出现过信封但没解析出东西」的降级标记（调用方据此留痕）。
func parseChairEnvelope(text string) (assignments []ChairAssignment, visible string, malformed bool) {
	lastPayload, found := lastChairEnvelopePayload(text)
	// 可见文本：剥除所有完整信封；未闭合的 open 尾巴整体丢弃（残片不得上屏）。
	var b strings.Builder
	rest := text
	for {
		open := strings.Index(rest, chairEnvelopeOpen)
		if open < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:open])
		closeIdx := strings.Index(rest[open:], chairEnvelopeClose)
		if closeIdx < 0 {
			break
		}
		rest = rest[open+closeIdx+len(chairEnvelopeClose):]
	}
	visible = strings.TrimSpace(b.String())
	if !found {
		// 有信封开头但无完整收口 = 输出中断的残片：按降级留痕，不当无事发生。
		return nil, visible, strings.Contains(text, chairEnvelopeOpen)
	}
	var payload chairEnvelopePayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(lastPayload)), &payload); err != nil {
		return nil, visible, true
	}
	assignments = make([]ChairAssignment, 0, len(payload.Assignments))
	for _, a := range payload.Assignments {
		id := strings.TrimSpace(a.AgentID)
		instruction := strings.TrimSpace(a.Instruction)
		if id == "" || instruction == "" {
			continue // 空条目静默丢弃：信封是辅助，不值得为空行报错。
		}
		assignments = append(assignments, ChairAssignment{AgentID: id, Instruction: trimRunes(instruction, chairInstructionRunes)})
		if len(assignments) >= chairAssignmentLimit {
			break
		}
	}
	return assignments, visible, false
}

// lastChairEnvelopePayload 返回最后一个完整信封的载荷。只认最后一个完整
// 信封：模型复述示例后再发真信封时，后者胜出；只复述示例不发真信封属提示词
// 纪律问题，与上游取舍一致。
func lastChairEnvelopePayload(text string) (string, bool) {
	idx := strings.LastIndex(text, chairEnvelopeOpen)
	for idx >= 0 {
		end := strings.Index(text[idx:], chairEnvelopeClose)
		if end >= 0 {
			return text[idx+len(chairEnvelopeOpen) : idx+end], true
		}
		idx = strings.LastIndex(text[:idx], chairEnvelopeOpen)
	}
	return "", false
}

// parseChairDecision 解析主席终局信号（E3）。与 parseChairEnvelope 共用
// 「最后一个完整信封」纪律；decision 缺省=continue，旧信封逐字兼容。
// 终局缺 detail、未知 decision、JSON 坏都返回 malformed，由调用方降级留痕。
func parseChairDecision(text string) (ChairDecision, bool) {
	payload, found := lastChairEnvelopePayload(text)
	if !found {
		return ChairDecision{Status: chairDecisionContinue}, false
	}
	if strings.TrimSpace(payload) == "" {
		return ChairDecision{Status: chairDecisionContinue}, true
	}
	var decoded chairEnvelopePayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &decoded); err != nil {
		return ChairDecision{Status: chairDecisionContinue}, true
	}
	status := strings.ToLower(strings.TrimSpace(decoded.Decision))
	status = strings.ReplaceAll(status, "_", "-")
	switch status {
	case "", chairDecisionContinue:
		return ChairDecision{Status: chairDecisionContinue}, false
	case chairDecisionCompleted, chairDecisionNeedsInput, chairDecisionBlocked:
		detail := strings.TrimSpace(decoded.Detail)
		if detail == "" {
			return ChairDecision{Status: chairDecisionContinue}, true
		}
		return ChairDecision{Status: status, Detail: trimRunes(detail, 600)}, false
	default:
		return ChairDecision{Status: chairDecisionContinue}, true
	}
}

// filterChairAssignments 只保留能定位到参会席位的派工；定向到缺席/未知席位
// 的条目返回计数，由调用方决定是否留痕。
func filterChairAssignments(assignments []ChairAssignment, order []string) ([]ChairAssignment, int) {
	known := make(map[string]bool, len(order))
	for _, id := range order {
		known[id] = true
	}
	kept := make([]ChairAssignment, 0, len(assignments))
	dropped := 0
	for _, a := range assignments {
		if known[a.AgentID] {
			kept = append(kept, a)
		} else {
			dropped++
		}
	}
	return kept, dropped
}

// chairInstructionFor 返回上一轮主席小结里对本席的定向派工（A1）。派工随
// moderation Contribution 持久化，此处纯查表——跳过/重试/恢复/重启后与
// 存档一致，无运行态台账。
func chairInstructionFor(session *Session, agentID string, round int) string {
	for i := len(session.Contributions) - 1; i >= 0; i-- {
		c := session.Contributions[i]
		if c.Phase != PhaseModeration || c.Round != round-1 {
			continue
		}
		for _, a := range c.ChairAssignments {
			if a.AgentID == agentID {
				return a.Instruction
			}
		}
	}
	return ""
}

// duplicatedAssignmentWarning 检测 A2c 重复派工（蒸馏自 group-goal-run.ts
// 的 memberId:instruction 三次阻断）：同一（席位, 归一化指令）在主席小结里
// 累计出现 ≥3 次即告警。返回值已按现有 Warnings 去重；一次最多报一条
// （addWarning 不去重，调用方逐轮调用时靠本函数的 Warnings 扫描保证不刷屏）。
func duplicatedAssignmentWarning(session *Session) string {
	counts := map[string]int{}
	sample := map[string]ChairAssignment{}
	for _, c := range session.Contributions {
		if c.Phase != PhaseModeration {
			continue
		}
		for _, a := range c.ChairAssignments {
			key := a.AgentID + "\x00" + normalizeChairInstruction(a.Instruction)
			counts[key]++
			sample[key] = a
		}
	}
	for key, n := range counts {
		if n < 3 {
			continue
		}
		a := sample[key]
		warning := fmt.Sprintf("主席定向派工已重复 %d 次（%s：%s）——应给收口动议或换核验路径，不应再原样派工", n, a.AgentID, a.Instruction)
		already := false
		for _, w := range session.Warnings {
			if w == warning {
				already = true
				break
			}
		}
		if !already {
			return warning
		}
	}
	return ""
}

// normalizeChairInstruction 归一化派工指令：折叠空白差异，只留实质文本。
func normalizeChairInstruction(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
