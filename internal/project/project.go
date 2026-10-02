package project

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/scene"
	"github.com/gaea/gaea/internal/snapshot"
	"github.com/gaea/gaea/internal/types"
)

// Manager 项目管理器 — 一部小说一个文件夹
type Manager struct {
	Dir  string // 项目根目录
	Meta *types.ProjectMeta

	// fileMu 串行化 Manager 内部的「整表读-改-写」复合操作（审计 P1 IN1-07）：
	// UpsertAnalysisV2 / syncRewriteIndex 都是 读 JSON→改→writeJSON 整表覆盖，
	// 并发时后写者用旧快照覆盖先写者丢更新（plans.json 同构问题在 app 层有
	// chapterPlanMu 专锁先例）。读路径不持锁：写走原子替换，读到的要么旧要么
	// 新，无半截文件。
	fileMu sync.Mutex
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

// WriteMeta 将当前元信息写回 project.json（UpdateProjectMeta 的落盘出口）。
// Version 不动——递增语义归 Open 的「打开次数」口径。
func (m *Manager) WriteMeta() error {
	if m.Meta == nil {
		return nil
	}
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
	return fileutil.AtomicWrite(filepath.Join(m.Dir, "worldview.md"), []byte(content), 0o600)
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
// ReadStyleDigest 读风格摘要档（长篇刀5）；文件缺失返回 nil（未构建态）。
func (m *Manager) ReadStyleDigest() (*types.StyleDigestFile, error) {
	df, err := loadJSON[types.StyleDigestFile](filepath.Join(m.Dir, "style_digest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return df, nil
}

// WriteStyleDigest 写风格摘要档（原子写）。
func (m *Manager) WriteStyleDigest(df *types.StyleDigestFile) error {
	return writeJSON(filepath.Join(m.Dir, "style_digest.json"), df)
}

// ClearStyleDigest 删除风格摘要档（未构建时 no-op）。
func (m *Manager) ClearStyleDigest() error {
	err := os.Remove(filepath.Join(m.Dir, "style_digest.json"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *Manager) WriteStyleFingerprint(sf *types.StyleFingerprintFile) error {
	return writeJSON(m.StyleFingerprintPath(), sf)
}

// WriteChapter 写章节文件 chapters/NNN.md（自动补零，原子写）
func (m *Manager) WriteChapter(num int, content string) error {
	return fileutil.AtomicWrite(m.ChapterPath(num), []byte(content), 0o600)
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
	return fileutil.AtomicWrite(m.ChapterBranchPath(num, branch), []byte(content), 0o600)
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

// chapterSummaryWithFile 一次扫描得到的一条章摘要及其来源文件名。
type chapterSummaryWithFile struct {
	File    string // chapters/ 下的摘要文件名（章号由此解析）
	Summary types.ChapterSummary
}

// readChapterSummariesWithFiles 扫描 chapters/ 取全部章摘要及其文件名，按文件名
// 升序（零填充数字 = 章号升序；分支章的字母后缀排在数字之后）。单个文件读失败
// 或 JSON 损坏跳过不中断（摘要缺失是增强面，不因脏档丢掉全库）。
func (m *Manager) readChapterSummariesWithFiles() ([]chapterSummaryWithFile, error) {
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

	out := make([]chapterSummaryWithFile, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(m.Dir, "chapters", name))
		if err != nil {
			continue
		}
		var s types.ChapterSummary
		if json.Unmarshal(data, &s) == nil {
			out = append(out, chapterSummaryWithFile{File: name, Summary: s})
		}
	}
	return out, nil
}

// ReadAllChapterSummaries 一次扫描读取所有章节摘要（替代逐个文件探测）。
// 只返回摘要本身；需要按章号过滤的调用方走 ReadLatestChapterSummaryBefore。
// 注意：结果含分支摘要（NNN{a,b,c}-summary.json）——分支浏览器等需要分支
// 剧情线的调用方用本入口；主线口径（大纲续写前情等）走
// ReadMainlineChapterSummaries，否则分支剧情会被当主线参考注入（N5）。
func (m *Manager) ReadAllChapterSummaries() ([]types.ChapterSummary, error) {
	items, err := m.readChapterSummariesWithFiles()
	if err != nil {
		return nil, err
	}
	summaries := make([]types.ChapterSummary, 0, len(items))
	for _, it := range items {
		summaries = append(summaries, it.Summary)
	}
	return summaries, nil
}

// ReadMainlineChapterSummaries 读取全部**主线**章节摘要（NNN-summary.json），
// 分支摘要（NNN{a,b,c}-summary.json）不参与——主线生成/续写的前情参考里
// 混入分支剧情线会污染走向（N5）。
func (m *Manager) ReadMainlineChapterSummaries() ([]types.ChapterSummary, error) {
	items, err := m.readChapterSummariesWithFiles()
	if err != nil {
		return nil, err
	}
	summaries := make([]types.ChapterSummary, 0, len(items))
	for _, it := range items {
		if mainChapterSummaryNum(it.File) <= 0 {
			continue
		}
		summaries = append(summaries, it.Summary)
	}
	return summaries, nil
}

// ReadLatestChapterSummaryBefore 取章号严格小于 chapterNum 的最近一章摘要；
// 没有合格摘要（含 chapterNum<=1 的第一章）返回 (nil, nil)。
//
// 章号取自**文件名**（chapters/NNN-summary.json / NNN{a,b,c}-summary.json 的前导
// 数字）：ChapterSummary 本身没有章号字段，而按文件名升序读出来的「最后一个」
// 是全书最后一章——生成第 1 章时若拿它当「上一章」，prompt 里会塞进全书结局
// （G5）。故这里显式解析文件名过滤，不用「取最后一个」的兜底。
// chapters/ 目录缺失视为「无摘要」（新项目正常态），不报错。
func (m *Manager) ReadLatestChapterSummaryBefore(chapterNum int) (*types.ChapterSummary, error) {
	if chapterNum <= 1 {
		return nil, nil
	}
	items, err := m.readChapterSummariesWithFiles()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	// 文件名升序 = 章号升序，正向扫描时「最后一个合格者」即最近一章。
	// 只认主线摘要（NNN-summary.json）：分支摘要 NNN{a,b,c}-summary.json 与主线
	// 同章号却排在后面，收进来会把分支剧情当成主线的「上一章」（N5 的放大器）。
	var best *types.ChapterSummary
	for i := range items {
		cn := mainChapterSummaryNum(items[i].File)
		if cn > 0 && cn < chapterNum {
			best = &items[i].Summary
		}
	}
	return best, nil
}

// mainChapterSummaryNum 解析主线章摘要文件名（"006-summary.json" → 6）；
// 分支摘要（"006a-summary.json"）与其它形状一律返回 0（不参与主线口径）。
func mainChapterSummaryNum(name string) int {
	const suffix = "-summary.json"
	if !strings.HasSuffix(name, suffix) {
		return 0
	}
	num := strings.TrimSuffix(name, suffix)
	if num == "" {
		return 0
	}
	for _, r := range num {
		if r < '0' || r > '9' {
			return 0
		}
	}
	return leadDigits(num)
}

// MaxChapterBodyNum 扫描 chapters/ 目录，返回**磁盘上已有正文**的最大章号；
// 一个正稿都没有时返回 0。
//
// 「已有正文」= 主线正稿 chapters/NNN.md 非空，或 v4 场景制下该章已有承载非空
// 正文的场景（只有场景、没有 blob 的章同样算占用）。分支章 NNNa.md 不计入
// 主线章号口径。目录缺失 = 0（新项目正常态）。
//
// 供补下等「续编章号」场景做磁盘侧基准（G2）：只按大纲 OrderIndex 续编会在
// 大纲被「续写」重排后撞上已有正稿。
func (m *Manager) MaxChapterBodyNum() (int, error) {
	return m.maxChapterNumMatching(m.chapterHasBodyContent)
}

// MaxChapterNum 扫描 chapters/ 目录，返回任意章文件（NNN.md / NNNa.md，含空文件）
// 的最大前导章号；无文件返回 0。目录缺失 = 0（新项目正常态）。
func (m *Manager) MaxChapterNum() (int, error) {
	return m.maxChapterNumMatching(func(int) bool { return true })
}

// maxChapterNumMatching 扫描 chapters/，返回满足 keep 的最大章号。章号有两个来源：
//   - 章文件 NNN[字母].md 的前导章号；
//   - v4 场景制章目录 chapters/NNN/（只有场景、没有 blob 的章同样占用章号——
//     只看 .md 会让「场景承载正文的章」在补下基准里消失，G2 又回到撞车）。
//
// 目录缺失视为空集（新项目正常态），不报错。
func (m *Manager) maxChapterNumMatching(keep func(num int) bool) (int, error) {
	entries, err := os.ReadDir(filepath.Join(m.Dir, "chapters"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	maxNum := 0
	consider := func(num int) {
		if num <= 0 || num <= maxNum || !keep(num) {
			return
		}
		maxNum = num
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			// 纯数字目录名 = 主线章目录（场景制，%03d）；「001a」这类分支目录不算主线章号。
			num := leadDigits(name)
			if num <= 0 || fmt.Sprintf("%03d", num) != name {
				continue
			}
			if m.chapterHasBodyContent(num) {
				consider(num)
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		consider(leadDigits(name))
	}
	return maxNum, nil
}

// chapterHasBodyContent 该章是否已有非空正文（主线 blob 或 v4 场景承载）。
func (m *Manager) chapterHasBodyContent(num int) bool {
	if blob, err := m.ReadChapter(num); err == nil && strings.TrimSpace(blob) != "" {
		return true
	}
	sm := m.SceneManager(num)
	metas, err := sm.List()
	if err != nil {
		return false
	}
	for _, meta := range metas {
		if sc, rerr := sm.Read(meta.ID); rerr == nil && strings.TrimSpace(sc.Content) != "" {
			return true
		}
	}
	return false
}

// leadDigits 取字符串前导十进制数字；没有前导数字返回 0
// （"006-summary.json" → 6，"006a-summary.json" → 6）。
func leadDigits(s string) int {
	n, started := 0, false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
			started = true
			continue
		}
		if started {
			break
		}
		return 0
	}
	return n
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

// writeJSON 序列化后经 fileutil.AtomicWrite 原子落盘（同目录临时文件 +
// RenameWithRetry 覆盖，失败保留旧文件）。perm 传 0o600 与旧私有实现的
// os.CreateTemp 默认权限位一致，收敛后行为不变。
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败 (%s): %w", path, err)
	}
	if err := fileutil.AtomicWrite(path, data, 0o600); err != nil {
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
	//
	// N12 修复：无法迁移的章（文件名解析不出章号 / 读失败）**收集后如实失败**，
	// 绝不静默跳过再照落 v4 标记——标记一落，IsV4() 为真，未迁移的 v3 章从此
	// 从读写视图里消失（ReadChapterAsStitch 优先 v4 视图），用户数据「没了」
	// 却无任何提示。
	chaptersDir := filepath.Join(m.Dir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return m.finalizeV4Migration()
		}
		return err
	}

	var skipped []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// 解析章节号
		name := e.Name()
		var chapterNum int
		if _, err := fmt.Sscanf(name, "%03d.md", &chapterNum); err != nil {
			skipped = append(skipped, name+"（文件名不是 NNN.md 章节格式）")
			continue
		}

		// 读旧内容
		oldPath := filepath.Join(chaptersDir, name)
		content, err := os.ReadFile(oldPath)
		if err != nil {
			skipped = append(skipped, name+"（读取失败: "+err.Error()+"）")
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
			_ = os.WriteFile(backupSummaryPath, summaryData, 0644)
		}
	}

	// 有未迁移章则失败且不落 v4 标记。v3 原文件非破坏保留（未删除），用户
	// 处理掉 listed 文件后重试即可。
	if len(skipped) > 0 {
		return fmt.Errorf("迁移中止：以下 %d 个章节文件无法迁移（未落 v4 标记，处理后可重试）：\n%s",
			len(skipped), strings.Join(skipped, "\n"))
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
// 用于替代 export/search/stats 等模块中 for i:=1;;i++ 的重复模式。
// 上界取磁盘最大章号（MaxChapterNum）：中间缺章（如 1、2、4）不再「缺口即停」
// ——旧实现 ReadChapter 第一个 err 就 break，缺号后的章全部漏扫（导出断尾、
// 体检漏章、总数低估）。缺口/空章/读失败按 continue 跳过续扫，不因一章坏档
// 丢掉全书其余章。
func (m *Manager) ForEachChapter(fn func(chapterNum int, content string) error) error {
	maxNum, err := m.MaxChapterNum()
	if err != nil {
		return err
	}
	for i := 1; i <= maxNum; i++ {
		content, err := m.ReadChapter(i)
		if err != nil || content == "" {
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
// 保持章号升序，写回走原子替换。整表读-改-写全程持 fileMu（审计 P1 IN1-07）。
func (m *Manager) UpsertAnalysisV2(res types.ChapterAnalysisResult) error {
	m.fileMu.Lock()
	defer m.fileMu.Unlock()
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
// 整表读-改-写全程持 fileMu（审计 P1 IN1-07，与 UpsertAnalysisV2 同口径）。
func (m *Manager) syncRewriteIndex(v types.RewriteVersion) error {
	m.fileMu.Lock()
	defer m.fileMu.Unlock()
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
