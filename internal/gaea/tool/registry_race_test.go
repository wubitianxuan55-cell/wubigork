package tool

import (
	"fmt"
	"sync"
	"testing"
)

// The tests in this file cover the Registry concurrency contract fixed alongside
// the sync.RWMutex introduction in tool.go:
//
//  1. TestRegistryConcurrent — writers (Add, RemovePrefix, SuspendPrefix/
//     ResumePrefix: the registry surface MCP hot-plug exercises from the Wails
//     binding goroutine via Controller.AddMCPServer / RemoveMCPServer) race
//     against readers (Get/Names/Schemas/Len/PersistWriteNames: the surface the
//     agent turn loop exercises via agent_stream.go). Before the fix this
//     data-raced on tools/order/hidden/canon/suspended and could corrupt the
//     maps; run with -race to verify.
//
//  2. TestRegistrySuspendedGhost — a suspended-prefix Add must not leave a
//     "ghost" name in order without a Tool behind it: before the fix, Add
//     appended to order before the suspended early-return, and Schemas() later
//     dereferenced the missing Tool and panicked.

// fakeMCPToolName builds a model-visible MCP tool name for a fake server, the
// same "mcp__<server>__<tool>" shape SplitMCPName expects.
func fakeMCPToolName(server string, i int) string {
	return fmt.Sprintf("mcp__%s__tool%d", server, i)
}

// TestRegistryConcurrent hammers one Registry from ≥8 goroutines mixing writes
// and reads across several independent rounds. Final-state assertions stay
// weak (interleavings are nondeterministic) but must hold after every round:
// no ghost names in order, every schema entry backed by a real tool.
func TestRegistryConcurrent(t *testing.T) {
	const (
		rounds  = 8  // independent registries
		workers = 12 // concurrent goroutines per round (≥8)
		iters   = 50 // operations per worker
	)

	for round := 0; round < rounds; round++ {
		r := NewRegistry()
		r.Add(stubTool{name: "bash"})
		r.Add(stubTool{name: "read_file"})

		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				server := fmt.Sprintf("srv%d", w%3) // three fake MCP servers
				prefix := "mcp__" + server + "__"
				for i := 0; i < iters; i++ {
					switch w % 4 {
					case 0: // MCP hot-plug connect: register tools, occasionally drop the server
						name := fakeMCPToolName(server, i%5)
						r.Add(stubTool{name: name})
						r.Get(name)
						if i%10 == 9 {
							r.RemovePrefix(prefix)
						}
					case 1: // per-session suspend / resume cycling (V10.0)
						if i%7 == 0 {
							r.SuspendPrefix(prefix)
						}
						r.Names()
						r.Len()
						if i%7 == 3 {
							r.ResumePrefix(prefix)
						}
					case 2: // turn loop: schema export + persist-write set
						for _, s := range r.Schemas() {
							_ = s.Description // touching the entry proves it is fully built
						}
						r.PersistWriteNames()
					default: // plain turn-loop lookups
						r.Get("bash")
						r.Get(fakeMCPToolName(server, 0))
						r.Names()
					}
				}
			}(w)
		}
		wg.Wait()

		// Post-round consistency: Names() and Len() agree, no duplicates,
		// every name resolves to a Tool (no ghosts), and every exported
		// schema is fully populated (a nil Tool would have panicked).
		names := r.Names()
		if len(names) != r.Len() {
			t.Fatalf("round %d: len(Names())=%d but Len()=%d", round, len(names), r.Len())
		}
		seen := make(map[string]bool, len(names))
		for _, n := range names {
			if seen[n] {
				t.Fatalf("round %d: duplicate name %q in order", round, n)
			}
			seen[n] = true
			if _, ok := r.Get(n); !ok {
				t.Fatalf("round %d: ghost name %q in Names() with no Tool behind it", round, n)
			}
		}
		for _, s := range r.Schemas() {
			if s.Name == "" || s.Description == "" || len(s.Parameters) == 0 {
				t.Fatalf("round %d: incomplete schema entry %+v", round, s)
			}
		}
	}
}

// TestRegistrySuspendedGhost proves a suspended-prefix Add is a clean no-op:
// Get reports absent, and Names()/Schemas() neither contain the name nor panic
// (before the fix, Schemas() dereferenced the missing Tool and panicked).
func TestRegistrySuspendedGhost(t *testing.T) {
	r := NewRegistry()
	r.Add(stubTool{name: "bash"})

	const ghost = "mcp__gone__tool"
	if n := r.SuspendPrefix("mcp__gone__"); n != 0 {
		t.Fatalf("SuspendPrefix on absent prefix returned %d, want 0", n)
	}

	// Add under a suspended prefix must be silently rejected.
	r.Add(stubTool{name: ghost})

	if _, ok := r.Get(ghost); ok {
		t.Errorf("Get(%q) returned ok=true, want false", ghost)
	}
	if r.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 (suspended Add must not leave a ghost in order)", r.Len())
	}
	for _, n := range r.Names() {
		if n == ghost {
			t.Errorf("Names() contains ghost name %q", ghost)
		}
	}

	// Schemas() must not contain the ghost — and must not panic, which the
	// unfixed code did here via a nil Tool's Description().
	schemas := r.Schemas()
	if len(schemas) != 1 {
		t.Fatalf("Schemas() returned %d entries, want 1", len(schemas))
	}
	if schemas[0].Name != "bash" {
		t.Fatalf("Schemas()[0].Name = %q, want %q", schemas[0].Name, "bash")
	}

	// Repeated suspended Adds must not accumulate ghost entries either.
	for i := 0; i < 3; i++ {
		r.Add(stubTool{name: ghost})
	}
	if r.Len() != 1 || len(r.Names()) != 1 {
		t.Fatalf("after repeated suspended Adds: Len()=%d Names()=%v, want 1/[bash]", r.Len(), r.Names())
	}

	// Resume re-enables registration of the same name.
	r.ResumePrefix("mcp__gone__")
	r.Add(stubTool{name: ghost})
	if _, ok := r.Get(ghost); !ok {
		t.Errorf("Get(%q) after ResumePrefix returned ok=false, want true", ghost)
	}
	if r.Len() != 2 {
		t.Fatalf("Len() = %d, want 2 after resume + Add", r.Len())
	}
}
