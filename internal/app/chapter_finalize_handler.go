package app

// ── 章节定稿回写（阶段八 8.4）────────────────────────────────
//
// 定稿 = 作者宣布本章正文为最终版的显式动作。与生成后自动门（runAutoGateAfter-
// Generation，基于生成时瞬时正文、只跑主线）的三点差异：
//   ① 基于定稿时的当前正文重新提取——作者改稿后自动门的结果已漂移；
//   ② 建议制：AI 草案 → 作者确认 → 才落库（红线延伸；草案经 ExtractOnly 纯提取
//      零落盘，不走 Analyze 的五路落盘）；
//   ③ 阶段合账落盘缓存——阶段收官章定稿时合成上一阶段总结合账，阶段开篇章生成
//      直接读缓存，不再每次重烧 LLM。
//
// 三件套（阶段摘要/角色状态/伏笔状态）对齐既有面板：角色状态=Progressions（8.2
// 状态卡，角色抽屉「状态演进」可见）；伏笔状态=foreshadows.json 登记表（复用
// analysisAgent.SyncForeshadows 幂等同步）；阶段摘要=stage-recaps.json 缓存。
//
// 分支章口径（与自动门 Q3 同律，登记表/合账/状态卡都是主线口径）：分支章定稿
// 只写定稿档（标记+留痕），三件套不落——Progressions 锚点是纯章号，分支条目会
// 污染主线投影；诚实申报在 Skipped 字段。
//
// 事务性：三件逐件尽力落库，失败不回滚已成功件、不阻塞定稿标记（final.json
// 永远落得下），失败件与原因如实进 Errors 留痕——作者可重试，成功件重放有幂等
// 保护（Progressions 按 Note 标记整批替换，伏笔按 stable_id 幂等 upsert）。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// finalizeProgressionNote 定稿回写的 Progression 备注标记（幂等键）：重定稿同
// 章先移除上一批同标记条目再追加，作者改稿重定不重复堆积。作者可见（备注不进
// prompt），文案自明。
func finalizeProgressionNote(chapterNum int) string {
	return fmt.Sprintf("第%d章定稿回写", chapterNum)
}

// isStageCloser 阶段收官章（第 10/20/… 章）：定稿时上一阶段素材齐了，合成合账
// 落缓存。第 1 章是全书开篇不算（stageLength=10，第 10 章是首阶段收官）。
func isStageCloser(chapterNum int) bool {
	return chapterNum > 0 && chapterNum%stageLength == 0
}

// stageCloserRange 收官章对应合账区间 [start, end]（第 10 章 → 1~10，与
// stageRange(11) 的开篇读取区间同一真相）。
func stageCloserRange(chapterNum int) (int, int) {
	end := chapterNum
	start := end - stageLength + 1
	if start < 1 {
		start = 1
	}
	return start, end
}

// finalizeCharDraft 角色状态草案条目（前端确认面板渲染形态）。
type finalizeCharDraft struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Field ∈ 8.2 白名单（personality/background/motivation/status），空=status。
	Field string `json:"field"`
	Text  string `json:"text"`
	Note  string `json:"note,omitempty"`
}

// FinalizeChapterDraft 定稿草案（**不落盘**）：基于当前正文纯提取三件套建议。
// 已定稿章返回 finalized=true + 既有定稿档（前端展示「重新定稿」语义）。无
// client / 无正文 / 坏分支如实报错。
func (a *writingState) FinalizeChapterDraft(chapterNum int, branch string) (map[string]interface{}, error) {
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号无效（%d）：定稿按章号索引，需为正整数", chapterNum)
	}
	if !validPlanBranch(branch) {
		return nil, fmt.Errorf("分支字母无效（%q）：仅支持主线（空）与 a/b/c", branch)
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	content := ""
	var err error
	if branch != "" {
		content, err = pm.ReadChapterBranch(chapterNum, branch)
	} else {
		content, err = pm.ReadChapter(chapterNum)
	}
	if err != nil || strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("第 %d%s 章正文为空或读取失败：空章不能定稿", chapterNum, branch)
	}

	out := map[string]interface{}{
		"chapterNum": chapterNum,
		"branch":     branch,
		"finalized":  false,
		"warnings":   []string{},
	}

	// 已定稿：幂等回带既有档。查看定稿状态零模型依赖（无 client 时到此为止，
	// 不做重新提取——重新定稿草案需要模型，由前端按 finalized 标记引导）。
	alreadyDrafted := false
	if cf, ok, gerr := pm.ReadChapterFinalization(chapterNum, branch); gerr == nil && ok && cf != nil {
		out["finalized"] = true
		out["finalization"] = cf
		alreadyDrafted = true
	}
	// 优先消费生成时过闸的章级变更声明（阶段八 8.5）：生成时模型顺手声明→五类
	// 门禁→随摘要落盘，定稿草案从这里零 LLM 预填充（角色+伏笔两件）。缺失/空
	// 草案才走下方 ExtractOnly 现提取（模型在写正文时最清楚改了什么，事后提取
	// 是兜底不是主路径）。声明路径零模型依赖——无 client 也照出草案。
	if declOut := draftFromDeclaration(pm, chapterNum, branch, out); declOut != nil {
		return declOut, nil
	}
	if a.analysisAgent == nil || a.clientRef() == nil {
		if alreadyDrafted {
			return out, nil
		}
		return nil, fmt.Errorf("AI 客户端未就绪：请先在「模型中心」配置可用引擎，再生成定稿草案")
	}

	// 纯提取（零落盘——草案未经作者确认，建议制红线）
	v2, err := a.analysisAgent.ExtractOnly(a.ctx, chapterNum, content)
	if err != nil {
		return nil, fmt.Errorf("定稿草案提取失败（模型不可用或返回不合法）：%w", err)
	}

	// 角色状态草案：V2 差分 → Progression 条目（按名对齐角色卡 ID；未命中进
	// warnings 诚实标注，不虚构角色）
	charDrafts := make([]finalizeCharDraft, 0, len(v2.CharacterStates))
	warnings := []string{}
	if cf, cerr := pm.ReadCharacters(); cerr == nil && cf != nil {
		byName := map[string]string{}
		for i := range cf.Characters {
			byName[cf.Characters[i].Name] = cf.Characters[i].ID
		}
		for _, cs := range v2.CharacterStates {
			name := strings.TrimSpace(cs.Name)
			id, ok := byName[name]
			if !ok {
				warnings = append(warnings, fmt.Sprintf("角色「%s」不在本书角色卡中，该条未纳入草案", name))
				continue
			}
			text := strings.TrimSpace(cs.NewState)
			if text == "" {
				continue
			}
			charDrafts = append(charDrafts, finalizeCharDraft{
				ID: id, Name: name, Field: "status", Text: text,
				Note: finalizeProgressionNote(chapterNum),
			})
		}
	} else {
		warnings = append(warnings, "角色卡读取失败，角色状态草案为空")
	}
	out["characters"] = charDrafts

	// 伏笔草案：V2 命中透传（planted/resolved 由确认面板勾选后经既有同步器落库）
	out["foreshadows"] = v2.Foreshadows

	// 阶段合账草案：仅阶段收官章（上一阶段素材齐备）；分支章不合账（主线口径）
	if branch == "" && isStageCloser(chapterNum) {
		start, end := stageCloserRange(chapterNum)
		digest := buildStageDigest(pm, start, end)
		if digest != "" {
			recap := strings.TrimSpace(a.synthesizeStageRecap(start, end, digest))
			out["stageRecap"] = map[string]interface{}{
				"stage":   (chapterNum-1)/stageLength + 1,
				"start":   start,
				"end":     end,
				"text":    recap, // LLM 失败时已是逐章摘要清单回落（与生成链同一降级）
				"fromLLM": recap != "" && recap != digest,
			}
		}
	}

	out["warnings"] = warnings
	return out, nil
}

// finalizeApplyPayload 作者确认后的落库载荷（草案可编辑后回传）。
type finalizeApplyPayload struct {
	Note string `json:"note"`
	// StageRecap 非空 Text=应用合账；Text 空/nil=不应用（含非收官章）。
	StageRecap *struct {
		Stage int    `json:"stage"`
		Start int    `json:"start"`
		End   int    `json:"end"`
		Text  string `json:"text"`
	} `json:"stage_recap"`
	Characters  []finalizeCharDraft   `json:"characters"`
	Foreshadows []types.ForeshadowHit `json:"foreshadows"`
}

// FinalizeChapterApply 作者确认落库：三件套事务性回写 + 定稿档留痕。分支章只
// 写定稿档（Skipped 申报）。任何一件失败不阻塞定稿，回执如实。
func (a *writingState) FinalizeChapterApply(chapterNum int, branch, payloadJSON string) (map[string]interface{}, error) {
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号无效（%d）：定稿按章号索引，需为正整数", chapterNum)
	}
	if !validPlanBranch(branch) {
		return nil, fmt.Errorf("分支字母无效（%q）：仅支持主线（空）与 a/b/c", branch)
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	content := ""
	var err error
	if branch != "" {
		content, err = pm.ReadChapterBranch(chapterNum, branch)
	} else {
		content, err = pm.ReadChapter(chapterNum)
	}
	if err != nil || strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("第 %d%s 章正文为空或读取失败：空章不能定稿", chapterNum, branch)
	}

	var payload finalizeApplyPayload
	if s := strings.TrimSpace(payloadJSON); s != "" {
		if err := json.Unmarshal([]byte(s), &payload); err != nil {
			return nil, fmt.Errorf("定稿载荷解析失败: %w", err)
		}
	}

	final := &types.ChapterFinalization{
		Version:     types.ChapterFinalizationVersion,
		ChapterNum:  chapterNum,
		Branch:      branch,
		FinalizedAt: time.Now().Format(time.RFC3339),
		Note:        strings.TrimSpace(payload.Note),
	}

	// 分支章：只标记+申报（登记表/合账/状态卡主线口径，Progressions 章号锚点
	// 无分支维度——分支条目会污染主线投影）
	if branch != "" {
		final.Skipped = "分支章定稿：角色状态卡/伏笔登记表/阶段合账均为主线口径，三件套不落库（分支只标记定稿）"
		if werr := pm.WriteChapterFinalization(final); werr != nil {
			return nil, fmt.Errorf("定稿档落盘失败: %w", werr)
		}
		return map[string]interface{}{
			"finalized": true, "applied": nil, "errors": []string{},
			"skipped": final.Skipped, "finalization": final,
		}, nil
	}

	applied := &types.FinalizeApplied{Characters: 0}
	errs := []string{}

	// ── 件一：角色状态 → Progressions（8.2 状态卡；Note 标记整批替换幂等）──
	if len(payload.Characters) > 0 {
		n, cerr := a.applyFinalizeProgressions(pm, chapterNum, payload.Characters)
		if cerr != nil {
			errs = append(errs, cerr.Error())
			slog.Warn("定稿回写角色件失败", "chapter", chapterNum, "error", cerr)
		} else {
			applied.Characters = n
		}
	}

	// ── 件二：伏笔状态 → 登记表（复用既有幂等同步器）──
	if len(payload.Foreshadows) > 0 && a.analysisAgent != nil {
		res := a.analysisAgent.SyncForeshadows(chapterNum, payload.Foreshadows)
		applied.Foreshadows = &types.FinalizeForeshadowSync{
			PlantedCount:  res.PlantedCount,
			ResolvedCount: res.ResolvedCount,
			CreatedCount:  res.CreatedCount,
			Errors:        res.Errors,
		}
		for _, e := range res.Errors {
			errs = append(errs, "伏笔同步: "+e)
		}
	}

	// ── 件三：阶段合账 → 缓存（阶段开篇章生成直接读，不再重烧）──
	if payload.StageRecap != nil && strings.TrimSpace(payload.StageRecap.Text) != "" {
		stage := payload.StageRecap.Stage
		if stage < 1 {
			stage = (chapterNum-1)/stageLength + 1
		}
		entry := types.StageRecapEntry{
			Start: payload.StageRecap.Start, End: payload.StageRecap.End,
			Text:        strings.TrimSpace(payload.StageRecap.Text),
			FinalizedAt: final.FinalizedAt, SourceChapter: chapterNum,
		}
		if entry.Start < 1 {
			entry.Start, entry.End = stageCloserRange(chapterNum)
		}
		if uerr := pm.UpsertStageRecap(stage, entry); uerr != nil {
			errs = append(errs, "阶段合账落盘失败: "+uerr.Error())
		} else {
			applied.StageRecap = &entry
		}
	}

	final.Applied = applied
	final.Errors = errs
	if werr := pm.WriteChapterFinalization(final); werr != nil {
		// 定稿档本身落盘失败：三件可能已落，如实报错（作者重试有幂等保护）
		return nil, fmt.Errorf("定稿档落盘失败（三件套回执未留痕，可重试）: %w", werr)
	}
	slog.Info("章节定稿完成", "chapter", chapterNum,
		"characters", applied.Characters, "foreshadows", applied.Foreshadows != nil,
		"stageRecap", applied.StageRecap != nil, "errors", len(errs))

	return map[string]interface{}{
		"finalized":    true,
		"applied":      applied,
		"errors":       errs,
		"finalization": final,
	}, nil
}

// applyFinalizeProgressions 角色件落库：对每个草案条目，先移除同角色同章的上一
// 批定稿回写条目（Note 标记幂等）再追加。读-改-写整表，一次落盘。
func (a *writingState) applyFinalizeProgressions(pm *project.Manager, chapterNum int, drafts []finalizeCharDraft) (int, error) {
	cf, err := pm.ReadCharacters()
	if err != nil {
		return 0, fmt.Errorf("读取角色失败: %w", err)
	}
	if cf == nil {
		return 0, fmt.Errorf("项目角色为空")
	}
	byID := map[string]*types.Character{}
	for i := range cf.Characters {
		byID[cf.Characters[i].ID] = &cf.Characters[i]
	}
	mark := finalizeProgressionNote(chapterNum)
	applied := 0
	for _, d := range drafts {
		c := byID[d.ID]
		if c == nil {
			return applied, fmt.Errorf("角色不存在: %s", d.ID)
		}
		text := strings.TrimSpace(d.Text)
		if text == "" {
			continue
		}
		field := strings.TrimSpace(d.Field)
		if field == "" {
			field = "status"
		}
		// 幂等：移除上一批同章回写条目（按 Note 精确匹配；作者手删过则 no-op）
		kept := c.Progressions[:0:0]
		for _, p := range c.Progressions {
			if p.Note != mark {
				kept = append(kept, p)
			}
		}
		note := strings.TrimSpace(d.Note)
		if note == "" {
			note = mark // 面板清空备注也保底带标记（幂等键不可丢）
		}
		kept = append(kept, types.CharacterProgression{
			Chapter: chapterNum, Field: field, Text: text, Note: note,
		})
		c.Progressions = kept
		applied++
	}
	if applied > 0 {
		if err := pm.WriteCharacters(cf); err != nil {
			return 0, fmt.Errorf("保存角色失败: %w", err)
		}
	}
	return applied, nil
}

// ChapterFinalizationGet 定稿状态查询（未定稿 = nil,nil 正常态）。
func (a *writingState) ChapterFinalizationGet(chapterNum int, branch string) (*types.ChapterFinalization, error) {
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号无效（%d）", chapterNum)
	}
	if !validPlanBranch(branch) {
		return nil, fmt.Errorf("分支字母无效（%q）", branch)
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	cf, ok, err := pm.ReadChapterFinalization(chapterNum, branch)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return cf, nil
}

// draftFromDeclaration 从章节摘要里过闸的变更声明草案化（零 LLM）。返回 nil=
// 无过闸声明或草案两件全空（调用方回落 ExtractOnly 现提取）。
func draftFromDeclaration(pm *project.Manager, chapterNum int, branch string, out map[string]interface{}) map[string]interface{} {
	var summary *types.ChapterSummary
	var err error
	if branch != "" {
		summary, err = pm.ReadChapterBranchSummary(chapterNum, branch)
	} else {
		summary, err = pm.ReadChapterSummary(chapterNum)
	}
	if err != nil || summary == nil || summary.DeclChanges == nil {
		return nil
	}
	decl := summary.DeclChanges
	charDrafts := make([]finalizeCharDraft, 0, len(decl.CharacterStates))
	if cf, cerr := pm.ReadCharacters(); cerr == nil && cf != nil {
		for _, st := range decl.CharacterStates {
			name := strings.TrimSpace(st.Name)
			c := types.FindCharacterByName(cf.Characters, name)
			if c == nil || strings.TrimSpace(st.New) == "" {
				continue // 门禁已过闸，此处防御性对齐（作者改卡后名字可能漂移）
			}
			charDrafts = append(charDrafts, finalizeCharDraft{
				ID: c.ID, Name: c.Name, Field: "status", Text: strings.TrimSpace(st.New),
				Note: finalizeProgressionNote(chapterNum),
			})
		}
	}
	fsDrafts := make([]types.ForeshadowHit, 0, len(decl.Foreshadows))
	for _, f := range decl.Foreshadows {
		if strings.TrimSpace(f.Action) == "" {
			continue
		}
		id := strings.TrimSpace(f.ID)
		fsDrafts = append(fsDrafts, types.ForeshadowHit{
			Title: f.Title, Content: f.Description, Type: strings.ToLower(f.Action),
			ReferenceStableID: id,
		})
	}
	if len(charDrafts) == 0 && len(fsDrafts) == 0 {
		return nil
	}
	out["characters"] = charDrafts
	out["foreshadows"] = fsDrafts
	out["fromDeclaration"] = true
	return out
}
