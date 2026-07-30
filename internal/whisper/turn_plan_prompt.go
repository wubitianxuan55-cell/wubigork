// Package whisper — turn_plan_prompt.go
// 100% 对齐 ackem shared/turnPlan.ts + shared/taskFrame.ts
// TurnPlan 统一契约：L0 任务理解中枢，规则先行 → LLM 增强
//
// 核心职责：
// 1. 纯规则检测用户消息的交付形态（散文/表格/列表）和目标（闲聊/列举/对比/解释/推荐）
// 2. 产出 TurnPlan 供 orchestrator 和 desktop-agent 消费
// 3. 决定路由：casual_chat → 轻语对话 / structured_chat → 知识卡 / extension_* → 桌面助手

package whisper

import (
	"regexp"
	"strings"
)

// ─── 类型定义 ──────────────────────────────────────────────────

// TaskGoal 用户期望的信息组织目标
type TaskGoal string

const (
	GoalCasual    TaskGoal = "casual"
	GoalList      TaskGoal = "list"
	GoalCompare   TaskGoal = "compare"
	GoalExplain   TaskGoal = "explain"
	GoalRecommend TaskGoal = "recommend"
)

// TaskDeliveryFormat 交付形态
type TaskDeliveryFormat string

const (
	DeliveryProse         TaskDeliveryFormat = "prose"
	DeliveryMarkdownTable TaskDeliveryFormat = "markdown_table"
	DeliveryBulletList    TaskDeliveryFormat = "bullet_list"
)

// TurnRouting 回合路由目标
type TurnRouting string

const (
	RouteCasualChat       TurnRouting = "casual_chat"
	RouteStructuredChat   TurnRouting = "structured_chat"
	RouteExtensionPlan    TurnRouting = "extension_plan"
	RouteExtensionAskPlan TurnRouting = "extension_ask_plan"
	RouteExtensionInvoke  TurnRouting = "extension_invoke"
)

// UserTaskFrame 用户任务框（L0 任务理解）
type UserTaskFrame struct {
	Goal           TaskGoal           `json:"goal"`
	Delivery       TaskDeliveryFormat `json:"delivery"`
	Subjects       []string           `json:"subjects"`
	NeedsSearch    bool               `json:"needsSearch"`
	SearchQuery    string             `json:"searchQuery,omitempty"`
	MergeWebSearch bool               `json:"mergeWebSearch"`
	FormatHint     string             `json:"formatHint,omitempty"`
	Source         string             `json:"source"` // "rules" | "llm" | "rules+llm"
}

// TaskFrameRuleHint 规则层可同步得到的局部结果
type TaskFrameRuleHint struct {
	Delivery       TaskDeliveryFormat
	Goal           TaskGoal
	MergeWebSearch bool
	NeedsLlmEnrich bool
}

// TurnPlan 回合计划（L0 中枢产出）
type TurnPlan struct {
	Routing             TurnRouting        `json:"routing"`
	Goal                TaskGoal           `json:"goal"`
	Delivery            TaskDeliveryFormat `json:"delivery"`
	Subjects            []string           `json:"subjects"`
	NeedsSearch         bool               `json:"needsSearch"`
	SearchQuery         string             `json:"searchQuery,omitempty"`
	MergeWebSearch      bool               `json:"mergeWebSearch"`
	FormatHint          string             `json:"formatHint,omitempty"`
	PlanTopic           string             `json:"planTopic,omitempty"`
	ExtensionID         string             `json:"extensionId,omitempty"`
	ExtensionConfidence float64            `json:"extensionConfidence,omitempty"`
	Reasoning           string             `json:"reasoning,omitempty"`
	Source              string             `json:"source"`
}

// TurnPlanRulePriors 规则层先验
type TurnPlanRulePriors struct {
	TaskFrameGoal       TaskGoal
	TaskFrameDelivery   TaskDeliveryFormat
	MergeWebSearch      bool
	ExplicitCreate      bool
	BareFeatureCreate   bool
	CapabilityProbe     bool
	ExplicitCreateTopic string
	BareFeatureTopic    string
}

// ─── 正则引擎 ──────────────────────────────────────────────────

var (
	tableFormatRE    = regexp.MustCompile(`列个表|列个.{0,24}表|画个表|画表格|列表|表格|清单|汇总成表|做成表|对照表|对比表|差距表|排成表|制成表`)
	bulletFormatRE   = regexp.MustCompile(`列出来|罗列|分条|逐条|一条一条|条目`)
	compareGoalRE    = regexp.MustCompile(`对比(?:一下)?|对照(?:一下)?|比较(?:一下)?|分别说说|各有什么|都有啥|都有哪些|哪个好|差距表`)
	listGoalRE       = regexp.MustCompile(`有哪些|有什么|都有什么|列举|罗列`)
	explicitSearchRE = regexp.MustCompile(`(?:帮我)?(?:搜|查)(?:一下|一搜|一查)?|联网搜|上网搜|搜索|查找`)
	// DELIVERY_WEAK_SIGNAL_RE — 弱交付信号：规则可能漏判，须交给 LLM / 强制 structured
	deliveryWeakSignalRE = regexp.MustCompile(`表|列个|列出|罗列|对比|对照|清单|分条|差距|排成|汇总|画个`)
)

// ─── 规则检测 ──────────────────────────────────────────────────

// DetectTaskFrameRules 纯规则：从用户原话检测交付形态与目标（无 LLM、无实体词表）
// 100% 对齐 ackem shared/taskFrame.ts detectTaskFrameRules
func DetectTaskFrameRules(userMessage string) TaskFrameRuleHint {
	t := strings.TrimSpace(userMessage)
	if t == "" {
		return TaskFrameRuleHint{
			Delivery:       DeliveryProse,
			Goal:           GoalCasual,
			MergeWebSearch: false,
			NeedsLlmEnrich: false,
		}
	}

	// 检测交付格式
	delivery := DeliveryProse
	if tableFormatRE.MatchString(t) {
		delivery = DeliveryMarkdownTable
	} else if bulletFormatRE.MatchString(t) {
		delivery = DeliveryBulletList
	} else if compareGoalRE.MatchString(t) {
		delivery = DeliveryMarkdownTable
	}

	// 检测目标
	goal := GoalCasual
	if compareGoalRE.MatchString(t) {
		goal = GoalCompare
	} else if listGoalRE.MatchString(t) {
		goal = GoalList
	} else if delivery == DeliveryMarkdownTable {
		goal = GoalCompare
	}

	// 检测是否需要搜索
	needsSearch := explicitSearchRE.MatchString(t)
	mergeWebSearch := needsSearch

	// 是否需要 LLM 增强（弱信号）
	needsLlm := deliveryWeakSignalRE.MatchString(t)

	return TaskFrameRuleHint{
		Delivery:       delivery,
		Goal:           goal,
		MergeWebSearch: mergeWebSearch,
		NeedsLlmEnrich: needsLlm,
	}
}

// BuildTurnPlanRulePriors 从用户消息构建规则层先验
// 100% 对齐 ackem shared/turnPlan.ts buildTurnPlanRulePriors
func BuildTurnPlanRulePriors(userMessage string) TurnPlanRulePriors {
	hint := DetectTaskFrameRules(userMessage)

	return TurnPlanRulePriors{
		TaskFrameGoal:     hint.Goal,
		TaskFrameDelivery: hint.Delivery,
		MergeWebSearch:    hint.MergeWebSearch,
		ExplicitCreate:    false, // 简化：不做扩展创建检测
		BareFeatureCreate: false,
		CapabilityProbe:   false,
	}
}

// BuildTurnPlanFromRules 纯规则构建 TurnPlan（不含 LLM）
func BuildTurnPlanFromRules(userMessage string, priors TurnPlanRulePriors) TurnPlan {
	plan := TurnPlan{
		Goal:           priors.TaskFrameGoal,
		Delivery:       priors.TaskFrameDelivery,
		MergeWebSearch: priors.MergeWebSearch,
		Source:         "rules",
	}

	// 路由决策
	if priors.TaskFrameGoal == GoalCasual && priors.TaskFrameDelivery == DeliveryProse {
		plan.Routing = RouteCasualChat
	} else {
		plan.Routing = RouteStructuredChat
	}

	// 搜索需求
	if priors.MergeWebSearch {
		plan.NeedsSearch = true
		plan.SearchQuery = userMessage
	}

	// 格式提示
	plan.FormatHint = buildFormatHintFromDelivery(priors.TaskFrameDelivery)

	return plan
}

// buildFormatHintFromDelivery 根据交付形态生成格式提示
func buildFormatHintFromDelivery(delivery TaskDeliveryFormat) string {
	switch delivery {
	case DeliveryMarkdownTable:
		return "请用 Markdown 表格组织信息，列名在前、按行对比。"
	case DeliveryBulletList:
		return "请用无序列表逐条列出，每条简短。"
	default:
		return ""
	}
}

// ─── 格式提示公共函数 ──────────────────────────────────────────

// BuildFormatHintFromDelivery 公开版本
func BuildFormatHintFromDelivery(delivery TaskDeliveryFormat) string {
	return buildFormatHintFromDelivery(delivery)
}

// ─── 辅助函数 ──────────────────────────────────────────────────

// IsDeliveryWeakSignal 检测用户消息是否包含弱交付信号
func IsDeliveryWeakSignal(msg string) bool {
	return deliveryWeakSignalRE.MatchString(msg)
}

// DefaultTurnPlan 默认闲聊 TurnPlan
func DefaultTurnPlan() TurnPlan {
	return TurnPlan{
		Routing:  RouteCasualChat,
		Goal:     GoalCasual,
		Delivery: DeliveryProse,
		Source:   "rules",
	}
}
