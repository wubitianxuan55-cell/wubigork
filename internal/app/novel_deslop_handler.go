package app

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/novelstyle"
)

// ── 刀 5 续 · 手动「一键去味」 ─────────────────────────────────────────
//
// 生成时 story-deslop 已自动去味（见 streamCreateChapter）；但用户手改/手写的
// 章节不会自动过。本绑定把 novelstyle.DeSlopRewrite 暴露成可手动的确定性
// 一键去味：对任意已有章节（v4 逐场景 / v3 整章）重写命中 AI 词并落盘。
// 纯确定性、零 LLM、零网络；返回改动统计供前端反馈。

func (a *writingState) DeSlopChapterAiTaste(chapterNum int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号非法")
	}

	totalChanges := 0
	avantScore := 0
	apresScore := 0

	if pm.IsV4() {
		sm := pm.SceneManager(chapterNum)
		metas, err := sm.List()
		if err != nil {
			return nil, err
		}
		for _, meta := range metas {
			sc, rerr := sm.Read(meta.ID)
			if rerr != nil || strings.TrimSpace(sc.Content) == "" {
				continue
			}
			b, _ := novelstyle.ScoreTextNoRef(sc.Content)
			rw, rep, derr := novelstyle.DeSlopRewrite(sc.Content, b)
			if derr == nil && rep != nil && rep.AfterScore < rep.BeforeScore && rw != "" {
				sc.Content = rw
				_ = sm.Write(sc)
				totalChanges += len(rep.Changes)
				avantScore += rep.BeforeScore
				apresScore += rep.AfterScore
			}
		}
	} else {
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, fmt.Errorf("读取章节失败: %w", err)
		}
		b, _ := novelstyle.ScoreTextNoRef(content)
		rw, rep, derr := novelstyle.DeSlopRewrite(content, b)
		if derr == nil && rep != nil && rep.AfterScore < rep.BeforeScore && rw != "" {
			if werr := pm.WriteChapter(chapterNum, rw); werr != nil {
				return nil, fmt.Errorf("保存章节失败: %w", werr)
			}
			totalChanges += len(rep.Changes)
			avantScore = rep.BeforeScore
			apresScore = rep.AfterScore
		}
	}

	return map[string]interface{}{
		"chapterNum":  chapterNum,
		"changes":     totalChanges,
		"beforeScore": avantScore,
		"afterScore":  apresScore,
		"done":        totalChanges > 0,
	}, nil
}
