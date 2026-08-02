package cache

import "strings"

// TaskKind classifies the type of task for tool filtering and context injection.
// Originally in router.go (V3.0); kept for skill/spawn/runtime compatibility.
type TaskKind string

const (
	KindFixBug       TaskKind = "fix_bug"
	KindWriteFeature TaskKind = "write_feature"
	KindReview       TaskKind = "review"
	KindExplain      TaskKind = "explain"
	KindResearch     TaskKind = "research"
	KindDefault      TaskKind = "default"
	// V4.0: non-code task kinds
	KindDataAnalysis TaskKind = "data_analysis"
	KindWriting      TaskKind = "writing"
	KindGeneral      TaskKind = "general"
	// V10.XX: simple query — short input with read-only keywords;
	// detected before other classifications so the agent can answer directly.
	KindSimple TaskKind = "simple"
)

// DomainConfig holds the routing result: classified kind and tool filter set.
// V6.0: simplified from original router.go DomainConfig.
type DomainConfig struct {
	Kind         TaskKind
	SkillFilter  []string
	ContextFocus []string
}

// Tool lists for skill layer classification (originally in router.go).
var readTools = []string{
	"read_file", "grep", "glob", "ls",
	"web_search", "web_fetch", "preview",
}

var editTools = []string{
	"edit_file", "multi_edit", "write_file",
	"delete_range", "delete_symbol", "notebook_edit",
}

var shellTools = []string{
	"bash", "bash_output", "kill_shell", "wait",
}

var metaTools = []string{
	"todo_write", "complete_step",
	"remember", "forget", "memory_search",
}

var subagentTools = []string{
	"explore", "research", "review", "security_review",
	"run_skill", "install_skill", "task",
}

// merge concatenates multiple string slices.
func merge(slices ...[]string) []string {
	var out []string
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

// mergeUnique appends unique elements from src to dst.
func mergeUnique(dst, src []string) []string {
	seen := make(map[string]bool, len(dst))
	for _, t := range dst {
		seen[t] = true
	}
	for _, t := range src {
		if !seen[t] {
			dst = append(dst, t)
			seen[t] = true
		}
	}
	return dst
}

// matchAnyWord reports whether s contains any keyword as a whole word.
func matchAnyWord(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if containsWord(s, kw) {
			return true
		}
	}
	return false
}

// containsWord checks whether needle appears as a word-boundary delimited
// substring in haystack (substring match like "api" inside "rapid" is ignored).
func containsWord(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	needleLower := strings.ToLower(needle)
	haystackLower := strings.ToLower(haystack)
	for i := 0; i <= len(haystackLower)-len(needleLower); i++ {
		if haystackLower[i:i+len(needleLower)] == needleLower {
			prev := byte(' ')
			if i > 0 {
				prev = haystackLower[i-1]
			}
			next := byte(' ')
			if i+len(needleLower) < len(haystackLower) {
				next = haystackLower[i+len(needleLower)]
			}
			if !isAlphaNum(prev) && !isAlphaNum(next) {
				return true
			}
		}
	}
	return false
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// IsSimpleQuery detects short read-only queries that may not need full planning.
// Criteria: ≤100 runes AND matches at least one simple-query keyword.
func IsSimpleQuery(lower string, raw string) bool {
	if len([]rune(raw)) > 100 {
		return false
	}
	simpleKeywords := []string{
		"explain", "what is", "what does", "how does", "how to",
		"who", "when", "where", "why", "tell me",
		"show me", "list", "find", "search", "look up",
		"check", "describe", "meaning", "document",
		"什么是", "怎么", "如何", "为什么", "谁",
		"列出", "查找", "搜索", "显示",
		"帮我", "告诉我",
	}
	return matchAnyWord(lower, simpleKeywords...)
}
