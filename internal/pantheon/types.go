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

	PhasePreparing    = "preparing"
	PhaseOpening      = "opening"
	PhaseDiscussion   = "discussion"
	PhaseSynthesis    = "synthesis"
	PhaseIndependent  = "independent"
	PhaseReview       = "review"
	PhaseChallenge    = "challenge"
	PhaseVerdict      = "verdict"
	PhaseCompleted    = "completed"
	PhaseFailed       = "failed"
	PhaseCancelled    = "cancelled"
	EventSessionStart = "session_started"
	EventPhase        = "phase"
	EventAgentStart   = "agent_started"
	EventAgentDelta   = "agent_delta"
	EventAgentOutput  = "agent_output"
	EventAgentError   = "agent_error"
	EventAgentSkipped = "agent_skipped"
	EventAgentRetry   = "agent_retry"
	EventToolStart    = "tool_started"
	EventToolResult   = "tool_result"
	EventVerdict      = "verdict"
	EventConsensus    = "consensus"
	EventSignOff      = "signoff"
	EventComplete     = "completed"
	EventError        = "error"
)

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
}

type Participant struct {
	AgentID    string `json:"agentId"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	StartedAt  int64  `json:"startedAt,omitempty"`
	FinishedAt int64  `json:"finishedAt,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Event is both the live UI stream payload and the durable session timeline.
type Event struct {
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
	ID         string `json:"id"`
	Phase      string `json:"phase"`
	Round      int    `json:"round,omitempty"`
	AgentID    string `json:"agentId"`
	AgentName  string `json:"agentName"`
	AnonLabel  string `json:"anonLabel,omitempty"`
	Content    string `json:"content"`
	Confidence string `json:"confidence,omitempty"`
	Consensus  string `json:"consensus,omitempty"`
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
	// 用户确认的是哪一版，签注与对外交付物不能是两份真相）。
	VerdictSHA256 string `json:"verdictSha256,omitempty"`
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
	Error         string         `json:"error,omitempty"`
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
