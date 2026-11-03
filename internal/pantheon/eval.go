package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Protocol-comparison labels are derived from the persisted request, so the
// evaluator does not need a second mutable "experiment" record to drift from
// the session it describes.
const (
	EvaluationVariantSingle            = "single"
	EvaluationVariantIndependent       = "independent_synthesis"
	EvaluationVariantRoundtable        = "roundtable"
	EvaluationVariantTargetedChallenge = "targeted_challenge"
	EvaluationVariantRedTeam           = "red_team"
	EvaluationVariantCouncilRelay      = "council_relay"
)

var evaluationCoreVariants = []string{
	EvaluationVariantSingle,
	EvaluationVariantIndependent,
	EvaluationVariantRoundtable,
	EvaluationVariantTargetedChallenge,
}

// EvaluationRun is a read-only projection of one stored session. Quality
// judgments that require a human reviewer are intentionally absent.
type EvaluationRun struct {
	SessionID             string            `json:"sessionId"`
	CaseKey               string            `json:"caseKey"`
	Title                 string            `json:"title"`
	Variant               string            `json:"variant"`
	Status                string            `json:"status"`
	Protocol              string            `json:"protocol"`
	Preset                string            `json:"preset"`
	AgentIDs              []string          `json:"agentIds,omitempty"`
	RuntimeVersions       map[string]string `json:"runtimeVersions,omitempty"`
	ParticipantCount      int               `json:"participantCount"`
	VersionedParticipants int               `json:"versionedParticipants"`
	ExpectedRounds        int               `json:"expectedRounds"`
	ActualRounds          int               `json:"actualRounds"`
	CallCount             int               `json:"callCount"`
	RetryCount            int               `json:"retryCount"`
	ErrorCount            int               `json:"errorCount"`
	ToolCallCount         int               `json:"toolCallCount"`
	WallDurationMs        int64             `json:"wallDurationMs"`
	AgentDurationMs       int64             `json:"agentDurationMs"`
	GapCount              int               `json:"gapCount"`
	DiscussionCount       int               `json:"discussionCount"`
	EvidenceHandledCount  int               `json:"evidenceHandledCount"`
	UnverifiedMentions    int               `json:"unverifiedMentions"`
	UserNoteCount         int               `json:"userNoteCount"`
	IssueCount            int               `json:"issueCount"`
	ResolvedIssueCount    int               `json:"resolvedIssueCount"`
	DissentIssueCount     int               `json:"dissentIssueCount"`
	AttestationCount      int               `json:"attestationCount"`
	UnknownAttestations   int               `json:"unknownAttestations"`
	VerdictPartial        bool              `json:"verdictPartial,omitempty"`
	VerdictRevised        bool              `json:"verdictRevised,omitempty"`
	SignOffDecision       string            `json:"signOffDecision,omitempty"`
	RunModes              []string          `json:"runModes,omitempty"`
	CreatedAt             int64             `json:"createdAt"`
}

type EvaluationVariantSummary struct {
	Variant           string  `json:"variant"`
	Sessions          int     `json:"sessions"`
	CompletedSessions int     `json:"completedSessions"`
	SignedOffSessions int     `json:"signedOffSessions"`
	AcceptedSessions  int     `json:"acceptedSessions"`
	AvgRounds         float64 `json:"avgRounds,omitempty"`
	AvgWallDurationMs int64   `json:"avgWallDurationMs,omitempty"`
	AvgCalls          float64 `json:"avgCalls,omitempty"`
	GapSessions       int     `json:"gapSessions,omitempty"`
	DissentIssues     int     `json:"dissentIssues,omitempty"`
	RevisedVerdicts   int     `json:"revisedVerdicts,omitempty"`
}

type EvaluationCaseCoverage struct {
	Key               string   `json:"key"`
	Title             string   `json:"title"`
	Variants          []string `json:"variants"`
	CompletedVariants []string `json:"completedVariants,omitempty"`
	MissingVariants   []string `json:"missingVariants,omitempty"`
}

type EvaluationReport struct {
	GeneratedAt         int64                      `json:"generatedAt"`
	RequiredCases       int                        `json:"requiredCases"`
	CaseCount           int                        `json:"caseCount"`
	QualifiedCases      int                        `json:"qualifiedCases"`
	RequiredVariants    []string                   `json:"requiredVariants"`
	UnsupportedVariants []string                   `json:"unsupportedVariants,omitempty"`
	GateReady           bool                       `json:"gateReady"`
	Variants            []EvaluationVariantSummary `json:"variants"`
	Cases               []EvaluationCaseCoverage   `json:"cases"`
	Runs                []EvaluationRun            `json:"runs"`
	Warnings            []string                   `json:"warnings,omitempty"`
}

// EvaluateSession derives operational comparison metrics without mutating the
// session. Cost remains explicitly unavailable because the harnesses do not
// emit token or price events.
func EvaluateSession(session *Session) EvaluationRun {
	if session == nil {
		return EvaluationRun{}
	}
	run := EvaluationRun{
		SessionID:        session.ID,
		CaseKey:          EvaluationCaseKey(session),
		Title:            session.Title,
		Variant:          EvaluationVariantOf(session),
		Status:           session.Status,
		Protocol:         session.Request.Protocol,
		Preset:           session.Request.Preset,
		AgentIDs:         evaluationAgentIDs(session),
		ParticipantCount: len(session.Participants),
		ExpectedRounds:   session.Request.Rounds,
		ActualRounds:     session.CurrentRound,
		GapCount:         len(session.Gaps),
		IssueCount:       len(session.Issues),
		AttestationCount: len(session.Attestations),
		VerdictRevised:   len(session.VerdictRevisions) > 0,
		CreatedAt:        session.CreatedAt,
	}
	if session.Verdict != nil {
		run.VerdictPartial = session.Verdict.Partial
		run.VerdictRevised = run.VerdictRevised || session.Verdict.RevisedFromSHA256 != ""
	}
	if session.SignOff != nil {
		run.SignOffDecision = session.SignOff.Decision
	}

	versions := map[string]string{}
	for _, participant := range session.Participants {
		if participant.AgentID != "" && strings.TrimSpace(participant.Version) != "" {
			versions[participant.AgentID] = strings.TrimSpace(participant.Version)
			run.VersionedParticipants++
		}
	}
	if len(versions) > 0 {
		run.RuntimeVersions = versions
	}

	runModes := map[string]bool{}
	run.UserNoteCount = 0
	for _, contribution := range session.Contributions {
		if contribution.Phase == PhaseUserNote || contribution.AgentID == "user" {
			run.UserNoteCount++
			continue
		}
		if contribution.Phase != PhaseOpening && contribution.Phase != PhaseDiscussion {
			continue
		}
		run.DiscussionCount++
		if strings.TrimSpace(contribution.EvidenceHandle) != "" {
			run.EvidenceHandledCount++
		}
		if strings.Contains(contribution.Content, "未核验") ||
			strings.Contains(contribution.Content, "未验证") {
			run.UnverifiedMentions++
		}
		if mode := strings.TrimSpace(contribution.RunMode); mode != "" {
			runModes[mode] = true
		}
	}
	for mode := range runModes {
		run.RunModes = append(run.RunModes, mode)
	}
	sort.Strings(run.RunModes)

	var completedAt int64
	for _, event := range session.Events {
		switch event.Kind {
		case EventAgentStart:
			run.CallCount++
		case EventAgentRetry:
			run.RetryCount++
		case EventAgentError:
			run.ErrorCount++
		case EventToolStart:
			run.ToolCallCount++
		case EventAgentOutput:
			run.AgentDurationMs += event.DurationMs
		case EventComplete:
			if event.At > completedAt {
				completedAt = event.At
			}
		}
	}
	if run.ActualRounds == 0 {
		for _, event := range session.Events {
			if event.Round > run.ActualRounds {
				run.ActualRounds = event.Round
			}
		}
	}
	finishedAt := completedAt
	if finishedAt == 0 {
		finishedAt = session.UpdatedAt
	}
	if finishedAt > session.CreatedAt {
		run.WallDurationMs = finishedAt - session.CreatedAt
	}

	for _, issue := range session.Issues {
		switch issue.Status {
		case "resolved":
			run.ResolvedIssueCount++
		case "dissent":
			run.DissentIssueCount++
		}
	}
	for _, attestation := range session.Attestations {
		if attestation.Position == AttestationUnknown {
			run.UnknownAttestations++
		}
	}
	return run
}

// EvaluationCaseKey groups repeated runs of the same brief while deliberately
// excluding title, protocol, preset and round count, which are experiment
// variables rather than case identity.
func EvaluationCaseKey(session *Session) string {
	if session == nil {
		return ""
	}
	req := session.Request
	materials := evaluationMaterials(session)
	payload := struct {
		Topic           string        `json:"topic"`
		Context         string        `json:"context"`
		SuccessCriteria string        `json:"successCriteria"`
		Workspace       string        `json:"workspace"`
		ParentSessionID string        `json:"parentSessionId,omitempty"`
		Materials       []InputDigest `json:"materials,omitempty"`
	}{
		Topic:           normalizeCaseText(req.Topic),
		Context:         normalizeCaseText(req.Context),
		SuccessCriteria: normalizeCaseText(req.SuccessCriteria),
		Workspace:       strings.TrimSpace(req.Workspace),
		ParentSessionID: strings.TrimSpace(req.ParentSessionID),
		Materials:       materials,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return "case_" + hex.EncodeToString(sum[:8])
}

// EvaluationVariantOf classifies a stored request into the R6 comparison
// vocabulary. Unsupported modes are not invented: fixed voting remains
// outside the Pantheon session protocol.
func EvaluationVariantOf(session *Session) string {
	if session == nil {
		return ""
	}
	switch evaluationPreset(session.Request.Preset) {
	case PresetTargetedChallenge:
		return EvaluationVariantTargetedChallenge
	case PresetRedTeam:
		return EvaluationVariantRedTeam
	}
	if len(evaluationAgentIDs(session)) <= 1 {
		return EvaluationVariantSingle
	}
	if session.Request.Protocol == ProtocolCouncil {
		return EvaluationVariantCouncilRelay
	}
	if session.Request.Rounds <= 1 && !session.Request.UntilConsensus {
		return EvaluationVariantIndependent
	}
	return EvaluationVariantRoundtable
}

// evaluationPreset keeps historical sessions without a preset readable
// as the default roundtable instead of creating an "unknown" comparison bucket.
func evaluationPreset(raw string) string {
	preset, err := NormalizePreset(raw)
	if err != nil {
		return raw
	}
	return preset
}

func evaluationAgentIDs(session *Session) []string {
	ids := uniqueNonEmpty(session.Request.AgentIDs)
	if len(ids) > 0 {
		return ids
	}
	for _, participant := range session.Participants {
		if participant.AgentID != "" {
			ids = append(ids, participant.AgentID)
		}
	}
	return uniqueNonEmpty(ids)
}

func normalizeCaseText(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

func evaluationMaterials(session *Session) []InputDigest {
	materials := make([]InputDigest, 0, len(session.InputManifest)+len(session.Request.Evidence)+len(session.Request.Attachments))
	if len(session.InputManifest) > 0 {
		for _, item := range session.InputManifest {
			// 图片附件已有文件摘要；共享识图是由本地视觉模型现生成的可变派生材料，
			// 不能让它把同一张图的重跑拆成不同议题。
			if strings.HasPrefix(item.Name, "共享识图 · ") {
				continue
			}
			path := ""
			if item.SHA256 == "" {
				path = item.Path
			}
			materials = append(materials, InputDigest{
				Kind: item.Kind, Name: item.Name, SHA256: item.SHA256,
				Path: path, Size: item.Size, Error: item.Error,
			})
		}
	} else {
		for _, item := range session.Request.Evidence {
			name := item.Name
			if name == "" {
				name = "未命名材料"
			}
			sum := sha256.Sum256([]byte(item.Content))
			materials = append(materials, InputDigest{
				Kind: "evidence", Name: name, SHA256: hex.EncodeToString(sum[:]),
				Size: int64(len(item.Content)),
			})
		}
		for _, attachment := range session.Request.Attachments {
			materials = append(materials, InputDigest{
				Kind: "attachment", Name: attachment.Name, Size: attachment.Size,
			})
		}
	}
	sort.Slice(materials, func(i, j int) bool {
		left := materials[i].Kind + "\x00" + materials[i].Name + "\x00" + materials[i].SHA256 + "\x00" + materials[i].Path + "\x00" + materials[i].Error
		right := materials[j].Kind + "\x00" + materials[j].Name + "\x00" + materials[j].SHA256 + "\x00" + materials[j].Path + "\x00" + materials[j].Error
		return left < right
	})
	return materials
}

// BuildEvaluationReport groups sessions by brief digest and summarizes only
// metrics that can be recomputed from session.json. Human review fields and
// token cost stay outside the gate rather than being guessed.
func BuildEvaluationReport(runs []EvaluationRun, requiredCases int) EvaluationReport {
	if requiredCases <= 0 {
		requiredCases = 20
	}
	report := EvaluationReport{
		GeneratedAt:         time.Now().UnixMilli(),
		RequiredCases:       requiredCases,
		RequiredVariants:    append([]string(nil), evaluationCoreVariants...),
		UnsupportedVariants: []string{"fixed_vote"},
		Runs:                append([]EvaluationRun(nil), runs...),
	}
	for index := range report.Runs {
		if report.Runs[index].CaseKey == "" {
			report.Runs[index].CaseKey = "case_unkeyed"
		}
	}
	sort.Slice(report.Runs, func(i, j int) bool {
		if report.Runs[i].CaseKey == report.Runs[j].CaseKey {
			if report.Runs[i].CreatedAt == report.Runs[j].CreatedAt {
				return report.Runs[i].SessionID < report.Runs[j].SessionID
			}
			return report.Runs[i].CreatedAt < report.Runs[j].CreatedAt
		}
		return report.Runs[i].CaseKey < report.Runs[j].CaseKey
	})

	type caseAggregate struct {
		title             string
		variants          map[string]bool
		completedVariants map[string]bool
	}
	cases := map[string]*caseAggregate{}
	summaries := map[string]*EvaluationVariantSummary{}
	var totalRounds, totalWall, totalCalls map[string]int64
	totalRounds = map[string]int64{}
	totalWall = map[string]int64{}
	totalCalls = map[string]int64{}
	incompleteVersionSnapshots := 0

	for _, run := range report.Runs {
		aggregate := cases[run.CaseKey]
		if aggregate == nil {
			aggregate = &caseAggregate{
				title:             run.Title,
				variants:          map[string]bool{},
				completedVariants: map[string]bool{},
			}
			cases[run.CaseKey] = aggregate
		}
		if aggregate.title == "" {
			aggregate.title = run.Title
		}
		aggregate.variants[run.Variant] = true

		summary := summaries[run.Variant]
		if summary == nil {
			summary = &EvaluationVariantSummary{Variant: run.Variant}
			summaries[run.Variant] = summary
		}
		summary.Sessions++
		if run.Status != StatusCompleted {
			if run.ParticipantCount == 0 || run.VersionedParticipants < run.ParticipantCount {
				incompleteVersionSnapshots++
			}
			continue
		}
		aggregate.completedVariants[run.Variant] = true
		summary.CompletedSessions++
		totalRounds[run.Variant] += int64(run.ActualRounds)
		totalWall[run.Variant] += run.WallDurationMs
		totalCalls[run.Variant] += int64(run.CallCount)
		if run.GapCount > 0 {
			summary.GapSessions++
		}
		summary.DissentIssues += run.DissentIssueCount
		if run.VerdictRevised {
			summary.RevisedVerdicts++
		}
		if run.SignOffDecision != "" {
			summary.SignedOffSessions++
			if run.SignOffDecision == "accepted" {
				summary.AcceptedSessions++
			}
		}
		if run.ParticipantCount == 0 || run.VersionedParticipants < run.ParticipantCount {
			incompleteVersionSnapshots++
		}
	}

	caseKeys := make([]string, 0, len(cases))
	for key := range cases {
		caseKeys = append(caseKeys, key)
	}
	sort.Strings(caseKeys)
	for _, key := range caseKeys {
		aggregate := cases[key]
		coverage := EvaluationCaseCoverage{
			Key:               key,
			Title:             aggregate.title,
			Variants:          sortedMapKeys(aggregate.variants),
			CompletedVariants: sortedMapKeys(aggregate.completedVariants),
		}
		for _, variant := range evaluationCoreVariants {
			if !aggregate.completedVariants[variant] {
				coverage.MissingVariants = append(coverage.MissingVariants, variant)
			}
		}
		if len(coverage.MissingVariants) == 0 {
			report.QualifiedCases++
		}
		report.Cases = append(report.Cases, coverage)
	}
	report.CaseCount = len(report.Cases)
	report.GateReady = report.QualifiedCases >= requiredCases

	for _, summary := range summaries {
		if summary.CompletedSessions > 0 {
			count := int64(summary.CompletedSessions)
			summary.AvgRounds = roundOne(float64(totalRounds[summary.Variant]) / float64(count))
			summary.AvgCalls = roundOne(float64(totalCalls[summary.Variant]) / float64(count))
			summary.AvgWallDurationMs = int64(math.Round(float64(totalWall[summary.Variant]) / float64(count)))
		}
		report.Variants = append(report.Variants, *summary)
	}
	sort.Slice(report.Variants, func(i, j int) bool {
		orderI, orderJ := evaluationVariantOrder(report.Variants[i].Variant), evaluationVariantOrder(report.Variants[j].Variant)
		if orderI == orderJ {
			return report.Variants[i].Variant < report.Variants[j].Variant
		}
		return orderI < orderJ
	})

	if report.QualifiedCases < requiredCases {
		report.Warnings = append(report.Warnings, fmt.Sprintf(
			"有效对照仅 %d/%d 个议题；每个议题须完成 %s",
			report.QualifiedCases, requiredCases, strings.Join(evaluationCoreVariants, "、"),
		))
	}
	missingCounts := map[string]int{}
	for _, coverage := range report.Cases {
		for _, variant := range coverage.MissingVariants {
			missingCounts[variant]++
		}
	}
	for _, variant := range evaluationCoreVariants {
		if missingCounts[variant] > 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf(
				"%d 个议题缺少已完成的 %s", missingCounts[variant], variant,
			))
		}
	}
	if incompleteVersionSnapshots > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf(
			"%d 场会话存在缺失的 Harness/模型版本快照；这些场次不能证明全员版本固定",
			incompleteVersionSnapshots,
		))
	}
	report.Warnings = append(report.Warnings,
		"版本快照只证明探针返回的 Harness 版本（Grok 席位为实际模型名）；CLI 席位的具体模型版本仍须在实验记录中另行固定。",
		"Token 与费用未入档，本报告不输出成本结论。",
		"事实错误、关键反例发现、未核验主张比例和再次使用意愿不能从会话自动推断，须另附人工评审。",
		"固定投票不属于当前众神殿协议；如需该基线，应使用外部同题实验数据，不能伪装成会话。",
	)
	return report
}

// EvaluateProtocols loads the durable council ledger and returns the same R6
// report exposed by the local CLI. It does not mutate or re-run any council.
func (m *Manager) EvaluateProtocols(requiredCases int) (EvaluationReport, error) {
	if m == nil || m.store == nil {
		return EvaluationReport{}, fmt.Errorf("众神殿数据存储未就绪")
	}
	m.mu.Lock()
	sessions, warnings, err := m.store.all()
	m.mu.Unlock()
	if err != nil {
		return EvaluationReport{}, err
	}
	runs := make([]EvaluationRun, 0, len(sessions))
	for _, session := range sessions {
		runs = append(runs, EvaluateSession(session))
	}
	report := BuildEvaluationReport(runs, requiredCases)
	report.Warnings = append(report.Warnings, warnings...)
	return report, nil
}

func FormatEvaluationMarkdown(report EvaluationReport, includeRuns bool) string {
	var b strings.Builder
	b.WriteString("# 众神殿协议对照报告\n\n")
	fmt.Fprintf(&b, "- 生成：%s\n", formatMillis(report.GeneratedAt))
	fmt.Fprintf(&b, "- 门禁：%s\n", readyLabel(report.GateReady))
	fmt.Fprintf(&b, "- 有效对照：%d / %d 个议题\n", report.QualifiedCases, report.RequiredCases)
	fmt.Fprintf(&b, "- 必需变体：%s\n", strings.Join(report.RequiredVariants, "、"))
	if len(report.UnsupportedVariants) > 0 {
		fmt.Fprintf(&b, "- 未纳入：%s（当前没有对应会话协议）\n", strings.Join(report.UnsupportedVariants, "、"))
	}
	b.WriteString("\n> 本报告只复算状态、轮次、调用、耗时、缺口、台账与签注；费用和人工质量判断不会被推测。\n\n")

	b.WriteString("## 变体总览\n\n")
	b.WriteString("| 变体 | 场次 | 完成 | 已签注 | 确认率 | 平均轮次 | 平均时长 | 平均调用 | 缺口场次 | 保留异议 |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, summary := range report.Variants {
		fmt.Fprintf(
			&b,
			"| %s | %d | %d | %d | %s | %.1f | %s | %.1f | %d | %d |\n",
			EvaluationVariantLabel(summary.Variant),
			summary.Sessions,
			summary.CompletedSessions,
			summary.SignedOffSessions,
			formatAcceptanceRate(summary.AcceptedSessions, summary.SignedOffSessions),
			summary.AvgRounds,
			formatEvaluationDuration(summary.AvgWallDurationMs),
			summary.AvgCalls,
			summary.GapSessions,
			summary.DissentIssues,
		)
	}

	b.WriteString("\n## 议题覆盖\n\n")
	b.WriteString("| 对照键 | 议题 | 已完成变体 | 缺少 |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, coverage := range report.Cases {
		fmt.Fprintf(
			&b,
			"| `%s` | %s | %s | %s |\n",
			coverage.Key,
			escapeMarkdownTable(coverage.Title),
			firstNonEmpty(strings.Join(coverage.CompletedVariants, "、"), "无"),
			firstNonEmpty(strings.Join(coverage.MissingVariants, "、"), "无"),
		)
	}
	if len(report.Cases) == 0 {
		b.WriteString("| 无 | 尚无议事记录 | 无 | 全部 |\n")
	}

	b.WriteString("\n## 门禁警示\n\n")
	for _, warning := range report.Warnings {
		b.WriteString("- ")
		b.WriteString(warning)
		b.WriteString("\n")
	}

	if includeRuns {
		b.WriteString("\n## 逐场指标\n\n")
		b.WriteString("| 会话 | 变体 | 状态 | 轮次 | 调用 | 重试 | 错误 | 时长 | 证据句柄 | 用户插话 | 未闭合异议 | 版本快照 |\n")
		b.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
		for _, run := range report.Runs {
			fmt.Fprintf(
				&b,
				"| `%s` | %s | %s | %d/%d | %d | %d | %d | %s | %d/%d | %d | %d | %s |\n",
				run.SessionID,
				EvaluationVariantLabel(run.Variant),
				run.Status,
				run.ActualRounds,
				run.ExpectedRounds,
				run.CallCount,
				run.RetryCount,
				run.ErrorCount,
				formatEvaluationDuration(run.WallDurationMs),
				run.EvidenceHandledCount,
				run.DiscussionCount,
				run.UserNoteCount,
				run.DissentIssueCount,
				formatRuntimeVersions(run.RuntimeVersions),
			)
		}
	}
	b.WriteString("\n---\n\n> 门禁未就绪时不得据本报告宣称某种协议更优。\n")
	return b.String()
}

func EvaluationVariantLabel(variant string) string {
	switch variant {
	case EvaluationVariantSingle:
		return "单席"
	case EvaluationVariantIndependent:
		return "独立多答"
	case EvaluationVariantRoundtable:
		return "圆桌会诊"
	case EvaluationVariantTargetedChallenge:
		return "定向质询"
	case EvaluationVariantRedTeam:
		return "红队评审"
	case EvaluationVariantCouncilRelay:
		return "圆桌接力（历史）"
	default:
		return firstNonEmpty(variant, "未分类")
	}
}

func sortedMapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func evaluationVariantOrder(variant string) int {
	for i, candidate := range evaluationCoreVariants {
		if candidate == variant {
			return i
		}
	}
	switch variant {
	case EvaluationVariantRedTeam:
		return len(evaluationCoreVariants)
	case EvaluationVariantCouncilRelay:
		return len(evaluationCoreVariants) + 1
	default:
		return len(evaluationCoreVariants) + 2
	}
}

func roundOne(value float64) float64 {
	return math.Round(value*10) / 10
}

func formatAcceptanceRate(accepted, signedOff int) string {
	if signedOff == 0 {
		return "未签注"
	}
	return fmt.Sprintf("%.0f%%（%d/%d）", float64(accepted)*100/float64(signedOff), accepted, signedOff)
}

func formatEvaluationDuration(ms int64) string {
	if ms <= 0 {
		return "未记录"
	}
	if ms < 60_000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	return fmt.Sprintf("%.1fm", float64(ms)/60_000)
}

func formatRuntimeVersions(versions map[string]string) string {
	if len(versions) == 0 {
		return "未记录"
	}
	keys := make([]string, 0, len(versions))
	for key := range versions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+versions[key])
	}
	return strings.Join(parts, "；")
}

func escapeMarkdownTable(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.ReplaceAll(value, "|", "\\|")
}

func readyLabel(ready bool) string {
	if ready {
		return "就绪"
	}
	return "未就绪"
}
