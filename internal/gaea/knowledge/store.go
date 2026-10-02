package knowledge

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// backend 是知识库存储引擎：fileBackend（~/.gaea/knowledge Markdown）或
// sqliteBackend（Hephaestus.db knowledge 表）。Store 持有 backend，调用方不变。
type backend interface {
	Save(e Entry) error
	Get(name string) (*Entry, error)
	Delete(name string) error
	// List/ReadAll 必须把 DB/文件读取失败如实返回 error（审计 P0#6 GA3-01）：
	// 「库里真的没有条目」与「库读不出来」不可用同一个空切片表示——吞错会让
	// 查重/导入预览把库故障判成空库，每行都判「新增」后静默重复入库。
	List() ([]EntrySummary, error)
	Index() string
	ReadAll() ([]Entry, error)
}

// Store manages the knowledge base entries. It stays a struct (not an
// interface) so existing call sites keep compiling; the engine is internal.
type Store struct {
	Dir     string // file backend: knowledge base directory
	backend backend
	mu      sync.RWMutex
	tfidf   tfidfCache // Search 的 TF-IDF 索引缓存（写路径失效，见 search_cache.go）
}

// Open initializes a file-backed Store at the given directory. The directory
// and INDEX.md are created if they don't exist.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	idxPath := filepath.Join(dir, "INDEX.md")
	if _, err := os.Stat(idxPath); os.IsNotExist(err) {
		initial := "# 知识库索引\n\n知识库目录。使用 knowledge_add 添加条目。\n"
		if err := os.WriteFile(idxPath, []byte(initial), 0o644); err != nil {
			return nil, fmt.Errorf("create INDEX.md: %w", err)
		}
	}
	return &Store{Dir: dir, backend: &fileBackend{dir: dir}}, nil
}

// OpenSQLite initializes a SQLite-backed Store on the Hephaestus.db knowledge
// table. A nil db yields an error.
func OpenSQLite(gdb *sql.DB) (*Store, error) {
	if gdb == nil {
		return nil, fmt.Errorf("knowledge db unavailable")
	}
	return &Store{backend: &sqliteBackend{db: gdb}}, nil
}

// Save writes an entry to storage and refreshes the index.
func (s *Store) Save(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.backend.Save(e); err != nil {
		return err
	}
	s.tfidf.invalidate() // 写路径失效：下次 Search 重建 TF-IDF 索引
	return nil
}

// Get reads and parses an entry by name.
func (s *Store) Get(name string) (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backend.Get(name)
}

// Delete removes an entry and updates the index.
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.backend.Delete(name); err != nil {
		return err
	}
	s.tfidf.invalidate() // 写路径失效：下次 Search 重建 TF-IDF 索引
	return nil
}

// List returns all entries with their metadata (without Body). 读失败返回
// error，调用方据此区分「库空」与「库不可读」（审计 P0#6 GA3-01）。
func (s *Store) List() ([]EntrySummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backend.List()
}

// Index returns the index text (INDEX.md content for file backend; rendered
// table for SQLite backend).
func (s *Store) Index() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backend.Index()
}

// ReadAll returns all full entries (used by Search and migrations). 读失败
// 返回 error，不再吞成空切片（审计 P0#6 GA3-01）。
func (s *Store) ReadAll() ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backend.ReadAll()
}

// ─── 文件后端（原实现）───────────────────────────────────────────

type fileBackend struct {
	dir string
}

func (b *fileBackend) Save(e Entry) error {
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	content := RenderFrontmatter(e) + e.Body
	path := filepath.Join(b.dir, FileName(e))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return b.rebuildIndex()
}

func (b *fileBackend) Get(name string) (*Entry, error) {
	path := filepath.Join(b.dir, safeFileName(name)+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("entry %q not found", name)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseFrontmatter(string(data))
}

func (b *fileBackend) Delete(name string) error {
	path := filepath.Join(b.dir, safeFileName(name)+".md")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return b.rebuildIndex()
}

func (b *fileBackend) List() ([]EntrySummary, error) {
	entries, err := b.ReadAll()
	if err != nil {
		return nil, err
	}
	summaries := make([]EntrySummary, 0, len(entries))
	for _, e := range entries {
		summaries = append(summaries, e.ToSummary())
	}
	SortEntrySummaries(summaries)
	return summaries, nil
}

func (b *fileBackend) Index() string {
	data, err := os.ReadFile(filepath.Join(b.dir, "INDEX.md"))
	if err != nil {
		return "# 知识库索引\n\n（索引不可用）\n"
	}
	return string(data)
}

func (b *fileBackend) ReadAll() ([]Entry, error) {
	paths, err := filepath.Glob(filepath.Join(b.dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", b.dir, err)
	}
	var result []Entry
	for _, path := range paths {
		if filepath.Base(path) == "INDEX.md" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			// 读失败（权限/损坏/同名目录）不再静默当空条目跳过：审计 P0#6
			// GA3-01，「无权限/损坏文件被静默当空条目」会让查重判「全部新增」。
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		e, err := ParseFrontmatter(string(data))
		if err != nil || e.Name == "" {
			// 无 frontmatter/无名的 Markdown 不是条目：按旧口径跳过
			// （属数据形状，不是读失败）。
			continue
		}
		result = append(result, *e)
	}
	return result, nil
}

func (b *fileBackend) rebuildIndex() error {
	entries, err := b.ReadAll()
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	var buf strings.Builder
	buf.WriteString("# 知识库索引\n\n")
	if len(entries) == 0 {
		buf.WriteString("（暂无条目。使用 knowledge_add 工具添加。）\n")
	} else {
		buf.WriteString("| 名称 | 标题 | 分类 | 状态 | 更新日期 |\n")
		buf.WriteString("|------|------|------|------|----------|\n")
		for _, e := range entries {
			dateStr := ""
			if !e.UpdatedAt.IsZero() {
				dateStr = e.UpdatedAt.Format("2006-01-02")
			}
			fmt.Fprintf(&buf, "| %s | %s | %s | %s | %s |\n",
				e.Name, e.Title, e.Category, e.Status, dateStr)
		}
	}
	return os.WriteFile(filepath.Join(b.dir, "INDEX.md"), []byte(buf.String()), 0o644)
}
