package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenuiValidateToolContract(t *testing.T) {
	g := genuiValidate{}
	if g.Name() != "genui_validate" {
		t.Fatalf("name = %q", g.Name())
	}
	if !g.ReadOnly() {
		t.Fatal("tool must be read-only")
	}
}

func TestGenuiValidateToolExecute(t *testing.T) {
	g := genuiValidate{}
	out, err := g.Execute(context.Background(), json.RawMessage(`{"spec":"{\"items\":[{\"type\":\"stat\",\"label\":\"营收\",\"value\":\"¥128k\"}]}"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "OK") {
		t.Fatalf("want OK, got %q", out)
	}

	bad, err := g.Execute(context.Background(), json.RawMessage(`{"spec":"{\"items\":[{\"type\":\"evil\"}]}"}`))
	if err != nil {
		t.Fatalf("bad execute: %v", err)
	}
	if !strings.Contains(bad, "❌") || !strings.Contains(bad, "unknown") && !strings.Contains(bad, "未知") {
		t.Fatalf("want ❌ with reason, got %q", bad)
	}

	if _, err := g.Execute(context.Background(), json.RawMessage(`{"spec":"  "}`)); err == nil {
		t.Fatal("empty spec should error")
	}
}
