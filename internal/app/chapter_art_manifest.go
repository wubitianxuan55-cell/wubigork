package app

// ── 章节插图清单（画室素材库雏形 · T1 后端）──
//
// 存 <项目根>/.gaea/play/art/chapter-art.json（play 红线，与 exports 并列的 art 目录）。
// 设计目的：GenerateSceneIllustration 落盘登记后追加一条；未来画室/章节 UI 经既有
// 读文件类绑定展示「本章历史配图/重生成变体/设为封面素材」，无需新增格式。
// 容错纪律：清单是辅助视图——文件损坏/缺失只返回空列表，不阻断生成主流程。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/project"
)

const chapterArtVersion = 1

// 上限为可变（测试可收紧覆盖折叠逻辑）。
var (
	chapterArtMaxEntries = 200 // 全项目清单上限（折叠最旧）
	chapterArtPerChapter = 12  // 单章保留上限（重生成变体窗口）
)

// chapterArtEntry 单条章节配图登记。
type chapterArtEntry struct {
	Chapter   int    `json:"chapter"`
	AssetID   string `json:"asset_id,omitempty"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

// chapterArtManifest 项目章节插图清单。
type chapterArtManifest struct {
	Version int               `json:"version"`
	Entries []chapterArtEntry `json:"entries"`
}

// chapterArtMu 清单写锁（与 ledger 锁独立；进程内单写者即可）。
var chapterArtMu sync.Mutex

// chapterArtManifestPath 清单文件路径。
func chapterArtManifestPath(pm *project.Manager) string {
	return filepath.Join(pm.Dir, ".gaea", "play", "art", "chapter-art.json")
}

// loadChapterArtManifest 读清单（损坏/缺失 → 空清单，不报错）。
func loadChapterArtManifest(pm *project.Manager) chapterArtManifest {
	m := chapterArtManifest{Version: chapterArtVersion}
	raw, err := os.ReadFile(chapterArtManifestPath(pm))
	if err != nil {
		return m
	}
	_ = json.Unmarshal(raw, &m) // 坏 JSON 保持空清单（辅助视图容错）
	if m.Version != chapterArtVersion {
		m.Version = chapterArtVersion
	}
	return m
}

// saveChapterArtManifest 原子写清单（临时文件 + 替换；Windows 下先移除旧文件）。
func saveChapterArtManifest(pm *project.Manager, m chapterArtManifest) error {
	p := chapterArtManifestPath(pm)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("创建章节插图清单目录失败: %w", err)
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("章节插图清单序列化失败: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), "chapter-art-*.tmp")
	if err != nil {
		return fmt.Errorf("创建清单临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("写入清单失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("关闭清单失败: %w", err)
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpName)
		return fmt.Errorf("移除旧清单失败: %w", err)
	}
	if err := os.Rename(tmpName, p); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("替换清单失败: %w", err)
	}
	return nil
}

// appendChapterArt 追加一条章节配图并折叠（单章与全项目双上限；失败只返回 error，
// 由调用方 warn，不阻断生成）。
func appendChapterArt(pm *project.Manager, chapter int, assetID, path string) error {
	chapterArtMu.Lock()
	defer chapterArtMu.Unlock()

	m := loadChapterArtManifest(pm)
	m.Entries = append(m.Entries, chapterArtEntry{
		Chapter:   chapter,
		AssetID:   assetID,
		Path:      path,
		CreatedAt: time.Now().Format(time.RFC3339),
	})
	// 单章只留最近 chapterArtPerChapter 条。
	perChapter := 0
	kept := make([]chapterArtEntry, 0, len(m.Entries))
	for i := len(m.Entries) - 1; i >= 0; i-- {
		e := m.Entries[i]
		if e.Chapter == chapter {
			perChapter++
			if perChapter > chapterArtPerChapter {
				continue
			}
		}
		kept = append(kept, e)
	}
	// 倒序还原为追加序后，再做全项目上限折叠（保留最近）。
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}
	if len(kept) > chapterArtMaxEntries {
		kept = kept[len(kept)-chapterArtMaxEntries:]
	}
	m.Entries = kept
	return saveChapterArtManifest(pm, m)
}

// listChapterArt 返回某章配图（最新在前）；chapter<=0 返回全部（最新在前）。
func listChapterArt(pm *project.Manager, chapter int) []chapterArtEntry {
	m := loadChapterArtManifest(pm)
	var out []chapterArtEntry
	for i := len(m.Entries) - 1; i >= 0; i-- {
		e := m.Entries[i]
		if chapter > 0 && e.Chapter != chapter {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ChapterArtList 章节配图清单读取（T1 绑定入口；未打开项目/无清单 = 空列表）。
func (w *writingState) ChapterArtList(chapter int) []chapterArtEntry {
	pm := w.getPM()
	if pm == nil {
		return nil
	}
	return listChapterArt(pm, chapter)
}
