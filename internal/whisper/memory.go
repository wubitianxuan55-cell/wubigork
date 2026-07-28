package whisper

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// ─── FactStore ────────────────────────────────────────────────

type FactStore struct {
	facts []MemoryFact
	byID  map[string]*MemoryFact
}

func NewFactStore() *FactStore {
	return &FactStore{byID: make(map[string]*MemoryFact)}
}

func (fs *FactStore) ListActive() []MemoryFact {
	var r []MemoryFact
	for _, f := range fs.facts {
		if f.Status == "active" { r = append(r, f) }
	}
	return r
}

func (fs *FactStore) Add(raw MemoryFact) MemoryFact {
	for i := range fs.facts {
		e := &fs.facts[i]
		if e.Status != "active" { continue }
		if e.Domain == raw.Domain && e.Subcategory == raw.Subcategory &&
			e.Subject == raw.Subject && jaccard(e.Summary, raw.Summary) > 0.42 {
			if raw.Weight > e.Weight { e.Weight = raw.Weight + 0.5 } else { e.Weight += 0.5 }
			if raw.Confidence > e.Confidence { e.Confidence = raw.Confidence }
			if len(raw.Summary) > len(e.Summary) { e.Summary = raw.Summary }
			e.Triggers = mergeStrs(e.Triggers, raw.Triggers)
			e.UpdatedAt = time.Now()
			return *e
		}
	}
	raw.CreatedAt = time.Now(); raw.UpdatedAt = time.Now()
	if raw.Status == "" { raw.Status = "active" }
	fs.facts = append(fs.facts, raw)
	fs.byID[raw.ID] = &fs.facts[len(fs.facts)-1]
	return raw
}

func (fs *FactStore) SearchByTriggers(msg string) []MemoryFact {
	m := strings.ToLower(msg)
	var r []MemoryFact
	for _, f := range fs.facts {
		if f.Status != "active" { continue }
		for _, t := range f.Triggers {
			if strings.Contains(m, strings.ToLower(t)) { r = append(r, f); break }
		}
	}
	return r
}

func ScoreRelevance(f MemoryFact, now time.Time, valence, aff float64) float64 {
	days := now.Sub(f.CreatedAt).Hours() / 24
	if days < 0 { days = 0 }
	lambda := 0.003
	decay := math.Exp(-lambda * days)
	score := f.Weight * decay * f.SelfRelevance * 1.5
	if math.Abs(f.Confidence-valence) < 0.3 {
		boost := 1.5
		if math.Abs(aff) >= 50 { boost = 1.2 }
		score *= boost
	}
	hours := now.Sub(f.UpdatedAt).Hours()
	if hours < 4 { score *= 1.8 }
	return score
}

func (fs *FactStore) SelectForInjection(budget int, minConf, valence, aff float64) []MemoryFact {
	now := time.Now()
	type sc struct{ f MemoryFact; s float64 }
	var ranked []sc
	for _, f := range fs.facts {
		if f.Status != "active" || f.Confidence < minConf { continue }
		ranked = append(ranked, sc{f, ScoreRelevance(f, now, valence, aff)})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })
	var r []MemoryFact
	chars := 0
	for _, x := range ranked {
		block := len([]rune(x.f.Summary)) + 40
		if chars+block > budget { break }
		r = append(r, x.f); chars += block
	}
	return r
}

func ComputeMemoryEcho(facts []MemoryFact) MemoryEcho {
	if len(facts) == 0 { return MemoryEcho{} }
	now := time.Now()
	var sw, aff, sec, aro, dom float64
	for _, f := range facts {
		days := now.Sub(f.CreatedAt).Hours() / 24
		if days < 0 { days = 0 }
		decay := math.Exp(-0.003 * days)
		w := 1.0 * decay * f.SelfRelevance * (f.Weight / 3.0)
		sw += w
		aff += 1.0 * w * 0.5
		if f.Confidence > 0.5 { sec += 0.3 * w } else { sec -= 0.3 * w }
		aro += 1.0 * w * 0.6
		dom += 0.1 * w * 0.4
	}
	if sw <= 0 { return MemoryEcho{} }
	return MemoryEcho{
		Aff: clampF(aff/sw, -2, 2), Sec: clampF(sec/sw, -2, 2),
		Aro: clampF(aro/sw, -2, 2), Dom: clampF(dom/sw, -2, 2),
	}
}

// ─── KnowledgeGraph ───────────────────────────────────────────

type Triple struct {
	ID, Subject, Predicate, Object string
	Confidence                     float64
	SourceFactIDs                  []string
	CreatedAt                      time.Time
}

type KnowledgeGraph struct {
	triples   []Triple
	entityIdx map[string][]int
}

func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{entityIdx: make(map[string][]int)}
}

func (kg *KnowledgeGraph) Add(subj, pred, obj string, conf float64, src []string) Triple {
	t := Triple{ID: generateID("kg"), Subject: subj, Predicate: pred, Object: obj,
		Confidence: conf, SourceFactIDs: src, CreatedAt: time.Now()}
	idx := len(kg.triples)
	kg.triples = append(kg.triples, t)
	kg.addIdx(subj, idx); kg.addIdx(obj, idx)
	return t
}

func (kg *KnowledgeGraph) addIdx(entity string, idx int) {
	key := strings.ToLower(entity)
	kg.entityIdx[key] = append(kg.entityIdx[key], idx)
}

func (kg *KnowledgeGraph) Query(text string, max int) []Triple {
	ql := strings.ToLower(text)
	type sc struct{ t Triple; s float64 }
	var rs []sc
	for _, t := range kg.triples {
		tl := strings.ToLower(t.Subject + " " + t.Predicate + " " + t.Object)
		s := 0.0
		for e := range kg.entityIdx { if strings.Contains(ql, e) { s += 3.0 } }
		for _, w := range strings.Fields(ql) { if len([]rune(w)) >= 2 && strings.Contains(tl, w) { s += 1.0 } }
		if s >= 0.1 { rs = append(rs, sc{t, s}) }
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].s > rs[j].s })
	if max > 0 && len(rs) > max { rs = rs[:max] }
	var out []Triple
	for _, x := range rs { out = append(out, x.t) }
	return out
}

func (kg *KnowledgeGraph) BuildContextBlock(text string, budget int) string {
	hits := kg.Query(text, 8)
	if len(hits) == 0 { return "" }
	var lines []string
	lines = append(lines, "【知识图谱】")
	chars := 0
	for _, t := range hits {
		line := "- " + t.Subject + " —" + t.Predicate + "→ " + t.Object
		if chars+len([]rune(line)) > budget { break }
		lines = append(lines, line); chars += len([]rune(line))
	}
	if len(lines) <= 1 { return "" }
	return strings.Join(lines, "\n")
}

// ─── 辅助 ────────────────────────────────────────────────────

func jaccard(a, b string) float64 {
	sa := make(map[rune]bool); for _, r := range a { sa[r] = true }
	sb := make(map[rune]bool); for _, r := range b { sb[r] = true }
	is := 0; for r := range sa { if sb[r] { is++ } }
	un := len(sa) + len(sb) - is
	if un == 0 { return 0 }
	return float64(is) / float64(un)
}

func mergeStrs(a, b []string) []string {
	seen := make(map[string]bool)
	var r []string
	for _, s := range a { if !seen[s] { seen[s] = true; r = append(r, s) } }
	for _, s := range b { if !seen[s] { seen[s] = true; r = append(r, s) } }
	return r
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
