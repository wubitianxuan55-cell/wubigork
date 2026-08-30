package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/screen"
)

// ─── routeIntent：能力执行层（v4.5 指令中枢 S4.2）──────────────

func TestRouteIntent_Navigate(t *testing.T) {
	a := newChatServiceTestApp(t)

	reply, handled := a.routeIntent("打开绘梦")
	if !handled {
		t.Fatal("导航指令应命中")
	}
	if !strings.Contains(reply, "绘梦") {
		t.Errorf("确认语应含板块名，实际 %q", reply)
	}

	// 不存在的板块 → 未命中（走聊天，不误导航）
	if _, handled := a.routeIntent("打开不存在的板块"); handled {
		t.Error("未知板块应按未命中处理")
	}
}

func TestRouteIntent_Status(t *testing.T) {
	a := newChatServiceTestApp(t)

	// chat_service_test 装配了 herdsman(本地) 引擎
	reply, handled := a.routeIntent("现在用什么模型")
	if !handled {
		t.Fatal("状态指令应命中")
	}
	if !strings.Contains(strings.ToLower(reply), "herdsman") {
		t.Errorf("状态摘要应含引擎名，实际 %q", reply)
	}
}

func TestRouteIntent_Reminder(t *testing.T) {
	a := newChatServiceTestApp(t)

	reply, handled := a.routeIntent("提醒我 5分钟后 喝水")
	if !handled {
		t.Fatal("提醒指令应命中")
	}
	if !strings.Contains(reply, "喝水") {
		t.Errorf("确认语应含事项，实际 %q", reply)
	}
	// 落盘：语音设的提醒在微信提醒列表可见（source=voice）
	list := a.whisperState.WeixinReminderList()
	if len(list) != 1 || list[0]["source"] != "voice" {
		t.Fatalf("提醒应落盘且 source=voice: %v", list)
	}

	// 无时间表达 → 命中 + 格式提示（不坠入聊天）
	if reply, handled := a.routeIntent("提醒我一下"); !handled || reply == "" {
		t.Errorf("无时间提醒应回格式提示，实际 handled=%v reply=%q", handled, reply)
	}
}

func TestRouteIntent_PassesThroughChat(t *testing.T) {
	a := newChatServiceTestApp(t)

	if _, handled := a.routeIntent("今天天气怎么样"); handled {
		t.Error("闲聊不应被指令路由接管")
	}
	if _, handled := a.routeIntent(""); handled {
		t.Error("空文本不应命中")
	}
}

func TestRouteIntent_GenerateImageGuard(t *testing.T) {
	a := newChatServiceTestApp(t)

	// mediaState 未装配（此测试 App 无媒体域）→ 生图能力优雅降级为未命中，
	// 让对话引擎接手解释，而不是报一堆错误。
	if _, handled := a.routeIntent("画一只橘猫"); handled {
		t.Error("mediaState 缺失时生图应降级为未命中")
	}
}

// S4.5 微信接统一路由：routeIntentWithResult 与 routeIntent 同源（包装关系），
// 命中返回 Reply+Handled，未命中返回零值——微信回调据此分流聊天管道。
func TestRouteIntentWithResultWrapsRouteIntent(t *testing.T) {
	a := newChatServiceTestApp(t)

	// 命中（status 无需副作用装配）：handled=true + Reply 非空
	res := a.routeIntentWithResult("现在用什么模型")
	if !res.Handled || res.Reply == "" {
		t.Fatalf("routeIntentWithResult(status) = %+v, want handled+reply", res)
	}
	// 与 routeIntent 返回一致（包装关系）
	reply, handled := a.routeIntent("现在用什么模型")
	if handled != res.Handled || reply != res.Reply {
		t.Fatalf("routeIntent 与 WithResult 不一致: (%q,%v) vs (%q,%v)", reply, handled, res.Reply, res.Handled)
	}

	// 未命中：零值（Handled=false、Reply 空、CardPath 空）
	res = a.routeIntentWithResult("今天天气怎么样")
	if res.Handled || res.Reply != "" || res.CardPath != "" {
		t.Fatalf("routeIntentWithResult(未命中) = %+v, want 零值", res)
	}
	if _, handled := a.routeIntent("今天天气怎么样"); handled {
		t.Fatal("routeIntent(未命中) 应返回 handled=false")
	}
}

// ─── GaeaRouteIntent：桌面命令面板入口（v4.7 S4.6）──────────────

// dry-run 预览：命中给「将发生什么」的诚实描述 + Action/Target，零副作用
// （提醒不落盘；导航不 emit——预览代码路径不触达执行层）。
func TestGaeaRouteIntent_DryRunPreview(t *testing.T) {
	a := newChatServiceTestApp(t)

	res := a.GaeaRouteIntent("打开绘梦", true)
	if !res.Handled || res.Action != "navigate" || res.Target != "imagegen" {
		t.Fatalf("导航预览 = %+v, want handled+navigate/imagegen", res)
	}
	if !strings.Contains(res.Reply, "将打开") || !strings.Contains(res.Reply, "绘梦") {
		t.Errorf("导航预览语应含「将打开」+板块名，实际 %q", res.Reply)
	}

	// 提醒 dry-run：零副作用（不落盘）+ 预览带事项与时间
	res = a.GaeaRouteIntent("提醒我 5分钟后 喝水", true)
	if !res.Handled || res.Action != "reminder" {
		t.Fatalf("提醒预览 = %+v, want handled+reminder", res)
	}
	if list := a.whisperState.WeixinReminderList(); len(list) != 0 {
		t.Fatalf("dry-run 提醒不应落盘: %v", list)
	}

	// 真执行（对照 dryRun=false）：命中且落盘 1 条
	if res := a.GaeaRouteIntent("提醒我 5分钟后 喝水", false); !res.Handled {
		t.Fatal("真执行提醒应命中")
	}
	if list := a.whisperState.WeixinReminderList(); len(list) != 1 {
		t.Fatalf("真执行提醒应落盘 1 条: %v", list)
	}
}

// dry-run 校验口径与执行层一致：未知板块 / 媒体域缺失 / 闲聊 → 未命中（零值），
// 面板不预览出一个执行不了的动作。
func TestGaeaRouteIntent_DryRunMiss(t *testing.T) {
	a := newChatServiceTestApp(t)

	if res := a.GaeaRouteIntent("打开不存在的板块", true); res.Handled {
		t.Errorf("未知板块 dry-run 应未命中，实际 %+v", res)
	}
	// 此测试 App 无媒体域 → 生图降级未命中（与 TestRouteIntent_GenerateImageGuard 同口径）
	if res := a.GaeaRouteIntent("画一只橘猫", true); res.Handled {
		t.Errorf("mediaState 缺失时生图 dry-run 应未命中，实际 %+v", res)
	}
	if res := a.GaeaRouteIntent("今天天气怎么样", true); res.Handled {
		t.Errorf("闲聊 dry-run 应未命中，实际 %+v", res)
	}
}

// GaeaRouteIntent(text,false) 与 routeIntentWithResult 同源一致（包装关系）：
// 语音/微信/面板三个入口共用同一执行层。
func TestGaeaRouteIntent_MatchesRouteIntentWithResult(t *testing.T) {
	a := newChatServiceTestApp(t)

	exec := a.GaeaRouteIntent("现在用什么模型", false)
	direct := a.routeIntentWithResult("现在用什么模型")
	if exec != direct {
		t.Fatalf("GaeaRouteIntent(exec) = %+v, routeIntentWithResult = %+v（应一致）", exec, direct)
	}
}

// 读屏能力（v4.7 S4.6 收口）：dry-run 出预览；真执行走 截屏→OCR 链——
// 测试环境无 OCR 服务/无显示器时走诚实失败回复，但一律 handled=true
// （能力已命中，失败要说出口，不坠回聊天管道）。
func TestGaeaRouteIntent_ReadScreen(t *testing.T) {
	a := newChatServiceTestApp(t)

	// dry-run：零副作用预览（不截屏）
	res := a.GaeaRouteIntent("读一下屏幕", true)
	if !res.Handled || res.Action != "read_screen" {
		t.Fatalf("读屏预览 = %+v, want handled+read_screen", res)
	}
	if !strings.Contains(res.Reply, "截取屏幕") {
		t.Errorf("读屏预览语应含「截取屏幕」，实际 %q", res.Reply)
	}

	// 真执行：handled=true + 非空回复（成功=文字/没识别出文字；失败=诚实报错）
	res = a.GaeaRouteIntent("读一下屏幕", false)
	if !res.Handled || res.Reply == "" {
		t.Fatalf("读屏执行 = %+v, want handled+reply", res)
	}
	t.Logf("读屏执行回复（环境相关）：%q", res.Reply)
}

// firstImageCardPath：生图产物 CardPath 提取（v4.8 接通微信产物回推的数据源）。
func TestFirstImageCardPath(t *testing.T) {
	if got := firstImageCardPath(map[string]interface{}{}); got != "" {
		t.Errorf("空结果应返回空串，实际 %q", got)
	}
	if got := firstImageCardPath(map[string]interface{}{"error": "x"}); got != "" {
		t.Errorf("错误结果应返回空串，实际 %q", got)
	}
	want := `D:\pics\a.png`
	res := map[string]interface{}{"images": []imageItem{{FilePath: want}, {FilePath: "b.png"}}}
	if got := firstImageCardPath(res); got != want {
		t.Errorf("应取首图路径 %q，实际 %q", want, got)
	}
	res = map[string]interface{}{"images": []imageItem{{FilePath: ""}}}
	if got := firstImageCardPath(res); got != "" {
		t.Errorf("空路径应返回空串，实际 %q", got)
	}
}

// 读屏纵深（v4.8）：显示器选择——dry-run 预览带屏幕编号；越界编号诚实报错
// （屏幕数在测试进程里实际枚举，环境无关）。
func TestGaeaRouteIntent_ReadScreenMonitors(t *testing.T) {
	a := newChatServiceTestApp(t)

	// 预览：主屏 / 第 2 块
	res := a.GaeaRouteIntent("读一下主屏", true)
	if !res.Handled || !strings.Contains(res.Reply, "主屏") {
		t.Errorf("主屏预览 = %+v", res)
	}
	res = a.GaeaRouteIntent("读第二块屏幕", true)
	if !res.Handled || !strings.Contains(res.Reply, "第 2 块屏幕") {
		t.Errorf("第2屏预览应含「第 2 块屏幕」: %+v", res)
	}

	// 越界：按本机实际屏幕数断言（枚举失败则整组跳过——环境无关口径）
	mons, err := screen.Monitors()
	if err != nil || len(mons) == 0 {
		t.Skipf("显示器枚举不可用（err=%v），跳过越界用例", err)
	}
	res = a.GaeaRouteIntent(fmt.Sprintf("读第 %d 块屏幕", len(mons)+1), false)
	if !res.Handled || !strings.Contains(res.Reply, fmt.Sprintf("只有 %d 块屏幕", len(mons))) {
		t.Errorf("越界读屏 = %+v, want 诚实报错「只有 %d 块屏幕」", res, len(mons))
	}

	// 单屏机器：第 1 块 = 合法请求，走真实截屏链（环境相关回复，仅断 handled）
	if len(mons) == 1 {
		res = a.GaeaRouteIntent("读第一块屏幕", false)
		if !res.Handled || res.Reply == "" {
			t.Errorf("单屏读第 1 块 = %+v, want handled+reply", res)
		}
	}
}
