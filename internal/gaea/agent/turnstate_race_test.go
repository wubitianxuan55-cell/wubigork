package agent

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestTurnStateConcurrent is the regression test for the audit P0 race.
//
// executeOne runs in up to 8 parallel goroutines (executeBatch → runParallel)
// and mutated per-turn state without any lock:
//   - staleWrittenFiles / staleReadFiles (stale-anchor tracking)
//   - repeatSuccessCounts (loop guard)
//   - bgJobStartedThisTurn / bgOutputReadThisTurn / bgJobKilledThisTurn /
//     bgStartKillStreak (background start-kill detection)
//
// The first batch only races on the lazy `map = make(...)` pointer stores (the
// race detector flags those); from the second batch on, the maps are shared and
// sibling goroutines hit "fatal error: concurrent map writes" — a runtime throw
// recover() cannot catch, which is exactly why the whole desktop app crashed.
// The scenario below mirrors the audit: one parallel batch with three
// write_file calls to different paths (distinct conflict keys keep them in the
// SAME parallel batch) plus two read_file calls, repeated across rounds with
// per-round jitter, while checkBgStartKillCycle runs concurrently the way the
// run loop paces it against in-flight batch goroutines.
//
// Before the turnMu fix this test reproduces the race (fatal without -race,
// detected by -race); with the fix it must stay green.
func TestTurnStateConcurrent(t *testing.T) {
	var writes, reads, bashes, kills int32
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "write_file", readOnly: false, calls: &writes})
	reg.Add(fakeTool{name: "read_file", readOnly: true, calls: &reads})
	reg.Add(fakeTool{name: "bash", readOnly: false, calls: &bashes})
	reg.Add(fakeTool{name: "kill_shell", readOnly: false, calls: &kills})
	a := New(nil, reg, NewSession(""), Options{}, event.Discard)

	const rounds = 60
	batchDone := make(chan struct{})
	go func() {
		defer close(batchDone)
		for r := 0; r < rounds; r++ {
			// Distinct paths AND arguments per round: distinct conflict keys
			// keep the three writes + two reads in ONE parallel batch; distinct
			// loop-guard signatures and fresh paths keep every call unblocked so
			// the full state-mutating hot path (recordRepeatSuccess, stale
			// tracking, bg-flag switch) executes every round. repeatSuccessCounts
			// is only reset by a real Run — never here — so uniqueness matters.
			calls := []provider.ToolCall{
				{ID: fmt.Sprintf("w1-%d", r), Name: "write_file", Arguments: fmt.Sprintf(`{"path":"wa-%d.txt","content":"r%d"}`, r, r)},
				{ID: fmt.Sprintf("w2-%d", r), Name: "write_file", Arguments: fmt.Sprintf(`{"path":"wb-%d.txt","content":"r%d"}`, r, r)},
				{ID: fmt.Sprintf("w3-%d", r), Name: "write_file", Arguments: fmt.Sprintf(`{"path":"wc-%d.txt","content":"r%d"}`, r, r)},
				{ID: fmt.Sprintf("r1-%d", r), Name: "read_file", Arguments: fmt.Sprintf(`{"path":"ra-%d.txt"}`, r)},
				{ID: fmt.Sprintf("r2-%d", r), Name: "read_file", Arguments: fmt.Sprintf(`{"path":"rb-%d.txt"}`, r)},
				// bg-flag writers: bash(run_in_background) and kill_shell set
				// bgJobStartedThisTurn / bgJobKilledThisTurn from batch
				// goroutines while the checker loop below reads+writes them.
				{ID: fmt.Sprintf("b-%d", r), Name: "bash", Arguments: `{"command":"go test ./...","run_in_background":true}`},
				{ID: fmt.Sprintf("k-%d", r), Name: "kill_shell", Arguments: `{}`},
			}
			results := a.executeBatch(context.Background(), calls)
			if len(results) != len(calls) {
				t.Errorf("round %d: got %d results for %d calls", r, len(results), len(calls))
			}
			for i, res := range results {
				// No call may be blocked or error: blocking would silently skip
				// the map writes the test is hammering (loop guard, stale guard).
				if strings.HasPrefix(res, "blocked:") || strings.HasPrefix(res, "error:") ||
					strings.HasPrefix(res, "tool panic:") {
					t.Errorf("round %d call %d (%s): unexpected result %q", r, i, calls[i].Name, truncateStr(res, 120))
				}
			}
			time.Sleep(time.Millisecond) // jitter: different interleaving each round
		}
	}()
	// The run loop calls checkBgStartKillCycle around batch boundaries; hammering
	// it concurrently with in-flight batches is what forces the bg-flag
	// handoff (executeOne writers vs checker reader/incrementer) under turnMu.
	for i := 0; i < rounds*2; i++ {
		a.checkBgStartKillCycle()
		time.Sleep(500 * time.Microsecond)
	}
	<-batchDone

	// Every call must have actually executed: a race swallowed by the batch
	// recover() would show up above as "tool panic:", a silently skipped or
	// blocked call shows up here as a counter mismatch.
	if got := atomic.LoadInt32(&writes); got != 3*rounds {
		t.Errorf("write_file executions = %d, want %d", got, 3*rounds)
	}
	if got := atomic.LoadInt32(&reads); got != 2*rounds {
		t.Errorf("read_file executions = %d, want %d", got, 2*rounds)
	}
	if got := atomic.LoadInt32(&bashes); got != rounds {
		t.Errorf("bash executions = %d, want %d", got, rounds)
	}
	if got := atomic.LoadInt32(&kills); got != rounds {
		t.Errorf("kill_shell executions = %d, want %d", got, rounds)
	}
}
