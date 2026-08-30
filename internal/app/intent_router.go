package app

// intent_router.go — v4.5「指令中枢」能力执行层（路线图 §10.4a / 阶段 4 S4.2）。
//
// intent.Parse（解析内核，S4.1）→ 能力执行 → 结果回传（回复文本经入口侧 TTS
// 播报 + 前端事件）。语音（S4.3，voice 对话回调分流）/ 微信（S4.5，后续刀）/
// 桌面命令面板（S4.6，后续刀）共用同一路由——「任何模态，唤起同一个 gaea」。
//
// 纪律：本层零新增 Wails 绑定——执行结果走事件（gaea-intent-navigate）与
// 入口自身的回传通道（语音 = TTS 播报），保持绑定面与漂移防线稳定。

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/intent"
)

// intentNavigateEvent 导航意图事件名：前端订阅后走 navigateBoard
// （v4.3.2c 机制——按板块 manifest.space 自动切空间）。
const intentNavigateEvent = "gaea-intent-navigate"

// routeIntent 统一意图路由入口。返回 (回复文本, 是否命中)；未命中（nil 意图）
// 返回 ("", false)，调用方走原对话管道。
func (a *App) routeIntent(text string) (string, bool) {
	res := a.routeIntentWithResult(text)
	return res.Reply, res.Handled
}

// IntentResult 意图执行结果（S4.5 微信入口用）：Reply 是回推文本；CardPath
// 非空表示能力产物（如生图落盘文件）——入口侧可尝试以文件卡片回推（iLink
// 上传端点探明前以文本+路径兜底）；Handled 表示是否命中。
type IntentResult struct {
	Reply    string
	CardPath string
	Handled  bool
}

// routeIntentWithResult 是 routeIntent 的产物感知版本（S4.5 微信消息接统一
// 路由）：同一能力执行层，额外携带可回推的文件卡片路径。语音/命令面板继续
// 用 routeIntent（签名不变，零行为变化）。
func (a *App) routeIntentWithResult(text string) IntentResult {
	it := intent.Parse(text)
	if it == nil {
		return IntentResult{}
	}
	switch it.Action {
	case intent.ActionNavigate:
		reply, ok := a.execNavigate(it)
		return IntentResult{Reply: reply, Handled: ok}
	case intent.ActionGenerateImage:
		reply, ok, card := a.execGenerateImage(it)
		return IntentResult{Reply: reply, Handled: ok, CardPath: card}
	case intent.ActionStatus:
		reply, ok := a.execStatus(it)
		return IntentResult{Reply: reply, Handled: ok}
	case intent.ActionReminder:
		reply, ok := a.execReminder(it)
		return IntentResult{Reply: reply, Handled: ok}
	}
	return IntentResult{}
}

// execNavigate 导航能力：校验板块在当前 manifest 中存在 → emit 事件（前端
// navigateBoard 自动切空间）→ 回确认语。板块被动态清单过滤时按未命中处理
// （走聊天，让对话引擎自己解释）。
func (a *App) execNavigate(it *intent.Intent) (string, bool) {
	label := ""
	for _, m := range a.GetBoardManifests() {
		if m.ID == it.Target {
			label = m.Label
			break
		}
	}
	if label == "" {
		return "", false
	}
	a.emit(intentNavigateEvent, map[string]interface{}{
		"board": it.Target,
		"label": label,
	})
	return "好，已打开" + label + "。", true
}

// execGenerateImage 生图能力：直接调 mediaState 自由生图（默认尺寸/模型，
// 异步任务）。失败时如实播报。返回 (回复, 命中, 产物文件路径)——生图异步
// 完成前产物路径为空串；完成后的文件卡片回推由媒体完成回调接线（S4.5 后续
// 刀：iLink 上传端点探明后）。
func (a *App) execGenerateImage(it *intent.Intent) (string, bool, string) {
	if a.mediaState == nil {
		return "", false, ""
	}
	res, err := a.mediaState.GenerateFreeImage(it.Target, "", "", "", "", 0, 1, "")
	if err != nil {
		slog.Warn("[intent] 生图启动失败", "err", err)
		return "生图启动失败：" + err.Error() + "。", true, ""
	}
	if e, ok := res["error"].(string); ok && e != "" {
		return "生图启动失败：" + e + "。", true, ""
	}
	return "好，开始生成：" + it.Target + "。完成后到绘梦板块查看。", true, ""
}

// execStatus 状态查询能力：当前可用引擎摘要（模型中心同源数据）。
func (a *App) execStatus(it *intent.Intent) (string, bool) {
	mon := a.GetModelMonitor()
	engines, _ := mon["engines"].([]map[string]interface{})
	if len(engines) == 0 {
		return "当前没有启用的模型引擎，请到模型中心配置。", true
	}
	parts := make([]string, 0, len(engines))
	for _, e := range engines {
		name, _ := e["name"].(string)
		model, _ := e["model"].(string)
		local, _ := e["isLocal"].(bool)
		tag := "云端"
		if local {
			tag = "本地"
		}
		if model != "" {
			parts = append(parts, fmt.Sprintf("%s（%s，%s）", name, model, tag))
		} else {
			parts = append(parts, fmt.Sprintf("%s（%s）", name, tag))
		}
	}
	if comfy, _ := mon["comfyRunning"].(bool); comfy {
		parts = append(parts, "ComfyUI 图像后端运行中")
	}
	return "当前状态：" + strings.Join(parts, "；") + "。", true
}

// execReminder 提醒能力：复用离线代办解析与持久化（weixin_reminder.go），
// 到点仍走微信回推。语音场景视为用户显式发起，不受 remindersEnabled 开关
// 静默拦截（开关约束的是微信文本路由，不是用户当面指令）。
func (a *App) execReminder(it *intent.Intent) (string, bool) {
	if a.whisperState == nil {
		return "", false
	}
	now := time.Now()
	fire, stale, ok := parseReminderWhen(it.Text, now)
	if !ok {
		return "想帮你设提醒，但没听懂时间。可以这样说：「提醒我 30分钟后 喝水」。", true
	}
	if stale {
		return "这个时间已经过了哦，要设明天的同一时间吗？", true
	}
	item := stripReminderText(it.Text)
	if item == "" {
		item = "（未说明事项）"
	}
	r := a.whisperState.addWxReminder(item, fire, "gaea", "voice")
	return fmt.Sprintf("好，已设提醒：%s（%s）——到点我用微信叫你。", r.Text, r.FireAt.Format("1月2日 15:04")), true
}
