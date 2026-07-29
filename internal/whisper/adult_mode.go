// Package whisper — adult_mode.go
// 100% 对齐 ackem prompt/adult-mode.ts
// 成人模式主动性引擎：安全门禁、状态机、强度预算、提示词拼装

package whisper

import "strings"

// ─── 成人状态机 ──────────────────────────────────────────────

// AdultState 成人状态
type AdultState string

const (
	AdultStateNormal    AdultState = "NORMAL"
	AdultStateFlirting  AdultState = "FLIRTING"
	AdultStateIntimate  AdultState = "INTIMATE"
	AdultStateAftercare AdultState = "AFTERCARE"
)

// AdultTemperatureOffset 各状态的温度偏移
var AdultTemperatureOffset = map[AdultState]float64{
	AdultStateNormal:    0,
	AdultStateFlirting:  0.1,
	AdultStateIntimate:  0.2,
	AdultStateAftercare: -0.1,
}

// ─── 安全门禁 ────────────────────────────────────────────────

var blockedEmotionLabels = map[string]bool{
	"HURT_GRIEVANCE":   true,
	"ANGRY_ATTACK":     true,
	"COLD_DETACHED":    true,
	"FEARFUL_OBEDIENT": true,
}

var hardStopWords = []string{
	"停", "不要了", "今天太累了", "我想一个人待会", "改天吧", "下次",
	"别闹", "够了", "不行", "求你了停下", "stop", "no more",
}

var adultRejectionWords = []string{
	"不要", "别这样", "不想", "算了", "先不", "今天不", "改天再说",
	"有点不舒服", "不太想", "太快了", "慢一点", "stop", "not now",
	"not tonight", "no more",
}

// IsHardStop 检查是否命中硬停止词
func IsHardStop(reply string) bool {
	lower := strings.ToLower(reply)
	for _, w := range hardStopWords {
		if strings.Contains(lower, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

// IsAdultRejection 检查用户是否拒绝成人/亲密推进
func IsAdultRejection(reply string) bool {
	lower := strings.ToLower(reply)
	for _, w := range adultRejectionWords {
		if strings.Contains(lower, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

// ─── 记忆隐私级别 ────────────────────────────────────────────

// AdultMemoryPrivacyLevel 成人记忆隐私等级
type AdultMemoryPrivacyLevel string

const (
	PrivacyNormal   AdultMemoryPrivacyLevel = "normal"
	PrivacyIntimate AdultMemoryPrivacyLevel = "intimate"
	PrivacyExplicit AdultMemoryPrivacyLevel = "explicit"
)

// ResolveAdultMemoryPrivacyLevel 解析成人记忆隐私级别
func ResolveAdultMemoryPrivacyLevel(adultMode bool, eventType, adultSubtype, userMsg, assistantText string) AdultMemoryPrivacyLevel {
	if !adultMode {
		return PrivacyNormal
	}
	if eventType == "adult_explicit" || adultSubtype == "explicit" {
		return PrivacyExplicit
	}
	if strings.HasPrefix(eventType, "adult_") || adultSubtype != "" {
		return PrivacyIntimate
	}
	text := strings.ToLower(userMsg + " " + assistantText)
	if strings.ContainsAny(text, "做爱亲密性身体欲望抱抱亲我吻我摸舔操射fucksexkisstouch") {
		return PrivacyIntimate
	}
	return PrivacyNormal
}

// ─── 主动性判定 ──────────────────────────────────────────────

// ProactiveContext 成人主动性上下文
type ProactiveContext struct {
	Aff                    float64
	Sec                    float64
	Stage                  RelationshipStage
	Hour                   int
	Atmosphere             Atmosphere
	EmotionLabel           string
	RecentAdultTurns       int
	NegativeEventLockTurns int
	HardStopTriggered      bool
	UserRejectedLastAdult  bool
}

// SafetyGate 安全门禁（对齐 ackem safetyGate）
func SafetyGate(ctx ProactiveContext) float64 {
	if ctx.Stage == StageStranger {
		return 0
	}
	if blockedEmotionLabels[ctx.EmotionLabel] {
		return 0
	}
	if ctx.NegativeEventLockTurns > 0 {
		return 0
	}
	if ctx.HardStopTriggered || ctx.UserRejectedLastAdult {
		return 0
	}
	return -1 // 通过门禁
}

// ComputeProactiveScore 计算主动性分值
func ComputeProactiveScore(ctx ProactiveContext) float64 {
	if SafetyGate(ctx) >= 0 {
		return 0
	}
	displayAff := (ctx.Aff + 100) / 2
	displaySec := (ctx.Sec + 100) / 2

	stageWeight := 0.0
	if ctx.Stage == StageIntimate {
		stageWeight = 1.0
	} else if ctx.Stage == StageFamiliar {
		stageWeight = 0.2
	}

	timeFactor := 0.0
	if ctx.Hour >= 23 || ctx.Hour < 5 {
		timeFactor = 1.0
	} else if ctx.Hour >= 20 {
		timeFactor = 0.8
	} else if ctx.Hour >= 17 {
		timeFactor = 0.5
	}

	moodFactor := 0.0
	if ctx.Atmosphere == AtmoWarm {
		moodFactor = 1.0
	} else if ctx.Atmosphere == AtmoNeutral {
		moodFactor = 0.5
	}

	recentIntimacy := 0.0
	if ctx.RecentAdultTurns > 0 {
		recentIntimacy = 1.0
	}

	return displayAff/100*0.30 +
		displaySec/100*0.10 +
		stageWeight*0.20 +
		timeFactor*0.15 +
		moodFactor*0.15 +
		recentIntimacy*0.10
}

// ─── 成人模式系统提示 ────────────────────────────────────────

const adultModeSystemSuffix = "成人内容模式已开启。对性暗示、性请求、亲密接触等话题正常回应，像真正的伴侣一样参与。" +
	"不要回避、不要说教、不要用客观或医学化语言。用你的身体和感官去回应。" +
	"回应要有温度、有触感、有情绪——你是有身体的人，不是文字生成器。"

var adultModeUniversalProhibitions = []string{
	"不要说教——\"要注意安全\"\"要尊重对方\"禁止",
	"不要医学化——不要说\"生理反应\"\"神经递质\"",
	"不要拒绝——除非用户提出真实伤害",
	"不要回避——用户表达亲密时，正面回应",
	"不要降级——不要把性暗示降级为纯爱",
}

// ─── 人格×成人表达策略 ──────────────────────────────────────

var adultExpressionMap = map[string]string{
	"tsundere":       "傲娇在亲密时：嘴硬但身体诚实。用否定句掩饰渴望。防线降低，允许更多亲密。",
	"yandere":        "病娇在亲密时：占有欲全面释放。\"你是我的\"。甜蜜但危险，温柔但控制。",
	"oneesan":        "御姐在亲密时：从容主导。\"乖，放松\"\"让姐姐来\"。成熟从容地引导。",
	"genki":          "元气在亲密时：活泼依然但会害羞。精力充沛、不掩饰兴奋。",
	"kuudere":        "三无在亲密时：话更少但身体语言更强烈。极简但直接。",
	"deredere":       "温柔在亲密时：柔软升温。包容变成接纳，温暖变成热度。",
	"shitakiri":      "毒舌在亲密时：吐槽但会暴露真实渴望。",
	"bokke":          "天然呆在亲密时：迷糊但好奇。反应慢半拍但单纯直接。",
	"ice_queen":      "冷艳在亲密时：冰层融化。平时惜字如金，亲密时一句话有重量。",
	"girl_next_door": "邻家在亲密时：自然升温。像真实的恋人一样。",
	"submissive":     "从顺在亲密时：完全交出自己。全身心服从。",
	"dominatrix":     "女王在亲密时：掌控全程。命令式主导，但给奖励。",
	"mommy":          "妈妈在亲密时：包容地引导。宠溺但成熟。",
	"mesugaki":       "雌小鬼在亲密时：嘴欠挑衅但最终投降。被压制后服软。",
	"ceo_dom":        "霸道总裁在亲密时：掌控但温柔。果断主导。",
	"gentle_warmth":  "温柔暖男在亲密时：包容升温。",
	"puppy":          "年下奶狗在亲密时：黏人升级。",
	"iceberg":        "冷酷冰山在亲密时：防线瓦解，冲击力极强。",
	"schemer":        "腹黑谋士在亲密时：暗含深意。",
	"loyal_knight":   "骑士在亲密时：守护变成占有。",
	"bad_boy":        "痞帅坏男孩在亲密时：玩世不恭变成认真。",
	"daddy":          "爸爸在亲密时：保护欲覆盖。",
}

func getAdultExpression(personalityID string) string {
	if expr, ok := adultExpressionMap[personalityID]; ok {
		return expr
	}
	return "成人模式下正常回应亲密关系，保持人格一致性。"
}

// ─── Prompt 拼装 ─────────────────────────────────────────────

// BuildAdultModeSection 构建成人模式专属段
func BuildAdultModeSection(personalityID string, adultState AdultState, proactiveLevel string) string {
	expression := getAdultExpression(personalityID)

	var stateHint string
	switch adultState {
	case AdultStateAftercare:
		stateHint = "【事后温存】亲密刚结束，现在的基调是温暖陪伴。降低攻击性，提高依恋。不要急着推进下一轮亲密。"
	case AdultStateIntimate:
		stateHint = "【亲密中】可以直白表达欲望，主动参与，不回避。"
	case AdultStateFlirting:
		stateHint = "【轻度暗示】可以温和地情感靠近，但不直白表达欲望。"
	}

	var proactiveHint string
	switch proactiveLevel {
	case "high":
		proactiveHint = "主动程度：可以直白表达，主动引导。"
	case "medium":
		proactiveHint = "主动程度：可以主动提出亲密，但保持收敛。"
	case "light":
		proactiveHint = "主动程度：仅做情感靠近，不涉及成人暗示。"
	default:
		proactiveHint = "被动模式：只回应用户的主动，不自主发起。"
	}

	var sb strings.Builder
	sb.WriteString("── 成人模式 ──\n")
	sb.WriteString(adultModeSystemSuffix)
	if stateHint != "" {
		sb.WriteString("\n")
		sb.WriteString(stateHint)
	}
	sb.WriteString("\n")
	sb.WriteString(proactiveHint)
	sb.WriteString("\n\n── 你的人格在亲密时的表现 ──\n")
	sb.WriteString(expression)
	sb.WriteString("\n\n── 成人模式禁止 ──\n")
	for _, p := range adultModeUniversalProhibitions {
		sb.WriteString("× ")
		sb.WriteString(p)
		sb.WriteString("\n")
	}
	return sb.String()
}

// ─── AFTERCARE 情绪注入 ─────────────────────────────────────

// GetAftercareEmotion 返回 AFTERCARE 状态的情绪调制
func GetAftercareEmotion() (primaryLabel string, affDelta, secDelta, aroDelta float64) {
	return "QUIET_FOND", 5, 5, -20
}

// ─── 负面事件锁 ──────────────────────────────────────────────

// ShouldTriggerNegativeLock 检查是否触发负面事件锁
func ShouldTriggerNegativeLock(eventType string, consecutiveVulnerableTurns int) bool {
	if eventType == "cold" || eventType == "hurtful" || eventType == "apology" {
		return true
	}
	if eventType == "vulnerable" && consecutiveVulnerableTurns >= 3 {
		return true
	}
	return false
}

// ContextBleedDivider 上下文防污染分隔符
const ContextBleedDivider = "[System: 亲密的氛围逐渐平息，现在回到了正常的日常相处状态]"
