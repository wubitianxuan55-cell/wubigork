// Package characterlib — 副本→库显式回写的预览与逐字段确认覆盖。
// 单向约束仍是默认：非空不覆盖；覆盖只发生在用户看过冲突清单并逐字段勾选的
// 那一次 ImportProjectCharacters 调用里，无后台自动同步。
// 身份/状态字段（RoleType/Arc/Status）任何路径都不进库——关联即快照，
// 项目内弧线以项目为准（docs/gaea-character-domain-survey-2026-09.md）。
package characterlib

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// descField 描述性字段的统一读写器：预览、空补、确认覆盖共用同一张表，
// 字段集与阶段四出口②的空补完全一致（PortraitURL 等媒体字段不在内）。
type descField struct {
	Key    string // 规范键（overwrites JSON 里的字段名）
	Label  string // 中文名（UI 冲突清单直接展示）
	Lib    func(*Character) string
	SetLib func(*Character, string)
	Proj   func(*types.Character) string
}

var descFields = []descField{
	{"gender", "性别", func(c *Character) string { return c.Gender }, func(c *Character, v string) { c.Gender = v }, func(c *types.Character) string { return c.Gender }},
	{"age", "年龄", func(c *Character) string { return c.Age }, func(c *Character, v string) { c.Age = v }, func(c *types.Character) string { return c.Age }},
	{"personality", "性格", func(c *Character) string { return c.Personality }, func(c *Character, v string) { c.Personality = v }, func(c *types.Character) string { return c.Personality }},
	{"background", "背景", func(c *Character) string { return c.Background }, func(c *Character, v string) { c.Background = v }, func(c *types.Character) string { return c.Background }},
	{"appearance", "外貌", func(c *Character) string { return c.Appearance }, func(c *Character, v string) { c.Appearance = v }, func(c *types.Character) string { return c.Appearance }},
	{"figure", "身形", func(c *Character) string { return c.Figure }, func(c *Character, v string) { c.Figure = v }, func(c *types.Character) string { return c.Figure }},
	{"motivation", "动机", func(c *Character) string { return c.Motivation }, func(c *Character, v string) { c.Motivation = v }, func(c *types.Character) string { return c.Motivation }},
	{"notes", "备注", func(c *Character) string { return c.Notes }, func(c *Character, v string) { c.Notes = v }, func(c *types.Character) string { return c.Notes }},
}

// normalizeFieldKey 归一用户提交的字段键（大小写/空白容错）；不在表内返回空串。
func normalizeFieldKey(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	for _, f := range descFields {
		if f.Key == k {
			return k
		}
	}
	return ""
}

// confirmedField 该角色的该字段是否被用户勾选确认覆盖。
func confirmedField(overwrites map[string][]string, charID, key string) bool {
	if len(overwrites) == 0 {
		return false
	}
	for _, f := range overwrites[charID] {
		if normalizeFieldKey(f) == key {
			return true
		}
	}
	return false
}

// ImportFieldConflict 一条非空冲突：库内与副本都有内容且不同，覆盖与否由用户逐条勾选。
type ImportFieldConflict struct {
	CharacterID   string `json:"characterId"`
	CharacterName string `json:"characterName"`
	Field         string `json:"field"`
	FieldLabel    string `json:"fieldLabel"`
	LibraryValue  string `json:"libraryValue"`
	ProjectValue  string `json:"projectValue"`
}

// ImportPreview 回写预览（只读）：import/fill 按既有规则必然发生（ID 未命中新建、
// 空字段补全）；conflicts 默认不覆盖，勾选确认后随 ImportProjectCharacters 的
// overwrites 生效。
type ImportPreview struct {
	Import    int                   `json:"import"`
	Fill      int                   `json:"fill"`
	Conflicts []ImportFieldConflict `json:"conflicts"`
}

// PreviewImport 只读预览：不写库、不改关联，UI 据此渲染逐字段确认清单。
func (s *Store) PreviewImport(chars []types.Character) (*ImportPreview, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	pv := &ImportPreview{Conflicts: []ImportFieldConflict{}}
	for i := range chars {
		ch := &chars[i]
		target, err := s.Get(ch.ID)
		if err != nil {
			return nil, err
		}
		if target == nil {
			pv.Import++
			continue
		}
		filled := false
		for _, f := range descFields {
			lib, proj := f.Lib(target), strings.TrimSpace(f.Proj(ch))
			if proj == "" {
				continue
			}
			if lib == "" {
				filled = true
			} else if lib != proj {
				pv.Conflicts = append(pv.Conflicts, ImportFieldConflict{
					CharacterID:   ch.ID,
					CharacterName: ch.Name,
					Field:         f.Key,
					FieldLabel:    f.Label,
					LibraryValue:  lib,
					ProjectValue:  proj,
				})
			}
		}
		if filled {
			pv.Fill++
		}
	}
	return pv, nil
}
