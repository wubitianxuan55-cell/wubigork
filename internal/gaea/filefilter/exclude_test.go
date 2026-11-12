package filefilter

import "testing"

func TestMatcherDefaultsAndUserPatterns(t *testing.T) {
	m := Compile([]string{"generated", "docs/private", "**/tmp", "*.bak", "  ", "**"})
	cases := []struct {
		rel  string
		name string
		want bool
	}{
		{"node_modules", "node_modules", true},
		{"src/node_modules", "node_modules", true},
		{".DS_Store", ".DS_Store", true},
		{"src/.tmp-cache", ".tmp-cache", true},
		{"a/generated", "generated", true},
		{"docs/private", "private", true},
		{"docs/public", "public", false},
		{"x/y/tmp", "tmp", true},
		{"build/out.bak", "out.bak", true},
		{"README.md", "README.md", false},
	}
	for _, tc := range cases {
		if got := m.Match(tc.rel, tc.name); got != tc.want {
			t.Errorf("Match(%q,%q)=%v want %v", tc.rel, tc.name, got, tc.want)
		}
	}
}

func TestMatcherAnchoredPathAndWildcards(t *testing.T) {
	m := Compile([]string{"cache/**/debug", "src/*.log", "docs/d?.md"})
	cases := []struct {
		rel  string
		name string
		want bool
	}{
		{"cache/debug", "debug", true},
		{"cache/a/b/debug", "debug", true},
		{"other/cache/debug", "debug", false},
		{"src/app.log", "app.log", true},
		{"src/sub/app.log", "app.log", false},
		{"docs/d1.md", "d1.md", true},
		{"docs/d.md", "d.md", false},
	}
	for _, tc := range cases {
		if got := m.Match(tc.rel, tc.name); got != tc.want {
			t.Errorf("Match(%q,%q)=%v want %v", tc.rel, tc.name, got, tc.want)
		}
	}
}

func TestMatcherLimitsInvalidPatterns(t *testing.T) {
	if IsDefaultNoise("") {
		t.Fatal("empty name must not be noise")
	}
	if Compile(nil).Match("src/main.go", "main.go") {
		t.Fatal("ordinary file must not be excluded")
	}
	long := make([]byte, PatternMaxLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, ok := normalizePattern(string(long)); ok {
		t.Fatal("overlong pattern must be dropped")
	}
	if _, ok := normalizePattern("**"); ok {
		t.Fatal("bare ** must be rejected")
	}
}
