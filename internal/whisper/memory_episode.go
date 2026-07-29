// Package whisper — memory_episode.go
// 100% 对齐 ackem memory/episodicStore.ts
// 情节记忆存储：CRUD + 关键词检索 + 情感加权

package whisper

import (
	"math"
	"strings"
	"time"
)

// ─── EpisodicStore ────────────────────────────────────────────

// EpisodicStore 情节记忆存储
type EpisodicStore struct {
	episodes []Episode
}

// NewEpisodicStore 创建空情节库
func NewEpisodicStore() *EpisodicStore {
	return &EpisodicStore{}
}

// Add 添加情节片段
func (es *EpisodicStore) Add(ep Episode) {
	if ep.ID == "" {
		ep.ID = genHexID()
	}
	if ep.CreatedAt.IsZero() {
		ep.CreatedAt = time.Now()
	}
	es.episodes = append(es.episodes, ep)
}

// Search 关键词检索（对齐 ackem episodic search）
func (es *EpisodicStore) Search(query string, max int) []Episode {
	now := time.Now()
	type scored struct {
		ep Episode
		s  float64
	}
	var ranked []scored
	ql := strings.ToLower(strings.TrimSpace(query))

	for _, ep := range es.episodes {
		s := 0.0
		// 关键词匹配
		for _, kw := range ep.Keywords {
			if strings.Contains(ql, strings.ToLower(kw)) {
				s += 1.0
			}
		}
		if strings.Contains(ql, strings.ToLower(ep.Summary)) {
			s += 0.5
		}
		if s > 0 {
			// 情感强度加权 + 时间衰减（半衰期 7 天）
			ageHours := now.Sub(ep.CreatedAt).Hours()
			s *= ep.EmotionalIntensity * math.Pow(2, -ageHours/168)
			ranked = append(ranked, scored{ep, s})
		}
	}

	// 降序排列
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].s > ranked[i].s {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	if max > 0 && len(ranked) > max {
		ranked = ranked[:max]
	}

	result := make([]Episode, len(ranked))
	for i, r := range ranked {
		result[i] = r.ep
	}
	return result
}

// ListAll 返回所有情节
func (es *EpisodicStore) ListAll() []Episode {
	result := make([]Episode, len(es.episodes))
	copy(result, es.episodes)
	return result
}

// Get 按 ID 查找
func (es *EpisodicStore) Get(id string) *Episode {
	for i := range es.episodes {
		if es.episodes[i].ID == id {
			return &es.episodes[i]
		}
	}
	return nil
}

// Count 情节数量
func (es *EpisodicStore) Count() int {
	return len(es.episodes)
}

// Latest 返回最新的情节（P2 新增）
func (es *EpisodicStore) Latest() *Episode {
	if len(es.episodes) == 0 {
		return nil
	}
	return &es.episodes[len(es.episodes)-1]
}
