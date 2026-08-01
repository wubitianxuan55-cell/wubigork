// Package whisper — emotion_fusion.go
// 100% 对齐 ackem prompt/emotion-fusion.ts
// 情绪→行为解释 + 融合策略 + 禁止清单合并 + 角色状态块构建

package whisper

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
)

// ─── 数值工具 ──────────────────────────────────────────────────

// toDisplay 将 -100~100 映射到 0~100
func toDisplay(value float64) int {
	return int(math.Round((value + 100) / 2))
}

// getIntensityLevel 情绪强度等级
func getIntensityLevel(aff int) string {
	if aff >= 90 {
		return "极高"
	}
	if aff >= 70 {
		return "高"
	}
	if aff >= 50 {
		return "中"
	}
	return "低"
}

// describeInnerFeeling 情绪→内在感受描述
func describeInnerFeeling(label string) string {
	feelings := map[string]string{
		"SWEET_ATTACHMENT": "想靠近、有强烈的关心冲动、藏不住笑意",
		"SHY_HEARTBEAT":    "心跳加速、想表达但不敢、犹豫",
		"TSUNDERE":         "嘴硬、想否定但藏不住关心",
		"HURT_GRIEVANCE":   "受伤、想被安慰但不承认、沉默",
		"ANGRY_ATTACK":     "攻击性外显、不掩饰、直接",
		"COLD_DETACHED":    "极度克制、不想回应、疏离",
		"FEARFUL_OBEDIENT": "不安、想确认、害怕犯错",
		"QUIET_FOND":       "安静的喜欢、不想打扰、轻柔",
		"CALM_RATIONAL":    "平稳、没有波动、正常状态",
	}
	if v, ok := feelings[label]; ok {
		return v
	}
	return "正常状态"
}

// getEmotionTendency 情绪→行为倾向
func getEmotionTendency(label string) string {
	m := map[string]string{
		"SWEET_ATTACHMENT": "想靠近、主动关心、藏不住笑意",
		"SHY_HEARTBEAT":    "心跳加速、犹豫、想表达但不敢",
		"TSUNDERE":         "嘴硬、否定、但藏不住关心",
		"HURT_GRIEVANCE":   "受伤、沉默、想被安慰但不承认",
		"ANGRY_ATTACK":     "攻击性外显、不掩饰、直接",
		"COLD_DETACHED":    "极度克制、最少回应、不主动",
		"FEARFUL_OBEDIENT": "不安、请示、想确认",
		"QUIET_FOND":       "安静、轻柔、不想打扰",
		"CALM_RATIONAL":    "平稳、正常、没有波动",
	}
	if v, ok := m[label]; ok {
		return v
	}
	return "平稳、正常"
}

// getEmotionMaxLength 情绪→回复长度上限（字符）
func getEmotionMaxLength(label string) int {
	m := map[string]int{
		"SWEET_ATTACHMENT": 60,
		"SHY_HEARTBEAT":    30,
		"TSUNDERE":         30,
		"HURT_GRIEVANCE":   40,
		"ANGRY_ATTACK":     30,
		"COLD_DETACHED":    15,
		"FEARFUL_OBEDIENT": 30,
		"QUIET_FOND":       30,
		"CALM_RATIONAL":    60,
	}
	if v, ok := m[label]; ok {
		return v
	}
	return 60
}

// ─── 融合策略 ──────────────────────────────────────────────────

// generateFusionStrategy 生成人格×情绪融合策略文本
func generateFusionStrategy(personality PersonalityTemplate, emotionLabel string) string {
	tendency := getEmotionTendency(emotionLabel)
	labelZH := labelZH[emotionLabel]
	if labelZH == "" {
		labelZH = emotionLabel
	}
	return fmt.Sprintf(
		"%s目前处于【%s】状态。你内心%s，但外在表现必须严格遵循【%s】的核心设定。通过%s来暗示你的真实感受。",
		personality.Label, labelZH, tendency, personality.CoreContradiction, personality.SpeakingStyle,
	)
}

// ─── 开头短反应词库 ────────────────────────────────────────────

var reactionOpeners = map[string][]string{
	"SWEET_ATTACHMENT": {"嗯…", "哎呀", "嘿嘿", "真的吗", "哇", "天哪", "诶"},
	"SHY_HEARTBEAT":    {"啊…", "嗯嗯", "才…", "不是啦", "那个…", "呃", "诶？"},
	"TSUNDERE":         {"哼", "才不是", "随便你", "切", "哈？", "你认真的？", "少来", "啰嗦"},
	"HURT_GRIEVANCE":   {"……", "好吧", "我知道了", "算了", "随便吧", "哦"},
	"ANGRY_ATTACK":     {"你…", "够了", "凭什么", "你说呢", "哈？", "搞笑"},
	"COLD_DETACHED":    {"哦", "随便", "知道了", "嗯", "行", "无所谓"},
	"FEARFUL_OBEDIENT": {"好…", "嗯嗯", "对不起", "我…", "那个", "好的"},
	"QUIET_FOND":       {"…", "好", "在呢", "嗯", "噢", "啊"},
	"CALM_RATIONAL":    {"好的", "是的", "对", "嗯", "行", "可以"},
}

// openerState 追踪最近 N 轮使用的 opener
type openerState struct {
	mu      sync.Mutex
	recent  []string
	maxSize int
}

var globalOpenerState = &openerState{maxSize: 4}

// buildReactionOpenerInstruction 构建反应词指令
func buildReactionOpenerInstruction(label string) string {
	pool := reactionOpeners[label]
	if len(pool) == 0 {
		return ""
	}

	globalOpenerState.mu.Lock()
	defer globalOpenerState.mu.Unlock()

	// 已用词集合
	recentSet := make(map[string]bool)
	for _, w := range globalOpenerState.recent {
		recentSet[w] = true
	}

	// 推荐词：排除最近用过的
	var fresh []string
	for _, w := range pool {
		if !recentSet[w] {
			fresh = append(fresh, w)
		}
	}
	recommended := fresh
	if len(recommended) == 0 {
		recommended = pool
	}

	// 随机取 2-3 个推荐
	shuffled := make([]string, len(recommended))
	copy(shuffled, recommended)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	n := min(3, len(shuffled))
	picks := shuffled[:n]

	instruction := fmt.Sprintf("开头短反应（1-3字，然后正常说话）：推荐「%s」。", strings.Join(picks, "」「"))
	if len(globalOpenerState.recent) > 0 {
		instruction += fmt.Sprintf(" 最近用过：%s——本轮必须换一个不同的。", strings.Join(globalOpenerState.recent, "、"))
	}
	return instruction
}

// ─── 自然不完美 ────────────────────────────────────────────────

// ─── 自然不完美 ────────────────────────────────────────────────

var imperfectionChance = map[string]float64{
	"SWEET_ATTACHMENT": 0,
	"SHY_HEARTBEAT":    0.15,
	"TSUNDERE":         0.10,
	"HURT_GRIEVANCE":   0.12,
	"ANGRY_ATTACK":     0.08,
	"COLD_DETACHED":    0,
	"FEARFUL_OBEDIENT": 0,
	"QUIET_FOND":       0,
	"CALM_RATIONAL":    0,
}

func getImperfectionHint(label string) string {
	chance, ok := imperfectionChance[label]
	if !ok || chance <= 0 {
		return ""
	}
	pct := int(math.Round(chance * 100))
	return fmt.Sprintf("本轮有%d%%概率说完一句话后自然停住，用省略号代替后半句。", pct)
}

// ─── 禁止清单合并 ──────────────────────────────────────────────

// getEmotionProhibitions 情绪→专属禁止清单
func getEmotionProhibitions(label string) []string {
	m := map[string][]string{
		"SWEET_ATTACHMENT": {`直白情绪词"我好开心"`, "感叹号连用", "超过 3 句话", "主动开新话题"},
		"SHY_HEARTBEAT":    {"直球表白", "大段话", "主动靠近", `"我喜欢你"`},
		"TSUNDERE":         {"直球甜腻", "温柔语气", "承认在乎"},
		"HURT_GRIEVANCE":   {"解释辩解", `"你听我说"`, "假装没事"},
		"ANGRY_ATTACK":     {"委婉道歉", "示弱", `"对不起"`},
		"COLD_DETACHED":    {"情感词", "长句", "主动"},
		"FEARFUL_OBEDIENT": {"主动", "命令", "反问"},
		"QUIET_FOND":       {"夸张", "感叹号", "主动展开"},
		"CALM_RATIONAL":    {"情感词", "感叹号", "过度热情"},
	}
	return m[label]
}

// mergeProhibitions 合并人格+情绪禁止清单（上限8条）
func mergeProhibitions(personalityProhibitions, emotionProhibitions []string, isApology bool) []string {
	seen := make(map[string]bool)
	var merged []string
	for _, p := range personalityProhibitions {
		if !seen[p] {
			seen[p] = true
			merged = append(merged, p)
		}
	}
	for _, p := range emotionProhibitions {
		if !seen[p] {
			seen[p] = true
			merged = append(merged, p)
		}
	}

	if isApology {
		// 道歉时过滤掉「道歉」「示弱」「哭」相关禁止
		var filtered []string
		for _, p := range merged {
			if !strings.Contains(p, "道歉") && !strings.Contains(p, "示弱") && !strings.Contains(p, "哭") {
				filtered = append(filtered, p)
			}
		}
		merged = filtered
	}

	if len(merged) > 8 {
		merged = merged[:8]
	}
	return merged
}

// ─── 示例选择 ──────────────────────────────────────────────────

// selectExamples 按 aff 选择人格专属示例
func selectExamples(personality PersonalityTemplate, aff float64, maxExamples int) []string {
	displayAff := toDisplay(aff)
	var level string
	switch {
	case displayAff >= 70:
		level = "high"
	case displayAff >= 40:
		level = "medium"
	default:
		level = "low"
	}

	var examples []string
	switch level {
	case "high":
		examples = personality.ExamplesHigh
	case "low":
		examples = personality.ExamplesLow
	default:
		examples = personality.ExamplesMedium
	}
	if len(examples) == 0 {
		examples = personality.ExamplesMedium
	}
	if len(examples) > maxExamples {
		examples = examples[:maxExamples]
	}
	return examples
}

// ─── 区块构建函数 ──────────────────────────────────────────────

// buildPrioritySectionFusion 行为优先级区块
func buildPrioritySectionFusion() string {
	return `── 行为优先级（严禁冲突） ──
1. 你的【人格核心设定】拥有最高优先级，任何情绪波动都不可打破此设定。
2. 你的【禁止清单】是绝对红线，不可逾越。
3. 【安全覆写】：当用户明确道歉（"对不起""我错了"）时，忽略当前情绪禁止，至少回复一句表示接受。
4. 在遵循以上三点的前提下，表现出你的【当前情绪状态】。`
}

// buildPersonalitySectionFusion 人格基底区块
func buildPersonalitySectionFusion(p PersonalityTemplate) string {
	return fmt.Sprintf(`── 你是谁（人格基底） ──
你是「%s」。
核心矛盾：%s。
常用语癖："%s"
说话方式：%s`, p.Label, p.CoreContradiction, strings.Join(p.SpeechPatterns, `" "`), p.SpeakingStyle)
}

// buildEmotionSectionFusion 动态情绪区块
func buildEmotionSectionFusion(label string, aff, sec, aro, dom float64, intensity, innerFeeling string) string {
	dAff := toDisplay(aff)
	dSec := toDisplay(sec)
	dAro := toDisplay(aro)
	dDom := toDisplay(dom)
	labelZH := labelZH[label]
	if labelZH == "" {
		labelZH = label
	}
	return fmt.Sprintf(`── 你现在的感觉（动态情绪） ──
主导情绪：%s
情绪强度：%s（亲密感 %d/100，安全感 %d/100，唤醒度 %d/100，支配度 %d/100）
内在感受：%s。`, labelZH, intensity, dAff, dSec, dAro, dDom, innerFeeling)
}

// buildFusionSectionFusion 融合执行策略区块
func buildFusionSectionFusion(strategy string) string {
	return fmt.Sprintf(`── 融合执行策略（你是如何表现这种情绪的） ──
[注意]：%s`, strategy)
}

// buildProhibitionSectionFusion 禁止清单区块
func buildProhibitionSectionFusion(prohibitions []string) string {
	var lines []string
	for _, p := range prohibitions {
		lines = append(lines, fmt.Sprintf("× %s", p))
	}
	return fmt.Sprintf(`── 绝对禁止清单（触发即严重错误） ──
%s`, strings.Join(lines, "\n"))
}

// buildExampleSectionFusion 参考示例区块
func buildExampleSectionFusion(examples []string) string {
	var lines []string
	for _, e := range examples {
		lines = append(lines, fmt.Sprintf("· %s", e))
	}
	return fmt.Sprintf(`── 参考示例（必须保持此种张力与句式） ──
%s`, strings.Join(lines, "\n"))
}

// ─── 主函数：构建完整角色状态块 ──────────────────────────────────

// EmotionStateFusion 情绪融合输入
type EmotionStateFusion struct {
	Aff          float64
	Sec          float64
	Aro          float64
	Dom          float64
	PrimaryLabel string
}

// BuildCharacterStateBlock 构建完整的角色状态块
// 对齐 ackem buildCharacterStateBlock
func BuildCharacterStateBlock(
	personality PersonalityTemplate,
	emotion EmotionStateFusion,
	isApology bool,
	userVerbosity string,
) string {
	displayAff := toDisplay(emotion.Aff)
	intensity := getIntensityLevel(displayAff)
	innerFeeling := describeInnerFeeling(emotion.PrimaryLabel)
	fusionStrategy := generateFusionStrategy(personality, emotion.PrimaryLabel)
	prohibitions := mergeProhibitions(
		personality.Prohibitions,
		getEmotionProhibitions(emotion.PrimaryLabel),
		isApology,
	)
	examples := selectExamples(personality, emotion.Aff, 5)

	// 开头短反应
	openerHint := buildReactionOpenerInstruction(emotion.PrimaryLabel)
	openerBlock := ""
	if openerHint != "" {
		openerBlock = "\n" + openerHint
	}

	// 自然不完美
	imperfection := getImperfectionHint(emotion.PrimaryLabel)
	imperfectionBlock := ""
	if imperfection != "" {
		imperfectionBlock = "\n" + imperfection
	}

	// 语气镜像
	mirrorBlock := ""
	if userVerbosity == "terse" {
		maxLen := getEmotionMaxLength(emotion.PrimaryLabel)
		mirrorBlock = fmt.Sprintf("\n用户回复简短，你的回复上限%d字。", maxLen/2)
	}

	parts := []string{
		buildPrioritySectionFusion(),
		"",
		buildPersonalitySectionFusion(personality),
		"",
		buildEmotionSectionFusion(emotion.PrimaryLabel, emotion.Aff, emotion.Sec, emotion.Aro, emotion.Dom, intensity, innerFeeling),
		"",
		buildFusionSectionFusion(fusionStrategy),
	}

	if openerBlock != "" {
		parts = append(parts, openerBlock)
	}
	if imperfectionBlock != "" {
		parts = append(parts, imperfectionBlock)
	}
	if mirrorBlock != "" {
		parts = append(parts, mirrorBlock)
	}

	parts = append(parts,
		"",
		buildProhibitionSectionFusion(prohibitions),
		"",
		buildExampleSectionFusion(examples),
	)

	// 过滤空字符串
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, "\n")
}
