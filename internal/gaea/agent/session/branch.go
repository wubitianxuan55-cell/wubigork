package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// PolicySnapshot 是会话的审批与断网策略随行存档（codex 蒸馏差距档案刀4，
// threads 行 sandbox_policy/approval_mode 随行折算）：会话创建时把生效策略
// 快照进 meta，恢复旧会话用存档策略而非当前全局。只记 controller 可恢复面
// （审批模式/权限级别/断网闸）；bash 沙箱模式是装配期工具配置，不在列。
type PolicySnapshot struct {
	// SandboxNetwork 是 [sandbox] network 的存档值（true=网络开）。
	SandboxNetwork bool   `json:"sandbox_network"`
	ApprovalMode   string `json:"approval_mode,omitempty"`
	// PermissionMode 是权限级别回退态（"allow"/"ask"/"deny"，ParseDecision 口径）。
	PermissionMode string `json:"permission_mode"`
}

// BranchMeta is the small sidecar record that turns flat session files into a
// navigable conversation tree. The conversation itself remains in the .jsonl
// file; metadata lives beside it at <session>.meta.
type BranchMeta struct {
	ID               string `json:"id"`
	Name             string `json:"name,omitempty"`
	ParentID         string `json:"parent_id,omitempty"`
	ForkTurn         int    `json:"fork_turn,omitempty"`
	ForkMessageIndex int    `json:"fork_message_index,omitempty"`
	// Space 是会话空间自描述（S2）：分支 meta 随会话目录归属写入，读端空值
	// 降级 work（旧 meta 无此字段）。
	Space     string    `json:"space,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// PolicySnapshot 是策略随行存档（刀4）；nil=旧会话无存档（恢复走现状）。
	PolicySnapshot *PolicySnapshot `json:"policy_snapshot,omitempty"`
}

// BranchInfo combines sidecar metadata with the session file details needed for
// pickers and tree rendering.
type BranchInfo struct {
	BranchMeta
	Path    string
	ModTime time.Time
	Preview string
	Turns   int
}

// BranchID extracts a stable identifier from a session file path.
func BranchID(path string) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}

// BranchMetaPath returns the .meta sidecar path for a session file.
func BranchMetaPath(sessionPath string) string {
	if sessionPath == "" {
		return ""
	}
	return sessionPath + ".meta"
}

// LoadBranchMeta reads a branch metadata sidecar file.
func LoadBranchMeta(sessionPath string) (BranchMeta, bool, error) {
	metaPath := BranchMetaPath(sessionPath)
	if metaPath == "" {
		return BranchMeta{}, false, nil
	}
	b, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return BranchMeta{}, false, nil
		}
		return BranchMeta{}, false, err
	}
	var m BranchMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return BranchMeta{}, false, fmt.Errorf("decode branch meta %s: %w", metaPath, err)
	}
	if m.ID == "" {
		m.ID = BranchID(sessionPath)
	}
	return m, true, nil
}

// SaveBranchMeta persists branch metadata to a sidecar file.
func SaveBranchMeta(sessionPath string, m BranchMeta) error {
	metaPath := BranchMetaPath(sessionPath)
	if metaPath == "" {
		return fmt.Errorf("empty session path")
	}
	now := time.Now().UTC()
	if m.ID == "" {
		m.ID = BranchID(sessionPath)
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return fileutil.AtomicWrite(metaPath, b, 0o644)
}

// EnsureBranchMeta loads existing branch metadata or creates a fresh record.
func EnsureBranchMeta(sessionPath string) (BranchMeta, error) {
	if sessionPath == "" {
		return BranchMeta{}, fmt.Errorf("empty session path")
	}
	if m, ok, err := LoadBranchMeta(sessionPath); err != nil || ok {
		return m, err
	}
	now := time.Now().UTC()
	m := BranchMeta{
		ID:        BranchID(sessionPath),
		Space:     SpaceForPath(sessionPath), // S2：空间自描述按目录归属写入
		CreatedAt: now,
		UpdatedAt: now,
	}
	return m, SaveBranchMeta(sessionPath, m)
}

// TouchBranchMeta updates the branch metadata's UpdatedAt timestamp.
func TouchBranchMeta(sessionPath string) error {
	m, err := EnsureBranchMeta(sessionPath)
	if err != nil {
		return err
	}
	return SaveBranchMeta(sessionPath, m)
}

// ListBranches returns all branchable sessions under dir.
func ListBranches(dir string) ([]BranchInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []BranchInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, e.Name())
		preview, turns := previewSession(path)
		if turns == 0 {
			continue
		}
		meta, ok, err := LoadBranchMeta(path)
		if err != nil {
			continue
		}
		if !ok {
			meta = BranchMeta{
				ID:        BranchID(path),
				CreatedAt: info.ModTime().UTC(),
				UpdatedAt: info.ModTime().UTC(),
			}
		}
		if meta.ID == "" {
			meta.ID = BranchID(path)
		}
		out = append(out, BranchInfo{
			BranchMeta: meta,
			Path:       path,
			ModTime:    info.ModTime(),
			Preview:    preview,
			Turns:      turns,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}
