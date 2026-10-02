package novelreview

// 小说板块优化批 2 · 线5 G9：字数表述核对（stated_length_accuracy）的证据区间
// 必须始终合法（start < end）。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/noveltext"
)

// assertValidSpans 断言证据区间非空、非越界、且 start < end。
func assertValidSpans(t *testing.T, runes []rune, spans []EvidenceSpan) {
	t.Helper()
	for i, ev := range spans {
		if ev.Start >= ev.End {
			t.Fatalf("证据 %d 区间非法（Start=%d End=%d）: %+v", i, ev.Start, ev.End, spans)
		}
		if ev.Start < 0 || ev.End > len(runes) {
			t.Fatalf("证据 %d 越界: %+v（文本 %d rune）", i, ev, len(runes))
		}
		if strings.TrimSpace(string(runes[ev.Start:ev.End])) == "" {
			t.Fatalf("证据 %d 落在空白上: %q", i, string(runes[ev.Start:ev.End]))
		}
	}
}

// TestDimWordcountExpr_BeforeBranchKeepsCoordinates G9：向前找引号的回退分支必须
// 接住自己的 start/end——旧实现丢弃坐标、沿用 after 分支未命中时的 qend=0，
// 产出 Start=i、End=0 的非法区间。本用例的引号实体在 after 的 20 rune 窗口之外，
// 必然走回退分支。
func TestDimWordcountExpr_BeforeBranchKeepsCoordinates(t *testing.T) {
	p := Platform{ID: "general", Label: "通用"}
	// 布局：「甲」在最前，其后 15 个垫字（> 跳过 after 前 4 rune 后的剩余 20 rune
	// 窗口的一半，使 after 分支必然落空），再是「这三个字」表述。
	text := "「甲」" + strings.Repeat("垫", 15) + "这三个字。"
	runes := []rune(text)
	d := dimWordcountExpr(p, runes, noveltext.SplitParagraphs(runes))

	if d.Verdict != verdictWarn || len(d.Evidence) == 0 {
		t.Fatalf("应命中字数表述不符并给出证据: %+v", d)
	}
	assertValidSpans(t, runes, d.Evidence)

	joined := ""
	for _, ev := range d.Evidence {
		joined += string(runes[ev.Start:ev.End])
	}
	if !strings.Contains(joined, "甲") || !strings.Contains(joined, "这三个字") {
		t.Fatalf("证据应并集引号实体与「这N个字」表述: %q", joined)
	}
}

// TestDimWordcountExpr_SpansValidAcrossPhrasings 两种语序（引号在前 / 引号在后）
// 与多段落下的证据区间都合法；字数相符时不给证据（反向守卫）。
func TestDimWordcountExpr_SpansValidAcrossPhrasings(t *testing.T) {
	p := Platform{ID: "general", Label: "通用"}
	for _, text := range []string{
		"这三个字「甲乙丙丁戊」，由此可见。",
		"他念出「甲乙丙丁戊」，这三个字分量很重。",
		"「甲乙丙丁戊」这三个字。" + strings.Repeat("垫", 40),
		"这三个字就在前面「甲乙丙丁戊」。",
		strings.Repeat("前", 25) + "他写下「甲」稍远" + strings.Repeat("垫", 25) + "这三个字。",
	} {
		runes := []rune(text)
		d := dimWordcountExpr(p, runes, noveltext.SplitParagraphs(runes))
		assertValidSpans(t, runes, d.Evidence)
	}

	// 反向守卫：字数相符（「甲乙丙」= 3 字，表述「这三个字」）不给证据。
	okText := "他写下「甲乙丙」这三个字。"
	okRunes := []rune(okText)
	if d := dimWordcountExpr(p, okRunes, noveltext.SplitParagraphs(okRunes)); len(d.Evidence) != 0 {
		t.Fatalf("字数相符不应给证据: %+v", d.Evidence)
	}
}

// TestReview_WordcountEvidenceSpansValid 端到端：整章评审里该维度的证据区间
// 恒满足 start < end（回归钉住 G9 的非法区间）。
func TestReview_WordcountEvidenceSpansValid(t *testing.T) {
	text := strings.Join([]string{
		"「甲」" + strings.Repeat("垫", 30) + "这三个字说得极重。",
		"少年握紧剑柄，一步步走进风雪里。",
		"他念出「甲乙丙丁戊」，这三个字分量很重。",
	}, "\n\n")
	rep := mustReview(t, text, "general", Options{})
	d := dimOf(t, rep, "stated_length_accuracy")
	assertValidSpans(t, []rune(text), d.Evidence)
}
