// Package office — archive.go
package office

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/whisper"
)

type factGroup struct {
	domain string
	subcat string
	facts  []whisper.MemoryFact
}

var archiveDomainZH = map[string]string{
	"IDENTITY": "self", "SOCIAL": "social", "DAILY_LIFE": "daily",
	"PURSUITS": "pursuits", "INNER_WORLD": "inner", "TEMPORAL": "temporal",
}

var archiveSubcatZH = map[string]string{
	"BASIC_PROFILE": "profile", "LIFE_STORY": "story", "VALUES_BELIEFS": "values",
	"SELF_PERCEPTION": "self", "OUR_BOND": "bond", "FAMILY": "family", "FRIENDS": "friends",
	"PARTNER": "gaea", "ROUTINES": "routines", "HEALTH": "health", "LIVING_SPACE": "living",
	"LIFESTYLE": "lifestyle", "CAREER": "career", "LEARNING": "learning", "GOALS": "goals",
	"PROJECTS": "projects", "PROCEDURES": "procedures", "TASTES": "tastes",
	"COMMITMENTS": "commitments", "PLANS": "plans", "MEMORIES": "memories",
}

type ExportStats struct {
	FilesWritten     int `json:"filesWritten"`
	FactsExported    int `json:"factsExported"`
	EpisodesExported int `json:"episodesExported"`
	CoreCount        int `json:"coreCount"`
}

func ExportMemoryArchive(dataRoot string, fs *whisper.FactStore, episodes []string) ExportStats {
	archiveDir := filepath.Join(dataRoot, "memory", "archive")
	os.MkdirAll(archiveDir, 0755)
	allFacts := fs.ListActive()
	coreFacts := fs.SelectCoreFacts(9999)
	stats := ExportStats{CoreCount: len(coreFacts)}
	groups := make(map[string]*factGroup)
	for _, f := range allFacts {
		key := f.Domain + "::" + f.Subcategory
		if g, ok := groups[key]; ok { g.facts = append(g.facts, f.MemoryFact) } else {
			groups[key] = &factGroup{domain: f.Domain, subcat: f.Subcategory, facts: []whisper.MemoryFact{f.MemoryFact}}
		}
	}
	var sortedKeys []string
	for k := range groups { sortedKeys = append(sortedKeys, k) }
	sort.Strings(sortedKeys)
	var domainFiles []string
	for _, key := range sortedKeys {
		g := groups[key]
		md := formatFactGroupMD(g)
		if md == "" { continue }
		dName := archiveDomainZH[g.domain]
		if dName == "" { dName = g.domain }
		sName := archiveSubcatZH[g.subcat]
		if sName == "" { sName = g.subcat }
		domainDir := filepath.Join(archiveDir, dName)
		os.MkdirAll(domainDir, 0755)
		filename := filepath.Join(domainDir, sName+".md")
		os.WriteFile(filename, []byte(md), 0644)
		domainFiles = append(domainFiles, filename)
		stats.FilesWritten++
		stats.FactsExported += len(g.facts)
	}
	if len(episodes) > 0 {
		epFile := filepath.Join(archiveDir, "episodes.md")
		os.WriteFile(epFile, []byte(formatEpisodesMD(episodes)), 0644)
		stats.FilesWritten++; stats.EpisodesExported = len(episodes)
	}
	idxFile := filepath.Join(archiveDir, "_index.md")
	os.WriteFile(idxFile, []byte(formatArchiveIndexMD(domainFiles, stats)), 0644)
	stats.FilesWritten++
	return stats
}

func formatFactGroupMD(g *factGroup) string {
	if len(g.facts) == 0 { return "" }
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s · %s\n\n> %d facts\n\n", g.domain, g.subcat, len(g.facts)))
	for _, f := range g.facts {
		core := ""
		if f.Tier == "core" { core = " ★" }
		sb.WriteString(fmt.Sprintf("- [%.0f%%] %s%s\n", f.Confidence*100, f.Summary, core))
	}
	return sb.String()
}

func formatEpisodesMD(episodes []string) string {
	var sb strings.Builder
	sb.WriteString("# Episodes\n\n")
	for i, ep := range episodes { sb.WriteString(fmt.Sprintf("## %d\n\n%s\n\n", i+1, ep)) }
	return sb.String()
}

func formatArchiveIndexMD(files []string, stats ExportStats) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Archive Index\n\n- Generated: %s\n- Facts: %d\n- Core: %d\n- Episodes: %d\n- Files: %d\n\n## Files\n\n",
		time.Now().Format("2006-01-02 15:04"), stats.FactsExported, stats.CoreCount, stats.EpisodesExported, stats.FilesWritten))
	for _, f := range files { rel, _ := filepath.Rel(filepath.Dir(f), f); sb.WriteString(fmt.Sprintf("- [%s](%s)\n", filepath.Base(f), rel)) }
	return sb.String()
}
