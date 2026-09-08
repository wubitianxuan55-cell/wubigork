package strutil

import (
	"strings"
	"testing"
)

func TestTitleSlugCanonical(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"chinese", "中文 标题", "中文-标题"},
		{"cement spec", "P.O 42.5 水泥", "p-o-42-5-水泥"},
		{"mixed separators", "  GAEa 项目 _ 预算.v2  ", "gaea-项目-预算-v2"},
		{"leading trailing separators", " --- leading and trailing --- ", "leading-and-trailing"},
		{"symbols collapse", "a!!?b", "a-b"},
		{"fallback", "", "entry"},
		{"all separators fallback", "!!!", "entry"},
		{"unicode letters kept", "Café Résumé 42", "café-résumé-42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TitleSlug(tt.title); got != tt.want {
				t.Fatalf("TitleSlug(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestTitleSlugTruncatesAtSixtyRunes(t *testing.T) {
	ascii := strings.Repeat("a", 61)
	if got, want := TitleSlug(ascii), strings.Repeat("a", 60); got != want {
		t.Fatalf("ascii truncate = %d runes, want 60", len([]rune(got)))
	}

	chinese := strings.Repeat("文", 61)
	if got, want := TitleSlug(chinese), strings.Repeat("文", 60); got != want {
		t.Fatalf("chinese truncate = %d runes, want 60", len([]rune(got)))
	}
}

// TestTitleSlugLegacyGoldenMatrix freezes the outputs collected by invoking all
// five former implementations on the same corpus before migration. It documents
// exactly where the new canonical behavior follows the majority rather than an
// individual old edge case.
func TestTitleSlugLegacyGoldenMatrix(t *testing.T) {
	const (
		costSlug = iota
		knowledgeSlug
		appSlug
		modelSlug
		controlSlug
		legacyCount
	)
	tests := []struct {
		title     string
		canonical string
		legacy    [legacyCount]string
		// follows records which old implementations the canonical result is
		// intended to match for this input.
		follows [legacyCount]bool
	}{
		{
			title:     "P.O 42.5 水泥",
			canonical: "p-o-42-5-水泥",
			legacy:    [legacyCount]string{"p-o-42-5-水泥", "p-o-42-5-水泥", "p-o-42-5-水泥", "po-425", "p-o-42-5-水泥"},
			follows:   [legacyCount]bool{true, true, true, false, true},
		},
		{
			title:     "  GAEa 项目 _ 预算.v2  ",
			canonical: "gaea-项目-预算-v2",
			legacy:    [legacyCount]string{"gaea-项目-预算-v2", "gaea-项目-预算-v2", "gaea-项目-预算-v2", "gaea-v2", "gaea-项目---预算-v2"},
			follows:   [legacyCount]bool{true, true, true, false, false},
		},
		{
			title:     "中文 标题",
			canonical: "中文-标题",
			legacy:    [legacyCount]string{"中文-标题", "中文-标题", "中文-标题", "engine", "中文-标题"},
			follows:   [legacyCount]bool{true, true, true, false, true},
		},
		{
			title:     "!!!",
			canonical: "entry",
			legacy:    [legacyCount]string{"cost", "entry", "entry", "engine", "!!!"},
			follows:   [legacyCount]bool{false, true, true, false, false},
		},
		{
			title:     "",
			canonical: "entry",
			legacy:    [legacyCount]string{"cost", "entry", "entry", "engine", ""},
			follows:   [legacyCount]bool{false, true, true, false, false},
		},
		{
			title:     " --- leading and trailing --- ",
			canonical: "leading-and-trailing",
			legacy:    [legacyCount]string{"leading-and-trailing", "leading-and-trailing", "leading-and-trailing", "leading-and-trailing", "----leading-and-trailing----"},
			follows:   [legacyCount]bool{true, true, true, true, false},
		},
		{
			title:     "测试引擎",
			canonical: "测试引擎",
			legacy:    [legacyCount]string{"测试引擎", "测试引擎", "测试引擎", "engine", "测试引擎"},
			follows:   [legacyCount]bool{true, true, true, false, true},
		},
		{
			title:     "My Engine!!",
			canonical: "my-engine",
			legacy:    [legacyCount]string{"my-engine", "my-engine", "my-engine", "my-engine", "my-engine!!"},
			follows:   [legacyCount]bool{true, true, true, true, false},
		},
		{
			title:     "Café Résumé 42",
			canonical: "café-résumé-42",
			legacy:    [legacyCount]string{"café-résumé-42", "café-résumé-42", "caf-r-sum-42", "caf-rsum-42", "café-résumé-42"},
			follows:   [legacyCount]bool{true, true, false, false, true},
		},
	}
	for _, tt := range tests {
		if got := TitleSlug(tt.title); got != tt.canonical {
			t.Errorf("TitleSlug(%q) = %q, canonical want %q", tt.title, got, tt.canonical)
		}
		for i, old := range tt.legacy {
			if matches := old == tt.canonical; matches != tt.follows[i] {
				t.Errorf("%q legacy[%d]=%q: canonical match=%v, documented=%v", tt.title, i, old, matches, tt.follows[i])
			}
		}
	}
}
