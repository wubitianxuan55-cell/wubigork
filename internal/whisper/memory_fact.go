// Package whisper — memory_fact.go
// 100% 对齐 ackem memory/factStore.ts
// 增强版 FactStore：内存去重、情绪调制、核心记忆筛选、隐私过滤

package whisper

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── Fact（增强版 MemoryFact wrapper）────────────────────────

// Fact 内部事实表示（包装 MemoryFact + 运行时字段）
type Fact struct {
	MemoryFact
	Active   bool   `json:"active"`
	RawTier  string `json:"tier"` // core/memory/scratch
}

func (f *Fact) IsActive() bool  { return f.Active && f.Status == "active" }
func (f *Fact) IsCore() bool    { return f.IsActive() && f.RawTier == "core" }

// ─── FactStore ────────────────────────────────────────────────

// FactStore 增强版事实库
type FactStore struct {
	mu       sync.RWMutex
	facts    []*Fact
	byID     map[string]*Fact
	autoRetireCount int
}

// NewFactStore 创建空事实库
func NewFactStore() *FactStore {
	return &FactStore{
		byID: make(map[string]*Fact),
	}
}

// ─── 核心操作 ──────────────────────────────────────────────────

// Add 添加事实，Jaccard 去重（对齐 ackem addFact）
func (fs *FactStore) Add(raw MemoryFact) *Fact {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 去重：同 domain + subcategory + subject + Jaccard > 阈值
	for _, f := range fs.facts {
		if !f.IsActive() {
			continue
		}
		if f.Domain == raw.Domain && f.Subcategory == raw.Subcategory &&
			f.Subject == raw.Subject && jaccardRaw(f.Summary, raw.Summary) > FactDedupThreshold {
			// 合并：weight boost
			if raw.Weight > f.Weight {
				f.Weight = raw.Weight + 0.5
			} else {
				f.Weight += 0.5
			}
			if raw.Confidence > f.Confidence {
				f.Confidence = raw.Confidence
			}
			if len(raw.Summary) > len(f.Summary) {
				f.Summary = raw.Summary
			}
			f.Triggers = mergeStrSlices(f.Triggers, raw.Triggers)
			f.UpdatedAt = time.Now()
			if raw.ID != "" {
				f.SourceTurnIndex = raw.SourceTurnIndex
				f.SourceSessionID = raw.SourceSessionID
			}
			return f
		}
	}

	// 新事实
	f := &Fact{
		MemoryFact: raw,
		Active:     true,
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now()
	}
	f.UpdatedAt = time.Now()
	if f.Status == "" {
		f.Status = "active"
	}
	if f.ID == "" {
		f.ID = genHexID()
	}
	f.RawTier = computeTier(f.Weight)

	fs.facts = append(fs.facts, f)
	fs.byID[f.ID] = f
	return f
}

// Get 按 ID 查找
func (fs *FactStore) Get(id string) *Fact {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.byID[id]
}

// ListActive 返回所有活跃事实
func (fs *FactStore) ListActive() []*Fact {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var r []*Fact
	for _, f := range fs.facts {
		if f.IsActive() {
			r = append(r, f)
		}
	}
	return r
}

// ListBySessionTurn 返回指定会话指定轮次的事实（用于冷启动关联）
func (fs *FactStore) ListBySessionTurn(sessionID string, turnIndex int) []*Fact {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var r []*Fact
	for _, f := range fs.facts {
		if f.IsActive() && f.SourceSessionID == sessionID && f.SourceTurnIndex == turnIndex {
			r = append(r, f)
		}
	}
	return r
}

// Count 活跃事实数
func (fs *FactStore) Count() int {
	return len(fs.ListActive())
}

// ─── 检索 ──────────────────────────────────────────────────────

// SearchByTriggers 触发词匹配检索
func (fs *FactStore) SearchByTriggers(msg string) []*Fact {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	m := strings.ToLower(msg)
	var r []*Fact
	for _, f := range fs.facts {
		if !f.IsActive() {
			continue
		}
		for _, t := range f.Triggers {
			if strings.Contains(m, strings.ToLower(t)) {
				r = append(r, f)
				break
			}
		}
	}
	return r
}

// ScoreRelevance 计算事实相关性得分（对齐 ackem scoreRelevance）
// 优先使用 EmotionalContext.valence 做情绪一致性比较；回退到 SelfRelevance 近似
func ScoreRelevance(f *Fact, now time.Time, valence, aff float64) float64 {
	days := now.Sub(f.CreatedAt).Hours() / 24
	if days < 0 {
		days = 0
	}
	decay := math.Exp(-0.003 * days)
	score := f.Weight * decay * f.SelfRelevance

	// 情绪一致性调制 — 优先使用 EmotionalContext
	var factValence float64
	if f.EmotionalContext != nil {
		factValence = f.EmotionalContext.Valence
		// 情绪强度参与基础分
		score = f.Weight * decay * f.SelfRelevance * (1 + f.EmotionalContext.Intensity*0.5)
	} else {
		factValence = clampF(f.SelfRelevance/2.0, 0, 1)
	}
	if math.Abs(factValence-math.Abs(valence)) < 0.3 {
		boost := 1.5
		if math.Abs(aff) >= 50 {
			boost = 1.2
		}
		score *= boost
	}

	// 近期 boost
	hours := now.Sub(f.UpdatedAt).Hours()
	if hours < RecencyBoostWindowHours {
		score *= RecencyBoostFactor
	}
	return score
}

// SelectForInjection 按 budget 选出最佳事实

// SelectForInjection 按 budget 选出最佳事实
func (fs *FactStore) SelectForInjection(budget int, minConf, valence, aff float64) []*Fact {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	now := time.Now()
	type pair struct {
		f *Fact
		s float64
	}
	var ranked []pair
	for _, f := range fs.facts {
		if !f.IsActive() || f.Confidence < minConf {
			continue
		}
		ranked = append(ranked, pair{f, ScoreRelevance(f, now, valence, aff)})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })

	var r []*Fact
	chars := 0
	for _, p := range ranked {
		block := len([]rune(p.f.Summary)) + 40
		if chars+block > budget {
			break
		}
		r = append(r, p.f)
		chars += block
	}
	return r
}

// ─── 核心记忆 ──────────────────────────────────────────────────

// SelectCoreFacts 返回核心记忆（weight >= 阈值），最多 max 条
func (fs *FactStore) SelectCoreFacts(max int) []*Fact {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var core []*Fact
	for _, f := range fs.facts {
		if f.IsActive() && f.Weight >= CoreMemoryWeightThreshold {
			core = append(core, f)
		}
	}
	sort.Slice(core, func(i, j int) bool { return core[i].Weight > core[j].Weight })
	if max > 0 && len(core) > max {
		core = core[:max]
	}
	return core
}

// ─── 情绪一致性 boost ────────────────────────────────────────

// MoodCongruentBoost 返回情绪一致性倍率
// MoodCongruentBoost 返回情绪一致性倍率
// P0修复: 原实现错误地用 Confidence 替代 valence；暂时使用 selfRelevance 近似
func ComputeMoodBoost(selfRelevance, valence, aff float64) float64 {
	selfRelNorm := clampF(selfRelevance/2.0, 0, 1)
	diff := math.Abs(selfRelNorm - math.Abs(valence))
	boost := 1.0
	if diff < MoodCongruentValenceDiff {
		boost = MoodCongruentBoost
		if math.Abs(aff) >= MoodCongruentExtremeThreshold {
			boost = MoodCongruentExtremeBoost
		}
	}
	return boost
}

// ─── 自动退役 ──────────────────────────────────────────────────

// AutoRetire 自动标记过期事实为 retired
func (fs *FactStore) AutoRetire() int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	now := time.Now()
	count := 0
	for _, f := range fs.facts {
		if !f.IsActive() {
			continue
		}
		days := now.Sub(f.CreatedAt).Hours() / 24
		if f.Confidence < 0.3 || (f.Weight < 0.5 && days > 30) {
			f.Status = "retired"
			f.Active = false
			f.RawTier = "scratch"
			count++
		}
	}
	fs.autoRetireCount += count
	return count
}

// ─── 隐私过滤 ──────────────────────────────────────────────────

// PrivacyFilter 按模式过滤事实
func (fs *FactStore) PrivacyFilter(adultMode bool) []*Fact {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var r []*Fact
	for _, f := range fs.facts {
		if !f.IsActive() {
			continue
		}
		if !adultMode && (f.PrivacyLevel == "intimate" || f.PrivacyLevel == "explicit") {
			continue
		}
		r = append(r, f)
	}
	return r
}

// ─── Embedding 去重占位 ──────────────────────────────────────

// DedupByEmbedding 占位方法，Phase 3 实现
func (fs *FactStore) DedupByEmbedding(embedding []float64) bool {
	return false
}

// ─── 更新与退役 ──────────────────────────────────────────────

// RetireFact 退役指定事实
func (fs *FactStore) RetireFact(id string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if f, ok := fs.byID[id]; ok {
		f.Status = "retired"
		f.Active = false
		f.RawTier = "scratch"
	}
}

// UpdateFact 更新事实字段（key→value 映射）
func (fs *FactStore) UpdateFact(id string, updates map[string]interface{}) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	f, ok := fs.byID[id]
	if !ok {
		return
	}
	if v, ok := updates["summary"].(string); ok {
		f.Summary = v
	}
	if v, ok := updates["weight"].(float64); ok {
		f.Weight = v
	}
	if v, ok := updates["confidence"].(float64); ok {
		f.Confidence = v
	}
	if v, ok := updates["subject"].(string); ok {
		f.Subject = v
	}
	if v, ok := updates["domain"].(string); ok {
		f.Domain = v
	}
	f.UpdatedAt = time.Now()
}

// ─── 辅助 ────────────────────────────────────────────────────

// ─── 辅助 ────────────────────────────────────────────────────

func computeTier(weight float64) string {
	if weight >= CoreMemoryWeightThreshold {
		return "core"
	}
	return "memory"
}

func genHexID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// jaccardRaw 计算两个字符串的字符级 Jaccard 相似度
func jaccardRaw(a, b string) float64 {
	sa := make(map[rune]bool)
	for _, r := range a {
		sa[r] = true
	}
	sb := make(map[rune]bool)
	for _, r := range b {
		sb[r] = true
	}
	is := 0
	for r := range sa {
		if sb[r] {
			is++
		}
	}
	un := len(sa) + len(sb) - is
	if un == 0 {
		return 0
	}
	return float64(is) / float64(un)
}

func mergeStrSlices(a, b []string) []string {
	seen := make(map[string]bool)
	var r []string
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			r = append(r, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			r = append(r, s)
		}
	}
	return r
}
