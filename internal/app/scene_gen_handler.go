package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 刀 1 · 接通生成→场景 ─────────────────────────────────────────────
//
// 现有 CreateChapter 走「整章 blob」（chapters/NNN.md），与 v4 场景制
// （SceneMeta.POVCharID/Location/TimeOfDay/Emotion/Tags）脱节。本文件补齐
// 场景级生成入口：CreateScene 建一个场景，GenerateScene 用 novelcontext
// 编译的「场景圣经」逐场景生成正文（POV 感知）并落盘到 scene.Manager。
// 前端 ChapterEditor（场景多文本框）可逐场景调用，把「整章 blob」升维成
// 真正由场景驱动的生成。不改动既有 CreateChapter（保持兼容），新增路径。
//
// 1A 接通：blob 章首次触碰场景 API 时物化为第 1 个场景（ensureBlobChapterScene），
// 阅读页逐场景生成从「无绑定 ID」变为可用；加场景走 CreateScene 真落盘。

// ensureBlobChapterScene 把「整章 blob 但一个场景都没有」的章节接进场景制：
// 幂等（已有场景或无 blob 正文时不动盘）；物化 = blob 全文成为 slug=chapter 的
// 首场景（status=done），标题取大纲章标题、缺省「第N章」。失败只降级不阻断——
// 调用方（GetChapterScenes/CreateScene）照常走原有流程。
func ensureBlobChapterScene(pm *project.Manager, chapterNum int) {
	sm := pm.SceneManager(chapterNum)
	metas, err := sm.List()
	if err != nil || len(metas) > 0 {
		return
	}
	blob, err := pm.ReadChapter(chapterNum)
	if err != nil || strings.TrimSpace(blob) == "" {
		return
	}
	title := gateOutlineTitle(pm, chapterNum)
	if title == "" {
		title = fmt.Sprintf("第%d章", chapterNum)
	}
	sc, err := sm.Create("chapter", title)
	if err != nil {
		slog.Warn("blob 章物化场景失败（继续）", "chapter", chapterNum, "error", err)
		return
	}
	sc.Content = blob
	sc.Meta.Status = types.SceneDone
	if err := sm.Write(sc); err != nil {
		slog.Warn("blob 章物化场景写盘失败（继续）", "chapter", chapterNum, "error", err)
		return
	}
	slog.Info("blob 章已物化为首场景", "chapter", chapterNum, "sceneId", sc.Meta.ID)
}

// syncBlobFromScenes 把场景库回写为整章 blob 投影（V4 且有场景时）：
// blob = 各场景正文按空段拼接（无分隔线，保导出/搜索/统计可读）。
// 场景写路径（逐场景保存/逐场景生成/重排/快照恢复）之后调用，保证 blob
// 消费方读到与场景一致的正文。失败只 warn 不阻断——blob 滞后可由下一次
// 场景写自愈。
func syncBlobFromScenes(pm *project.Manager, chapterNum int) {
	sm := pm.SceneManager(chapterNum)
	metas, err := sm.List()
	if err != nil || len(metas) == 0 {
		return
	}
	parts := make([]string, 0, len(metas))
	for _, meta := range metas {
		sc, rerr := sm.Read(meta.ID)
		if rerr != nil {
			continue
		}
		parts = append(parts, sc.Content)
	}
	if len(parts) == 0 {
		return
	}
	if werr := pm.WriteChapter(chapterNum, strings.Join(parts, "\n\n")); werr != nil {
		slog.Warn("场景回写 blob 投影失败（继续）", "chapter", chapterNum, "error", werr)
	}
}

// rebuildScenesFromBlob 整章重写（CreateChapter 主线完成点）后的场景对齐：
// 已有场景全部删除，再从新 blob 物化单场景。整章重写后旧拆分与旧 POV 元数据
// 不再对应新正文，重置=诚实可预期（blob 章物化规则与 ensureBlobChapterScene
// 一致）；无场景的章 no-op（交给惰性物化）。
func rebuildScenesFromBlob(pm *project.Manager, chapterNum int) {
	sm := pm.SceneManager(chapterNum)
	metas, err := sm.List()
	if err != nil || len(metas) == 0 {
		return
	}
	for _, meta := range metas {
		if derr := sm.Delete(meta.ID); derr != nil {
			slog.Warn("场景重置删除失败（继续）", "chapter", chapterNum, "scene", meta.ID, "error", derr)
		}
	}
	ensureBlobChapterScene(pm, chapterNum)
	slog.Info("整章重写后场景已重置", "chapter", chapterNum)
}

// CreateScene 在指定章节下创建一个 v4 场景。
// 若该章还是「纯 blob、无场景」状态，先物化首场景再建新场景（1A 接通）。
func (a *writingState) CreateScene(chapterNum int, slug string, title string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号非法")
	}
	if slug == "" {
		slug = "scene"
	}
	if title == "" {
		title = "新场景"
	}
	ensureBlobChapterScene(pm, chapterNum)
	sm := pm.SceneManager(chapterNum)
	sc, err := sm.Create(slug, title)
	if err != nil {
		return nil, fmt.Errorf("创建场景失败: %w", err)
	}
	return sceneToMap(sc), nil
}

// GenerateScene 用 novelcontext 场景圣经，为指定场景生成正文并落盘。
// 非流式（返回完整正文 + AI 味分 + 去味报告）；minWords<=0 用默认 800。
// GenerateScene 用 novelcontext 场景圣经，为指定场景生成正文并落盘。
// 非流式（返回完整正文 + AI 味分 + 去味报告）；minWords<=0 用默认 800。
// 长篇刀2起：场景卡字段（Goal/Conflict/Turn/Outcome/Sequel/ExitHook）非空时
// 注入 prompt 作工艺约束，并注入上一场的衔接（Outcome/ExitHook）；无卡场景
// prompt 与旧版逐字节一致（零回归）。手动路径不上卡闸（兼容既有流；整章
// 逐场景流才闸——NovelChapterScenesGenerate）。
func (a *writingState) GenerateScene(chapterNum int, sceneID string, plotReq string, minWords int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	sm := pm.SceneManager(chapterNum)
	prev := scenePrevMeta(sm, sceneID)
	scene, deslop, err := a.generateSceneCore(context.Background(), pm, chapterNum, sceneID, plotReq, minWords, prev, "")
	if err != nil {
		return nil, err
	}
	content := scene.Content

	var outlineWarning string
	if err := a.markOutlineDone(pm, chapterNum, ""); err != nil {
		// 场景已落盘，大纲标记失败不回滚正文——但必须如实带出（N8 同型：
		// 原 `_ =` 吞错，作者以为本章已标完成，大纲面板却还是「未写」）。
		slog.Warn("场景生成：大纲状态标记失败", "chapter", chapterNum, "error", err)
		outlineWarning = fmt.Sprintf("正文已保存，但大纲状态标记失败：%v", err)
	}

	score := 0
	if ts, terr := novelstyle.ScoreTextNoRef(content); terr == nil && ts != nil {
		score = ts.Score
	}
	res := map[string]interface{}{
		"scene":   sceneToMap(scene),
		"content": content,
		"aiTaste": score,
		"words":   len([]rune(content)),
	}
	if deslop != nil {
		res["deSlop"] = deslop
	}
	if outlineWarning != "" {
		res["outlineWarning"] = outlineWarning
	}
	return res, nil
}

func sceneToMap(s *types.Scene) map[string]interface{} {
	return map[string]interface{}{
		"id":        s.Meta.ID,
		"slug":      s.Meta.Slug,
		"title":     s.Meta.Title,
		"summary":   s.Meta.Summary,
		"povCharId": s.Meta.POVCharID,
		"location":  s.Meta.Location,
		"timeOfDay": s.Meta.TimeOfDay,
		"emotion":   s.Meta.Emotion,
		"tags":      s.Meta.Tags,
		"status":    string(s.Meta.Status),
		"wordCount": s.Meta.WordCount,
		"order":     s.Meta.Order,
		"content":   s.Content,

		// 场景卡创作学字段（长篇刀2；omitempty 语义——空值输出空串保持键在，
		// 前端窄化统一按 string 收）
		"goal":      s.Meta.Goal,
		"conflict":  s.Meta.Conflict,
		"turn":      s.Meta.Turn,
		"outcome":   s.Meta.Outcome,
		"sequel":    s.Meta.Sequel,
		"exit_hook": s.Meta.ExitHook,
	}
}

// markOutlineDone 把本章大纲节点标为已写（AP1-06 收敛：递归找节点并入
// findOutlineNode；分支语义在谓词——认调用方传入的 branch，主线传 ""）；
// 写盘失败如实返回错误（调用方决定呈现方式）。
func (a *writingState) markOutlineDone(pm *project.Manager, chapterNum int, branch string) error {
	of, err := pm.ReadOutlines()
	if err != nil {
		return err
	}
	if of == nil {
		return nil
	}
	node := findOutlineNode(of.Nodes, func(n *types.OutlineNode) bool {
		return n.OrderIndex == chapterNum && n.Branch == branch
	})
	if node == nil {
		return nil // 大纲里没有该章节点（场景章未建节点）不算错
	}
	node.Status = types.OutlineDone
	return pm.WriteOutlines(of)
}
