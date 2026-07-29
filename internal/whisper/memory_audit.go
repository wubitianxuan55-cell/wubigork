package whisper

import (
	"fmt"
	"sort"
	"strings"
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

// BuildMemoryAuditReport builds a memory audit report from the FactStore
func BuildMemoryAuditReport(
	fs *FactStore,
	episodeCount int,
	mode AuditMode,
	includeAvoid bool,
	page int,
) *MemoryAuditReport {
	if fs == nil {
		return &MemoryAuditReport{Mode: mode}
	}

	allFactsPtr := fs.ListActive()
	allFacts := factsToMemoryFacts(allFactsPtr)
	coreFacts := fs.SelectCoreFacts(9999)
	avoidCount := 0

	stats := MemoryAuditStats{
		TotalFacts:    len(allFacts),
		TotalEpisodes: episodeCount,
		CoreFacts:     len(coreFacts),
	}

	report := &MemoryAuditReport{
		Mode:  mode,
		Stats: stats,
	}

	var selectedFacts []MemoryFact
	switch mode {
	case AuditStatsOnly:
		// stats only, no facts
	case AuditCurated, AuditSelfReport:
		selectedFacts = curateFacts(allFacts, curatedMaxFacts)
	case AuditFullDump:
		start := page * fullDumpPageSize
		end := start + fullDumpPageSize
		if start >= len(allFacts) {
			start = 0
		}
		if end > len(allFacts) {
			end = len(allFacts)
		}
		selectedFacts = allFacts[start:end]
		report.Page = page
		report.HasMore = end < len(allFacts)
	}

	// Filter avoid (sensitive facts)
	if !includeAvoid {
		var filtered []MemoryFact
		for _, f := range selectedFacts {
			if f.Sensitivity == "avoid" {
				avoidCount++
				continue
			}
			filtered = append(filtered, f)
		}
		selectedFacts = filtered
	}
	stats.AvoidFacts = avoidCount

	report.Facts = selectedFacts
	report.Timeline = buildAuditTimeline(allFacts)
	report.DomainStats = buildDomainStats(allFacts, selectedFacts)

	return report
}

// curateFacts selects the most relevant facts up to maxCount
func curateFacts(facts []MemoryFact, maxCount int) []MemoryFact {
	if len(facts) <= maxCount {
		return facts
	}

	type scored struct {
		fact  MemoryFact
		score float64
	}
	var scoredFacts []scored
	for _, f := range facts {
		s := scoreFact(f)
		scoredFacts = append(scoredFacts, scored{f, s})
	}
	sort.Slice(scoredFacts, func(i, j int) bool {
		return scoredFacts[i].score > scoredFacts[j].score
	})

	// Pick core tier first
	var selected []MemoryFact
	domainPicked := map[string]bool{}
	for _, s := range scoredFacts {
		if len(selected) >= maxCount {
			break
		}
		if s.fact.Tier == "core" {
			selected = append(selected, s.fact)
			domainPicked[s.fact.Domain] = true
		}
	}

	// Fill by score with domain diversity
	for _, s := range scoredFacts {
		if len(selected) >= maxCount {
			break
		}
		alreadySelected := false
		for _, sel := range selected {
			if sel.ID == s.fact.ID {
				alreadySelected = true
				break
			}
		}
		if alreadySelected {
			continue
		}
		if !domainPicked[s.fact.Domain] {
			selected = append(selected, s.fact)
			domainPicked[s.fact.Domain] = true
			continue
		}
		selected = append(selected, s.fact)
	}

	return selected
}

// scoreFact computes a relevance score for a fact
func scoreFact(f MemoryFact) float64 {
	s := f.Weight * f.Confidence
	if f.Tier == "core" {
		s += 2
	}
	return s
}

// buildAuditTimeline extracts timeline items from facts
func buildAuditTimeline(facts []MemoryFact) []AuditTimelineItem {
	var tl []AuditTimelineItem
	for _, f := range facts {
		if f.AgeMeta != nil && f.AgeMeta.BirthdayMMDD != "" {
			tl = append(tl, AuditTimelineItem{
				Type:  "birthday",
				Label: f.Subject,
				Date:  f.AgeMeta.BirthdayMMDD,
			})
		}
	}
	return tl
}

// buildDomainStats computes per-domain statistics
func buildDomainStats(allFacts, selectedFacts []MemoryFact) []AuditDomainStat {
	dt, dl := map[string]int{}, map[string]int{}
	for _, f := range allFacts {
		dt[f.Domain]++
	}
	for _, f := range selectedFacts {
		dl[f.Domain]++
	}
	order := []string{"IDENTITY", "SOCIAL", "INNER_WORLD", "DAILY_LIFE", "TEMPORAL", "KNOWLEDGE"}
	var stats []AuditDomainStat
	for _, domain := range order {
		if dt[domain] > 0 || dl[domain] > 0 {
			label := domainLabels[domain]
			if label == "" {
				label = domain
			}
			stats = append(stats, AuditDomainStat{
				Domain: domain,
				Label:  label,
				Total:  dt[domain],
				Listed: dl[domain],
			})
		}
	}
	return stats
}


// FormatMemoryAuditMarkdown formats the audit report as markdown text
func FormatMemoryAuditMarkdown(report *MemoryAuditReport, mode AuditMode) string {
	if report == nil {
		return "[no report]"
	}

	var sb strings.Builder

	switch mode {
	case AuditStatsOnly:
		sb.WriteString("## Memory Audit (Stats)\n\n")
	case AuditCurated:
		sb.WriteString("## Memory Audit (Curated)\n\n")
	case AuditSelfReport:
		sb.WriteString("## Memory Self-Report\n\n")
	case AuditFullDump:
		sb.WriteString("## Memory Full Dump\n\n")
	}

	sb.WriteString(fmt.Sprintf("[total] %d facts", report.Stats.TotalFacts))
	if report.Stats.CoreFacts > 0 {
		sb.WriteString(fmt.Sprintf(" (core: %d)", report.Stats.CoreFacts))
	}
	if report.Stats.TotalEpisodes > 0 {
		sb.WriteString(fmt.Sprintf(" [episodes] %d", report.Stats.TotalEpisodes))
	}
	sb.WriteString("\n\n")

	if len(report.DomainStats) > 0 {
		sb.WriteString("[Domains]\n")
		for _, ds := range report.DomainStats {
			sb.WriteString(fmt.Sprintf("  %s: %d items", ds.Label, ds.Total))
			if ds.Listed > 0 && ds.Listed != ds.Total {
				sb.WriteString(fmt.Sprintf(" (listed: %d)", ds.Listed))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(report.Facts) > 0 {
		sb.WriteString("[Memory Items]\n")
		for i, f := range report.Facts {
			domain := domainLabels[f.Domain]
			if domain == "" {
				domain = f.Domain
			}
			subcat := subcatLabels[f.Subcategory]
			if subcat == "" {
				subcat = f.Subcategory
			}
			core := ""
			if f.Tier == "core" {
				core = " *"
			}
			sb.WriteString(fmt.Sprintf("%d. [%s/%s] %s%s\n",
				i+1, domain, subcat, f.Summary, core))
		}
	}

	if len(report.Timeline) > 0 {
		sb.WriteString("\n[Key Dates]\n")
		for _, t := range report.Timeline {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", t.Label, t.Date))
		}
	}

	if report.HasMore {
		sb.WriteString(fmt.Sprintf("\n(page %d, more available)", report.Page+1))
	}

	result := sb.String()
	if (mode == AuditCurated || mode == AuditSelfReport || mode == AuditStatsOnly) &&
		len([]rune(result)) > curatedMaxChars {
		result = string([]rune(result)[:curatedMaxChars]) + "\n..."
	}

	return result
}

// factsToMemoryFacts converts []*Fact to []MemoryFact
func factsToMemoryFacts(facts []*Fact) []MemoryFact {
	result := make([]MemoryFact, len(facts))
	for i, f := range facts {
		result[i] = f.MemoryFact
	}
	return result
}
