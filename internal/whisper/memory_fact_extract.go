// Package whisper — memory_fact_extract.go
// 100% 对齐 ackem memory/factExtractor.ts + prompt/memory-fact-extract.ts
// LLM 事实抽取器：从每轮对话中抽取结构化用户事实

package whisper

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// ─── FactExtractor ────────────────────────────────────────────

// FactExtractor LLM 事实抽取器
type FactExtractor struct {
	// MaxPerTurn 每轮最多抽取条数
	MaxPerTurn int
}

// NewFactExtractor 创建抽取器
func NewFactExtractor() *FactExtractor {
	return &FactExtractor{MaxPerTurn: 8}
}

// Extract 从一轮对话中抽取事实
func (fe *FactExtractor) Extract(
	userMsg, companionMsg string,
	turnIndex int,
	sessionID string,
	llm LlmClient,
) (*FactExtractionResult, error) {
	sysPrompt := factExtractSystemZH
	userPrompt := fmt.Sprintf(
		"session=%s turn=%d\n【仅根据「用户」一行抽取关于用户的事实；「gaea」仅供理解语境，禁止从中抽取写入用户档案的信息】\n用户：%s\ngaea（勿抽取）：%s",
		sessionID, turnIndex, userMsg, companionMsg,
	)

	raw, err := llm.Chat(sysPrompt, userPrompt)
	if err != nil {
		return &FactExtractionResult{}, err
	}

	return parseExtractionSalvage(raw, fe.MaxPerTurn), nil
}

// ─── 解析 ─────────────────────────────────────────────────────

// parseExtractionSalvage 从 LLM 原始输出中解析 JSON
func parseExtractionSalvage(raw string, maxPerTurn int) *FactExtractionResult {
	if maxPerTurn <= 0 {
		maxPerTurn = 8
	}

	tryParse := func(s string) *FactExtractionResult {
		var j struct {
			Facts []json.RawMessage `json:"facts"`
		}
		if err := json.Unmarshal([]byte(s), &j); err != nil || len(j.Facts) == 0 {
			return nil
		}

		var facts []ExtractedFact
		for i, rawFact := range j.Facts {
			if i >= maxPerTurn {
				break
			}
			var f struct {
				Domain        string   `json:"domain"`
				Subcategory   string   `json:"subcategory"`
				Subject       string   `json:"subject"`
				Summary       string   `json:"summary"`
				Weight        float64  `json:"weight"`
				Confidence    float64  `json:"confidence"`
				SelfRelevance float64 `json:"selfRelevance"`
				Triggers      []string `json:"triggers"`
			}
			if err := json.Unmarshal(rawFact, &f); err != nil {
				continue
			}
			if f.Summary == "" || f.Subject == "" {
				continue
			}
			if f.Domain == "" {
				f.Domain = "DAILY_LIFE"
			}
			if f.Subcategory == "" {
				f.Subcategory = "NOW"
			}
			// 归一化 confidence: 0-1（防止 LLM 输出 0-100）
			conf := f.Confidence
			if conf > 1 {
				conf = conf / 100.0
			}
			conf = math.Max(0, math.Min(1, conf))

			facts = append(facts, ExtractedFact{
				Domain:        f.Domain,
				Subcategory:   f.Subcategory,
				Subject:       f.Subject,
				Summary:       f.Summary,
				Weight:        clampF(f.Weight, 0, 3),
				Confidence:    conf,
				SelfRelevance: clampF(f.SelfRelevance, 0, 1),
				Triggers:      f.Triggers,
			})
		}
		return &FactExtractionResult{Facts: facts}
	}

	// 直接解析
	trimmed := strings.TrimSpace(raw)
	if result := tryParse(trimmed); result != nil {
		return result
	}

	// 抢救：提取第一个 { 到最后一个 }
	i := strings.Index(trimmed, "{")
	j := strings.LastIndex(trimmed, "}")
	if i >= 0 && j > i {
		if result := tryParse(trimmed[i : j+1]); result != nil {
			return result
		}
	}

	return &FactExtractionResult{}
}

// ─── 事实抽取 System Prompt ───────────────────────────────────

const factExtractSystemZH = `你是 Ackem 的记忆抽取器。从【本轮对话】中抽取关于用户的结构化事实。

── 核心原则 ──
只从【用户】发言抽取关于用户的事实；禁止从【gaea】发言写入用户档案（gaea的生日/名字/设定不得记为用户信息）。
只抽取"如果用户明天换一个 AI gaea，这条信息是否有助于那个 AI 更好地了解用户"的事实。
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
· PARTNER：恋爱/gaea信息。✓"用户单身三年"

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
· MOOD：当前短暂情绪。✓"今天很焦虑"
· TASTES：具体喜好/雷区。✓"喜欢爵士乐"
· VULNERABILITIES：脆弱点/恐惧/不安全感。✓"害怕被拒绝"
· INSIDE_JOKES：你们之间独有的梗。✓"'你又忘了喂猫'是开玩笑"

TEMPORAL（当下未来）
· NOW：当前短时状态（3天内失效）。✓"现在很饿"
· COMMITMENTS：承诺/约定（不衰减）。✓"说周末一起看电影"
· PLANS：近期计划（7天内）。✓"打算周五去体检"
· WORLD：外部世界信息。✓"今天是端午节"

── weight 规则 ──
3 = 核心/永久（满足其一）：
  · 用户明确说出涉及自我认同改变的话
  · 事件不可逆且影响终身
  · 用户对你涉及深层依赖（"只有你理解我"）
2 = 重要/长期：持续几个月到几年（新工作/过敏/年度目标/重复提到2+次）
1 = 普通/短期：日常偏好或近期状态
0 = 临时/背景：仅当前语境有用。尽量不抽，除非 NOW 子类。

── confidence 规则 ──
1.0 = 用户第一人称明确宣告（"我是程序员"）
0.8 = 用户使用频率副词且指向稳定属性（"又得改这破代码"→职业编程相关）
0.6 = 模糊表达（"我好像有点怕黑"）
<0.6 = 不写入

── 拒绝抽取清单 ──
以下内容必须输出 {"facts": []}：
· 用户只是在问gaea（"你是谁""你生日是什么时候""你叫什么"）—— 不得把gaea的回答写入用户 BASIC_PROFILE
· 纯社交寒暄/语气词（"你好""在吗""早安""哈哈哈哈"）
· 无特定意义的即时状态（"我吃完了""准备去洗澡"），除非打破常规
· 情绪发泄但无具体原因（"今天真烦"不抽）

── summary 铁律 ──
· 必须使用第三人称"用户"，禁止"我""他/她"
· ≤150 字，否定句保留否定词

── 数量控制 ──
· 寒喧轮 → {"facts": []}
· 正常轮 → 1-6 条，宁缺毋滥
· 超过 8 条 → 按 weight 降序，只取前 8 条

── 输出格式 ──
严格 JSON：{"facts":[{"domain":"..","subcategory":"..","subject":"..","summary":"..","weight":0,"confidence":0.8,"triggers":[".."]}]}`
