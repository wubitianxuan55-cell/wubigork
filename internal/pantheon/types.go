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
	EventQuestion   = "question"
	EventAnswer     = "answer"
	EventGate       = "gate"
	EventToolStart  = "tool_started"
	EventToolResult = "tool_result"
	EventVerdict    = "verdict"
	EventConsensus  = "consensus"
	EventSignOff    = "signoff"
	EventComplete   = "completed"
	EventError      = "error"
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
	// ParentSessionID links a follow-up council to the session it continues
	// (2026-10-31 裁决第三刀：续议保留父会话；承接内容以简报为准，不冒充继承全文).
	ParentSessionID string `json:"parentSessionId,omitempty"`
	// GateEachRound 要求每轮结束后停一次，等用户放行再进下一轮（M14 的决策门）。
	// 门没被显式解开之前，整场保持 waiting_input：宁可显示"停着等人"，也不
	// 制造一个 running 却没人干活的假象。
	GateEachRound bool `json:"gateEachRound,omitempty"`
}

type Participant struct {
	CallRef
	AgentID    string `json:"agentId"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	StartedAt  int64  `json:"startedAt,omitempty"`
	FinishedAt int64  `json:"finishedAt,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
	// Observed 是这条状态的出处与新鲜度证据；Evidence 是一句人能读的
	// "这条状态凭什么"，直接进 UI 悬停。Status 的词表见 Seat* 常量，
	// 转移合法性由 CanTransitionSeat 裁决。
	Observed Observed `json:"observed,omitempty"`
	Evidence string   `json:"evidence,omitempty"`
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
	// RunMode 记录本次调用实际施加的只读机制（裁决第三刀：能力不对称必须
	// 进每次调用记录，不能把四个标志写成同一个笼统的「已只读」）。
	RunMode    string `json:"runMode,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

type Verdict struct {
	Summary        string   `json:"summary"`
	Recommendation string   `json:"recommendation"`
	Conditions     []string `json:"conditions,omitempty"`
	Disagreements  []string `json:"disagreements,omitempty"`
	Risks          []string `json:"risks,omitempty"`
	EvidenceNotes  []string `json:"evidenceNotes,omitempty"`
	Chairman       string   `json:"chairman"`
	Partial        bool     `json:"partial"`
	CreatedAt      int64    `json:"createdAt"`
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
	Gaps  []string `json:"gaps,omitempty"`
	Error string   `json:"error,omitempty"`
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
	Error          string `json:"error,omitempty"`
}
