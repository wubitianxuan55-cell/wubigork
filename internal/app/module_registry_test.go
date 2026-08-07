package app

import (
	"strings"
	"testing"
)

func TestModuleRegistryDispatch(t *testing.T) {
	reg := NewModuleRegistry()
	if err := reg.Register(Module{
		ID: "office", Name: "方案",
		Intents: []string{"create"},
		Handle: func(input map[string]any) (map[string]any, error) {
			return map[string]any{"created": true, "title": input["title"]}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	out, err := reg.Dispatch("office", "create", map[string]any{"title": "土壤修复标书"})
	if err != nil {
		t.Fatal(err)
	}
	if out["created"] != true || out["title"] != "土壤修复标书" {
		t.Fatalf("out = %+v", out)
	}
	if _, err := reg.Dispatch("novel", "create_chapter", nil); err == nil || !strings.Contains(err.Error(), "unknown module") {
		t.Fatalf("unknown module err = %v", err)
	}
	if _, err := reg.Dispatch("office", "unknown_intent", nil); err == nil || !strings.Contains(err.Error(), "unknown intent") {
		t.Fatalf("unknown intent err = %v", err)
	}
}
