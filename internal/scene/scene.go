package scene

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// Manager 场景管理器 — 管理一个章节目录下的所有场景
// 目录布局: chapters/NNN/scenes/MMM-slug.md + MMM-slug.meta.json
type Manager struct {
	dir string // chapters/NNN/scenes/
}

// NewManager 创建场景管理器
func NewManager(chapterDir string) *Manager {
	return &Manager{dir: filepath.Join(chapterDir, "scenes")}
}

// ensureDir 确保 scenes/ 目录存在
func (m *Manager) ensureDir() error {
	return os.MkdirAll(m.dir, 0755)
}

// ── CRUD ──────────────────────────────────────────────────────

// Create 创建新场景（自动分配 ID）
// slug: URL 友好短名，如 "opening" / "confrontation"
func (m *Manager) Create(slug, title string) (*types.Scene, error) {
	if err := m.ensureDir(); err != nil {
		return nil, err
	}

	// 自动编号
	existing, err := m.List()
	if err != nil {
		return nil, err
	}
	nextNum := len(existing) + 1
	id := fmt.Sprintf("%03d-%s", nextNum, sanitizeSlug(slug))

	// 确保 ID 不重复
	for _, s := range existing {
		if s.ID == id {
			id = fmt.Sprintf("%03d-%s-%d", nextNum, sanitizeSlug(slug), time.Now().UnixNano()%1000)
			break
		}
	}

	meta := types.SceneMeta{
		ID:     id,
		Slug:   sanitizeSlug(slug),
		Title:  title,
		Status: types.SceneDraft,
		Order:  nextNum,
	}

	scene := &types.Scene{
		Meta:    meta,
		Content: "",
	}

	if err := m.Write(scene); err != nil {
		return nil, err
	}
	return scene, nil
}

// Read 读取场景（正文 + 元数据）
func (m *Manager) Read(sceneID string) (*types.Scene, error) {
	meta, err := m.readMeta(sceneID)
	if err != nil {
		return nil, fmt.Errorf("读取场景元数据 %s: %w", sceneID, err)
	}

	content, err := os.ReadFile(m.contentPath(sceneID))
	if err != nil {
		if os.IsNotExist(err) {
			content = []byte("")
		} else {
			return nil, fmt.Errorf("读取场景正文 %s: %w", sceneID, err)
		}
	}

	return &types.Scene{Meta: *meta, Content: string(content)}, nil
}

// Write 写入场景（正文 + 元数据）
func (m *Manager) Write(scene *types.Scene) error {
	if err := m.ensureDir(); err != nil {
		return err
	}

	// 计算字数并更新
	scene.Meta.WordCount = len([]rune(scene.Content))

	if err := m.writeMeta(&scene.Meta); err != nil {
		return err
	}

	return os.WriteFile(m.contentPath(scene.Meta.ID), []byte(scene.Content), 0644)
}

// UpdateMeta 仅更新场景元数据
func (m *Manager) UpdateMeta(sceneID string, meta *types.SceneMeta) error {
	meta.ID = sceneID
	return m.writeMeta(meta)
}

// Delete 删除场景文件（正文+元数据）
func (m *Manager) Delete(sceneID string) error {
	// 删正文
	if err := os.Remove(m.contentPath(sceneID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	// 删元数据
	if err := os.Remove(m.metaPath(sceneID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ── 集合操作 ────────────────────────────────────────────────

// List 列出所有场景元数据，按 Order 排序
func (m *Manager) List() ([]types.SceneMeta, error) {
	if err := m.ensureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.SceneMeta{}, nil
		}
		return nil, err
	}

	var metas []types.SceneMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		sceneID := strings.TrimSuffix(e.Name(), ".meta.json")
		meta, err := m.readMeta(sceneID)
		if err != nil {
			continue
		}
		metas = append(metas, *meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Order < metas[j].Order
	})

	return metas, nil
}

// Stitch 拼接所有场景为连续正文（Scrivenings 模式）
// 场景间用 "\n\n---\n\n" 分隔
func (m *Manager) Stitch() (string, error) {
	metas, err := m.List()
	if err != nil {
		return "", err
	}

	var parts []string
	for _, meta := range metas {
		content, err := os.ReadFile(m.contentPath(meta.ID))
		if err != nil {
			continue
		}
		parts = append(parts, string(content))
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

// Reorder 重排场景顺序 — sceneIDs 是新顺序
func (m *Manager) Reorder(sceneIDs []string) error {
	for i, id := range sceneIDs {
		meta, err := m.readMeta(id)
		if err != nil {
			return fmt.Errorf("场景 %s 不存在: %w", id, err)
		}
		meta.Order = i + 1
		if err := m.writeMeta(meta); err != nil {
			return err
		}
	}
	return nil
}

// ── 文件路径助手 ────────────────────────────────────────────

func (m *Manager) contentPath(sceneID string) string {
	return filepath.Join(m.dir, sceneID+".md")
}

func (m *Manager) metaPath(sceneID string) string {
	return filepath.Join(m.dir, sceneID+".meta.json")
}

func (m *Manager) readMeta(sceneID string) (*types.SceneMeta, error) {
	data, err := os.ReadFile(m.metaPath(sceneID))
	if err != nil {
		return nil, err
	}
	var meta types.SceneMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	meta.ID = sceneID // 文件名的 ID 是真相
	return &meta, nil
}

func (m *Manager) writeMeta(meta *types.SceneMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.metaPath(meta.ID), data, 0644)
}

// sanitizeSlug 清理 slug 为安全的文件名片段
func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, s)
	if s == "" {
		s = "scene"
	}
	return s
}
