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

	"github.com/gaea/gaea/internal/scene"
	"github.com/gaea/gaea/internal/snapshot"
	"github.com/gaea/gaea/internal/types"
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
	versionMarker := filepath.Join(dir, ".gaea", "v4")
	if err := os.MkdirAll(filepath.Join(dir, ".gaea"), 0755); err != nil {
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

// WriteWorldview 写世界观为 markdown（向后兼容，原子写）
func (m *Manager) WriteWorldview(content string) error {
	return writeFileAtomic(filepath.Join(m.Dir, "worldview.md"), []byte(content))
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

// StyleFingerprintPath 文风指纹参考档路径（fingerprint.json）
func (m *Manager) StyleFingerprintPath() string {
	return filepath.Join(m.Dir, "fingerprint.json")
}

// ReadStyleFingerprint 读文风指纹参考档（不存在时返回 os 文件错误）
func (m *Manager) ReadStyleFingerprint() (*types.StyleFingerprintFile, error) {
	return loadJSON[types.StyleFingerprintFile](m.StyleFingerprintPath())
}

// WriteStyleFingerprint 写文风指纹参考档（原子写）
func (m *Manager) WriteStyleFingerprint(sf *types.StyleFingerprintFile) error {
	return writeJSON(m.StyleFingerprintPath(), sf)
}

// WriteChapter 写章节文件 chapters/NNN.md（自动补零，原子写）
func (m *Manager) WriteChapter(num int, content string) error {
	return writeFileAtomic(m.ChapterPath(num), []byte(content))
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

// ChapterBranchPath 返回分支章节文件路径 chapters/NNN{a,b,c}.md
func (m *Manager) ChapterBranchPath(num int, branch string) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d%s.md", num, branch))
}

// WriteChapterBranch 写分支章节（原子写）
func (m *Manager) WriteChapterBranch(num int, branch string, content string) error {
	return writeFileAtomic(m.ChapterBranchPath(num, branch), []byte(content))
}

// ReadChapterBranch 读分支章节
func (m *Manager) ReadChapterBranch(num int, branch string) (string, error) {
	data, err := os.ReadFile(m.ChapterBranchPath(num, branch))
	if err != nil {
		return "", err
	}
	return string(data), nil
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

// WriteChapterBranchSummary 写分支章节摘要 chapters/NNN{a,b,c}-summary.json
func (m *Manager) WriteChapterBranchSummary(num int, branch string, summary *types.ChapterSummary) error {
	return writeJSON(m.ChapterBranchSummaryPath(num, branch), summary)
}

// ReadChapterBranchSummary 读分支章节摘要
func (m *Manager) ReadChapterBranchSummary(num int, branch string) (*types.ChapterSummary, error) {
	return loadJSON[types.ChapterSummary](m.ChapterBranchSummaryPath(num, branch))
}

func (m *Manager) ChapterBranchSummaryPath(num int, branch string) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d%s-summary.json", num, branch))
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

// writeFileAtomic 原子写文件：先在目标同目录写临时文件 <name>.tmp-<随机>，
// 写入并 fsync 后 os.Rename 覆盖目标。任何一步失败都清理临时文件、保留旧文件，
// 避免崩溃或并发写把 characters.json/outline.json/章节等用户数据写坏。
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败 (%s): %w", path, err)
	}
	tmpPath := tmp.Name()
	// 无论成败都清理临时文件（rename 成功后该路径已不存在，Remove 报错可忽略）
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("写入临时文件失败 (%s): %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("同步临时文件失败 (%s): %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败 (%s): %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("替换文件失败 (%s): %w", path, err)
	}
	return nil
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败 (%s): %w", path, err)
	}
	if err := writeFileAtomic(path, data); err != nil {
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
	// 兼容旧品牌：新标记 .gaea/v4，旧项目 .wubigork/v4 同样识别
	if _, err := os.Stat(filepath.Join(m.Dir, ".gaea", "v4")); err == nil {
		return true
	}
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
	markerDir := filepath.Join(m.Dir, ".gaea")
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

// ReadAnalysisV2File 读 analysis-v2.json（V2 章节分析落盘）；文件缺失返回空文件
// 与 nil error（新项目正常态）。
func (m *Manager) ReadAnalysisV2File() (*types.AnalysisV2File, error) {
	f, err := loadJSON[types.AnalysisV2File](filepath.Join(m.Dir, "analysis-v2.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return &types.AnalysisV2File{Items: []types.ChapterAnalysisResult{}}, nil
		}
		return nil, err
	}
	return f, nil
}

// UpsertAnalysisV2 按章号 upsert 一条 V2 分析结果（同章重分析覆盖不追加），
// 保持章号升序，写回走原子替换。
func (m *Manager) UpsertAnalysisV2(res types.ChapterAnalysisResult) error {
	f, err := m.ReadAnalysisV2File()
	if err != nil {
		return err
	}
	replaced := false
	for i := range f.Items {
		if f.Items[i].ChapterNum == res.ChapterNum {
			f.Items[i] = res
			replaced = true
			break
		}
	}
	if !replaced {
		f.Items = append(f.Items, res)
	}
	sort.Slice(f.Items, func(i, j int) bool { return f.Items[i].ChapterNum < f.Items[j].ChapterNum })
	return writeJSON(filepath.Join(m.Dir, "analysis-v2.json"), f)
}

// ChapterMemoriesPath 单章故事记忆文件路径（memories/MMM-<n>-memory.json）。
func (m *Manager) ChapterMemoriesPath(chapterNum int) string {
	return filepath.Join(m.Dir, "memories", fmt.Sprintf("MMM-%03d-memory.json", chapterNum))
}

// ReadChapterMemories 读单章故事记忆；文件缺失返回空文件与 nil error（正常态）。
func (m *Manager) ReadChapterMemories(chapterNum int) (*types.StoryMemoryFile, error) {
	f, err := loadJSON[types.StoryMemoryFile](m.ChapterMemoriesPath(chapterNum))
	if err != nil {
		if os.IsNotExist(err) {
			return &types.StoryMemoryFile{Items: []types.StoryMemory{}}, nil
		}
		return nil, err
	}
	return f, nil
}

// WriteChapterMemories 整章替换写单章故事记忆（确定性 ID + 整文件替换 =
// 重分析幂等，规避 MuMu D1「重分析不清旧数据」）。空 items = 删除该章记忆文件
// （重分析后无合格记忆时不留陈旧档）。目录不存在自动创建。
func (m *Manager) WriteChapterMemories(chapterNum int, items []types.StoryMemory) error {
	path := m.ChapterMemoriesPath(chapterNum)
	if len(items) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 memories 目录失败: %w", err)
	}
	return writeJSON(path, &types.StoryMemoryFile{Items: items})
}

// ReadAllChapterMemories 读取全部章节的故事记忆（memories/MMM-*.json），
// 按章号升序拍平；单个文件损坏跳过不中断（记忆是增强数据，不因脏档丢全书）。
func (m *Manager) ReadAllChapterMemories() ([]types.StoryMemory, error) {
	entries, err := os.ReadDir(filepath.Join(m.Dir, "memories"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]types.StoryMemory, 0, 64)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "MMM-") || !strings.HasSuffix(name, "-memory.json") {
			continue
		}
		f, err := loadJSON[types.StoryMemoryFile](filepath.Join(m.Dir, "memories", name))
		if err != nil {
			continue // 脏档跳过
		}
		out = append(out, f.Items...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChapterNum < out[j].ChapterNum })
	return out, nil
}

// ── 重写版本库（t4-C3，契约 rewrites/<chapterNum>/）──────────────
//
// index.json 只存索引（无全文，防 MuMu 式 1.74MB 全量下发）；全文按
// <id>.json 按需读。Save 时 ID 为空则生成（rw-<chapter>-<unixnano>）。

// RewriteChapterDir 重写版本目录。
func (m *Manager) RewriteChapterDir(chapterNum int) string {
	return filepath.Join(m.Dir, "rewrites", fmt.Sprintf("%d", chapterNum))
}

// SaveRewriteVersion 保存重写版本（全文文件 + 索引条目同步）。
// v.ID 为空则生成；v.CreatedAt 为零值则取当前时间。
func (m *Manager) SaveRewriteVersion(v *types.RewriteVersion) error {
	if v == nil {
		return fmt.Errorf("版本为空")
	}
	if v.ID == "" {
		v.ID = fmt.Sprintf("rw-%d-%d", v.ChapterNum, time.Now().UnixNano())
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now()
	}
	dir := m.RewriteChapterDir(v.ChapterNum)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建重写版本目录失败: %w", err)
	}
	if err := writeJSON(filepath.Join(dir, v.ID+".json"), v); err != nil {
		return err
	}
	return m.syncRewriteIndex(*v)
}

// syncRewriteIndex 把版本摘要合并进 index.json（存在则更新，不存在追加）。
func (m *Manager) syncRewriteIndex(v types.RewriteVersion) error {
	dir := m.RewriteChapterDir(v.ChapterNum)
	idxPath := filepath.Join(dir, "index.json")
	idx := types.RewriteVersionIndex{
		ID: v.ID, ChapterNum: v.ChapterNum, Mode: v.Mode, Status: v.Status,
		Similarity: v.Similarity, CreatedAt: v.CreatedAt,
	}
	var list []types.RewriteVersionIndex
	if raw, err := os.ReadFile(idxPath); err == nil {
		var old struct {
			Items []types.RewriteVersionIndex `json:"items"`
		}
		if json.Unmarshal(raw, &old) == nil {
			list = old.Items
		}
	}
	replaced := false
	for i := range list {
		if list[i].ID == v.ID {
			list[i] = idx
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, idx)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) }) // 时间倒序
	return writeJSON(idxPath, &struct {
		Items []types.RewriteVersionIndex `json:"items"`
	}{Items: list})
}

// ListRewriteVersions 重写版本索引（时间倒序，无全文）。
func (m *Manager) ListRewriteVersions(chapterNum int) ([]types.RewriteVersionIndex, error) {
	idxPath := filepath.Join(m.RewriteChapterDir(chapterNum), "index.json")
	raw, err := os.ReadFile(idxPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.RewriteVersionIndex{}, nil
		}
		return nil, err
	}
	var old struct {
		Items []types.RewriteVersionIndex `json:"items"`
	}
	if err := json.Unmarshal(raw, &old); err != nil {
		return nil, fmt.Errorf("解析重写索引失败: %w", err)
	}
	return old.Items, nil
}

// GetRewriteVersion 读单个重写版本全文（含 OriginalContent/NewContent 快照）。
func (m *Manager) GetRewriteVersion(chapterNum int, id string) (*types.RewriteVersion, error) {
	if id == "" || strings.ContainsAny(id, `/\`) {
		return nil, fmt.Errorf("非法版本 ID")
	}
	return loadJSON[types.RewriteVersion](filepath.Join(m.RewriteChapterDir(chapterNum), id+".json"))
}

// UpdateRewriteVersion 状态流转/审计字段更新（全文文件与索引条目同步写）。
func (m *Manager) UpdateRewriteVersion(v *types.RewriteVersion) error {
	if v == nil || v.ID == "" {
		return fmt.Errorf("版本为空")
	}
	dir := m.RewriteChapterDir(v.ChapterNum)
	if err := writeJSON(filepath.Join(dir, v.ID+".json"), v); err != nil {
		return err
	}
	return m.syncRewriteIndex(*v)
}

// ChapterAnnotationsPath 单章标注文件路径（analysis/annotations/MMM.json）。
func (m *Manager) ChapterAnnotationsPath(chapterNum int) string {
	return filepath.Join(m.Dir, "analysis", "annotations", fmt.Sprintf("%03d.json", chapterNum))
}

// SaveChapterAnnotations 保存单章标注（分析完成后由分析代理写入）。
func (m *Manager) SaveChapterAnnotations(chapterNum int, items []types.Annotation) error {
	path := m.ChapterAnnotationsPath(chapterNum)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建标注目录失败: %w", err)
	}
	return writeJSON(path, &types.AnnotationFile{ChapterNum: chapterNum, Items: items})
}

// ReadChapterAnnotations 读单章标注；文件缺失返回空文件与 nil error（正常态）。
func (m *Manager) ReadChapterAnnotations(chapterNum int) (*types.AnnotationFile, error) {
	f, err := loadJSON[types.AnnotationFile](m.ChapterAnnotationsPath(chapterNum))
	if err != nil {
		if os.IsNotExist(err) {
			return &types.AnnotationFile{Items: []types.Annotation{}}, nil
		}
		return nil, err
	}
	return f, nil
}
