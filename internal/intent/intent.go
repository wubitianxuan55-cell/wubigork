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
	ActionNavigate      Action = "navigate"       // 打开/切换板块（Target = 板块 id）
	ActionGenerateImage Action = "generate_image" // 画一张…（Target = 画面描述）
	ActionEditImage     Action = "edit_image"     // 改这张图…（Target = 编辑指令，v4.9 对话式改图）
	ActionStatus        Action = "status"         // 查询状态（Target = 模型/引擎）
	ActionReminder      Action = "reminder"       // 离线代办提醒（时间解析在执行层）
	ActionReadScreen    Action = "read_screen"    // 读一下屏幕（屏幕感知：截屏 + OCR，v4.7 S4.6）
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
	{"微信助手", "weixin"}, {"微信", "weixin"},
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

// 改图（对话式改图，v4.9）：保守双门槛，(a) 编辑语义动词 与 (b) 指代已收到
// 的图 必须同时满足才命中——宁漏勿误。「这张图好看吗」（无编辑动词）、「
// 把背景音乐关掉」（指代不是图）都不命中；「改图/修图/P图」词头本身即指代
// 刚发来的那张图，同时计入两门槛。
var reEditImageVerb = regexp.MustCompile(
	`改图|修图|(?i:P图)|重绘|修改|改成|换成|换上|修成|(?i:P成)|调成|变成|去掉|删除|删掉|加上|添加|改|换`)
var reEditImageRef = regexp.MustCompile(
	`这张图|这图|那个图|那张图|这个图|上一张图|刚才那(?:张)?图|图中|图里|它|背景|改图|修图|(?i:P图)`)

// reEditImageLead 指代前缀（剥出编辑指令用）：语气词 + 可选把/将 + 指代 +
// 连接助词/标点。只剥句首前缀，句中指代保留（「去掉图里的路人」整句即编辑
// 指令）；剥完没有实质内容按未命中（宁漏勿误）。
var reEditImageLead = regexp.MustCompile(
	`^(?:请|麻烦|帮我|给我)*(?:把|将)?` +
		`(?:这张图|这图|那个图|那张图|这个图|上一张图|刚才那(?:张)?图|它|背景|图中|图里|改图|修图|(?i:P图))` +
		`(?:的|里|中|里面|之中的)?[，,：:、\s]*`)

// extractEditImageQuery 提取编辑指令全文：剥掉句首指代前缀（「把这张图的」
// 整段去掉），剥不动就回原文（引擎侧整句照常可用作编辑指令）；剥完只剩
// 标点/空白视为空串（调用方按未命中处理）。
func extractEditImageQuery(t string) string {
	if loc := reEditImageLead.FindStringIndex(t); loc != nil && loc[0] == 0 {
		t = t[loc[1]:]
	}
	q := strings.TrimSpace(t)
	if strings.Trim(q, "。，,：:、！!？?…～~ ") == "" {
		return ""
	}
	return q
}

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
// 优先级：导航 > 生图 > 改图 > 读屏 > 状态 > 提醒（窄规则优先，宽匹配殿后）。
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

	// ②.5 改图（v4.9 对话式改图）：编辑动词 × 图片指代 双门槛；生图优先序
	// 不变（上一跳已返回），「画一张图」绝不被改图误吞。Target = 去掉指代
	// 前缀的编辑指令全文。
	if reEditImageVerb.MatchString(t) && reEditImageRef.MatchString(t) {
		if q := extractEditImageQuery(t); q != "" {
			return &Intent{Action: ActionEditImage, Target: q, Text: t}
		}
	}

	// ③ 读屏（屏幕感知；明确指向「屏幕」才命中；Target 带显示器选择，v4.8）
	if reReadScreen.MatchString(t) {
		return &Intent{Action: ActionReadScreen, Target: readScreenTarget(t), Text: t}
	}

	// ④ 状态
	if reStatus.MatchString(t) {
		return &Intent{Action: ActionStatus, Target: "model", Text: t}
	}

	// ⑤ 提醒
	if reReminder.MatchString(t) {
		return &Intent{Action: ActionReminder, Target: "", Text: t}
	}

	return nil
}
