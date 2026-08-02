package knowledge

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
)

// Service is the shared entry point for the knowledge base. All consumers —
// the agent's knowledge_add/knowledge_search tools, the Knowledge 板块 UI
// bindings, and future feature modules (轻语/方案 etc.) — go through this
// single instance instead of each opening the store independently.
//
// 知识库定位：显式、可编辑、可分类检索的工程知识条目（规范/案例/经验），
// 与记忆系统（隐式跨会话事实，internal/gaea/memory）明确区分。
type Service struct {
	mu    sync.Mutex
	store *Store
	err   error
}

var (
	globalMu sync.Mutex
	global   *Service
)

// DefaultDir returns the knowledge base directory (~/.gaea/knowledge).
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gaea/knowledge"
	}
	return filepath.Join(home, ".gaea", "knowledge")
}

// Global returns the process-wide knowledge Service. The same service is
// shared by every module, so the store is opened exactly once per process.
func Global() *Service {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		global = &Service{}
	}
	return global
}

// Store returns the underlying store, opening the default directory lazily on
// first access. A failed open is remembered so subsequent calls return the
// same error instead of retrying.
func (s *Service) Store() (*Store, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	if s.store == nil {
		// 主脑 Hephaestus.db 为默认后端；旧 Markdown 知识库首次打开自动迁移。
		gdb := db.GetDatabase(config.MemoryUserDir())
		if gdb == nil {
			st, err := Open(DefaultDir())
			if err != nil {
				s.err = err
				return nil, err
			}
			s.store = st
			return s.store, nil
		}
		if _, err := MigrateLegacyKnowledge(gdb, DefaultDir()); err != nil {
			log.Printf("[hephaestus] 知识库迁移失败: %v", err)
		}
		st, err := OpenSQLite(gdb)
		if err != nil {
			s.err = err
			return nil, err
		}
		s.store = st
	}
	return s.store, nil
}

// SetStoreForTest overrides the process-wide store (test isolation). Tests call
// this with a temp-dir store so they never touch the real Hephaestus.db or
// ~/.gaea/knowledge.
func SetStoreForTest(st *Store) {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		global = &Service{}
	}
	global.mu.Lock()
	global.store = st
	global.err = nil
	global.mu.Unlock()
}

// ResetForTest clears the process-wide service so tests can open a fresh dir.
func ResetForTest() {
	globalMu.Lock()
	global = nil
	globalMu.Unlock()
}
