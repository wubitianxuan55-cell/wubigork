package tool

import (
	"fmt"
	"sync"
	"testing"
)

// The tests in this file cover the Registry concurrency contract fixed alongside
// the sync.RWMutex introduction in tool.go:
//
// TestRegistryConcurrent — writers (Add, RemovePrefix: the registry surface MCP
// hot-plug exercises from the Wails binding goroutine via
// Controller.AddMCPServer / RemoveMCPServer) race against readers
// (Get/Names/Schemas/Len/PersistWriteNames: the surface the agent turn loop
// exercises via agent_stream.go). Before the fix this data-raced on
// tools/order/hidden/canon and could corrupt the maps; run with -race to verify.
// Final-state assertions stay weak (interleavings are nondeterministic) but must
// hold after every round: no ghost names in order (Add appends to order and
// stores the Tool in one locked critical section), every schema entry backed by
// a real tool.

// fakeMCPToolName builds a model-visible MCP tool name for a fake server, the
// same "mcp__<server>__<tool>" shape SplitMCPName expects.
func fakeMCPToolName(server string, i int) string {
	return fmt.Sprintf("mcp__%s__tool%d", server, i)
}

// TestRegistryConcurrent hammers one Registry from ≥8 goroutines mixing writes
// and reads across several independent rounds.
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
					case 1: // server reconnect churn: drop-then-re-add the same set
						if i%7 == 0 {
							r.RemovePrefix(prefix)
						}
						r.Names()
						r.Len()
						if i%7 == 3 {
							r.Add(stubTool{name: fakeMCPToolName(server, i%5)})
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
