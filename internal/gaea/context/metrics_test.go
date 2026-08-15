package context

import (
	"reflect"
	"sync"
	"testing"
)

func TestCacheMetricsRecordCompact(t *testing.T) {
	m := NewCacheMetrics()
	m.RecordCompact(1000, 0.001)
	m.RecordCompact(500, 0.001)
	r := m.Report()
	if r.SavedByCompact != 1500 {
		t.Errorf("SavedByCompact = %d, want 1500", r.SavedByCompact)
	}
	if r.CompactionCount != 2 {
		t.Errorf("CompactionCount = %d, want 2", r.CompactionCount)
	}
	if r.SavedUSD != 1.5 {
		t.Errorf("SavedUSD = %v, want 1.5", r.SavedUSD)
	}
}

func TestCacheMetricsRecordFork(t *testing.T) {
	m := NewCacheMetrics()
	m.RecordFork(2000, 0.002)
	m.RecordFork(3000, 0)
	r := m.Report()
	if r.SavedByFork != 5000 {
		t.Errorf("SavedByFork = %d, want 5000", r.SavedByFork)
	}
	if r.ForkCount != 2 {
		t.Errorf("ForkCount = %d, want 2", r.ForkCount)
	}
	// zero pricePerToken contributes no USD
	if r.SavedUSD != 4.0 {
		t.Errorf("SavedUSD = %v, want 4.0", r.SavedUSD)
	}
}

func TestCacheMetricsLayerSizesAndCacheTurns(t *testing.T) {
	m := NewCacheMetrics()
	m.SetLayerSizes(120, 340, 5)
	m.RecordCacheTurn(100, 20)
	m.RecordCacheTurn(300, 10)
	m.RecordCacheBreak()
	m.SetL3Version(3)
	r := m.Report()
	if r.L1Size != 120 || r.L2Size != 340 || r.L4Messages != 5 {
		t.Errorf("layer sizes = L1 %d L2 %d L4 %d", r.L1Size, r.L2Size, r.L4Messages)
	}
	if r.CacheHitTokens != 400 || r.CacheMissTokens != 30 {
		t.Errorf("cache tokens = hit %d miss %d, want 400/30", r.CacheHitTokens, r.CacheMissTokens)
	}
	if r.BreakCount != 1 {
		t.Errorf("BreakCount = %d, want 1", r.BreakCount)
	}
	if r.L3Version != 3 {
		t.Errorf("L3Version = %d, want 3", r.L3Version)
	}
}

func TestCacheMetricsChildMerge(t *testing.T) {
	parent := NewCacheMetrics()
	child := parent.NewChild()
	child.RecordCompact(500, 0.001)
	child.RecordFork(1000, 0.002)
	child.SetLayerSizes(10, 20, 1)

	// Report aggregates live children
	r := parent.Report()
	if r.SavedByCompact != 500 || r.SavedByFork != 1000 {
		t.Errorf("aggregated children = compact %d fork %d, want 500/1000", r.SavedByCompact, r.SavedByFork)
	}
	if r.SavedUSD != 2.5 {
		t.Errorf("aggregated USD = %v, want 2.5", r.SavedUSD)
	}

	parent.MergeChild(child)
	r2 := parent.Report()
	if r2.SavedByCompact != 500 || r2.SavedByFork != 1000 {
		t.Errorf("after MergeChild = compact %d fork %d", r2.SavedByCompact, r2.SavedByFork)
	}
	// child is removed from the children list after merge
	if len(parent.children) != 0 {
		t.Errorf("children after merge = %d, want 0", len(parent.children))
	}
}

func TestCacheMetricsMergeChildIncrementsForkCount(t *testing.T) {
	parent := NewCacheMetrics()
	child := parent.NewChild()
	child.RecordFork(100, 0.001)
	parent.MergeChild(child)
	r := parent.Report()
	// MergeChild adds child.ForkCount + 1 (the child itself is a fork)
	if r.ForkCount != 2 {
		t.Errorf("ForkCount after merge = %d, want 2 (child 1 + itself)", r.ForkCount)
	}
}

func TestCacheMetricsReportAggregatesLiveChildAllFields(t *testing.T) {
	parent := NewCacheMetrics()
	child := parent.NewChild()
	child.RecordCompact(500, 0.001)
	child.RecordFork(1000, 0.002)
	child.RecordCacheTurn(400, 30)
	child.RecordCacheTurn(200, 10)
	child.RecordCacheBreak()
	child.RecordCacheBreak()

	r := parent.Report()
	if r.SavedByCompact != 500 {
		t.Errorf("SavedByCompact = %d, want 500", r.SavedByCompact)
	}
	if r.CompactionCount != 1 {
		t.Errorf("CompactionCount = %d, want 1", r.CompactionCount)
	}
	if r.SavedByFork != 1000 {
		t.Errorf("SavedByFork = %d, want 1000", r.SavedByFork)
	}
	if r.SavedUSD != 2.5 {
		t.Errorf("SavedUSD = %v, want 2.5", r.SavedUSD)
	}
	if r.CacheHitTokens != 600 {
		t.Errorf("CacheHitTokens = %d, want 600", r.CacheHitTokens)
	}
	if r.CacheMissTokens != 40 {
		t.Errorf("CacheMissTokens = %d, want 40", r.CacheMissTokens)
	}
	if r.BreakCount != 2 {
		t.Errorf("BreakCount = %d, want 2", r.BreakCount)
	}
	// The live child itself counts as one fork.
	if r.ForkCount != 2 {
		t.Errorf("ForkCount = %d, want 2 (child 1 + itself)", r.ForkCount)
	}
}

func TestCacheMetricsMergeChildMatchesReport(t *testing.T) {
	parent := NewCacheMetrics()
	parent.SetLayerSizes(100, 200, 3)
	parent.SetL3Version(2)

	child := parent.NewChild()
	child.RecordCompact(500, 0.001)
	child.RecordCompact(300, 0)
	child.RecordFork(1000, 0.002)
	child.RecordCacheTurn(400, 30)
	child.RecordCacheTurn(200, 10)
	child.RecordCacheBreak()
	child.SetLayerSizes(10, 20, 1)
	child.SetL3Version(7)

	before := parent.Report()
	parent.MergeChild(child)
	after := parent.Report()

	if !reflect.DeepEqual(before, after) {
		t.Errorf("Report before MergeChild differs from after:\nbefore = %+v\nafter  = %+v", before, after)
	}
	// Spot-check that every aggregation field was carried over.
	if after.SavedByCompact != 800 {
		t.Errorf("SavedByCompact = %d, want 800", after.SavedByCompact)
	}
	if after.CompactionCount != 2 {
		t.Errorf("CompactionCount = %d, want 2", after.CompactionCount)
	}
	if after.SavedByFork != 1000 {
		t.Errorf("SavedByFork = %d, want 1000", after.SavedByFork)
	}
	if after.SavedUSD != 2.5 {
		t.Errorf("SavedUSD = %v, want 2.5", after.SavedUSD)
	}
	if after.CacheHitTokens != 600 {
		t.Errorf("CacheHitTokens = %d, want 600", after.CacheHitTokens)
	}
	if after.CacheMissTokens != 40 {
		t.Errorf("CacheMissTokens = %d, want 40", after.CacheMissTokens)
	}
	if after.BreakCount != 1 {
		t.Errorf("BreakCount = %d, want 1", after.BreakCount)
	}
	if after.ForkCount != 2 {
		t.Errorf("ForkCount = %d, want 2 (child 1 + itself)", after.ForkCount)
	}
	// Layer sizes/version are the parent's own values, not aggregated.
	if after.L1Size != 100 || after.L2Size != 200 || after.L4Messages != 3 || after.L3Version != 2 {
		t.Errorf("parent layer fields changed: %+v", after)
	}
}

func TestCacheMetricsForkCountMergeReportConsistent(t *testing.T) {
	parent := NewCacheMetrics()
	child1 := parent.NewChild()
	child1.RecordFork(100, 0.001)
	child1.RecordFork(200, 0.001)
	child2 := parent.NewChild() // no own forks

	live := parent.Report()
	// child1 contributes ForkCount 2 + 1 (itself); child2 contributes 0 + 1.
	if live.ForkCount != 4 {
		t.Errorf("live Report ForkCount = %d, want 4", live.ForkCount)
	}

	parent.MergeChild(child1)
	parent.MergeChild(child2)
	merged := parent.Report()
	if merged.ForkCount != live.ForkCount {
		t.Errorf("merged ForkCount = %d, want %d (same caliber as live Report)", merged.ForkCount, live.ForkCount)
	}
	if merged.ForkCount != 4 {
		t.Errorf("merged ForkCount = %d, want 4", merged.ForkCount)
	}
}

func TestCacheMetricsMergeChildDuplicateIgnored(t *testing.T) {
	parent := NewCacheMetrics()
	child := parent.NewChild()
	child.RecordCompact(500, 0.001)
	child.RecordFork(1000, 0.002)
	child.RecordCacheTurn(400, 30)
	child.RecordCacheBreak()

	parent.MergeChild(child)
	parent.MergeChild(child)
	parent.MergeChild(child)

	r := parent.Report()
	if r.SavedByCompact != 500 || r.CompactionCount != 1 {
		t.Errorf("duplicate merge double-counted compact: %+v", r)
	}
	if r.SavedByFork != 1000 {
		t.Errorf("duplicate merge double-counted fork: %+v", r)
	}
	if r.SavedUSD != 2.5 {
		t.Errorf("duplicate merge double-counted USD: %+v", r)
	}
	if r.CacheHitTokens != 400 || r.CacheMissTokens != 30 {
		t.Errorf("duplicate merge double-counted cache tokens: %+v", r)
	}
	if r.BreakCount != 1 {
		t.Errorf("duplicate merge double-counted break: %+v", r)
	}
	if r.ForkCount != 2 {
		t.Errorf("ForkCount = %d, want 2 (child 1 + itself)", r.ForkCount)
	}
	if len(parent.children) != 0 {
		t.Errorf("children after merge = %d, want 0", len(parent.children))
	}
	if !child.merged {
		t.Error("child.merged not set after MergeChild")
	}
}

func TestCacheMetricsMergeChildConcurrentRecordNoRace(t *testing.T) {
	parent := NewCacheMetrics()
	child := parent.NewChild()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				child.RecordCacheTurn(1, 1)
				child.RecordCompact(1, 0.001)
				child.RecordFork(1, 0.002)
				child.RecordCacheBreak()
			}
		}()
	}

	// Merge while the child is still being recorded: MergeChild snapshots
	// under child.mu, so this must not race (verified under -race).
	parent.MergeChild(child)
	wg.Wait()

	// Merged snapshot is internally consistent (no torn reads).
	r := parent.Report()
	if int(r.SavedByCompact) != r.CompactionCount {
		t.Errorf("compact tokens (%d) != compaction count (%d)", r.SavedByCompact, r.CompactionCount)
	}
	if int(r.SavedByFork) != r.ForkCount-1 {
		t.Errorf("fork tokens (%d) != fork count-1 (%d)", r.SavedByFork, r.ForkCount-1)
	}
}

func TestCacheMetricsMergeChildConcurrentDuplicateNoRace(t *testing.T) {
	parent := NewCacheMetrics()
	child := parent.NewChild()
	child.RecordCompact(500, 0.001)
	child.RecordFork(1000, 0.002)
	child.RecordCacheTurn(400, 30)
	child.RecordCacheBreak()

	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parent.MergeChild(child)
		}()
	}
	wg.Wait()

	r := parent.Report()
	if r.ForkCount != 2 {
		t.Errorf("ForkCount = %d, want 2 (child 1 + itself)", r.ForkCount)
	}
	if r.SavedByCompact != 500 || r.CompactionCount != 1 {
		t.Errorf("concurrent duplicate merge double-counted compact: %+v", r)
	}
	if r.SavedByFork != 1000 || r.SavedUSD != 2.5 {
		t.Errorf("concurrent duplicate merge double-counted fork/usd: %+v", r)
	}
	if r.CacheHitTokens != 400 || r.CacheMissTokens != 30 || r.BreakCount != 1 {
		t.Errorf("concurrent duplicate merge double-counted cache/break: %+v", r)
	}
	if len(parent.children) != 0 {
		t.Errorf("children after merge = %d, want 0", len(parent.children))
	}
}
