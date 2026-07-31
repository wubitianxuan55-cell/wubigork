package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store manages a directory of knowledge entries (.md files with frontmatter).
type Store struct {
	Dir string
	mu  sync.RWMutex
}

// Open initializes a Store at the given directory. The directory and INDEX.md
// are created if they don't exist.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}

	s := &Store{Dir: dir}

	// Create INDEX.md if it doesn't exist.
	idxPath := filepath.Join(dir, "INDEX.md")
	if _, err := os.Stat(idxPath); os.IsNotExist(err) {
		initial := "# 知识库索引\n\n知识库目录。使用 knowledge_add 添加条目。\n"
		if err := os.WriteFile(idxPath, []byte(initial), 0644); err != nil {
			return nil, fmt.Errorf("create INDEX.md: %w", err)
		}
	}

	return s, nil
}

// Save writes an entry to disk. It renders the frontmatter + body and writes
// to {name}.md, then updates INDEX.md.
func (s *Store) Save(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now

	content := RenderFrontmatter(e) + e.Body
	path := filepath.Join(s.Dir, FileName(e))

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return s.rebuildIndex()
}

// Get reads and parses an entry by name.
func (s *Store) Get(name string) (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.Dir, safeFileName(name)+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("entry %q not found", name)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return ParseFrontmatter(string(data))
}

// Delete removes an entry file and updates INDEX.md.
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.Dir, safeFileName(name)+".md")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return s.rebuildIndex()
}

// List returns all entries with their metadata (without Body).
func (s *Store) List() []EntrySummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.readAll()
	summaries := make([]EntrySummary, 0, len(entries))
	for _, e := range entries {
		summaries = append(summaries, e.ToSummary())
	}
	SortEntrySummaries(summaries)
	return summaries
}

// Index returns the content of INDEX.md.
func (s *Store) Index() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.Dir, "INDEX.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "# 知识库索引\n\n（索引不可用）\n"
	}
	return string(data)
}

// readAll reads all .md entry files (excluding INDEX.md) and returns them.
func (s *Store) readAll() []Entry {
	entries, _ := filepath.Glob(filepath.Join(s.Dir, "*.md"))
	var result []Entry
	for _, path := range entries {
		if filepath.Base(path) == "INDEX.md" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		e, err := ParseFrontmatter(string(data))
		if err != nil || e.Name == "" {
			continue
		}
		result = append(result, *e)
	}
	return result
}

// rebuildIndex regenerates INDEX.md from all entries.
func (s *Store) rebuildIndex() error {
	entries := s.readAll()
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	var b strings.Builder
	b.WriteString("# 知识库索引\n\n")
	if len(entries) == 0 {
		b.WriteString("（暂无条目。使用 knowledge_add 工具添加。）\n")
	} else {
		b.WriteString("| 名称 | 标题 | 分类 | 状态 | 更新日期 |\n")
		b.WriteString("|------|------|------|------|----------|\n")
		for _, e := range entries {
			dateStr := ""
			if !e.UpdatedAt.IsZero() {
				dateStr = e.UpdatedAt.Format("2006-01-02")
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				e.Name, e.Title, e.Category, e.Status, dateStr)
		}
	}

	idxPath := filepath.Join(s.Dir, "INDEX.md")
	return os.WriteFile(idxPath, []byte(b.String()), 0644)
}
