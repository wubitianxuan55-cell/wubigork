package whisper

import (
)

// AuditMode represents the audit report mode
type AuditMode string

const (
	AuditStatsOnly  AuditMode = "stats_only"
	AuditCurated    AuditMode = "curated_audit"
	AuditSelfReport AuditMode = "self_report"
	AuditFullDump   AuditMode = "full_dump"
)

const (
	curatedMaxFacts    = 20
	curatedMinWeight   = 2.0
	curatedMinConf     = 0.65
	curatedMaxEpisodes = 5
	curatedMaxChars    = 2000
	fullDumpPageSize   = 40
)

// MemoryAuditStats contains statistics for the memory audit
type MemoryAuditStats struct {
	TotalFacts    int `json:"totalFacts"`
	TotalEpisodes int `json:"totalEpisodes"`
	CoreFacts     int `json:"coreFacts"`
	AvoidFacts    int `json:"avoidFacts"`
}

// MemoryAuditReport is the full audit report
type MemoryAuditReport struct {
	Mode        AuditMode           `json:"mode"`
	GeneratedAt string              `json:"generatedAt"`
	Stats       MemoryAuditStats    `json:"stats"`
	Facts       []MemoryFact        `json:"facts"`
	Timeline    []AuditTimelineItem `json:"timeline,omitempty"`
	Episodes    []string            `json:"episodes,omitempty"`
	DomainStats []AuditDomainStat   `json:"domainStats"`
	Page        int                 `json:"page,omitempty"`
	HasMore     bool                `json:"hasMore,omitempty"`
}

// AuditTimelineItem represents a timeline entry
type AuditTimelineItem struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Date  string `json:"date"`
}

// AuditDomainStat holds per-domain statistics
type AuditDomainStat struct {
	Domain string `json:"domain"`
	Label  string `json:"label"`
	Total  int    `json:"total"`
	Listed int    `json:"listed"`
}

var domainLabels = map[string]string{
	"IDENTITY":    "Identity",
	"SOCIAL":      "Social",
	"INNER_WORLD": "Inner World",
	"DAILY_LIFE":  "Daily Life",
	"TEMPORAL":    "Temporal",
	"KNOWLEDGE":   "Knowledge",
}

var subcatLabels = map[string]string{
	"BASIC_PROFILE": "Basic Profile",
	"FAMILY":        "Family",
	"TASTES":        "Tastes",
	"HEALTH":        "Health",
	"HABITS":        "Habits",
	"BELIEFS":       "Beliefs",
	"GOALS":         "Goals",
}







