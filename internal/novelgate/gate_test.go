package novelgate

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func codes(issues []Issue) map[string]string {
	m := map[string]string{}
	for _, i := range issues {
		m[i.Code] = i.Severity
	}
	return m
}

func TestOutlineContractIssues(t *testing.T) {
	full := types.OutlineNode{Title: "第一章 起", Summary: "谁在哪做什么", KeyPoints: []string{"a", "b"}, Emotion: "紧张"}
	if got := OutlineContractIssues(full); len(got) != 0 {
		t.Fatalf("齐备的大纲不应报问题: %+v", got)
	}
	empty := codes(OutlineContractIssues(types.OutlineNode{}))
	for _, want := range []string{"outline_title_empty", "outline_summary_empty", "outline_keypoints_empty", "outline_emotion_empty"} {
		if _, ok := empty[want]; !ok {
			t.Fatalf("缺项未报出 %s: %+v", want, empty)
		}
	}
	if empty["outline_summary_empty"] != "S2" {
		t.Fatalf("计划为空应为 S2: %v", empty["outline_summary_empty"])
	}
}

func TestChapterQualityIssues_EmptyAndClean(t *testing.T) {
	if got := ChapterQualityIssues("   "); len(got) != 1 || got[0].Code != "chapter_empty" || got[0].Severity != "S1" {
		t.Fatalf("空正文应 S1: %+v", got)
	}
	clean := strings.Repeat("他推开门，雨还在下，屋檐下的灯影在水面上摇晃了很久很久，像一段没有说完的话。", 8)
	if got := ChapterQualityIssues(clean); len(got) != 0 {
		t.Fatalf("正常文本不应误报: %+v", got)
	}
}

func TestChapterQualityIssues_TelegraphStyle(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 40; i++ {
		sb.WriteString("他站住。她回头。风很大。灯灭了。雨停了。")
	}
	got := codes(ChapterQualityIssues(sb.String()))
	if got["telegraph_style"] == "" && got["sentences_too_short"] == "" {
		t.Fatalf("通篇超短句应报电报体/句长偏短: %+v", got)
	}
}

func TestChapterQualityIssues_ParagraphTooLong(t *testing.T) {
	got := codes(ChapterQualityIssues(strings.Repeat("字", paragraphMaxRunes+50)))
	if got["paragraph_too_long"] == "" {
		t.Fatalf("超长段落应报出: %+v", got)
	}
}

func TestChapterQualityIssues_Punctuation(t *testing.T) {
	got := codes(ChapterQualityIssues("怎么会这样！！！他不明白。" + strings.Repeat("然后呢…………", 4)))
	if got["punctuation_pileup"] == "" {
		t.Fatalf("连续感叹号应报出: %+v", got)
	}
	if got["ellipsis_overuse"] == "" {
		t.Fatalf("省略号滥用应报出: %+v", got)
	}
}

func TestChapterQualityIssues_PeriodHeavy(t *testing.T) {
	got := codes(ChapterQualityIssues(strings.Repeat("他说完了。她走了。天黑了。灯灭了。", 8)))
	if got["period_heavy"] == "" {
		t.Fatalf("通篇句号化应报出: %+v", got)
	}
}

func TestChapterQualityIssues_EvidenceCarried(t *testing.T) {
	found := false
	for _, i := range ChapterQualityIssues("怎么会这样！！！") {
		if i.Code == "punctuation_pileup" {
			found = true
			if i.Evidence == "" || i.Message == "" {
				t.Fatalf("发现应带证据与说明: %+v", i)
			}
		}
	}
	if !found {
		t.Fatal("未捕到标点堆砌")
	}
}
