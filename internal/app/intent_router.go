package app

// intent_router.go — v4.5「指令中枢」能力执行层（路线图 §10.4a / 阶段 4 S4.2）。
//
// intent.Parse（解析内核，S4.1）→ 能力执行 → 结果回传（回复文本经入口侧 TTS
// 播报 + 前端事件）。语音（S4.3，voice 对话回调分流）/ 微信（S4.5，回调分流）/
// 桌面命令面板（S4.6，GaeaRouteIntent 绑定 + dry-run 预览-确认制）共用同一路由
// ——「任何模态，唤起同一个 gaea」。
//
// 纪律：语音/微信入口零新增 Wails 绑定——执行结果走事件（gaea-intent-navigate）
// 与入口自身的回传通道（语音 = TTS 播报）。S4.6 例外（显式豁免）：命令面板是
// 前端入口，Wails 绑定是其唯一回传通道——新增 GaeaRouteIntent 一个绑定，且
// dryRun=true 只解析不执行（搜索框不是整句指令入口，预览-确认制落实宁漏勿误）。

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/screen"
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
// 上传端点探明前以文本+路径兜底）；Handled 表示是否命中。S4.6 起附带
// Action/Target（dry-run 预览与前端指令卡片用；omitempty 不影响既有消费）。
type IntentResult struct {
	Reply    string `json:"reply"`
	CardPath string `json:"cardPath,omitempty"`
	Handled  bool   `json:"handled"`
	Action   string `json:"action,omitempty"`
	Target   string `json:"target,omitempty"`
}

// routeIntentWithResult 是 routeIntent 的产物感知版本（S4.5 微信消息接统一
// 路由）：同一能力执行层，额外携带可回推的文件卡片路径。语音/微信继续用
// routeIntent（签名不变，零行为变化）。
func (a *App) routeIntentWithResult(text string) IntentResult {
	return a.routeIntentMode(text, false)
}

// GaeaRouteIntent 桌面命令面板前端入口（v4.7 S4.6）：dryRun=true 只解析与
// 校验（零副作用，能力不执行），前端据此渲染「指令」预览卡；用户显式确认
// （点击执行/回车）后才以 dryRun=false 真执行。搜索框收的是任意搜索词而非
// 整句指令——预览-确认制是「宁漏勿误」纪律在面板面的落地。
func (a *App) GaeaRouteIntent(text string, dryRun bool) IntentResult {
	return a.routeIntentMode(text, dryRun)
}

// routeIntentMode 统一执行路径：dryRun 走 intentPreview（零副作用），否则
// 按动作分派能力执行层。
func (a *App) routeIntentMode(text string, dryRun bool) IntentResult {
	it := intent.Parse(text)
	if it == nil {
		return IntentResult{}
	}
	if dryRun {
		return a.intentPreview(it)
	}
	switch it.Action {
	case intent.ActionNavigate:
		reply, ok := a.execNavigate(it)
		return IntentResult{Reply: reply, Handled: ok, Action: string(it.Action), Target: it.Target}
	case intent.ActionGenerateImage:
		reply, ok, card := a.execGenerateImage(it)
		return IntentResult{Reply: reply, Handled: ok, CardPath: card, Action: string(it.Action), Target: it.Target}
	case intent.ActionStatus:
		reply, ok := a.execStatus(it)
		return IntentResult{Reply: reply, Handled: ok, Action: string(it.Action), Target: it.Target}
	case intent.ActionReminder:
		reply, ok := a.execReminder(it)
		return IntentResult{Reply: reply, Handled: ok, Action: string(it.Action), Target: it.Target}
	case intent.ActionReadScreen:
		reply, ok := a.execReadScreen(it)
		return IntentResult{Reply: reply, Handled: ok, Action: string(it.Action), Target: it.Target}
	}
	return IntentResult{}
}

// boardLabel 板块在当前 manifest 中的展示名；不存在返回空串（动态清单可能
// 过滤板块——导航按未命中处理）。
func (a *App) boardLabel(id string) string {
	for _, m := range a.GetBoardManifests() {
		if m.ID == id {
			return m.Label
		}
	}
	return ""
}

// intentPreview dry-run 预览（S4.6）：不执行任何能力，只给「将发生什么」的
// 诚实描述。校验口径与执行层一致：板块不在 manifest / 媒体域缺失 / 提醒域
// 缺失都按未命中（零值）返回，避免面板预览出一个执行不了的动作。
func (a *App) intentPreview(it *intent.Intent) IntentResult {
	switch it.Action {
	case intent.ActionNavigate:
		label := a.boardLabel(it.Target)
		if label == "" {
			return IntentResult{}
		}
		return IntentResult{Reply: "将打开「" + label + "」板块", Action: string(it.Action), Target: it.Target, Handled: true}
	case intent.ActionGenerateImage:
		if a.mediaState == nil {
			return IntentResult{}
		}
		return IntentResult{Reply: "将生成图片：" + it.Target + "（默认模型与尺寸，完成后到绘梦查看）", Action: string(it.Action), Target: it.Target, Handled: true}
	case intent.ActionStatus:
		return IntentResult{Reply: "将查询当前模型引擎状态", Action: string(it.Action), Target: it.Target, Handled: true}
	case intent.ActionReminder:
		if a.whisperState == nil {
			return IntentResult{}
		}
		fire, stale, ok := parseReminderWhen(it.Text, time.Now())
		switch {
		case !ok:
			return IntentResult{Reply: "将设提醒（时间没听懂，执行后会提示正确格式）", Action: string(it.Action), Target: it.Target, Handled: true}
		case stale:
			return IntentResult{Reply: "将设提醒（该时间已过，执行后会询问是否顺延到明天同一时间）", Action: string(it.Action), Target: it.Target, Handled: true}
		}
		return IntentResult{Reply: fmt.Sprintf("将设提醒：%s（%s）——到点用微信叫你", stripReminderText(it.Text), fire.Format("1月2日 15:04")), Action: string(it.Action), Target: it.Target, Handled: true}
	case intent.ActionReadScreen:
		switch label := "将截取屏幕并识别屏幕上的文字（OCR）"; {
		case it.Target == "screen:primary":
			return IntentResult{Reply: "将截取主屏并识别屏幕上的文字（OCR）", Action: string(it.Action), Target: it.Target, Handled: true}
		case strings.HasPrefix(it.Target, "screen:"):
			if n, err := strconv.Atoi(strings.TrimPrefix(it.Target, "screen:")); err == nil && n > 0 {
				label = fmt.Sprintf("将截取第 %d 块屏幕并识别屏幕上的文字（OCR）", n)
			}
			return IntentResult{Reply: label, Action: string(it.Action), Target: it.Target, Handled: true}
		default:
			return IntentResult{Reply: label, Action: string(it.Action), Target: it.Target, Handled: true}
		}
	}
	return IntentResult{}
}

// execNavigate 导航能力：校验板块在当前 manifest 中存在 → emit 事件（前端
// navigateBoard 自动切空间）→ 回确认语。板块被动态清单过滤时按未命中处理
// （走聊天，让对话引擎自己解释）。
func (a *App) execNavigate(it *intent.Intent) (string, bool) {
	label := a.boardLabel(it.Target)
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

// execReadScreen 屏幕感知能力（v4.7 S4.6 收口「读一下屏幕」；v4.8 读屏纵深）：
// 截屏（可按显示器选择：Target="screen" 整个虚拟屏 / "screen:N" 第 N 块 /
// "screen:primary" 主屏）→ OCR → 文本回传（语音入口经 TTS 朗读，命令面板入口
// 内联展示）。截屏仅在用户显式说出指令时触发——触点层入口（语音/面板/微信
// 指令）已承担「显式发起」语义；截屏文件进系统临时目录，即用即删，不落工作区
// （read_screen_keep_last 开启时额外在 .gaea/exports 留「屏幕-最近.png」滚动
// 覆盖一份）。OCR 长文本（>300 字）且 read_screen_summary 开启时先用本地模型
// 压成口语化摘要再朗读（摘要只走本地 Herdsman，屏幕内容不出本机；摘要不可用
// 退回 300 字截断）。
func (a *App) execReadScreen(it *intent.Intent) (string, bool) {
	img, err := a.captureScreenTarget(it.Target)
	if err != nil {
		slog.Warn("[intent] 截屏失败", "err", err)
		return "截屏失败：" + err.Error() + "。", true
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "截屏处理失败：" + err.Error() + "。", true
	}
	tmp, err := os.CreateTemp("", "gaea-screen-*.png")
	if err != nil {
		return "截屏暂存失败：" + err.Error() + "。", true
	}
	path := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(path)
		return "截屏暂存失败：" + err.Error() + "。", true
	}
	if err := tmp.Close(); err != nil {
		os.Remove(path)
		return "截屏暂存失败：" + err.Error() + "。", true
	}
	defer os.Remove(path)
	text, err := a.GaeaOCRText(path)
	if err != nil {
		slog.Warn("[intent] 屏幕文字识别失败", "err", err)
		return "屏幕文字识别失败：" + err.Error() + "。", true
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "屏幕上没有识别出文字。", true
	}
	// 留档（可选，默认关）：exports 会进工位检索面——开关文案必须明示这一权衡。
	keepNote := ""
	if a.cfg != nil && a.cfg.GetReadScreenKeepLast() {
		if p, kerr := a.keepScreenShot(path); kerr == nil {
			keepNote = "（截图已留档：" + p + "）"
		} else {
			slog.Warn("[intent] 读屏截图留档失败", "err", kerr)
		}
	}
	// TTS/面板展示口径：长文本截断（完整内容引导走识图/截图流程）；
	// 摘要可用且开启时优先摘要（更短的口语化朗读）。
	runes := []rune(text)
	if len(runes) > 300 {
		if a.cfg != nil && a.cfg.GetReadScreenSummary() {
			if sum, ok := a.summarizeReadScreenText(text); ok {
				return "屏幕上的内容概要：" + sum + keepNote, true
			}
		}
		text = string(runes[:300]) + "……（屏幕文字较长，已截断）"
	}
	return "屏幕上的文字如下：" + text + keepNote, true
}

// captureScreenTarget 按 Target 选择捕获区域：缺省/未知值 = 整个虚拟屏
// （多显示器合并，向后兼容 v4.7）；"screen:N" = 第 N 块显示器（越界诚实
// 报错）；"screen:primary" = 主屏。
func (a *App) captureScreenTarget(target string) (image.Image, error) {
	if rest, ok := strings.CutPrefix(target, "screen:"); ok {
		mons, merr := screen.Monitors()
		if merr != nil {
			return nil, merr
		}
		if rest == "primary" {
			for _, m := range mons {
				if m.Primary {
					return screen.CaptureArea(m.X, m.Y, m.W, m.H)
				}
			}
			return nil, fmt.Errorf("没找到主屏（枚举到 %d 块，均无主屏标记）", len(mons))
		}
		n, perr := strconv.Atoi(rest)
		if perr != nil || n < 1 {
			return nil, fmt.Errorf("屏幕编号没听懂：%q", rest)
		}
		if n > len(mons) {
			return nil, fmt.Errorf("这台电脑只有 %d 块屏幕，读不了第 %d 块", len(mons), n)
		}
		m := mons[n-1]
		return screen.CaptureArea(m.X, m.Y, m.W, m.H)
	}
	return screen.Capture()
}

// summarizeReadScreenText 把 OCR 长文本压成 ≤200 字口语化摘要（读屏纵深
// v4.8）。隐私纪律：只走本地 Herdsman——路由若回退到云端引擎（herdsman
// 不可用/停用）则放弃摘要，屏幕内容不出本机；任何一步失败返回 ("", false)，
// 调用方退回 300 字截断（v4.7 既有口径，零行为回归）。
func (a *App) summarizeReadScreenText(text string) (string, bool) {
	if a == nil || a.core == nil || a.cfg == nil || a.engineMgr == nil || !a.cfg.GetReadScreenSummary() {
		return "", false
	}
	eng, model, _ := a.routeHerdsmanLocal("office", "screen-local")
	if eng != "herdsman" || model == "" {
		return "", false
	}
	prov, err := provider.NewLLM("", provider.Config{Name: "read-screen-summary", Model: model, Engine: eng})
	if err != nil {
		return "", false
	}
	// 输入截断，避免本地模型上下文/超时失控（先例 visionAINormalize 6000 rune）。
	src := text
	if rs := []rune(src); len(rs) > 6000 {
		src = string(rs[:6000])
	}
	const sysPrompt = "你是屏幕朗读助手。把屏幕 OCR 文本压缩成不超过 200 字的口语化中文摘要，" +
		"保留关键数字、标题与待办事项，去掉窗口栏、菜单按钮名与重复噪音，直接输出摘要正文，不要任何解释。"

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	ch, err := prov.Stream(ctx, provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: sysPrompt},
			{Role: provider.RoleUser, Content: "屏幕 OCR 文本：\n" + src},
		},
		Temperature: 0,
	})
	if err != nil {
		return "", false
	}
	var out strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", false
		case chunk, ok := <-ch:
			if !ok {
				if s := strings.TrimSpace(out.String()); s != "" {
					return s, true
				}
				return "", false
			}
			switch chunk.Type {
			case provider.ChunkText:
				out.WriteString(chunk.Text)
			case provider.ChunkError:
				return "", false
			}
		}
	}
}

// keepScreenShot 把最近一次读屏截图以固定名「屏幕-最近.png」滚动覆盖存进
// 当前空间 exports 目录（read_screen_keep_last 开启时）。返回相对工作区路径
// 供回复与前端定位。
func (a *App) keepScreenShot(tmpPath string) (string, error) {
	dir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(dir, "屏幕-最近.png")
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", err
	}
	rel, rerr := filepath.Rel(gaeaCwd(), out)
	if rerr != nil {
		return filepath.ToSlash(out), nil
	}
	return filepath.ToSlash(rel), nil
}
