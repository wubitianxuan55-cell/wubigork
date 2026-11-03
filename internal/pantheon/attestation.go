package pantheon

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	attestationPositionMarker  = regexp.MustCompile(`(?im)^\s*(?:立场|复核立场|position)\s*[:：]\s*(.+)$`)
	attestationReasonMarker    = regexp.MustCompile(`(?im)^\s*(?:理由|关键理由|reason)\s*[:：]\s*(.*)$`)
	attestationConditionMarker = regexp.MustCompile(`(?im)^\s*(?:条件|适用条件|conditions?)\s*[:：]\s*(.*)$`)
	attestationIssuesMarker    = regexp.MustCompile(`(?im)^\s*(?:相关分歧|分歧 ID|issue ids?)\s*[:：]\s*(.*)$`)
)

func normalizeAttestationPosition(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.Trim(value, "`'\"。.;；:：()（）[]【】")
	switch value {
	case "支持", "同意", "support", "supported", "sign":
		return AttestationSupport
	case "有条件支持", "条件支持", "conditional", "conditionally support":
		return AttestationConditional
	case "反对", "不同意", "oppose", "opposed", "reject":
		return AttestationOppose
	default:
		return AttestationUnknown
	}
}

func parseAttestation(raw, agentID, agentName, verdictHash string) Attestation {
	result := Attestation{
		AgentID:       agentID,
		AgentName:     agentName,
		Position:      AttestationUnknown,
		VerdictSHA256: verdictHash,
		CreatedAt:     time.Now().UnixMilli(),
	}
	for _, line := range strings.Split(raw, "\n") {
		if match := attestationPositionMarker.FindStringSubmatch(line); len(match) > 1 {
			result.Position = normalizeAttestationPosition(match[1])
			continue
		}
		if match := attestationReasonMarker.FindStringSubmatch(line); len(match) > 1 {
			result.Reason = trimRunes(strings.TrimSpace(match[1]), 1200)
			continue
		}
		if match := attestationConditionMarker.FindStringSubmatch(line); len(match) > 1 {
			result.Conditions = splitAttestationList(match[1])
			continue
		}
		if match := attestationIssuesMarker.FindStringSubmatch(line); len(match) > 1 {
			result.IssueIDs = splitAttestationList(match[1])
		}
	}
	if result.Position == AttestationUnknown {
		result.Error = "复核输出缺少可识别的立场（支持/有条件支持/反对）"
	}
	if result.Reason == "" && result.Error == "" {
		result.Error = "复核输出缺少关键理由"
	}
	return result
}

func splitAttestationList(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '；' || r == ';' || r == '，' || r == ',' || r == '、' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" || value == "无" || strings.EqualFold(value, "none") {
			continue
		}
		out = append(out, trimRunes(value, 300))
	}
	return out
}

func attestationPrompt(verdict *Verdict, issues []Issue, gaps []string) string {
	payload, _ := json.MarshalIndent(verdict, "", "  ")
	var issueBlock strings.Builder
	for _, issue := range issues {
		status := issue.Status
		fmt.Fprintf(&issueBlock, "- %s [%s] %s；依据=%s；翻转条件=%s；核验=%s\n",
			issue.ID, status, issue.Summary, firstNonEmpty(issue.Evidence, "未提供"),
			firstNonEmpty(issue.FlipCondition, "未提供"), firstNonEmpty(issue.Verification, "未提供"))
	}
	if issueBlock.Len() == 0 {
		issueBlock.WriteString("- 无未决台账\n")
	}
	gapText := "无"
	if len(gaps) > 0 {
		gapText = strings.Join(gaps, "；")
	}
	return fmt.Sprintf(`你是众神殿的独立复核席。现在不继续辩论，也不替主持人改写结论；只检查“如果此刻由用户签注，你是否会签”。

【待签注小结】
%s

【缺口席位】
%s

【分歧台账】
%s

不得参考其他复核席的立场。只输出四行，不要 Markdown 围栏：
立场：支持 或 有条件支持 或 反对
理由：一条最关键的理由
条件：必须写入结论的条件；没有写「无」，多项用「；」分隔
相关分歧：台账 ID，多项用「、」分隔；没有写「无」

立场判定：
- 支持：按当前版本可直接签注；
- 有条件支持：方向可以，但条件必须补进结论；
- 反对：结论、证据或适用条件存在必须阻断签注的问题。`, string(payload), gapText, issueBlock.String())
}

func revisionPrompt(verdict *Verdict, attestations []Attestation, issues []Issue, gaps []string) string {
	payload, _ := json.MarshalIndent(verdict, "", "  ")
	var reviews strings.Builder
	for _, attestation := range attestations {
		fmt.Fprintf(&reviews, "- %s：%s；理由=%s；条件=%s；分歧=%s\n",
			attestation.AgentName,
			attestationPositionLabel(attestation.Position),
			firstNonEmpty(attestation.Reason, attestation.Error, "未提供"),
			firstNonEmpty(strings.Join(attestation.Conditions, "；"), "无"),
			firstNonEmpty(strings.Join(attestation.IssueIDs, "、"), "无"))
	}
	var issueBlock strings.Builder
	for _, issue := range issues {
		fmt.Fprintf(&issueBlock, "- %s [%s] %s\n", issue.ID, issue.Status, issue.Summary)
	}
	return fmt.Sprintf(`你是众神殿汇总席。独立复核已出现反对意见；请只修订一次，保留原结论中仍成立的部分，并吸收复核事实。

【原小结】
%s

【独立复核】
%s

【分歧台账】
%s

【缺口】
%s

只输出一个 JSON 对象，不要 Markdown 围栏。沿用原小结字段，并额外填写：
"revisionReason": "哪些反对意见被吸收、哪些仍保留及理由（≤200字）"

不得删除仍在台账中的异议，不得把反对票改写成共识，不得声称已经执行任何行动。`,
		string(payload), reviews.String(), issueBlock.String(), firstNonEmpty(strings.Join(gaps, "；"), "无"))
}

func attestationPositionLabel(position string) string {
	switch position {
	case AttestationSupport:
		return "支持"
	case AttestationConditional:
		return "有条件支持"
	case AttestationOppose:
		return "反对"
	default:
		return "未识别"
	}
}

func (m *Manager) runIndependentAttestations(
	ctx context.Context,
	session *Session,
	statuses []AgentStatus,
	verdict *Verdict,
	issues []Issue,
	gaps []string,
) []Attestation {
	if verdict == nil || len(statuses) == 0 {
		return nil
	}
	hash := verdictFingerprint(verdict)
	results := make([]Attestation, len(statuses))
	var wg sync.WaitGroup
	for i, status := range statuses {
		results[i] = Attestation{
			AgentID: status.ID, AgentName: status.Name,
			Position: AttestationUnknown, VerdictSHA256: hash,
			CreatedAt: time.Now().UnixMilli(),
		}
		wg.Add(1)
		go func(index int, seat AgentStatus) {
			defer wg.Done()
			out, _, _, err := m.runAgentWithRetry(
				ctx, session, seat, attestationPrompt(verdict, issues, gaps),
				PhaseAttestation, 0,
			)
			if err != nil {
				results[index].Error = err.Error()
				return
			}
			results[index] = parseAttestation(out, seat.ID, seat.Name, hash)
		}(i, status)
	}
	wg.Wait()
	for _, result := range results {
		m.recordAttestationEvent(session, result)
	}
	return results
}

func (m *Manager) recordAttestationEvent(session *Session, attestation Attestation) {
	text := fmt.Sprintf("%s 独立复核：%s",
		attestation.AgentName, attestationPositionLabel(attestation.Position))
	if attestation.Reason != "" {
		text += " —— " + attestation.Reason
	} else if attestation.Error != "" {
		text += " —— " + attestation.Error
	}
	m.mu.Lock()
	event := m.appendEventLocked(session, Event{
		Phase: PhaseAttestation, AgentID: attestation.AgentID,
		Kind: EventAttestation, Text: text, At: time.Now().UnixMilli(),
	})
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) reviseAfterAttestations(
	ctx context.Context,
	session *Session,
	chairman AgentStatus,
	verdict *Verdict,
	attestations []Attestation,
	issues []Issue,
	gaps []string,
) (*Verdict, []VerdictRevision) {
	opposed := make([]Attestation, 0, len(attestations))
	reasons := make([]string, 0, len(attestations))
	for _, attestation := range attestations {
		if attestation.Position != AttestationOppose {
			continue
		}
		opposed = append(opposed, attestation)
		reasons = append(reasons, fmt.Sprintf("%s：%s",
			attestation.AgentName, firstNonEmpty(attestation.Reason, attestation.Error, "反对")))
	}
	if len(opposed) == 0 {
		return verdict, nil
	}
	original := *verdict
	originalSHA := verdictFingerprint(verdict)
	out, _, _, err := m.runAgentWithRetry(
		ctx, session, chairman,
		revisionPrompt(verdict, attestations, issues, gaps),
		PhaseAttestation, 0,
	)
	if err != nil || !strings.Contains(out, "{") {
		reason := "独立复核存在反对，但主席修订失败：" + firstNonEmpty(errString(err), "未返回结构化 JSON")
		m.addWarning(session, reason)
		return verdict, nil
	}
	revised := parseVerdict(out, chairman.Name, 0)
	if strings.TrimSpace(revised.Summary) == "" && strings.TrimSpace(revised.Recommendation) == "" {
		m.addWarning(session, "独立复核后的主席修订为空，保留原小结")
		return verdict, nil
	}
	normalized, warnings := validateIssueReviews(revised.IssueReviews, issues)
	revised.IssueReviews = normalized
	for _, warning := range warnings {
		m.addWarning(session, warning)
	}
	revised.Partial = revised.Partial || verdict.Partial
	revised.RevisedFromSHA256 = originalSHA
	if strings.TrimSpace(revised.RevisionReason) == "" {
		revised.RevisionReason = trimRunes(strings.Join(reasons, "；"), 400)
	}
	for _, reason := range reasons {
		if !containsString(revised.Disagreements, reason) {
			revised.Disagreements = append(revised.Disagreements, "独立复核反对："+reason)
		}
	}
	if len(revised.ProposedActions) == 0 && len(verdict.ProposedActions) > 0 {
		revised.ProposedActions = append([]ProposedAction(nil), verdict.ProposedActions...)
	}
	return revised, []VerdictRevision{{
		Verdict: original,
		SHA256:  originalSHA,
		Reason:  trimRunes(strings.Join(reasons, "；"), 600),
		At:      time.Now().UnixMilli(),
	}}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func mergeAttestationFindings(verdict *Verdict, attestations []Attestation) *Verdict {
	if verdict == nil {
		return verdict
	}
	for _, attestation := range attestations {
		if attestation.Position == AttestationConditional {
			for _, condition := range attestation.Conditions {
				verdict.Conditions = appendUniqueStrings(verdict.Conditions, condition)
			}
		}
		if attestation.Position != AttestationOppose {
			continue
		}
		reason := firstNonEmpty(attestation.Reason, attestation.Error, "未提供理由")
		entry := "独立复核反对：" + attestation.AgentName + "：" + reason
		if !containsString(verdict.Disagreements, entry) {
			verdict.Disagreements = append(verdict.Disagreements, entry)
		}
	}
	return verdict
}
