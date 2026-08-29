// Package intent — v4.5「指令中枢」意图解析（路线图 §10.4a / 阶段 4 S4.1）。
//
// 触点层「同内核多入口」的解析内核：语音 / 微信 / 桌面命令面板共用的
// 「意图 → 能力」第一跳。纯函数规则引擎——零延迟、零成本、可表驱动测试；
// LLM 兜底分类器未来作为一个宽匹配 Capability 挂在 Router 尾部（本包留位）。
//
// 纪律：宁可漏判（走聊天管道）也不误判——闲聊「画得不错」绝不能触发一张废图。
package intent

import (
	"regexp"
	"strings"
)

// Action 意图动作（执行层据此分派能力）。
type Action string

const (
	ActionNavigate      Action = "navigate"       // 打开/切换板块（Target = 板块 id）
	ActionGenerateImage Action = "generate_image" // 画一张…（Target = 画面描述）
	ActionStatus        Action = "status"         // 查询状态（Target = 模型/引擎）
	ActionReminder      Action = "reminder"       // 离线代办提醒（时间解析在执行层）
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

// 提醒类（宽匹配放最后；时间解析在执行层）。
var reReminder = regexp.MustCompile(`提醒|叫我`)

// Parse 解析一条自然语言指令；未命中返回 nil。
// 优先级：导航 > 生图 > 状态 > 提醒（窄规则优先，宽匹配殿后）。
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

	// ③ 状态
	if reStatus.MatchString(t) {
		return &Intent{Action: ActionStatus, Target: "model", Text: t}
	}

	// ④ 提醒
	if reReminder.MatchString(t) {
		return &Intent{Action: ActionReminder, Target: "", Text: t}
	}

	return nil
}
