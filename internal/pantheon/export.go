package pantheon

import (
	"fmt"
	"strings"
	"time"
)

func ExportMarkdown(session *Session) string {
	if session == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("# 众神殿议事录 · ")
	b.WriteString(session.Title)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "- 议事 ID：`%s`\n", session.ID)
	fmt.Fprintf(&b, "- 状态：%s\n", session.Status)
	fmt.Fprintf(&b, "- 创建：%s\n", formatMillis(session.CreatedAt))
	fmt.Fprintf(&b, "- 更新：%s\n", formatMillis(session.UpdatedAt))
	if session.Request.UntilConsensus {
		fmt.Fprintf(
			&b,
			"- 讨论模式：不限轮次，至全员一致（安全上限 %d 轮）\n- 实际轮次：%d\n\n",
			session.Request.Rounds,
			session.CurrentRound,
		)
	} else {
		fmt.Fprintf(&b, "- 讨论轮数：%d\n\n", session.Request.Rounds)
	}
	if session.Request.Workspace != "" {
		b.WriteString("## 议事工作空间\n\n")
		fmt.Fprintf(&b, "`%s`\n\n", session.Request.Workspace)
	}

	b.WriteString("## 议题\n\n")
	b.WriteString(session.Request.Topic)
	b.WriteString("\n\n")
	if session.Request.Context != "" {
		b.WriteString("### 背景与约束\n\n")
		b.WriteString(session.Request.Context)
		b.WriteString("\n\n")
	}
	if session.Request.SuccessCriteria != "" {
		b.WriteString("### 成功标准\n\n")
		b.WriteString(session.Request.SuccessCriteria)
		b.WriteString("\n\n")
	}

	b.WriteString("## 参与者\n\n")
	for _, p := range session.Participants {
		fmt.Fprintf(&b, "- %s（`%s`）：%s", p.Name, p.AgentID, p.Status)
		if p.DurationMs > 0 {
			fmt.Fprintf(&b, "，%dms", p.DurationMs)
		}
		if p.Error != "" {
			fmt.Fprintf(&b, "；%s", p.Error)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if session.Verdict != nil {
		b.WriteString("## 讨论小结\n\n")
		if session.Verdict.Partial {
			b.WriteString("> ⚠ 本次小结不完整，必须结合原始发言人工复核。\n\n")
		}
		b.WriteString("### 综合建议\n\n")
		b.WriteString(session.Verdict.Recommendation)
		b.WriteString("\n\n")
		if session.Verdict.Summary != "" {
			b.WriteString("### 讨论概况\n\n")
			b.WriteString(session.Verdict.Summary)
			b.WriteString("\n\n")
		}
		writeList(&b, "适用条件", session.Verdict.Conditions)
		writeList(&b, "保留分歧", session.Verdict.Disagreements)
		writeList(&b, "开放问题与风险", session.Verdict.Risks)
		writeList(&b, "证据注记", session.Verdict.EvidenceNotes)
		fmt.Fprintf(&b, "- 主持与汇总：%s\n\n", session.Verdict.Chairman)
	}
	if session.SignOff != nil {
		labels := map[string]string{
			"accepted": "确认小结", "needs_revision": "需继续讨论", "rejected": "暂不采纳",
		}
		b.WriteString("## 用户确认\n\n")
		fmt.Fprintf(&b, "- 决定：%s\n", labels[session.SignOff.Decision])
		fmt.Fprintf(&b, "- 时间：%s\n", formatMillis(session.SignOff.At))
		if session.SignOff.Note != "" {
			fmt.Fprintf(&b, "- 备注：%s\n", session.SignOff.Note)
		}
		b.WriteString("\n")
	}

	b.WriteString("## 讨论时间线\n\n")
	for _, event := range session.Events {
		actor := ""
		if event.AgentID != "" {
			actor = " · " + event.AgentID
		}
		fmt.Fprintf(&b, "### %02d · %s%s\n\n", event.Seq, event.Phase, actor)
		fmt.Fprintf(&b, "%s\n\n", formatMillis(event.At))
		if event.Error != "" {
			fmt.Fprintf(&b, "**错误：** %s\n\n", event.Error)
		}
		if event.Text != "" {
			b.WriteString(event.Text)
			b.WriteString("\n\n")
		}
	}

	if len(session.Warnings) > 0 {
		b.WriteString("## 警示\n\n")
		for _, warning := range session.Warnings {
			b.WriteString("- ")
			b.WriteString(warning)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n---\n\n")
	b.WriteString("> 本纪要保留多轮讨论与分歧，不等同于用户确认或外部事实保证。\n")
	return b.String()
}

func writeList(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("### ")
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func formatMillis(ms int64) string {
	if ms <= 0 {
		return "未知"
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
}
