package pantheon

import (
	"fmt"
	"strings"
)

// 第二次蒸馏 E2：主席派工的投递回执。
//
// 它是纯派生，不落盘、不新增表：把 session.Contributions/Events 当作唯一
// 真相，按存档顺序重算「这一条派工之后，被派席有没有发言 / 失败 / 跳过 /
// 缺席」。因此重试、跳过、恢复、重启之后，回执与存档始终一致。
//
// 最重要的语义边界：spoke 只证明「下一轮发过言」，不证明「完成了指令」。
// 是否落实由主席读贡献判断；宿主不做语义判断，也不给席位打分。
const (
	ReceiptSpoke       = "spoke"
	ReceiptSilent      = "silent"
	ReceiptFailed      = "failed"
	ReceiptSkipped     = "skipped"
	ReceiptUnavailable = "unavailable"
	ReceiptPending     = "pending"
)

// ChairAssignmentReceipt 是一条派工的投递回执。
type ChairAssignmentReceipt struct {
	Round          int    `json:"round"`
	AgentID        string `json:"agentId"`
	AgentName      string `json:"agentName"`
	Instruction    string `json:"instruction"`
	Status         string `json:"status"`
	ContributionID string `json:"contributionId,omitempty"`
	Detail         string `json:"detail,omitempty"`
}

// chairAssignmentReceiptsForRound 返回某轮主席小结派工的回执。
func chairAssignmentReceiptsForRound(session *Session, round int) []ChairAssignmentReceipt {
	all := chairAssignmentReceipts(session)
	out := make([]ChairAssignmentReceipt, 0, len(all))
	for _, receipt := range all {
		if receipt.Round == round {
			out = append(out, receipt)
		}
	}
	return out
}

// chairAssignmentReceipts 从存档派生全部派工回执。调用方若可能与会话写并发，
// 应先持 m.mu；本函数本身不加锁，便于导出与单测。
func chairAssignmentReceipts(session *Session) []ChairAssignmentReceipt {
	if session == nil {
		return nil
	}
	names := make(map[string]string, len(session.Participants))
	statuses := make(map[string]string, len(session.Participants))
	for _, participant := range session.Participants {
		names[participant.AgentID] = participant.Name
		statuses[participant.AgentID] = participant.Status
	}

	var receipts []ChairAssignmentReceipt
	for index, contribution := range session.Contributions {
		if contribution.Phase != PhaseModeration || len(contribution.ChairAssignments) == 0 {
			continue
		}
		moderationSeq := moderationEventSeq(session, contribution)
		for _, assignment := range contribution.ChairAssignments {
			receipt := ChairAssignmentReceipt{
				Round:       contribution.Round,
				AgentID:     assignment.AgentID,
				AgentName:   names[assignment.AgentID],
				Instruction: assignment.Instruction,
			}
			if receipt.AgentName == "" {
				receipt.AgentName = assignment.AgentID
			}
			if spoke := firstContributionAfter(session.Contributions, index, assignment.AgentID); spoke != nil {
				receipt.Status = ReceiptSpoke
				receipt.ContributionID = spoke.ID
				receipt.Detail = "下一轮已发言（未证明完成指令）"
				receipts = append(receipts, receipt)
				continue
			}
			if status, detail := seatOutcomeAfter(session, contribution, moderationSeq, assignment.AgentID); status != "" {
				receipt.Status = status
				receipt.Detail = detail
				receipts = append(receipts, receipt)
				continue
			}
			if statuses[assignment.AgentID] == SeatUnavailable {
				receipt.Status = ReceiptUnavailable
				receipt.Detail = "席位不可用"
				receipts = append(receipts, receipt)
				continue
			}
			if contribution.Round < session.CurrentRound {
				receipt.Status = ReceiptSilent
				receipt.Detail = "该轮未发言，也未记录失败/跳过"
			} else {
				receipt.Status = ReceiptPending
				if session.Status == StatusCompleted || session.Status == StatusFailed || session.Status == StatusCancelled {
					receipt.Detail = "议事在下一轮前结束"
				} else {
					receipt.Detail = "下一轮尚未开始"
				}
			}
			receipts = append(receipts, receipt)
		}
	}
	return receipts
}

func firstContributionAfter(contributions []Contribution, moderationIndex int, agentID string) *Contribution {
	for index := moderationIndex + 1; index < len(contributions); index++ {
		if contributions[index].AgentID == agentID {
			return &contributions[index]
		}
	}
	return nil
}

// seatOutcomeAfter 找派工之后该席的失败/跳过事件；错误优先于跳过，因为
// 「先跳过、后重试仍失败」时读者最需要看到的是失败原因。
func seatOutcomeAfter(session *Session, moderation Contribution, moderationSeq int64, agentID string) (string, string) {
	var failed, skipped *Event
	for index := range session.Events {
		event := &session.Events[index]
		if event.AgentID != agentID {
			continue
		}
		if moderationSeq > 0 && event.Seq <= moderationSeq {
			continue
		}
		if moderationSeq <= 0 && event.At < moderation.CreatedAt {
			continue
		}
		switch event.Kind {
		case EventAgentError:
			failed = event
		case EventAgentSkipped:
			skipped = event
		}
	}
	if failed != nil {
		return ReceiptFailed, shortEvidence(failed.Text)
	}
	if skipped != nil {
		return ReceiptSkipped, "该席在派工后的轮次被跳过"
	}
	return "", ""
}

func moderationEventSeq(session *Session, contribution Contribution) int64 {
	var seq int64
	for _, event := range session.Events {
		if event.Kind != EventAgentOutput {
			continue
		}
		if contribution.CallID != "" && event.CallID != contribution.CallID {
			continue
		}
		if contribution.AttemptID != "" && event.AttemptID != contribution.AttemptID {
			continue
		}
		if event.Seq > seq {
			seq = event.Seq
		}
	}
	return seq
}

// receiptStatusLabel 是回执状态的人话；UI/导出/提示词三处必须用同一措辞，
// 防止把 spoke 读成 completed。
func receiptStatusLabel(status string) string {
	switch status {
	case ReceiptSpoke:
		return "已发言（未证明完成）"
	case ReceiptSilent:
		return "未发言"
	case ReceiptFailed:
		return "失败"
	case ReceiptSkipped:
		return "已跳过"
	case ReceiptUnavailable:
		return "缺席"
	case ReceiptPending:
		return "未到轮"
	default:
		return status
	}
}

func formatChairReceipts(receipts []ChairAssignmentReceipt) string {
	if len(receipts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n上一轮派工回执（只证明该席是否在下一轮发言，不证明完成）：\n")
	for _, receipt := range receipts {
		fmt.Fprintf(&b, "- %s：%s", receipt.AgentName, receiptStatusLabel(receipt.Status))
		if receipt.ContributionID != "" {
			fmt.Fprintf(&b, "（贡献 ID %s）", receipt.ContributionID)
		}
		if receipt.Detail != "" && receipt.Status != ReceiptSpoke {
			fmt.Fprintf(&b, "——%s", receipt.Detail)
		}
		b.WriteString("\n")
	}
	b.WriteString("未落实的派工：重新指派、改成收口动议，或写进风险；不要把「已发言」写成「已完成」。\n")
	return b.String()
}
