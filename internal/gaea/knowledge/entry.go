// Package knowledge provides a file-based knowledge base for engineering
// knowledge entries with YAML-like frontmatter and Markdown body.
package knowledge

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Entry categories.
const (
	CatStandard   = "规范标准"
	CatCase       = "工程案例"
	CatExperience = "经验总结"
	CatMaterial   = "材料工艺"
	CatRegulation = "法规政策"
	CatSurvey     = "调查报告"
	CatDesign     = "设计方案"
	CatOther      = "其他"
)

// Entry represents a single knowledge entry with frontmatter metadata and
// Markdown body.
type Entry struct {
	Name       string
	Title      string
	Category   string
	Phase      string
	Discipline string
	Tags       []string
	Status     string
	Version    int
	Author     string
	Reviewer   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Source     string
	Body       string
}

// ParseFrontmatter parses a complete file string (---\n...\n---\nbody) into
// an Entry. Returns error if the frontmatter is malformed.
func ParseFrontmatter(data string) (*Entry, error) {
	e := &Entry{}

	// Find opening fence.
	lines := strings.SplitN(data, "\n", 2)
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		// No frontmatter — treat whole content as body.
		e.Body = data
		return e, nil
	}

	// Find closing fence.
	rest := lines[1]
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		// No closing fence — treat as body.
		e.Body = data
		return e, nil
	}

	fmStr := rest[:idx]
	bodyStr := rest[idx+5:] // skip "\n---\n"

	e.Body = strings.TrimLeft(bodyStr, "\n")

	// Parse each line of frontmatter.
	fmLines := strings.Split(fmStr, "\n")
	var key, listAccum []string
	for _, line := range fmLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for list continuation (- item).
		if strings.HasPrefix(line, "- ") && len(key) > 0 {
			listAccum = append(listAccum, strings.TrimPrefix(line, "- "))
			continue
		}

		// Flush previous accumulated list.
		if len(listAccum) > 0 {
			setField(e, strings.Join(key, "/"), strings.Join(listAccum, ","))
			listAccum = nil
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"'`)

		if v == "" {
			// Could be a list header or section header.
			key = append(key, strings.ToLower(k))
			continue
		}

		// Single value.
		key = []string{strings.ToLower(k)}
		setField(e, key[0], v)
		key = nil
	}

	// Flush any remaining list.
	if len(listAccum) > 0 {
		setField(e, strings.Join(key, "/"), strings.Join(listAccum, ","))
	}

	return e, nil
}

// RenderFrontmatter renders an Entry's metadata as YAML-like frontmatter
// suitable for writing to a .md file.
func RenderFrontmatter(e Entry) string {
	var b strings.Builder
	b.WriteString("---\n")
	writeFmField(&b, "name", e.Name)
	writeFmField(&b, "title", e.Title)
	writeFmField(&b, "category", e.Category)
	writeFmField(&b, "phase", e.Phase)
	writeFmField(&b, "discipline", e.Discipline)
	if len(e.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, t := range e.Tags {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	}
	writeFmField(&b, "status", e.Status)
	if e.Version > 0 {
		fmt.Fprintf(&b, "version: %d\n", e.Version)
	}
	writeFmField(&b, "author", e.Author)
	writeFmField(&b, "reviewer", e.Reviewer)
	if !e.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "created_at: %s\n", e.CreatedAt.Format(time.RFC3339))
	}
	if !e.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, "updated_at: %s\n", e.UpdatedAt.Format(time.RFC3339))
	}
	writeFmField(&b, "source", e.Source)
	b.WriteString("---\n")
	return b.String()
}

// FileName returns the filename (with .md extension) for the entry.
func FileName(e Entry) string {
	return safeFileName(e.Name) + ".md"
}

// safeFileName converts a name to a safe filename.
func safeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// writeFmField writes "key: value\n" if value is non-empty.
func writeFmField(b *strings.Builder, key, value string) {
	if value != "" {
		fmt.Fprintf(b, "%s: %s\n", key, value)
	}
}

// setField sets a field on Entry based on the frontmatter key.
func setField(e *Entry, key, value string) {
	switch key {
	case "name":
		e.Name = value
	case "title":
		e.Title = value
	case "category":
		e.Category = value
	case "phase":
		e.Phase = value
	case "discipline":
		e.Discipline = value
	case "tags":
		// Tags are comma-separated from YAML list conversion.
		var tags []string
		for _, t := range strings.Split(value, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
		e.Tags = tags
	case "status":
		e.Status = value
	case "version":
		fmt.Sscanf(value, "%d", &e.Version)
	case "author":
		e.Author = value
	case "reviewer":
		e.Reviewer = value
	case "created_at":
		e.CreatedAt, _ = time.Parse(time.RFC3339, value)
	case "updated_at":
		e.UpdatedAt, _ = time.Parse(time.RFC3339, value)
	case "source":
		e.Source = value
	}
}

// EntrySummary is a lightweight view of an Entry (without Body).
type EntrySummary struct {
	Name      string
	Title     string
	Category  string
	Tags      []string
	Status    string
	UpdatedAt time.Time
}

// ToSummary creates an EntrySummary from an Entry.
func (e Entry) ToSummary() EntrySummary {
	return EntrySummary{
		Name:      e.Name,
		Title:     e.Title,
		Category:  e.Category,
		Tags:      e.Tags,
		Status:    e.Status,
		UpdatedAt: e.UpdatedAt,
	}
}

// SortEntrySummaries sorts summaries by UpdatedAt descending.
func SortEntrySummaries(list []EntrySummary) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})
}
