// Package filefilter is the shared explorer exclusion and compact-folder
// authority. It intentionally deals only in workspace-relative paths; callers
// remain responsible for resolving and fencing those paths.
package filefilter

import (
	"regexp"
	"strings"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

const (
	PatternLimit     = 64
	PatternMaxLength = 256
)

type patternKind uint8

const (
	patternName patternKind = iota
	patternPath
	patternSuffix
)

type compiledPattern struct {
	kind patternKind
	re   *regexp.Regexp
}

// Matcher applies gaea's fixed noise list plus an optional user glob list.
type Matcher struct {
	user []compiledPattern
}

// Compile normalizes user patterns and always returns a matcher carrying the
// fixed noise rules. Invalid patterns are dropped rather than breaking a tree.
func Compile(raw []string) *Matcher {
	m := &Matcher{}
	for _, item := range raw {
		if len(m.user) >= PatternLimit {
			break
		}
		p, ok := normalizePattern(item)
		if !ok {
			continue
		}
		re, err := globToRegexp(p.body)
		if err != nil {
			continue
		}
		m.user = append(m.user, compiledPattern{kind: p.kind, re: re})
	}
	return m
}

// Match reports whether one workspace-relative entry should be hidden.
// Windows path semantics are case-insensitive, matching gaea's desktop host.
func (m *Matcher) Match(rel, name string) bool {
	if IsDefaultNoise(name) {
		return true
	}
	if m == nil || len(m.user) == 0 {
		return false
	}
	rel = normalizeRel(rel)
	name = strings.ToLower(strings.TrimSpace(name))
	for _, p := range m.user {
		switch p.kind {
		case patternName:
			if p.re.MatchString(name) {
				return true
			}
		case patternPath:
			if rel != "" && p.re.MatchString(rel) {
				return true
			}
		case patternSuffix:
			parts := strings.Split(rel, "/")
			if p.re.MatchString(name) {
				return true
			}
			for i := 0; i < len(parts)-1; i++ {
				if p.re.MatchString(strings.Join(parts[i:], "/")) {
					return true
				}
			}
		}
	}
	return false
}

// IsDefaultNoise is the fixed part of the explorer exclusion policy.
func IsDefaultNoise(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if wspath.IsSkipped(n) || strings.HasPrefix(n, ".tmp") {
		return true
	}
	switch n {
	case ".ds_store", "thumbs.db", "desktop.ini":
		return true
	}
	return false
}

type normalizedPattern struct {
	kind patternKind
	body string
}

func normalizePattern(raw string) (normalizedPattern, bool) {
	p := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if p == "" || len(p) > PatternMaxLength {
		return normalizedPattern{}, false
	}
	for strings.HasPrefix(p, "./") {
		p = p[2:]
	}
	p = strings.TrimLeft(p, "/")
	p = strings.TrimRight(p, "/")
	kind := patternName
	if strings.HasPrefix(p, "**/") {
		kind = patternSuffix
		p = strings.TrimPrefix(p, "**/")
	}
	if p == "" || p == "**" {
		return normalizedPattern{}, false
	}
	if kind == patternName && strings.Contains(p, "/") {
		kind = patternPath
	}
	return normalizedPattern{kind: kind, body: strings.ToLower(p)}, true
}

func normalizeRel(rel string) string {
	rel = strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/")
	rel = strings.Trim(rel, "/")
	for strings.HasPrefix(rel, "./") {
		rel = rel[2:]
	}
	return strings.ToLower(rel)
}

func globToRegexp(body string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString(`(?i)^`)
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '*':
			if i+1 < len(body) && body[i+1] == '*' {
				if i+2 < len(body) && body[i+2] == '/' {
					b.WriteString(`(?:.*/)?`)
					i += 2
				} else {
					b.WriteString(`.*`)
					i++
				}
			} else {
				b.WriteString(`[^/]*`)
			}
		case '?':
			b.WriteString(`[^/]`)
		default:
			b.WriteString(regexp.QuoteMeta(string(body[i])))
		}
	}
	b.WriteString(`$`)
	return regexp.Compile(b.String())
}
