package builtin

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestCompleteStepCompactSchemaPublishesKindEnum(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal((completeStep{}).CompactSchema(), &schema); err != nil {
		t.Fatal(err)
	}
	props := schema["properties"].(map[string]any)
	evidence := props["evidence"].(map[string]any)
	items := evidence["items"].(map[string]any)
	kind := items["properties"].(map[string]any)["kind"].(map[string]any)
	raw := kind["enum"].([]any)
	var got []string
	for _, v := range raw {
		got = append(got, v.(string))
	}
	for _, want := range []string{"verification", "diff", "files", "manual"} {
		if !slices.Contains(got, want) {
			t.Fatalf("compact complete_step enum missing %q: %v", want, got)
		}
	}
}

func TestCompleteStepInvalidKindReturnsCopyableExample(t *testing.T) {
	_, err := (completeStep{}).Execute(context.Background(), json.RawMessage(
		`{"step":"1","result":"done","evidence":[{"kind":"command","summary":"tests passed"}]}`))
	if err == nil {
		t.Fatal("invalid evidence kind should fail")
	}
	for _, want := range []string{"verification|diff|files|manual", "Copyable complete_step example", `"kind":"verification"`} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}
