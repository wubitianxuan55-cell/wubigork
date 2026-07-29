// Package whisper — user_activity.go
// 100% 对齐 ackem context/userActivity.ts
// 用户活动检测：关键词分类 + 时态判断 + 工作日启发式

package whisper

import (
	"math"
	"strings"
)

// categoryRule 分类规则
type categoryRule struct {
	category UserActivityCategory
	keywords []string
	weight   int
}

var categoryRules = []categoryRule{
	{ActivityTravel, []string{"旅游", "出游", "出差", "出发", "到了", "景点", "酒店", "航班", "刚回"}, 3},
	{ActivityStudy, []string{"学习", "复习", "考试", "作业", "上课", "论文", "备考", "考研"}, 3},
	{ActivityWork, []string{"工作", "加班", "开会", "项目", "ddl", "上班", "办公", "赶工", "deadline", "下班"}, 3},
	{ActivityEntertainment, []string{"游戏", "打游戏", "minecraft", "mc", "追剧", "看电影", "副本", "steam"}, 3},
	{ActivitySocial, []string{"聚会", "约会", "陪爸", "陪妈", "朋友", "见面", "聚餐"}, 2},
	{ActivityHealth, []string{"健身", "医院", "跑步", "运动", "锻炼", "看病"}, 3},
	{ActivityRest, []string{"睡觉", "休息", "累了", "熬夜", "补觉", "困"}, 2},
	{ActivityDaily, []string{"吃饭", "通勤", "买菜", "家务", "做饭", "外卖"}, 1},
}

var futureMarkers = []string{"明天", "下周", "打算", "计划", "要去", "准备", "即将", "后天"}
var pastMarkers = []string{"刚", "结束", "昨天", "刚才", "玩完", "刚回", "刚结束", "回来"}
var presentMarkers = []string{"正在", "在写", "在玩", "路上", "到了", "现在"}

var categoryLabels = map[UserActivityCategory]string{
	ActivityRest:          "休息",
	ActivityWork:          "工作",
	ActivityStudy:         "学习",
	ActivityTravel:        "出游",
	ActivitySocial:        "社交",
	ActivityEntertainment: "娱乐",
	ActivityDaily:         "日常",
	ActivityHealth:        "健康",
	ActivityUnknown:       "未知",
}

var tenseLabels = map[ActivityTense]string{
	TenseFuture:  "将来",
	TensePresent: "进行中",
	TensePast:    "刚结束",
}

// ResolveUserActivityInput 活动推断输入
type ResolveUserActivityInput struct {
	RecentUserSnippets  []string
	MemoryFactSummaries []string
	Time                TimeRuntimeContext
	GameActive          bool
}

// scoreCategories 关键词打分
func scoreCategories(text string) map[UserActivityCategory]int {
	scores := make(map[UserActivityCategory]int)
	lower := strings.ToLower(text)
	for _, rule := range categoryRules {
		hit := 0
		for _, kw := range rule.keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				hit += rule.weight
			}
		}
		if hit > 0 {
			scores[rule.category] += hit
		}
	}
	return scores
}

// resolveTense 判断时态
func resolveTense(text string, category UserActivityCategory) ActivityTense {
	hasFuture := containsAny(text, futureMarkers)
	hasPast := containsAny(text, pastMarkers)
	hasPresent := containsAny(text, presentMarkers)

	if category == ActivityTravel {
		if hasFuture {
			return TenseFuture
		}
		if hasPast || strings.Contains(text, "刚回") || strings.Contains(text, "回来") {
			return TensePast
		}
		if hasPresent || strings.Contains(text, "到了") || strings.Contains(text, "路上") {
			return TensePresent
		}
	}

	if hasFuture && !hasPast {
		return TenseFuture
	}
	if hasPast && !hasFuture {
		return TensePast
	}
	if hasPresent {
		return TensePresent
	}
	return TensePresent
}

// weekdayWorkPrior 工作日工作时间启发式
func weekdayWorkPrior(t TimeRuntimeContext) *UserActivityCategory {
	if t.IsWeekend {
		return nil
	}
	if t.Hour >= 9 && t.Hour < 18 {
		c := ActivityWork
		return &c
	}
	return nil
}

// buildActivityLabel 构建活动标签
func buildActivityLabel(category UserActivityCategory, tense ActivityTense) string {
	if category == ActivityUnknown {
		return "暂无法判断用户在做什么"
	}
	return categoryLabels[category] + "·" + tenseLabels[tense]
}

// collectSources 收集推断来源
func collectSources(text string, category UserActivityCategory, gameActive bool) []string {
	var sources []string
	if gameActive {
		sources = append(sources, "gamemode:active")
	}
	lower := strings.ToLower(text)
	for _, rule := range categoryRules {
		if rule.category != category {
			continue
		}
		for _, kw := range rule.keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				sources = append(sources, "keyword:"+string(rule.category))
				break
			}
		}
		break
	}
	if len(sources) == 0 && category != ActivityUnknown {
		sources = append(sources, "time:heuristic")
	}
	return sources
}

const planOverrideMinConfidence = 0.7

// ResolveUserActivity 规则推断用户生活场景大类 + 时态
func ResolveUserActivity(input ResolveUserActivityInput) UserActivityContext {
	// 游戏活跃直接判定
	if input.GameActive {
		return UserActivityContext{
			Category:   ActivityEntertainment,
			Tense:      TensePresent,
			Label:      buildActivityLabel(ActivityEntertainment, TensePresent),
			Confidence: 0.85,
			Source:     []string{"gamemode:active"},
		}
	}

	// 拼接语料
	var parts []string
	parts = append(parts, input.RecentUserSnippets...)
	parts = append(parts, input.MemoryFactSummaries...)
	text := strings.ToLower(strings.Join(parts, " "))

	scores := scoreCategories(text)
	category := ActivityUnknown
	best := 0
	for cat, score := range scores {
		if score > best {
			best = score
			category = cat
		}
	}

	// 工作日启发式回退
	if category == ActivityUnknown {
		if prior := weekdayWorkPrior(input.Time); prior != nil {
			category = *prior
			best = 1
		}
	}

	if category == ActivityUnknown {
		return UserActivityContext{
			Category:   ActivityUnknown,
			Tense:      TensePresent,
			Label:      buildActivityLabel(ActivityUnknown, TensePresent),
			Confidence: 0,
			Source:     []string{"insufficient"},
		}
	}

	tense := resolveTense(text, category)
	confidence := math.Min(0.95, 0.35+float64(best)*0.12)
	confidence = math.Round(confidence*100) / 100

	return UserActivityContext{
		Category:   category,
		Tense:      tense,
		Label:      buildActivityLabel(category, tense),
		Confidence: confidence,
		Source:     collectSources(text, category, false),
	}
}

// containsAny 检查文本是否包含任一标记
func containsAny(text string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}
