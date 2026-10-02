// External test package (tool_test) on purpose: the production
// CompactDescriptor implementers below all import this package, so an
// in-package test file could not import them back.
package tool_test

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/largefile"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"

	// Side-effect import: registers every compile-time built-in tool so the
	// tool.Builtins() enumeration below covers the whole builtin compact
	// table (builtin/compact.go).
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// productionCompactTools enumerates every production CompactDescriptor that is
// reachable through exported constructors:
//
//   - every self-registering built-in (tool/builtin — compactDesc/compactSchema
//     tables in builtin/compact.go),
//   - agent.TaskTool ("task"),
//   - skill run_skill / install_skill. boot's unexported skillUseCountTool
//     wrapper (boot/skilluse.go) forwards Compact* to its wrapped run_skill,
//     so pinning run_skill pins the wrapper's served schema too,
//   - largefile summarize_file.
//
// Not mechanically reachable from here, and why that is safe:
//   - plugin remoteTool (plugin/plugin.go): CompactSchema returns t.Schema()
//     verbatim, so the subset relation holds by construction (also pinned by
//     plugin's own canonicalize_test.go).
//   - internal/app tools (ocr/semantic_search/image_gen/diagram/routine_llm/
//     translate_text/fact_*): unexported types behind the Wails App wiring.
//     Their compact schemas were manually verified against their full schemas
//     during the GA2-03 audit; changes there need the same one-line subset
//     check added inside that package.
func productionCompactTools(t *testing.T) map[string]tool.Tool {
	t.Helper()
	tools := map[string]tool.Tool{}
	for _, tl := range tool.Builtins() {
		tools[tl.Name()] = tl
	}
	// Nil collaborators are fine: only the static description/schema accessors
	// are exercised below, never Execute/Description-dependent wiring.
	tools["task"] = agent.NewTaskTool(nil, nil, tool.NewRegistry(), 20, 100_000, 0.3, "", "", nil)
	tools["run_skill"] = skill.NewRunSkillTool(nil, nil)
	tools["install_skill"] = skill.NewInstallSkillTool(nil, nil)
	tools["summarize_file"] = largefile.NewSummarizeTool(nil)

	for _, want := range []string{"read_file", "bash", "browser_click", "task", "run_skill", "install_skill", "summarize_file"} {
		if _, ok := tools[want]; !ok {
			t.Fatalf("enumeration lost %q — productionCompactTools no longer covers all implementers", want)
		}
	}
	return tools
}

// TestCompactSchemaIsSubsetOfFullSchema is the GA2-03 drift pin. FilteredSchemas
// serves the compact variant unconditionally for any tool implementing
// tool.CompactDescriptor, so the full Schema() a tool also carries is dead as
// served output — its only remaining job is source of truth for the compact
// variant. This test keeps that derivation honest: every field (and required
// entry) the compact schema exposes must exist in the full schema, recursively
// through properties/items. A compact field dropped from (or renamed in) the
// full schema — the "drifted dead asset" failure mode — fails here.
func TestCompactSchemaIsSubsetOfFullSchema(t *testing.T) {
	tools := productionCompactTools(t)
	found := 0
	for name, tl := range tools {
		cd, ok := tl.(tool.CompactDescriptor)
		if !ok {
			continue
		}
		found++
		var full, compact any
		if err := json.Unmarshal(tl.Schema(), &full); err != nil {
			t.Errorf("%s: full Schema() is not valid JSON: %v", name, err)
			continue
		}
		if err := json.Unmarshal(cd.CompactSchema(), &compact); err != nil {
			t.Errorf("%s: CompactSchema() is not valid JSON: %v", name, err)
			continue
		}
		assertSchemaSubset(t, name, "", full, compact)
	}
	// Guard the sweep itself: if the compact implementations evaporate the pin
	// would pass vacuously.
	if found < 40 {
		t.Fatalf("only %d CompactDescriptor implementers enumerated, want >= 40", found)
	}
}

// assertSchemaSubset asserts compact ⊆ full for one schema node:
//   - compact.required ⊆ full.required
//   - compact.required ⊆ compact.properties (well-formedness)
//   - every compact property exists in full and recurses
//   - compact "type", where present on both sides, matches
//   - compact.items exists in full and recurses
func assertSchemaSubset(t *testing.T, toolName, path string, full, compact any) {
	t.Helper()
	cf, ok := compact.(map[string]any)
	if !ok {
		return // leaf scalar — nothing to compare at this node
	}
	ff, _ := full.(map[string]any)

	if ff == nil {
		t.Errorf("%s%s: compact schema node exists but full Schema() has no object here — drift", toolName, path)
		return
	}

	// type consistency where both sides declare one
	if ct, ok := cf["type"].(string); ok {
		if ft, ok := ff["type"].(string); ok && ct != ft {
			t.Errorf("%s%s: compact type %q != full type %q — drift", toolName, path, ct, ft)
		}
	}

	// required: compact ⊆ full
	reqF := map[string]bool{}
	for _, r := range arr(ff["required"]) {
		if s, ok := r.(string); ok {
			reqF[s] = true
		}
	}
	propsC, _ := cf["properties"].(map[string]any)
	for _, r := range arr(cf["required"]) {
		s, ok := r.(string)
		if !ok {
			continue
		}
		if !reqF[s] {
			t.Errorf("%s%s: compact requires %q but full schema does not — drift", toolName, path, s)
		}
		if _, ok := propsC[s]; !ok {
			t.Errorf("%s%s: compact requires %q but its own properties lack it — malformed compact schema", toolName, path, s)
		}
	}

	// properties: keys compact ⊆ full, recurse
	propsF, _ := ff["properties"].(map[string]any)
	for k, v := range propsC {
		fv, ok := propsF[k]
		if !ok {
			t.Errorf("%s%s: compact property %q missing from full Schema() — drifted dead asset", toolName, path, k)
			continue
		}
		assertSchemaSubset(t, toolName, path+".properties."+k, fv, v)
	}

	// items: recurse when compact declares one
	if itemsC, ok := cf["items"]; ok {
		assertSchemaSubset(t, toolName, path+".items", ff["items"], itemsC)
	}
}

func arr(v any) []any {
	a, _ := v.([]any)
	return a
}
