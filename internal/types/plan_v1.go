package types

// ── 章节计划闭环（v4.422.0，规格 docs/gaea-longform-novel-system-2026-09.md §7）──
//
// 论点：写一章时模型的"意图输入"此前只有一句自由文本 plotReq（模板槽位
// plot_req），承载意图的 ChapterPlan（plot_v2.go:11-22）自 v4.278 落库后零消费。
// 本组契约把「这一章为什么存在」落成可落盘、可审批、可硬闸、可回写的数据：
//   plans.json（ChapterPlanFile）→ 生成前预检（PlanGateReport）→ 写后偏差（PlanDeviation）。
// 全部判据确定性；AI 只负责**提案**，落盘一律经作者审批（作者是上帝）。

// ChapterPlanFile chapters/plans.json 落盘结构。
//
// key 用十进制章节号字符串（而非 int）：JSON 稳定、与既有 chapters/NNN.md 命名口径
// 一致，且便于跨章去重时按序遍历。缺失文件=空正常态；损坏文件**不覆盖**（调用方报错）。
type ChapterPlanFile struct {
	Version int                    `json:"version"`
	Plans   map[string]ChapterPlan `json:"plans"`
}

// PlanProblem 计划问题单条（确定性判据；与 novelgate.Issue 同形但不跨包依赖，
// 避免 types ↔ novelgate 环依赖）。
type PlanProblem struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // S1|S2 视为阻断；S3|S4 仅提示
	Message  string `json:"message"`
	Evidence string `json:"evidence,omitempty"`
}

// 计划齐备性判据（规格 §7.1-3）：六项齐备即可生成；缺任一即硬闸。
//
// 码表是**契约面**（前端按 code 分类显示）；`PlanProblemMissingPlan` 与
// novelgate.PlanMissingCode 同值（后者是硬闸实现位置，两处必须一致）。
const (
	PlanProblemMissingPlan       = "plan_missing" // 尚无本章计划（S1，硬闸主判据）
	PlanProblemMissingGoal       = "plan_goal_empty"
	PlanProblemMissingKeyEvents  = "plan_key_events_insufficient"
	PlanProblemMissingConflict   = "plan_conflict_empty"
	PlanProblemMissingEnding     = "plan_ending_empty"
	PlanProblemMissingEmotion    = "plan_emotion_empty"
	PlanProblemMissingCharacters = "plan_focus_empty"
	PlanProblemEventDuplicated   = "plan_event_duplicated"
)

// PlanMinKeyEvents 关键事件下限（types.ChapterPlan 注释口径为 2-4 条）。
const PlanMinKeyEvents = 2

// PlanGateReport 生成前预检结果（硬闸唯一判据来源）。
type PlanGateReport struct {
	ChapterNum    int           `json:"chapterNum"`
	Allowed       bool          `json:"allowed"` // 是否允许生成（= !Blocking）
	HasPlan       bool          `json:"hasPlan"`
	Missing       []string      `json:"missing,omitempty"`       // 缺失齐备性字段的中文标签
	PlanProblems  []PlanProblem `json:"planProblems,omitempty"`  // 计划自身问题（齐备性 + 跨章重复）
	OutlineIssues []PlanProblem `json:"outlineIssues,omitempty"` // 大纲节点契约问题（既有 novelgate 语义）
	Blocking      bool          `json:"blocking"`                // 存在 S1/S2 即 true
}

// PlanDeviation 计划 vs 实际的偏差报告（写后回写；缺失/未分析均为正常态）。
type PlanDeviation struct {
	ChapterNum      int      `json:"chapterNum"`
	HasPlan         bool     `json:"hasPlan"`
	Analyzed        bool     `json:"analyzed"`
	MissingEvents   []string `json:"missingEvents,omitempty"` // 计划列出、实际分析未命中
	EndingMismatch  bool     `json:"endingMismatch"`          // 结尾类型与计划不符
	PlannedEnding   string   `json:"plannedEnding,omitempty"`
	ActualEnding    string   `json:"actualEnding,omitempty"`
	EmotionDrift    bool     `json:"emotionDrift"` // 情绪走向与计划不符
	PlannedEmotion  string   `json:"plannedEmotion,omitempty"`
	ActualEmotion   string   `json:"actualEmotion,omitempty"`
	DuplicateEvents []string `json:"duplicateEvents,omitempty"` // 与其它章重复的关键事件
	Summary         string   `json:"summary,omitempty"`         // 一句话偏差概要
	NextSuggestion  string   `json:"nextSuggestion,omitempty"`  // 下一章计划建议（仅建议，不落盘）
}
