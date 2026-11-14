package contextview

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func TestDnaAtReplaysReadOrderForRequestHeaders(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "first question"}),
		entry(3, "request_header", headerPayload("system v1", "read_file", "bash")),
		entry(4, "assistant_message", map[string]any{"text": "first answer"}),
		entry(5, "user_message", map[string]any{"content": "second question"}),
		entry(7, "request_header", headerPayload("system v1", "read_file", "bash")),
	}
	samples := DnaAt(entries, []int64{3, 7, 99}, 1_000_000)
	if len(samples) != 3 || !samples[0].Ok || !samples[1].Ok || samples[2].Ok {
		t.Fatalf("samples = %+v", samples)
	}
	if len(samples[0].Bands) < 3 || samples[0].Bands[0].Cat != catSystem || samples[0].Bands[1].Cat != catTools {
		t.Fatalf("read-order head = %+v", samples[0].Bands)
	}
	if samples[0].Bands[1].Label != "read_file" || samples[0].Bands[2].Label != "bash" {
		t.Fatalf("tool schemas should be separate bands: %+v", samples[0].Bands[:3])
	}
	firstCats := dnaCats(samples[0].Bands)
	if firstCats[len(firstCats)-1] == catAssistant {
		t.Fatalf("first request must not see its later answer: %v", firstCats)
	}
	secondCats := dnaCats(samples[1].Bands)
	foundAssistant := false
	for _, cat := range secondCats {
		if cat == catAssistant {
			foundAssistant = true
		}
	}
	if !foundAssistant {
		t.Fatalf("second request should include first answer: %v", secondCats)
	}
	if samples[2].Bands == nil || len(samples[2].Bands) != 0 {
		t.Fatalf("unknown seq must be empty, got %+v", samples[2])
	}
}

func TestDnaAtRejectsNonHeader(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "user_message", map[string]any{"content": "hi"}),
	}
	sample := DnaAt(entries, []int64{1}, 0)[0]
	if sample.Ok {
		t.Fatalf("non-header seq must not be accepted: %+v", sample)
	}
}

func dnaCats(bands []DnaBand) []string {
	out := make([]string, 0, len(bands))
	for _, band := range bands {
		out = append(out, band.Cat)
	}
	return out
}
