package app

// 场景卡与场景级生成（长篇刀2，规格 进度计划/gaea-novel-scene-cards-20260929.md）。
//
// 三个新面：
//  1. NovelChapterScenesGenerate —— 整章按场景卡逐场景生成（流式 scene-gen-stream
//     事件；卡缺 Goal/Conflict 写前闸可覆盖；复用 chapterGenMu 同章互斥与
//     CancelCreateChapter 取消；完成 Stitch→blob + SceneRefs 回写）。
//  2. NovelSceneRewrite —— 单场景 whole 重写（快照先行+版本库 mode=scene 留痕，
//     他场与正稿不动；局部重写仍禁场景章——选段↔场景映射维持观察池）。
//  3. NovelSceneCardsPropose —— 从章计划/大纲节点拆场景卡骨架，提案不落盘
//     （确认制），作者审批后走既有 CreateScene+SaveSceneMeta。
//
// 卡注入与上一场衔接由 buildSceneUserPrompt 统一供给：无卡无前场时 prompt 与
// 既有 GenerateScene 逐字节一致（旧场景零回归）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"log/slog"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/maturecraft"
	"github.com/gaea/gaea/internal/novelcontext"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/scene"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// sceneGenStreamChannel 逐场景生成流事件通道（与 create-chapter-stream 分离：
// 负载语义不同——按场景推进而非按 chunk 推进）。
const sceneGenStreamChannel = "scene-gen-stream"

// sceneCardSection 渲染场景卡工艺区段（§1.4 的正面解：把「戏要干什么、怎么算
// 写成了」从作者脑内变成 prompt 硬约束）。meta 无任何卡字段且无前场衔接时
// 返回空串（prompt 零变化）。
func sceneCardSection(meta *types.SceneMeta, prev *types.SceneMeta) string {
	var b strings.Builder
	line := func(label, val string) {
		if strings.TrimSpace(val) != "" {
			fmt.Fprintf(&b, "- %s：%s\n", label, strings.TrimSpace(val))
		}
	}
	if meta != nil && (meta.Goal != "" || meta.Conflict != "" || meta.Turn != "" ||
		meta.Outcome != "" || meta.Sequel != "" || meta.ExitHook != "") {
		b.WriteString("## 本场景卡（这场戏的工艺约束，正文必须落实）\n")
		line("场景目标（这场戏要什么）", meta.Goal)
		line("冲突（谁·什么在阻挡）", meta.Conflict)
		line("价值转折（这场戏的价值从什么变成什么）", meta.Turn)
		line("结果义务（写完必须成立的状态变化）", meta.Outcome)
		line("余波（反应·两难·决定）", meta.Sequel)
		line("退出钩子（拉住读者进下一场）", meta.ExitHook)
	}
	if prev != nil && (prev.Outcome != "" || prev.ExitHook != "") {
		b.WriteString("\n## 上一场衔接（本场景从这里的余波开始）\n")
		line("上一场结果", prev.Outcome)
		line("上一场退出钩子", prev.ExitHook)
	}
	return strings.TrimRight(b.String(), "\n")
}

// buildSceneUserPrompt 场景生成 user prompt。无卡无衔接时与既有拼装逐字节一致
// （回归底线）；有卡时在尾部追加结构化区段。
func buildSceneUserPrompt(minWords int, scene *types.Scene, chapterNum int, plotReq, bible string, prev *types.SceneMeta) string {
	user := fmt.Sprintf("请写出本场景正文，直接开始，不要前言、标题或元信息，不少于%d字。\n\n场景：%s\n章节号：%d\n剧情要求：%s\n\n%s",
		minWords, scene.Meta.Title, chapterNum, plotReq, bible)
	if card := sceneCardSection(&scene.Meta, prev); card != "" {
		user += "\n\n" + card
	}
	return user
}

// scenePrevMeta 按 Order 取指定场景的前一场元数据（衔接注入用；首场/找不到
// 返回 nil）。
func scenePrevMeta(sm *scene.Manager, sceneID string) *types.SceneMeta {
	metas, err := sm.List()
	if err != nil {
		return nil
	}
	idx := -1
	for i := range metas {
		if metas[i].ID == sceneID {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return nil
	}
	return &metas[idx-1]
}

// generateSceneCore 单场景生成内核（GenerateScene 与整章逐场景流共用）。
// ctx 取消即停（整章流的取消语义）；extraPlan 为章计划区段（手动单场景路径
// 传空=现状零变化）。
func (a *writingState) generateSceneCore(ctx context.Context, pm *project.Manager, chapterNum int, sceneID, plotReq string, minWords int, prev *types.SceneMeta, extraPlan string) (*types.Scene, *novelstyle.RewriteReport, error) {
	if a.clientRef() == nil {
		return nil, nil, fmt.Errorf("AI client not ready")
	}
	sm := pm.SceneManager(chapterNum)
	scene, err := sm.Read(sceneID)
	if err != nil {
		return nil, nil, fmt.Errorf("读取场景失败: %w", err)
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
		return nil, nil, fmt.Errorf("未找到可用模型（可能离线）")
	}

	system := "你是正在写这本书的作者。用场景和动作说话，不解释，不煽情，让读者感受到发生了什么。" +
		"严格遵守角色与领域设定，不 OOC，不提前揭穿伏笔。"
	// v4.439：成人向工艺区段（P1 槽同源文本）——场景路径无模板槽，system 追加，
	// 非成人向零追加。
	if sec := maturecraft.CraftSection(pm.Meta.Mature); sec != "" {
		system += "\n\n" + sec
	}
	user := buildSceneUserPrompt(minWords, scene, chapterNum, plotReq, bible, prev)
	if extraPlan != "" {
		user += "\n\n" + extraPlan
	}

	reqCtx := ctx
	if reqCtx == nil {
		reqCtx = context.Background()
	}
	reply, err := a.clientRef().ChatSimpleStreamWithOptions(reqCtx, model, system, user, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.8, MaxTokens: 4096,
	})
	if err != nil {
		if reqCtx.Err() != nil {
			return nil, nil, fmt.Errorf("已取消（%v）", reqCtx.Err())
		}
		return nil, nil, fmt.Errorf("场景生成失败: %w", err)
	}

	content := strings.TrimSpace(reply)
	// 去 AI 味后处理（分数未改善不落盘，与手动路径同口径）
	var deslop *novelstyle.RewriteReport
	if rx, rep, derr := novelstyle.DeSlopRewrite(content, nil); derr == nil && rep != nil && rep.AfterScore < rep.BeforeScore && rx != "" {
		content = rx
		deslop = rep
	}
	scene.Content = content
	scene.Meta.WordCount = len([]rune(content))
	if err := sm.Write(scene); err != nil {
		return nil, nil, fmt.Errorf("保存场景失败: %w", err)
	}
	syncBlobFromScenes(pm, chapterNum)
	return scene, deslop, nil
}

// NovelChapterScenesGenerate 整章按场景卡逐场景生成（长篇刀2 主入口）。
//
// 语义：按 Order 逐场景生成并**每场完成即落盘**；场景卡缺 Goal 或 Conflict 时
// 写前闸拒绝并点名（allowMissingCard=true 显式跳过缺卡场景——刀1 覆盖语义）；
// 任意一场生成失败即停（已完成场景保留，事件如实报告）；全部完成 Stitch→
// WriteChapter blob（读路径兼容）+ SceneRefs 回写 + 大纲标记。
// 互斥与取消：复用 chapterGenKey(chapterNum, "")——与整章生成同章互斥，
// CancelCreateChapter 同款取消（取消后已完成场景保留、未生成场景不开始）。
func (a *writingState) NovelChapterScenesGenerate(chapterNum int, allowMissingCard bool) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI client not ready")
	}
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号非法")
	}
	sm := pm.SceneManager(chapterNum)
	metas, err := sm.List()
	if err != nil {
		return nil, fmt.Errorf("读取场景列表失败: %w", err)
	}
	if len(metas) == 0 {
		return nil, fmt.Errorf("本章还没有场景：先在编辑区建场景并填场景卡（ⓘ 弹窗），或用「AI 拆场景卡」生成骨架")
	}

	// 写前闸：缺 Goal 或 Conflict 的场景点名（可显式跳过）。手动单场景
	// GenerateScene 不上闸（兼容既有流）。
	var missing []string
	for i := range metas {
		if strings.TrimSpace(metas[i].Goal) == "" || strings.TrimSpace(metas[i].Conflict) == "" {
			missing = append(missing, fmt.Sprintf("「%s」（缺 %s）", metas[i].Title, sceneCardMissingFields(&metas[i])))
		}
	}
	if len(missing) > 0 && !allowMissingCard {
		return nil, fmt.Errorf("第%d章逐场景生成未通过写前闸：以下场景的场景卡缺目标/冲突，先补卡或显式跳过：%s。场景卡在编辑区场景 ⓘ 弹窗填写",
			chapterNum, strings.Join(missing, "、"))
	}

	key := chapterGenKey(chapterNum, "")
	ctx, cancel, err := a.registerChapterGen(key, chapterNum, "")
	if err != nil {
		return nil, err
	}

	// 章计划区段（刀1 资产）：逐场景生成以计划为纲
	planSection := a.sceneGenPlanSection(pm, chapterNum)

	a.chapterGenWG.Add(1)
	go func() {
		defer a.chapterGenWG.Done()
		defer a.unregisterChapterGen(key, cancel)
		done, skipped := 0, 0
		for i := range metas {
			if ctx.Err() != nil {
				a.emit(sceneGenStreamChannel, map[string]interface{}{
					"type": "cancelled", "chapterNum": chapterNum,
					"done": done, "skipped": skipped, "total": len(metas),
				})
				return
			}
			m := metas[i]
			if (strings.TrimSpace(m.Goal) == "" || strings.TrimSpace(m.Conflict) == "") && allowMissingCard {
				skipped++
				a.emit(sceneGenStreamChannel, map[string]interface{}{
					"type": "scene-skipped", "chapterNum": chapterNum,
					"sceneID": m.ID, "title": m.Title, "index": i + 1, "total": len(metas),
				})
				continue
			}
			var prev *types.SceneMeta
			if i > 0 {
				prev = &metas[i-1]
			}
			scene, _, serr := a.generateSceneCore(ctx, pm, chapterNum, m.ID, "", 0, prev, planSection)
			if serr != nil {
				a.emit(sceneGenStreamChannel, map[string]interface{}{
					"type": "error", "chapterNum": chapterNum, "sceneID": m.ID,
					"title": m.Title, "error": fmt.Sprintf("第 %d/%d 场「%s」生成失败：%v（已完成 %d 场已落盘保留）", i+1, len(metas), m.Title, serr, done),
				})
				return
			}
			done++
			a.emit(sceneGenStreamChannel, map[string]interface{}{
				"type": "scene-done", "chapterNum": chapterNum,
				"sceneID": m.ID, "title": m.Title, "index": i + 1, "total": len(metas),
				"words": scene.Meta.WordCount,
			})
		}

		// 全部完成：Stitch→blob（读路径兼容）+ SceneRefs 回写 + 大纲标记
		totalWords := 0
		if stitched, serr := sm.Stitch(); serr == nil && strings.TrimSpace(stitched) != "" {
			if werr := pm.WriteChapter(chapterNum, stitched); werr != nil {
				a.emit(sceneGenStreamChannel, map[string]interface{}{
					"type": "error", "chapterNum": chapterNum,
					"error": fmt.Sprintf("场景已生成但整章落盘失败：%v（各场景文件仍在，可重试）", werr),
				})
				return
			}
			totalWords = len([]rune(stitched))
		}
		a.syncSceneRefs(pm, chapterNum, metas)
		if merr := a.markOutlineDone(pm, chapterNum, ""); merr != nil {
			a.emit(sceneGenStreamChannel, map[string]interface{}{
				"type": "done", "chapterNum": chapterNum, "outlineWarning": merr.Error(),
				"scenes": len(metas), "done": done, "skipped": skipped, "totalWords": totalWords,
			})
			return
		}
		a.emit(sceneGenStreamChannel, map[string]interface{}{
			"type": "done", "chapterNum": chapterNum,
			"scenes": len(metas), "done": done, "skipped": skipped, "totalWords": totalWords,
		})
	}()

	return map[string]interface{}{"started": true, "scenes": len(metas), "missingCards": len(missing)}, nil
}

// sceneCardMissingFields 列出卡缺失字段名（闸提示用）。
func sceneCardMissingFields(m *types.SceneMeta) string {
	var fields []string
	if strings.TrimSpace(m.Goal) == "" {
		fields = append(fields, "目标")
	}
	if strings.TrimSpace(m.Conflict) == "" {
		fields = append(fields, "冲突")
	}
	return strings.Join(fields, "·")
}

// sceneGenPlanSection 逐场景生成的章计划区段（刀1 契约复用）+ 故事层切片（刀3）。
// 两者皆空返回空（prompt 零变化）；空 spine 零注入。
func (a *writingState) sceneGenPlanSection(pm *project.Manager, chapterNum int) string {
	out := ""
	if pf, err := pm.ReadChapterPlans(); err == nil && pf != nil {
		if plan, ok := pf.Plans[fmt.Sprintf("%d", chapterNum)]; ok {
			if sec := buildChapterPlanSection(&plan); sec != "" {
				out = "## 本章计划（所有场景合力完成它，不得遗漏关键事件）\n" + sec
			}
		}
	}
	// 刀3：故事层切片（节拍位/活跃线程/未解问题/弧线水位）——与整章生成同源。
	if sp := a.storySpineSection(pm, chapterNum); sp != "" {
		if out != "" {
			out += "\n"
		}
		out += sp
	}
	// 刀5：作者风格约束——与整章生成同源；零 digest 零注入。
	if dg := a.styleDigestSection(pm); dg != "" {
		if out != "" {
			out += "\n"
		}
		out += dg
	}
	return out
}

// syncSceneRefs 把章的场景 ID 列表回写进大纲节点 SceneRefs（G10 启用）。
// AP1-06 收敛：递归找节点并入 findOutlineNode（N13 纪律），分支语义在谓词
// （只认主线 Branch==""，与改前一致）；读-改-写只动目标节点的 SceneRefs 字段，
// 找到节点即写盘（无变化只跳过字段赋值，磁盘照写——与改前行为一致）。
func (a *writingState) syncSceneRefs(pm *project.Manager, chapterNum int, metas []types.SceneMeta) {
	ids := make([]string, 0, len(metas))
	for i := range metas {
		ids = append(ids, metas[i].ID)
	}
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		return
	}
	node := findOutlineNode(of.Nodes, func(n *types.OutlineNode) bool {
		return n.OrderIndex == chapterNum && n.Branch == ""
	})
	if node == nil {
		return // 大纲无该章节点（场景章未建节点）不算错
	}
	if strings.Join(node.SceneRefs, "\x00") != strings.Join(ids, "\x00") {
		node.SceneRefs = ids // 无变化不动字段
	}
	if err := pm.WriteOutlines(of); err != nil {
		slog.Warn("SceneRefs 回写失败", "chapter", chapterNum, "error", err)
	}
}

// NovelSceneRewrite 单场景 whole 重写（场景级重写解禁，刀2）。
//
// 语义：快照先行（SnapshotStore）→ 卡字段+上一场衔接进指令 → rewrite-chapter
// 模板重写 → 写回该场景 → 版本库留痕（mode=scene，Original/NewContent 存整章
// 前后 Stitch 快照，回滚=整章恢复安全）→ Stitch→blob 同步。**他场与章内其他
// 内容一字不动**（字节级验证见测试）。局部重写仍禁场景章（选段↔场景映射
// 维持观察池）。
func (a *writingState) NovelSceneRewrite(chapterNum int, sceneID string, instruction string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI client not ready")
	}
	if strings.TrimSpace(instruction) == "" {
		return nil, fmt.Errorf("重写指令为空：告诉模型这场戏要怎么改")
	}
	sm := pm.SceneManager(chapterNum)
	scene, err := sm.Read(sceneID)
	if err != nil {
		return nil, fmt.Errorf("读取场景失败: %w", err)
	}

	// 重写前整章快照（版本库 OriginalContent 语义=全文，回滚安全）
	beforeFull, _ := pm.ReadChapterAsStitch(chapterNum)

	// 快照先行（场景级历史，编辑区可 Restore）
	if _, cerr := pm.SnapshotStore(chapterNum).Capture(sceneID, scene.Content, "AI 场景重写前", "ai-rewrite"); cerr != nil {
		slog.Warn("场景重写前快照失败（继续重写）", "scene", sceneID, "error", cerr)
	}

	tmpl := a.eng.Get("rewrite-chapter")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 rewrite-chapter 模板文件")
	}
	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return nil, fmt.Errorf("未找到可用模型（可能离线）")
	}

	// 指令组装：重写指令 + 原场景正文 + 卡字段约束 + 上一场衔接
	var b strings.Builder
	fmt.Fprintf(&b, "重写指令：%s\n\n", strings.TrimSpace(instruction))
	if card := sceneCardSection(&scene.Meta, nil); card != "" {
		b.WriteString(card)
		b.WriteString("\n\n（以上场景卡是这场戏的既定约束：目标/冲突/结果义务不得改写偏离，只改写实现方式）\n")
	}
	if prev := scenePrevMeta(sm, sceneID); prev != nil {
		if sec := sceneCardSection(nil, prev); sec != "" {
			b.WriteString(sec)
			b.WriteString("\n")
		}
	}
	system := tmpl.BuildSystemPrompt("")
	user := tmpl.BuildUserPrompt(map[string]string{
		"chapter_content":          scene.Content,
		"modification_instruction": strings.TrimRight(b.String(), "\n"),
		"prev_summary":             "",
		"mature_craft":             maturecraft.CraftSection(pm.Meta.Mature), // v4.439：场景重写不洗掉成人向内容
	})

	reqCtx := a.ctx
	if reqCtx == nil {
		reqCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(reqCtx, 10*time.Minute)
	defer cancel()
	reply, err := a.clientRef().ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.7, MaxTokens: 8192, TimeoutMinutes: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("场景重写生成失败: %w", err)
	}
	newContent := strings.TrimSpace(reply)
	if newContent == "" {
		return nil, fmt.Errorf("重写结果为空，已放弃（不落任何数据）")
	}

	scene.Content = newContent
	scene.Meta.WordCount = len([]rune(newContent))
	if err := sm.Write(scene); err != nil {
		return nil, fmt.Errorf("写回场景失败: %w", err)
	}
	syncBlobFromScenes(pm, chapterNum)

	afterFull, _ := pm.ReadChapterAsStitch(chapterNum)
	v := &types.RewriteVersion{
		ChapterNum:        chapterNum,
		Mode:              types.RewriteModeScene,
		Status:            types.RewriteCompleted,
		Source:            "custom",
		CustomInstr:       instruction,
		OriginalContent:   beforeFull,
		OriginalWordCount: len([]rune(beforeFull)),
		NewContent:        afterFull,
		NewWordCount:      len([]rune(afterFull)),
	}
	if err := pm.SaveRewriteVersion(v); err != nil {
		return nil, fmt.Errorf("保存重写版本失败: %w", err)
	}

	return map[string]interface{}{
		"versionId": v.ID, "mode": string(types.RewriteModeScene),
		"sceneID": sceneID, "title": scene.Meta.Title,
		"sceneWordCount":   scene.Meta.WordCount,
		"chapterWordCount": len([]rune(afterFull)),
	}, nil
}

// sceneCardProposal 提案面的一张卡（不落盘形态；确认后由前端走
// CreateScene+SaveSceneMeta 落卡）。
type sceneCardProposal struct {
	Title    string `json:"title"`
	Goal     string `json:"goal"`
	Conflict string `json:"conflict"`
	Turn     string `json:"turn"`
	Outcome  string `json:"outcome"`
	Sequel   string `json:"sequel"`
	ExitHook string `json:"exit_hook"`
}

// NovelSceneCardsPropose 从章计划+大纲节点+前章摘要拆场景卡骨架（提案制不落盘）。
// 返回 2-5 张卡；正文与卡由作者审批后经既有绑定落库（确认红线同族）。
func (a *writingState) NovelSceneCardsPropose(chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI client not ready")
	}

	// 章计划（刀1）：拆卡的意图来源；无计划时用大纲节点兜底
	planJSON := ""
	if pf, err := pm.ReadChapterPlans(); err == nil && pf != nil {
		if plan, ok := pf.Plans[fmt.Sprintf("%d", chapterNum)]; ok {
			if raw, merr := json.Marshal(plan); merr == nil {
				planJSON = string(raw)
			}
		}
	}
	nodeDesc := ""
	if of, err := pm.ReadOutlines(); err == nil && of != nil {
		if node := findOutlineNodeByNum(of.Nodes, chapterNum); node != nil {
			nodeDesc = strings.TrimSpace("标题：" + node.Title + "\n摘要：" + node.Summary)
			if len(node.KeyPoints) > 0 {
				nodeDesc += "\n要点：" + strings.Join(node.KeyPoints, " / ")
			}
		}
	}
	if planJSON == "" && nodeDesc == "" {
		return nil, fmt.Errorf("本章既没有章节计划也没有大纲节点（标题/摘要），AI 无从拆卡：先到「章节计划」补计划或在大纲里填本章节点")
	}

	prevSummary := ""
	if s, err := pm.ReadLatestChapterSummaryBefore(chapterNum); err == nil && s != nil {
		prevSummary = s.Summary
	}

	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return nil, fmt.Errorf("未找到可用模型（可能离线）")
	}
	system := "你是长篇小说的结构编辑。把「这一章要完成什么」拆成 2-5 个场景卡：每场戏一个目标、一个阻挡、一次价值转折、一个必须成立的结果。场景之间用结果→钩子衔接，合力完成本章计划的关键事件，不得遗漏、不得重复。" +
		"只输出一个 JSON 数组，第一个字符是 [，最后一个字符是 ]，不要任何解释文字或 Markdown 围栏。"
	var b strings.Builder
	if planJSON != "" {
		b.WriteString("本章计划（七字段）：\n" + planJSON + "\n\n")
	}
	if nodeDesc != "" {
		b.WriteString("本章大纲节点：\n" + nodeDesc + "\n\n")
	}
	if prevSummary != "" {
		b.WriteString("上一章摘要（衔接参考）：\n" + util.Truncate(prevSummary, 300) + "\n\n")
	}
	b.WriteString(`请拆场景卡。每张卡的字段：{"title":"场景名（2-6字）","goal":"这场戏要什么（POV 视角的欲望，一句话）","conflict":"谁·什么在阻挡（一句话）","turn":"价值从什么变成什么（一句话）","outcome":"结果：写完必须成立的状态变化（谁得到/失去了什么）","sequel":"余波：反应·两难·决定（一句话，可空）","exit_hook":"退出钩子（拉住读者进下一场，一句话）"}`)

	propCtx := a.ctx
	if propCtx == nil {
		propCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(propCtx, 5*time.Minute)
	defer cancel()
	reply, err := a.clientRef().ChatSimpleStreamWithOptions(ctx, model, system, b.String(), ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.4, MaxTokens: 3000, TimeoutMinutes: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("拆卡失败: %w", err)
	}
	var cards []sceneCardProposal
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &cards); err != nil {
		return nil, fmt.Errorf("拆卡结果解析失败: %w", err)
	}
	if len(cards) == 0 {
		return nil, fmt.Errorf("拆卡结果为空")
	}
	out := make([]map[string]interface{}, 0, len(cards))
	for i := range cards {
		out = append(out, map[string]interface{}{
			"title": cards[i].Title, "goal": cards[i].Goal, "conflict": cards[i].Conflict,
			"turn": cards[i].Turn, "outcome": cards[i].Outcome,
			"sequel": cards[i].Sequel, "exit_hook": cards[i].ExitHook,
		})
	}
	return out, nil
}
