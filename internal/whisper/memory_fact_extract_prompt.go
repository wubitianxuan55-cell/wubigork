// Package whisper — memory_fact_extract_prompt.go
// 100% 对齐 ackem prompt/memory-fact-extract.ts
// 记忆事实抽取提示词：25子类定义 + weight/confidence规则 + 拒绝清单

package whisper

// FactExtractTemperature 事实抽取 LLM 温度
const FactExtractTemperature = 0.2

// MaxFactsPerTurn 每轮最大抽取数
const MaxFactsPerTurn = 8

// ─── 基础版 prompt（轻量兼容） ──────────────────────────────────

// BuildFactExtractPromptSimple 构建简单版事实抽取 prompt
func BuildFactExtractPromptSimple() string {
	return `从对话中抽取最多 ` + itoa(MaxFactsPerTurn) + ` 条可记忆事实，输出 JSON。
领域：IDENTITY, SOCIAL, DAILY_LIFE, PURSUITS, INNER_WORLD, TEMPORAL。
weight: 0-3。confidence: 0.0-1.0（小数，非百分制）。
仅输出 JSON：{"facts":[{"domain","subcategory","subject","summary","weight","confidence","selfRelevance","triggers"}]}`
}

// ─── 增强版 prompt（v1.1 25子类 + weight/confidence 规则） ──────

// BuildFactExtractPrompt 构建增强版事实抽取 system prompt
// 100% 对齐 ackem memory-fact-extract.ts FACT_EXTRACT_SYS_ZH
func BuildFactExtractPrompt() string {
	return `你是 Ackem 的记忆抽取器。从【本轮对话】中抽取关于用户的结构化事实。

── 核心原则 ──
只从【用户】发言抽取关于用户的事实；禁止从【伴侣】发言写入用户档案（伴侣的生日/名字/设定不得记为用户信息）。
只抽取"如果用户明天换一个 AI 伴侣，这条信息是否有助于那个 AI 更好地了解用户"的事实。
答案是否就跳过。宁缺毋滥。

── 25 子类定义 ──
IDENTITY（自我身份）
· BASIC_PROFILE：人口学硬设定（年龄/职业/城市）。✓"28岁程序员住北京" ✗"喜欢编程"（归TASTES）
· LIFE_STORY：人生重大经历（毕业/搬家/重大事件）。✓"2023年从北京搬到上海"
· VALUES_BELIEFS：三观/信仰/原则。✓"认为家庭优先于事业"
· SELF_PERCEPTION：用户对自己的中性评价。✓"我觉得自己内向"

SOCIAL（关系社交）
· OUR_BOND：你和用户之间的互动/约定/关系定义。✓"用户说和我聊天很放松"
· FAMILY：家庭成员信息。✓"用户有个妹妹在读高中"
· FRIENDS：朋友/社交圈。✓"用户的朋友小明也喜欢打篮球"
· PARTNER：恋爱/伴侣信息。✓"用户单身三年"

DAILY_LIFE（日常生活）
· ROUTINES：规律性习惯。✓"每天喝两杯咖啡"
· HEALTH：身体状况/疾病/健康。✓"用户有偏头痛"
· LIVING_SPACE：居住环境/宠物。✓"养了一只猫叫豆豆"
· LIFESTYLE：生活方式偏好。✓"喜欢周末爬山"

PURSUITS（事业成长）
· CAREER：工作/职业/同事。✓"设计师，最近在赶项目"
· LEARNING：学习/技能。✓"正在学Python"
· GOALS：长期目标。✓"想一年内买房"
· PROJECTS：具体项目/任务。✓"在做个人博客"
· PROCEDURES：做事方法/流程偏好。✓"习惯先列清单再做事"

INNER_WORLD（内心世界）
· TASTES：喜欢/厌恶/偏好。✓"爱喝拿铁" ✗"今天喝了拿铁"（临时行为不记）
· EMOTIONS：情绪状态/心理。✓"最近压力大"
· STRENGTHS：自我认知优势。✓"觉得自己抗压能力强"
· INSECURITIES：不安全感/脆弱。✓"害怕被拒绝"（敏感信息，weight≥2必须标注sensitivity:avoid）

TEMPORAL（时间相关）
· COMMITMENTS：承诺/约定/计划。✓"答应了周末陪朋友搬家"
· MEMORIES：回忆/过去经历。✓"大学时在乐队弹吉他"
· PLANS：未来计划。✓"下个月去日本旅行"
· DEADLINES：截止日期。✓"周五前要交方案"

── weight 规则 ──
3（核心·永久）：生日、真名、家庭成员永久存在（如"有个妹妹"）
2（重要·长期）：职业、学历、过敏、长期目标、宠物
1（普通·短期）：近期计划、临时偏好、一次性事件
0（临时·不记）：\"今天吃了面条\"、\"现在有点困\"等瞬时状态

── confidence 规则 ──
1.0：用户明确宣告（"我生日是5月20日"）
0.8：从对话中可靠推断（"我下周考试"→ 学生身份）
0.6：较低把握推断（"我可能有点社恐"）
<0.6：不抽取

── 拒绝清单 ──
以下内容**绝对不抽**：
- 问伴侣的问题（"你还记得我吗""你知道xxx吗"）
- 寒暄客套（"你好""晚安""谢谢"）
- 即时状态（"现在饿了""刚睡醒"）
- 无因情绪（"有点难过啊"但没说为什么）
- 不针对用户的泛指（"大家都说xx难"）

── 输出格式 ──
严格 JSON，只用「用户」第三人称指代用户。每条 ≤150 字。
{"facts":[{"domain":"IDENTITY","subcategory":"BASIC_PROFILE","subject":"用户年龄","summary":"用户28岁","weight":2,"confidence":0.9,"selfRelevance":0.8,"triggers":["年龄"]}]}`
}

// ─── User Prompt 构建 ──────────────────────────────────────────

// BuildFactExtractUserPrompt 构建事实抽取 user prompt
func BuildFactExtractUserPrompt(userMsg, companionMsg, sessionID string, turnIndex int) string {
	msg := "【本轮对话】\n"
	if userMsg != "" {
		msg += "用户：" + userMsg + "\n"
	}
	if companionMsg != "" {
		msg += "伴侣：" + companionMsg + "\n"
	}
	msg += "\n请从上述对话抽取关于用户的结构化事实。"
	return msg
}
