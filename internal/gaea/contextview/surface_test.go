package contextview

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func TestSurfaceAtReplaysHistoricalRequest(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "first question"}),
		entry(3, "request_header", headerPayload("system v1", "read_file")),
		entry(4, "assistant_message", map[string]any{"text": "first answer"}),
		entry(5, "user_message", map[string]any{"content": "second question"}),
		entry(7, "request_header", headerPayload("system v1", "read_file")),
	}

	first, ok := SurfaceAt(entries, 3, 1_000_000)
	if !ok || first.Seq != 3 {
		t.Fatalf("first surface = %+v, ok=%v", first, ok)
	}
	if surfaceNodeText(first.Nodes, catUser) != "first question" {
		t.Fatalf("first request user = %q", surfaceNodeText(first.Nodes, catUser))
	}
	if surfaceNodeText(first.Nodes, catAssistant) != "" {
		t.Fatalf("first request must not see its later assistant answer: %+v", first.Nodes)
	}

	second, ok := SurfaceAt(entries, 7, 1_000_000)
	if !ok {
		t.Fatal("second request not found")
	}
	if surfaceNodeText(second.Nodes, catAssistant) != "first answer" {
		t.Fatalf("second request assistant = %q", surfaceNodeText(second.Nodes, catAssistant))
	}
	userCount := 0
	for _, node := range second.Nodes {
		if node.Cat == catUser {
			userCount++
		}
	}
	if userCount != 2 {
		t.Fatalf("second request should see both user messages, got %d: %+v", userCount, second.Nodes)
	}
}

func TestSurfaceAtRejectsNonHeaderAndUnknownSeq(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "user_message", map[string]any{"content": "hi"}),
		entry(2, "request_header", headerPayload("sys", "read_file")),
	}
	if _, ok := SurfaceAt(entries, 1, 1_000_000); ok {
		t.Fatal("user_message seq must not be accepted as a historical request")
	}
	if _, ok := SurfaceAt(entries, 99, 1_000_000); ok {
		t.Fatal("unknown seq must not fall back to live surface")
	}
}

func surfaceNodeText(nodes []SurfaceNode, cat string) string {
	for _, node := range nodes {
		if node.Cat == cat {
			return node.Text
		}
	}
	return ""
}
