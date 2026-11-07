package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// 第二次蒸馏 E1：把一次 attempt 的 tool_started/tool_result 事件压成宿主
// 可核对的摘要。它只回答「宿主看见了几次工具、几次成功、几次失败、有没有
// 同一调用连续失败」，不回答「这席说得对不对」。
//
// 三条边界：
//   - 只读事件，不接受模型在正文里自报的数字；
//   - 证据不进任何提示词，避免模型为了「好看」刷工具；
//   - 不落命令/输出原文，只落计数与工具名，避免把凭证带进纪要。
func (m *Manager) buildToolEvidence(session *Session, ref CallRef, observable bool) *ToolEvidence {
	if !observable {
		return nil
	}
	if ref.CallID == "" && ref.AttemptID == "" {
		return nil
	}
	m.mu.Lock()
	events := make([]Event, 0, 8)
	for _, event := range session.Events {
		if eventMatchesCallRef(event, ref) {
			events = append(events, event)
		}
	}
	m.mu.Unlock()
	return evidenceFromEvents(events)
}

func eventMatchesCallRef(event Event, ref CallRef) bool {
	if ref.AttemptID != "" {
		return event.AttemptID == ref.AttemptID
	}
	return event.CallID == ref.CallID
}

// evidenceFromEvents 是纯函数，便于单测与回放。
func evidenceFromEvents(events []Event) *ToolEvidence {
	evidence := &ToolEvidence{}
	seenCall := map[string]bool{}
	seenResult := map[string]bool{}
	toolStats := map[string]*toolStat{}

	type failureStreak struct {
		key   string
		tool  string
		count int
		max   int
	}
	var streak failureStreak

	for _, event := range events {
		name := strings.TrimSpace(event.ToolName)
		if name == "" {
			name = "工具"
		}
		id := strings.TrimSpace(event.ToolID)
		if id == "" {
			id = fmt.Sprintf("seq-%d", event.Seq)
		}
		switch event.Kind {
		case EventToolStart:
			if seenCall[id] {
				continue
			}
			seenCall[id] = true
			evidence.Calls++
			toolStatFor(toolStats, name).Count++
		case EventToolResult:
			if !seenCall[id] {
				seenCall[id] = true
				evidence.Calls++
				toolStatFor(toolStats, name).Count++
			}
			if seenResult[id] {
				continue
			}
			seenResult[id] = true
			switch strings.ToLower(strings.TrimSpace(event.ToolStatus)) {
			case "completed":
				evidence.OK++
				stat := toolStatFor(toolStats, name)
				stat.OK++
				if strings.TrimSpace(event.ToolOutput) != "" {
					// 非空锚：对宿主看到的有界预览取指纹，只落哈希不落原文；
					// 同一工具多次非空返回时首个胜出（确定性，回放可复核）。
					evidence.NonEmpty++
					stat.NonEmpty++
					if stat.Digest == "" {
						stat.Digest = toolOutputDigest(event.ToolOutput)
					}
				}
				streak = failureStreak{}
			case "failed":
				evidence.Failed++
				key := name + "\x00" + normalizeToolInput(event.ToolInput)
				if streak.key == key {
					streak.count++
				} else {
					streak = failureStreak{key: key, tool: name, count: 1}
				}
				if streak.count > streak.max {
					streak.max = streak.count
					streak.tool = name
				}
			default:
				// 未回报的结果既不算成功也不算失败，并重置连续失败：
				// 宿主看不到结论，不能把它当成同一失败链的一环。
				streak = failureStreak{}
			}
		}
	}
	if streak.max >= 2 {
		evidence.RepeatFailures = streak.max
		evidence.RepeatTool = streak.tool
	}
	evidence.Tools = topToolCounts(toolStats, 6)
	return evidence
}

// toolStat 是单个工具名的证据计数。Digest 取该工具首个非空成功输出的指纹。
type toolStat struct {
	Count    int
	OK       int
	NonEmpty int
	Digest   string
}

func toolStatFor(stats map[string]*toolStat, name string) *toolStat {
	stat, ok := stats[name]
	if !ok {
		stat = &toolStat{}
		stats[name] = stat
	}
	return stat
}

// toolOutputDigest 是宿主侧的非空锚：sha256 前 16 hex。作用是存档回放时的
// 同一性锚点与「有没有内容」的宿主可核事实，不是内容摘要，绝不进提示词。
func toolOutputDigest(output string) string {
	sum := sha256.Sum256([]byte(output))
	return hex.EncodeToString(sum[:])[:16]
}

// promptSnapshot 是对账线的输入侧锚（M1）：对实际发出的提示词取指纹与字数。
// 与 ToolEvidence 的输出侧非空锚合成完整对账线——席位「看到了什么」与
// 「跑出了什么」宿主都可控。只落哈希不落原文；指纹口径与 toolOutputDigest
// 一致（16 hex）。
func promptSnapshot(prompt string) (string, int) {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])[:16], len([]rune(prompt))
}

// seatPromptSnapshot 是 runOne 的唯一盖章入口：哈希必须锚 clampPrompt 之后
// 真正交给 adapter 的那段——截断前的原文哈希会在超长任务书上撒谎。
func seatPromptSnapshot(prompt string) (string, int) {
	return promptSnapshot(clampPrompt(prompt))
}

// evidenceMismatchNote 是共享账对账线（轻刀①②合流点）：席位陈词标注了
// 【工具核验】但本次调用的工具证据里没有任何非空返回时，给一行宿主对账
// 警示。只陈述事实不下判断；证据为 nil（通道不上报轨迹）时不置一词。
func evidenceMismatchNote(evidence *ToolEvidence, content string) string {
	if evidence == nil {
		return ""
	}
	if evidence.NonEmpty > 0 {
		return ""
	}
	if !strings.Contains(content, "【工具核验】") {
		return ""
	}
	return "宿主对账：本席标注【工具核验】，但本次调用没有任何非空工具返回——该标注不可对证，引用前请自行核验。"
}

func normalizeToolInput(input string) string {
	return strings.Join(strings.Fields(input), " ")
}

func topToolCounts(stats map[string]*toolStat, limit int) []ToolCount {
	if len(stats) == 0 {
		return nil
	}
	out := make([]ToolCount, 0, len(stats))
	for name, stat := range stats {
		if stat.Count <= 0 {
			continue
		}
		out = append(out, ToolCount{Name: name, Count: stat.Count, OK: stat.OK, NonEmpty: stat.NonEmpty, Digest: stat.Digest})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// toolEvidenceLine 是导出用的一行人话。nil 明确写「不上报」，不写「0 次」。
func toolEvidenceLine(evidence *ToolEvidence) string {
	if evidence == nil {
		return "该席不上报工具轨迹"
	}
	parts := []string{fmt.Sprintf("%d 次工具调用", evidence.Calls)}
	detail := make([]string, 0, 4)
	if evidence.OK > 0 {
		detail = append(detail, fmt.Sprintf("%d 成功", evidence.OK))
	}
	if evidence.Failed > 0 {
		detail = append(detail, fmt.Sprintf("%d 失败", evidence.Failed))
	}
	if evidence.NonEmpty > 0 {
		detail = append(detail, fmt.Sprintf("%d 次有非空返回", evidence.NonEmpty))
	}
	if unresolved := evidence.Calls - evidence.OK - evidence.Failed; unresolved > 0 {
		detail = append(detail, fmt.Sprintf("%d 未回报", unresolved))
	}
	if len(detail) > 0 {
		parts[0] += "（" + strings.Join(detail, " / ") + "）"
	}
	if evidence.RepeatFailures >= 2 {
		parts = append(parts, fmt.Sprintf("同一工具连续失败 %d 次：%s", evidence.RepeatFailures, evidence.RepeatTool))
	}
	return "宿主观察：" + strings.Join(parts, "；")
}

func hasCapability(status AgentStatus, name string) bool {
	for _, capability := range status.Capabilities {
		if strings.EqualFold(strings.TrimSpace(capability), name) {
			return true
		}
	}
	return false
}
