package project

// project_stores.go — 逐章派生存储与扩展域（批 28 IN1-01 文件拆分，自
// project.go 原位搬移，零逻辑改动）：SceneManager/SnapshotStore 子存储、
// ReadChapterAsStitch/ForEachChapter 章节遍历、AnalysisV2/章节记忆
// StoryMemory、重写版本 RewriteVersion 族（含索引同步）、章节批注
// Annotation。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/scene"
	"github.com/gaea/gaea/internal/snapshot"
	"github.com/gaea/gaea/internal/types"
)

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
