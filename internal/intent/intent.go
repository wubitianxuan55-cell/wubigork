// Package intent — v4.5「指令中枢」意图解析（路线图 §10.4a / 阶段 4 S4.1）。
//
// 触点层「同内核多入口」的解析内核：语音 / 微信 / 桌面命令面板共用的
// 「意图 → 能力」第一跳。纯函数规则引擎——零延迟、零成本、可表驱动测试；
// LLM 兜底分类器未来作为一个宽匹配 Capability 挂在 Router 尾部（本包留位）。
//
// 纪律：宁可漏判（走聊天管道）也不误判——闲聊「画得不错」绝不能触发一张废图。
package intent

import (
	"fmt"
	"regexp"
	"strings"
)

// Action 意图动作（执行层据此分派能力）。
type Action string

const (
	ActionNavigate       Action = "navigate"         // 打开/切换板块（Target = 板块 id）
	ActionGenerateImage  Action = "generate_image"   // 画一张…（Target = 画面描述）
	ActionStatus         Action = "status"           // 查询状态（Target = 模型/引擎）
	ActionReminder       Action = "reminder"         // 离线代办提醒（时间解析在执行层）
	ActionReadScreen     Action = "read_screen"      // 读一下屏幕（屏幕感知：截屏 + OCR，v4.7 S4.6）
	ActionSendLatestFile Action = "send_latest_file" // 把(刚才/最新)文件/产物发给我（最新产物文件卡回推，v4.41 微信文件收发）
)

// Intent 解析结果；Parse 未命中返回 nil（调用方走原聊天管道）。
type Intent struct {
	Action Action
	Target string
	Text   string // 原文
}

// ─── 板块别名表（中文产品词汇表；值 = board manifest id）────────
// 长别名必须排在同前缀短别名之前（贪婪匹配）。执行层仍需校验 id 在
// 当前 manifest 中存在（动态清单可能过滤板块）。

var boardAliases = []struct{ alias, id string }{
	{"首页", "home"}, {"主页", "home"},
	{"轻语", "chat"}, {"会客厅", "chat"}, {"聊天", "chat"},
	{"小说", "novel"}, {"写作", "novel"},
	{"绘梦", "imagegen"}, {"画室", "imagegen"}, {"生图", "imagegen"}, {"画图", "imagegen"},
	{"办公", "gaea"},
	{"造价数据库", "cost"}, {"造价库", "cost"}, {"造价", "cost"}, {"成本库", "cost"},
	{"编程", "code"}, {"编码", "code"}, {"代码", "code"},
	{"记忆中枢", "memoryhub"}, {"记忆", "memoryhub"}, {"知识库", "memoryhub"},
	{"模型中心", "modelcenter"}, {"模型管理", "modelcenter"}, {"模型", "modelcenter"},
	{"角色库", "characterlib"}, {"角色", "characterlib"}, {"人物", "characterlib"},
	{"设置", "settings"}, {"偏好", "settings"},
	{"青鸟", "weixin"}, {"微信助手", "weixin"}, {"微信", "weixin"}, // 青鸟 = v4.48 板块名
}

// lookupBoard 别名匹配。
func lookupBoard(text string) (string, bool) {
	for _, a := range boardAliases {
		if strings.Contains(text, a.alias) {
			return a.id, true
		}
	}
	return "", false
}

// ─── 规则 ─────────────────────────────────────────────────────

// 强导航动词：出现即导航意图成立（仍需板块别名命中）。
var reNavStrong = regexp.MustCompile(`(?:打开|打开一下|进入|切换到|切到|跳到|转到|看看|看一下)`)

// 弱导航动词：仅「回/去 + 首页/主页」这种强搭配才算。
var reNavHome = regexp.MustCompile(`(?:回到|回|返回|去)(?:首页|主页)`)

// 生图：画/生成 + 非口语续接。「画得不错/画了半天/画过」不命中（字符类排除）；
// 描述至少 1 字符——「画猫」成立。
var reImage = regexp.MustCompile(
	`^(?:请|麻烦|帮我)?(?:给我)?画(?:一张|一幅|一个|张|出)?([^得过了]{1,})$` +
		`|^(?:请|麻烦|帮我)?(?:给我)?生成(?:一张|一幅|一个)?(?:的)?(?:图片|图像|画|一张图)(?:[，,：:]|一下)?(.*)$`)

// 状态查询。
var reStatus = regexp.MustCompile(`(?:现在)?(?:用的?|在用|是什么|什么|哪个)(?:模型|引擎)|(?:模型|引擎)(?:状态|怎么样|是啥)|当前(?:模型|引擎)`)

// 产物推送（v4.41 微信文件收发刀）：用户要把「最新的那份产物」经文件卡回推。
// 三式全部首尾锚定 + 方向词「发…给我/发我」强约束 + 名词白名单，宁漏勿误：
//   - 把字式（①）：「把刚才的文件发给我」「把最新的报告发我」「帮我把表格传给我」；
//   - 发我式（②）：「发我一下刚才的文件」「发我最新产物」；
//   - 发送式（③）：只放行产物义强名词（产物|报告|成品）——「发送产物」「发报告」；
//     裸「文件/文档/表格」无方向词时语义含混（可能是要发给第三者），不命中。
//
// 不命中样例：「发消息给我」（消息不在白名单）、「把文件删了」（无发送动词）、
// 「发文件给他」（方向不是 我）、「提醒我把报告发给老板」（句首提醒，归提醒位）。
var reSendLatestFile = regexp.MustCompile(
	`^(?:请|麻烦|帮我)?把(?:刚才|最新|刚刚|上一个|上一份)?(?:的)?(?:那个|这个|这份)?` +
		`(?:文件|报告|文档|表格|产物|成品)(?:发|传)(?:给|到)?我(?:一下)?(?:吧|吗|么)?$` +
		`|^(?:请|麻烦|帮我)?发我(?:一下|一份)?(?:刚才|最新|刚刚)?(?:的)?` +
		`(?:文件|报告|文档|表格|产物|成品)(?:吧|吗|么)?$` +
		`|^(?:请|麻烦|帮我)?(?:给我)?发送?(?:一下|一份)?(?:刚才|最新|刚刚)?(?:的)?` +
		`(?:产物|报告|成品)(?:给我|给我一下)?(?:吧|吗|么)?$` +
		// v4.41.2 放宽第四式：指代/名词 + 尾式「发给我」（真机实证「重新整理后
		// 发给我」这类口语未命中导致坠回聊天管道）。仍要求句中有产物名词或
		// 指代词，裸「发给我」不命中（宁漏勿误）。
		`|^(?:请|麻烦|帮我)?(?:把|将)?(?:它|这个|那个|这份|那些|刚才的?)(?:文件|报告|文档|表格|产物|成品)?` +
		`(?:直接)?(?:发|传)(?:给|到)?我(?:一下)?(?:吧|吗|么)?$`)

// reWxModifyAndSend 「改完再发」复合请求：「整理/修改/润色…后发给我」。微信侧
// 没有文档改写回传闭环——这类请求命中后由执行层给诚实能力答复，绝不能坠回
// 聊天管道（v4.41 真机实证：模型会声称「已整理好发你」而实际什么都没发）。
var reWxModifyAndSend = regexp.MustCompile(
	`(?:整理|重排|修改|改好|改完|润色|完善|修正|校对|优化|处理|排版)` +
		`[^。？！]{0,12}(?:之?后|再|然后)?(?:请|麻烦|帮我)?(?:把|将)?(?:它|这个|那个|这份)?` +
		`(?:文件|报告|文档|表格|产物|成品)?(?:直接)?(?:发|传)(?:给|到)?我(?:一下)?(?:吧|吗|么)?$`)

// reWxReminderish 提醒让位守卫：消息含提醒/定时字样时产物推送分支整体让位
// （「提醒我发产物给我」这类罕见组合归提醒位，宁漏勿误）。
var reWxReminderish = regexp.MustCompile(`提醒|定时`)

// 读屏（屏幕感知，S4.6 收口；v4.8 读屏纵深扩展显示器选择）：必须明确指向
// 「屏幕」——读/念/看/识别/截 + 屏幕/主屏/副屏/第N屏，或「屏幕上有什么/写了
// 什么」。窄规则纪律：不含裸「读/看」（那是导航/闲聊），主屏/副屏同样必须
// 带动词（「主屏幕坏了」不触发截图）。
var reReadScreen = regexp.MustCompile(
	`(?:读|念|看|识别|截).{0,2}(?:屏幕|主屏|副屏|第\s*[0-9一二三四五六七八九十]+\s*(?:块|个)?\s*屏幕?)` +
		`|屏幕.{0,2}(?:读|念|看)|屏幕上.{0,4}(?:什么|啥)|读屏`)

// 读屏序数修饰（读屏纵深 v4.8）：「第二块屏幕/第2屏」选中第 N 块显示器；
// 「主屏/副屏」选中主/副。仅在 reReadScreen 已命中的分支里解释——不扩大
// 误触发面。
var reScreenOrdinal = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十]+)\s*(?:块|个)?\s*屏`)
var reScreenPrimary = regexp.MustCompile(`主屏`)
var reScreenSecondary = regexp.MustCompile(`副屏`)

// readScreenTarget 从读屏命中文本解析显示器选择：缺省整屏（多显示器合并），
// "screen:N" = 第 N 块（1 起），"screen:primary" = 主屏，副屏按第 2 块处理。
// 序数解析不出（如「第百屏」）诚实回退整屏——宁漏勿误。
func readScreenTarget(t string) string {
	if m := reScreenOrdinal.FindStringSubmatch(t); m != nil {
		if n := parseCnOrdinal(m[1]); n > 0 {
			return fmt.Sprintf("screen:%d", n)
		}
	}
	if reScreenPrimary.MatchString(t) {
		return "screen:primary"
	}
	if reScreenSecondary.MatchString(t) {
		return "screen:2"
	}
	return "screen"
}

// parseCnOrdinal 中文/数字序数 → int。支持纯数字、一..九、「十」「十X」「X十」
// 「X十Y」组合（X/Y ∈ 一..九）；不支持返回 0。
func parseCnOrdinal(s string) int {
	return cnOrdinal(s)
}

var cnDigits = map[rune]int{
	'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
	'六': 6, '七': 7, '八': 8, '九': 9,
}

// cnOrdinal 「第二/2/十一/二十三」→ int；解析不出返回 0。
func cnOrdinal(s string) int {
	if s == "" {
		return 0
	}
	// 纯阿拉伯数字。
	allDigit := true
	for _, r := range s {
		if r < '0' || r > '9' {
			allDigit = false
			break
		}
	}
	if allDigit {
		n := 0
		for _, r := range s {
			n = n*10 + int(r-'0')
		}
		if n > 99 {
			return 0 // 序数面板：超过两位数的「屏幕编号」视为没听懂
		}
		return n
	}
	// 中文组合：[X]十[Y] / 单字 / 纯十。
	n := 0
	if i := strings.IndexRune(s, '十'); i >= 0 {
		x := 1
		if i > 0 {
			x = 0
			for _, r := range s[:i] {
				d, ok := cnDigits[r]
				if !ok {
					return 0
				}
				x = x*10 + d
			}
		}
		n += x * 10
		rest := s[i+len("十"):]
		for _, r := range rest {
			d, ok := cnDigits[r]
			if !ok {
				return 0
			}
			n += d
		}
		return n
	}
	for _, r := range s {
		d, ok := cnDigits[r]
		if !ok {
			return 0
		}
		n = n*10 + d
	}
	if n > 99 {
		return 0
	}
	return n
}

// 提醒类（宽匹配放最后；时间解析在执行层）。
var reReminder = regexp.MustCompile(`提醒|叫我`)

// Parse 解析一条自然语言指令；未命中返回 nil。
// 优先级：导航 > 生图 > 读屏 > 状态 > 产物推送 > 提醒（窄规则优先，宽匹配殿后）。
func Parse(text string) *Intent {
	t := strings.TrimSpace(text)
	t = strings.TrimRight(t, "。.！!？?？")
	t = strings.TrimSpace(t)
	if t == "" {
		return nil
	}

	// ① 导航：板块别名命中 + （强动词 或 回/去+首页 搭配）
	if id, ok := lookupBoard(t); ok {
		if reNavStrong.MatchString(t) || reNavHome.MatchString(t) {
			return &Intent{Action: ActionNavigate, Target: id, Text: t}
		}
	}

	// ② 生图
	if m := reImage.FindStringSubmatch(t); m != nil {
		desc := strings.TrimSpace(m[1])
		if desc == "" {
			desc = strings.TrimSpace(m[2])
		}
		if desc != "" {
			return &Intent{Action: ActionGenerateImage, Target: desc, Text: t}
		}
	}

	// ②.5 读屏（屏幕感知；明确指向「屏幕」才命中；Target 带显示器选择，v4.8）
	if reReadScreen.MatchString(t) {
		return &Intent{Action: ActionReadScreen, Target: readScreenTarget(t), Text: t}
	}

	// ④ 状态
	if reStatus.MatchString(t) {
		return &Intent{Action: ActionStatus, Target: "model", Text: t}
	}

	// ④.5 产物推送（v4.41）：锚定窄规则，先于宽匹配的提醒位（「提醒我把报告
	// 发给老板」句首带提醒字样，锚定式不命中、自然落到提醒位，互不干扰）。
	// v4.41.2：含提醒/定时字样整体让位；「整理…后发给我」复合请求（不含产物
	// 名词、进不了 reSendLatestFile）独立识别，命中后由执行层给诚实能力答复
	//（Target=modify_and_send）——绝不能坠回聊天管道。
	if !reWxReminderish.MatchString(t) {
		if reSendLatestFile.MatchString(t) {
			target := "latest"
			if reWxModifyAndSend.MatchString(t) {
				target = "modify_and_send"
			}
			return &Intent{Action: ActionSendLatestFile, Target: target, Text: t}
		}
		if reWxModifyAndSend.MatchString(t) {
			return &Intent{Action: ActionSendLatestFile, Target: "modify_and_send", Text: t}
		}
	}

	// ⑤ 提醒
	if reReminder.MatchString(t) {
		return &Intent{Action: ActionReminder, Target: "", Text: t}
	}

	return nil
}
