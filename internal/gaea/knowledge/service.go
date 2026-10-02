package knowledge

import (
	"database/sql"
	"log"
	"log/slog"
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
	// migrationErr / migrationRan 记录旧 Markdown 知识库迁移状态。迁移失败
	// 不阻断知识库（库仍可读写），但必须能被启动路径与面板看见：失败时库处于
	// 「部分迁移」状态，与「迁移完成」在用起来之前不可区分（审计 GA3-13）。
	migrationErr error
	migrationRan bool
}

// MigrationState 旧知识库迁移状态快照。Ran=本进程尝试过迁移；Failed=迁移
// 失败（此时知识库可用但可能是部分迁移）；Reason=失败原因。零值表示本进程
// 尚未触发迁移。
type MigrationState struct {
	Ran    bool
	Failed bool
	Reason string
}

// MigrationState 返回旧 Markdown 知识库迁移状态，供启动路径与知识库面板
// 提示用户「库是部分迁移的，别据此判断条目丢失」（审计 GA3-13）。
func (s *Service) MigrationState() MigrationState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := MigrationState{Ran: s.migrationRan, Failed: s.migrationErr != nil}
	if s.migrationErr != nil {
		st.Reason = s.migrationErr.Error()
	}
	return st
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
		st, err := s.openSQLite(gdb, DefaultDir())
		if err != nil {
			s.err = err
			return nil, err
		}
		s.store = st
	}
	return s.store, nil
}

// openSQLite 打开 SQLite 后端并先尝试旧 Markdown 迁移。迁移失败记入
// migrationErr（MigrationState 可查）但**不**阻断 Store——知识库仍可用，
// 只是可能处于部分迁移状态；启动路径/面板据此提示（审计 GA3-13）。
// legacyDir 单独传参，便于测试用临时目录覆盖「迁移失败可见」路径。
// 调用方须持有 s.mu。
func (s *Service) openSQLite(gdb *sql.DB, legacyDir string) (*Store, error) {
	s.migrationRan = true
	if _, err := MigrateLegacyKnowledge(gdb, legacyDir); err != nil {
		s.migrationErr = err
		log.Printf("[hephaestus] 知识库迁移失败: %v", err)
		slog.Warn("knowledge: 旧 Markdown 知识库迁移失败，库为部分迁移状态", "error", err)
	}
	return OpenSQLite(gdb)
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
