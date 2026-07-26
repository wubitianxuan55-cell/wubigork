package project

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/scene"
	"github.com/wubigork/wubigork/internal/snapshot"
	"github.com/wubigork/wubigork/internal/types"
)

// Manager 项目管理器 — 一部小说一个文件夹
type Manager struct {
	Dir  string // 项目根目录
	Meta *types.ProjectMeta
}

// Create 新建小说项目目录
// 生成: project.json, worldview.md, characters.json, outline.json, chapters/, foreshadows.json
func Create(dir, title, genre, style, description string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建项目目录失败: %w", err)
	}

	now := time.Now()
	meta := &types.ProjectMeta{
		SchemaVersion: 1,
		Title:         title,
		Genre:         genre,
		Style:         style,
		Description:   description,
		CreatedAt:     now,
		LastOpenedAt:  now,
		Version:       1,
	}

	// project.json
	if err := writeJSON(filepath.Join(dir, "project.json"), meta); err != nil {
		return nil, err
	}

	// worldview.json（结构化，空）
	wf := types.WorldviewFile{Sections: DefaultSections()}
	if err := writeJSON(filepath.Join(dir, "worldview.json"), wf); err != nil {
		return nil, err
	}

	// characters.json（空）
	cf := types.CharacterFile{
		Characters:    []types.Character{},
		Organizations: []types.Organization{},
		Relationships: []types.Relationship{},
	}
	if err := writeJSON(filepath.Join(dir, "characters.json"), cf); err != nil {
		return nil, err
	}

	// outline.json（空）
	of := types.OutlineFile{
		Nodes: []types.OutlineNode{},
	}
	if err := writeJSON(filepath.Join(dir, "outline.json"), of); err != nil {
		return nil, err
	}

	// foreshadows.json（空）
	ff := types.ForeshadowFile{
		Items: []types.Foreshadow{},
	}
	if err := writeJSON(filepath.Join(dir, "foreshadows.json"), ff); err != nil {
		return nil, err
	}

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

	return &Manager{Dir: dir, Meta: meta}, nil
}

// Open 打开已有小说项目目录
func Open(dir string) (*Manager, error) {
	meta, err := loadJSON[types.ProjectMeta](filepath.Join(dir, "project.json"))
	if err != nil {
		return nil, fmt.Errorf("无效的项目目录（缺少 project.json）: %w", err)
	}

	meta.LastOpenedAt = time.Now()
	meta.Version++
	if err := writeJSON(filepath.Join(dir, "project.json"), meta); err != nil {
		slog.Warn("更新 project.json 最后打开时间失败", "error", err)
	}

	return &Manager{Dir: dir, Meta: meta}, nil
}

// Close 关闭项目（保存元信息）
func (m *Manager) Close() error {
	if m.Meta == nil {
		return nil
	}
	m.Meta.LastOpenedAt = time.Now()
	return writeJSON(filepath.Join(m.Dir, "project.json"), m.Meta)
}

// ── 文件读写辅助 ──────────────────────────────────────────────

// ReadWorldview 读世界观（优先 worldview.json，fallback worldview.md）
func (m *Manager) ReadWorldview() (string, error) {
	// 优先读结构化文件
	wf, err := m.ReadWorldviewFile()
	if err == nil && len(wf.Sections) > 0 {
		return wf.ToMarkdown(), nil
	}
	// fallback: 读取旧 worldview.md（不写入，迁移由 ReadWorldviewFile 负责）
	data, err := os.ReadFile(filepath.Join(m.Dir, "worldview.md"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteWorldview 写世界观为 markdown（向后兼容）
func (m *Manager) WriteWorldview(content string) error {
	return os.WriteFile(filepath.Join(m.Dir, "worldview.md"), []byte(content), 0644)
}

// ReadWorldviewFile 读 worldview.json（不存在时从 worldview.md 自动迁移）
func (m *Manager) ReadWorldviewFile() (*types.WorldviewFile, error) {
	wf, err := loadJSON[types.WorldviewFile](filepath.Join(m.Dir, "worldview.json"))
	if err == nil {
		// 已有足够 section → 直接返回
		if len(wf.Sections) >= 2 {
			return wf, nil
		}
		// 只有 1 个 section 且是旧的 "main" section → 需要迁移
		if len(wf.Sections) == 1 && wf.Sections[0].ID == "main" {
			// fall through to migration
		} else if len(wf.Sections) >= 1 {
			// 已有至少 1 个真实 section（不是旧的 main）→ 保留不迁移
			return wf, nil
		}
	}

	// 需要迁移：读取旧 worldview.md
	oldContent := ""
	if err == nil && len(wf.Sections) == 1 && wf.Sections[0].Content != "" {
		oldContent = wf.Sections[0].Content
	} else {
		data, ferr := os.ReadFile(filepath.Join(m.Dir, "worldview.md"))
		if ferr == nil {
			oldContent = strings.TrimSpace(string(data))
		}
	}
	// 如果完全没有旧内容，创建空结构
	if oldContent == "" || oldContent == "# 世界观" || strings.HasPrefix(oldContent, "# 世界观") {
		sections := DefaultSections()
		if err := writeJSON(filepath.Join(m.Dir, "worldview.json"), &types.WorldviewFile{Sections: sections}); err != nil {
			return nil, err
		}
		return &types.WorldviewFile{Sections: sections}, nil
	}

	sections := DefaultSections()
	sections = append([]types.WorldviewSection{{
		ID:      "legacy",
		Title:   "📋 旧版世界观（请整理到下方各维度）",
		Content: oldContent,
		Order:   0,
	}}, sections...)

	if err := writeJSON(filepath.Join(m.Dir, "worldview.json"), &types.WorldviewFile{Sections: sections}); err != nil {
		return nil, err
	}
	return &types.WorldviewFile{Sections: sections}, nil
}

// WriteWorldviewFile 写 worldview.json
func (m *Manager) WriteWorldviewFile(wf *types.WorldviewFile) error {
	return writeJSON(filepath.Join(m.Dir, "worldview.json"), wf)
}

// ReadCharacters 读 characters.json
func (m *Manager) ReadCharacters() (*types.CharacterFile, error) {
	return loadJSON[types.CharacterFile](filepath.Join(m.Dir, "characters.json"))
}

// WriteCharacters 写 characters.json
func (m *Manager) WriteCharacters(cf *types.CharacterFile) error {
	return writeJSON(filepath.Join(m.Dir, "characters.json"), cf)
}

// ReadOutlines 读 outline.json
func (m *Manager) ReadOutlines() (*types.OutlineFile, error) {
	return loadJSON[types.OutlineFile](filepath.Join(m.Dir, "outline.json"))
}

// WriteOutlines 写 outline.json
func (m *Manager) WriteOutlines(of *types.OutlineFile) error {
	return writeJSON(filepath.Join(m.Dir, "outline.json"), of)
}

// ReadForeshadows 读 foreshadows.json
func (m *Manager) ReadForeshadows() (*types.ForeshadowFile, error) {
	return loadJSON[types.ForeshadowFile](filepath.Join(m.Dir, "foreshadows.json"))
}

// WriteForeshadows 写 foreshadows.json
func (m *Manager) WriteForeshadows(ff *types.ForeshadowFile) error {
	return writeJSON(filepath.Join(m.Dir, "foreshadows.json"), ff)
}

// WriteChapter 写章节文件 chapters/NNN.md（自动补零）
func (m *Manager) WriteChapter(num int, content string) error {
	return os.WriteFile(m.ChapterPath(num), []byte(content), 0644)
}

// ReadChapter 读章节文件
func (m *Manager) ReadChapter(num int) (string, error) {
	data, err := os.ReadFile(m.ChapterPath(num))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ChapterPath 返回章节文件路径 chapters/NNN.md
func (m *Manager) ChapterPath(num int) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d.md", num))
}

// WriteChapterSummary 写章节摘要 chapters/NNN-summary.json
func (m *Manager) WriteChapterSummary(num int, summary *types.ChapterSummary) error {
	return writeJSON(m.ChapterSummaryPath(num), summary)
}

// ReadChapterSummary 读章节摘要
func (m *Manager) ReadChapterSummary(num int) (*types.ChapterSummary, error) {
	return loadJSON[types.ChapterSummary](m.ChapterSummaryPath(num))
}

func (m *Manager) ChapterSummaryPath(num int) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d-summary.json", num))
}

// ReadAllChapterSummaries 一次扫描读取所有章节摘要（替代逐个文件探测）
func (m *Manager) ReadAllChapterSummaries() ([]types.ChapterSummary, error) {
	entries, err := os.ReadDir(filepath.Join(m.Dir, "chapters"))
	if err != nil {
		return nil, err
	}

	// 收集匹配的文件名并排序（零填充数字，字典序即数值序）
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "-summary.json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	summaries := make([]types.ChapterSummary, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(m.Dir, "chapters", name))
		if err != nil {
			continue
		}
		var s types.ChapterSummary
		if json.Unmarshal(data, &s) == nil {
			summaries = append(summaries, s)
		}
	}
	return summaries, nil
}

// ── Lorebook ──────────────────────────────────────────────

// ReadLorebook 读取 lorebook.json（不存在时返回空）
func (m *Manager) ReadLorebook() (*types.LorebookFile, error) {
	path := filepath.Join(m.Dir, "lorebook.json")
	lf, err := loadJSON[types.LorebookFile](path)
	if err != nil {
		return &types.LorebookFile{}, nil
	}
	return lf, nil
}

// WriteLorebook 写入 lorebook.json
func (m *Manager) WriteLorebook(lf *types.LorebookFile) error {
	return writeJSON(filepath.Join(m.Dir, "lorebook.json"), lf)
}

// ── 上下文构建 ──────────────────────────────────────────────

// LoadContext 加载完整 ProjectContext（Phase 1 简化版，不含 Memory）
func (m *Manager) LoadContext(currentOutlineID string) (*types.ProjectContext, error) {
	worldview, err := m.ReadWorldview()
	if err != nil {
		slog.Warn("LoadContext: 读取世界观失败", "error", err)
	}
	chars, err := m.ReadCharacters()
	if err != nil {
		slog.Warn("LoadContext: 读取角色失败", "error", err)
	}
	outlines, err := m.ReadOutlines()
	if err != nil {
		slog.Warn("LoadContext: 读取大纲失败", "error", err)
	}
	foreshadows, err := m.ReadForeshadows()
	if err != nil {
		slog.Warn("LoadContext: 读取伏笔失败", "error", err)
	}

	ctx := &types.ProjectContext{
		Project:   *m.Meta,
		Worldview: worldview,
	}

	if chars != nil {
		ctx.Characters = chars.Characters
		ctx.Organizations = chars.Organizations
		ctx.Relationships = chars.Relationships
	}
	if outlines != nil {
		ctx.Outlines = outlines.Nodes
		ctx.StoryThread = outlines.StoryThread
		// 找到当前大纲节点
		for i := range outlines.Nodes {
			findNode(outlines.Nodes[i], currentOutlineID, &ctx.CurrentOutline)
		}
		// 找到当前节点的父卷上下文
		for i := range outlines.Nodes {
			if parent := findParentVolume(outlines.Nodes[i], currentOutlineID); parent != nil {
				ctx.VolumeContext = fmt.Sprintf("卷: %s\n卷摘要: %s\n卷关键点: %s",
					parent.Title,
					parent.Summary,
					strings.Join(parent.KeyPoints, " / "),
				)
				break
			}
		}
	}
	if foreshadows != nil {
		ctx.Foreshadows = foreshadows.Items
	}

	return ctx, nil
}

func findNode(node types.OutlineNode, targetID string, result **types.OutlineNode) {
	if *result != nil {
		return
	}
	if node.ID == targetID {
		*result = &node
		return
	}
	for _, child := range node.Children {
		findNode(child, targetID, result)
	}
}

// findParentVolume 查找包含 targetID 子节点的卷节点
func findParentVolume(node types.OutlineNode, targetID string) *types.OutlineNode {
	for _, child := range node.Children {
		if child.ID == targetID {
			return &node
		}
		if len(child.Children) > 0 {
			if result := findParentVolume(child, targetID); result != nil {
				return result
			}
		}
	}
	return nil
}

// ── 内部辅助 ─────────────────────────────────────────────────

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败 (%s): %w", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败 (%s): %w", path, err)
	}
	return nil
}

func loadJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", path, err)
	}
	return &val, nil
}

func DefaultSections() []types.WorldviewSection {
	return []types.WorldviewSection{
		{ID: "era", Title: "时代背景", Content: "", Order: 1},
		{ID: "geography", Title: "地理风貌", Content: "", Order: 2},
		{ID: "factions", Title: "势力格局", Content: "", Order: 3},
		{ID: "rules", Title: "规则体系", Content: "", Order: 4},
		{ID: "culture", Title: "文化习俗", Content: "", Order: 5},
		{ID: "history", Title: "历史事件", Content: "", Order: 6},
	}
}

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
			return m.finalizeV4Migration()
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// 解析章节号
		name := e.Name()
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
		}
	}

	return m.finalizeV4Migration()
}

func (m *Manager) finalizeV4Migration() error {
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
	return m.ReadChapter(num)
}

// ForEachChapter 遍历所有存在的章节，回调返回 error 时跳过该章（continue）
// 用于替代 export/search/stats 等模块中 for i:=1;;i++ 的重复模式
func (m *Manager) ForEachChapter(fn func(chapterNum int, content string) error) error {
	for i := 1; ; i++ {
		content, err := m.ReadChapter(i)
		if err != nil {
			break
		}
		if content == "" {
			continue
		}
		if err := fn(i, content); err != nil {
			continue
		}
	}
	return nil
}
