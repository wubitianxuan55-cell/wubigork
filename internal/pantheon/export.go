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
	if session.Request.ParentSessionID != "" {
		fmt.Fprintf(
			&b,
			"- 父会话：`%s`（续议承接以简报为准，不继承全文）\n",
			session.Request.ParentSessionID,
		)
	}
	if session.Request.UntilConsensus {
		fmt.Fprintf(
			&b,
			"- 讨论模式：不限轮次，至全员一致（安全上限 %d 轮）\n- 实际轮次：%d\n",
			session.Request.Rounds,
			session.CurrentRound,
		)
	} else {
		fmt.Fprintf(&b, "- 讨论轮数：%d\n", session.Request.Rounds)
	}
	// 2026-10-31 裁决第二刀：路径标签入档；token/费用 harness 未暴露，
	// 缺失必须明示不可比，避免把耗时差异误读成成本结论。
	fmt.Fprintf(
		&b,
		"- 议事路径：%s\n- 对照注记：费用未采集（harness 未暴露 token/费用事件）；耗时仅可同席自比\n\n",
		ProtocolLabel(session.Request.Protocol),
	)
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
	if len(session.Request.Attachments) > 0 {
		b.WriteString("### 图片附件\n\n")
		for _, attachment := range session.Request.Attachments {
			fmt.Fprintf(&b, "- %s（%s，%dKB）\n", attachment.Name, attachment.MIME, (attachment.Size+1023)/1024)
		}
		b.WriteString("\n")
	}
	writeInputManifest(&b, session)

	b.WriteString("## 参与者\n\n")
	for _, p := range session.Participants {
		fmt.Fprintf(&b, "- %s（`%s`）：%s", p.Name, p.AgentID, p.Status)
		if p.DurationMs > 0 {
			fmt.Fprintf(&b, "，%dms", p.DurationMs)
		}
		// 只读机制逐席入档，不写笼统的「已只读」（2026-10-31 裁决第三刀）。
		if runMode := seatRunMode(session, p.AgentID); runMode != "" {
			fmt.Fprintf(&b, "；只读经 %s", runMode)
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
		if session.SignOff.VerdictSHA256 != "" {
			fmt.Fprintf(&b, "- 绑定小结版本：`sha256:%s`\n", session.SignOff.VerdictSHA256)
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

	writeStopSignals(&b, session)
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

// RenderFinalDocument produces the compact deliverable owned by the council
// chair. The full audit trail remains in ExportMarkdown/session.json; this file
// is the document users can read without replaying the discussion timeline.
func RenderFinalDocument(session *Session) string {
	if session == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(session.Title)
	b.WriteString("\n\n")
	b.WriteString("> 众神殿最终讨论文档\n\n")
	b.WriteString("## 基本信息\n\n")
	fmt.Fprintf(&b, "- 议题：%s\n", session.Request.Topic)
	fmt.Fprintf(&b, "- 完成时间：%s\n", formatMillis(session.UpdatedAt))
	fmt.Fprintf(&b, "- 讨论轮次：%d\n", session.CurrentRound)
	fmt.Fprintf(&b, "- 议事路径：%s（费用未采集，耗时仅可同席自比）\n", ProtocolLabel(session.Request.Protocol))
	if session.Request.Workspace != "" {
		fmt.Fprintf(&b, "- 工作空间：`%s`\n", session.Request.Workspace)
	}
	if session.Request.ParentSessionID != "" {
		fmt.Fprintf(
			&b,
			"- 父会话：`%s`（续议承接以简报为准，不继承全文）\n",
			session.Request.ParentSessionID,
		)
	}
	if session.Verdict != nil {
		fmt.Fprintf(&b, "- 主持与汇总：%s\n", session.Verdict.Chairman)
	}
	b.WriteString("\n")

	if session.Verdict != nil {
		b.WriteString("## 最终建议\n\n")
		b.WriteString(session.Verdict.Recommendation)
		b.WriteString("\n\n")
		if session.Verdict.Partial {
			b.WriteString("> 注意：部分参与者未返回有效观点，本结论需结合原始发言复核。\n\n")
		}
		if session.Verdict.Summary != "" {
			b.WriteString("## 讨论结论\n\n")
			b.WriteString(session.Verdict.Summary)
			b.WriteString("\n\n")
		}
		writeList(&b, "适用条件", session.Verdict.Conditions)
		writeList(&b, "保留分歧", session.Verdict.Disagreements)
		writeList(&b, "风险与开放问题", session.Verdict.Risks)
		writeList(&b, "证据依据", session.Verdict.EvidenceNotes)
	}

	b.WriteString("## 参与者\n\n")
	for _, participant := range session.Participants {
		fmt.Fprintf(&b, "- %s（`%s`）：%s\n", participant.Name, participant.AgentID, participant.Status)
	}
	b.WriteString("\n## 各席观点\n\n")
	for _, contribution := range session.Contributions {
		if contribution.Phase == PhaseSynthesis {
			continue
		}
		if contribution.DurationMs > 0 {
			fmt.Fprintf(
				&b,
				"### 第 %d 轮 · %s（耗时 %.1fs）\n\n",
				contribution.Round, contribution.AgentName, float64(contribution.DurationMs)/1000,
			)
		} else {
			fmt.Fprintf(&b, "### 第 %d 轮 · %s\n\n", contribution.Round, contribution.AgentName)
		}
		if contribution.RunMode != "" {
			fmt.Fprintf(&b, "_运行方式：%s_\n\n", contribution.RunMode)
		}
		b.WriteString(contribution.Content)
		if contribution.Consensus != "" {
			// The protocol line was stripped from the body; write the signal
			// back so the deliverable carries the stop evidence.
			fmt.Fprintf(&b, "\n\n> 共识状态：%s（协议行提取）\n", consensusLabel(contribution.Consensus))
		}
		b.WriteString("\n\n")
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
		if session.SignOff.VerdictSHA256 != "" {
			fmt.Fprintf(&b, "- 绑定小结版本：`sha256:%s`\n", session.SignOff.VerdictSHA256)
		}
		b.WriteString("\n")
	}
	if len(session.Warnings) > 0 {
		b.WriteString("## 运行警示\n\n")
		for _, warning := range session.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
		b.WriteString("\n")
	}
	b.WriteString("---\n\n")
	b.WriteString("> 本文档由主持人根据各席公开观点整理，不等同于用户确认、专业意见或外部事实保证。\n")
	return b.String()
}

// writeStopSignals makes the stop condition auditable: why the council stopped
// and what each seat actually signalled in the final round — including seats
// that went absent or never declared a signal. Without this section, consensus
// lines stripped from the body leave no human-readable trace of the stop
// decision (2026-10-31 裁决第一刀).
func writeStopSignals(b *strings.Builder, session *Session) {
	if !session.Request.UntilConsensus {
		return
	}
	consensusRound := 0
	for _, event := range session.Events {
		if event.Kind == EventConsensus && event.Round > consensusRound {
			consensusRound = event.Round
		}
	}
	b.WriteString("## 停止信号\n\n")
	if consensusRound > 0 {
		fmt.Fprintf(b, "- 结果：第 %d 轮达成全员一致，提前停止\n", consensusRound)
	} else {
		fmt.Fprintf(b, "- 结果：达到安全上限 %d 轮，未达成全员一致\n", session.Request.Rounds)
	}
	lastRound := session.CurrentRound
	if lastRound <= 0 {
		for _, c := range session.Contributions {
			if c.Round > lastRound {
				lastRound = c.Round
			}
		}
	}
	signals := make(map[string]string)
	for _, c := range session.Contributions {
		if c.Round == lastRound {
			signals[c.AgentID] = c.Consensus
		}
	}
	if len(signals) > 0 {
		entries := make([]string, 0, len(session.Participants))
		for _, p := range session.Participants {
			signal, ok := signals[p.AgentID]
			switch {
			case !ok:
				entries = append(entries, p.AgentID+"=缺席")
			case signal == "":
				entries = append(entries, p.AgentID+"=未声明")
			default:
				entries = append(entries, p.AgentID+"="+consensusLabel(signal))
			}
		}
		if len(entries) > 0 {
			fmt.Fprintf(b, "- 末轮（第 %d 轮）各席：%s\n", lastRound, strings.Join(entries, "；"))
		}
	}
	b.WriteString("\n")
}

// writeInputManifest prints the start-time input snapshot: brief evidence and
// attachment files with their sha256, so re-runs and path comparisons can
// detect material drift (2026-10-31 裁决第三刀).
func writeInputManifest(b *strings.Builder, session *Session) {
	if len(session.InputManifest) == 0 {
		return
	}
	b.WriteString("### 材料快照（开始时哈希）\n\n")
	for _, digest := range session.InputManifest {
		kind := "材料"
		if digest.Kind == "attachment" {
			kind = "附件"
		}
		if digest.Error != "" {
			fmt.Fprintf(b, "- %s · %s：%s\n", kind, digest.Name, digest.Error)
			continue
		}
		fmt.Fprintf(b, "- %s · %s：`sha256:%s`（%d 字节）", kind, digest.Name, digest.SHA256, digest.Size)
		if digest.Path != "" {
			fmt.Fprintf(b, " `%s`", digest.Path)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// seatRunMode finds the read-only mechanism recorded for a seat's calls.
func seatRunMode(session *Session, agentID string) string {
	for _, contribution := range session.Contributions {
		if contribution.AgentID == agentID && contribution.RunMode != "" {
			return contribution.RunMode
		}
	}
	return ""
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
