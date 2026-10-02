package app

// 质量收敛闭环（长篇刀4，规格 进度计划/gaea-novel-converge-20260930.md）。
//
// 判据纯函数 convergeCheck（AI 味 ≤ 目标且 S1/S2 归零；S3 不阻断）+
// 流式闭环 NovelChapterConverge（每轮：轮前单元快照→定向修补→复检→三态判停：
// converged / no-improve 按单元回滚本轮 / max-rounds 剩余如实报告；每轮完成
// 版本库留痕 mode=converge）+ dry-run 预检 NovelChapterConvergePreview。
// 规格 §1.7「全是报告，没有闭环、没有收敛判据」的正面解。

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/maturecraft"
	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// convergeStreamChannel 收敛闭环流事件通道。
const convergeStreamChannel = "converge-stream"

// 收敛判据默认值。
const (
	convergeDefaultTargetTaste = 40 // AI 味目标分（0-100 越低越好；≤40=可读性达标线）
	convergeDefaultMaxRounds   = 2  // 轮次上限（避免无限重写）
)

// ConvergeCheck 判据纯函数结果（零 LLM）。
type ConvergeCheck struct {
	TasteScore int               `json:"taste_score"` // 0-100，越高越像 AI
	S1S2       []novelgate.Issue `json:"s1s2"`        // 阻断级确定性信号（空=过）
	S3         []novelgate.Issue `json:"s3"`          // 提示级（不阻断）
	Converged  bool              `json:"converged"`
}

// convergeCheck 判据：AI 味 ≤ target 且 S1/S2 归零（S3 仅提示不阻断）。
func convergeCheck(text string, target int) ConvergeCheck {
	if target <= 0 {
		target = convergeDefaultTargetTaste
	}
	score := 0
	if ts, err := novelstyle.ScoreTextNoRef(text); err == nil && ts != nil {
		score = ts.Score
	}
	ck := ConvergeCheck{TasteScore: score, S1S2: []novelgate.Issue{}, S3: []novelgate.Issue{}}
	for _, is := range novelgate.ChapterQualityIssues(text) {
		if severityBlocking(is.Severity) {
			ck.S1S2 = append(ck.S1S2, is)
		} else {
			ck.S3 = append(ck.S3, is)
		}
	}
	ck.Converged = score <= target && len(ck.S1S2) == 0
	return ck
}

func clampConvergeTarget(target int) int {
	if target <= 0 {
		return convergeDefaultTargetTaste
	}
	return target
}

// convergeUnits 收敛修补的工作单元（场景制章=每场景一单元；blob 章=整章一单元）。
type convergeUnit struct {
	id      string
	isScene bool
	text    string
}

// convergeCollectUnits 读章节工作单元。
func convergeCollectUnits(pm *project.Manager, chapterNum int) ([]convergeUnit, error) {
	if !pm.IsV4() {
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, fmt.Errorf("读取章节失败: %w", err)
		}
		return []convergeUnit{{id: "", isScene: false, text: content}}, nil
	}
	metas, err := pm.SceneManager(chapterNum).List()
	if err != nil {
		return nil, fmt.Errorf("读取场景列表失败: %w", err)
	}
	units := make([]convergeUnit, 0, len(metas))
	for _, m := range metas {
		sc, rerr := pm.SceneManager(chapterNum).Read(m.ID)
		if rerr != nil {
			continue
		}
		units = append(units, convergeUnit{id: m.ID, isScene: true, text: sc.Content})
	}
	// v4 项目但本章无场景文件（迁移章/手写 blob 章）：回落整章单单元——
	// ReadChapterAsStitch 同款兼容口径，不能把混合态章拒之门外。
	if len(units) == 0 {
		content, cerr := pm.ReadChapter(chapterNum)
		if cerr != nil || strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("本章没有可修补的内容（无场景且无正文）")
		}
		return []convergeUnit{{id: "", isScene: false, text: content}}, nil
	}
	return units, nil
}

// convergeIssuesInstruction 把 S1/S2 清单 + AI 味要点拼成定向修补指令。
func convergeIssuesInstruction(ck ConvergeCheck, target int) string {
	var b strings.Builder
	if len(ck.S1S2) > 0 {
		b.WriteString("以下确定性问题必须修复：\n")
		for _, is := range ck.S1S2 {
			if is.Evidence != "" {
				fmt.Fprintf(&b, "- [%s] %s（%s）\n", is.Code, is.Message, is.Evidence)
			} else {
				fmt.Fprintf(&b, "- [%s] %s\n", is.Code, is.Message)
			}
		}
	}
	if ck.TasteScore > target {
		fmt.Fprintf(&b, "- 当前 AI 味 %d 分（目标 ≤%d）：消除空洞比喻、情绪直述、连接词堆砌与四字格堆叠，长短句自然交错\n", ck.TasteScore, target)
	}
	return strings.TrimRight(b.String(), "\n")
}

// NovelChapterConvergePreview 收敛预检（dry-run 零修补）：当前判据状态+计划轮次。
func (a *writingState) NovelChapterConvergePreview(chapterNum int, targetTaste int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	full, err := pm.ReadChapterAsStitch(chapterNum)
	if err != nil || strings.TrimSpace(full) == "" {
		return nil, fmt.Errorf("读取章节失败（%v）——收敛修补需要已有正文", err)
	}
	ck := convergeCheck(full, targetTaste)
	return map[string]interface{}{
		"chapterNum": chapterNum, "tasteScore": ck.TasteScore,
		"targetTaste": clampConvergeTarget(targetTaste),
		"s1s2":        ck.S1S2, "s3": ck.S3, "converged": ck.Converged,
		"planRounds": convergeDefaultMaxRounds,
	}, nil
}

// NovelChapterConverge 收敛闭环（流式 converge-stream）。三态判停：
//   - converged：达判据即停（不烧多余轮）；
//   - stopped(no-improve)：本轮分数未降且 S1/S2 未减 → **按单元回滚本轮**并停
//     （越磨越糟是真实风险，闭环的第一职责是不把稿子磨坏）；
//   - stopped(max-rounds)：轮次耗尽，剩余项如实报告（不假称收敛）。
//
// 每轮完成后版本库留痕 mode=converge（Original=轮前整章，New=轮后整章——
// 轨迹即版本历史，可逐轮恢复）。已收敛章零轮直答。
func (a *writingState) NovelChapterConverge(chapterNum int, maxRounds int, targetTaste int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI client not ready")
	}
	if maxRounds <= 0 {
		maxRounds = convergeDefaultMaxRounds
	}
	target := clampConvergeTarget(targetTaste)

	beforeFull, err := pm.ReadChapterAsStitch(chapterNum)
	if err != nil || strings.TrimSpace(beforeFull) == "" {
		return nil, fmt.Errorf("读取章节失败（%v）——收敛修补需要已有正文", err)
	}
	if ck := convergeCheck(beforeFull, target); ck.Converged {
		return map[string]interface{}{"started": true, "alreadyConverged": true, "rounds": 0}, nil
	}

	// 审计 P0#4（源 AP1-01）：收敛协程与章节生成同族，必须同样登记/可取消/
	// 计入 chapterGenWG——否则waitGensDone（测试与 Windows T.TempDir 清理的
	// 竞态收口）等不到它，且收敛进行中可再发起同章生成/收敛互踩写文件。
	key := chapterGenKey(chapterNum, "")
	ctx, cancel, err := a.registerChapterGen(key, chapterNum, "")
	if err != nil {
		return nil, err
	}

	a.chapterGenWG.Add(1)
	go func() {
		defer a.chapterGenWG.Done()
		defer a.unregisterChapterGen(key, cancel)
		cur := beforeFull
		startTaste := convergeCheck(beforeFull, target).TasteScore
		for r := 1; r <= maxRounds; r++ {
			if ctx.Err() != nil {
				a.emit(convergeStreamChannel, map[string]interface{}{
					"type": "cancelled", "chapterNum": chapterNum, "round": r,
				})
				return
			}
			ckStart := convergeCheck(cur, target)
			a.emit(convergeStreamChannel, map[string]interface{}{
				"type": "round-start", "chapterNum": chapterNum, "round": r,
				"taste": ckStart.TasteScore, "s1s2": len(ckStart.S1S2),
			})

			newText, applied, unitBefore, cerr := a.convergeRound(ctx, pm, chapterNum, cur, ckStart, target)
			if ctx.Err() != nil {
				a.emit(convergeStreamChannel, map[string]interface{}{
					"type": "cancelled", "chapterNum": chapterNum, "round": r,
				})
				return
			}
			if cerr != nil {
				a.emit(convergeStreamChannel, map[string]interface{}{
					"type": "error", "chapterNum": chapterNum, "round": r, "error": cerr.Error(),
				})
				return
			}
			ckEnd := convergeCheck(newText, target)
			a.emit(convergeStreamChannel, map[string]interface{}{
				"type": "round-done", "chapterNum": chapterNum, "round": r,
				"before": ckStart.TasteScore, "after": ckEnd.TasteScore, "applied": applied,
			})

			if ckEnd.Converged {
				a.saveConvergeVersion(pm, chapterNum, cur, newText, r)
				a.emit(convergeStreamChannel, map[string]interface{}{
					"type": "converged", "chapterNum": chapterNum, "rounds": r,
					"taste": ckEnd.TasteScore, "remainingS3": len(ckEnd.S3),
				})
				return
			}
			if ckEnd.TasteScore >= ckStart.TasteScore && len(ckEnd.S1S2) >= len(ckStart.S1S2) {
				if rbErr := convergeWriteBackUnits(pm, chapterNum, unitBefore); rbErr != nil {
					a.emit(convergeStreamChannel, map[string]interface{}{
						"type": "error", "chapterNum": chapterNum,
						"error": fmt.Sprintf("本轮未改善且回滚失败：%v（当前稿=本轮产物，可从重写历史恢复）", rbErr),
					})
					return
				}
				a.emit(convergeStreamChannel, map[string]interface{}{
					"type": "stopped", "chapterNum": chapterNum, "reason": "no-improve",
					"rounds": r, "taste": ckStart.TasteScore,
					"message": fmt.Sprintf("第 %d 轮修补未降低 AI 味（%d→%d），已回滚到本轮前——剩余问题建议人工定向处理（局部重写/去味）", r, ckStart.TasteScore, ckEnd.TasteScore),
				})
				return
			}
			a.saveConvergeVersion(pm, chapterNum, cur, newText, r)
			cur = newText
		}
		ckFinal := convergeCheck(cur, target)
		a.emit(convergeStreamChannel, map[string]interface{}{
			"type": "stopped", "chapterNum": chapterNum, "reason": "max-rounds",
			"rounds": maxRounds, "taste": ckFinal.TasteScore, "remaining": len(ckFinal.S1S2),
			"message": fmt.Sprintf("已修补 %d 轮：AI 味 %d→%d，仍有 %d 项 S1/S2 信号未清——剩余项建议人工定向处理", maxRounds, startTaste, ckFinal.TasteScore, len(ckFinal.S1S2)),
		})
	}()

	return map[string]interface{}{"started": true, "rounds": maxRounds}, nil
}

// convergeRound 一轮修补：定向 LLM 重写（S1/S2 清单+AI 味要点进指令）+ 句级
// AI 味修补（rewriteUnit 自带安全闸）。返回轮后整章、生效句数与**轮前单元快照**
// （no-improve 回滚按单元写回，不整章硬对齐——场景章边界不可牺牲）。
// ctx=生成链请求级 context（P0#4）：在飞的定向重写随取消中止，单元循环逐
// 单元判停——否则取消后下一单元的句级修补链（rewriteUnit 暂走 a.ctx）会重新
// 发起请求，取消看似无效。ctx 贯通到 rewriteUnit 另刀。
func (a *writingState) convergeRound(ctx context.Context, pm *project.Manager, chapterNum int, full string, ck ConvergeCheck, target int) (string, int, []convergeUnit, error) {
	units, err := convergeCollectUnits(pm, chapterNum)
	if err != nil {
		return full, 0, nil, err
	}
	unitBefore := make([]convergeUnit, len(units))
	copy(unitBefore, units)
	if len(ck.S1S2) == 0 && ck.TasteScore <= target {
		return full, 0, unitBefore, nil // 无可修补项
	}
	instr := convergeIssuesInstruction(ck, target)
	appliedTotal, changed := 0, false
	for i := range units {
		if ctx.Err() != nil {
			return full, appliedTotal, unitBefore, ctx.Err()
		}
		if strings.TrimSpace(units[i].text) == "" {
			continue
		}
		text := units[i].text
		// 步骤1：S1/S2 清单存在 → 整单元定向重写（带单元级安全闸：分须降才采纳）
		if len(ck.S1S2) > 0 {
			if nt, rerr := a.convergeRewriteUnit(ctx, text, instr); rerr == nil && strings.TrimSpace(nt) != "" && nt != text {
				tb, _ := novelstyle.ScoreTextNoRef(text)
				ta, _ := novelstyle.ScoreTextNoRef(nt)
				if ta.Score < tb.Score {
					text, changed = nt, true
				}
			}
		}
		if ctx.Err() != nil {
			// 步骤1→步骤2 之间的判停不可省：步骤1 在飞请求被取消中止后，
			// 若直接落进步骤2，句级修补链（rewriteUnit 走 a.ctx）会重新发起
			// 一个不受取消约束的请求，收敛协程对其「不可取消」。
			return full, appliedTotal, unitBefore, ctx.Err()
		}
		// 步骤2：句级 AI 味修补（自带安全闸：未改善不落盘）；ctx 贯通后取消
		// 也能中止在飞请求（P0#4，S1S2 为空时这是本轮唯一的 LLM 调用）。
		if rw, _, _, applied, uerr := a.rewriteUnit(ctx, chapterNum, text); uerr == nil && applied > 0 {
			text, changed = rw, true
			appliedTotal += applied
		}
		if text != units[i].text {
			units[i].text = text
			if werr := writeBackRewrittenFn(pm, chapterNum, units[i].id, units[i].isScene, units[i].text); werr != nil {
				return full, appliedTotal, unitBefore, fmt.Errorf("写回修补单元失败: %w", werr)
			}
		}
	}
	if !changed {
		return full, appliedTotal, unitBefore, nil
	}
	out, err := pm.ReadChapterAsStitch(chapterNum)
	if err != nil {
		return full, appliedTotal, unitBefore, err
	}
	return out, appliedTotal, unitBefore, nil
}

// convergeRewriteUnit 整单元定向重写（rewrite-chapter 模板 + 定向指令）。
// ctx 为生成链请求级 context（P0#4）：取消时在飞请求随 ctx 中止（此前用
// a.ctx——取消后本轮 LLM 请求继续烧完，收敛链对取消无感知）。
func (a *writingState) convergeRewriteUnit(ctx context.Context, text, instruction string) (string, error) {
	tmpl := a.eng.Get("rewrite-chapter")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 rewrite-chapter 模板文件")
	}
	// v4.439：成人向工艺区段随修补注入——收敛修补只修文字层，不得顺手把直白戏
	// 洗成纯爱。pm 缺失（理论上不发生：主链已取过）按非成人向零注入降级。
	matureLevel := ""
	if pm := a.getPM(); pm != nil && pm.Meta != nil {
		matureLevel = pm.Meta.Mature
	}
	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return "", fmt.Errorf("未找到可用模型（可能离线）")
	}
	if ctx == nil {
		ctx = a.ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	reply, err := a.clientRef().ChatSimpleStreamWithOptions(ctx, model,
		tmpl.BuildSystemPrompt(""), tmpl.BuildUserPrompt(map[string]string{
			"chapter_content":          text,
			"modification_instruction": instruction + "\n（保持情节、人物与篇幅不变，只修文字层问题）",
			"prev_summary":             "",
			"mature_craft":             maturecraft.CraftSection(matureLevel),
		}), ai.ChatSimpleOptions{EngineID: eng, Feature: "novel", Temperature: 0.7, MaxTokens: 8192, TimeoutMinutes: 10})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(reply), nil
}

// convergeWriteBackUnits 按单元写回轮前文本（no-improve 回滚）。
func convergeWriteBackUnits(pm *project.Manager, chapterNum int, units []convergeUnit) error {
	for i := range units {
		if werr := writeBackRewrittenFn(pm, chapterNum, units[i].id, units[i].isScene, units[i].text); werr != nil {
			return werr
		}
	}
	return nil
}

// saveConvergeVersion 每轮完成的版本留痕（mode=converge，轨迹=版本历史）。
func (a *writingState) saveConvergeVersion(pm *project.Manager, chapterNum int, before, after string, round int) {
	v := &types.RewriteVersion{
		ChapterNum:        chapterNum,
		Mode:              types.RewriteModeConverge,
		Status:            types.RewriteCompleted,
		Source:            "custom",
		CustomInstr:       fmt.Sprintf("收敛修补第 %d 轮", round),
		OriginalContent:   before,
		OriginalWordCount: len([]rune(before)),
		NewContent:        after,
		NewWordCount:      len([]rune(after)),
	}
	if err := pm.SaveRewriteVersion(v); err != nil {
		slog.Warn("收敛轮留痕失败（正文已生效，轨迹缺一轮）", "chapter", chapterNum, "round", round, "error", err)
	}
}
