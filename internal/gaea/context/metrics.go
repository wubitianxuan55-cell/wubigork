package context

import "sync"

// CacheMetrics tracks token savings and cache stability across a session.
// It supports hierarchical aggregation: child sub-agents report into their
// parent, and the root aggregates all children.
type CacheMetrics struct {
	mu sync.Mutex

	// Layer sizes (bytes)
	L1Size     int
	L2Size     int
	L4Messages int

	// Token savings
	SavedByCompact int64
	SavedByFork    int64
	ForkCount      int

	// Monetary savings (estimated)
	SavedUSD       float64
	SavedLatencyMs int64

	// Cache hit/miss (全会话)
	CacheHitTokens  int64
	CacheMissTokens int64
	BreakCount      int

	// Stability
	L2StableSince   int64
	L3Version       int
	CompactionCount int

	// Hierarchy
	parent   *CacheMetrics
	children []*CacheMetrics

	// merged marks a child whose metrics have already been merged into its
	// parent; MergeChild ignores children with this flag set to avoid
	// double counting.
	merged bool
}

// NewCacheMetrics creates a root-level metrics tracker.
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{}
}

// NewChild creates a child metrics tracker that reports to this parent.
func (m *CacheMetrics) NewChild() *CacheMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	child := &CacheMetrics{parent: m}
	m.children = append(m.children, child)
	return child
}

// MergeChild aggregates a completed child's metrics into the parent.
// It snapshots the child under the child's own lock and merges the same
// aggregation field set used by Report. Each child may be merged at most
// once: repeated calls for the same child are ignored.
func (m *CacheMetrics) MergeChild(child *CacheMetrics) {
	if child == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// Mark the child as merged under its own lock so duplicate-merge
	// detection is atomic even while the child is being recorded elsewhere.
	child.mu.Lock()
	if child.merged {
		child.mu.Unlock()
		return
	}
	child.merged = true
	child.mu.Unlock()

	// Snapshot the child under its own lock (child.Report) to avoid a torn
	// read of a child that is still being recorded concurrently.
	cr := child.Report()

	m.SavedByCompact += cr.SavedByCompact
	m.SavedByFork += cr.SavedByFork
	m.ForkCount += cr.ForkCount + 1
	m.SavedUSD += cr.SavedUSD
	m.SavedLatencyMs += cr.SavedLatencyMs
	m.CompactionCount += cr.CompactionCount
	m.CacheHitTokens += cr.CacheHitTokens
	m.CacheMissTokens += cr.CacheMissTokens
	m.BreakCount += cr.BreakCount

	// Remove child from children list so Report no longer aggregates it.
	for i, c := range m.children {
		if c == child {
			m.children = append(m.children[:i], m.children[i+1:]...)
			break
		}
	}
}

// RecordCompact records tokens saved by compaction.
func (m *CacheMetrics) RecordCompact(savedTokens int64, pricePerToken float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SavedByCompact += savedTokens
	m.CompactionCount++
	if pricePerToken > 0 {
		m.SavedUSD += float64(savedTokens) * pricePerToken
	}
}

// RecordFork records tokens saved by fork cache inheritance.
func (m *CacheMetrics) RecordFork(savedTokens int64, pricePerToken float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SavedByFork += savedTokens
	m.ForkCount++
	if pricePerToken > 0 {
		m.SavedUSD += float64(savedTokens) * pricePerToken
	}
}

// SetLayerSizes records the current layer sizes.
func (m *CacheMetrics) SetLayerSizes(l1, l2 int, l4msgs int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.L1Size = l1
	m.L2Size = l2
	m.L4Messages = l4msgs
}

// RecordCacheTurn accumulates per-turn cache hit/miss tokens.
func (m *CacheMetrics) RecordCacheTurn(hit, miss int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHitTokens += hit
	m.CacheMissTokens += miss
}

// RecordCacheBreak increments the break counter.
func (m *CacheMetrics) RecordCacheBreak() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BreakCount++
}

// SetL3Version records the current skill profile version.
func (m *CacheMetrics) SetL3Version(v int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.L3Version = v
}

// Report returns an aggregated metrics snapshot.
func (m *CacheMetrics) Report() CacheReport {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := CacheReport{
		L1Size:          m.L1Size,
		L2Size:          m.L2Size,
		L3Version:       m.L3Version,
		L4Messages:      m.L4Messages,
		SavedByCompact:  m.SavedByCompact,
		SavedByFork:     m.SavedByFork,
		ForkCount:       m.ForkCount,
		SavedUSD:        m.SavedUSD,
		SavedLatencyMs:  m.SavedLatencyMs,
		CompactionCount: m.CompactionCount,
		CacheHitTokens:  m.CacheHitTokens,
		CacheMissTokens: m.CacheMissTokens,
		BreakCount:      m.BreakCount,
	}

	// Aggregate children. The +1 fork count for the child itself follows the
	// same semantic as MergeChild.
	for _, c := range m.children {
		cr := c.Report()
		total.SavedByCompact += cr.SavedByCompact
		total.SavedByFork += cr.SavedByFork
		total.ForkCount += cr.ForkCount + 1
		total.SavedUSD += cr.SavedUSD
		total.SavedLatencyMs += cr.SavedLatencyMs
		total.CompactionCount += cr.CompactionCount
		total.CacheHitTokens += cr.CacheHitTokens
		total.CacheMissTokens += cr.CacheMissTokens
		total.BreakCount += cr.BreakCount
	}

	return total
}

// CacheReport is a read-only snapshot of cache metrics.
type CacheReport struct {
	L1Size          int     `json:"l1Size"`
	L2Size          int     `json:"l2Size"`
	L3Version       int     `json:"l3Version"`
	L4Messages      int     `json:"l4Messages"`
	SavedByCompact  int64   `json:"savedByCompact"`
	SavedByFork     int64   `json:"savedByFork"`
	ForkCount       int     `json:"forkCount"`
	SavedUSD        float64 `json:"savedUsd"`
	SavedLatencyMs  int64   `json:"savedLatencyMs"`
	CompactionCount int     `json:"compactionCount"`
	// V5.30: 全会话缓存命中统计
	CacheHitTokens  int64 `json:"cacheHitTokens"`
	CacheMissTokens int64 `json:"cacheMissTokens"`
	BreakCount      int   `json:"breakCount"`
}
