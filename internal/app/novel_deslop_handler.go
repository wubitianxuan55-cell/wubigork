package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gaeaconfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
)

// ── 刀 5 续 · 手动「一键去味」 ─────────────────────────────────────────
//
// 生成时 story-deslop 已自动去味（见 streamCreateChapter）；但用户手改/手写的
// 章节不会自动过。本绑定把 novelstyle.DeSlopRewrite 暴露成可手动的确定性
// 一键去味：对任意已有章节（v4 逐场景 / v3 整章）重写命中 AI 词并落盘。
// 纯确定性、零 LLM、零网络；返回改动统计供前端反馈。

// novelWordsOnce 保证词表覆盖文件每进程只探测一次（词表是增强面：覆盖文件
// 不在场/解析失败都静默保内置默认，不挡去味主功能）。
var novelWordsOnce sync.Once

// ensureNovelStyleWords 按惯例目录加载词表覆盖（v4.225 规范知识出内核）：
// <cwd>/.gaea|/.agents|/.agent|/.claude/skills/novel-deslop/words.json，
// .gaea 优先。用户改词表不改代码不发版。
func ensureNovelStyleWords() {
	novelWordsOnce.Do(func() {
		for _, base := range gaeaconfig.ConventionDirs { // .gaea 最优先
			p := filepath.Join(gaeaCwd(), base, "skills", "novel-deslop", "words.json")
			if _, err := os.Stat(p); err == nil {
				_ = novelstyle.LoadWordsFile(p)
				return
			}
		}
	})
}

// ensureNovelStylePatterns 按惯例目录加载模式表覆盖（oh-story T2 内核消费；
// 同 words.json 纪律：.gaea/skills/novel-deslop/patterns.json 整体替换）。
var novelPatternsOnce sync.Once

func ensureNovelStylePatterns() {
	novelPatternsOnce.Do(func() {
		for _, base := range gaeaconfig.ConventionDirs { // .gaea 最优先
			p := filepath.Join(gaeaCwd(), base, "skills", "novel-deslop", "patterns.json")
			if _, err := os.Stat(p); err == nil {
				_ = novelstyle.LoadPatternsFile(p)
				return
			}
		}
	})
}

// bookWhitelist 读书级白名单（oh-story T2：<项目目录>/.deslop-whitelist，
// 一行一显式授权片段）。无文件不建空表（gates.json A.whitelist 口径）；
// 读取失败按无豁免处理并如实记日志，不挡去味主功能。
func bookWhitelist(pm *project.Manager) []string {
	if pm == nil {
		return nil
	}
	wl, err := novelstyle.LoadWhitelistFile(filepath.Join(pm.Dir, ".deslop-whitelist"))
	if err != nil {
		slog.Warn("读取书级白名单失败（按无豁免处理）", "error", err)
		return nil
	}
	return wl
}

func (a *writingState) DeSlopChapterAiTaste(chapterNum int) (map[string]interface{}, error) {
	ensureNovelStyleWords()
	ensureNovelStylePatterns()
	pm := a.getPM()
	wl := bookWhitelist(pm)
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
			novelstyle.ApplyWhitelist(b, sc.Content, wl)
			rw, rep, derr := novelstyle.DeSlopRewriteEx(sc.Content, b, wl)
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
		novelstyle.ApplyWhitelist(b, content, wl)
		rw, rep, derr := novelstyle.DeSlopRewriteEx(content, b, wl)
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
