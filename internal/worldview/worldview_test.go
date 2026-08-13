package worldview

import "testing"

func TestExtractSectionUpdates(t *testing.T) {
	reply := "前面说明\n" +
		"---WORLDVIEW_SECTION_UPDATE---\n" +
		`{"id":"era","content":"灵气复苏的时代"}` + "\n" +
		"---END_UPDATE---\n" +
		"---WORLDVIEW_SECTION_UPDATE---\n" +
		`{"id":"rules","content":"修仙者需渡劫"}` + "\n" +
		"---END_UPDATE---"

	updates := extractSectionUpdates(reply)
	if got := updates["era"]; got != "灵气复苏的时代" {
		t.Fatalf("era = %q, want 灵气复苏的时代", got)
	}
	if got := updates["rules"]; got != "修仙者需渡劫" {
		t.Fatalf("rules = %q, want 修仙者需渡劫", got)
	}
}

func TestExtractSectionUpdatesHandlesMissingEndMarker(t *testing.T) {
	reply := "---WORLDVIEW_SECTION_UPDATE---\n" +
		`{"id":"era","content":"未闭合内容"}`

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("extractSectionUpdates should not panic: %v", r)
		}
	}()
	_ = extractSectionUpdates(reply)
}

func TestExtractMarkdownBlock(t *testing.T) {
	reply := "以下是完整设定：\n```markdown\n# 世界观\n九州大陆\n```\n结束"
	if got := extractMarkdownBlock(reply); got != "# 世界观\n九州大陆" {
		t.Fatalf("got %q", got)
	}
}
