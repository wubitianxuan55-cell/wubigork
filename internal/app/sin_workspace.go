package app

// ── 原罪工作空间（书架绑定）──2026-11-15 升格刀
//
// 模型（用户拍板）：一书一故事，书下挂多个会话（像办公工作区+会话）。
//   - 书 = 书架小说项目（含 project.json 且在书架目录内，fail-closed 校验）；
//   - 会话 = chat_topics mode=sin（存量）/ 引擎会话（升格后新会话）；
//   - 创作上下文（选角/画风/便签/大纲/记忆）挂书：键 = 书路径 slug；
//     绑定旧故事时把 storyID 键的数据一次性迁移到书键（失败保留原文件）。
//
// 隔离纪律：本文件是 sin 自有存储，不写办公工作区、不碰办公绑定面。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/fileutil"
)

const sinWorkspaceVersion = 1

var sinWorkspaceMu sync.Mutex

// sinBookSlug 书路径 → 存储键（路径分隔符/盘符归一 '-'，单段文件名安全）。
// sin 自有实现（隔离纪律：不改 memory 包私有 slugify）。
func sinBookSlug(absPath string) string {
	r := strings.NewReplacer(string(os.PathSeparator), "-", "/", "-", "\\", "-", ":", "-")
	return r.Replace(absPath)
}

// sinWorkspacesPath 绑定表文件（sin/workspaces.json）。
func sinWorkspacesPath() string {
	return filepath.Join(sinRoot(), "workspaces.json")
}

// sinWorkspaceBinding 一条会话→书绑定。
type sinWorkspaceBinding struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

// sinWorkspaceDoc 绑定表：会话 id → 书。
type sinWorkspaceDoc struct {
	Version  int                            `json:"version"`
	Bindings map[string]sinWorkspaceBinding `json:"bindings"`
}

func emptySinWorkspaceDoc() sinWorkspaceDoc {
	return sinWorkspaceDoc{Version: sinWorkspaceVersion, Bindings: map[string]sinWorkspaceBinding{}}
}

// loadSinWorkspaces 读取绑定表（缺失/损坏 → 空表；损坏留档，与 cast.json 同口径）。
func loadSinWorkspaces() sinWorkspaceDoc {
	doc := emptySinWorkspaceDoc()
	raw, err := os.ReadFile(sinWorkspacesPath())
	if err != nil {
		return doc
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		stamp := sinWorkspacesPath() + ".corrupt-" + time.Now().Format("20060102-150405")
		if renameErr := os.Rename(sinWorkspacesPath(), stamp); renameErr == nil {
			slog.Warn("原罪工作空间绑定表损坏，已留档后按空表继续", "archived", stamp, "error", err)
		} else {
			slog.Warn("原罪工作空间绑定表损坏，按空表继续（留档失败）", "path", sinWorkspacesPath(), "error", err)
		}
		return emptySinWorkspaceDoc()
	}
	if doc.Bindings == nil {
		doc.Bindings = map[string]sinWorkspaceBinding{}
	}
	doc.Version = sinWorkspaceVersion
	return doc
}

// saveSinWorkspaces 原子写回绑定表。
func saveSinWorkspaces(doc sinWorkspaceDoc) error {
	if err := os.MkdirAll(sinRoot(), 0o755); err != nil {
		return fmt.Errorf("创建原罪数据目录失败: %w", err)
	}
	doc.Version = sinWorkspaceVersion
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := fileutil.AtomicWrite(sinWorkspacesPath(), b, 0o644); err != nil {
		return fmt.Errorf("保存工作空间绑定失败: %w", err)
	}
	return nil
}

// SinWorkspaceInfo 绑定视图（前端书架栏消费）。
type SinWorkspaceInfo struct {
	Bound bool   `json:"bound"`
	Path  string `json:"path"`
	Title string `json:"title"`
}

// sinValidateBookPath 书路径校验（fail-closed）：绝对路径、含 project.json、
// 必须在书架目录内（与 DeleteProject 同款 containment 判据）。
// 返回清洗后的绝对路径与项目标题（project.json 解析不到时回退目录名）。
func (a *App) sinValidateBookPath(path string) (string, string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", "", fmt.Errorf("书路径为空")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", "", fmt.Errorf("书路径解析失败: %w", err)
	}
	abs = filepath.Clean(abs)
	metaPath := filepath.Join(abs, "project.json")
	if _, err := os.Stat(metaPath); err != nil {
		return "", "", fmt.Errorf("不是有效的小说项目（缺 project.json）: %s", filepath.ToSlash(abs))
	}
	novelsDir := ""
	if a.cfg != nil {
		novelsDir = a.cfg.NovelsDir
	}
	if novelsDir != "" {
		absNovels, err := filepath.Abs(novelsDir)
		if err == nil && !strings.HasPrefix(abs, absNovels+string(filepath.Separator)) {
			return "", "", fmt.Errorf("书必须位于书架目录内: %s", filepath.ToSlash(absNovels))
		}
	}
	title := filepath.Base(abs)
	if meta, err := loadProjectMeta(metaPath); err == nil && meta != nil && strings.TrimSpace(meta.Title) != "" {
		title = strings.TrimSpace(meta.Title)
	}
	return abs, title, nil
}

// SinWorkspaceGet 读取某会话的书绑定。
func (a *App) SinWorkspaceGet(topicID string) (SinWorkspaceInfo, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return SinWorkspaceInfo{}, err
	}
	sinWorkspaceMu.Lock()
	defer sinWorkspaceMu.Unlock()
	b, ok := loadSinWorkspaces().Bindings[topicID]
	if !ok {
		return SinWorkspaceInfo{Bound: false}, nil
	}
	return SinWorkspaceInfo{Bound: true, Path: b.Path, Title: b.Title}, nil
}

// SinWorkspaceBindings 全量绑定清单（书架栏分组用；板块级视图不做话题守卫）。
func (a *App) SinWorkspaceBindings() []SinWorkspaceBindingView {
	sinWorkspaceMu.Lock()
	defer sinWorkspaceMu.Unlock()
	doc := loadSinWorkspaces()
	out := make([]SinWorkspaceBindingView, 0, len(doc.Bindings))
	for topicID, b := range doc.Bindings {
		out = append(out, SinWorkspaceBindingView{TopicID: topicID, Path: b.Path, Title: b.Title})
	}
	return out
}

// SinWorkspaceBindingView 绑定清单条目。
type SinWorkspaceBindingView struct {
	TopicID string `json:"topicId"`
	Path    string `json:"path"`
	Title   string `json:"title"`
}

// SinWorkspaceSet 绑定会话到书（校验 + 保存 + 存量键迁移）。
// 迁移：notes 文件与 cast/styles 键从 storyID 复制到书 slug 键——已有目标键
// 时不覆盖（书键是权威，多会话共享）；原键保留（解绑回退不丢数据）。
func (a *App) SinWorkspaceSet(topicID string, projectPath string) (SinWorkspaceInfo, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return SinWorkspaceInfo{}, err
	}
	abs, title, err := a.sinValidateBookPath(projectPath)
	if err != nil {
		return SinWorkspaceInfo{}, err
	}
	slug := sinBookSlug(abs)
	sinWorkspaceMu.Lock()
	doc := loadSinWorkspaces()
	doc.Bindings[topicID] = sinWorkspaceBinding{Path: abs, Title: title}
	if err := saveSinWorkspaces(doc); err != nil {
		sinWorkspaceMu.Unlock()
		return SinWorkspaceInfo{}, err
	}
	sinWorkspaceMu.Unlock()

	a.sinMigrateStoryKeysToBook(topicID, slug)
	return SinWorkspaceInfo{Bound: true, Path: abs, Title: title}, nil
}

// SinWorkspaceUnbind 解除会话绑定（书键数据保留，其他会话不受影响）。
func (a *App) SinWorkspaceUnbind(topicID string) error {
	if err := a.sinTopicGuard(topicID); err != nil {
		return err
	}
	sinWorkspaceMu.Lock()
	defer sinWorkspaceMu.Unlock()
	doc := loadSinWorkspaces()
	delete(doc.Bindings, topicID)
	return saveSinWorkspaces(doc)
}

// sinMigrateStoryKeysToBook 存量键迁移：notes 文件与 cast/styles 键
// storyID → 书 slug（复制不移动、目标已存在不覆盖、失败只告警不阻断绑定）。
func (a *App) sinMigrateStoryKeysToBook(topicID, slug string) {
	if topicID == "" || slug == "" || topicID == slug {
		return
	}
	// 1) 便签/大纲文件
	if src, err := sinNotesPath(topicID); err == nil {
		if _, statErr := os.Stat(src); statErr == nil {
			dst, err := sinNotesPath(slug)
			if err == nil {
				if _, dstErr := os.Stat(dst); os.IsNotExist(dstErr) {
					raw, readErr := os.ReadFile(src)
					if readErr == nil {
						if writeErr := fileutil.AtomicWrite(dst, raw, 0o644); writeErr != nil {
							slog.Warn("原罪底稿迁移写入失败（原文件保留）", "src", src, "dst", dst, "error", writeErr)
						}
					} else {
						slog.Warn("原罪底稿迁移读取失败（原文件保留）", "src", src, "error", readErr)
					}
				}
			}
		}
	}
	// 2) 选角/画风键
	sinCastMu.Lock()
	doc := loadSinCast()
	migrated := false
	if ids, ok := doc.Casts[topicID]; ok && len(ids) > 0 {
		if _, exists := doc.Casts[slug]; !exists {
			doc.Casts[slug] = ids
			migrated = true
		}
	}
	if style, ok := doc.Styles[topicID]; ok && style != "" {
		if _, exists := doc.Styles[slug]; !exists {
			doc.Styles[slug] = style
			migrated = true
		}
	}
	if migrated {
		if err := saveSinCast(doc); err != nil {
			slog.Warn("原罪选角/画风键迁移保存失败（原键保留）", "error", err)
		}
	}
	sinCastMu.Unlock()
}

// sinEngineSessionDir 书工作区的引擎会话目录（升格管线会话真源：
// 书/.gaea/sessions/play/，与办公 WorkspaceSessionDir 同机制不同根）。
func sinEngineSessionDir(bookPath string) string {
	return gaeaConfig.WorkspaceSessionDir(bookPath, "play")
}
