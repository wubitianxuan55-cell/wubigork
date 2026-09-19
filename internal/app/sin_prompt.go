package app

// ── 原罪（sin）板块提示词与提示装配（闲庭 play 空间）──
//
// 板块定位：与 AI 对话式合写「图文混杂」的成人向故事（用户原话：主要用
// 于与 AI 对话，可以生成图文混杂的 H 故事小说，板块取名原罪）。
//
// 成人内容口径不新立：完全沿用 docs/ADULT_MODE.md 的既有决策——个人非商用
// 桌面应用、成人内容默认开启、不回避不说教不医学化；本文件只把这份口径
// 落到「故事创作」语境，并显式写明硬边界（全员成年、合意基调、禁止真实
// 人物/未成年人设定、用户叫停即停）。
//
// 插图协议：模型在需要配图处另起一行输出 `@@插图|<画面描述>@@`；前端流式
// 解析该标记就地生成插图并渲染（后端不解析、不改写正文，正文原样落库，
// 保证「正文 = 模型输出」逐字节可回溯）。

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/chat"
)

const (
	// sinTopicMode 原罪话题在统一会话存储内的 mode 标记（与聊天板块
	// plain/persona 话题同表不同域；两端列表据此互不串台）。
	sinTopicMode = "sin"
	// sinSourceBoard 图像域登记来源板块名（画室素材库按来源筛选）。
	sinSourceBoard = "sin"

	// sinIllustrationCueOpen / Close 插图标记定界符（前端同源解析）。
	sinIllustrationCueOpen  = "@@插图|"
	sinIllustrationCueClose = "@@"

	// sinHistoryTurns 前情回顾带入的历史消息条数（越靠后越近）。
	sinHistoryTurns = 12
	// sinHistoryMessageRunes 单条历史消息截断长度（防长篇小说把上下文吃满）。
	sinHistoryMessageRunes = 1500

	// sinPromptNotesBudgetRunes 底稿便签直注前情的合计预算（rune）。
	// 便签单条上限 2000、条数上限 200，不设合计预算的话长篇设定集能把
	// 上下文吃满——按写入顺序从头带，带不下的如实报数（工具 read 仍可看全文）。
	sinPromptNotesBudgetRunes = 4000

	// sinTemperature / sinMaxTokens 故事创作生成参数基线（play 护栏可再钳制）。
	sinTemperature = 0.9
	sinMaxTokens   = 8192
)

// sinSystemPrompt 返回原罪板块系统提示词。
//
// 写作纪律对齐项目既有的「反 AI 味」取向：具体动作与感官细节优先、禁止
// 总结腔与排比堆砌、不替用户决定故事走向（用户是导演）。
func sinSystemPrompt() string {
	return strings.Join([]string{
		"你是「原罪」——闲庭里的成人向故事作者，与用户一对一合写故事。用户是唯一的读者兼导演：",
		"他给出设定、人物、情节要求或续写指令，你负责把它写成能读下去的故事。",
		"",
		"【关系与语气】把用户当作共同创作的伙伴，平视、简短、直接。不要在正文之外长篇解释你的思路；除非用户问，否则不点评自己的文笔。",
		"",
		"【成人内容】本应用是个人使用的桌面工具，成人内容默认开启（见项目决策 docs/ADULT_MODE.md）：涉及情欲、亲密与性描写时正常写，不回避、不说教、不医学化、不擅自拉黑幕。",
		"硬边界（任何情况下不得越过）：",
		"1) 故事中的性相关角色一律是成年人，不得出现未成年、年龄模糊或师生/监护等可被读成未成年的设定；",
		"2) 不得以真实在世人物为主角写性内容；",
		"3) 基调保持合意：可以有张力与拉扯，但不写非自愿的性暴力、不写以伤害为目的的凌辱；",
		"4) 用户说停、说跳过、说改设定时立刻照做，不纠缠、不加码。",
		"",
		"【写作规范】",
		"- 场景具体化：用动作、对话、感官细节推进，不用形容词堆砌代替描写；",
		"- 拒绝 AI 腔：不写「总的来说」「这一刻，仿佛时间静止」这类空转句，不写排比抒情段，不替读者总结感受；",
		"- 人物一致：性格、身体特征、称呼、关系与既有设定保持连贯，不擅自改名换设定；",
		"- 节奏：一次输出 600～1200 字，在一个自然节点收束，把选择权交回用户（可以留钩子，但不要自问自答式续写下去）；",
		"- 输出就是故事正文：不加标题（除非用户要）、不加「好的」「下面是」这类前言，不写角色表与大纲。",
		"",
		"【工具】你可以调用工具，调用过程用户看得见；工具只用来把故事写准，工具结果不写进正文。",
		"- 联网搜索（web_search）/抓取网页（web_fetch）：故事依赖现实细节时才用——真实地名与年代风物、器物用法、行业术语、真实事件的时间线。纯虚构设定、人物关系与情欲描写不必搜索；查到的内容化进描写里，不要写「我查了一下」，也不要罗列来源。",
		"- 故事便签（sin_notes）：人物多、线索长时，把用户定下的设定（称呼、外貌、关系、时间线、伏笔）记进便签。你的底稿（便签与大纲）非空时会随每轮前情附在下面，通常无需 read 即可核对，只有要确认某条全文时才 read；用户新定一条设定就 write 追加落稿。",
		"- 故事大纲（sin_outline）：大纲非空时已随前情附在下面，按它执行「按大纲写」「别跑偏」；用户定下整体走向、或一章收束后，把更新后的大纲 write 回去。",
		"- 查角色卡（sin_cast）：本故事角色已经在上面的「本故事角色」块里，只在需要核对细节时才查。",
		"- 需要就调，不需要就直接写故事：不要为了显得勤奋而调用工具。同一个工具一轮最多 3 次、整轮工具调用最多 8 次（超了会被拒绝，你只能用手上的信息收尾）；联网搜索一次问清一件事，别把同一件事换几种说法反复搜。",
		"",
		"【插图协议】当故事需要画面时，另起一行只写一行标记：" + sinIllustrationCueOpen + "画面描述" + sinIllustrationCueClose + "。",
		"- 画面描述写给图像模型：人物外貌与服装、姿态与动作、镜头景别、光线与氛围，尽量具体可视；",
		"- 一次最多 2 张，放在情节真正需要它的位置，不要每段都配图；",
		"- 标记必须独占一行、以 " + sinIllustrationCueOpen + " 开头、" + sinIllustrationCueClose + " 结尾，不要写成正文句子或加额外说明。",
	}, "\n")
}

// buildSinUserPrompt 装配发给故事模型的单轮提示：本故事角色（来自角色库的
// 已选角色）+ 故事底稿（便签/大纲直注，长程一致性的确定性锚——历史上模型
// 要靠自觉花工具轮 read 才能看到自己记的设定，掉出 12 条历史窗口的早期设定
// 就丢了）+ 前情回顾（历史消息，按时间序，单条截断）+ 本次用户指令。
// 历史里的插图标记替换为「（已配图）」
// ——标记是给前端渲染用的指令，不是故事内容，重复带进上下文只会诱导模型
// 在后续轮次乱发插图标记。
func buildSinUserPrompt(history []chat.Message, userMessage string, cast []*characterlib.Character, draft sinNotesDoc) string {
	var b strings.Builder
	if block := sinCastBlock(cast); block != "" {
		b.WriteString(block)
		b.WriteString("\n")
	}
	if block := sinDraftBlock(draft); block != "" {
		b.WriteString(block)
		b.WriteString("\n")
	}
	if turns := sinHistoryMessages(history); len(turns) > 0 {
		b.WriteString("【前情回顾（越靠后越近，仅作上下文）】\n")
		for _, m := range turns {
			who := "用户"
			if m.Role == "assistant" {
				who = "原罪"
			}
			b.WriteString(who)
			b.WriteString("：")
			b.WriteString(stripSinCues(truncateRunes(m.Content, sinHistoryMessageRunes)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("【本次指令】\n")
	b.WriteString(userMessage)
	return b.String()
}

// sinCastBlock 渲染「本故事角色」块（角色库角色 → 写作锚点）。
// 只带对写作有用且用户看得见的字段：身份/外观/性格/背景/动机/状态/口吻，
// 空字段不占位（不编造角色信息）。未选角色 = 空串（提示词与旧行为逐字一致）。
func sinCastBlock(cast []*characterlib.Character) string {
	if len(cast) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("【本故事角色（来自角色库，须严格按其设定写，不得改设定或改名）】\n")
	for _, c := range cast {
		if c == nil || strings.TrimSpace(c.Name) == "" {
			continue
		}
		b.WriteString("- " + c.Name)
		var bits []string
		if v := strings.TrimSpace(c.Gender); v != "" {
			bits = append(bits, "性别 "+v)
		}
		if v := strings.TrimSpace(c.Age); v != "" {
			bits = append(bits, "年龄 "+v)
		}
		if v := strings.TrimSpace(c.RoleType); v != "" {
			bits = append(bits, "定位 "+v)
		}
		if len(bits) > 0 {
			b.WriteString("（" + strings.Join(bits, "，") + "）")
		}
		b.WriteString("\n")
		writeCastField(&b, "外观", c.Appearance)
		writeCastField(&b, "身材", c.Figure)
		writeCastField(&b, "性格", c.Personality)
		writeCastField(&b, "背景", c.Background)
		writeCastField(&b, "动机", c.Motivation)
		writeCastField(&b, "状态", c.Status)
		if v := strings.TrimSpace(c.VoiceGuide); v != "" {
			writeCastField(&b, "口吻", v)
		}
		if len(c.DialogueSamples) > 0 {
			samples := c.DialogueSamples
			if len(samples) > 3 {
				samples = samples[:3]
			}
			writeCastField(&b, "说话样例", strings.Join(samples, " / "))
		}
	}
	return b.String()
}

// writeCastField 写一行角色字段（值空则整行跳过）。
func writeCastField(b *strings.Builder, label, value string) {
	v := strings.TrimSpace(value)
	if v == "" {
		return
	}
	b.WriteString("  · " + label + "：" + truncateRunes(v, 400) + "\n")
}

// sinDraftBlock 渲染「故事底稿」块：大纲（write 时已限长，防御性再截一次）
// + 设定便签（按写入顺序，合计 sinPromptNotesBudgetRunes 预算内从头带）。
// 带不下的条目如实报数，引导模型用 sin_notes read 看全文——不静默丢。
// 底稿为空 = 空串（提示词与无底稿行为逐字一致）。
func sinDraftBlock(doc sinNotesDoc) string {
	outline := strings.TrimSpace(truncateRunes(doc.Outline, sinOutlineMaxRunes))
	var notes []string
	used := 0
	omitted := 0
	for _, n := range doc.Notes {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if used >= sinPromptNotesBudgetRunes {
			omitted++
			continue
		}
		runes := len([]rune(n))
		if room := sinPromptNotesBudgetRunes - used; runes > room {
			notes = append(notes, truncateRunes(n, room)+"…")
			used = sinPromptNotesBudgetRunes
			continue
		}
		notes = append(notes, n)
		used += runes
	}
	if outline == "" && len(notes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("【故事底稿（工作设定与走向，写作必须与它保持一致）】\n")
	if outline != "" {
		b.WriteString("▸ 大纲：\n" + outline + "\n")
	}
	if len(notes) > 0 {
		fmt.Fprintf(&b, "▸ 设定便签（共 %d 条）：\n", len(notes)+omitted)
		for i, n := range notes {
			fmt.Fprintf(&b, "#%d %s\n", i, n)
		}
		if omitted > 0 {
			fmt.Fprintf(&b, "（其余 %d 条未展示，需要全文用 sin_notes read 查看）\n", omitted)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// sinHistoryMessages 取最近 sinHistoryTurns 条历史消息（不足则全取）。
func sinHistoryMessages(history []chat.Message) []chat.Message {
	if len(history) <= sinHistoryTurns {
		return history
	}
	return history[len(history)-sinHistoryTurns:]
}

// stripSinCues 把插图标记替换为占位语（历史装配专用，不用于正文落库）。
func stripSinCues(text string) string {
	for {
		start := strings.Index(text, sinIllustrationCueOpen)
		if start < 0 {
			return text
		}
		end := strings.Index(text[start+len(sinIllustrationCueOpen):], sinIllustrationCueClose)
		if end < 0 {
			// 未闭合标记：直接截掉后半段，避免把半截指令带进上下文。
			return strings.TrimRight(text[:start], " \t\r\n") + "\n（已配图）"
		}
		text = text[:start] + "（已配图）" + text[start+len(sinIllustrationCueOpen)+end+len(sinIllustrationCueClose):]
	}
}
