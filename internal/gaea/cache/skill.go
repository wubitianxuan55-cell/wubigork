package cache

import (
	"strings"
)

// SkillProfile is the L3 execution policy for a task kind and version.
// It defines which tools are active, what behavioural hint to inject,
// and the conditions for version promotion.
//
// V3.0: replaces GoalRouter's hardcoded tool sets with versioned profiles.
// VerificationPolicy defines the verification requirements for a task kind.
type VerificationPolicy struct {
	RequireTests    bool
	RequireBuild    bool
	AutoReview      bool
	RequireCitation bool
}

type SkillProfile struct {
	Kind             TaskKind
	Tools            []string
	PromptHint       string
	Temperature      float64
	MaxSteps         int
	CompactThreshold float64
	RetryLimit       int
	Verification     VerificationPolicy
}

// SkillLayer is the L3 cache domain — intent classification with static,
// versioned profiles. It replaces GoalRouter's monolithic Route() with
// classify → select profile.
//
// Profiles are indexed by [TaskKind][version]. V5.0 removed the Learner
// (adaptive version promotion), so every kind defines exactly one profile
// (version 1) and Route() always selects it; the [TaskKind][int] shape is
// kept only because Route's defensive fallback reads it that way.
type SkillLayer struct {
	current SkillProfile
	version int
}

// Profiles defines the skill profile for each task kind. V5.0: exactly one
// version (1) per kind — the v2/v3 entries died with the Learner and are
// pinned unreachable by TestProfilesSingleVersionPerKind.
var Profiles = map[TaskKind]map[int]SkillProfile{
	KindFixBug: {
		1: {Kind: KindFixBug, Tools: merge(readTools, editTools, shellTools, metaTools), PromptHint: "Reproduce first. Batch all file reads and searches in one response. Read → isolate → fix → verify.", Temperature: 0.3, MaxSteps: 20, RetryLimit: 3, Verification: VerificationPolicy{RequireBuild: true}},
	},
	KindWriteFeature: {
		1: {Kind: KindWriteFeature, Tools: merge(readTools, editTools, shellTools, metaTools), PromptHint: "Design first. Read all relevant files in one batch. Keep changes minimal.", Temperature: 0.5, MaxSteps: 20, RetryLimit: 3, Verification: VerificationPolicy{RequireBuild: true}},
	},
	KindReview: {
		1: {Kind: KindReview, Tools: merge(readTools, metaTools), PromptHint: "Read all changed files at once. Check correctness, security, tests. Do NOT edit.", Temperature: 0, MaxSteps: 5, Verification: VerificationPolicy{AutoReview: true}},
	},
	KindExplain: {
		1: {Kind: KindExplain, Tools: merge(readTools, metaTools), PromptHint: "Read relevant code in one batch. Explain with references. Do NOT edit.", Temperature: 0, MaxSteps: 5},
	},
	KindResearch: {
		1: {Kind: KindResearch, Tools: merge(readTools, metaTools), PromptHint: "Search broadly first. Batch web searches and reads together. Cite sources.", Temperature: 0.7, MaxSteps: 30, Verification: VerificationPolicy{RequireCitation: true}},
	},
	KindDefault: {
		1: {Kind: KindDefault, Tools: nil, PromptHint: "Batch independent tool calls in a single response.", Temperature: 0.5, MaxSteps: 20, RetryLimit: 3},
	},
	// V4.0: non-code task kinds
	KindDataAnalysis: {
		1: {Kind: KindDataAnalysis, Tools: merge(readTools, shellTools, metaTools), PromptHint: "Load then explore. Batch independent data reads together. Load → explore → transform.", Temperature: 0.3, MaxSteps: 25, RetryLimit: 3, Verification: VerificationPolicy{RequireCitation: true}},
	},
	KindWriting: {
		1: {Kind: KindWriting, Tools: merge(readTools, editTools, metaTools), PromptHint: "Read references first. Batch research in one step. Draft → revise → polish.", Temperature: 0.7, MaxSteps: 15},
	},
	KindGeneral: {
		1: {Kind: KindGeneral, Tools: nil, PromptHint: "Gather context first. Batch independent tool calls together.", Temperature: 0.5, MaxSteps: 20, RetryLimit: 3},
	},
	KindSimple: {
		1: {Kind: KindSimple, Tools: merge(readTools, metaTools), PromptHint: "This is a simple query. Answer concisely with references. Do NOT edit.", Temperature: 0, MaxSteps: 5},
	},
}

// NewSkillLayer creates a SkillLayer with static profiles (V5.0: Learner removed).
func NewSkillLayer() *SkillLayer {
	return &SkillLayer{version: 1}
}

// Route classifies the input and returns the matching SkillProfile.
// V5.0: the Learner is gone — version 1 is the only version defined, so
// the lookup can never select anything else (pinned by
// TestRouteAlwaysSelectsVersionOne).
func (l *SkillLayer) Route(input string) SkillProfile {
	kind := classifyIntent(input)

	const version = 1
	profile, ok := Profiles[kind][version]
	if !ok {
		// Unknown kind: fall back to the default profile.
		profile = Profiles[KindDefault][version]
	}

	l.current = profile
	l.version = version
	return profile
}

// CurrentProfile returns the currently active profile.
func (l *SkillLayer) CurrentProfile() SkillProfile { return l.current }

// CurrentVersion returns the current profile version.
func (l *SkillLayer) CurrentVersion() int { return l.version }

// classifyIntent is the core classification logic (extracted from GoalRouter.Route).
func classifyIntent(input string) TaskKind {
	lower := strings.ToLower(input)

	// V10.XX: simple query detection — short input with read-only keywords.
	// Returns KindSimple so the agent can answer directly without tools.
	if IsSimpleQuery(lower, input) {
		return KindSimple
	}

	if matchAnyWord(lower,
		"fix", "bug", "repair", "crash",
		"panic", "exception",
		"defect", "patch", "debug",
		"issue", "error", "fail", "broken",
		"wrong", "incorrect", "typo", "not working",
	) {
		return KindFixBug
	}
	if matchAnyWord(lower,
		"add", "create", "feature", "develop", "build",
		"implement", "refactor", "rewrite",
		"update", "change", "modify", "new", "extend",
	) {
		return KindWriteFeature
	}
	if matchAnyWord(lower,
		"review", "audit", "code review", "pr review",
		"inspect", "check", "validate", "verify",
	) {
		return KindReview
	}
	if matchAnyWord(lower,
		"explain", "analyze", "how does", "what does",
		"describe",
		"how to", "why", "what is", "meaning",
		"document", "tell me about",
	) {
		return KindExplain
	}
	// V4.0: non-code task classification
	if matchAnyWord(lower,
		"csv", "excel", "spreadsheet", "chart", "plot",
		"statistics", "data analysis", "visualize data",
		"pandas", "dataframe", "sql", "query",
	) {
		return KindDataAnalysis
	}
	if matchAnyWord(lower,
		"write", "article", "blog", "essay", "report",
		"draft", "polish",
	) && !matchAnyWord(lower, "code", "program", "function", "script") {
		return KindWriting
	}
	if matchAnyWord(lower,
		"help", "assist", "suggest", "recommend", "idea",
		"brainstorm", "plan", "organize", "summarize",
	) {
		return KindGeneral
	}
	return KindDefault
}
