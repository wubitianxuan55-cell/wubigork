package pantheon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

var safeSessionID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type sessionStore struct {
	root string
}

func newSessionStore(root string) (*sessionStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("众神殿数据目录为空")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("创建众神殿数据目录: %w", err)
	}
	return &sessionStore{root: root}, nil
}

func (s *sessionStore) sessionDir(id string) (string, error) {
	if !safeSessionID.MatchString(id) {
		return "", fmt.Errorf("非法议事 ID %q", id)
	}
	return filepath.Join(s.root, "sessions", id), nil
}

func (s *sessionStore) save(session *Session) error {
	if session == nil {
		return errors.New("议事为空")
	}
	dir, err := s.sessionDir(session.ID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化议事: %w", err)
	}
	return fileutil.AtomicWriteSync(filepath.Join(dir, "session.json"), data, 0o600)
}

func (s *sessionStore) get(id string) (*Session, error) {
	dir, err := s.sessionDir(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("议事 %s 不存在", id)
		}
		return nil, fmt.Errorf("读取议事 %s: %w", id, err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("解析议事 %s: %w", id, err)
	}
	return &session, nil
}

func (s *sessionStore) list() ([]SessionSummary, error) {
	dir := filepath.Join(s.root, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SessionSummary{}, nil
		}
		return nil, fmt.Errorf("读取议事项列表: %w", err)
	}
	out := make([]SessionSummary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !safeSessionID.MatchString(entry.Name()) {
			continue
		}
		session, err := s.get(entry.Name())
		if err != nil {
			continue
		}
		summary := SessionSummary{
			ID:         session.ID,
			Title:      session.Title,
			Workspace:  session.Request.Workspace,
			Status:     session.Status,
			Phase:      session.Phase,
			CreatedAt:  session.CreatedAt,
			UpdatedAt:  session.UpdatedAt,
			AgentCount: len(session.Participants),
			Error:      session.Error,
		}
		if session.Verdict != nil {
			summary.VerdictSummary = session.Verdict.Recommendation
			if summary.VerdictSummary == "" {
				summary.VerdictSummary = session.Verdict.Summary
			}
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt == out[j].UpdatedAt {
			return out[i].ID > out[j].ID
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out, nil
}

func (s *sessionStore) delete(id string) error {
	dir, err := s.sessionDir(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "session.json")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("议事 %s 不存在", id)
		}
		return err
	}
	return os.RemoveAll(dir)
}

func (s *sessionStore) loadOverrides() (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(s.root, "agents.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("解析 Agent 路径配置: %w", err)
	}
	return out, nil
}

func (s *sessionStore) saveOverrides(overrides map[string]string) error {
	data, err := json.MarshalIndent(overrides, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWriteSync(filepath.Join(s.root, "agents.json"), data, 0o600)
}
