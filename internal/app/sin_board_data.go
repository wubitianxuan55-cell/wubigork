package app

// ── 原罪板块·板载数据绑定（升格刀6a）──────────────────────────────────
//
// 右栏创作面板（五卡）在升格引擎时代按**书 slug 键**读写创作上下文；legacy
// 同名绑定（SinCastGet/SinNotesGet/SinStoryStyleGet…）带 sinTopicGuard 守卫
// （key 必须是 chat.db 里的 sin 话题）——书 slug 不是话题，直接复用会被
// fail-closed 拒绝。本文件提供无话题守卫的板载变体：键直读/直写同一存储
// （cast.json/notes/<key>.json），存储核完全复用（loadSinCast/saveSinNotes/…）。
//
// 隔离纪律：这些绑定只写 sin 自有数据面；fail-closed 面收窄为「键字符集校验
// （sinNotesKeyOK）+ 角色库悬空 id 过滤」，与 legacy 同强度（只是守卫对象从
// 「话题存在」换成「键可安全作文件名」——slug 由 SinWorkspaceSet 校验产生）。

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/gaea/gaea/internal/characterlib"
)

// SinBoardCastGet 读取书键的已选角色（无话题守卫版）。
func (a *App) SinBoardCastGet(key string) ([]string, error) {
	if !sinNotesKeyOK(key) {
		return nil, fmt.Errorf("存储键非法: %q", key)
	}
	sinCastMu.Lock()
	defer sinCastMu.Unlock()
	out := loadSinCast().Casts[key]
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// SinBoardCastSet 保存书键的角色选择（同 SinCastSet 语义：悬空 id 丢弃/去重
// 保序/封顶；空清单=清空）。返回生效清单。
func (a *App) SinBoardCastSet(key string, ids []string) ([]string, error) {
	if !sinNotesKeyOK(key) {
		return nil, fmt.Errorf("存储键非法: %q", key)
	}
	effective := a.sinResolveCastIDs(ids)
	sinCastMu.Lock()
	defer sinCastMu.Unlock()
	doc := loadSinCast()
	if len(effective) == 0 {
		delete(doc.Casts, key)
	} else {
		doc.Casts[key] = effective
	}
	if err := saveSinCast(doc); err != nil {
		return nil, err
	}
	return effective, nil
}

// SinBoardNotesGet 读取书键的便签+大纲（无话题守卫版；缺失/损坏=空文档）。
func (a *App) SinBoardNotesGet(key string) (SinNotesView, error) {
	if !sinNotesKeyOK(key) {
		return SinNotesView{}, fmt.Errorf("存储键非法: %q", key)
	}
	path, err := sinNotesPath(key)
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

// SinBoardNotesSave 面板编辑保存（无话题守卫版；冲突/限长口径与 SinNotesSave
// 逐字一致——锁内基线比对，冲突带 sinConflictPrefix 前缀）。
func (a *App) SinBoardNotesSave(key string, baseline string, outline string, notes string, force bool) (SinNotesView, error) {
	if !sinNotesKeyOK(key) {
		return SinNotesView{}, fmt.Errorf("存储键非法: %q", key)
	}
	var newNotes []string
	if err := json.Unmarshal([]byte(notes), &newNotes); err != nil {
		return SinNotesView{}, fmt.Errorf("便签列表格式错误: %w", err)
	}
	if len(newNotes) > sinNotesMaxItems {
		return SinNotesView{}, fmt.Errorf("便签条数超上限（%d > %d）", len(newNotes), sinNotesMaxItems)
	}
	cleaned := make([]string, 0, len(newNotes))
	for _, n := range newNotes {
		t := truncateRunes(n, sinNoteMaxRunes)
		if trimmed := t; strings.TrimSpace(trimmed) == "" {
			continue
		}
		cleaned = append(cleaned, t)
	}
	newNotes = cleaned
	newOutline := truncateRunes(strings.TrimSpace(outline), sinOutlineMaxRunes)

	var base sinNotesDoc
	if err := json.Unmarshal([]byte(baseline), &base); err != nil {
		return SinNotesView{}, fmt.Errorf("编辑基线格式错误: %w", err)
	}
	path, err := sinNotesPath(key)
	if err != nil {
		return SinNotesView{}, err
	}
	sinNotesMu.Lock()
	defer sinNotesMu.Unlock()
	current := loadSinNotes(path)
	if !force && (current.Outline != base.Outline || !slices.Equal(current.Notes, base.Notes)) {
		return SinNotesView{}, fmt.Errorf("%s便签/大纲已被其他端更新，请确认后再保存", sinConflictPrefix)
	}
	next := sinNotesDoc{Version: sinNotesVersion, Notes: newNotes, Outline: newOutline}
	if err := saveSinNotes(path, next); err != nil {
		return SinNotesView{}, err
	}
	return SinNotesView{Notes: next.Notes, Outline: next.Outline}, nil
}

// SinBoardStyleGet 读取书键的统一插图画风（无话题守卫版）。
func (a *App) SinBoardStyleGet(key string) (string, error) {
	if !sinNotesKeyOK(key) {
		return "", fmt.Errorf("存储键非法: %q", key)
	}
	sinCastMu.Lock()
	defer sinCastMu.Unlock()
	return loadSinCast().Styles[key], nil
}

// SinBoardStyleSet 设置/清空书键的统一插图画风（返回生效值）。
func (a *App) SinBoardStyleSet(key string, style string) (string, error) {
	if !sinNotesKeyOK(key) {
		return "", fmt.Errorf("存储键非法: %q", key)
	}
	style = truncateRunes(strings.TrimSpace(style), sinStoryStyleMaxRunes)
	sinCastMu.Lock()
	defer sinCastMu.Unlock()
	doc := loadSinCast()
	if style == "" {
		delete(doc.Styles, key)
	} else {
		doc.Styles[key] = style
	}
	if err := saveSinCast(doc); err != nil {
		return "", err
	}
	return style, nil
}

// sinBoardCastCharacters 书键角色详情（提示词注入与 legacy sinCastCharacters
// 同构——键不同而已；引擎路径已在用）。
func (a *App) sinBoardCastCharacters(key string) []*characterlib.Character {
	sinCastMu.Lock()
	ids := append([]string(nil), loadSinCast().Casts[key]...)
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
