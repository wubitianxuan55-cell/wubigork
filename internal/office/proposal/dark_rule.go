// Package proposal — 暗标规则库
package proposal

import (
	"encoding/json"
	"strings"
	"unicode"
)

// DarkRuleOptions 暗标格式限制
type DarkRuleOptions struct {
	NoBold       bool `json:"noBold"`
	NoItalic     bool `json:"noItalic"`
	NoUnderline  bool `json:"noUnderline"`
	NoColor      bool `json:"noColor"`
	NoEmoji      bool `json:"noEmoji"`
	NoSpecial    bool `json:"noSpecial"`
	NoEmptyLines bool `json:"noEmptyLines"`
}

// DarkRule 暗标规则
type DarkRule struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Options     DarkRuleOptions `json:"options"`
	Enabled     bool            `json:"enabled"`
}

// Apply 按规则清理 Markdown 内容
func (r DarkRule) Apply(md string) string {
	if r.Options.NoBold {
		md = strings.ReplaceAll(md, "**", "")
		md = strings.ReplaceAll(md, "__", "")
	}
	if r.Options.NoItalic {
		md = strings.ReplaceAll(md, "*", "")
		md = strings.ReplaceAll(md, "_", "")
	}
	if r.Options.NoUnderline {
		md = strings.ReplaceAll(md, "<u>", "")
		md = strings.ReplaceAll(md, "</u>", "")
		md = strings.ReplaceAll(md, "~~", "")
	}
	if r.Options.NoEmoji {
		md = stripEmoji(md)
	}
	if r.Options.NoSpecial {
		md = stripSpecial(md)
	}
	if r.Options.NoEmptyLines {
		lines := strings.Split(md, "\n")
		var out []string
		prevEmpty := false
		for _, l := range lines {
			empty := strings.TrimSpace(l) == ""
			if empty && prevEmpty {
				continue
			}
			out = append(out, l)
			prevEmpty = empty
		}
		md = strings.Join(out, "\n")
	}
	return md
}

func stripEmoji(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0x1F300 && r <= 0x1FAFF || r >= 0x2600 && r <= 0x27BF {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func stripSpecial(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func marshalOptions(o DarkRuleOptions) string {
	data, _ := json.Marshal(o)
	return string(data)
}

// DefaultDarkRules 内置暗标规则
func DefaultDarkRules() []DarkRule {
	return []DarkRule{{
		ID: "soil-dark", Name: "土壤修复通用暗标规则",
		Description: "全文无加粗/斜体/下划线/彩色/emoji/特殊符号，压缩多余空行",
		Options: DarkRuleOptions{
			NoBold: true, NoItalic: true, NoUnderline: true, NoColor: true,
			NoEmoji: true, NoSpecial: true, NoEmptyLines: true,
		},
		Enabled: true,
	}}
}
