package authorprefs

import "time"

type Source string

const (
	SourceExplicitUser       Source = "explicit_user"
	SourceAcceptedSuggestion Source = "accepted_suggestion"
	SourceManual             Source = "manual"
)

type Status string

const (
	StatusAccepted Status = "accepted"
	StatusPending  Status = "pending"
	StatusConflict Status = "conflict"
)

type Preference struct {
	ID         string   `json:"id"`
	Key        string   `json:"key,omitempty"`
	Kind       string   `json:"kind"`
	Scope      string   `json:"scope"` // global | genre | workflow | book
	Text       string   `json:"text"`
	Evidence   []string `json:"evidence,omitempty"`
	Source     Source   `json:"source"`
	Status     Status   `json:"status"`
	Importance int      `json:"importance,omitempty"`
	Revision   int64    `json:"revision"`
	UpdatedAt  string   `json:"updatedAt"`
}

type Event struct {
	ID         string     `json:"id"`
	Action     string     `json:"action"` // remember | decide | replace | forget
	Preference Preference `json:"preference,omitempty"`
	TargetID   string     `json:"targetId,omitempty"`
	Accept     bool       `json:"accept,omitempty"`
	Text       string     `json:"text,omitempty"`
}

type Snapshot struct {
	Version       int               `json:"version"`
	Revision      int64             `json:"revision"`
	Preferences   []Preference      `json:"preferences"`
	AppliedEvents map[string]string `json:"appliedEvents"`
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339Nano) }
