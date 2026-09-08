package strutil

import (
	"strings"
	"unicode"
)

// titleSlugMaxRunes is the canonical title-slug cap shared by knowledge and
// cost keys. It is counted in runes so a Chinese-heavy title gets the same
// logical length limit as an ASCII title.
const titleSlugMaxRunes = 60

// TitleSlugFallback is returned when a title contains no Unicode letters or
// digits. Knowledge import and the app layer formed this plurality among the
// five former implementations.
const TitleSlugFallback = "entry"

// TitleSlug converts a human-readable title to a deterministic slug.
//
// Canonical semantics, selected from the majority of the five former domain
// implementations:
//   - trim surrounding Unicode whitespace, then lowercase;
//   - keep every Unicode letter or digit (Chinese characters are letters);
//   - fold every other rune, including whitespace and punctuation, into one
//     '-' run;
//   - trim leading/trailing '-';
//   - use "entry" when nothing remains (knowledge/app used this fallback,
//     while cost/modelengine had domain-specific fallbacks);
//   - cap at 60 runes before returning.
//
// Domain wrappers may still translate the fallback or apply a stricter
// alphabet, but must not duplicate this normalization loop.
func TitleSlug(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		return TitleSlugFallback
	}
	if runes := []rune(name); len(runes) > titleSlugMaxRunes {
		return string(runes[:titleSlugMaxRunes])
	}
	return name
}
