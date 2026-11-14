package agent

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func TestHeaderToolsFromSchemasPreservesAuthoritativeSource(t *testing.T) {
	got := headerToolsFromSchemas([]provider.ToolSchema{{
		Name:        "read_file",
		Description: "read",
		Parameters:  []byte(`{"type":"object"}`),
		Source:      "builtin-workspace",
	}})
	if len(got) != 1 || got[0].Source != "builtin-workspace" {
		t.Fatalf("header tool source = %+v", got)
	}
	if got[0].Name != "read_file" || got[0].Schema == "" {
		t.Fatalf("header tool payload incomplete: %+v", got[0])
	}
}
