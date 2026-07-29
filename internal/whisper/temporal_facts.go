// Package whisper — temporal_facts.go
// 100% 对齐 ackem context/temporalFacts.ts
// 从事实库加载 PLANS/COMMITMENTS 事实

package whisper

// LoadTemporalFacts 从事实库加载 PLANS/COMMITMENTS（CTX-B）
func LoadTemporalFacts(store *FactStore) []TemporalFactRef {
	if store == nil {
		return nil
	}
	var refs []TemporalFactRef
	for _, f := range store.ListActive() {
		if f.Subcategory == "PLANS" || f.Subcategory == "COMMITMENTS" {
			refs = append(refs, TemporalFactRef{
				Subcategory: f.Subcategory,
				Summary:     f.Summary,
			})
		}
	}
	return refs
}
