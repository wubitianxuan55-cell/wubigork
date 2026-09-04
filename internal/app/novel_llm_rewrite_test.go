package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/novelstyle"
)

func TestSplitSentences_Chinese(t *testing.T) {
	content := "他攥紧了拳头，指节发白。她笑了笑，没说话。他说：“明天见。”"
	sents := splitSentences(content)
	if len(sents) != 3 {
		t.Fatalf("应为 3 句，got %d: %#v", len(sents), sents)
	}
	if sents[0].text != "他攥紧了拳头，指节发白。" {
		t.Fatalf("句0错误: %q", sents[0].text)
	}
	if sents[1].text != "她笑了笑，没说话。" {
		t.Fatalf("句1错误: %q", sents[1].text)
	}
	if !strings.Contains(sents[2].text, "明天见") {
		t.Fatalf("句2错误: %q", sents[2].text)
	}
	// rune 偏移应自洽：用子串能还原。
	for _, s := range sents {
		if got := string([]rune(content)[s.start:s.end]); strings.TrimSpace(got) != s.text {
			t.Fatalf("偏移还原不匹配: %q vs %q", got, s.text)
		}
	}
}

func TestPickFlaggedSentences_DedupAndMax(t *testing.T) {
	content := "第一句没有毛病。他的眼帘微微上扬，内心充满了震撼。他又缓缓抬头。"
	sents := splitSentences(content)
	// 构造一个覆盖第 2 句两个 span 的 score（去重后只该命中第 2 句一次）。
	score := &novelstyle.TasteScore{
		Score: 60,
		Issues: []novelstyle.TasteIssue{
			{Start: 6, End: 9, Reason: "黑名单", Severity: "high"},
			{Start: 12, End: 16, Reason: "情绪直述", Severity: "medium"},
		},
	}
	picked := pickFlaggedSentences(content, score, sents, 10)
	if len(picked) != 1 {
		t.Fatalf("去重后应为 1 句，got %d", len(picked))
	}
	if picked[0].text != sents[1].text {
		t.Fatalf("应命中第 2 句，got %q", picked[0].text)
	}
}

func TestSplitSentences_NoControlReturnsWhole(t *testing.T) {
	content := "一句话没有标点"
	sents := splitSentences(content)
	if len(sents) != 1 || sents[0].text != content {
		t.Fatalf("无标点应整段为一句: %#v", sents)
	}
}
