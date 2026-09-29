package project

// story_spine.json 存取（长篇刀3）：整书层骨架的读写与确定性校验。
// 模式与 plan_store 同款（loadJSON/writeJSON 原子写、目录缺失/文件缺失=空表）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// StorySpinePath 故事脊椎路径 story_spine.json（项目根，与 outline.json 平级）。
func (m *Manager) StorySpinePath() string {
	return filepath.Join(m.Dir, "story_spine.json")
}

// ReadStorySpine 读故事脊椎；文件缺失返回空骨架（Version 归一），不算错。
func (m *Manager) ReadStorySpine() (*types.StorySpine, error) {
	sp, err := loadJSON[types.StorySpine](m.StorySpinePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &types.StorySpine{Version: types.StorySpineVersion}, nil
		}
		return nil, err
	}
	if sp == nil {
		return &types.StorySpine{Version: types.StorySpineVersion}, nil
	}
	if sp.Version <= 0 {
		sp.Version = types.StorySpineVersion
	}
	return sp, nil
}

// WriteStorySpine 写故事脊椎（原子写）。校验先行（ValidateStorySpine）。
func (m *Manager) WriteStorySpine(sp *types.StorySpine) error {
	if sp == nil {
		return fmt.Errorf("故事骨架为空")
	}
	if err := ValidateStorySpine(sp); err != nil {
		return err
	}
	return writeJSON(m.StorySpinePath(), sp)
}

// ValidateStorySpine 确定性校验：章号非负、状态/枚举合法、必填标识非空。
// 校验失败整单拒绝（不静默吞——与 SaveSceneMeta 的 status 纪律一致）。
func ValidateStorySpine(sp *types.StorySpine) error {
	if sp.Version <= 0 {
		sp.Version = types.StorySpineVersion
	}
	seenArc := map[string]bool{}
	for i := range sp.Arcs {
		a := &sp.Arcs[i]
		if strings.TrimSpace(a.Name) == "" {
			return fmt.Errorf("弧线 %d 缺角色名", i+1)
		}
		if seenArc[a.Name] {
			return fmt.Errorf("弧线角色名重复：%s", a.Name)
		}
		seenArc[a.Name] = true
		for j := range a.Beats {
			b := &a.Beats[j]
			if b.Chapter <= 0 {
				return fmt.Errorf("弧线「%s」水位点 %d 章号非法（%d，需 >0）", a.Name, j+1, b.Chapter)
			}
			switch b.Stage {
			case "", types.ArcStageSetup, types.ArcStageProvoke, types.ArcStageEscalate, types.ArcStageProof:
			default:
				return fmt.Errorf("弧线「%s」水位点 %d 阶段非法 %q（setup/provoke/escalate/proof）", a.Name, j+1, b.Stage)
			}
		}
	}
	seenBeat := map[string]bool{}
	for i := range sp.Beats {
		b := &sp.Beats[i]
		if strings.TrimSpace(b.ID) == "" {
			return fmt.Errorf("节拍 %d 缺 ID", i+1)
		}
		if seenBeat[b.ID] {
			return fmt.Errorf("节拍 ID 重复：%s", b.ID)
		}
		seenBeat[b.ID] = true
		if b.Chapter < 0 {
			return fmt.Errorf("节拍「%s」章号非法（%d，需 ≥0；0=未绑定）", b.Name, b.Chapter)
		}
		switch b.Status {
		case "", types.BeatStatusPlanned, types.BeatStatusHit:
		default:
			return fmt.Errorf("节拍「%s」状态非法 %q（planned/hit）", b.Name, b.Status)
		}
	}
	seenThread := map[string]bool{}
	for i := range sp.Threads {
		t := &sp.Threads[i]
		if strings.TrimSpace(t.ID) == "" {
			return fmt.Errorf("支线 %d 缺 ID", i+1)
		}
		if seenThread[t.ID] {
			return fmt.Errorf("支线 ID 重复：%s", t.ID)
		}
		seenThread[t.ID] = true
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("支线「%s」缺名称", t.ID)
		}
		if t.OpenChapter <= 0 {
			return fmt.Errorf("支线「%s」开启章号非法（%d，需 >0）", t.Name, t.OpenChapter)
		}
		switch t.Status {
		case types.ThreadStatusActive, types.ThreadStatusParked, types.ThreadStatusClosed, types.ThreadStatusAbandoned:
		default:
			return fmt.Errorf("支线「%s」状态非法 %q（active/parked/closed/abandoned）", t.Name, t.Status)
		}
		switch t.MICEType {
		case "", types.MICEQuery, types.MICECharacter, types.MICEEvent, types.MICEPlace, types.MICEThing:
		default:
			return fmt.Errorf("支线「%s」MICE 类型非法 %q", t.Name, t.MICEType)
		}
	}
	seenQ := map[string]bool{}
	for i := range sp.OpenQuestions {
		q := &sp.OpenQuestions[i]
		if strings.TrimSpace(q.ID) == "" {
			return fmt.Errorf("未解问题 %d 缺 ID", i+1)
		}
		if seenQ[q.ID] {
			return fmt.Errorf("未解问题 ID 重复：%s", q.ID)
		}
		seenQ[q.ID] = true
		if strings.TrimSpace(q.Question) == "" {
			return fmt.Errorf("未解问题「%s」缺内容", q.ID)
		}
		if q.RaisedChapter <= 0 {
			return fmt.Errorf("未解问题「%s」提出章号非法（%d，需 >0）", q.ID, q.RaisedChapter)
		}
		switch q.Status {
		case types.QuestionOpen, types.QuestionAnswered:
		default:
			return fmt.Errorf("未解问题「%s」状态非法 %q（open/answered）", q.ID, q.Status)
		}
	}
	return nil
}
