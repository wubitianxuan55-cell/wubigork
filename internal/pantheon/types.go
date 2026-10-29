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
	AgentIDs        []string       `json:"agentIds"`
	ChairmanAgentID string         `json:"chairmanAgentId"`
	Rounds          int            `json:"rounds"`
	UntilConsensus  bool           `json:"untilConsensus,omitempty"`
	MaxRounds       int            `json:"maxRounds,omitempty"`
	ReadOnly        bool           `json:"readOnly"`
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
	Warnings      []string       `json:"warnings,omitempty"`
	Error         string         `json:"error,omitempty"`
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
