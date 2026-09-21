package agent

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// 压缩救援阶梯（Distilled from Reasonix fold_ladder.go / truncate.go /
// compact_safe_prefix.go）：一次压缩事务的降档序列——
//  1. cache-aligned 全量重放摘要（默认，命中热缓存且保真）；
//  2. slim 有界转录档：折叠区本身大到来不及重放时（正是溢出场景），
//     逐消息截头送摘要——把「摘要请求自己溢出→机械裸计数」的命运换成真摘要；
//  3. 投影截断终级：摘要无法成形（压缩无进展）时，先抹大工具结果、
//     再整单元丢最老，装 marker 消息——有损，但让回合能继续。
//
// canonical 侧纪律与 PruneStaleToolResults 相同：改动前先归档。
const (
	// summaryPlanReserveTokens 为摘要请求的输出+协议预留；小窗口按 5% 边际
	// 放大（上游 summaryPlanMarginRatio 同值——固定预留对大窗口太薄）。
	summaryPlanReserveTokens = 2048
	summaryPlanMarginRatio   = 0.05
	// slimPerMsgFloorChars 单消息截头下限：低于此摘要只见碎片，不如不送。
	slimPerMsgFloorChars = 200
	// slimCallArgCap 工具调用参数在 slim 档的保留上限（write_file 全文参数
	// 是折叠区膨胀主源，摘要不需要全文）。
	slimCallArgCap = 160
)

// slimmedMarker 标记一条消息在 slim 档被截短（含丢弃字符数）。
const slimmedMarker = " […slimmed: %d chars dropped to fit the summarizer]"

// summaryPromptBudget 返回单次摘要请求的提示词 token 预算；无窗口配置返回 0
// （不设阶梯——手动 /compact 与无窗 provider 维持现状）。
func (a *AgentRunner) summaryPromptBudget() int {
	w := a.compaction.Window
	if w <= 0 {
		return 0
	}
	reserve := summaryPlanReserveTokens
	if m := int(float64(w) * summaryPlanMarginRatio); m > reserve {
		reserve = m
	}
	return w - reserve
}

// slimFoldForSummary 把折叠区裁到预算内：按体积比例逐消息截头（下限
// slimPerMsgFloorChars），工具参数截到 slimCallArgCap。返回 nil 表示折叠区
// 本身无需裁剪。system 指令与末条指令消息不在折叠区内，由调用方原样保留。
func slimFoldForSummary(a *AgentRunner, fold []provider.Message, budgetTokens int) []provider.Message {
	if budgetTokens <= 0 || len(fold) == 0 {
		return nil
	}
	total := estimateMessagesTokens(fold)
	if total <= budgetTokens {
		return nil
	}
	targetChars := charsOfMessages(fold) * budgetTokens / total
	perMsg := targetChars / len(fold)
	if perMsg < slimPerMsgFloorChars {
		perMsg = slimPerMsgFloorChars
	}
	out := make([]provider.Message, len(fold))
	for i, m := range fold {
		out[i] = slimMessage(m, perMsg)
	}
	return out
}

// slimMessage 截短单条消息：正文保头 perMsg 字符+marker；工具调用保留
// ID/名称、参数截到 slimCallArgCap。骨架（角色/配对键）不动，摘要仍能
// 分辨对话结构。
func slimMessage(m provider.Message, perMsg int) provider.Message {
	out := m
	if c := len(m.Content); c > perMsg {
		out.Content = m.Content[:perMsg] + fmt.Sprintf(slimmedMarker, c-perMsg)
	}
	if len(m.ToolCalls) > 0 {
		calls := make([]provider.ToolCall, len(m.ToolCalls))
		copy(calls, m.ToolCalls)
		for i, tc := range calls {
			if a := len(tc.Arguments); a > slimCallArgCap {
				calls[i].Arguments = tc.Arguments[:slimCallArgCap] + fmt.Sprintf(slimmedMarker, a-slimCallArgCap)
			}
		}
		out.ToolCalls = calls
	}
	return out
}

// ─── 刀1b: 投影截断终级（Distilled from Reasonix truncate.go
// rescueByTruncation）───────────────────────────────────────────────────────

// truncatedHistoryMarker 记录截断 Rescue 丢弃的整单元消息数。
const truncatedHistoryMarker = "[earlier conversation truncated to fit the context window: %d messages removed]"

// truncateProtectShare 收窄截断救援的保护区：摘要规划给的尾预算（≥2000
// token）在「窗口太小压不完」的困境里会把可救内容全部划成圣地——救援
// 只保护 target/4（上游 truncateProtectShare=4），minRecentKeep 条数的
// 下限由 tailStart 自持。
const truncateProtectShare = 4

// truncateRescue 把会话估算压到 target（= 压缩触发水位）以下：保护区之外，
// 先抹大工具结果（老→新；KeepErrors/KeepProtected 豁免），仍超再整单元
// （assistant+其工具结果 / 独立 user 消息）丢最老，并在 head 后装 marker。
// 改动前归档；任何变更都会推进 RewriteVersion。返回是否做了变更。
func (a *AgentRunner) truncateRescue() bool {
	if a.compaction.Window <= 0 {
		return false
	}
	target := int(float64(a.compaction.Window) * a.ratio())
	msgs := a.session.Messages
	head := a.pinnedPrefixLen(msgs)
	start := tailStart(msgs, head, target/truncateProtectShare, a.tokPerChar(), minRecentKeep)
	if start <= head {
		return false
	}
	est := func(ms []provider.Message) int {
		return int(float64(charsOfMessages(ms)) * a.tokPerChar())
	}
	if est(msgs) < target {
		return false
	}
	// 先归档再动手（与 prune 同纪律）。
	if a.compaction.ArchiveDir != "" {
		if _, err := archiveMessages(a.compaction.ArchiveDir, msgs[head:start]); err != nil {
			a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
				Text: "truncation rescue skipped: archive failed: " + err.Error()})
			return false
		}
	}
	next := append([]provider.Message(nil), msgs...)
	elided, dropped := 0, 0

	// Phase A: 抹大工具结果，老→新，够到 target 即停。
	for i := head; i < start && est(next) >= target; i++ {
		m := next[i]
		if m.Role != provider.RoleTool || len(m.Content) < minPruneBytes || strings.HasPrefix(m.Content, prunedMarker) {
			continue
		}
		if a.keepPolicy&KeepErrors != 0 && isErrorMessage(m) {
			continue
		}
		if a.keepPolicy&KeepProtected != 0 && isProtectedToolResult(m) {
			continue
		}
		placeholder := fmt.Sprintf("%s%s, %d bytes dropped by truncation rescue]", prunedMarker, m.Name, len(m.Content))
		next[i] = provider.Message{Role: provider.RoleTool, Content: placeholder, ToolCallID: m.ToolCallID, Name: m.Name}
		elided++
	}

	// Phase B: 仍超则整单元丢最老（含摘要/策略保留的单元跳过，保配对完整）。
	if est(next) >= target {
		var kept []provider.Message
		var out []provider.Message
		out = append(out, next[:head]...)
		i := head
		for i < start {
			unitEnd := i + 1
			if next[i].Role == provider.RoleAssistant && len(next[i].ToolCalls) > 0 {
				ids := toolCallIDs(next[i])
				for unitEnd < start && next[unitEnd].Role == provider.RoleTool && ids[next[unitEnd].ToolCallID] {
					unitEnd++
				}
			}
			unit := next[i:unitEnd]
			protect := false
			for _, m := range unit {
				if isCompactionSummary(m) || (a.keepPolicy&KeepErrors != 0 && isErrorMessage(m)) ||
					(a.keepPolicy&KeepProtected != 0 && isProtectedToolResult(m)) {
					protect = true
					break
				}
			}
			if protect || est(next) < target {
				kept = append(kept, unit...)
			} else {
				dropped += len(unit)
			}
			i = unitEnd
		}
		if dropped > 0 {
			out = append(out, provider.Message{
				Role:    provider.RoleUser,
				Content: fmt.Sprintf(truncatedHistoryMarker, dropped),
			})
			out = append(out, kept...)
			out = append(out, next[start:]...)
			next = out
		}
	}

	if elided == 0 && dropped == 0 {
		return false
	}
	a.session.Replace(next)
	a.session.IncrementRewrite()
	a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: fmt.Sprintf(
		"truncation rescue applied: %d tool results elided, %d messages dropped (lossy last resort)",
		elided, dropped)})
	return true
}
