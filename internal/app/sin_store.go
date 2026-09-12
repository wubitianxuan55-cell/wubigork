package app

// ── 原罪板块自有存储（硬隔离落点）──
//
// 边界（用户口径 2026-09-12「原罪需要与办公板块硬隔离」）：
//   - 原罪的会话/消息：统一聊天存储（chat.db，mode=sin）——不在办公工作区，也不进办公记忆；
//   - 原罪的插图产物：<用户配置目录>/gaea/sin/art（本文件 sinArtDir）——不写办公
//     工作区 .gaea/、不依赖办公的 ImageSaveDir 与小说项目目录；
//   - 原罪的角色选择：<用户配置目录>/gaea/sin/cast.json（本文件）——按故事 id 存
//     角色库角色 id 列表，作为「使用角色库内容」的桥，但角色库本身是跨板块共享
//     资产层（不是办公数据面），故不违反硬隔离。
//
// 容错纪律：cast.json 损坏/缺失只当空表（辅助配置，不阻断故事创作）；
// 保存原子（临时文件 + rename），进程内单写者由 sinCastMu 串行化。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/characterlib"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

const (
	// sinCastVersion cast.json 版本（未来结构变更时按版本迁移）。
	sinCastVersion = 1
	// sinCastMaxPerStory 单个故事最多带入的角色数（防提示词被角色设定吃满）。
	sinCastMaxPerStory = 8
)

var sinCastMu sync.Mutex

// sinRoot 原罪板块数据根（用户配置目录下，与办公工作区硬隔离）。
func sinRoot() string {
	return filepath.Join(gaeaConfig.MemoryUserDir(), "sin")
}

// sinArtDir 原罪插图落盘目录（不存在时生成失败由调用方如实上报）。
func sinArtDir() string {
	return filepath.Join(sinRoot(), "art")
}

// sinCastPath 角色选择配置文件（sin/cast.json）。
func sinCastPath() string {
	return filepath.Join(sinRoot(), "cast.json")
}

// sinCastDoc cast.json 结构：故事 id → 角色库角色 id 列表。
type sinCastDoc struct {
	Version int                 `json:"version"`
	Casts   map[string][]string `json:"casts"`
}

// loadSinCast 读取角色选择表（文件缺失/损坏 → 空表，辅助配置容错）。
func loadSinCast() sinCastDoc {
	doc := sinCastDoc{Version: sinCastVersion, Casts: map[string][]string{}}
	raw, err := os.ReadFile(sinCastPath())
	if err != nil {
		return doc
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		slog.Warn("原罪角色配置损坏，按空表继续", "path", sinCastPath(), "error", err)
		return sinCastDoc{Version: sinCastVersion, Casts: map[string][]string{}}
	}
	if doc.Casts == nil {
		doc.Casts = map[string][]string{}
	}
	doc.Version = sinCastVersion
	return doc
}

// saveSinCast 原子写回角色选择表（临时文件 + rename；失败向上抛）。
func saveSinCast(doc sinCastDoc) error {
	dir := sinRoot()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建原罪数据目录失败: %w", err)
	}
	doc.Version = sinCastVersion
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := sinCastPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("写入角色配置失败: %w", err)
	}
	if err := os.Rename(tmp, sinCastPath()); err != nil {
		return fmt.Errorf("保存角色配置失败: %w", err)
	}
	return nil
}

// SinCastGet 读取某故事已选角色（角色库 id 列表，按选择顺序）。
func (a *App) SinCastGet(topicID string) ([]string, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return nil, err
	}
	sinCastMu.Lock()
	defer sinCastMu.Unlock()
	out := loadSinCast().Casts[topicID]
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// SinCastSet 保存某故事的角色选择，返回**生效清单**：
//   - 只保留角色库真实存在的 id（悬空 id 丢弃，不静默留脏数据）；
//   - 去重、保序、截断到 sinCastMaxPerStory；
//   - 空清单 = 清空该故事的角色。
func (a *App) SinCastSet(topicID string, ids []string) ([]string, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return nil, err
	}
	effective := a.sinResolveCastIDs(ids)
	sinCastMu.Lock()
	defer sinCastMu.Unlock()
	doc := loadSinCast()
	if len(effective) == 0 {
		delete(doc.Casts, topicID)
	} else {
		doc.Casts[topicID] = effective
	}
	if err := saveSinCast(doc); err != nil {
		return nil, err
	}
	return effective, nil
}

// sinResolveCastIDs 过滤/去重/截断角色 id（角色库未初始化或 id 不存在 → 丢弃）。
func (a *App) sinResolveCastIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		if a.charLib == nil {
			break
		}
		if c, err := a.charLib.Get(id); err != nil || c == nil {
			continue // 悬空 id：丢弃（不写进故事提示词）
		}
		seen[id] = true
		out = append(out, id)
		if len(out) >= sinCastMaxPerStory {
			break
		}
	}
	return out
}

// sinCastCharacters 取某故事的角色详情（用于提示词注入；按选择顺序，
// 悬空 id 自动跳过）。角色库未初始化 = 空（故事照常写，只是没有角色锚）。
func (a *App) sinCastCharacters(topicID string) []*characterlib.Character {
	sinCastMu.Lock()
	ids := append([]string(nil), loadSinCast().Casts[topicID]...)
	sinCastMu.Unlock()
	out := make([]*characterlib.Character, 0, len(ids))
	if a.charLib == nil {
		return out
	}
	for _, id := range ids {
		c, err := a.charLib.Get(id)
		if err != nil || c == nil {
			continue
		}
		out = append(out, c)
	}
	return out
}

// SinNotesView 右栏「设定/大纲」面板的只读视图（sin/notes/<故事 id>.json 的
// 便签与大纲两份工作底稿；写作时由 sin_notes/sin_outline 工具读写）。
type SinNotesView struct {
	Notes   []string `json:"notes"`
	Outline string   `json:"outline"`
}

// SinNotesGet 读取某故事的便签（设定集）与大纲，供前端右栏面板展示。
// 文件缺失/损坏 = 空文档（与工具侧 loadSinNotes 同口径，辅助数据不阻断）。
func (a *App) SinNotesGet(topicID string) (SinNotesView, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return SinNotesView{}, err
	}
	path, err := sinNotesPath(topicID)
	if err != nil {
		return SinNotesView{}, err
	}
	sinNotesMu.Lock()
	defer sinNotesMu.Unlock()
	doc := loadSinNotes(path)
	if doc.Notes == nil {
		doc.Notes = []string{}
	}
	return SinNotesView{Notes: doc.Notes, Outline: doc.Outline}, nil
}
