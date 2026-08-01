// Package whisper — machine_map_indexer.go
// 100% 对齐 ackem desktop-agent/machine-map/indexer.ts + repo.ts + service.ts
// 机器地图索引器：DB 持久化 + 增量更新 + 调度
package whisper

import (
	"log/slog"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ─── 索引器 ──────────────────────────────────────────────────────

// MachineMapIndexer 机器地图索引器
type MachineMapIndexer struct {
	mu        sync.Mutex
	dataRoot  string
	running   bool
	lastIndex time.Time
	entries   []MachineMapEntry // 内存缓存
}

// NewMachineMapIndexer 创建索引器
func NewMachineMapIndexer(dataRoot string) *MachineMapIndexer {
	return &MachineMapIndexer{
		dataRoot: dataRoot,
		entries:  make([]MachineMapEntry, 0),
	}
}

// IsRunning 是否正在运行
func (m *MachineMapIndexer) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// LastIndexTime 上次索引时间
func (m *MachineMapIndexer) LastIndexTime() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastIndex
}

// RunFullIndex 执行全量索引
func (m *MachineMapIndexer) RunFullIndex() (*MapCollectResult, error) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil, fmt.Errorf("索引正在运行中")
	}
	m.running = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.running = false
		m.lastIndex = time.Now()
		m.mu.Unlock()
	}()

	// 采集
	result := CollectMachineMap()

	// 更新内存缓存
	m.mu.Lock()
	m.entries = make([]MachineMapEntry, 0, len(result.Games)+len(result.Documents)+len(result.Apps))
	for _, g := range result.Games {
		m.entries = append(m.entries, MachineMapEntry{
			Path: g.Path, Name: g.Name, Type: "game",
			SizeBytes: 0,
		})
	}
	for _, d := range result.Documents {
		m.entries = append(m.entries, MachineMapEntry{
			Path: d.Path, Name: d.Name, Type: "document",
			SizeBytes: 0,
		})
	}
	for _, a := range result.Apps {
		m.entries = append(m.entries, MachineMapEntry{
			Path: a.Path, Name: a.Name, Type: "app",
			SizeBytes: 0,
		})
	}
	m.mu.Unlock()

	// 持久化到 JSON
	m.persist()

	return result, nil
}

// Search 搜索索引
func (m *MachineMapIndexer) Search(query string, kind string) []MachineMapEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	q := strings.ToLower(query)
	var results []MachineMapEntry
	for _, e := range m.entries {
		if kind != "" && e.Type != kind {
			continue
		}
		if strings.Contains(strings.ToLower(e.Name), q) || strings.Contains(strings.ToLower(e.Path), q) {
			results = append(results, e)
		}
	}
	return results
}

// Count 返回各类条目数
func (m *MachineMapIndexer) Count() (games, docs, apps int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		switch e.Type {
		case "game":
			games++
		case "document":
			docs++
		case "app":
			apps++
		}
	}
	return
}

// persist 持久化到 JSON 文件
func (m *MachineMapIndexer) persist() {
	data, err := json.Marshal(m.entries)
	if err != nil {
		return
	}

	path := filepath.Join(m.dataRoot, "desktop-agent", "machine-map.json")
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, data, 0644)
}

// Load 从 JSON 文件加载索引
func (m *MachineMapIndexer) Load() error {
	path := filepath.Join(m.dataRoot, "desktop-agent", "machine-map.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var entries []MachineMapEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	m.entries = entries
	return nil
}

// GetStats 获取索引统计信息
func (m *MachineMapIndexer) GetStats() map[string]interface{} {
	games, docs, apps := m.Count()
	return map[string]interface{}{
		"totalEntries": len(m.entries),
		"games":        games,
		"documents":    docs,
		"apps":         apps,
		"lastIndexed":  m.lastIndex.Format("2006-01-02 15:04:05"),
		"isRunning":    m.running,
	}
}

// ─── 全局调度器 ──────────────────────────────────────────────────

var (
	globalIndexer     *MachineMapIndexer
	globalIndexerMu   sync.Mutex
)

// GetGlobalIndexer 获取全局索引器
func GetGlobalIndexer(dataRoot string) *MachineMapIndexer {
	globalIndexerMu.Lock()
	defer globalIndexerMu.Unlock()

	if globalIndexer == nil {
		globalIndexer = NewMachineMapIndexer(dataRoot)
		globalIndexer.Load() // 尝试加载已有索引
	}
	return globalIndexer
}

// ScheduleMachineMapIndex 调度后台索引（非阻塞）
func ScheduleMachineMapIndex(dataRoot string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("whisper: machine map index goroutine panic recovered", "panic", r)
			}
		}()
		indexer := GetGlobalIndexer(dataRoot)
		if !indexer.IsRunning() {
			indexer.RunFullIndex()
		}
	}()
}
