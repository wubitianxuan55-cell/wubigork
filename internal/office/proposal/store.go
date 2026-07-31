// Package proposal — 方案存储层（JSON 文件）
package proposal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/google/uuid"
)

// Store 方案存储
type Store struct {
	mu      sync.RWMutex
	dir     string
	idxPath string
}
// NewStore 创建存储实例
func NewStore(dataRoot string) *Store {
	dir := filepath.Join(dataRoot, "office", "proposals")
	os.MkdirAll(dir, 0755)
	return &Store{
		dir:     dir,
		idxPath: filepath.Join(dir, "index.json"),
	}
}

// indexEntry 索引条目（轻量）
type indexEntry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	Template  string `json:"template"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

// List 列出所有方案
func (s *Store) List() ([]Proposal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := s.readIndex()
	if err != nil {
		return nil, err
	}

	var result []Proposal
	for _, e := range entries {
		p, err := s.loadUnsafe(e.ID)
		if err != nil {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

// Get 获取单个方案
func (s *Store) Get(id string) (*Proposal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnsafe(id)
}

// Create 创建方案
func (s *Store) Create(title, template, requirements, category string, sections []ProposalSection) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	ts := now()
	for i := range sections {
		sections[i].ID = uuid.New().String()
		sections[i].ProposalID = id
		sections[i].Status = "pending"
	}
	p := &Proposal{
		ID:           id,
		Title:        title,
		Category:     category,
		Status:       "draft",
		Sections:     sections,
		CreatedAt:    ts,
		UpdatedAt:    ts,
	}
	if err := s.saveUnsafe(p); err != nil {
		return nil, err
	}
	if err := s.addIndex(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Update 更新方案
func (s *Store) Update(p *Proposal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.UpdatedAt = now()
	return s.saveUnsafe(p)
}

// Delete 删除方案
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, id+".json")
	os.Remove(path)
	return s.removeIndex(id)
}

// ─── 内部方法 ────────────────────────────────────────────

func (s *Store) proposalPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *Store) saveUnsafe(p *Proposal) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.proposalPath(p.ID), data, 0644)
}

func (s *Store) loadUnsafe(id string) (*Proposal, error) {
	data, err := os.ReadFile(s.proposalPath(id))
	if err != nil {
		return nil, err
	}
	var p Proposal
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) readIndex() ([]indexEntry, error) {
	data, err := os.ReadFile(s.idxPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []indexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) writeIndex(entries []indexEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.idxPath, data, 0644)
}

func (s *Store) addIndex(p *Proposal) error {
	entries, _ := s.readIndex()
	entries = append(entries, indexEntry{
		ID: p.ID, Title: p.Title, Category: p.Category, Template: p.Template,
		Status: p.Status, UpdatedAt: p.UpdatedAt,
	})
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt > entries[j].UpdatedAt
	})
	return s.writeIndex(entries)
}

func (s *Store) removeIndex(id string) error {
	entries, _ := s.readIndex()
	var filtered []indexEntry
	for _, e := range entries {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	return s.writeIndex(filtered)
}
