// Package tool defines the Tool abstraction and a Registry. Built-in tools live
// in tool/builtin and self-register via init(); plugin-provided tools are added
// to a runtime Registry alongside the enabled built-ins. The agent sees only a
// *Registry, never the global built-in set directly.
package tool

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// Tool is a capability the model can invoke.
type Tool interface {
	Name() string
	Description() string
	// Schema returns the JSON Schema for the tool's parameters.
	Schema() json.RawMessage
	// Execute parses the model-generated raw JSON args and returns result text
	// to feed back to the model.
	Execute(ctx context.Context, args json.RawMessage) (string, error)
	// ReadOnly reports whether the tool has no observable side effects on the
	// host. The agent parallelises a batch of tool calls only when every call
	// in the batch is ReadOnly; mixed batches stay sequential so write/read
	// ordering is preserved. bash and plugin tools must return false because
	// their effects can't be inferred statically from args.
	ReadOnly() bool
}

// ToolContext carries session-scoped information that a tool may need beyond
// its arguments: the conversation history, the calling agent, and identifiers
// for the session, message, and tool call. Borrowed from opencode.
type ToolContext struct {
	SessionID  string
	MessageID  string
	AgentName  string
	ToolCallID string
	// Messages is the full conversation history up to (but not including)
	// the current tool call. Read-only — tools must not mutate it.
	Messages []provider.Message
}

// ContextualTool is an optional interface a Tool may implement to receive
// richer session context alongside the standard context.Context. Tools that
// don't implement this continue to work with Execute(ctx, args) alone.
// Borrowed from opencode's ToolContext pattern.
type ContextualTool interface {
	Tool
	ExecuteWithContext(ctx context.Context, tc ToolContext, args json.RawMessage) (string, error)
}

// CompactDescriptor is an optional capability a Tool may implement. When present,
// CompactDescription replaces Description and CompactSchema replaces Schema in
// the provider-facing tool list, significantly reducing per-turn prompt tokens.
// Tools that don't implement this fall back to their full Description + Schema.
type CompactDescriptor interface {
	CompactDescription() string
	CompactSchema() json.RawMessage
}

// PersistWriteTool is an optional capability a Tool may implement to declare
// that it performs persistent writes to shared stores (cost library / memory /
// knowledge base / skill files). Sub-agents are never given such tools — they
// run on a headless approval channel where inheriting them would bypass the
// parent's per-item confirmation — and interactive gates may treat them as
// always-ask. The set is derived from this marker (see IsPersistWrite and
// Registry.PersistWriteNames), so registering a new persistent-write tool is a
// one-line declaration; no call-site whitelists need updating.
type PersistWriteTool interface {
	PersistWrite() bool
}

// IsPersistWrite reports whether t is marked as a persistent-write tool.
func IsPersistWrite(t Tool) bool {
	pw, ok := t.(PersistWriteTool)
	return ok && pw.PersistWrite()
}

// --- process-global built-in set (populated by builtin subpackage init) ---

var builtins = map[string]Tool{}

// RegisterBuiltin registers a compile-time built-in tool. Intended for init().
// It panics on a duplicate name, which is a compile-time wiring mistake.
func RegisterBuiltin(t Tool) {
	name := t.Name()
	if _, dup := builtins[name]; dup {
		panic("tool: duplicate built-in " + name)
	}
	builtins[name] = t
}

// Builtins returns all registered built-in tools, sorted by name.
func Builtins() []Tool {
	names := make([]string, 0, len(builtins))
	for n := range builtins {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Tool, 0, len(names))
	for _, n := range names {
		out = append(out, builtins[n])
	}
	return out
}

// LookupBuiltin returns a registered built-in by name.
func LookupBuiltin(name string) (Tool, bool) {
	t, ok := builtins[name]
	return t, ok
}

// --- per-run registry instance ---

// Registry is a per-run set of tools: enabled built-ins plus plugin tools.
// V6.0 P8: supports hiding tools from the model schema while keeping them callable.
//
// Concurrency: mu guards every field below. Readers (Get/Len/Names/
// PersistWriteNames/Schemas/FilteredSchemas) take RLock; writers (Add/Hide/
// HideUnlessOnly/RemovePrefix/SuspendPrefix/ResumePrefix) take Lock. This is
// required because MCP hot-plug mutates the registry from the Wails binding
// goroutine (Controller.AddMCPServer → Add, RemoveMCPServer/DisconnectMCPServer
// → RemovePrefix) while the agent turn loop reads it concurrently
// (agent_stream.go → Schemas).
//
// Lock order: Registry.mu is a leaf lock. No method may call tool-implemented
// code (Name/Schema/Description/CompactDescription/CompactSchema/PersistWrite)
// or another Registry method while holding mu. Values that require tool code —
// the canonicalized schema in Add, descriptions in FilteredSchemas, persist-write
// markers in PersistWriteNames — are computed from a snapshot after the lock is
// released, so a tool implementation can never re-enter (and deadlock against)
// the registry.
type Registry struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	order     []string
	hidden    map[string]bool            // V6.0 P8: hidden from schema but still callable
	canon     map[string]json.RawMessage // V10.0: schema canonicalized once on Add, reused by Schemas()
	suspended map[string]bool            // V10.0: MCP prefixes temporarily disabled per-session
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}, hidden: map[string]bool{}, canon: map[string]json.RawMessage{}, suspended: map[string]bool{}}
}

// Add inserts (or replaces) a tool, preserving first-seen order.
// V10.0: canonicalizes the schema once here — Schemas() reuses the cached result.
//
// Ghost-name rule: a tool whose name carries a suspended prefix (see
// SuspendPrefix) is silently rejected BEFORE anything is appended to order, so
// a rejected Add never leaves a name in order without a Tool behind it (which
// would make Schemas() dereference a nil Tool and panic). The order append and
// the tools/canon stores happen in one locked critical section, so they cannot
// be observed (or left) inconsistent.
//
// Schema canonicalization runs BEFORE the write lock is taken: Schema() is
// tool-implemented code and Registry.mu is a leaf lock (see Registry doc). Note
// this means Schema() is invoked even when the Add ends up suspended-rejected;
// implementations must keep it side-effect free (as for every registration).
func (r *Registry) Add(t Tool) {
	name := t.Name()
	canon := provider.CanonicalizeSchema(t.Schema())

	r.mu.Lock()
	defer r.mu.Unlock()
	for prefix := range r.suspended {
		if strings.HasPrefix(name, prefix) {
			return // silently reject — prefix is suspended; do not touch order
		}
	}
	if _, ok := r.tools[name]; !ok {
		r.order = append(r.order, name)
	}
	r.tools[name] = t
	r.canon[name] = canon
}

// Hide removes a tool from the model-visible schema list without unregistering it.
// Hidden tools remain callable via Get(). V6.0 P8: reduces model cognitive load.
func (r *Registry) Hide(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hidden[name] = true
}

// HideUnlessOnly hides each given name only when the registry also contains at
// least one of the alternatives — so the model always has at least one way to
// perform the operation. V6.0 P8.
func (r *Registry) HideUnlessOnly(names []string, alternatives []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hasAlt := false
	for _, a := range alternatives {
		if _, ok := r.tools[a]; ok {
			hasAlt = true
			break
		}
	}
	if !hasAlt {
		return // don't hide if no alternative available
	}
	for _, n := range names {
		r.hidden[n] = true
	}
}

// MCPNamePrefix is the namespace every MCP tool name carries: the
// model-visible name is "mcp__<server>__<tool>".
const MCPNamePrefix = "mcp__"

// SplitMCPName splits a model-visible MCP tool name "mcp__<server>__<tool>" into
// its server and tool parts. ok is false for non-MCP (built-in) names and for
// malformed names missing either part.
func SplitMCPName(name string) (server, tool string, ok bool) {
	if !strings.HasPrefix(name, MCPNamePrefix) {
		return "", "", false
	}
	rest := name[len(MCPNamePrefix):]
	parts := strings.SplitN(rest, "__", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// RemovePrefix unregisters every tool whose name starts with prefix — used to
// drop an MCP server's "mcp__<server>__" namespace when it's disconnected — and
// returns the count removed.
func (r *Registry) RemovePrefix(prefix string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.order[:0]
	removed := 0
	for _, name := range r.order {
		if strings.HasPrefix(name, prefix) {
			delete(r.tools, name)
			delete(r.canon, name)
			removed++
			continue
		}
		kept = append(kept, name)
	}
	r.order = kept
	return removed
}

// SuspendPrefix unregisters every tool whose name starts with prefix, and
// prevents future Add calls for that prefix until ResumePrefix is called.
// Used for per-session MCP disables — an in-flight background handshake
// may attempt to re-add tools for the suspended prefix.
func (r *Registry) SuspendPrefix(prefix string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.suspended[prefix] = true
	kept := r.order[:0]
	removed := 0
	for _, name := range r.order {
		if strings.HasPrefix(name, prefix) {
			delete(r.tools, name)
			delete(r.canon, name)
			removed++
			continue
		}
		kept = append(kept, name)
	}
	r.order = kept
	return removed
}

// ResumePrefix allows future Add calls for a previously suspended prefix.
func (r *Registry) ResumePrefix(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.suspended, prefix)
}

// Get looks up a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Len returns the number of registered tools.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.order)
}

// Names returns the registered tool names in insertion order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// PersistWriteNames returns the names of all registered tools marked as
// persistent-write tools (PersistWriteTool), in registration order. Sub-agent
// registry filtering and approval gates derive their forbidden/always-ask sets
// from this so a newly marked tool takes effect automatically.
//
// The (name, tool) pairs are snapshotted under RLock; the PersistWrite marker
// calls happen after the lock is released — tool code must never run under
// Registry.mu (leaf lock, see Registry doc).
func (r *Registry) PersistWriteNames() []string {
	r.mu.RLock()
	type pair struct {
		name string
		t    Tool
	}
	pairs := make([]pair, 0, len(r.order))
	for _, name := range r.order {
		if t, ok := r.tools[name]; ok {
			pairs = append(pairs, pair{name: name, t: t})
		}
	}
	r.mu.RUnlock()

	out := make([]string, 0, 2)
	for _, p := range pairs {
		if IsPersistWrite(p.t) {
			out = append(out, p.name)
		}
	}
	return out
}

// Schemas exports tool definitions in stable name order for the provider.
// When a tool implements CompactDescriptor, the compact versions are used
// instead of the full Description + Schema, reducing per-turn prompt tokens.
// V6.0 P8: hidden tools are excluded from the schema list.
// V10.0: standard schemas use pre-canonicalized cache from Add().
func (r *Registry) Schemas() []provider.ToolSchema {
	return r.FilteredSchemas(nil)
}

// FilteredSchemas is like Schemas but only includes tools whose names appear
// in the names slice. When names is nil or empty, all non-hidden tools are
// included (equivalent to Schemas()). Tools not found in the registry are
// silently skipped.
//
// Registry state (order, tools, hidden set, canonical schemas) is snapshotted
// under a single RLock — no nested locking — and Description/CompactDescription/
// CompactSchema are called after the lock is released, so tool code can never
// re-enter the registry while it is locked (leaf-lock rule, see Registry doc).
// A name that has no Tool behind it is skipped rather than dereferenced: the
// schema export can never panic on a nil Tool.
func (r *Registry) FilteredSchemas(names []string) []provider.ToolSchema {
	r.mu.RLock()
	var filter map[string]bool
	if len(names) > 0 {
		filter = make(map[string]bool, len(names))
		for _, n := range names {
			filter[n] = true
		}
	}
	type snapshot struct {
		name  string
		t     Tool
		canon json.RawMessage
	}
	snaps := make([]snapshot, 0, len(r.order))
	for _, name := range r.order {
		if r.hidden[name] {
			continue
		}
		if filter != nil && !filter[name] {
			continue
		}
		t, ok := r.tools[name]
		if !ok {
			continue // defensive ghost-name guard: skip instead of nil-dereference
		}
		snaps = append(snaps, snapshot{name: name, t: t, canon: r.canon[name]})
	}
	r.mu.RUnlock()

	sort.Slice(snaps, func(i, j int) bool { return snaps[i].name < snaps[j].name })

	out := make([]provider.ToolSchema, 0, len(snaps))
	for _, s := range snaps {
		desc := s.t.Description()
		if cd, ok := s.t.(CompactDescriptor); ok {
			desc = cd.CompactDescription()
			schema := cd.CompactSchema()
			// Compact schemas are context-dependent — canonicalize inline.
			out = append(out, provider.ToolSchema{
				Name:        s.t.Name(),
				Description: desc,
				Parameters:  provider.CanonicalizeSchema(schema),
			})
		} else {
			// Standard schema — use pre-canonicalized cache from Add().
			out = append(out, provider.ToolSchema{
				Name:        s.t.Name(),
				Description: desc,
				Parameters:  s.canon,
			})
		}
	}
	return out
}
