// Package whisper — machine_map_store.go
// 100% 对齐 ackem desktop-agent/machine-map/repo.ts + service.ts
// Machine-Map 增强版持久化：JSON 存储 + 增量更新 + 过期检测 + 统计

package whisper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ─── 存储条目类型 ──────────────────────────────────────────────

// MapStoreEntry 持久化条目
type MapStoreEntry struct {
	Category     string    `json:"category"` // game / document / app
	DisplayName  string    `json:"displayName"`
	Path         string    `json:"path"`
	Source       string    `json:"source"`
	Confidence   float64   `json:"confidence"`
	FirstSeenAt  time.Time `json:"firstSeenAt"`
	LastVerified time.Time `json:"lastVerifiedAt"`
	Active       bool      `json:"active"`
	DedupeKey    string    `json:"dedupeKey"`
}

// MapStoreStats 存储统计
type MapStoreStats struct {
	TotalGames     int       `json:"totalGames"`
	TotalDocuments int       `json:"totalDocuments"`
	TotalApps      int       `json:"totalApps"`
	LastIndexed    time.Time `json:"lastIndexedAt"`
	IsStale        bool      `json:"isStale"`
}

// ─── MachineMapStore ───────────────────────────────────────────

// MachineMapStore 增强版 machine-map 持久化存储
type MachineMapStore struct {
	mu        sync.RWMutex
	dataRoot  string
	entries   []MapStoreEntry
	indexedAt time.Time
}

// NewMachineMapStore 创建存储实例
func NewMachineMapStore(dataRoot string) *MachineMapStore {
	s := &MachineMapStore{
		dataRoot: dataRoot,
	}
	s.Load()
	return s
}

// storagePath 持久化文件路径
func (s *MachineMapStore) storagePath() string {
	return filepath.Join(s.dataRoot, "desktop-agent", "machine-map-v2.json")
}

// ─── CRUD ──────────────────────────────────────────────────────

// UpsertEntries 批量 upsert 条目（增量更新）
// dedupeKey 相同则更新，否则新增
func (s *MachineMapStore) UpsertEntries(newEntries []MapStoreEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	keyIndex := make(map[string]int)
	for i, e := range s.entries {
		keyIndex[e.DedupeKey] = i
	}

	newKeys := make(map[string]bool)
	for _, ne := range newEntries {
		if ne.DedupeKey == "" {
			continue
		}
		newKeys[ne.DedupeKey] = true

		if idx, exists := keyIndex[ne.DedupeKey]; exists {
			// 更新已有条目
			old := &s.entries[idx]
			old.DisplayName = ne.DisplayName
			old.Path = ne.Path
			old.LastVerified = now
			old.Active = true
			// 保留更高置信度的来源
			if sourceRank(ne.Source) > sourceRank(old.Source) {
				old.Source = ne.Source
				old.Confidence = ne.Confidence
			}
		} else {
			// 新增
			ne.FirstSeenAt = now
			ne.LastVerified = now
			ne.Active = true
			s.entries = append(s.entries, ne)
		}
	}

	// 标记未再出现的旧条目为 inactive
	for i := range s.entries {
		if !newKeys[s.entries[i].DedupeKey] {
			s.entries[i].Active = false
		}
	}

	s.indexedAt = now
}

// DeactivateByCategory 按类别标记 inactive
func (s *MachineMapStore) DeactivateByCategory(category string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].Category == category {
			s.entries[i].Active = false
		}
	}
}

// ListActive 列出活跃条目
func (s *MachineMapStore) ListActive(category string) []MapStoreEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if category == "" {
		var result []MapStoreEntry
		for _, e := range s.entries {
			if e.Active {
				result = append(result, e)
			}
		}
		return result
	}

	var result []MapStoreEntry
	for _, e := range s.entries {
		if e.Active && e.Category == category {
			result = append(result, e)
		}
	}
	return result
}

// Count 统计数量
func (s *MachineMapStore) Count(category string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, e := range s.entries {
		if !e.Active {
			continue
		}
		if category == "" || e.Category == category {
			count++
		}
	}
	return count
}

// ─── 统计 ──────────────────────────────────────────────────────

// GetStats 获取存储统计
func (s *MachineMapStore) GetStats() MapStoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := MapStoreStats{
		LastIndexed: s.indexedAt,
		IsStale:     time.Since(s.indexedAt) > 24*time.Hour,
	}
	for _, e := range s.entries {
		if !e.Active {
			continue
		}
		switch e.Category {
		case "game":
			stats.TotalGames++
		case "document":
			stats.TotalDocuments++
		case "app":
			stats.TotalApps++
		}
	}
	return stats
}

// IsStale 是否过期（超过24小时未更新）
func (s *MachineMapStore) IsStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.indexedAt) > 24*time.Hour
}

// ─── 持久化 ────────────────────────────────────────────────────

// Persist 持久化到 JSON 文件
func (s *MachineMapStore) Persist() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(struct {
		Entries   []MapStoreEntry `json:"entries"`
		IndexedAt time.Time       `json:"indexedAt"`
		Version   string          `json:"version"`
	}{
		Entries:   s.entries,
		IndexedAt: s.indexedAt,
		Version:   "v2",
	}, "", "  ")
	if err != nil {
		return err
	}

	path := s.storagePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load 从 JSON 文件加载
func (s *MachineMapStore) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.storagePath()
	data, err := os.ReadFile(path)
	if err != nil {
		s.entries = nil
		s.indexedAt = time.Time{}
		return
	}

	var stored struct {
		Entries   []MapStoreEntry `json:"entries"`
		IndexedAt time.Time       `json:"indexedAt"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		s.entries = nil
		s.indexedAt = time.Time{}
		return
	}

	s.entries = stored.Entries
	s.indexedAt = stored.IndexedAt
}

// ─── 全局单例 ──────────────────────────────────────────────────

var (
	globalStore     *MachineMapStore
	globalStoreMu   sync.Mutex
	globalStoreRoot string
)

// GetMachineMapStore 获取全局 machine-map 存储单例
func GetMachineMapStore(dataRoot string) *MachineMapStore {
	globalStoreMu.Lock()
	defer globalStoreMu.Unlock()

	if globalStore != nil && globalStoreRoot == dataRoot {
		return globalStore
	}

	globalStore = NewMachineMapStore(dataRoot)
	globalStoreRoot = dataRoot
	return globalStore
}
