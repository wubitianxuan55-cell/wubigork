package authorprefs

import (
	"sort"
	"strings"
)

const QueryByteLimit = 2048

type QueryResult struct {
	Lines   []string     `json:"lines"`
	Text    string       `json:"text"`
	Omitted []string     `json:"omitted,omitempty"`
	Items   []Preference `json:"items"`
}

func mergeAccepted(ap, bp Snapshot, kind, scope string) []Preference {
	byKey := map[string]Preference{}
	rest := make([]Preference, 0)
	add := func(p Preference, override bool) {
		if p.Status != StatusAccepted {
			return
		}
		if kind != "" && p.Kind != kind {
			return
		}
		if scope != "" && p.Scope != scope {
			return
		}
		if p.Key != "" {
			if old, ok := byKey[p.Key]; !ok || override || old.ID == p.ID {
				byKey[p.Key] = p
			}
			return
		}
		rest = append(rest, p)
	}
	for _, p := range ap.Preferences {
		add(p, false)
	}
	for _, p := range bp.Preferences {
		add(p, true)
	}
	out := rest
	for _, p := range byKey {
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Importance != out[j].Importance {
			return out[i].Importance > out[j].Importance
		}
		if out[i].UpdatedAt != out[j].UpdatedAt {
			return out[i].UpdatedAt > out[j].UpdatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func renderQuery(items []Preference) QueryResult {
	res := QueryResult{Lines: []string{}, Items: items}
	var b strings.Builder
	for _, p := range items {
		line := "- [" + p.ID + "] " + strings.TrimSpace(p.Text)
		extra := len(line)
		if b.Len() > 0 {
			extra++
		}
		if b.Len()+extra > QueryByteLimit {
			res.Omitted = append(res.Omitted, p.ID)
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
		res.Lines = append(res.Lines, line)
	}
	res.Text = b.String()
	return res
}
