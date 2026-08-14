package context

import "testing"

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
