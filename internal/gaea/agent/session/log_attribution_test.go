package session

// B5 工具归因（第四批 P3）：会话日志 payload 的 Model/Turn 落盘与旧日志兼容。
//
// 口径：盖戳存在→payload 带 model/turn 字段；未盖戳→字段整体缺省
// （omitempty），旧日志与新日志在未盖戳路径上逐字节同形。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

func TestEntryFromEventToolAttributionStamped(t *testing.T) {
	e := event.Event{
		Kind: event.ToolDispatch,
		Turn: 3,
		Tool: event.Tool{ID: "c9", Name: "bash", Model: "test-model"},
	}
	entry, err := EntryFromEvent(e, 1700000000)
	if err != nil {
		t.Fatalf("EntryFromEvent: %v", err)
	}
	var p map[string]any
	if err := json.Unmarshal(entry.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if p["model"] != "test-model" {
		t.Errorf("payload model = %v, want test-model", p["model"])
	}
	if num, ok := p["turn"].(float64); !ok || num != 3 {
		t.Errorf("payload turn = %v, want 3", p["turn"])
	}
}

func TestEntryFromEventToolAttributionOmitted(t *testing.T) {
	e := event.Event{
		Kind: event.ToolDispatch,
		Tool: event.Tool{ID: "c1", Name: "bash"},
	}
	entry, err := EntryFromEvent(e, 1700000000)
	if err != nil {
		t.Fatalf("EntryFromEvent: %v", err)
	}
	s := string(entry.Payload)
	if strings.Contains(s, "\"model\"") || strings.Contains(s, "\"turn\"") {
		t.Errorf("unstamped payload must omit model/turn keys, got %s", s)
	}
}

func TestEntryFromEventResultAndHeaderAttribution(t *testing.T) {
	res := event.Event{
		Kind: event.ToolResult,
		Turn: 2,
		Tool: event.Tool{ID: "c2", Name: "grep", Output: "x", Model: "m-r"},
	}
	re, err := EntryFromEvent(res, 1700000000)
	if err != nil {
		t.Fatalf("EntryFromEvent(result): %v", err)
	}
	if !strings.Contains(string(re.Payload), "\"model\":\"m-r\"") || !strings.Contains(string(re.Payload), "\"turn\":2") {
		t.Errorf("result payload missing attribution: %s", re.Payload)
	}

	hdr := event.Event{
		Kind:   event.RequestHeader,
		Header: event.RequestHeaderInfo{Provider: "deepseek", Model: "deepseek-chat"},
	}
	he, err := EntryFromEvent(hdr, 1700000000)
	if err != nil {
		t.Fatalf("EntryFromEvent(header): %v", err)
	}
	if !strings.Contains(string(he.Payload), "\"model\":\"deepseek-chat\"") {
		t.Errorf("header payload missing model: %s", he.Payload)
	}
}
