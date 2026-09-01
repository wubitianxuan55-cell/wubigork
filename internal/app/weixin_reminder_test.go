package app

import (
	"strings"
	"testing"
	"time"
)

// reminderTestApp 构造带临时数据根的 whisperState（提醒文件落 tmp，互不污染）。
func reminderTestApp(t *testing.T) *whisperState {
	t.Helper()
	return newChatServiceTestApp(t).whisperState
}

func reminderNow() time.Time {
	// 固定 2026-09-01（周二）10:00 本地时区：绝对/顺延断言均可静态推算。
	return time.Date(2026, 9, 1, 10, 0, 0, 0, time.Local)
}

// ─── parseReminderWhen 表驱动 ─────────────────────────────────

func TestParseReminderWhen(t *testing.T) {
	now := reminderNow()
	cases := []struct {
		name    string
		text    string
		want    time.Time
		stale   bool
		wantOK  bool
	}{
		{name: "N分钟后", text: "30分钟后 喝水", want: now.Add(30 * time.Minute), wantOK: true},
		{name: "N分后", text: "5分后 起身", want: now.Add(5 * time.Minute), wantOK: true},
		{name: "N小时后", text: "2小时后 开会", want: now.Add(2 * time.Hour), wantOK: true},
		{name: "中文数字小时后", text: "三个小时后 修图", want: now.Add(3 * time.Hour), wantOK: true},
		{name: "N天后", text: "2天后 体检", want: now.AddDate(0, 0, 2), wantOK: true},
		{name: "今天未来时刻", text: "今天18:30 下班", want: time.Date(2026, 9, 1, 18, 30, 0, 0, time.Local), wantOK: true},
		{name: "今天已过时刻stale", text: "今天8点 晨会", want: time.Date(2026, 9, 1, 8, 0, 0, 0, time.Local), stale: true, wantOK: true},
		{name: "明天时刻", text: "明天早上9点 站会", want: time.Date(2026, 9, 2, 9, 0, 0, 0, time.Local), wantOK: true},
		{name: "明早合并词", text: "明早8点半 跑步", want: time.Date(2026, 9, 2, 8, 30, 0, 0, time.Local), wantOK: true},
		{name: "明晚换算", text: "明晚8点 追剧", want: time.Date(2026, 9, 2, 20, 0, 0, 0, time.Local), wantOK: true},
		{name: "后天中文数字", text: "后天十点 洗牙", want: time.Date(2026, 9, 3, 10, 0, 0, 0, time.Local), wantOK: true},
		{name: "裸时刻已过顺延", text: "18:30 接孩子", want: time.Date(2026, 9, 1, 18, 30, 0, 0, time.Local), wantOK: true},
		{name: "裸时刻已过则明天", text: "9点 晨会", want: time.Date(2026, 9, 2, 9, 0, 0, 0, time.Local), wantOK: true},
		{name: "下午中文数字", text: "下午三点 取件", want: time.Date(2026, 9, 1, 15, 0, 0, 0, time.Local), wantOK: true},
		{name: "点半", text: "明天下午2点半 复诊", want: time.Date(2026, 9, 2, 14, 30, 0, 0, time.Local), wantOK: true},
		{name: "无段词字面解释", text: "明天9点 站会", want: time.Date(2026, 9, 2, 9, 0, 0, 0, time.Local), wantOK: true},
		{name: "点分", text: "明天9点15分 值机", want: time.Date(2026, 9, 2, 9, 15, 0, 0, time.Local), wantOK: true},
		{name: "全角冒号", text: "明天14：30 评审", want: time.Date(2026, 9, 2, 14, 30, 0, 0, time.Local), wantOK: true},
		{name: "无时间不成立", text: "明天再说吧", wantOK: false},
		{name: "纯闲聊不成立", text: "今天天气怎么样", wantOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, stale, ok := parseReminderWhen(tc.text, now)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v，期望 %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if !got.Equal(tc.want) {
				t.Fatalf("时间 = %v，期望 %v", got, tc.want)
			}
			if stale != tc.stale {
				t.Fatalf("stale = %v，期望 %v", stale, tc.stale)
			}
		})
	}
}

// ─── stripReminderText ────────────────────────────────────────

func TestStripReminderText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"提醒我 30分钟后 喝水", "喝水"},
		{"明天早上9点开站会", "开站会"},
		{"提醒我18:30接孩子", "接孩子"},
		{"记得明天2点半去复诊", "去复诊"},
		{"提醒我开会", "开会"}, // 无时间表达：事项仍应剥离触发词
	}
	for _, tc := range cases {
		if got := stripReminderText(tc.in); got != tc.want {
			t.Errorf("stripReminderText(%q) = %q，期望 %q", tc.in, got, tc.want)
		}
	}
}

// ─── tryWxReminder 路由 ───────────────────────────────────────

func TestTryWxReminder_HitsAndPersists(t *testing.T) {
	a := reminderTestApp(t)

	reply, handled := a.tryWxReminder("提醒我 30分钟后 站会", "gaea")
	if !handled {
		t.Fatal("提醒类消息应被路由接管")
	}
	if !strings.Contains(reply, "站会") {
		t.Errorf("确认回复应含事项，实际 %q", reply)
	}
	list := a.WeixinReminderList()
	if len(list) != 1 {
		t.Fatalf("应有 1 条提醒，实际 %d", len(list))
	}
	if list[0]["status"] != wxReminderStatusPending || list[0]["source"] != "weixin" {
		t.Errorf("提醒状态/来源不符: %v", list[0])
	}

	// 重启恢复：新实例加载同一数据根
	b := &whisperState{whisperDataRoot: a.whisperDataRoot}
	b.loadReminders()
	if len(b.reminders) != 1 || b.reminders[0].Text != "站会" {
		t.Errorf("重启后提醒应恢复，实际 %+v", b.reminders)
	}
}

func TestTryWxReminder_PassesThroughChat(t *testing.T) {
	a := reminderTestApp(t)
	if _, handled := a.tryWxReminder("今天天气怎么样", "gaea"); handled {
		t.Error("闲聊不应被提醒路由接管")
	}
	// 提醒关键词命中但无时间表达：接管并回「格式提示」，不坠入闲聊
	reply, handled := a.tryWxReminder("帮我提醒一下", "gaea")
	if !handled {
		t.Error("提醒关键词命中应接管（即使解析失败回提示）")
	}
	if reply == "" {
		t.Error("解析失败应回复格式提示")
	}
}

func TestTryWxReminder_DisabledConfig(t *testing.T) {
	a := reminderTestApp(t)
	if err := a.WeixinReminderSetConfig(`{"remindersEnabled": false}`); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	if _, handled := a.tryWxReminder("提醒我 5分钟后 喝水", "gaea"); handled {
		t.Error("开关关闭后提醒路由不应接管")
	}
	if err := a.WeixinReminderSetConfig(`{"remindersEnabled": true}`); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	if _, handled := a.tryWxReminder("提醒我 5分钟后 喝水", "gaea"); !handled {
		t.Error("开关重开后提醒路由应接管")
	}
}

// ─── tickReminders 回推循环 ───────────────────────────────────

func TestTickReminders_PushAndPersist(t *testing.T) {
	a := reminderTestApp(t)
	now := reminderNow()

	a.addWxReminder("站会", now.Add(-time.Minute), "gaea", "weixin")
	a.addWxReminder("喝水", now.Add(time.Hour), "gaea", "weixin") // 未到期

	var pushed []string
	n := a.tickReminders(now, func(assistantID, text string) error {
		pushed = append(pushed, assistantID+"|"+text)
		return nil
	})
	if n != 1 || len(pushed) != 1 {
		t.Fatalf("应推送 1 条，实际 %d（%v）", n, pushed)
	}
	if !strings.Contains(pushed[0], "站会") {
		t.Errorf("推送内容应含事项，实际 %q", pushed[0])
	}
	// 已推送标 done 且落盘；未到期保持 pending
	list := a.WeixinReminderList()
	byStatus := map[string]int{}
	for _, r := range list {
		byStatus[r["status"].(string)]++
	}
	if byStatus[wxReminderStatusDone] != 1 || byStatus[wxReminderStatusPending] != 1 {
		t.Fatalf("状态分布不符: %v", byStatus)
	}

	// 重启恢复后 done 不再重推
	b := &whisperState{whisperDataRoot: a.whisperDataRoot}
	b.loadReminders()
	if n := b.tickReminders(now, func(string, string) error { return nil }); n != 0 {
		t.Errorf("done 提醒不应重推，实际 %d", n)
	}
}

func TestTickReminders_FailRetryThenGiveUp(t *testing.T) {
	a := reminderTestApp(t)
	now := reminderNow()
	a.addWxReminder("复诊", now.Add(-time.Minute), "gaea", "weixin")

	// 前 4 次失败：保持 pending
	for i := 0; i < wxReminderMaxFails-1; i++ {
		if n := a.tickReminders(now, func(string, string) error {
			return errStubWxPush
		}); n != 0 {
			t.Fatalf("失败轮不应计数推送，第 %d 轮", i)
		}
	}
	list := a.WeixinReminderList()
	if list[0]["status"] != wxReminderStatusPending || list[0]["failCount"] != wxReminderMaxFails-1 {
		t.Fatalf("失败计数不符: %v", list[0])
	}
	// 第 5 次成功：done
	if n := a.tickReminders(now.Add(time.Second), func(string, string) error { return nil }); n != 1 {
		t.Fatal("重试成功应推送")
	}
}

var errStubWxPush = &wxPushError{"mock push down"}

type wxPushError struct{ s string }

func (e *wxPushError) Error() string { return e.s }

func TestTickReminders_GivesUpAfterMaxFails(t *testing.T) {
	a := reminderTestApp(t)
	now := reminderNow()
	a.addWxReminder("取件", now.Add(-time.Minute), "gaea", "weixin")

	for i := 0; i < wxReminderMaxFails; i++ {
		a.tickReminders(now, func(string, string) error { return errStubWxPush })
	}
	list := a.WeixinReminderList()
	if list[0]["status"] != wxReminderStatusFailed {
		t.Fatalf("连续 %d 次失败应标 failed，实际 %v", wxReminderMaxFails, list[0]["status"])
	}
	// failed 不再重试
	if n := a.tickReminders(now, func(string, string) error { return nil }); n != 0 {
		t.Error("failed 提醒不应再推送")
	}
}

// ─── 绑定面校验 ───────────────────────────────────────────────

func TestWeixinReminderBindings(t *testing.T) {
	a := reminderTestApp(t)
	// 合法未来时间取真实时钟 +1h，而非固定的 reminderNow()+1h：WeixinReminderAdd
	// 的「必须在未来」校验按真实 time.Now() 判定——固定基准在当日 11:00 后运行
	// 会误报失败（2026-09-01 实测时间炸弹），本测试又无静态推算诉求。
	fire := time.Now().Add(time.Hour).Format(time.RFC3339)

	res, err := a.WeixinReminderAdd("交周报", fire)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := a.WeixinReminderAdd("", fire); err == nil {
		t.Error("空事项应报错")
	}
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	if _, err := a.WeixinReminderAdd("过去时间", past); err == nil {
		t.Error("过去时间应报错")
	}
	if _, err := a.WeixinReminderAdd("坏时间", "not-a-time"); err == nil {
		t.Error("非法时间应报错")
	}

	list := a.WeixinReminderList()
	if len(list) != 1 || list[0]["text"] != "交周报" {
		t.Fatalf("列表不符: %v", list)
	}

	if err := a.WeixinReminderDelete(res["id"].(string)); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := a.WeixinReminderDelete("wxr-404"); err == nil {
		t.Error("删除不存在提醒应报错")
	}
	if len(a.WeixinReminderList()) != 0 {
		t.Error("删除后列表应为空")
	}

	cfg, err := a.WeixinReminderConfig()
	if err != nil || cfg["remindersEnabled"] != true {
		t.Fatalf("默认配置应为开启: %v %v", cfg, err)
	}
}
