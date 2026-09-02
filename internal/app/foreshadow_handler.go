package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// ── 伏笔登记表（闭环：登记→埋设→回收）─────────────────────────

// SaveForeshadows 全量写回伏笔登记表 — 手工登记/状态流转/描述编辑/删除的统一入口。
// 前端持有完整列表（AI 条目 + manual_ 手工条目）整体写回；底层 project.writeJSON
// 走临时文件 + rename 原子替换，不会写坏用户数据。
//
// 手工条目不会被 Analyze 的 syncForeshadows 冲掉：那边是按 ID 合并
// （只更新/追加 AI 认识的 ID，manual_ 等它不认识的 ID 原样保留），见
// internal/analysis/analysis.go syncForeshadows。
func (a *writingState) SaveForeshadows(itemsJSON string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	var items []types.Foreshadow
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return fmt.Errorf("解析伏笔列表失败: %w", err)
	}

	validStatus := map[types.ForeshadowStatus]bool{
		types.ForeshadowPlanted:  true,
		types.ForeshadowHinted:   true,
		types.ForeshadowRevealed: true,
	}
	seen := make(map[string]bool, len(items))
	now := time.Now().UnixMilli()
	for i := range items {
		it := &items[i]
		if it.ID == "" {
			// 空 ID 兜底：manual_ 前缀 + 毫秒时间戳，与 AI stable ID
			// （{category}_{chapter}_{hash}）天然不冲突。
			it.ID = fmt.Sprintf("manual_%d", now)
		}
		if !validStatus[it.Status] {
			return fmt.Errorf("伏笔 %s 状态非法: %q（允许 planted/hinted/revealed）", it.ID, it.Status)
		}
		if seen[it.ID] {
			return fmt.Errorf("伏笔 ID 重复: %s", it.ID)
		}
		seen[it.ID] = true
	}
	if items == nil {
		// 全量清空写回时保持 items 为 [] 而非 null，避免读回端出现 null items
		items = []types.Foreshadow{}
	}
	return pm.WriteForeshadows(&types.ForeshadowFile{Items: items})
}
