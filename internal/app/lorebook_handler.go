package app

import (
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/types"
)

// ── Lorebook ──────────────────────────────────────────────────

// GetLorebookEntries 获取所有词条
func (a *writingState) GetLorebookEntries() map[string]interface{} {
	pm := a.getPM()
	if pm == nil {
		return nil
	}
	lf, err := pm.ReadLorebook()
	if err != nil {
		return nil
	}
	return map[string]interface{}{
		"entries": lf.Entries,
	}
}

// SaveLorebookEntry 保存/更新一个词条
func (a *writingState) SaveLorebookEntry(entryJSON string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	lf, err := pm.ReadLorebook()
	if err != nil {
		return err
	}
	var entry types.LorebookEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return fmt.Errorf("解析词条失败: %w", err)
	}
	if entry.Key == "" || entry.Content == "" {
		return fmt.Errorf("词条 key 和 content 不能为空")
	}

	// 更新或追加
	found := false
	for i := range lf.Entries {
		if lf.Entries[i].Key == entry.Key {
			lf.Entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		lf.Entries = append(lf.Entries, entry)
	}
	return pm.WriteLorebook(lf)
}

// DeleteLorebookEntry 删除词条
func (a *writingState) DeleteLorebookEntry(key string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	lf, err := pm.ReadLorebook()
	if err != nil {
		return err
	}
	filtered := make([]types.LorebookEntry, 0, len(lf.Entries))
	for _, e := range lf.Entries {
		if e.Key != key {
			filtered = append(filtered, e)
		}
	}
	lf.Entries = filtered
	return pm.WriteLorebook(lf)
}
