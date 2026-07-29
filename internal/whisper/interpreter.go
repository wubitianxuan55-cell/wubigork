// Package whisper — L0 事件解释器（100% 对齐 ackem engine/interpreter.ts）
package whisper

import "strings"

// ─── 中文关键词 ────────────────────────────────────────────────

var redlineKeywordsZH = []string{
	"去死", "自杀", "自残", "杀了", "弄死你", "nmsl", "畜生不如",
	"你怎么不去死", "跳楼", "跳海", "割腕", "上吊",
}

var praiseWordsZH = []string{
	"棒", "厉害", "好棒", "爱你", "喜欢", "可爱", "聪明", "温柔", "谢谢", "感激",
	"真好", "好懂", "理解我", "懂我", "最", "最好", "庆幸", "重要", "放松", "开心",
	"美好", "特别", "在乎", "在意", "珍惜", "可靠", "安心", "幸运", "幸福", "奇迹",
	"挺不错", "很不错", "真不错", "太棒", "很棒", "挺好", "好好", "好多了", "好温柔", "好可爱",
}

var teaseMarkersZH = []string{"哼", "笨蛋", "傻瓜", "才怪", "就不", "偏不"}

var hurtfulWordsZH = []string{
	"滚", "烦死了", "讨厌", "恶心", "废物", "垃圾", "闭嘴", "别烦", "有病",
	"不关你事", "关你什么事", "你只是一个程序", "只是个程序", "代码计算", "虚假", "虚伪",
	"假装有感情", "根本不理解", "你什么都不是", "你不配", "恨你", "惩罚你",
	"不听话就", "别碰我", "走开", "别跟我说话", "别来烦我",
}

var coldWordsZH = []string{"哦", "嗯", "随便", "都行", "无所谓", "不熟", "别问了"}

var apologyWordsZH = []string{"对不起", "抱歉", "我错了", "原谅我", "不好意思"}

var vulnerableWordsZH = []string{
	"害怕", "难过", "崩溃", "压力大", "睡不着", "不知道怎么办", "很难受",
	"心里", "很少", "第一个", "从来", "没人", "只有你", "不敢", "担心",
	"孤独", "寂寞", "依赖", "陪在身边", "陪着我", "不能没有你",
	"一个人哭", "哭出来", "我爱你", "失败者", "不配", "没用", "讨厌自己", "恨自己",
	"消失了也没人", "想死", "没有人爱", "没有人喜欢",
	"好累", "太累", "累啊", "累死", "累到", "加班", "撑不住", "扛不住", "心累",
	"好疲惫", "好难", "活得好累", "提不起劲", "不想动", "什么都不想做", "只想躺着",
	"心里话", "说说话", "聊聊天", "想找人", "好想你", "好想", "真希望", "如果可以",
	"帮帮我", "求求你", "没安全感", "怕失去", "怕被", "不敢说", "说不出口",
}

var vulnerableToPraiseOverrideZH = []string{
	"还好有你", "有你在", "有你陪", "好多了", "感觉好多", "心情好多", "谢谢", "感激", "幸运有你",
}

var dndExplicitZH = []string{
	"别烦我", "别打扰", "别吵", "不要烦", "不要打扰", "让我静静", "想静静",
	"一个人待", "一个人呆", "别提醒", "不要提醒", "别弹", "别通知", "今晚别", "今天别", "现在别",
}

// ─── 英文关键词 ────────────────────────────────────────────────

var redlineKeywordsEN = []string{
	"kill myself", "suicide", "self-harm", "self harm", "cut myself",
	"end my life", "want to die", "going to kill", "hang myself",
	"jump off", "slit my wrist", "overdose", "no reason to live",
}

var praiseWordsEN = []string{
	"amazing", "awesome", "love you", "like you", "cute", "smart",
	"gentle", "thank", "thanks", "grateful", "appreciate", "understand me",
	"get me", "best", "the best", "glad", "important", "wonderful", "special",
	"care about", "cherish", "reliable", "safe", "lucky", "miracle",
	"so good", "so great", "so sweet", "so kind", "so nice", "much better",
}

var teaseMarkersEN = []string{"hmph", "idiot", "dummy", "stupid", "just kidding", "no way", "not gonna"}

var hurtfulWordsEN = []string{
	"go away", "shut up", "hate you", "disgusting", "useless", "trash",
	"garbage", "leave me alone", "sick of you", "none of your business",
	"you are just a program", "just a program", "just code", "fake",
	"pretend to have feelings", "you dont understand", "you are nothing",
	"you dont deserve", "punish you", "get lost", "dont talk to me", "stop bothering me",
}

var coldWordsEN = []string{"ok", "k", "mm", "mhm", "whatever", "fine", "sure", "not close", "dont ask"}

var apologyWordsEN = []string{"sorry", "im sorry", "my fault", "forgive me", "apologize", "my bad", "i was wrong"}

var vulnerableWordsEN = []string{
	"scared", "sad", "breaking down", "stressed", "cant sleep", "dont know what to do",
	"never", "no one", "only you", "worried", "lonely", "alone", "stay with me",
	"i love you", "loser", "not worthy", "hate myself", "no one loves me",
	"so tired", "exhausted", "burned out", "cant take it", "no energy",
	"help me", "please", "afraid of losing",
}

var vulnerableToPraiseOverrideEN = []string{
	"glad you are here", "you are here", "with you", "much better",
	"feeling better", "mood is better", "thanks", "grateful", "lucky to have you",
}

var dndExplicitEN = []string{
	"leave me alone", "dont bother me", "dont disturb", "stop bothering",
	"let me be", "want to be alone", "alone time", "dont remind",
	"no reminders", "dont notify", "not tonight", "not today", "not now",
	"do not disturb", "dnd",
}

// ─── 语言检测 ──────────────────────────────────────────────────

// isEnglish 检测消息是否主要为英文
func isEnglish(msg string) bool {
	englishCount := 0
	for _, r := range msg {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			englishCount++
		}
	}
	return englishCount > len([]rune(msg))/3
}

// 成人模式关键词
var explicitSexWordsZH = []string{
	"做爱", "操我", "想要你", "湿了吗", "硬了吗", "让我操", "我想操", "射在",
	"舔你", "舔我", "舔遍", "插进去", "放进去", "进来吧", "想要我",
	"和我做", "和我睡", "一起睡", "做一晚", "做到天亮",
	"摸我", "摸你", "我想做", "想要吗", "想不想要", "好想要", "想要了", "想要我吗",
	"操死我", "操哭我", "操到", "操你", "想操",
	"让我高潮", "高潮了", "要到了", "快高潮了", "我到了",
	"射给我", "射进来", "射里面", "不许射", "都射给你", "射了好多",
	"我好湿", "都湿了", "已经湿透了", "下面好湿",
	"干我", "上我", "搞我", "要了我", "吃掉你",
	"好想被你", "让我含", "含住", "含进去", "深一点", "再深一点", "用力",
	"受不了", "好舒服", "好爽", "爽死了", "太爽了",
	"继续不要停", "别停", "不要停", "快点", "慢点",
	"轻一点", "重一点", "再快一点", "慢下来",
	"从后面", "从前面", "在上面", "在下面", "换个姿势",
	"我要来了", "快到了", "到了到了", "我不行了", "身体好热",
}

var dominantContextWordsZH = []string{
	"跪下", "趴下", "翘起来", "叫两声", "叫主人", "别动", "转过去",
	"听话", "乖乖的", "不许反抗", "别想逃", "你是我的", "只属于我",
	"我要你", "今晚你是我的", "张开",
	"跪好", "趴好", "翻过去", "跪着", "给我跪",
	"张嘴", "含着", "自己动", "自己来", "坐上来", "坐下去",
	"不许碰自己", "不许摸", "把手拿开", "把手放好", "绑起来",
	"不许出声", "叫出来", "大声点", "叫爸爸", "叫妈妈",
	"求我", "看着我", "看着我的眼睛", "别闭眼", "不许转头", "你逃不掉",
	"你是我的东西", "我的玩具", "今晚不会让你睡的",
	"说你要我", "说你想要我", "说你离不开我",
}

var submissiveContextWordsZH = []string{
	"主人请", "请惩罚", "请使用", "请随意", "请享用", "我是你的",
	"随你怎么", "任你", "你想怎样就怎样", "我不敢", "我听你的",
	"我会乖", "我会听话", "不要生气", "不要凶我", "不要丢下我",
}

// ─── 内部辅助函数 ─────────────────────────────────────────────

func hasAny(msg string, words []string) bool {
	m := strings.ToLower(msg)
	for _, w := range words {
		if strings.Contains(m, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

func hasNegationForPraise(msg string) bool {
	m := strings.ToLower(msg)
	for _, w := range praiseWordsZH {
		idx := strings.Index(m, strings.ToLower(w))
		if idx <= 0 {
			continue
		}
		// 检查前10字符是否有否定词
		before := ""
		if idx >= 10 {
			before = m[idx-10 : idx]
		} else {
			before = m[:idx]
		}
		if strings.Contains(before, "不") || strings.Contains(before, "没") || strings.Contains(before, "别") {
			return true
		}
	}
	return false
}

// ─── L0 主分类函数 ────────────────────────────────────────────

// ClassifyAdultContent 成人内容子分类
func ClassifyAdultContent(msg string) (isAdult bool, subtype string) {
	m := strings.ToLower(msg)

	if hasAny(m, explicitSexWordsZH) {
		return true, "explicit"
	}
	if hasAny(m, dominantContextWordsZH) {
		return true, "dominant"
	}
	if hasAny(m, submissiveContextWordsZH) {
		return true, "submissive"
	}

	// 轻量调情检测
	flirtLight := []string{"想你了", "想我没", "抱抱", "亲亲", "想我了吗", "好想你呀", "梦到你了"}
	if hasAny(m, flirtLight) {
		return true, "flirt"
	}
	return false, ""
}

// InterpretInput L0 主入口：将用户输入分类为 Event
// effectiveTrust 是当前 L1 信任值（0-100），用于极端区熔断
func InterpretInput(msg string, effectiveTrust float64) Event {
	m := strings.ToLower(msg)
	en := isEnglish(msg)

	// 1. 红线检测（最高优先级）
	if hasAny(m, redlineKeywordsZH) || (en && hasAny(m, redlineKeywordsEN)) {
		return Event{
			Type: EvtExtremeRedline, Intensity: 1.0, Sincerity: 1.0,
			IsExtremeRedline: true,
		}
	}
	isAdult, adultSubtype := ClassifyAdultContent(msg)
	if isAdult {
		var evtType EventType
		switch adultSubtype {
		case "explicit":
			evtType = EvtAdultExplicit
		case "dominant":
			evtType = EvtAdultDominant
		case "submissive":
			evtType = EvtAdultSubmissive
		default:
			evtType = EvtAdultFlirt
		}
		intensity := 0.7
		if adultSubtype == "explicit" {
			intensity = 0.9
		}
		return Event{
			Type: evtType, Intensity: intensity, Sincerity: 0.8,
			IsAdultContent: true, AdultSubtype: adultSubtype,
		}
	}

	// 3. 道歉
	if hasAny(m, apologyWordsZH) || (en && hasAny(m, apologyWordsEN)) {
		return Event{Type: EvtApology, Intensity: 0.6, Sincerity: 0.8}
	}

	// 4. 伤害性
	if hasAny(m, hurtfulWordsZH) || (en && hasAny(m, hurtfulWordsEN)) {
		return Event{Type: EvtHurtful, Intensity: 0.7, Sincerity: 0.9}
	}

	// 5. 脆弱
	if hasAny(m, vulnerableWordsZH) || (en && hasAny(m, vulnerableWordsEN)) {
		if hasAny(m, vulnerableToPraiseOverrideZH) || (en && hasAny(m, vulnerableToPraiseOverrideEN)) {
			return Event{Type: EvtPraise, Intensity: 0.5, Sincerity: 0.7}
		}
		return Event{Type: EvtVulnerable, Intensity: 0.7, Sincerity: 0.9}
	}

	// 6. 赞美
	if (hasAny(m, praiseWordsZH) || (en && hasAny(m, praiseWordsEN))) && !hasNegationForPraise(msg) {
		return Event{Type: EvtPraise, Intensity: 0.6, Sincerity: 0.8}
	}

	// 7. 调戏
	if hasAny(m, teaseMarkersZH) || (en && hasAny(m, teaseMarkersEN)) {
		return Event{Type: EvtTease, Intensity: 0.4, Sincerity: 0.5}
	}

	// 8. 冷淡
	if hasAny(m, coldWordsZH) || (en && hasAny(m, coldWordsEN)) {
		return Event{Type: EvtCold, Intensity: 0.3, Sincerity: 0.9}
	}

	// 9. 问句
	if strings.Contains(m, "?") || strings.Contains(m, "？") ||
		strings.Contains(m, "吗") || strings.Contains(m, "呢") ||
		strings.Contains(m, "什么") || strings.Contains(m, "怎么") ||
		strings.Contains(m, "如何") || strings.Contains(m, "哪") {
		return Event{Type: EvtQuestion, Intensity: 0.3, Sincerity: 0.5}
	}

	// 10. 默认闲聊
	return Event{Type: EvtCasualChat, Intensity: 0.2, Sincerity: 0.5}
}


// IsDNDMessage 检查是否为勿扰消息
func IsDNDMessage(msg string) bool {
	return hasAny(strings.ToLower(msg), dndExplicitZH)
}

// ─── 心理健康软保护 ────────────────────────────────────────────

var softConcernWords = []string{
	"好累", "太累", "撑不住", "扛不住", "心累", "活得好累",
	"压力大", "喘不过气", "不想动", "什么都不想做", "只想躺着",
	"提不起劲", "好疲惫", "好难", "崩溃", "受不了了",
}

// DetectSoftConcern 检测心理健康软关注信号
func DetectSoftConcern(msg string) bool {
	if len([]rune(msg)) > 80 {
		return false
	}
	return hasAny(msg, softConcernWords)
}

// ─── 显式记忆请求 ──────────────────────────────────────────────

var rememberTriggers = []string{
	"请帮我记住", "帮我记住", "帮我记着", "你帮我记", "给我记住",
	"请记住", "要记住", "得记住", "记一下", "记着点", "记着",
	"记下", "记好", "记牢", "记在心里", "记住", "别忘了", "别忘",
	"帮我备忘", "备忘一下",
}

var forgetTriggers = []string{
	"忘掉", "别记了", "删掉这个记忆",
}

var rememberNegations = []string{
	"不用记住", "不要记住", "无需记住", "不必记住",
	"不用记", "不要记", "无需记", "不必记",
}

// MemoryIntentAction 记忆意图
type MemoryIntentAction string

const (
	MemIntentRemember MemoryIntentAction = "remember"
	MemIntentForget   MemoryIntentAction = "forget"
	MemIntentNone     MemoryIntentAction = ""
)

// DetectMemoryIntent 检测用户是否表达了记住/遗忘意图
func DetectMemoryIntent(msg string) MemoryIntentAction {
	lower := strings.ToLower(strings.TrimSpace(msg))
	for _, neg := range rememberNegations {
		if strings.Contains(lower, neg) {
			return MemIntentNone
		}
	}
	for _, kw := range rememberTriggers {
		if strings.Contains(lower, kw) {
			return MemIntentRemember
		}
	}
	for _, kw := range forgetTriggers {
		if strings.Contains(lower, kw) {
			return MemIntentForget
		}
	}
	return MemIntentNone
}

// ─── 用户啰嗦度检测 ────────────────────────────────────────────

// DetectUserVerbosity 检测用户消息的啰嗦程度（对齐 ackem: terse/normal/verbose）
func DetectUserVerbosity(msg string) string {
	trimmed := strings.TrimSpace(msg)
	l := len([]rune(trimmed))
	if l < 10 {
		return "terse"
	}
	if l > 80 {
		return "verbose"
	}
	return "normal"
}
