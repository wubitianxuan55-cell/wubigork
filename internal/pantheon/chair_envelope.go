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
	// 议题对齐度（2026-11-16）：主席轮间小结对「本轮各席是否锚定议题核心
	// 目标」的结构化判定。此前偏航判定只在最终汇总出现一次——轮次花完才
	// 发现漂了；对齐度把它前移成每轮可观测信号。
	chairAlignmentOnTrack = "on-track"
	chairAlignmentDrift   = "drift"
	// chairDriftPullBackThreshold 连续判定漂移达到该轮数，宿主向全员注入
	// 拉回指令。1 轮可能是误判，不动；连续 2 轮才认定真漂移。
	chairDriftPullBackThreshold = 2
)

// ChairAlignment 是信封里主席的议题对齐度判定。Status 空=主席未报（旧提示
// 词或漏填），宿主不作任何推断——缺失不等于在轨。
type ChairAlignment struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

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
	Alignment   string            `json:"alignment,omitempty"`
	DriftNote   string            `json:"driftNote,omitempty"`
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

// parseChairAlignment 解析主席小结信封里的议题对齐度判定。与
// parseChairDecision 共用「最后一个完整信封」纪律；alignment 缺省=未判定
// （返回空 Status，不算 malformed——旧小结与漏填都走这条，不制造告警噪音）；
// 值无法识别时同样返回空 Status 并标 malformed，由调用方降级留痕。
func parseChairAlignment(text string) (ChairAlignment, bool) {
	payload, found := lastChairEnvelopePayload(text)
	if !found {
		return ChairAlignment{}, false
	}
	if strings.TrimSpace(payload) == "" {
		return ChairAlignment{}, true
	}
	var decoded chairEnvelopePayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &decoded); err != nil {
		return ChairAlignment{}, true
	}
	status := strings.ToLower(strings.TrimSpace(decoded.Alignment))
	status = strings.ReplaceAll(status, "_", "-")
	switch status {
	case "":
		return ChairAlignment{}, false
	case chairAlignmentOnTrack:
		return ChairAlignment{Status: chairAlignmentOnTrack}, false
	case chairAlignmentDrift:
		return ChairAlignment{Status: chairAlignmentDrift, Note: trimRunes(strings.TrimSpace(decoded.DriftNote), 200)}, false
	default:
		return ChairAlignment{}, true
	}
}

// consecutiveDriftRounds 统计存档里「最新连续 drift 判定」的小结轮数（从
// 最新主席小结往回数，遇到非 drift 小结即停）。入参是追加当前轮之前的
// 贡献存档——当前轮是否 drift 由调用方自行 +1。判定缺失（主席未报）按
// 保守处理：断开连击，不把「没说」当成「漂了」。
func consecutiveDriftRounds(contributions []Contribution) int {
	streak := 0
	for i := len(contributions) - 1; i >= 0; i-- {
		c := contributions[i]
		if c.Phase != PhaseModeration {
			continue
		}
		if c.ChairAlignment != chairAlignmentDrift {
			break
		}
		streak++
	}
	return streak
}

// chairPullBackNotice 是宿主在连续漂移达到阈值时追加到主席小结可见文本
// 末尾的拉回指令。小结全文会作为下一轮议程注入全员提示词，因此追加在这里
// 即达全员，无需改动各提示词构造器；【宿主】署名保证不与主席原话混淆。
func chairPullBackNotice(round, streak int, note string) string {
	text := fmt.Sprintf("连续 %d 轮主席判定议题对齐度为「漂移」", streak)
	if note != "" {
		text += "：" + note
	} else {
		text += "。"
	}
	return fmt.Sprintf("\n\n【宿主拉回（第 %d 轮）】%s 下一轮全员发言的第一段必须先答「本轮论证如何推进议题核心目标」；与核心目标无关的展开一律移入遗留台账，不得另起炉灶。", round, text)
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
