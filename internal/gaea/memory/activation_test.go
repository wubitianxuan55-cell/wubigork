package memory

import (
	"reflect"
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

func equalStrings(got, want []string) bool {
	if got == nil && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}

// 发射面三态与渲染：只收 Pinned 且有正文；Name 升序（前缀稳定）；超长正文
// 截断加省略标且记入 Truncated（截断=正文未完整在场，召回去重不得排除它）。
func TestBuildPinnedFactsFiltersSortsTruncates(t *testing.T) {
	res := BuildPinnedFacts(pinnedFixture(), 0)
	block := res.Block
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
	// 发射面：alpha/beta 全入块；alpha 截断；无溢出。
	if !equalStrings(res.Emitted, []string{"alpha", "beta"}) {
		t.Fatalf("Emitted = %v, want [alpha beta]", res.Emitted)
	}
	if !equalStrings(res.Truncated, []string{"alpha"}) {
		t.Fatalf("Truncated = %v, want [alpha]", res.Truncated)
	}
	if len(res.Omitted) != 0 {
		t.Fatalf("Omitted = %v, want empty", res.Omitted)
	}
	// 确定性：同输入逐字节稳定（前缀缓存的硬前提）。
	if again := BuildPinnedFacts(pinnedFixture(), 0); again.Block != block {
		t.Fatal("same input must render byte-identically")
	}
}

// 预算诚实 + 溢出提示：整行放不下即不计入并记 Omitted；块尾追加预算内的
// 溢出提示（点名 Omitted，提示自身计入块预算）；预算连块头都放不下返回空块。
func TestBuildPinnedFactsBudgetAndOverflowNotice(t *testing.T) {
	mems := []Memory{
		{Name: "a", Pinned: true, Body: "短正文"},
		{Name: "b", Pinned: true, Body: strings.Repeat("长", pinnedFactsBodyCap)},
	}
	// 默认预算：两条全进块，无溢出无提示。
	res := BuildPinnedFacts(mems, 0)
	if !equalStrings(res.Emitted, []string{"a", "b"}) || len(res.Omitted) != 0 {
		t.Fatalf("default budget: Emitted=%v Omitted=%v", res.Emitted, res.Omitted)
	}
	if strings.Contains(res.Block, "未入块") {
		t.Fatalf("no overflow expected under default budget: %q", res.Block)
	}

	// 小预算：b 出局；块内仍含 a 的正文行；溢出提示点名 b；总长 ≤ 预算；
	// 发射行在提示之前。
	res = BuildPinnedFacts(mems, 120)
	if got := utf8.RuneCountInString(res.Block); got > 120 {
		t.Fatalf("block must stay within budget: %d runes", got)
	}
	if !equalStrings(res.Emitted, []string{"a"}) {
		t.Fatalf("Emitted = %v, want [a]", res.Emitted)
	}
	if !equalStrings(res.Omitted, []string{"b"}) {
		t.Fatalf("Omitted = %v, want [b]", res.Omitted)
	}
	if !strings.Contains(res.Block, "- a：短正文") {
		t.Fatalf("emitted line must survive, got %q", res.Block)
	}
	if !strings.Contains(res.Block, "未入块") || !strings.Contains(res.Block, "b") {
		t.Fatalf("overflow notice must name the omitted entry, got %q", res.Block)
	}
	if strings.Index(res.Block, "- a：") > strings.Index(res.Block, "未入块") {
		t.Fatalf("notice must come after emitted lines, got %q", res.Block)
	}

	// 纯提示块：所有条目都进不了块时，提示仍在场（模型至少知道有固化记忆
	// 未入块、去哪里取）。
	res = BuildPinnedFacts([]Memory{{Name: "b", Pinned: true, Body: strings.Repeat("长", pinnedFactsBodyCap)}}, 80)
	if res.Block == "" {
		t.Fatalf("notice-only block expected, Omitted=%v", res.Omitted)
	}
	if !equalStrings(res.Emitted, nil) || !equalStrings(res.Omitted, []string{"b"}) {
		t.Fatalf("notice-only block: Emitted=%v Omitted=%v", res.Emitted, res.Omitted)
	}
	if !strings.Contains(res.Block, "未入块") {
		t.Fatalf("notice-only block must carry the notice, got %q", res.Block)
	}

	// 病态预算：连最短提示都放不下——诚实零注入，Omitted 仍如实带回（召回
	// 侧不排除未发射条目，两处皆失不可能发生）。
	res = BuildPinnedFacts(mems, 40)
	if res.Block != "" {
		t.Fatalf("pathological budget must yield empty block, got %q", res.Block)
	}
	if !equalStrings(res.Omitted, []string{"a", "b"}) {
		t.Fatalf("Omitted must be carried even on empty block: %v", res.Omitted)
	}

	// 预算连块头都放不下：全空。
	res = BuildPinnedFacts(mems, 2)
	if res.Block != "" || len(res.Emitted)+len(res.Omitted) != 0 {
		t.Fatalf("budget smaller than header must yield empty result: %+v", res)
	}
}

// 空集合/无固化/全空正文 → 零注入（前缀逐字节不变）。
func TestBuildPinnedFactsEmpty(t *testing.T) {
	cases := [][]Memory{
		nil,
		{{Name: "plain", Body: "未固化"}},
		{{Name: "p", Pinned: true, Body: "  \n "}},
	}
	for i, mems := range cases {
		if got := BuildPinnedFacts(mems, 0); got.Block != "" {
			t.Fatalf("case %d: must return empty block, got %q", i, got.Block)
		}
	}
}
