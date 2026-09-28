package novelgate

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/types"
)

// ── 章节计划齐备性 + 跨章去重（刀1 线B，规格 docs/gaea-longform-novel-system-2026-09.md §7.1-3）──
//
// 判据全部确定性、零 IO：调用方（internal/app 的写前预检）负责读盘，本文件只做
// 「这份计划能不能当抓手」的机械判定。硬闸只拦「没有抓手」，**不评判计划质量**
// （§7.3：质量仍由写后门 + 作者判断）。

// PlanMissingCode 整份计划不存在的判据码。
//
// 契约 types/plan_v1.go 的常量集覆盖「六项齐备性 + 跨章重复」；「这一章根本没做计划」
// 是更前置的一态，单独给码便于前端区分「补一份计划」与「补某个字段」两种入口。
const PlanMissingCode = "plan_missing"

// PlanContractIssues 计划齐备性 + 跨章关键事件去重（确定性，零 IO）。
//
// plan 为 nil → 返回单条 S1（"本章尚未制定计划"）。
// others 为其它章节的计划（用于跨章 key_events 去重；不含本章自身）。去重比对用
// 归一化口径（去空白 + 全半角统一 + 小写），命名重复章号取 other.SubIndex
// （ChapterPlan 里唯一的全局序字段；<=0 时退化为「其它章」不编造章号）。
//
// 严重度：缺计划/关键事件不足/跨章重复 = 阻断级（S1/S2），缺情绪基调/角色焦点 = 提示
// 级（S3）——与 types.PlanProblem 注释「S1|S2 视为阻断；S3|S4 仅提示」一致。
func PlanContractIssues(chapterNum int, plan *types.ChapterPlan, others []types.ChapterPlan) []types.PlanProblem {
	_ = chapterNum // 保留章号入参：消息不含章号（调用方错误文本已带章号），签名与线C 绑定对其
	if plan == nil {
		return []types.PlanProblem{{
			Code:     PlanMissingCode,
			Severity: "S1",
			Message: "本章尚未制定计划：先补章节计划——叙事目标 · 关键事件（≥2 条）· 冲突类型 · " +
				"结尾类型 · 情绪基调 · 角色焦点，缺一项模型就少一分抓手",
		}}
	}

	var out []types.PlanProblem

	if strings.TrimSpace(plan.NarrativeGoal) == "" {
		out = append(out, types.PlanProblem{
			Code:     types.PlanProblemMissingGoal,
			Severity: "S2",
			Message: "本章计划缺少「叙事目标」：一句话写清这一章为什么存在（如「拿到残符并暴露内鬼」），" +
				"否则模型不知道这章要完成什么",
		})
	}

	// 空白条目不算关键事件（[" ", ""] 与「0 条」同罪）。
	if n := countMeaningfulEvents(plan.KeyEvents); n < types.PlanMinKeyEvents {
		out = append(out, types.PlanProblem{
			Code:     types.PlanProblemMissingKeyEvents,
			Severity: "S2",
			Message: fmt.Sprintf("本章计划的关键事件不足（当前 %d 条，至少 %d 条）：逐条列出本章必须落地的事件，"+
				"写完后正文要能一一对上", n, types.PlanMinKeyEvents),
			Evidence: strings.Join(meaningfulEvents(plan.KeyEvents), " / "),
		})
	}

	if strings.TrimSpace(plan.ConflictType) == "" {
		out = append(out, types.PlanProblem{
			Code:     types.PlanProblemMissingConflict,
			Severity: "S2",
			Message: "本章计划缺少「冲突类型」：写明谁与什么冲突（人与环境 / 人际冲突 / 内心冲突 / 人与规则），" +
				"没有冲突的章不值得写",
		})
	}

	if strings.TrimSpace(plan.EndingType) == "" {
		out = append(out, types.PlanProblem{
			Code:     types.PlanProblemMissingEnding,
			Severity: "S2",
			Message: "本章计划缺少「结尾类型」：从 悬念 / 冲突升级 / 情节转折 / 情感收尾 / 自然过渡 中选一个，" +
				"它决定本章最后一段往哪儿落",
		})
	}

	if strings.TrimSpace(plan.EmotionalTone) == "" {
		out = append(out, types.PlanProblem{
			Code:     types.PlanProblemMissingEmotion,
			Severity: "S3",
			Message:  "本章计划缺少「情绪基调」：一句话写清情绪走向（如「压抑递进，末尾松一口气」），便于节奏控制",
		})
	}

	if len(meaningfulEvents(plan.CharacterFocus)) == 0 {
		out = append(out, types.PlanProblem{
			Code:     types.PlanProblemMissingCharacters,
			Severity: "S3",
			Message:  "本章计划缺少「角色焦点」：列出本章的视角/核心角色（至少 1 个），避免镜头乱飘",
		})
	}

	out = append(out, duplicateEventProblems(plan.KeyEvents, others)...)
	return out
}

// duplicateEventProblems 逐条比对本章关键事件与其它章的计划：归一化后相等即 S1，
// 消息点名重复章号（第 N 章重复：<事件>），Evidence 放本章事件原文。
// 同一条事件对多个章重复只报一次（避免噪声盖过真问题）。
func duplicateEventProblems(events []string, others []types.ChapterPlan) []types.PlanProblem {
	var out []types.PlanProblem
	for _, ev := range meaningfulEvents(events) {
		norm := normalizePlanEvent(ev)
		if norm == "" {
			continue
		}
		for _, other := range others {
			if !containsNormalizedEvent(other.KeyEvents, norm) {
				continue
			}
			out = append(out, types.PlanProblem{
				Code:     types.PlanProblemEventDuplicated,
				Severity: "S1",
				Message:  duplicateEventMessage(other, ev),
				Evidence: ev,
			})
			break
		}
	}
	return out
}

// duplicateEventMessage 组装跨章重复消息：有章号点名章号，无章号不编造。
func duplicateEventMessage(other types.ChapterPlan, ev string) string {
	if other.SubIndex > 0 {
		return fmt.Sprintf("关键事件与第%d章重复：%s——跨章关键事件不得重复，请改写这一条或与那一章合并", other.SubIndex, ev)
	}
	return fmt.Sprintf("关键事件与其它章重复：%s——跨章关键事件不得重复，请改写或合并", ev)
}

// containsNormalizedEvent 判断 events 中是否存在归一化后等于 norm 的条目。
func containsNormalizedEvent(events []string, norm string) bool {
	for _, ev := range events {
		if normalizePlanEvent(ev) == norm {
			return true
		}
	}
	return false
}

// meaningfulEvents 过滤空白条目并保留原文（判重与渲染共用同一口径）。
func meaningfulEvents(items []string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s := strings.TrimSpace(it); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// countMeaningfulEvents 非空白条目数（PlanMinKeyEvents 口径）。
func countMeaningfulEvents(items []string) int {
	return len(meaningfulEvents(items))
}

// planEventPunctFold 中日韩专属标点折叠：全角 ASCII（U+FF01-U+FF5E）由
// normalizePlanEvent 统一减 0xFEE0，这里只补 CJK 自己的标点。
var planEventPunctFold = map[rune]rune{
	'。': '.',
	'、': ',',
	'「': '"',
	'」': '"',
}

// normalizePlanEvent 关键事件归一化：去空白（含全角空格）+ 全角转半角 + CJK 标点折叠 +
// 小写。跨章去重在这个口径上判等，所以「雨夜 夺符！」与「雨夜夺符!」视为同一事件。
func normalizePlanEvent(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if r >= 0xFF01 && r <= 0xFF5E {
			r -= 0xFEE0
		} else if m, ok := planEventPunctFold[r]; ok {
			r = m
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
