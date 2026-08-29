package app

import (
	"strings"
	"testing"
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
