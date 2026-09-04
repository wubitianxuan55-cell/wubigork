package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/narrative"
)

// ── 刀 6 · 叙事状态（narrative 审批制结算）────────────────────────────
//
// AI 只能产出 StatePatch（建议），无法直接改状态；作者审批后经
// narrative.AuthorizeAndSettle 才写入 append-only 账本并重建 state.json。
// 这里暴露三个绑定：读状态 / 评估（构造 patch 供前端审）/ 审批结算。

func (a *writingState) narrativeJournal() (*narrative.Journal, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	return narrative.Open(pm.Dir)
}

// GetNovelState 返回当前叙事状态账本快照（Version + Entities）。
func (a *writingState) GetNovelState() (map[string]interface{}, error) {
	j, err := a.narrativeJournal()
	if err != nil {
		return nil, err
	}
	snap, err := j.Snapshot()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"version":  snap.Version,
		"entities": snap.Entities,
	}, nil
}

// BuildNovelStatePatch 从章节摘要构造一个状态补丁（供前端审阅/审批）。
// 确定性：仅把出场角色登记为 alive + 自身知情的实体状态，作为审批基线。
// 产出的 patch 必须经 SettleNovelState(approved=true) 才生效。
func (a *writingState) BuildNovelStatePatch(chapterNum int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	sum, err := pm.ReadChapterSummary(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节摘要失败: %w", err)
	}
	var upserts []narrative.EntityState
	if sum != nil {
		for _, name := range sum.CharactersAppeared {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			upserts = append(upserts, narrative.EntityState{
				ID:      "char:" + name,
				Name:    name,
				Type:    "character",
				Status:  "alive",
				KnownBy: []string{name},
			})
		}
	}
	patch := narrative.StatePatch{
		ID:         fmt.Sprintf("ch%03d-ai", chapterNum),
		Chapter:    chapterNum,
		Reason:     "AI 按章节摘要建议的叙事状态基线",
		Upserts:    upserts,
		ProposedBy: "ai",
	}
	if err := narrative.ValidateStatePatch(patch); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"patch":  patch,
		"amount": len(upserts),
	}, nil
}

// SettleNovelState 作者审批结算：approved=false 不入账本（AI 建议不生效），
// approved=true 才写 append-only 账本并重建 state.json。返回结算后快照。
func (a *writingState) SettleNovelState(patchJSON string, approved bool) (map[string]interface{}, error) {
	j, err := a.narrativeJournal()
	if err != nil {
		return nil, err
	}
	var patch narrative.StatePatch
	if err := json.Unmarshal([]byte(patchJSON), &patch); err != nil {
		return nil, fmt.Errorf("解析状态补丁失败: %w", err)
	}
	if err := narrative.ValidateStatePatch(patch); err != nil {
		return nil, err
	}
	snap, err := narrative.AuthorizeAndSettle(j, patch, approved)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"approved": approved,
		"version":  snap.Version,
		"entities": snap.Entities,
	}, nil
}
