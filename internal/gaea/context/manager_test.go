package context

import (
	stdctx "context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cache"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// stubTool implements tool.Tool for FilteredSchemas tests.
type stubTool struct {
	name   string
	desc   string
	schema json.RawMessage
}

func (s stubTool) Name() string                                                { return s.name }
func (s stubTool) Description() string                                         { return s.desc }
func (s stubTool) Schema() json.RawMessage                                     { return s.schema }
func (s stubTool) Execute(_ stdctx.Context, _ json.RawMessage) (string, error) { return "", nil }
func (s stubTool) ReadOnly() bool                                              { return true }

func newTestManager() *ContextManager {
	identity := NewIdentityLayer("TEST_IDENTITY")
	rt := NewRuntimeLayer()
	rt.SetProject(ProjectState{Language: "Go", RootPath: "wubigrok"})
	skill := cache.NewSkillLayer()
	flow := NewFlowLayer(DefaultCompactPolicy())
	return NewContextManager(identity, rt, skill, flow)
}

func TestContextManagerProcessFirstTurn(t *testing.T) {
	cm := newTestManager()
	tc := cm.ProcessFirstTurn("fix the bug in main.go")

	if tc.Profile.Kind != KindFixBug {
		t.Errorf("Profile.Kind = %q, want fix_bug", tc.Profile.Kind)
	}
	if tc.Temperature != 0.3 {
		t.Errorf("Temperature = %v, want 0.3", tc.Temperature)
	}
	if tc.MaxSteps != 20 || tc.RetryLimit != 3 {
		t.Errorf("MaxSteps/RetryLimit = %d/%d, want 20/3", tc.MaxSteps, tc.RetryLimit)
	}
	if !cm.Runtime().IsLocked() {
		t.Error("runtime must be locked after first turn")
	}
	if cm.Metrics().L3Version != 1 {
		t.Errorf("L3Version = %d, want 1", cm.Metrics().L3Version)
	}
	// L1 + L2 system messages, empty flow
	if len(tc.SystemPrompt) != 2 {
		t.Fatalf("SystemPrompt = %d messages, want 2", len(tc.SystemPrompt))
	}
	if tc.SystemPrompt[0].Role != provider.RoleSystem || tc.SystemPrompt[0].Content != "TEST_IDENTITY" {
		t.Errorf("first message = %+v, want system TEST_IDENTITY", tc.SystemPrompt[0])
	}
	if tc.SystemPrompt[1].Role != provider.RoleSystem {
		t.Errorf("second message role = %q, want system", tc.SystemPrompt[1].Role)
	}
}

func TestContextManagerProcessTurn(t *testing.T) {
	cm := newTestManager()
	cm.ProcessFirstTurn("fix the bug in main.go")
	tc := cm.ProcessTurn()
	if tc.Profile.Kind != KindFixBug {
		t.Errorf("ProcessTurn Profile.Kind = %q, want fix_bug", tc.Profile.Kind)
	}
	if len(tc.SystemPrompt) != 2 {
		t.Errorf("ProcessTurn SystemPrompt = %d messages, want 2", len(tc.SystemPrompt))
	}
}

func TestContextManagerAssemblePrompt(t *testing.T) {
	cm := newTestManager()
	cm.Flow().Add(provider.Message{Role: provider.RoleUser, Content: "hello"})
	cm.Flow().Add(provider.Message{Role: provider.RoleAssistant, Content: "hi"})

	msgs := cm.AssemblePrompt()
	if len(msgs) != 4 {
		t.Fatalf("AssemblePrompt = %d messages, want 4 (L1+L2+flow)", len(msgs))
	}
	if msgs[0].Role != provider.RoleSystem || msgs[0].Content != "TEST_IDENTITY" {
		t.Errorf("msgs[0] = %+v, want L1 system", msgs[0])
	}
	if msgs[1].Role != provider.RoleSystem {
		t.Errorf("msgs[1] role = %q, want system (L2)", msgs[1].Role)
	}
	if msgs[2].Role != provider.RoleUser || msgs[2].Content != "hello" {
		t.Errorf("msgs[2] = %+v, want flow user hello", msgs[2])
	}
	if msgs[3].Role != provider.RoleAssistant || msgs[3].Content != "hi" {
		t.Errorf("msgs[3] = %+v, want flow assistant hi", msgs[3])
	}

	r := cm.Metrics()
	if r.L1Size != len("TEST_IDENTITY") {
		t.Errorf("L1Size = %d, want %d", r.L1Size, len("TEST_IDENTITY"))
	}
	if r.L4Messages != 2 {
		t.Errorf("L4Messages = %d, want 2", r.L4Messages)
	}
	if r.L2Size <= 0 {
		t.Errorf("L2Size = %d, want > 0 (project state)", r.L2Size)
	}
}

func TestContextManagerForkIndependent(t *testing.T) {
	cm := newTestManager()
	cm.OnFileEdited("src/a.go")

	child := cm.Fork(ForkIndependent, "research task")
	if child.Identity().SystemPrompt() != "TEST_IDENTITY" {
		t.Error("child identity must share L1 system prompt")
	}
	if got := child.Runtime().RecentEdits(); len(got) != 0 {
		t.Errorf("ForkIndependent session edits = %v, want empty (session isolated)", got)
	}
	// project state is shared
	if !strings.Contains(child.Runtime().SystemPrompt(), "wubigrok") {
		t.Error("child runtime must contain project RootPath wubigrok")
	}
	if child.Flow().Len() != 0 {
		t.Errorf("child flow Len = %d, want 0", child.Flow().Len())
	}
}

func TestContextManagerForkCollaborative(t *testing.T) {
	cm := newTestManager()
	cm.OnFileEdited("src/a.go")

	child := cm.Fork(ForkCollaborative, "review task")
	edits := child.Runtime().RecentEdits()
	if len(edits) != 1 || edits[0].Path != "src/a.go" {
		t.Errorf("collaborative child edits = %v, want [src/a.go]", edits)
	}
}

func TestContextManagerRecordMetrics(t *testing.T) {
	cm := newTestManager()
	cm.RecordCompact(100, 0.001)
	cm.RecordFork(50, 0.002)
	r := cm.Metrics()
	if r.SavedByCompact != 100 || r.SavedByFork != 50 {
		t.Errorf("metrics = %+v, want compact 100 fork 50", r)
	}
	// no-op RecordOutcome must not panic
	cm.RecordOutcome(KindFixBug, true)
}

func TestIdentityFingerprint(t *testing.T) {
	a := NewIdentityLayer("stable prompt")
	b := NewIdentityLayer("stable prompt")
	if a.ContentHash() != b.ContentHash() {
		t.Error("identical prompts must produce identical fingerprints")
	}
	c := NewIdentityLayer("changed prompt")
	if a.ContentHash() == c.ContentHash() {
		t.Error("different prompts must produce different fingerprints")
	}
	if len(a.ContentHash()) != 64 {
		t.Errorf("hash length = %d, want 64 (sha256 hex)", len(a.ContentHash()))
	}
}

func TestIdentityHashRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l := NewIdentityLayer("stable prompt")
	if err := l.SaveHash(dir); err != nil {
		t.Fatalf("SaveHash: %v", err)
	}
	if !l.LoadAndCompareHash(dir) {
		t.Error("LoadAndCompareHash must be true after SaveHash (cache warm)")
	}
	// changed prompt → cache cold (miss)
	changed := NewIdentityLayer("different prompt")
	if changed.LoadAndCompareHash(dir) {
		t.Error("changed prompt must report cache cold")
	}
	// missing dir → cold
	if NewIdentityLayer("x").LoadAndCompareHash(filepath.Join(t.TempDir(), "missing")) {
		t.Error("missing dir must report cache cold")
	}
	// empty dir → cold
	if NewIdentityLayer("x").LoadAndCompareHash("") {
		t.Error("empty dir must report cache cold")
	}
}

func TestIdentityFilteredSchemas(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(stubTool{name: "read_file", desc: "read", schema: json.RawMessage(`{"type":"object"}`)})
	reg.Add(stubTool{name: "edit_file", desc: "edit", schema: json.RawMessage(`{"type":"object"}`)})

	identity := NewIdentityLayer("prompt")
	identity.SetRegistry(reg)
	schemas := identity.FilteredSchemas([]string{"read_file"})
	if len(schemas) != 1 || schemas[0].Name != "read_file" {
		t.Errorf("FilteredSchemas = %+v, want only read_file", schemas)
	}
	// unknown name is skipped
	schemas = identity.FilteredSchemas([]string{"read_file", "nonexistent"})
	if len(schemas) != 1 || schemas[0].Name != "read_file" {
		t.Errorf("FilteredSchemas with unknown = %+v, want only read_file", schemas)
	}
	// nil registry → nil
	noReg := NewIdentityLayer("p")
	if got := noReg.FilteredSchemas([]string{"read_file"}); got != nil {
		t.Errorf("nil registry FilteredSchemas = %v, want nil", got)
	}
}
