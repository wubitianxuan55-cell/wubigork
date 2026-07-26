# Phase 4.0 — 数据基础层 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构数据层：新增场景引擎（场景级 CRUD + 元数据 + 拼接视图）、项目目录 v3→v4 自动迁移（非破坏性向后兼容）、快照引擎（diff 存储 + 时间线）

**Architecture:** 新增 `internal/scene/`（场景原子单元管理）、`internal/snapshot/`（行级 diff 快照）。扩展 `internal/types/`（Scene/SceneMeta/Snapshot 类型）。升级 `internal/project/`（v4 目录结构 + 自动迁移）。现有 Agent 通过适配层保持兼容，不破坏任何现有 API。

**Tech Stack:** Go 1.22+, 纯标准库 + 现有依赖, 文件即真相原则, 零新依赖

## Global Constraints

- 文件即真相：所有数据源是文件，不引入数据库
- 向后兼容：v4 能打开 v3 项目，自动迁移且不破坏原始数据
- 零新依赖：不引入任何新的 Go module 依赖
- 不破坏现有 API：所有现有 Wails 绑定方法签名不变
- 纯本地：快照存本地文件系统，diff 算法用标准库实现
- 测试覆盖：每个新包必须有单元测试
- 遵循现有 slog 日志规范

---

## File Structure

| 文件 | 职责 | 操作 |
|------|------|------|
| `internal/types/types.go` | 新增 Scene/SceneMeta/Snapshot 类型，扩展 OutlineNode | 修改 |
| `internal/scene/scene.go` | SceneManager — 场景 CRUD、拼接、重排、元数据管理 | 新建 |
| `internal/scene/scene_test.go` | SceneManager 单元测试 | 新建 |
| `internal/snapshot/snapshot.go` | SnapshotStore — 创建/列出/恢复/对比快照 | 新建 |
| `internal/snapshot/snapshot_test.go` | SnapshotStore 单元测试 | 新建 |
| `internal/project/project.go` | 升级 Create/Open 到 v4 目录结构，新增 MigrateV3ToV4，新增 Scene/Snapshot 便捷方法 | 修改 |
| `internal/project/project_test.go` | v4 迁移测试 | 新建 |
| `internal/chapter/chapter.go` | Generate 适配场景存储（写单场景到 chapters/NNN/scenes/） | 修改 |
| `internal/app/chapter_handler.go` | GenerateChapter 适配场景存储 | 修改 |
| `internal/analysis/evolution.go` | EvolveAfterChapter 适配新目录结构 | 修改 |

---

### Task 1: 新增 Scene/Snapshot 类型定义

**Files:**
- Modify: `internal/types/types.go` — 在 LorebookFile 和 StoryMemory 之间插入新类型

**Interfaces:**
- Produces: `type SceneMeta struct` — 场景元数据
- Produces: `type Scene struct` — 场景完整数据（元数据+正文）
- Produces: `type SceneStatus string` — 场景状态常量
- Produces: `type Snapshot struct` — 快照条目
- Produces: `type SnapshotChain struct` — 场景的快照链
- Produces: `const` 块 — SceneStatus 枚举值
- Modifies: `OutlineNode.SceneRefs []string` — 新增字段，引用场景 ID

- [ ] **Step 1: 在 types.go 的 LorebookFile 和 StoryMemory 之间插入新类型**

在 `internal/types/types.go` 第 203 行（`// ── 故事记忆` 注释之前）插入以下代码：

```go
// ── 场景（v4 原子写作单元）────────────────────────────────

// SceneStatus 场景写作状态
type SceneStatus string

const (
	SceneDraft    SceneStatus = "draft"
	SceneRevising SceneStatus = "revising"
	SceneDone     SceneStatus = "done"
	ScenePaused   SceneStatus = "paused"
)

// SceneMeta 场景元数据，存储为 scenes/MMM-slug.meta.json
type SceneMeta struct {
	ID        string      `json:"id"`        // 唯一标识，如 "001-opening"
	Slug      string      `json:"slug"`      // URL 友好短名
	Title     string      `json:"title"`     // 场景名
	Summary   string      `json:"summary"`   // 一句话概要
	POVCharID string      `json:"pov_char_id,omitempty"`   // POV 角色 ID
	Location  string      `json:"location,omitempty"`       // 地点
	TimeOfDay string      `json:"time_of_day,omitempty"`    // 时间（黎明/早晨/下午/黄昏/夜晚/深夜）
	Emotion   string      `json:"emotion,omitempty"`        // 情感基调
	Tags      []string    `json:"tags,omitempty"`           // 标签: climax/action/dialogue/...
	Status    SceneStatus `json:"status"`
	WordCount int         `json:"word_count"`
	Order     int         `json:"order"` // 在章节内的排序
}

// Scene 场景完整数据（元数据 + 正文）
type Scene struct {
	Meta    SceneMeta `json:"meta"`
	Content string    `json:"content"` // markdown 正文
}

// ── 快照（场景版本历史）───────────────────────────────────

// Snapshot 单个快照 — 存储行级增量 diff
type Snapshot struct {
	ID        string    `json:"id"`        // 快照 ID（时间戳）
	SceneID   string    `json:"scene_id"`  // 所属场景 ID
	Timestamp time.Time `json:"timestamp"` // 快照时间
	Label     string    `json:"label,omitempty"`   // 可选标签，如 "AI 改写前"
	Trigger   string    `json:"trigger,omitempty"` // 触发原因: "manual" / "ai-rewrite" / "ai-generate"
	DiffLines []DiffLine `json:"diff_lines"`       // 行级 diff（相对于上一快照或原始内容）
	WordCount int       `json:"word_count"`        // 快照时的字数
}

// DiffLine 行级差异条目 — 简单 unified diff 格式
type DiffLine struct {
	Type    string `json:"type"`    // "same" / "add" / "del"
	Content string `json:"content"` // 该行文本
	LineNum int    `json:"line_num"`
}

// SnapshotChain 一个场景的快照链（按时间排序）
type SnapshotChain struct {
	SceneID   string     `json:"scene_id"`
	Snapshots []Snapshot `json:"snapshots"`
}
```

- [ ] **Step 2: 在 OutlineNode 增加 SceneRefs 字段**

找到 `OutlineNode` struct（约第 122 行），在 `ChapterFile` 字段后、`OrderIndex` 字段前插入：

```go
	SceneRefs   []string          `json:"scene_refs,omitempty"`   // v4: 关联的场景 ID 列表
```

- [ ] **Step 3: 编译验证类型定义**

```bash
cd D:\AI\wubigork && go build ./internal/types/
```

Expected: 编译成功，无错误。

- [ ] **Step 4: Commit**

```bash
git add internal/types/types.go
git commit -m "feat(types): add Scene/SceneMeta/Snapshot types for v4 data foundation"
```

---

### Task 2: 场景引擎 — SceneManager 核心实现

**Files:**
- Create: `internal/scene/scene.go`
- Create: `internal/scene/scene_test.go`

**Interfaces:**
- Produces: `type Manager struct` — 场景管理器，绑定到一个章节目录
- Produces: `func NewManager(chapterDir string) *Manager`
- Produces: `func (m *Manager) Create(slug, title string) (*types.Scene, error)` — 创建新场景文件
- Produces: `func (m *Manager) Read(sceneID string) (*types.Scene, error)` — 读取场景
- Produces: `func (m *Manager) Write(scene *types.Scene) error` — 写入场景正文+元数据
- Produces: `func (m *Manager) UpdateMeta(sceneID string, meta *types.SceneMeta) error` — 仅更新元数据
- Produces: `func (m *Manager) Delete(sceneID string) error` — 删除场景
- Produces: `func (m *Manager) List() ([]types.SceneMeta, error)` — 列出所有场景元数据（按 Order 排序）
- Produces: `func (m *Manager) Stitch() (string, error)` — Scrivenings 拼接视图
- Produces: `func (m *Manager) Reorder(sceneIDs []string) error` — 重排场景顺序

- [ ] **Step 1: 创建 scene.go — Manager struct 和 NewManager**

```go
package scene

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/types"
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
```

- [ ] **Step 2: 实现 Create 方法**

```go
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
```

- [ ] **Step 3: 实现 Read/Write 方法**

```go
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

// ── 文件路径助手 ──

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
```

- [ ] **Step 4: 实现 List/Stitch/Reorder 方法**

```go
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
```

- [ ] **Step 5: 编译验证**

```bash
cd D:\AI\wubigork && go build ./internal/scene/
```

Expected: 编译成功。

- [ ] **Step 6: 编写单元测试 scene_test.go**

```go
package scene

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_CreateAndRead(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-test")
	defer os.RemoveAll(dir)

	m := NewManager(dir)

	// 创建场景
	scene, err := m.Create("opening", "Opening Scene")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if scene.Meta.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if scene.Meta.Status != "draft" {
		t.Fatalf("expected status draft, got %s", scene.Meta.Status)
	}
	if scene.Meta.Order != 1 {
		t.Fatalf("expected order 1, got %d", scene.Meta.Order)
	}

	// 验证文件存在
	if _, err := os.Stat(m.contentPath(scene.Meta.ID)); os.IsNotExist(err) {
		t.Fatal("content file not created")
	}
	if _, err := os.Stat(m.metaPath(scene.Meta.ID)); os.IsNotExist(err) {
		t.Fatal("meta file not created")
	}

	// 写入内容
	scene.Content = "# Hello\n\nThis is a test."
	if err := m.Write(scene); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 读取
	got, err := m.Read(scene.Meta.ID)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got.Content != scene.Content {
		t.Fatalf("content mismatch: got %q, want %q", got.Content, scene.Content)
	}
	if got.Meta.WordCount != len([]rune(scene.Content)) {
		t.Fatalf("word count mismatch: got %d, want %d", got.Meta.WordCount, len([]rune(scene.Content)))
	}
}

func TestManager_ListAndStitch(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-stitch")
	defer os.RemoveAll(dir)

	m := NewManager(dir)

	// 创建多个场景
	s1, _ := m.Create("a", "First")
	s1.Content = "Content one"
	m.Write(s1)

	s2, _ := m.Create("b", "Second")
	s2.Content = "Content two"
	m.Write(s2)

	s3, _ := m.Create("c", "Third")
	s3.Content = "Content three"
	m.Write(s3)

	// List
	metas, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(metas) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(metas))
	}
	if metas[0].Order != 1 || metas[1].Order != 2 || metas[2].Order != 3 {
		t.Fatal("orders not sequential")
	}

	// Stitch
	stitched, err := m.Stitch()
	if err != nil {
		t.Fatalf("Stitch failed: %v", err)
	}
	if stitched != "Content one\n\n---\n\nContent two\n\n---\n\nContent three" {
		t.Fatalf("stitch mismatch: %q", stitched)
	}
}

func TestManager_Reorder(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-reorder")
	defer os.RemoveAll(dir)

	m := NewManager(dir)

	s1, _ := m.Create("a", "First")
	s2, _ := m.Create("b", "Second")
	s3, _ := m.Create("c", "Third")

	// 反序排列
	err := m.Reorder([]string{s3.Meta.ID, s2.Meta.ID, s1.Meta.ID})
	if err != nil {
		t.Fatalf("Reorder failed: %v", err)
	}

	metas, _ := m.List()
	if metas[0].ID != s3.Meta.ID || metas[1].ID != s2.Meta.ID || metas[2].ID != s1.Meta.ID {
		t.Fatal("reorder did not take effect")
	}
}

func TestManager_Delete(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-delete")
	defer os.RemoveAll(dir)

	m := NewManager(dir)
	scene, _ := m.Create("tmp", "Temp")
	scene.Content = "temporary"
	m.Write(scene)

	if err := m.Delete(scene.Meta.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证文件已删除
	if _, err := os.Stat(m.contentPath(scene.Meta.ID)); !os.IsNotExist(err) {
		t.Fatal("content file not deleted")
	}
	if _, err := os.Stat(m.metaPath(scene.Meta.ID)); !os.IsNotExist(err) {
		t.Fatal("meta file not deleted")
	}
}
```

- [ ] **Step 7: 运行测试验证**

```bash
cd D:\AI\wubigork && go test ./internal/scene/ -v
```

Expected: 4 tests PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/scene/scene.go internal/scene/scene_test.go
git commit -m "feat(scene): add SceneManager — atomic scene CRUD, stitching, reordering"
```

---

### Task 3: 快照引擎 — SnapshotStore 实现

**Files:**
- Create: `internal/snapshot/snapshot.go`
- Create: `internal/snapshot/snapshot_test.go`

**Interfaces:**
- Produces: `type Store struct` — 快照存储
- Produces: `func NewStore(sceneDir string) *Store`
- Produces: `func (s *Store) Capture(sceneID, content, label, trigger string) (*types.Snapshot, error)`
- Produces: `func (s *Store) List(sceneID string) ([]types.Snapshot, error)`
- Produces: `func (s *Store) Restore(snapshotID, sceneID string) (string, error)`
- Produces: `func (s *Store) Diff(sceneID string, fromID, toID string) ([]types.DiffLine, error)`

- [ ] **Step 1: 创建 snapshot.go — Store struct 和 NewStore**

```go
package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/types"
)

// Store 快照存储管理器
// 快照目录: scenes/<sceneID>.snapshots/<snapshotID>.json
type Store struct {
	dir string // chapters/NNN/scenes/
}

// NewStore 创建快照存储
func NewStore(sceneDir string) *Store {
	return &Store{dir: sceneDir}
}

// snapDir 返回特定场景的快照目录
func (s *Store) snapDir(sceneID string) string {
	return filepath.Join(s.dir, sceneID+".snapshots")
}

// ensureSnapDir 确保快照目录存在
func (s *Store) ensureSnapDir(sceneID string) error {
	return os.MkdirAll(s.snapDir(sceneID), 0755)
}
```

- [ ] **Step 2: 实现 Capture — 创建快照**

```go
// Capture 创建场景当前状态的快照
// content: 当前正文，label: 可选标签，trigger: 触发原因
func (s *Store) Capture(sceneID, content, label, trigger string) (*types.Snapshot, error) {
	if err := s.ensureSnapDir(sceneID); err != nil {
		return nil, err
	}

	// 获取上一个快照用于 diff
	prevSnaps, err := s.List(sceneID)
	if err != nil {
		return nil, err
	}

	var prevContent string
	if len(prevSnaps) > 0 {
		// 重建上一个快照的完整内容
		prevContent, err = s.rebuild(sceneID, prevSnaps)
		if err != nil {
			prevContent = "" // 重建失败则视为无前序内容
		}
	}

	// 计算行级 diff
	diffLines := computeDiff(prevContent, content)

	now := time.Now()
	id := fmt.Sprintf("%d", now.UnixNano())

	snap := types.Snapshot{
		ID:        id,
		SceneID:   sceneID,
		Timestamp: now,
		Label:     label,
		Trigger:   trigger,
		DiffLines: diffLines,
		WordCount: len([]rune(content)),
	}

	// 写入快照文件
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.snapDir(sceneID), id+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return &snap, nil
}

// computeDiff 计算简单的行级 unified diff
// 返回的行序列可以用于重建内容：same/add 行组成新内容，del 行被排除
func computeDiff(oldContent, newContent string) []types.DiffLine {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var result []types.DiffLine

	// 简单逐行比较（LCS 太大材小用，线性扫描对文本场景够用）
	oi, ni := 0, 0
	for oi < len(oldLines) && ni < len(newLines) {
		if oldLines[oi] == newLines[ni] {
			result = append(result, types.DiffLine{
				Type: "same", Content: newLines[ni], LineNum: ni + 1,
			})
			oi++
			ni++
		} else {
			// 先尝试在新内容中找旧行（前向搜索 3 行）
			found := false
			for look := ni + 1; look < ni+4 && look < len(newLines); look++ {
				if newLines[look] == oldLines[oi] {
					// 中间的是新增行
					for ; ni < look; ni++ {
						result = append(result, types.DiffLine{
							Type: "add", Content: newLines[ni], LineNum: ni + 1,
						})
					}
					found = true
					break
				}
			}
			if !found {
				result = append(result, types.DiffLine{
					Type: "del", Content: oldLines[oi], LineNum: oi + 1,
				})
				oi++
				// 也添加新行（替换）
				if ni < len(newLines) {
					result = append(result, types.DiffLine{
						Type: "add", Content: newLines[ni], LineNum: ni + 1,
					})
					ni++
				}
			}
		}
	}

	// 剩余的旧行 → 删除
	for oi < len(oldLines) {
		result = append(result, types.DiffLine{
			Type: "del", Content: oldLines[oi], LineNum: oi + 1,
		})
		oi++
	}

	// 剩余的新行 → 新增
	for ni < len(newLines) {
		result = append(result, types.DiffLine{
			Type: "add", Content: newLines[ni], LineNum: ni + 1,
		})
		ni++
	}

	return result
}
```

- [ ] **Step 3: 实现 List/Restore/Rebuild**

```go
// List 列出场景的所有快照（按时间排序）
func (s *Store) List(sceneID string) ([]types.Snapshot, error) {
	dir := s.snapDir(sceneID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Snapshot{}, nil
		}
		return nil, err
	}

	var snaps []types.Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var snap types.Snapshot
		if json.Unmarshal(data, &snap) == nil {
			snaps = append(snaps, snap)
		}
	}

	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Timestamp.Before(snaps[j].Timestamp)
	})

	return snaps, nil
}

// Restore 恢复到指定快照的内容
func (s *Store) Restore(snapshotID, sceneID string) (string, error) {
	snaps, err := s.List(sceneID)
	if err != nil {
		return "", err
	}

	// 找到目标快照在列表中的位置
	targetIdx := -1
	for i, snap := range snaps {
		if snap.ID == snapshotID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return "", fmt.Errorf("快照 %s 不存在", snapshotID)
	}

	// 从头重建到目标快照
	return s.rebuildTo(sceneID, snaps[:targetIdx+1])
}

// rebuild 从快照链重建完整内容
func (s *Store) rebuild(sceneID string, snaps []types.Snapshot) (string, error) {
	return s.rebuildTo(sceneID, snaps)
}

// rebuildTo 从快照链的前 N 个重建内容
func (s *Store) rebuildTo(sceneID string, snaps []types.Snapshot) (string, error) {
	if len(snaps) == 0 {
		return "", nil
	}

	// 从第一个快照开始，逐个应用 diff
	var lines []string

	for _, snap := range snaps {
		var newLines []string
		for _, dl := range snap.DiffLines {
			switch dl.Type {
			case "same", "add":
				newLines = append(newLines, dl.Content)
			case "del":
				// 跳过
			}
		}
		lines = newLines // 每次快照的输出是下一次的输入（但我们的 diff 是相对前一个完整内容的）
	}

	// 实际上上面的逻辑是错误的——当前 diff 是相对于 prevContent 的
	// 正确做法：从空开始，按顺序重建
	content := ""
	for _, snap := range snaps {
		content = applyDiff(content, snap.DiffLines)
	}
	return content, nil
}

// applyDiff 将 diff 应用到旧内容上生成新内容
func applyDiff(oldContent string, diffs []types.DiffLine) string {
	oldLines := strings.Split(oldContent, "\n")
	if oldContent == "" {
		oldLines = []string{}
	}

	var result []string
	oi := 0

	for _, dl := range diffs {
		switch dl.Type {
		case "same":
			if oi < len(oldLines) && oldLines[oi] == dl.Content {
				result = append(result, dl.Content)
				oi++
			} else {
				// fallback: 直接使用 diff 中的内容
				result = append(result, dl.Content)
			}
		case "add":
			result = append(result, dl.Content)
		case "del":
			oi++ // 跳过旧行
		}
	}

	// 追加 diff 未覆盖的旧行
	for oi < len(oldLines) {
		result = append(result, oldLines[oi])
		oi++
	}

	return strings.Join(result, "\n")
}

// Diff 比较两个快照之间的差异
func (s *Store) Diff(sceneID, fromID, toID string) ([]types.DiffLine, error) {
	fromContent, err := s.Restore(fromID, sceneID)
	if err != nil {
		return nil, fmt.Errorf("读取 from 快照失败: %w", err)
	}
	toContent, err := s.Restore(toID, sceneID)
	if err != nil {
		return nil, fmt.Errorf("读取 to 快照失败: %w", err)
	}
	return computeDiff(fromContent, toContent), nil
}
```

- [ ] **Step 4: 编译验证**

```bash
cd D:\AI\wubigork && go build ./internal/snapshot/
```

Expected: 编译成功。

- [ ] **Step 5: 编写测试 snapshot_test.go**

```go
package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStore_CaptureAndRestore(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-snapshot-test")
	defer os.RemoveAll(dir)

	store := NewStore(dir)

	// 第一版快照
	content1 := "line one\nline two\nline three"
	snap1, err := store.Capture("test-scene", content1, "v1", "manual")
	if err != nil {
		t.Fatalf("Capture v1 failed: %v", err)
	}

	// 第二版快照（修改）
	content2 := "line one\nline two modified\nline three\nline four"
	snap2, err := store.Capture("test-scene", content2, "v2", "ai-rewrite")
	if err != nil {
		t.Fatalf("Capture v2 failed: %v", err)
	}

	// 列出快照
	snaps, err := store.List("test-scene")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}

	// 恢复到 v1
	restored1, err := store.Restore(snap1.ID, "test-scene")
	if err != nil {
		t.Fatalf("Restore v1 failed: %v", err)
	}
	if strings.TrimSpace(restored1) != strings.TrimSpace(content1) {
		t.Fatalf("restore v1 mismatch:\n got: %q\nwant: %q", restored1, content1)
	}

	// 恢复到 v2
	restored2, err := store.Restore(snap2.ID, "test-scene")
	if err != nil {
		t.Fatalf("Restore v2 failed: %v", err)
	}
	if strings.TrimSpace(restored2) != strings.TrimSpace(content2) {
		t.Fatalf("restore v2 mismatch:\n got: %q\nwant: %q", restored2, content2)
	}
}

func TestStore_Diff(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-snapshot-diff")
	defer os.RemoveAll(dir)

	store := NewStore(dir)

	content1 := "line one\nline two\nline three"
	snap1, _ := store.Capture("diff-scene", content1, "v1", "manual")

	content2 := "line one\nline two modified\nline four"
	snap2, _ := store.Capture("diff-scene", content2, "v2", "manual")

	diffs, err := store.Diff("diff-scene", snap1.ID, snap2.ID)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected non-empty diff")
	}
}

func TestStore_EmptyScene(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-snapshot-empty")
	defer os.RemoveAll(dir)

	store := NewStore(dir)
	snaps, err := store.List("nonexistent")
	if err != nil {
		t.Fatalf("List on nonexistent scene failed: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(snaps))
	}
}
```

- [ ] **Step 6: 运行测试验证**

```bash
cd D:\AI\wubigork && go test ./internal/snapshot/ -v
```

Expected: 3 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/snapshot/snapshot.go internal/snapshot/snapshot_test.go
git commit -m "feat(snapshot): add SnapshotStore — diff-based versioning, restore, diff"
```

---

### Task 4: 项目目录 v3→v4 升级

**Files:**
- Modify: `internal/project/project.go`
- Create: `internal/project/project_test.go`

**Interfaces:**
- Modifies: `Create()` — 创建 v4 目录结构（chapters/ 下每章有 scenes/ 子目录）
- Modifies: `Open()` — 检测 v3 项目并自动迁移
- Produces: `func (m *Manager) MigrateV3ToV4() error` — 显式迁移
- Produces: `func (m *Manager) IsV4() bool` — 检测是否为 v4 结构
- Produces: `func (m *Manager) SceneManager(chapterNum int) *scene.Manager` — 获取场景管理器
- Produces: `func (m *Manager) SnapshotStore(chapterNum int) *snapshot.Store` — 获取快照存储
- Produces: `func (m *Manager) ReadChapterAsStitch(num int) (string, error)` — v4 兼容读取

- [ ] **Step 1: 升级 Create() 创建 v4 目录结构**

在 `internal/project/project.go` 的 `Create` 函数中，替换 chapters/ 目录创建逻辑（第 78-80 行）：

将：
```go
	// chapters/ 目录
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		return nil, err
	}
```

替换为：
```go
	// chapters/ 目录（v4: 每章一个子目录含 scenes/）
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		return nil, err
	}
	// v4 标记文件
	versionMarker := filepath.Join(dir, ".wubigork", "v4")
	if err := os.MkdirAll(filepath.Join(dir, ".wubigork"), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(versionMarker, []byte("4"), 0644); err != nil {
		return nil, err
	}
```

- [ ] **Step 2: 在 project.go 末尾添加 IsV4 和 MigrateV3ToV4 方法**

在 `internal/project/project.go` 文件末尾（`DefaultSections` 函数之后）添加：

```go
// ── v4 支持 ──────────────────────────────────────────────────

// IsV4 检测项目是否为 v4 目录结构
func (m *Manager) IsV4() bool {
	_, err := os.Stat(filepath.Join(m.Dir, ".wubigork", "v4"))
	return err == nil
}

// MigrateV3ToV4 将 v3 项目迁移到 v4 结构（非破坏性）
// 原始 v3 文件备份到 _v3_backup/ 子目录
func (m *Manager) MigrateV3ToV4() error {
	if m.IsV4() {
		return nil // 已经是 v4
	}

	// 备份目录
	backupDir := filepath.Join(m.Dir, "_v3_backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 读取所有已有章节，迁移到 v4 结构
	chaptersDir := filepath.Join(m.Dir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return m.finalizeV4Migration(nil)
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// 解析章节号
		name := e.Name()
		if len(name) < 7 {
			continue
		}
		var chapterNum int
		if _, err := fmt.Sscanf(name, "%03d.md", &chapterNum); err != nil {
			continue
		}

		// 读旧内容
		oldPath := filepath.Join(chaptersDir, name)
		content, err := os.ReadFile(oldPath)
		if err != nil {
			continue
		}

		// 备份旧文件
		backupPath := filepath.Join(backupDir, name)
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			slog.Warn("v4迁移: 备份文件失败", "file", name, "error", err)
		}

		// 创建 v4 场景
		chDir := filepath.Join(chaptersDir, fmt.Sprintf("%03d", chapterNum))
		sm := scene.NewManager(chDir)
		sceneObj, err := sm.Create("chapter", fmt.Sprintf("第%d章", chapterNum))
		if err != nil {
			return fmt.Errorf("迁移第%d章失败: %w", chapterNum, err)
		}
		sceneObj.Content = string(content)
		if err := sm.Write(sceneObj); err != nil {
			return fmt.Errorf("写入迁移场景失败: %w", err)
		}

		// 迁移摘要文件（如果有）
		oldSummaryPath := filepath.Join(chaptersDir, fmt.Sprintf("%03d-summary.json", chapterNum))
		if summaryData, err := os.ReadFile(oldSummaryPath); err == nil {
			backupSummaryPath := filepath.Join(backupDir, fmt.Sprintf("%03d-summary.json", chapterNum))
			os.WriteFile(backupSummaryPath, summaryData, 0644)
			// 保持摘要在原位置（向后兼容读取）
		}
	}

	return m.finalizeV4Migration(nil)
}

func (m *Manager) finalizeV4Migration(err error) error {
	if err != nil {
		return err
	}
	// 写入 v4 标记
	markerDir := filepath.Join(m.Dir, ".wubigork")
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(markerDir, "v4"), []byte("4"), 0644)
}

// SceneManager 获取指定章节的场景管理器
func (m *Manager) SceneManager(chapterNum int) *scene.Manager {
	return scene.NewManager(filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d", chapterNum)))
}

// SnapshotStore 获取指定章节的快照存储
func (m *Manager) SnapshotStore(chapterNum int) *snapshot.Store {
	return snapshot.NewStore(filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d", chapterNum), "scenes"))
}

// ReadChapterAsStitch 以 v4 拼接视图读取章节（向后兼容 v3 blob 读取）
func (m *Manager) ReadChapterAsStitch(num int) (string, error) {
	if m.IsV4() {
		sm := m.SceneManager(num)
		content, err := sm.Stitch()
		if err == nil && content != "" {
			return content, nil
		}
	}
	// fallback: v3 blob 读取
	return m.ReadChapter(num)
}
```

- [ ] **Step 3: 添加 import**

在 `internal/project/project.go` 的 import 块中添加：
```go
	"github.com/wubigork/wubigork/internal/scene"
	"github.com/wubigork/wubigork/internal/snapshot"
```

- [ ] **Step 4: 编译验证**

```bash
cd D:\AI\wubigork && go build ./internal/project/
```

Expected: 编译成功。

- [ ] **Step 5: 编写迁移测试**

```go
package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateV4Project(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-v4-create")
	defer os.RemoveAll(dir)

	pm, err := Create(dir, "Test Novel", "Fantasy", "Default", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !pm.IsV4() {
		t.Fatal("new project should be v4")
	}

	// 验证 .wubigork/v4 标记文件存在
	if _, err := os.Stat(filepath.Join(dir, ".wubigork", "v4")); os.IsNotExist(err) {
		t.Fatal("v4 marker not created")
	}

	// 验证场景管理器可工作
	sm := pm.SceneManager(1)
	if sm == nil {
		t.Fatal("SceneManager returned nil")
	}
}

func TestMigrateV3ToV4(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-v3-migrate")
	defer os.RemoveAll(dir)

	// 模拟 v3 项目结构
	chaptersDir := filepath.Join(dir, "chapters")
	os.MkdirAll(chaptersDir, 0755)

	// 写 v3 格式文件
	os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{
		"schema_version": 1,
		"title": "Migration Test",
		"genre": "Fantasy",
		"style": "Default",
		"created_at": "2026-01-01T00:00:00Z",
		"last_opened_at": "2026-01-01T00:00:00Z"
	}`), 0644)

	os.WriteFile(filepath.Join(chaptersDir, "001.md"), []byte("# Chapter One\n\nOnce upon a time..."), 0644)
	os.WriteFile(filepath.Join(chaptersDir, "001-summary.json"), []byte(`{"title":"Chapter One","summary":"A beginning"}`), 0644)

	// 打开
	pm, err := Open(dir)
	if err != nil {
		t.Fatalf("Open v3 project failed: %v", err)
	}

	// 迁移
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("MigrateV3ToV4 failed: %v", err)
	}

	// 验证 v4 标记
	if !pm.IsV4() {
		t.Fatal("should be v4 after migration")
	}

	// 验证备份
	if _, err := os.Stat(filepath.Join(dir, "_v3_backup", "001.md")); os.IsNotExist(err) {
		t.Fatal("v3 backup not created")
	}

	// 验证场景可读
	content, err := pm.ReadChapterAsStitch(1)
	if err != nil {
		t.Fatalf("ReadChapterAsStitch failed: %v", err)
	}
	if content != "# Chapter One\n\nOnce upon a time..." {
		t.Fatalf("content mismatch: %q", content)
	}
}
```

- [ ] **Step 6: 运行测试验证**

```bash
cd D:\AI\wubigork && go test ./internal/project/ -v -run "TestCreateV4Project|TestMigrateV3ToV4"
```

Expected: 2 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/project/project.go internal/project/project_test.go
git commit -m "feat(project): add v4 directory structure, auto-migration, scene/snapshot accessors"
```

---

### Task 5: 集成 — 章节 Agent 适配场景存储

**Files:**
- Modify: `internal/chapter/chapter.go` — Generate 写完内容后存为场景
- Modify: `internal/app/chapter_handler.go` — GenerateChapter 改用场景存储
- Modify: `internal/analysis/evolution.go` — EvolveAfterChapter 适配 v4

**Interfaces:**
- Modifies: `Agent.Generate()` — 内部：写完章节后存到 v4 场景目录
- Produces: `Agent.saveAsScenes()` — 内部方法：把全文存为章节的单场景

- [ ] **Step 1: 在 chapter.go Generate 内部集成场景存储**

在 `internal/chapter/chapter.go` 的 `Generate` 方法中，找到写文件的位置。当前写文件的位置在 goroutine 的 quality check 通过后。我们需要在写文件时同时写入 v4 场景格式。

在 goroutine 中（约第 210 行附近，写完 `m.pm.WriteChapter(chapterNum, finalContent)` 之后），添加场景写入：

找到如下代码段（约第 240 行附近）：
```go
		// 写入章节文件
		if err := a.pm.WriteChapter(chapterNum, finalContent); err != nil {
```

修改为：
```go
		// 写入章节文件（v3 兼容）
		if err := a.pm.WriteChapter(chapterNum, finalContent); err != nil {
			slog.Error("写入章节文件失败", "chapter", chapterNum, "error", err)
		}

		// v4: 同时写入场景格式
		if err := a.saveAsScenes(chapterNum, finalContent, summary); err != nil {
			slog.Warn("v4: 写入场景格式失败", "chapter", chapterNum, "error", err)
		}
```

- [ ] **Step 2: 在 chapter.go 添加 saveAsScenes 辅助方法**

在 `internal/chapter/chapter.go` 文件末尾添加：

```go
// saveAsScenes 将章节全文存为 v4 场景格式（单场景迁移）
// v4 项目：写为 scenes/001-chapter.md + meta.json
// v3 项目：跳过（由 WriteChapter 处理）
func (a *Agent) saveAsScenes(chapterNum int, content string, summary *types.ChapterSummary) error {
	if !a.pm.IsV4() {
		return nil // v3 项目不写场景
	}

	sm := a.pm.SceneManager(chapterNum)
	
	// 检查是否已有场景
	existing, _ := sm.List()
	if len(existing) > 0 {
		// 已有场景，更新第一个场景的内容
		sceneObj, err := sm.Read(existing[0].ID)
		if err != nil {
			return err
		}
		sceneObj.Content = content
		if summary != nil {
			sceneObj.Meta.Summary = summary.Summary
			sceneObj.Meta.Emotion = summary.EmotionTone
		}
		return sm.Write(sceneObj)
	}

	// 创建新场景
	title := fmt.Sprintf("第%d章", chapterNum)
	if summary != nil && summary.Title != "" {
		title = summary.Title
	}

	sceneObj, err := sm.Create("chapter", title)
	if err != nil {
		return err
	}
	sceneObj.Content = content
	if summary != nil {
		sceneObj.Meta.Summary = summary.Summary
		sceneObj.Meta.Emotion = summary.EmotionTone
		sceneObj.Meta.Status = types.SceneDone
	}
	return sm.Write(sceneObj)
}
```

需要在文件头部添加 import：
```go
	"github.com/wubigork/wubigork/internal/scene"
```

- [ ] **Step 3: 在 analysis/evolution.go 添加 v4 支持**

在 `EvolveAfterChapter` 函数中（约第 52 行），读取章节内容部分，增加 v4 优先读取：

找到读取章节内容的位置。当前 `EvolveAfterChapter` 接收 `chapterContent string` 作为参数（已由调用方传入），所以不需要修改。但我们需要确认调用方 `chapter_handler.go` 中的 `GenerateChapter` 能正确传内容。

当前的 `GenerateChapter` handler 在流式 goroutine 中已经正确传递了 `fullText`。Evolution 调用的是 `a.analysisAgent.EvolveAfterChapter(chapterNum, fullText, summary)`，这里的 `fullText` 就是完整正文，不依赖文件读取。所以 evolution.go 不需要修改。

- [ ] **Step 4: 编译验证整个项目**

```bash
cd D:\AI\wubigork && go build ./...
```

Expected: 全项目编译成功，零错误。

- [ ] **Step 5: 运行所有测试**

```bash
cd D:\AI\wubigork && go test ./internal/scene/ ./internal/snapshot/ ./internal/project/ -v
```

Expected: 所有测试 PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/chapter/chapter.go internal/app/chapter_handler.go
git commit -m "feat(integration): chapter agent writes to v4 scene format alongside v3 blob"
```

---

### Task 6: Wails Handler 暴露 v4 API

**Files:**
- Modify: `internal/app/chapter_handler.go` — 新增场景/v4 相关绑定方法

**Interfaces:**
- Produces: `func (a *App) GetChapterScenes(chapterNum int) ([]map[string]interface{}, error)` — 获取章节的场景列表
- Produces: `func (a *App) SaveScene(chapterNum int, sceneID string, content string) error` — 保存单个场景
- Produces: `func (a *App) ReorderScenes(chapterNum int, sceneIDs []string) error` — 重排场景
- Produces: `func (a *App) CreateSnapshot(sceneID string, chapterNum int, label string) (map[string]interface{}, error)` — 手动创建快照
- Produces: `func (a *App) ListSnapshots(sceneID string, chapterNum int) ([]map[string]interface{}, error)` — 列出快照
- Produces: `func (a *App) RestoreSnapshot(snapshotID string, sceneID string, chapterNum int) error` — 恢复快照
- Produces: `func (a *App) MigrateProjectToV4() error` — 手动触发迁移

- [ ] **Step 1: 在 chapter_handler.go 末尾添加 v4 绑定方法**

```go
// ── v4 场景 API ──────────────────────────────────────────────

// GetChapterScenes 获取章节的场景列表
func (a *App) GetChapterScenes(chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	if !pm.IsV4() {
		// v3 项目：返回单场景视图
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, err
		}
		return []map[string]interface{}{{
			"id":      fmt.Sprintf("%03d-chapter", chapterNum),
			"title":   fmt.Sprintf("第%d章", chapterNum),
			"content": content,
			"status":  "done",
			"order":   1,
		}}, nil
	}

	sm := pm.SceneManager(chapterNum)
	scenes, err := sm.List()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, meta := range scenes {
		scene, err := sm.Read(meta.ID)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":         scene.Meta.ID,
			"slug":       scene.Meta.Slug,
			"title":      scene.Meta.Title,
			"summary":    scene.Meta.Summary,
			"povCharId":  scene.Meta.POVCharID,
			"location":   scene.Meta.Location,
			"timeOfDay":  scene.Meta.TimeOfDay,
			"emotion":    scene.Meta.Emotion,
			"tags":       scene.Meta.Tags,
			"status":     string(scene.Meta.Status),
			"wordCount":  scene.Meta.WordCount,
			"order":      scene.Meta.Order,
			"content":    scene.Content,
		})
	}
	return result, nil
}

// SaveScene 保存单个场景
func (a *App) SaveScene(chapterNum int, sceneID string, content string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}

	if !pm.IsV4() {
		return pm.WriteChapter(chapterNum, content)
	}

	sm := pm.SceneManager(chapterNum)
	scene, err := sm.Read(sceneID)
	if err != nil {
		return err
	}
	scene.Content = content
	return sm.Write(scene)
}

// ReorderScenes 重排场景顺序
func (a *App) ReorderScenes(chapterNum int, sceneIDs []string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if !pm.IsV4() {
		return nil // v3 项目无场景可排
	}
	return pm.SceneManager(chapterNum).Reorder(sceneIDs)
}

// CreateSnapshot 手动创建场景快照
func (a *App) CreateSnapshot(sceneID string, chapterNum int, label string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	var content string
	if pm.IsV4() {
		scene, err := pm.SceneManager(chapterNum).Read(sceneID)
		if err != nil {
			return nil, err
		}
		content = scene.Content
	} else {
		var err error
		content, err = pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, err
		}
	}

	store := pm.SnapshotStore(chapterNum)
	snap, err := store.Capture(sceneID, content, label, "manual")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        snap.ID,
		"timestamp": snap.Timestamp,
		"label":     snap.Label,
		"wordCount": snap.WordCount,
	}, nil
}

// ListSnapshots 列出场景的所有快照
func (a *App) ListSnapshots(sceneID string, chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	store := pm.SnapshotStore(chapterNum)
	snaps, err := store.List(sceneID)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, snap := range snaps {
		result = append(result, map[string]interface{}{
			"id":        snap.ID,
			"timestamp": snap.Timestamp,
			"label":     snap.Label,
			"trigger":   snap.Trigger,
			"wordCount": snap.WordCount,
		})
	}
	return result, nil
}

// RestoreSnapshot 恢复到指定快照
func (a *App) RestoreSnapshot(snapshotID string, sceneID string, chapterNum int) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}

	store := pm.SnapshotStore(chapterNum)
	content, err := store.Restore(snapshotID, sceneID)
	if err != nil {
		return err
	}

	if pm.IsV4() {
		scene, err := pm.SceneManager(chapterNum).Read(sceneID)
		if err != nil {
			return err
		}
		scene.Content = content
		return pm.SceneManager(chapterNum).Write(scene)
	}

	return pm.WriteChapter(chapterNum, content)
}

// MigrateProjectToV4 手动触发项目迁移到 v4
func (a *App) MigrateProjectToV4() error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	return pm.MigrateV3ToV4()
}

// IsProjectV4 检查当前项目是否为 v4 结构
func (a *App) IsProjectV4() bool {
	pm := a.getPM()
	if pm == nil {
		return false
	}
	return pm.IsV4()
}
```

- [ ] **Step 2: 编译验证**

```bash
cd D:\AI\wubigork && go build ./...
```

Expected: 全项目编译成功。

- [ ] **Step 3: Commit**

```bash
git add internal/app/chapter_handler.go
git commit -m "feat(handlers): expose v4 scene/snapshot/migration Wails bindings"
```

---

## Verification Checklist

Phase 4.0 完成验收：

- [ ] `go build ./...` 全项目编译通过
- [ ] `go test ./internal/scene/... -v` — 4 个场景引擎测试通过
- [ ] `go test ./internal/snapshot/... -v` — 3 个快照引擎测试通过
- [ ] `go test ./internal/project/... -v -run "TestCreateV4Project|TestMigrateV3ToV4"` — 2 个迁移测试通过
- [ ] 新建项目 = v4 结构（有 `.wubigork/v4` 标记）
- [ ] 打开 v3 项目 + `MigrateProjectToV4()` = 自动备份 + 创建场景
- [ ] 章节生成后写入 v4 场景格式
- [ ] v3 项目读写行为完全不受影响
- [ ] `wails build` 生成 exe 成功
