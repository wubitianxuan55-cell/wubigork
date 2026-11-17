package strutil

import (
	_ "embed"
	"encoding/json"
	"math"
	"sync"
	"unicode/utf8"
)

// output_budget.json is the cross-language budget contract. The frontend imports
// the same file, so transport/display limits cannot drift into separate policies.
//
//go:embed output_budget.json
var outputBudgetJSON []byte

type Mode int

const (
	ModeBytes Mode = iota
	ModeTokens
	// ModeRunes is retained for call sites whose historical public parameter is a
	// character count. New transport budgets should prefer bytes or tokens.
	ModeRunes
)

type Policy struct {
	Mode  Mode
	Limit int

	allowanceApplied bool
}

func BytesPolicy(limit int) Policy {
	return Policy{Mode: ModeBytes, Limit: max(limit, 0)}
}

func TokensPolicy(limit int) Policy {
	return Policy{Mode: ModeTokens, Limit: max(limit, 0)}
}

func RunesPolicy(limit int) Policy {
	return Policy{Mode: ModeRunes, Limit: max(limit, 0)}
}

// ByteBudget is the byte-equivalent budget used by serializers. Tokens use the
// same 4:1 approximation as the distilled upstream policy.
func (p Policy) ByteBudget() int {
	switch p.Mode {
	case ModeBytes:
		return max(p.Limit, 0)
	case ModeTokens, ModeRunes:
		return max(p.Limit, 0) * 4
	default:
		return 0
	}
}

func (p Policy) TokenBudget() int {
	switch p.Mode {
	case ModeTokens:
		return max(p.Limit, 0)
	default:
		b := p.ByteBudget()
		return (b + 3) / 4
	}
}

// WithSerializationAllowance applies the 1.2x headroom once. Repeated calls are
// deliberately idempotent so callers cannot silently compound the budget.
func (p Policy) WithSerializationAllowance() Policy {
	if p.allowanceApplied || p.Limit <= 0 {
		return p
	}
	p.Limit = int(math.Ceil(float64(p.Limit) * SharedOutputBudget().Transport.SerializationAllowance))
	p.allowanceApplied = true
	return p
}

type Truncation struct {
	Text          string
	Truncated     bool
	OriginalBytes int
	RetainedBytes int
}

// TruncateMiddleChars keeps the marker outside the budget.
func TruncateMiddleChars(s string, policy Policy, marker string) Truncation {
	return truncateMiddle(s, policy, marker, false)
}

// TruncateMiddleWithMarker keeps the marker inside the budget. If the marker
// itself does not fit, the result degrades to the longest fitting prefix.
func TruncateMiddleWithMarker(s string, policy Policy, marker string) Truncation {
	return truncateMiddle(s, policy, marker, true)
}

func truncateMiddle(s string, policy Policy, marker string, markerInBudget bool) Truncation {
	originalBytes := len(s)
	limit := policyLimit(policy)
	if limit <= 0 || sizeInMode(s, policy.Mode) <= limit {
		return Truncation{Text: s, OriginalBytes: originalBytes, RetainedBytes: originalBytes}
	}

	markerSize := sizeInMode(marker, policy.Mode)
	budget := limit
	if markerInBudget {
		budget -= markerSize
		if budget <= 0 {
			head := takeHead(s, limit, policy.Mode)
			return Truncation{
				Text: head, Truncated: true, OriginalBytes: originalBytes, RetainedBytes: len(head),
			}
		}
	}

	headRatio := SharedOutputBudget().Transport.HeadRatio
	if headRatio <= 0 || headRatio >= 1 {
		headRatio = 0.5
	}
	left := int(math.Floor(float64(budget) * headRatio))
	right := budget - left
	head := takeHead(s, left, policy.Mode)
	tail := takeTail(s, right, policy.Mode)
	text := head + marker + tail
	return Truncation{
		Text:          text,
		Truncated:     true,
		OriginalBytes: originalBytes,
		RetainedBytes: len(head) + len(tail),
	}
}

func policyLimit(policy Policy) int {
	if policy.Limit <= 0 {
		return 0
	}
	switch policy.Mode {
	case ModeBytes:
		return policy.Limit
	case ModeTokens:
		return policy.Limit * 4
	case ModeRunes:
		return policy.Limit
	default:
		return 0
	}
}

func sizeInMode(s string, mode Mode) int {
	if mode == ModeRunes {
		return utf8.RuneCountInString(s)
	}
	return len(s)
}

func takeHead(s string, limit int, mode Mode) string {
	if limit <= 0 || s == "" {
		return ""
	}
	if mode == ModeRunes {
		return TruncateRunes(s, limit)
	}
	if len(s) <= limit {
		return s
	}
	end := limit
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end]
}

func takeTail(s string, limit int, mode Mode) string {
	if limit <= 0 || s == "" {
		return ""
	}
	if mode == ModeRunes {
		runes := []rune(s)
		if len(runes) <= limit {
			return s
		}
		return string(runes[len(runes)-limit:])
	}
	if len(s) <= limit {
		return s
	}
	start := len(s) - limit
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}

type OutputBudget struct {
	SchemaVersion int `json:"schemaVersion"`
	Transport     struct {
		ByteLimit                  int     `json:"byteLimit"`
		HeadRatio                  float64 `json:"headRatio"`
		SerializationAllowance     float64 `json:"serializationAllowance"`
		AgentPreviewByteLimit      int     `json:"agentPreviewByteLimit"`
		RepeatArgumentPreviewBytes int     `json:"repeatArgumentPreviewBytes"`
		ArchiveContentByteLimit    int     `json:"archiveContentByteLimit"`
		ControllerSummaryRunes     int     `json:"controllerSummaryRunes"`
	} `json:"transport"`
	Display struct {
		GenericPreviewLines  int `json:"genericPreviewLines"`
		ToolCardPreviewLines int `json:"toolCardPreviewLines"`
		HeadLines            int `json:"headLines"`
		TailLines            int `json:"tailLines"`
		LineWidth            int `json:"lineWidth"`
		ByteLimit            int `json:"byteLimit"`
		MaxRenderedRows      int `json:"maxRenderedRows"`
	} `json:"display"`
}

var (
	outputBudgetOnce sync.Once
	outputBudget     OutputBudget
)

func SharedOutputBudget() OutputBudget {
	outputBudgetOnce.Do(func() {
		if err := json.Unmarshal(outputBudgetJSON, &outputBudget); err != nil {
			panic("invalid embedded output_budget.json: " + err.Error())
		}
	})
	return outputBudget
}

func DefaultTransportPolicy() Policy {
	return BytesPolicy(SharedOutputBudget().Transport.ByteLimit)
}
