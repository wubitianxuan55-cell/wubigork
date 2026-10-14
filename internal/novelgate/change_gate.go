package novelgate

// ── 章级变更声明门禁（阶段八 8.5）────────────────────────────
//
// 生成章时模型按 ---CHAPTER_CHANGES--- 输出结构化声明（人物状态/伏笔/时间线），
// 本文件做**确定性**五类校验（零 LLM）：
//   1. unknown_entity   未知实体——声明的角色名不在角色卡（名字/别名，含排除词剔除）
//   2. missing_reference 引用存在——resolved 必须挂回登记表既有条目且未回收过
//   3. state_regression 状态连续——角色水位章号（存活/心理）> 本章号时声明变更=回退
//   4. cast_mismatch    出场对账——声明变更的角色在正文零提及（名字+别名）
//   5. invalid_action   伏笔动作——action 枚举外 / planted 缺描述
//
// 口径：五类核心判据全 S2（拒收级——声明不进落库链），timeline 缺失 S3（提示不
// 拒收）。**正文永不因声明被拒收而失效**（正文是主产物；拒收的是声明的落库）。
// 过闸声明随章节摘要落盘，作为定稿回写（8.4）草案的预填充。

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// ChangeDeclarationIssues 变更声明五类校验（确定性、零 IO）。
//   - decl：nil/零字段=无声明（返回空，调用方按「未声明」软降级，不算违规）
//   - chars：角色卡全量（含别名/排除词）
//   - ffs：伏笔登记表全量（resolved 引用对账）
//   - content：本章正文（出场对账语料）
//   - chapterNum：本章章号（水位连续性基准；≤0 跳过水位类检查——无章号语境）
func ChangeDeclarationIssues(decl *types.ChapterChangeDeclaration, chars []types.Character,
	ffs []types.Foreshadow, chapterNum int, content string) []Issue {
	if decl == nil {
		return nil
	}
	issues := make([]Issue, 0, 4)
	cleanContent := types.StripExcludeHits(content, chars)
	byID := make(map[string]*types.Foreshadow, len(ffs))
	for i := range ffs {
		byID[ffs[i].ID] = &ffs[i]
	}

	// ── 件一：角色状态声明 ──
	for _, st := range decl.CharacterStates {
		name := strings.TrimSpace(st.Name)
		c := types.FindCharacterByName(chars, name)
		if c == nil {
			issues = append(issues, Issue{Code: "decl_unknown_entity", Severity: "S2",
				Message:  fmt.Sprintf("变更声明的角色「%s」不在本书角色卡中（名字或别名均未命中）", name),
				Evidence: name})
			continue // 未知实体：水位/出场两查无从谈起
		}
		// ⑤ 出场对账：声明变更但正文零提及（名字+别名，剔除排除词后）
		if !types.CharacterDetected(cleanContent, *c) {
			issues = append(issues, Issue{Code: "decl_cast_mismatch", Severity: "S2",
				Message:  fmt.Sprintf("角色「%s」被声明状态变更，但本章正文未提及此人（名字或别名零命中）", name),
				Evidence: name})
		}
		// ③ 状态连续：水位章号 > 本章号 = 回退改写（重跑低章不得覆盖高章状态）
		if chapterNum > 0 {
			if strings.EqualFold(strings.TrimSpace(st.Kind), "survival") {
				if !types.SurvivalStatusAdvanceAllowed(*c, chapterNum) && c.StatusChangedChapter > chapterNum {
					issues = append(issues, Issue{Code: "decl_state_regression", Severity: "S2",
						Message: fmt.Sprintf("角色「%s」的存活状态已在第 %d 章变更，本章（第 %d 章）不得回退改写", name, c.StatusChangedChapter, chapterNum),
						Evidence: fmt.Sprintf("水位=%d > 本章=%d", c.StatusChangedChapter, chapterNum)})
				}
			} else if !types.StateAdvanceAllowed(*c, chapterNum) && c.StateUpdatedChapter > chapterNum {
				issues = append(issues, Issue{Code: "decl_state_regression", Severity: "S2",
					Message: fmt.Sprintf("角色「%s」的心理状态已在第 %d 章变更，本章（第 %d 章）不得回退改写", name, c.StateUpdatedChapter, chapterNum),
					Evidence: fmt.Sprintf("水位=%d > 本章=%d", c.StateUpdatedChapter, chapterNum)})
			}
		}
	}

	// ── 件二：伏笔声明 ──
	for _, f := range decl.Foreshadows {
		action := strings.ToLower(strings.TrimSpace(f.Action))
		switch action {
		case "planted":
			if strings.TrimSpace(f.Description) == "" {
				issues = append(issues, Issue{Code: "decl_invalid_action", Severity: "S2",
					Message:  fmt.Sprintf("伏笔「%s」声明埋设（planted）但缺描述——埋设必须说明埋了什么", truncEvidence(f.Title)),
					Evidence: f.ID})
			}
		case "hinted":
			// 提示（hinted）无前置约束：可指向已埋条目也可泛指（门禁从宽——
			// 提示不改登记表状态机，仅作者参考）
		case "resolved":
			id := strings.TrimSpace(f.ID)
			if id == "" {
				issues = append(issues, Issue{Code: "decl_missing_reference", Severity: "S2",
					Message:  fmt.Sprintf("伏笔「%s」声明回收（resolved）但未引用既有条目 ID——回收必须挂回登记表", truncEvidence(f.Title)),
					Evidence: ""})
				continue
			}
			existing := byID[id]
			if existing == nil {
				issues = append(issues, Issue{Code: "decl_missing_reference", Severity: "S2",
					Message:  fmt.Sprintf("回收声明引用的伏笔 ID %q 不在登记表中（引用存在校验失败）", id),
					Evidence: id})
				continue
			}
			if existing.RevealedIn != "" || existing.Status == types.ForeshadowRevealed {
				issues = append(issues, Issue{Code: "decl_missing_reference", Severity: "S2",
					Message:  fmt.Sprintf("伏笔「%s」已在第 %s 章回收，不得重复回收", truncEvidence(existing.Title), chapterFileOf(existing.RevealedIn)),
					Evidence: id})
			}
		default:
			issues = append(issues, Issue{Code: "decl_invalid_action", Severity: "S2",
				Message:  fmt.Sprintf("伏笔动作 %q 不在 planted/hinted/resolved 三值内", f.Action),
				Evidence: f.ID})
		}
	}

	// ── 时间线：缺失提示（不拒收）──
	if strings.TrimSpace(decl.Timeline) == "" {
		issues = append(issues, Issue{Code: "decl_timeline_missing", Severity: "S3",
			Message: "变更声明缺时间线推进（timeline）——建议补一句本章时间推进到哪"})
	}
	return issues
}

// DeclBlockingIssues 过闸判定：存在 S2 即拒收（声明不进落库链）。
func DeclBlockingIssues(issues []Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, it := range issues {
		if it.Severity == "S2" {
			out = append(out, it)
		}
	}
	return out
}

func truncEvidence(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > 20 {
		return string(r[:20]) + "…"
	}
	return s
}

func chapterFileOf(file string) string {
	file = strings.TrimSuffix(strings.TrimSuffix(file, ".md"), "")
	return strings.TrimSpace(file)
}
