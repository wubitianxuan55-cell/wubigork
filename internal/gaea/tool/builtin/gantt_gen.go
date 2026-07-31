package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(ganttGen{}) }

// ganttGen generates a Mermaid Gantt diagram from task data.
type ganttGen struct{}

func (ganttGen) Name() string { return "gantt_gen" }

func (ganttGen) Description() string {
	return "根据任务数组生成 Mermaid 甘特图 Markdown 代码块。支持任务名称、开始日期、结束日期和依赖关系。"
}

func (ganttGen) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "title":{"type":"string","description":"甘特图标题（可选）"},
  "tasks":{
    "type":"array",
    "description":"任务列表",
    "items":{
      "type":"object",
      "properties":{
        "name":{"type":"string","description":"任务名称"},
        "start":{"type":"string","description":"开始日期，格式 YYYY-MM-DD 或相对值如 'today', 'd1'"},
        "end":{"type":"string","description":"结束日期，格式 YYYY-MM-DD 或相对工期如 '+5d', '3d'"},
        "duration":{"type":"string","description":"工期（替代end），如 '5d', '2w', '1m'"},
        "depends":{"type":"string","description":"依赖的上一个任务名称（可选）"},
        "section":{"type":"string","description":"所属分组（可选）"}
      },
      "required":["name"]
    }
  }
},
"required":["tasks"]
}`)
}

func (ganttGen) ReadOnly() bool { return true }

func (ganttGen) CompactDescription() string     { return compactDesc["gantt_gen"] }
func (ganttGen) CompactSchema() json.RawMessage { return compactSchema["gantt_gen"] }

type ganttTask struct {
	Name     string `json:"name"`
	Start    string `json:"start,omitempty"`
	End      string `json:"end,omitempty"`
	Duration string `json:"duration,omitempty"`
	Depends  string `json:"depends,omitempty"`
	Section  string `json:"section,omitempty"`
}

func (ganttGen) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Title string      `json:"title,omitempty"`
		Tasks []ganttTask `json:"tasks"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if len(p.Tasks) == 0 {
		return "", fmt.Errorf("tasks array must not be empty")
	}

	var b strings.Builder
	b.WriteString("```mermaid\n")
	b.WriteString("gantt\n")

	title := p.Title
	if title == "" {
		title = "项目甘特图"
	}
	fmt.Fprintf(&b, "  title %s\n", title)
	b.WriteString("  dateFormat  YYYY-MM-DD\n")
	b.WriteString("  axisFormat  %m-%d\n")

	// Try to determine a base date from tasks' start values
	baseDate := time.Now().Truncate(24 * time.Hour)
	for _, t := range p.Tasks {
		if t.Start != "" && !isRelativeDate(t.Start) {
			if parsed, err := time.Parse("2006-01-02", t.Start); err == nil {
				if parsed.Before(baseDate) {
					baseDate = parsed
				}
			}
		}
	}
	fmt.Fprintf(&b, "  baseDate %s\n\n", baseDate.Format("2006-01-02"))

	for i, t := range p.Tasks {
		section := strings.TrimSpace(t.Section)

		// Determine the start string for Mermaid
		startStr := t.Start
		if startStr == "" || startStr == "today" {
			startStr = baseDate.Format("2006-01-02")
			if i > 0 {
				// Default to previous task's end or start
				startStr = "after " + sanitizeTaskName(p.Tasks[i-1].Name)
			}
		} else if isRelativeDate(startStr) {
			// Convert relative like "+1d", "d1" to "after <prev>"
			startStr = "after " + sanitizeTaskName(t.Depends)
			if t.Depends == "" && i > 0 {
				startStr = "after " + sanitizeTaskName(p.Tasks[i-1].Name)
			}
		}

		// Determine end or duration
		endStr := t.End
		if endStr == "" {
			endStr = t.Duration
		}
		if endStr == "" {
			endStr = "1d"
		}
		// Convert durations like "3d", "2w", "1m" to Mermaid-compatible end dates
		if strings.HasSuffix(endStr, "d") || strings.HasSuffix(endStr, "w") || strings.HasSuffix(endStr, "m") {
			// For Mermaid, we can use relative durations via "after" syntax
			// Actually Mermaid supports: task_name, start_date, end_date
			// For duration-based, we'll try to compute an end date from start
			if parsedStart, err := time.Parse("2006-01-02", startStr); err == nil {
				if strings.HasSuffix(endStr, "d") {
					d := parseDuration(endStr[:len(endStr)-1])
					endStr = parsedStart.AddDate(0, 0, d).Format("2006-01-02")
				} else if strings.HasSuffix(endStr, "w") {
					d := parseDuration(endStr[:len(endStr)-1])
					endStr = parsedStart.AddDate(0, 0, d*7).Format("2006-01-02")
				} else if strings.HasSuffix(endStr, "m") {
					d := parseDuration(endStr[:len(endStr)-1])
					endStr = parsedStart.AddDate(0, d, 0).Format("2006-01-02")
				}
			} else {
				// Can't parse start as date, use absolute duration from base
				if strings.HasSuffix(endStr, "d") {
					d := parseDuration(endStr[:len(endStr)-1])
					endStr = baseDate.AddDate(0, 0, d).Format("2006-01-02")
				} else if strings.HasSuffix(endStr, "w") {
					d := parseDuration(endStr[:len(endStr)-1])
					endStr = baseDate.AddDate(0, 0, d*7).Format("2006-01-02")
				} else if strings.HasSuffix(endStr, "m") {
					d := parseDuration(endStr[:len(endStr)-1])
					endStr = baseDate.AddDate(0, d, 0).Format("2006-01-02")
				}
			}
		}

		// Mermaid syntax: <section>, <name>, <start>, <end>
		sanitizedName := sanitizeTaskName(t.Name)
		if section != "" {
			fmt.Fprintf(&b, "  section %s\n", section)
		}
		fmt.Fprintf(&b, "  %s : %s, %s\n", sanitizedName, startStr, endStr)
	}

	b.WriteString("```\n")
	return b.String(), nil
}

func sanitizeTaskName(name string) string {
	// Replace characters that might confuse Mermaid
	r := strings.NewReplacer("\"", "'", ":", " -", "\n", " ", "\r", "")
	return r.Replace(name)
}

// isRelativeDate checks if a date string is relative (not an absolute YYYY-MM-DD).
func isRelativeDate(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == "today" || s == "now" {
		return true
	}
	_, err := time.Parse("2006-01-02", s)
	return err != nil
}

func parseDuration(s string) int {
	var d int
	fmt.Sscanf(s, "%d", &d)
	if d <= 0 {
		return 1
	}
	return d
}
