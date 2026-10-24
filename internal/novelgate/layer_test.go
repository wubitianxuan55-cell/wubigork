package novelgate

import (
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestGateIssuesCarryThreeLayers(t *testing.T) {
	plan := OutlineContractIssues(types.OutlineNode{})
	if len(plan) == 0 || plan[0].Layer != "story" {
		t.Fatalf("outline issues should be story layer: %+v", plan)
	}

	text := "他走。她走。他又走。"
	for i := 0; i < 25; i++ {
		text += "她笑！！！。"
	}
	quality := ChapterQualityIssues(text)
	foundNarrative, foundExpression := false, false
	for _, issue := range quality {
		switch issue.Layer {
		case "narrative":
			foundNarrative = true
		case "expression":
			foundExpression = true
		case "story":
			t.Fatalf("lexical quality gate must not invent story issue: %+v", issue)
		}
	}
	if !foundNarrative || !foundExpression {
		t.Fatalf("quality layers narrative=%v expression=%v issues=%+v", foundNarrative, foundExpression, quality)
	}
}
