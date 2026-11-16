package agent

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
)

// toolSources extracts display-only evidence metadata from the exact tool
// output already sent to the model. It never changes prompt content.
func toolSources(name, args, output string) []event.ToolSource {
	switch name {
	case "web_search":
		if sources := parseWebSearchSources(output); len(sources) > 0 {
			return sources
		}
		if u := toolURL(args); u != "" {
			return []event.ToolSource{{Title: u, URL: u, Source: "web_search"}}
		}
	case "web_fetch":
		if u := toolURL(args); u != "" {
			return []event.ToolSource{{Title: u, URL: u, Source: "web_fetch"}}
		}
	}
	return nil
}

func successfulToolSources(name, args, output, errMsg string) []event.ToolSource {
	if strings.TrimSpace(errMsg) != "" || hasToolErrorPrefix(output) {
		return nil
	}
	return toolSources(name, args, output)
}

func parseWebSearchSources(output string) []event.ToolSource {
	var rows []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
		Source  string `json:"source"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &rows); err != nil {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]event.ToolSource, 0, len(rows))
	for _, row := range rows {
		raw := strings.TrimSpace(row.URL)
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			continue
		}
		key := strings.ToLower(parsed.Scheme + "://" + parsed.Host + strings.TrimRight(parsed.Path, "/"))
		if parsed.RawQuery != "" {
			key += "?" + parsed.RawQuery
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		title := strings.TrimSpace(row.Title)
		if title == "" {
			title = parsed.Hostname()
		}
		out = append(out, event.ToolSource{
			Title:   title,
			URL:     raw,
			Snippet: strings.TrimSpace(row.Snippet),
			Source:  strings.TrimSpace(row.Source),
		})
	}
	return out
}

func toolURL(rawArgs string) string {
	var args struct {
		URL string `json:"url"`
	}
	if json.Unmarshal([]byte(rawArgs), &args) != nil {
		return ""
	}
	raw := strings.TrimSpace(args.URL)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return raw
}
