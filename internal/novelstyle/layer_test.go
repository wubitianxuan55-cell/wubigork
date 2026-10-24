package novelstyle

import "testing"

func TestTasteIssuesCarryLayers(t *testing.T) {
	Cases := []struct {
		text  string
		layer string
	}{
		{"他的眸光流转，仿佛星辰坠落，宛如月光。", "expression"},
		{"她缓缓抬头。她缓缓转身。她缓缓离开。", "expression"},
		{"她感到非常悲伤，心中充满了难以言说的痛苦。", "narrative"},
	}
	for _, tc := range Cases {
		score, err := ScoreTextNoRef(tc.text)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, issue := range score.Issues {
			if issue.Layer == "" {
				t.Fatalf("issue missing layer: %+v", issue)
			}
			if issue.Layer == tc.layer {
				found = true
			}
		}
		if !found {
			t.Errorf("text %q produced layers %+v, want %s", tc.text, score.Issues, tc.layer)
		}
	}
}

func TestCleanTextDoesNotInventStoryLayer(t *testing.T) {
	score, err := ScoreTextNoRef("雨停在傍晚。她把伞靠在门边，进屋后先洗手，再坐到桌前。")
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range score.Issues {
		if issue.Layer == "story" {
			t.Fatalf("deterministic lexical engine must not invent story-level issue: %+v", issue)
		}
	}
}
