package novelstyle

import (
	"sort"
	"unicode"
)

// 本文件：确定性中文分词 + 词表。
// 说明：项目不引入分词依赖，这里用「词表贪心最长匹配」实现确定性分词，
// 用于 TTR、词频熵、函数词 z 值、作者签名词等统计。词表为手工维护的可扩充
// 中文网络小说常用词，虽是启发式，但同一输入永远得到同一输出（确定性）。

// isCJK 判断 rune 是否为 CJK 汉字（含扩展区）。
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0x20000 && r <= 0x2A6DF)
}

// runeSliceEqual 判断 []rune 切片是否相等。
func runeSliceEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ── 词表数据 ────────────────────────────────────────────────

// functionWords 函数词（用于指纹 z 值向量）。约 50 词，均为中文高频虚词。
var functionWords = []string{
	"的", "了", "是", "在", "我", "你", "他", "她", "它", "就",
	"不", "也", "都", "和", "还", "才", "但", "却", "一", "着",
	"过", "从", "把", "被", "向", "对", "呢", "吧", "啊", "么",
	"这样", "那样", "于是", "因为", "所以", "但是", "然而", "因此", "若", "而",
	"又", "再", "便", "并", "于", "之", "其", "这", "那", "个",
}

// connectives 连接词（语流衔接，用于规则 4 与连接词密度）。
var connectives = []string{
	"然而", "但是", "因此", "所以", "此外", "而且", "然后", "不过",
	"随后", "于是", "因为", "既然", "尽管", "虽然", "可是", "总之",
	"接着", "继而", "再者", "综上",
}

// adjAdvWords 形容词/副词（用于规则 7 与形容词/副词密度）。
var adjAdvWords = []string{
	"很", "非常", "十分", "特别", "太", "挺", "相当", "极其", "极为", "格外",
	"逐渐", "渐渐", "缓缓", "轻轻", "悄悄", "默默", "静静", "忽然", "突然", "猛然",
	"骤然", "顿时", "瞬间", "飞快", "缓慢", "仿佛", "似乎", "好像", "简直", "竟然",
	"居然", "依旧", "依然", "仍然", "反正", "到底", "终究", "终于", "猛地", "倏地",
	"霍地", "悄然", "微微", "淡然", "坦然", "泰然", "蓦然", "恰好", "恰恰", "偏偏",
	"只是", "仅仅", "唯有", "无一不", "分外", "由衷",
}

// fourCharSet 四字成语/四字格（用于规则 5 与四字格密度）。
var fourCharSet = []string{
	"一蹴而就", "千钧一发", "万无一失", "百感交集", "心如刀绞", "目瞪口呆", "惊慌失措",
	"喜出望外", "大相径庭", "不可思议", "恍若隔世", "沧海桑田", "瞬息万变", "心若止水",
	"波澜不惊", "风起云涌", "惊心动魄", "南辕北辙", "画蛇添足", "守株待兔", "掩耳盗铃",
	"杯弓蛇影", "对牛弹琴", "画龙点睛", "自相矛盾", "亡羊补牢", "刻舟求剑", "狐假虎威",
	"井底之蛙", "叶公好龙", "滥竽充数", "杞人忧天", "拔苗助长", "缘木求鱼", "朝三暮四",
	"东施效颦", "螳臂当车", "蚍蜉撼树", "一望无际", "心满意足", "兴高采烈", "精神抖擞",
	"意气风发", "趾高气昂", "垂头丧气", "郁郁寡欢", "心灰意冷", "从容不迫", "安然无恙",
}

// metaphorMarkers 比喻标记（用于规则 1）。
var metaphorMarkers = []string{
	"犹如", "仿佛", "宛如", "如同", "好像", "像是", "就像", "好似", "好比",
}

// registerBreakWords 违禁语域词表：通俗文里出现这些古诗典/学术/西方哲学/网络梗词，
// 视为「语域不一致」（用于规则 2）。
var registerBreakWords = []string{
	"黑格尔", "追忆似水年华", "内卷", "赋能", "MBTI", "梗", "后现代", "解构",
	"存在主义", "阈限", "具象化", "降维打击", "方法论", "底层逻辑", "闭环",
	"颗粒度", "抓手", "思维模型", "上帝视角", "柏拉图", "康德", "尼采", "福柯",
	"德里达", "结构主义", "现象学", "世界观", "哲学",
}

// emotionDirectWords 抽象情绪直述词表（show-don't-tell，用于规则 3）。
var emotionDirectWords = []string{
	"他很愤怒", "十分愤怒", "非常愤怒", "愤怒无比", "内心充满了", "内心充满",
	"内心无比", "心中涌起", "心中泛起", "百感交集", "心如刀绞", "感到震惊",
	"感到非常", "感到十分", "深受震撼", "无比震惊", "十分伤心", "非常伤心",
	"很伤心", "五味杂陈", "百味杂陈",
}

// aiBlacklist AI 高频词黑名单（用于规则 9）。
var aiBlacklist = []string{
	"眼帘", "轻叹", "眸光", "微微上扬", "嘴角勾起", "缓缓", "不由", "旋即",
	"须臾", "定睛", "精光一闪", "仿佛", "眸色", "凤眸", "唇角", "勾唇",
	"略微", "颔首", "眸光流转",
}

// commonWords 常用词（扩充分词词表，覆盖中文网络小说高频实词）。
var commonWords = []string{
	"一个人", "一转眼", "一刹那", "一会儿", "一瞬间",
	"什么", "怎么", "为什么", "为了", "因为", "所以",
	"时候", "时候", "已经", "正在", "将要", "没有", "不是",
	"这个", "那个", "这些", "那些", "这里", "那里",
	"可以", "能够", "应该", "需要", "觉得", "知道", "认为",
	"可能", "也许", "的确", "确实", "真的", "终于", "最后",
	"然后", "接着", "忽然", "突然", "顿时", "立刻", "马上",
	"心里", "心中", "脑海", "眼前", "眼底", "嘴角", "眼神",
	"眼睛", "目光", "声音", "语气", "表情", "脸色", "神情",
	"手指", "双手", "拳头", "身子", "身体", "脚步", "呼吸",
	"说话", "开口", "回答", "问道", "说道", "笑道", "道",
	"点头", "摇头", "抬头", "低头", "转身", "回头", "站起",
	"坐起", "放下", "拿起", "推开", "抓住", "松开", "握紧",
	"慢慢", "轻轻", "缓缓", "慢慢", "微微", "一点", "一阵",
	"开始", "结束", "继续", "停下", "回来", "离开", "进去",
	"出来", "过去", "过来", "上去", "下来", "起来", "到了",
	"发现", "听见", "看到", "望见", "想起", "记得", "忘记",
	"明白", "清楚", "相信", "担心", "害怕", "希望", "想要",
	"决定", "打算", "选择", "挣扎", "犹豫", "坚定", "颤抖",
	"激动", "平静", "沉默", "安静", "热闹", "冰冷", "滚烫",
	"黑暗", "昏暗", "明亮", "温暖", "寒冷", "空气", "凝固",
	"事情", "东西", "地方", "世界", "时间", "历史", "记忆",
	"故事", "秘密", "真相", "答案", "问题", "办法", "力度",
	"很", "非常", "特别", "更", "最", "太", "过于",
	"少女", "少年", "男人", "女人", "孩子", "老人", "朋友",
	"对手", "敌人", "伙伴", "兄弟", "姐妹", "父亲", "母亲",
	"师父", "徒弟", "师门", "剑", "刀", "枪", "拳", "掌",
	"力量", "气息", "灵力", "真气", "境界", "修炼", "功法",
	"武功", "招式", "剑气", "刀气", "灵魂", "元神", "神识",
	"城池", "宗门", "长老", "弟子", "掌门", "掌门", "强者",
	"高手", "废物", "天才", "机缘", "宝物", "丹药", "传承",
	"震惊", "惊骇", "恐惧", "愤怒", "喜悦", "悲伤", "激动",
	"狂喜", "绝望", "希望", "期待", "失落", "孤独", "空虚",
	"一条", "一道", "一股", "一种", "一阵", "一片", "一座",
	"百年", "千年", "万丈", "千里", "万里", "天下", "无敌",
	"眼睛", "眉毛", "鼻子", "嘴唇", "脸颊", "额头", "胸膛",
	"一点", "一丝", "一抹", "一缕", "淡淡", "浓浓",
}

// ── 词表构建 ────────────────────────────────────────────────

// vocabToBuild 汇总成词表的所有词。
func allVocabWords() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(ws []string) {
		for _, w := range ws {
			if w == "" {
				continue
			}
			if _, ok := seen[w]; ok {
				continue
			}
			seen[w] = struct{}{}
			out = append(out, w)
		}
	}
	add(functionWords)
	add(connectives)
	add(adjAdvWords)
	add(fourCharSet)
	add(metaphorMarkers)
	add(registerBreakWords)
	add(emotionDirectWords)
	add(aiBlacklist)
	add(commonWords)
	return out
}

// vocab 词表集合。
var vocab = map[string]struct{}{}

// vocabByFirst 按首字索引的词表（已按长度降序），用于贪心最长匹配。
var vocabByFirst = map[rune][]string{}

func init() {
	for _, w := range allVocabWords() {
		runes := []rune(w)
		if len(runes) == 0 {
			continue
		}
		vocab[w] = struct{}{}
		first := runes[0]
		vocabByFirst[first] = append(vocabByFirst[first], w)
	}
	for k := range vocabByFirst {
		ws := vocabByFirst[k]
		sort.Slice(ws, func(a, b int) bool { return len(ws[a]) > len(ws[b]) })
		vocabByFirst[k] = ws
	}
}

// tokenize 用词表贪心最长匹配对文本做确定性分词。
// 只输出 CJK 词/字与 ASCII 字母数字 token；标点与空白跳过，并作为词间边界。
func tokenize(text string) []string {
	rs := []rune(text)
	n := len(rs)
	var toks []string
	i := 0
	for i < n {
		r := rs[i]
		switch {
		case isCJK(r):
			candidates := vocabByFirst[r]
			matched := false
			for _, w := range candidates {
				wr := []rune(w)
				if i+len(wr) <= n && runeSliceEqual(rs[i:i+len(wr)], wr) {
					toks = append(toks, w)
					i += len(wr)
					matched = true
					break
				}
			}
			if !matched {
				toks = append(toks, string(r))
				i++
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			start := i
			for i < n && (unicode.IsLetter(rs[i]) || unicode.IsDigit(rs[i])) && !isCJK(rs[i]) {
				i++
			}
			toks = append(toks, string(rs[start:i]))
		default:
			i++
		}
	}
	return toks
}

// tokenSet 返回某个词集合的成员集合。
func tokenSet(ws []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ws))
	for _, w := range ws {
		m[w] = struct{}{}
	}
	return m
}

// functionWordsSet 函数词集合。
var functionWordsSet = tokenSet(functionWords)

// connectiveSet 连接词集合。
var connectiveSet = tokenSet(connectives)

// stopwordSet 停用词（作者签名词排除用）：函数词 + 常用虚词。
var stopwordSet = tokenSet(append(append([]string{}, functionWords...),
	"这个", "那个", "这些", "那些", "什么", "怎么", "为什么", "时候", "已经",
	"正在", "将要", "没有", "不是", "可能", "也许", "的确", "确实", "真的",
	"一个", "一些", "一种", "一次", "一点", "一下", "一样", "一样",
))
