package memory

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// ─── activation 二维：固化正文随会话快照（v4.378）──────────────────────────

func pinnedFixture() []Memory {
	return []Memory{
		{Name: "beta", Pinned: true, Body: "第二条固化正文"},
		{Name: "alpha", Pinned: true, Body: strings.Repeat("长", pinnedFactsBodyCap+50)},
		{Name: "plain", Body: "未固化不进块"},
		{Name: "empty-body", Pinned: true, Body: "   "},
	}
}

// 只收 Pinned 且有正文；Name 升序（前缀稳定）；超长正文截断加省略标。
func TestBuildPinnedFactsBlockFiltersSortsTruncates(t *testing.T) {
	block := BuildPinnedFactsBlock(pinnedFixture(), 0)
	if !strings.HasPrefix(block, pinnedFactsHeader+"\n") {
		t.Fatalf("block must start with header, got %q", block)
	}
	if strings.Contains(block, "未固化不进块") {
		t.Fatal("non-pinned facts must be excluded")
	}
	iAlpha := strings.Index(block, "- alpha：")
	iBeta := strings.Index(block, "- beta：")
	if iAlpha < 0 || iBeta < 0 {
		t.Fatalf("both pinned facts must appear, got %q", block)
	}
	if iAlpha > iBeta {
		t.Fatalf("entries must be Name-ascending (prefix stability), got %q", block)
	}
	// alpha 超长：整条截断到 bodyCap 内（含省略标）且以省略标收尾。
	line := block[iAlpha : strings.Index(block[iAlpha:], "\n")+iAlpha]
	body := strings.TrimPrefix(line, "- alpha：")
	if got := utf8.RuneCountInString(body); got > pinnedFactsBodyCap {
		t.Fatalf("truncated body must stay within cap, got %d runes", got)
	}
	if !strings.HasSuffix(body, pinnedFactsTruncMark) {
		t.Fatalf("truncated body must end with the ellipsis mark, got %q", body[len(body)-20:])
	}
	if strings.Contains(block, "\n\n") {
		t.Fatalf("body must be single-line, got %q", block)
	}
}

// 预算诚实：整行放不下即停；预算连块头都放不下返回空串。
func TestBuildPinnedFactsBlockBudget(t *testing.T) {
	mems := []Memory{
		{Name: "a", Pinned: true, Body: "短正文"},
		{Name: "b", Pinned: true, Body: strings.Repeat("长", pinnedFactsBodyCap)},
	}
	block := BuildPinnedFactsBlock(mems, 40) // 只容得下 a
	if !strings.Contains(block, "- a：短正文") {
		t.Fatalf("small entry must fit, got %q", block)
	}
	if strings.Contains(block, "- b：") {
		t.Fatalf("entry that cannot fit whole must be omitted, got %q", block)
	}
	if got := BuildPinnedFactsBlock(mems, 2); got != "" {
		t.Fatalf("budget smaller than header must yield empty block, got %q", got)
	}
}

// 空集合/无固化/全空正文 → 空串（零注入，前缀逐字节不变）。
func TestBuildPinnedFactsBlockEmpty(t *testing.T) {
	cases := [][]Memory{
		nil,
		{{Name: "plain", Body: "未固化"}},
		{{Name: "p", Pinned: true, Body: "  \n "}},
	}
	for i, mems := range cases {
		if got := BuildPinnedFactsBlock(mems, 0); got != "" {
			t.Fatalf("case %d: must return empty block, got %q", i, got)
		}
	}
}
