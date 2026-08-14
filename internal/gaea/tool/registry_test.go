package tool

import (
	"context"
	"encoding/json"
	"testing"
)

// stubTool is a minimal Tool for registry tests.
type stubTool struct {
	name   string
	schema json.RawMessage
}

func (s stubTool) Name() string        { return s.name }
func (s stubTool) Description() string { return s.name + " desc" }
func (s stubTool) Schema() json.RawMessage {
	if len(s.schema) > 0 {
		return s.schema
	}
	return json.RawMessage(`{"type":"object"}`)
}
func (s stubTool) Execute(context.Context, json.RawMessage) (string, error) { return "", nil }
func (s stubTool) ReadOnly() bool                                           { return true }

// TestRegistryRemovePrefix proves an MCP server's namespaced tools are dropped as
// a group on disconnect, leaving built-ins and other servers' tools — and their
// insertion order — intact.
func TestRegistryRemovePrefix(t *testing.T) {
	r := NewRegistry()
	r.Add(stubTool{name: "bash"})
	r.Add(stubTool{name: "mcp__fs__read"})
	r.Add(stubTool{name: "mcp__fs__write"})
	r.Add(stubTool{name: "mcp__stripe__charge"})

	if got := r.RemovePrefix("mcp__fs__"); got != 2 {
		t.Fatalf("RemovePrefix returned %d, want 2", got)
	}
	if r.Len() != 2 {
		t.Fatalf("registry has %d tools after removal, want 2", r.Len())
	}
	if _, ok := r.Get("mcp__fs__read"); ok {
		t.Errorf("mcp__fs__read should be gone")
	}
	if _, ok := r.Get("mcp__stripe__charge"); !ok {
		t.Errorf("another server's tool should survive")
	}
	want := []string{"bash", "mcp__stripe__charge"}
	got := r.Names()
	if len(got) != len(want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names = %v, want %v (order preserved)", got, want)
		}
	}

	// Removing a prefix that matches nothing is a no-op.
	if got := r.RemovePrefix("mcp__nope__"); got != 0 {
		t.Errorf("RemovePrefix on absent prefix returned %d, want 0", got)
	}
}

// TestRegistrySchemasSorted proves Schemas() emits tool definitions in
// deterministic alphabetical order regardless of insertion order, so a logically
// identical tool set produces a stable provider-facing request prefix (prompt
// cache reuse). Names() must stay in insertion order — only the provider export
// is sorted.
func TestRegistrySchemasSorted(t *testing.T) {
	r := NewRegistry()
	// Add deliberately out of alphabetical order.
	insertion := []string{"write", "bash", "read", "apply_patch"}
	for _, n := range insertion {
		r.Add(stubTool{name: n})
	}

	var got []string
	for _, s := range r.Schemas() {
		got = append(got, s.Name)
	}
	want := []string{"apply_patch", "bash", "read", "write"}
	if len(got) != len(want) {
		t.Fatalf("Schemas() names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Schemas() names = %v, want %v (alphabetical)", got, want)
		}
	}

	// The sort must not leak into Names(): display order stays insertion order.
	gotNames := r.Names()
	for i := range insertion {
		if gotNames[i] != insertion[i] {
			t.Fatalf("Names() = %v, want %v (insertion order)", gotNames, insertion)
		}
	}
}

func TestRegistrySchemasStableAndCanonical(t *testing.T) {
	r := NewRegistry()
	r.Add(stubTool{
		name:   "zeta",
		schema: json.RawMessage(`{"type":"object","required":["b","a"],"properties":{"b":{"type":"string"},"a":{"type":"string"}}}`),
	})
	r.Add(stubTool{
		name:   "alpha",
		schema: json.RawMessage(`{"required":["y","x"],"type":"object"}`),
	})

	schemas := r.Schemas()
	if len(schemas) != 2 {
		t.Fatalf("Schemas returned %d entries, want 2", len(schemas))
	}
	if schemas[0].Name != "alpha" || schemas[1].Name != "zeta" {
		t.Fatalf("Schemas order = %q, %q; want alpha, zeta", schemas[0].Name, schemas[1].Name)
	}
	if got, want := string(schemas[0].Parameters), `{"required":["x","y"],"type":"object"}`; got != want {
		t.Fatalf("alpha schema = %s, want %s", got, want)
	}
	if got, want := string(schemas[1].Parameters), `{"properties":{"a":{},"b":{}},"required":["a","b"],"type":"object"}`; got != want {
		t.Fatalf("zeta schema = %s, want %s", got, want)
	}
}

func TestRegistrySchemasCanonicalizesEquivalentOrdering(t *testing.T) {
	first := NewRegistry()
	first.Add(stubTool{
		name:   "same",
		schema: json.RawMessage(`{"type":"object","required":["b","a"],"properties":{"b":{"description":"bee","type":"string"},"a":{"type":"integer"}}}`),
	})

	second := NewRegistry()
	second.Add(stubTool{
		name:   "same",
		schema: json.RawMessage(`{"properties":{"a":{"type":"integer"},"b":{"type":"string","description":"bee"}},"required":["a","b"],"type":"object"}`),
	})

	firstSchemas := first.Schemas()
	secondSchemas := second.Schemas()
	if got, want := string(firstSchemas[0].Parameters), string(secondSchemas[0].Parameters); got != want {
		t.Fatalf("equivalent schemas canonicalized differently:\n  first:  %s\n  second: %s", got, want)
	}
}

// persistWriterStub is a stubTool marked as a persistent-write tool.
type persistWriterStub struct {
	stubTool
}

func (persistWriterStub) PersistWrite() bool { return true }

// TestPersistWriteNamesAndMarker 验证 T6-2.5 注册表化：带 PersistWrite 标记的
// 工具被 IsPersistWrite 识别，并按注册顺序出现在 Registry.PersistWriteNames()
// 中；未标记工具不出现。这是子代理禁写集合与审批硬门的数据来源。
func TestPersistWriteNamesAndMarker(t *testing.T) {
	r := NewRegistry()
	r.Add(stubTool{name: "read_file"})
	r.Add(persistWriterStub{stubTool: stubTool{name: "cost_save"}})
	r.Add(stubTool{name: "bash"})
	r.Add(persistWriterStub{stubTool: stubTool{name: "remember"}})

	if IsPersistWrite(stubTool{name: "bash"}) {
		t.Error("未标记工具不应被识别为持久化写工具")
	}
	if !IsPersistWrite(persistWriterStub{stubTool: stubTool{name: "remember"}}) {
		t.Error("带标记工具应被识别为持久化写工具")
	}

	got := r.PersistWriteNames()
	want := []string{"cost_save", "remember"}
	if len(got) != len(want) {
		t.Fatalf("PersistWriteNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("PersistWriteNames = %v, want %v（注册顺序）", got, want)
		}
	}

	// 覆盖替换（Add 同名）后标记仍生效；PersistWriteNames 不含重复项。
	r.Add(persistWriterStub{stubTool: stubTool{name: "cost_save"}})
	if got := r.PersistWriteNames(); len(got) != 2 {
		t.Fatalf("覆盖后 PersistWriteNames = %v, want 2 项", got)
	}
}
