// Package proposal — Markdown 块解析（供 docx 渲染）
package proposal

import "strings"

type mdBlock struct {
	kind  string // heading / para / list / table / code
	level int
	text  string
	rows  [][]string
	lang  string
}

func parseMarkdownBlocks(md string) []mdBlock {
	lines := strings.Split(md, "\n")
	var blocks []mdBlock
	var cur *mdBlock
	flush := func() {
		if cur != nil {
			blocks = append(blocks, *cur)
			cur = nil
		}
	}
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			if cur != nil && cur.kind == "code" {
				flush()
				continue
			}
			flush()
			cur = &mdBlock{kind: "code", lang: strings.TrimPrefix(trimmed, "```")}
			continue
		}
		if cur != nil && cur.kind == "code" {
			cur.text += line + "\n"
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			flush()
			n := 0
			for n < len(trimmed) && trimmed[n] == '#' {
				n++
			}
			blocks = append(blocks, mdBlock{kind: "heading", level: n, text: strings.TrimSpace(trimmed[n:])})
			continue
		}
		if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "|") {
			cells := strings.Split(strings.Trim(trimmed, "|"), "|")
			var row []string
			for _, c := range cells {
				row = append(row, strings.TrimSpace(c))
			}
			allDash := true
			for _, c := range row {
				if strings.Trim(c, "-: ") != "" {
					allDash = false
					break
				}
			}
			if allDash {
				continue // 分隔行
			}
			if cur == nil || cur.kind != "table" {
				flush()
				cur = &mdBlock{kind: "table"}
			}
			cur.rows = append(cur.rows, row)
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			flush()
			blocks = append(blocks, mdBlock{kind: "list", text: strings.TrimSpace(trimmed[2:])})
			continue
		}
		if cur == nil || cur.kind != "para" {
			flush()
			cur = &mdBlock{kind: "para"}
		}
		if cur.text != "" {
			cur.text += "\n"
		}
		cur.text += line
	}
	flush()
	return blocks
}
