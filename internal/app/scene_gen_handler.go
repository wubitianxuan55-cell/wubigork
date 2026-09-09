package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/novelcontext"
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
func (a *writingState) GenerateScene(chapterNum int, sceneID string, plotReq string, minWords int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.client == nil {
		return nil, fmt.Errorf("AI client not ready")
	}
	sm := pm.SceneManager(chapterNum)
	scene, err := sm.Read(sceneID)
	if err != nil {
		return nil, fmt.Errorf("读取场景失败: %w", err)
	}
	if minWords <= 0 {
		minWords = 800
	}

	// 编译 POV 感知场景圣经（失败静默降级，不阻断生成）。
	bible := ""
	if b, berr := novelcontext.CompileSceneBible(pm, chapterNum, scene); berr == nil && b != nil {
		bible = b.Render(ctxSceneBibleBudget)
	}

	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return nil, fmt.Errorf("未找到可用模型（可能离线）")
	}

	system := "你是正在写这本书的作者。用场景和动作说话，不解释，不煽情，让读者感受到发生了什么。" +
		"严格遵守角色与领域设定，不 OOC，不提前揭穿伏笔。"
	user := fmt.Sprintf("请写出本场景正文，直接开始，不要前言、标题或元信息，不少于%d字。\n\n场景：%s\n章节号：%d\n剧情要求：%s\n\n%s",
		minWords, scene.Meta.Title, chapterNum, plotReq, bible)

	reply, err := a.client.ChatSimpleStreamWithOptions(context.Background(), model, system, user, ai.ChatSimpleOptions{
		EngineID: eng, Temperature: 0.8, MaxTokens: 4096,
	})
	if err != nil {
		return nil, fmt.Errorf("场景生成失败: %w", err)
	}

	content := strings.TrimSpace(reply)
	var deslop *novelstyle.RewriteReport
	if rx, rep, derr := novelstyle.DeSlopRewrite(content, nil); derr == nil && rep != nil && rep.AfterScore < rep.BeforeScore && rx != "" {
		content = rx
		deslop = rep
	}

	scene.Content = content
	if err := sm.Write(scene); err != nil {
		return nil, fmt.Errorf("保存场景失败: %w", err)
	}
	syncBlobFromScenes(pm, chapterNum)
	a.markOutlineDone(pm, chapterNum, "")

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
	}
}

func (a *writingState) markOutlineDone(pm *project.Manager, chapterNum int, branch string) {
	if of, err := pm.ReadOutlines(); err == nil && of != nil {
		for i := range of.Nodes {
			if of.Nodes[i].OrderIndex == chapterNum && of.Nodes[i].Branch == branch {
				of.Nodes[i].Status = types.OutlineDone
				break
			}
		}
		_ = pm.WriteOutlines(of)
	}
}
