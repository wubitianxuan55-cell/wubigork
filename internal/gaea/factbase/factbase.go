// Package factbase stores a per-conversation 事实底座 (fact base): the
// structured facts an office task settles on before any deliverable is
// produced. It is the shared source of truth for multi-form outputs — a report,
// a slide deck and a spreadsheet generated from the same facts stay consistent
// with each other and with the source material they cite.
package factbase

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// Fact is one settled fact in the base.
type Fact struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Source    string    `json:"source,omitempty"`
	Category  string    `json:"category,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Base is an in-memory fact base. The zero value is ready to use.
type Base struct {
	Facts []Fact `json:"facts"`
}

// Add inserts or replaces the fact with the same key (case-insensitive).
// An empty value removes the fact, which lets the model correct itself.
func (b *Base) Add(key, value, source, category string, now time.Time) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	norm := strings.ToLower(key)
	for i := range b.Facts {
		if strings.ToLower(b.Facts[i].Key) == norm {
			if strings.TrimSpace(value) == "" {
				b.Facts = append(b.Facts[:i], b.Facts[i+1:]...)
			} else {
				b.Facts[i].Value = value
				b.Facts[i].Source = source
				b.Facts[i].Category = category
				b.Facts[i].UpdatedAt = now
			}
			return
		}
	}
	if strings.TrimSpace(value) == "" {
		return
	}
	b.Facts = append(b.Facts, Fact{
		Key: key, Value: value, Source: source, Category: category, UpdatedAt: now,
	})
}

// Clear empties the base.
func (b *Base) Clear() { b.Facts = nil }

// Sorted returns facts ordered by key, stable for rendering and diffing.
func (b *Base) Sorted() []Fact {
	out := append([]Fact(nil), b.Facts...)
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Key) < strings.ToLower(out[j].Key) })
	return out
}

// Markdown renders the base as a copy-ready Markdown table.
func (b *Base) Markdown() string {
	facts := b.Sorted()
	if len(facts) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## 事实底座\n\n")
	sb.WriteString("| 事实 | 内容 | 来源 | 分类 |\n")
	sb.WriteString("| --- | --- | --- | --- |\n")
	for _, f := range facts {
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
			mdCell(f.Key), mdCell(f.Value), mdCell(f.Source), mdCell(f.Category))
	}
	return sb.String()
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(s)
}

// Store persists a Base to one JSON file (atomic writes, per-session path).
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore returns a store bound to path; the file is created lazily on first
// write and need not exist yet.
func NewStore(path string) *Store { return &Store{path: path} }

// PathFor derives the fact-base file for a session file: a sibling named
// <session>-facts.json, so deleting/renaming the session keeps or drops the
// facts together with it.
func PathFor(sessionPath string) string {
	if sessionPath == "" {
		return ""
	}
	dir := filepath.Dir(sessionPath)
	base := filepath.Base(sessionPath)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return filepath.Join(dir, base+"-facts.json")
}

// Add loads, upserts one fact and saves. When path is empty it is a no-op that
// returns an error so callers can tell the model the session isn't ready.
func (s *Store) Add(key, value, source, category string) error {
	if s.path == "" {
		return fmt.Errorf("事实底座尚未绑定会话")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.loadLocked()
	if err != nil {
		return err
	}
	b.Add(key, value, source, category, time.Now())
	return s.saveLocked(b)
}

// Clear empties the persisted base.
func (s *Store) Clear() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &Base{}
	return s.saveLocked(b)
}

// Snapshot returns a copy of the persisted facts (empty when missing).
func (s *Store) Snapshot() (*Base, error) {
	if s.path == "" {
		return &Base{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

// Markdown returns the persisted base as Markdown ("" when empty/missing).
func (s *Store) Markdown() (string, error) {
	b, err := s.Snapshot()
	if err != nil {
		return "", err
	}
	return b.Markdown(), nil
}

func (s *Store) loadLocked() (*Base, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Base{}, nil
		}
		return nil, fmt.Errorf("读取事实底座 %s: %w", s.path, err)
	}
	var b Base
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("解析事实底座 %s: %w", s.path, err)
	}
	return &b, nil
}

func (s *Store) saveLocked(b *Base) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化事实底座: %w", err)
	}
	if err := fileutil.AtomicWrite(s.path, data, 0o644); err != nil {
		return fmt.Errorf("保存事实底座 %s: %w", s.path, err)
	}
	return nil
}
