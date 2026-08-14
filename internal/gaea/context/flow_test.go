package context

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// stubStore is a minimal in-memory MessageStore for injection tests.
type stubStore struct {
	msgs []provider.Message
}

func (s *stubStore) Append(m provider.Message) error {
	s.msgs = append(s.msgs, m)
	return nil
}

func (s *stubStore) Range(start, end int) ([]provider.Message, error) {
	if start < 0 {
		start = 0
	}
	if end > len(s.msgs) {
		end = len(s.msgs)
	}
	if start >= end {
		return nil, nil
	}
	return s.msgs[start:end], nil
}

func (s *stubStore) Len() int { return len(s.msgs) }

func (s *stubStore) Truncate(n int) error {
	if n < len(s.msgs) {
		s.msgs = s.msgs[:n]
	}
	return nil
}

func (s *stubStore) Close() error { return nil }

func TestFlowLayerAddAndMessages(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	flow.Add(provider.Message{Role: provider.RoleUser, Content: "u1"})
	flow.Add(provider.Message{Role: provider.RoleAssistant, Content: "a1"})
	if flow.Len() != 2 {
		t.Fatalf("Len = %d, want 2", flow.Len())
	}
	msgs := flow.Messages()
	if len(msgs) != 2 || msgs[0].Content != "u1" || msgs[1].Content != "a1" {
		t.Errorf("Messages = %v, want [u1 a1]", msgs)
	}

	flow.ReplaceMessages([]provider.Message{{Role: provider.RoleUser, Content: "replacement"}})
	if flow.Len() != 1 {
		t.Fatalf("Len after ReplaceMessages = %d, want 1", flow.Len())
	}
	msgs = flow.Messages()
	if len(msgs) != 1 || msgs[0].Content != "replacement" {
		t.Errorf("Messages after replace = %v, want [replacement]", msgs)
	}
}

func TestFlowLayerMessagesEmpty(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	msgs := flow.Messages()
	if msgs == nil || len(msgs) != 0 {
		t.Errorf("empty flow Messages = %v, want empty non-nil slice", msgs)
	}
}

func TestFlowLayerStoreInjection(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	stub := &stubStore{}
	flow.SetStore(stub)
	flow.Add(provider.Message{Role: provider.RoleUser, Content: "via-stub"})
	if stub.Len() != 1 || stub.msgs[0].Content != "via-stub" {
		t.Errorf("stub store did not receive message: %+v", stub.msgs)
	}
	// nil is a no-op
	flow.SetStore(nil)
	if flow.Store() != MessageStore(stub) {
		t.Error("SetStore(nil) must keep the existing store")
	}
}

func TestFlowLayerDetailRingEviction(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	// max ring = 5 by default; push 6 with a low-importance entry
	flow.PushDetailWithImportance("high1", 0.9)
	flow.PushDetailWithImportance("low", 0.1)
	flow.PushDetailWithImportance("mid1", 0.5)
	flow.PushDetailWithImportance("mid2", 0.6)
	flow.PushDetailWithImportance("mid3", 0.7)
	flow.PushDetailWithImportance("high2", 0.95)

	entries := flow.rings.entries
	if len(entries) != 5 {
		t.Fatalf("ring size = %d, want 5 after eviction", len(entries))
	}
	for _, e := range entries {
		if e.Content == "low" {
			t.Error("low-importance entry should have been evicted")
		}
	}
}

func TestFlowLayerDetailRingTieBreakOldest(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	// deterministic clock so equal-importance entries have distinct timestamps
	seq := int64(0)
	oldNow := nowNano
	nowNano = func() int64 { seq += 1000; return seq }
	defer func() { nowNano = oldNow }()

	flow.PushDetailWithImportance("oldest", 0.5)
	flow.PushDetailWithImportance("mid", 0.5)
	flow.PushDetailWithImportance("newest", 0.5)
	flow.PushDetailWithImportance("extra", 0.5)
	flow.PushDetailWithImportance("extra2", 0.5)
	flow.PushDetailWithImportance("extra3", 0.5)
	flow.PushDetailWithImportance("extra4", 0.5)

	entries := flow.rings.entries
	if len(entries) != 5 {
		t.Fatalf("ring size = %d, want 5", len(entries))
	}
	// the two oldest (same importance) must be evicted first
	for _, e := range entries {
		if e.Content == "oldest" || e.Content == "mid" {
			t.Errorf("oldest entry %q should have been evicted on importance tie", e.Content)
		}
	}
}

func TestFlowLayerRecentDetail(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	flow.PushDetailWithImportance("The API key is expired for module auth", 0.9)
	flow.PushDetailWithImportance("Cache hit ratio dropped after restart", 0.7)

	if got := flow.RecentDetail("API KEY"); got == "" {
		t.Error("RecentDetail should match case-insensitively")
	}
	if got := flow.RecentDetail("cache"); !strings.Contains(got, "Cache hit ratio") {
		t.Errorf("RecentDetail(\"cache\") = %q, want the cache detail", got)
	}
	if got := flow.RecentDetail("nonexistent"); got != "" {
		t.Errorf("RecentDetail(missing) = %q, want empty", got)
	}
	if got := flow.RecentDetail(""); got != "" {
		t.Errorf("RecentDetail(\"\") = %q, want empty", got)
	}
}

func TestFlowLayerDynamicMax(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	if flow.dynamicMax() != 3 {
		t.Errorf("dynamicMax(0 msgs) = %d, want 3", flow.dynamicMax())
	}
	for i := 0; i < 20; i++ {
		flow.Add(provider.Message{Role: provider.RoleUser, Content: "m"})
	}
	if flow.dynamicMax() != 5 {
		t.Errorf("dynamicMax(20 msgs) = %d, want 5", flow.dynamicMax())
	}
	for i := 0; i < 30; i++ {
		flow.Add(provider.Message{Role: provider.RoleUser, Content: "m"})
	}
	if flow.dynamicMax() != 8 {
		t.Errorf("dynamicMax(50 msgs) = %d, want 8", flow.dynamicMax())
	}
}

func TestFlowLayerPushDetailImportanceFunc(t *testing.T) {
	flow := NewFlowLayer(CompactPolicy{
		ImportanceFunc: func(detail string) float64 {
			if strings.Contains(detail, "critical") {
				return 1.0
			}
			return 0.2
		},
	})
	flow.PushDetail("critical: payment gateway down")
	flow.PushDetail("minor: typo in docs")
	flow.PushDetail("minor2: cosmetic")
	flow.PushDetail("minor3: formatting")

	entries := flow.rings.entries
	if len(entries) != 3 {
		t.Fatalf("ring size = %d, want 3 (dynamicMax for empty store)", len(entries))
	}
	foundCritical := false
	for _, e := range entries {
		if e.Content == "critical: payment gateway down" {
			foundCritical = true
			if e.Importance != 1.0 {
				t.Errorf("critical importance = %v, want 1.0", e.Importance)
			}
		}
	}
	if !foundCritical {
		t.Error("critical detail was evicted despite highest importance")
	}
}

func TestFlowLayerCompactPolicyAccessors(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	if flow.Window() != 0 || flow.Ratio() != 0.8 || flow.TailTokens() != 16384 {
		t.Errorf("default policy = win %d ratio %v tail %d, want 0/0.8/16384",
			flow.Window(), flow.Ratio(), flow.TailTokens())
	}
	flow.SetCompactPolicy(CompactPolicy{Window: 1000, Ratio: 0.5, TailTokens: 512})
	if flow.Window() != 1000 || flow.Ratio() != 0.5 || flow.TailTokens() != 512 {
		t.Errorf("updated policy not reflected")
	}
	p := flow.CompactPolicy()
	if p.Window != 1000 || p.Ratio != 0.5 || p.TailTokens != 512 {
		t.Errorf("CompactPolicy() = %+v", p)
	}
}

func TestFlowLayerDetailDir(t *testing.T) {
	flow := NewFlowLayer(DefaultCompactPolicy())
	flow.SetDetailDir(".gaea/deep")
	if flow.DetailDir() != ".gaea/deep" {
		t.Errorf("DetailDir = %q, want .gaea/deep", flow.DetailDir())
	}
}
