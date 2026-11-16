package app

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/event"
)

func TestGaeaToolSourcesWireAndResync(t *testing.T) {
	sources := []event.ToolSource{{
		Title:   "Go 1.24",
		URL:     "https://go.dev/doc/go1.24",
		Snippet: "release notes",
		Source:  "bing",
	}}
	result := event.Event{Kind: event.ToolResult, Tool: event.Tool{
		ID: "call-1", Name: "web_search", Output: `[]`, Sources: sources,
	}}
	payload := gaeaEventMap(result)
	tool, ok := payload["tool"].(map[string]interface{})
	if !ok {
		t.Fatalf("tool payload = %#v", payload["tool"])
	}
	got, ok := tool["sources"].([]event.ToolSource)
	if !ok || len(got) != 1 || got[0].URL != sources[0].URL {
		t.Fatalf("wire sources = %#v", tool["sources"])
	}

	dispatch, err := session.EntryFromEvent(event.Event{Kind: event.ToolDispatch, Tool: event.Tool{
		ID: "call-1", Name: "web_search", Args: `{"query":"go"}`,
	}}, 100)
	if err != nil {
		t.Fatal(err)
	}
	logged, err := session.EntryFromEvent(result, 101)
	if err != nil {
		t.Fatal(err)
	}
	items := foldResyncItems([]session.LogEntry{dispatch, logged})
	if len(items) != 1 || len(items[0].Sources) != 1 || items[0].Sources[0].Title != "Go 1.24" {
		t.Fatalf("resync items = %+v", items)
	}
}
