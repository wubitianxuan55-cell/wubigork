// Package pantheon implements Gaea's local heterogeneous-agent council.
//
// The package owns the deliberation protocol and durable council record. It
// deliberately does not import internal/app: adapters execute external agent
// CLIs, while persistence and event delivery remain injectable for tests.
package pantheon

const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"

	// §4.1 补两个会话态：waiting_input 是"有未答问题"，议事挂起而不是失败；
	// interrupted 是"重启时正在跑，但没观测到收声"——没人说它失败，只是没人
	// 看见。四个既有取值不变。
	StatusWaitingInput = "waiting_input"
	StatusInterrupted  = "interrupted"

	PhasePreparing   = "preparing"
	PhaseOpening     = "opening"
	PhaseDiscussion  = "discussion"
	PhaseSynthesis   = "synthesis"
	PhaseIndependent = "independent"
	PhaseReview      = "review"
	PhaseChallenge   = "challenge"
	PhaseVerdict     = "verdict"
	PhaseAttestation = "attestation"
	PhaseCompleted   = "completed"
	PhaseFailed      = "failed"
	PhaseCancelled   = "cancelled"
	// PhaseUserNote 用户中途评论（2026-11-03 拍板）：不打断当前发言，
	// 以贡献身份入档并全文注入后续席位提示词。
	PhaseUserNote = "user_note"
	// PhaseModeration 主席轮间小结（2026-11-03 拍板）：每轮收尾后主席提炼
	// 真分歧+裁决建议+下轮必答，注入后续全员提示词——治「轮次膨胀无收口」。
	PhaseModeration   = "moderation"
	EventSessionStart = "session_started"
	EventPhase        = "phase"
	EventAgentStart   = "agent_started"
	EventAgentDelta   = "agent_delta"
	EventAgentOutput  = "agent_output"
	EventAgentError   = "agent_error"
	EventAgentSkipped = "agent_skipped"
	EventAgentRetry   = "agent_retry"
	EventUserNote     = "user_note"
	// §4.5 决策门三事件：提问、回答、门自身（开/关/驳回）。载荷见
	// QuestionPayload / AnswerPayload / GatePayload。
	EventQuestion    = "question"
	EventAnswer      = "answer"
	EventGate        = "gate"
	EventToolStart   = "tool_started"
	EventToolResult  = "tool_result"
	EventVerdict     = "verdict"
	EventConsensus   = "consensus"
	EventDIDUsed     = "did_used"
	EventDIDStop     = "did_stop"
	EventAttestation = "attestation"
	EventSignOff     = "signoff"
	EventComplete    = "completed"
	EventError       = "error"
)

// CallRef identifies one logical seat call and the concrete execution attempt
// that produced an observation. CallID survives retries in the same round;
// AttemptID changes for every process or model invocation.
type CallRef struct {
	CallID         string `json:"callId,omitempty"`
	AttemptID      string `json:"attemptId,omitempty"`
	RetryOfAttempt string `json:"retryOfAttempt,omitempty"`
	Attempt        int    `json:"attempt,omitempty"`
}

// 席位状态词表（§4.1）。这些值原先以裸字符串散落在 manager.go，落地前逐条
// 对过：waiting / running / finished / failed / skipped / retrying /
// unavailable / cancelled 八个已有取值与之逐字一致；needs_input 与
// finished_unverified 是本轮新增的两个"没说完成"的态。状态之间能不能走由
// seatstate.go 的转移表裁决，不再靠自由字符串约定。
const (
	SeatWaiting            = "waiting"
	SeatRunning            = "running"
	SeatNeedsInput         = "needs_input"
	SeatFinished           = "finished"
	SeatFinishedUnverified = "finished_unverified"
	SeatFailed             = "failed"
	SeatSkipped            = "skipped"
	SeatUnavailable        = "unavailable"
	SeatCancelled          = "cancelled"
	SeatRetrying           = "retrying"
)

// 观测出处（Observed.Origin）。取值是封闭集合：加一个出处等于加一种证据，
// 必须同时说清它能证明什么。
const (
	OriginAdapter  = "adapter"
	OriginLauncher = "launcher"
	OriginProcess  = "process"
	OriginEvent    = "event"
	OriginUser     = "user"
	OriginRecovery = "recovery"
)

// Observed 记录一条席位状态是被谁、在什么序号上、什么时候看到的。它对付的
// 是"看起来完成了"：状态本身是断言，Observed 是这条断言的出处与新鲜度。
//
// 四条边界（跨来源不可比，是这里最容易踩的坑）：
//   - Origin 取 adapter|launcher|process|event|user|recovery，标明这条状态
//     是谁报的；
//   - AuthorityID 每次应用启动重新生成，禁止跨进程比较序号：重启前后的
//     revision 不可比，跨 authority 只能退回 At 时间戳，否则会得出"状态倒退
//     了"的假结论并触发一次本不该发生的重试；
//   - Incarnation 在同一席位重新绑定（重试、换后端、换图片通道）+1，用来
//     分辨"同一序号的另一辈子"；
//   - Revision 在同一个 authority 内单调递增，但只比较先后、不计数（别的
//     观测者共用同一计数器，计数会留空洞）。
type Observed struct {
	Origin      string `json:"origin,omitempty"`
	AuthorityID string `json:"authorityId,omitempty"`
	Incarnation int64  `json:"incarnation,omitempty"`
	Revision    int64  `json:"revision,omitempty"`
	At          int64  `json:"at,omitempty"`
}

// AgentStatus is a capability probe for one locally installed agent harness.
// Executable is a display/debug path, not a credential or protocol secret.
type AgentStatus struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Available    bool     `json:"available"`
	Executable   string   `json:"executable,omitempty"`
	Version      string   `json:"version,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// EvidenceItem is user-supplied material copied into the council brief.
// Workspace scoping is carried separately in StartRequest so every harness can
// resolve project conventions without switching Gaea's global workspace.
type EvidenceItem struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Attachment is a user-picked image made available to a council session.
// Files stay in place; the host validates them and passes their paths to the
// Agent harnesses without copying or modifying the workspace.
type Attachment struct {
	Name string `json:"name"`
	Path string `json:"path"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
}

// WorkspaceOption is a directory that can scope one council session. Selecting
// a workspace never switches the rest of the Gaea application.
type WorkspaceOption struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Current bool   `json:"current"`
	Recent  bool   `json:"recent,omitempty"`
}

// StartRequest is the durable brief for one council session.
type StartRequest struct {
	Title           string         `json:"title"`
	Topic           string         `json:"topic"`
	Context         string         `json:"context"`
	SuccessCriteria string         `json:"successCriteria"`
	Workspace       string         `json:"workspace,omitempty"`
	Evidence        []EvidenceItem `json:"evidence,omitempty"`
	Attachments     []Attachment   `json:"attachments,omitempty"`
	AgentIDs        []string       `json:"agentIds"`
	ChairmanAgentID string         `json:"chairmanAgentId"`
	Rounds          int            `json:"rounds"`
	UntilConsensus  bool           `json:"untilConsensus,omitempty"`
	MaxRounds       int            `json:"maxRounds,omitempty"`
	ReadOnly        bool           `json:"readOnly"`
	// Protocol records which deliberation path produced the session
	// (2026-10-31 裁决第二刀：对照实验必须可表达、可比对必须可打标签).
	Protocol string `json:"protocol,omitempty"`
	// Preset is the user-visible deliberation shape. Protocol remains the
	// information-visibility axis; targeted challenge forces an independent
	// opening and then privileges issue-directed challenge/response.
	Preset string `json:"preset,omitempty"`
	// ParentSessionID links a follow-up council to the session it continues
	// (2026-10-31 裁决第三刀：续议保留父会话；承接内容以简报为准，不冒充继承全文).
	ParentSessionID string `json:"parentSessionId,omitempty"`
	// GateEachRound 要求每轮结束后停一次，等用户放行再进下一轮（M14 的决策门）。
	// 门没被显式解开之前，整场保持 waiting_input：宁可显示"停着等人"，也不
	// 制造一个 running 却没人干活的假象。
	GateEachRound bool `json:"gateEachRound,omitempty"`
}

// SeatUsage 是一次席位调用结束后，由外部 CLI 自带运行时上报的可核对账目。
// 目前只有 DSH headless --json 会上报（session 事件给引擎会话 ID，step_end
// 事件给每步 usage）；引擎不上报时保持零值，前端显示「未上报」，不用 0 冒充
// 命中率。它回答两个此前只能靠猜的问题：这一席到底调没调那个引擎
// （EngineSessionID 在引擎自己的会话库里真实存在），缓存到底命中多少
// （CacheReadTokens/(InputTokens+CacheReadTokens)）。
type SeatUsage struct {
	EngineSessionID string `json:"engineSessionId,omitempty"`
	Steps           int    `json:"steps,omitempty"`
	InputTokens     int64  `json:"inputTokens,omitempty"`
	CacheReadTokens int64  `json:"cacheReadTokens,omitempty"`
	OutputTokens    int64  `json:"outputTokens,omitempty"`
}

// CacheHitPercent 返回 prompt 前缀缓存命中率（0~100）；没有上报账目时返回 -1，
// 与「上报了但确实 0%」区分开——后者是真的没命中，前者是压根没数。
func (u SeatUsage) CacheHitPercent() float64 {
	total := u.InputTokens + u.CacheReadTokens
	if total <= 0 {
		return -1
	}
	return float64(u.CacheReadTokens) * 100 / float64(total)
}

type Participant struct {
	CallRef
	AgentID string `json:"agentId"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Status  string `json:"status"`
	// Version 是开场探针当时看到的 Harness 版本；Grok 席位记录实际模型名。
	// 对照实验必须在当时固定并留下证据，不能事后用“当前版本”回填历史。
	Version    string `json:"version,omitempty"`
	StartedAt  int64  `json:"startedAt,omitempty"`
	FinishedAt int64  `json:"finishedAt,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
	// Observed 是这条状态的出处与新鲜度证据；Evidence 是一句人能读的
	// "这条状态凭什么"，直接进 UI 悬停。Status 的词表见 Seat* 常量，
	// 转移合法性由 CanTransitionSeat 裁决。
	Observed Observed `json:"observed,omitempty"`
	Evidence string   `json:"evidence,omitempty"`
	// EngineSessionID 是该席位在外部引擎里实际使用的会话 ID（目前 DSH 会报）。
	// 它有两个用途：下一轮用 --session-id 续接同一引擎会话（前缀缓存命中、
	// 不再重复探索），以及给用户一个可去引擎侧核对的「真调用」证据。
	EngineSessionID string `json:"engineSessionId,omitempty"`
}

// 决策门动作（GatePayload.Action）。dismiss 必须写理由，所以 Reason 在
// dismiss 时不是可选装饰。
const (
	GateActionOpen    = "open"
	GateActionResolve = "resolve"
	GateActionDismiss = "dismiss"
)

// QuestionPayload 是一个席位向用户提问的载荷（§4.5）。QuestionID 与 Options
// 缺一即无效：不完整的门挡不住任何东西（M14 的准入），校验由写入方做，
// 这里只承载形状。
type QuestionPayload struct {
	QuestionID string   `json:"questionId"`
	Options    []string `json:"options,omitempty"`
	AskedBy    string   `json:"askedBy,omitempty"`
	At         int64    `json:"at"`
}

// AnswerPayload 是一次回答。同一条回答只允许被一个消费者取走并标记已投递
// （M9），所以 QuestionID 必须精确；若该 attempt 已终止，回答保留为孤儿而
// 不是静默丢弃。
type AnswerPayload struct {
	QuestionID string `json:"questionId"`
	Body       string `json:"body,omitempty"`
	By         string `json:"by,omitempty"`
	At         int64  `json:"at"`
}

// GatePayload 记录一次决策门的开/关/驳回（M14）。
type GatePayload struct {
	Action string `json:"action"` // open | resolve | dismiss
	Reason string `json:"reason,omitempty"`
}

// Event is both the live UI stream payload and the durable session timeline.
type Event struct {
	CallRef
	Seq        int64  `json:"seq"`
	SessionID  string `json:"sessionId"`
	Phase      string `json:"phase"`
	Round      int    `json:"round,omitempty"`
	AgentID    string `json:"agentId,omitempty"`
	Kind       string `json:"kind"`
	Text       string `json:"text,omitempty"`
	ToolID     string `json:"toolId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	ToolStatus string `json:"toolStatus,omitempty"`
	ToolInput  string `json:"toolInput,omitempty"`
	ToolOutput string `json:"toolOutput,omitempty"`
	At         int64  `json:"at"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
	// 决策门载荷：Kind 为 question / answer / gate 时各带一种。Event 既有
	// 的约定是"每种事件带自己的字段"（ToolID/ToolName/ToolStatus...），这里
	// 沿用同一做法，不引入通用 any 载荷。
	Question *QuestionPayload `json:"question,omitempty"`
	Answer   *AnswerPayload   `json:"answer,omitempty"`
	Gate     *GatePayload     `json:"gate,omitempty"`
	// DIDKeys 记录这一轮被消费的核验动作指纹；恢复时据此避免把同一核验
	// 动作重复算成继续讨论的理由。
	DIDKeys []string `json:"didKeys,omitempty"`
}

// ToolTrace is the provider-neutral representation of one live tool action.
// Input and Output are bounded previews, never full provider transcripts.
type ToolTrace struct {
	ID     string
	Name   string
	Status string
	Input  string
	Output string
	Error  string
}

// Discussion actions are rendered as protocol lines at the tail of a
// contribution. They are deliberately provider-neutral: heterogeneous CLI
// output must not be required to produce a complete JSON object.
const (
	DiscussionActClaim     = "claim"
	DiscussionActChallenge = "challenge"
	DiscussionActRespond   = "respond"
	DiscussionActRevise    = "revise"
	DiscussionActReinforce = "reinforce"
	DiscussionActBroaden   = "broaden"

	ResponseAccept          = "accept"
	ResponsePartiallyAccept = "partial"
	ResponseReject          = "reject"
)

// Contribution preserves the raw model output. Structured metadata is kept
// deliberately small because heterogeneous CLI output is not guaranteed to be
// parseable JSON.
type Contribution struct {
	CallRef
	ID         string `json:"id"`
	Phase      string `json:"phase"`
	Round      int    `json:"round,omitempty"`
	AgentID    string `json:"agentId"`
	AgentName  string `json:"agentName"`
	AnonLabel  string `json:"anonLabel,omitempty"`
	Content    string `json:"content"`
	Confidence string `json:"confidence,omitempty"`
	Consensus  string `json:"consensus,omitempty"`
	// Residual 是「剩余分歧」协议行的原文（2026-11-03 V2 P1 停机判据）：
	// 自报「一致」但保留实质分歧的发言，停机门必须读它——空/「无」才放行。
	Residual string `json:"residual,omitempty"`
	// ResidualEvidence / ResidualFlipCondition / ResidualVerification 把一条
	// 分歧从「一句话」补成可闭合的记录：凭什么保留、什么证据能改变判断、
	// 下一步由谁用什么方法核验。模型缺失时保留空值，前端明确显示待补。
	ResidualEvidence      string `json:"residualEvidence,omitempty"`
	ResidualFlipCondition string `json:"residualFlipCondition,omitempty"`
	ResidualVerification  string `json:"residualVerification,omitempty"`
	// ResidualAgainst 指向异议所针对的主张贡献，供宿主归因「主张方」。
	ResidualAgainst string `json:"residualAgainst,omitempty"`
	// DiscussionAct / TargetContributionID / ChallengedClaim / EvidenceHandle
	// 把普通发言变成可定位的讨论动作。字段全部可选：动作缺失或解析失败时，
	// 原文仍然有效，不能让格式问题吞掉一次发言。
	DiscussionAct        string `json:"discussionAct,omitempty"`
	TargetContributionID string `json:"targetContributionId,omitempty"`
	TargetIssueID        string `json:"targetIssueId,omitempty"`
	ChallengedClaim      string `json:"challengedClaim,omitempty"`
	EvidenceHandle       string `json:"evidenceHandle,omitempty"`
	// ResponseVerdict 仅在回应既有质疑时使用：accept / partial / reject。
	ResponseVerdict string `json:"responseVerdict,omitempty"`
	// RunMode 记录本次调用实际施加的只读机制（裁决第三刀：能力不对称必须
	// 进每次调用记录，不能把四个标志写成同一个笼统的「已只读」）。
	RunMode    string `json:"runMode,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
	DurationMs int64  `json:"durationMs,omitempty"`
	// Usage 是本次调用的引擎侧账目（步数/缓存命中/token）。nil = 该引擎不上报，
	// 不是「0 次调用」——前端按「未上报」显示。
	Usage *SeatUsage `json:"usage,omitempty"`
}

// Issue 是分歧台账条目（2026-11-03 V2 P1）：按分歧点（非按席位）记账、贯穿
// 轮次注入、逐条闭合——宿主从「剩余分歧」协议行自动开条，不依赖模型自觉。
type Issue struct {
	ID        string `json:"id"`
	Round     int    `json:"round"`
	AgentID   string `json:"agentId"`
	AgentName string `json:"agentName"`
	Summary   string `json:"summary"`
	// Origin 区分条目来源：challenge=定向质疑直接开条；residual=从剩余分歧行兜底开条。
	// 两条入口共用同一台账，不能生成两套并行真相。
	Origin string `json:"origin,omitempty"`
	// TargetContributionID / ChallengedClaim 是质疑的关系骨架；目标无法定位时留空，
	// 由 UI 明确显示“未定位”，不按措辞相似度猜测。
	TargetContributionID string `json:"targetContributionId,omitempty"`
	ChallengedClaim      string `json:"challengedClaim,omitempty"`
	// 主张方与异议方分开保存；主张方只从模型明确引用的贡献归因，未引用时留空，
	// 不用“其他席位都算”这种猜测补位。
	ClaimAgentIDs     []string `json:"claimAgentIds,omitempty"`
	ClaimAgentNames   []string `json:"claimAgentNames,omitempty"`
	DissentAgentIDs   []string `json:"dissentAgentIds,omitempty"`
	DissentAgentNames []string `json:"dissentAgentNames,omitempty"`
	CounterpointIDs   []string `json:"counterpointIds,omitempty"`
	// Evidence 是支持保留异议的证据句柄（贡献 ID、文件、命令或明确标注的未核验项）。
	Evidence string `json:"evidence,omitempty"`
	// FlipCondition 说明出现什么新证据时，提出者承诺重新判断。
	FlipCondition string `json:"flipCondition,omitempty"`
	// Verification 是下一项可执行、能减少不确定性的核验动作。
	Verification string `json:"verification,omitempty"`
	// ResponseVerdict 与响应证据记录被质疑方最近一次明确回应；它只更新条目，
	// 不能代替提出者撤回，也不会自动把分歧标成已解决。
	ResponseVerdict         string   `json:"responseVerdict,omitempty"`
	ResponseEvidence        string   `json:"responseEvidence,omitempty"`
	ResponseAgentIDs        []string `json:"responseAgentIds,omitempty"`
	ResponseAgentNames      []string `json:"responseAgentNames,omitempty"`
	ResponseContributionIDs []string `json:"responseContributionIds,omitempty"`
	// Blocking 表示该分歧曾实际阻止把全员一致当作停机条件。
	Blocking bool `json:"blocking,omitempty"`
	// Disposition 是台账去向：等待回应、第 N 轮撤回或结案仍保留。
	Disposition string `json:"disposition,omitempty"`
	// ResolutionEvidence 记录撤回时引用的证据；空值意味着翻转未举证。
	ResolutionEvidence string `json:"resolutionEvidence,omitempty"`
	// LastSeenIn 记录同一分歧最近一次仍被保留的轮次。
	LastSeenIn int `json:"lastSeenIn,omitempty"`
	// Status: open=未闭合 / resolved=提出者后续轮撤回（须翻转举证） /
	// dissent=讨论结束仍保留（合法终态，收口不等于消灭分歧）。
	Status     string `json:"status"`
	ResolvedAt int64  `json:"resolvedAt,omitempty"`
	ResolvedIn int    `json:"resolvedIn,omitempty"`
}

// IssueReview 是主席对一条既有分歧的结构化审阅建议。它不改变宿主维护的
// Open/Resolved/Dissent 状态；只有提出者后续撤回才会改状态。Outcome:
// resolved=建议撤回 / retain=建议保留 / user_decision=交用户裁决 /
// needs_verification=先补核验。
type IssueReview struct {
	IssueID  string `json:"issueId"`
	Outcome  string `json:"outcome"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence,omitempty"`
}

// ProposedAction is a sign-off review artifact, never an execution request.
// It turns the verdict into a reviewable plan while keeping execution in a
// separate mode with its own permissions and workspace.
type ProposedAction struct {
	ID                 string   `json:"id,omitempty"`
	Action             string   `json:"action"`
	Rationale          string   `json:"rationale,omitempty"`
	DependsOn          []string `json:"dependsOn,omitempty"`
	AcceptanceEvidence string   `json:"acceptanceEvidence,omitempty"`
	SuggestedOwner     string   `json:"suggestedOwner,omitempty"`
	Risk               string   `json:"risk,omitempty"`
}

const (
	AttestationSupport     = "support"
	AttestationConditional = "conditional"
	AttestationOppose      = "oppose"
	AttestationUnknown     = "unknown"
)

type Attestation struct {
	AgentID       string   `json:"agentId"`
	AgentName     string   `json:"agentName"`
	Position      string   `json:"position"`
	Reason        string   `json:"reason,omitempty"`
	Conditions    []string `json:"conditions,omitempty"`
	IssueIDs      []string `json:"issueIds,omitempty"`
	VerdictSHA256 string   `json:"verdictSha256,omitempty"`
	Error         string   `json:"error,omitempty"`
	CreatedAt     int64    `json:"createdAt"`
}

type VerdictRevision struct {
	Verdict Verdict `json:"verdict"`
	SHA256  string  `json:"sha256"`
	Reason  string  `json:"reason"`
	At      int64   `json:"at"`
}

type Verdict struct {
	Summary           string           `json:"summary"`
	Recommendation    string           `json:"recommendation"`
	Conditions        []string         `json:"conditions,omitempty"`
	Disagreements     []string         `json:"disagreements,omitempty"`
	Risks             []string         `json:"risks,omitempty"`
	EvidenceNotes     []string         `json:"evidenceNotes,omitempty"`
	IssueReviews      []IssueReview    `json:"issueReviews,omitempty"`
	ProposedActions   []ProposedAction `json:"proposedActions,omitempty"`
	RevisedFromSHA256 string           `json:"revisedFromSha256,omitempty"`
	RevisionReason    string           `json:"revisionReason,omitempty"`
	Chairman          string           `json:"chairman"`
	Partial           bool             `json:"partial"`
	CreatedAt         int64            `json:"createdAt"`
}

type SignOff struct {
	Decision string `json:"decision"`
	Note     string `json:"note,omitempty"`
	At       int64  `json:"at"`
	// VerdictSHA256 绑定被签注的小结版本（裁决第三刀：转发件要能证明
	// 用户确认的是哪一版，签注与对外交付物不能是两份真相）。O-H 起哈希范围
	// 含 Gaps：确认的是"这一版小结加上这些没算成功的席位"。
	VerdictSHA256 string `json:"verdictSha256,omitempty"`
	// Gaps 是签注那一刻的缺口快照，和哈希里的内容一致，便于对外展示。
	Gaps []string `json:"gaps,omitempty"`
}

type Session struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Status        string         `json:"status"`
	Phase         string         `json:"phase"`
	CurrentRound  int            `json:"currentRound,omitempty"`
	CreatedAt     int64          `json:"createdAt"`
	UpdatedAt     int64          `json:"updatedAt"`
	Request       StartRequest   `json:"request"`
	Participants  []Participant  `json:"participants"`
	Events        []Event        `json:"events"`
	Contributions []Contribution `json:"contributions"`
	Verdict       *Verdict       `json:"verdict,omitempty"`
	SignOff       *SignOff       `json:"signoff,omitempty"`
	ResultPath    string         `json:"resultPath,omitempty"`
	InputManifest []InputDigest  `json:"inputManifest,omitempty"`
	Warnings      []string       `json:"warnings,omitempty"`
	// Gaps 列出结案时不能算成功的席位（finished_unverified | unavailable |
	// skipped），并进入 SignOff 的哈希范围（§4.2）：断线的第五席留在缺口里，
	// 不能悄悄算成弃权。
	Gaps []string `json:"gaps,omitempty"`
	// Issues 是分歧台账（2026-11-03 V2 P1）：从「剩余分歧」协议行自动开条，
	// 贯穿轮次注入 prompt，逐条闭合；结案仍 open 的转 dissent——收口不等于
	// 消灭分歧，消灭的是「没人记得它」。
	Issues           []Issue           `json:"issues,omitempty"`
	Attestations     []Attestation     `json:"attestations,omitempty"`
	VerdictRevisions []VerdictRevision `json:"verdictRevisions,omitempty"`
	Error            string            `json:"error,omitempty"`
}

// InputDigest pins one start-time input so later re-runs and path comparisons
// keep a stable evidence baseline (2026-10-31 裁决第三刀：真正会漂移的是盘上
// 活文件——简报正文已随 session.json 落盘，附件文件在讨论期间可被改动).
type InputDigest struct {
	Kind   string `json:"kind"` // evidence | attachment
	Name   string `json:"name"`
	Path   string `json:"path,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SessionSummary is the list payload. It intentionally omits full transcripts.
type SessionSummary struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Workspace      string `json:"workspace,omitempty"`
	Status         string `json:"status"`
	Phase          string `json:"phase"`
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
	AgentCount     int    `json:"agentCount"`
	VerdictSummary string `json:"verdictSummary,omitempty"`
	// IssueCount / OpenIssueCount / DissentCount 让档案栏不必打开整场讨论，
	// 也能看出一场议题留下了多少分歧、其中多少仍待收口。
	IssueCount     int    `json:"issueCount,omitempty"`
	OpenIssueCount int    `json:"openIssueCount,omitempty"`
	DissentCount   int    `json:"dissentCount,omitempty"`
	Error          string `json:"error,omitempty"`
}
