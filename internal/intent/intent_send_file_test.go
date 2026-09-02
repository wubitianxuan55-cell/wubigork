package intent

import "testing"

// intent_send_file_test.go — 产物推送意图（v4.41 微信文件收发刀）：reSendLatestFile
// 命中/不命中表驱动样例（宁漏勿误：方向词「发…给我/发我」+ 名词白名单双门槛）。

// TestParse_SendLatestFile_Hits 命中样例（≥6）：全部应命中 ActionSendLatestFile。
func TestParse_SendLatestFile_Hits(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"把刚才文件", "把刚才的文件发给我"},
		{"把最新报告", "把最新的报告发我"},
		{"帮我表格", "帮我把最新的表格发给我"},
		{"传给我", "把那个文档传给我"},
		{"发我一下", "发我一下刚才的文件"},
		{"发我产物", "发我最新产物"},
		{"发送产物", "发送产物"},
		{"裸发报告", "发报告"},
		{"发送最新报告", "发送最新报告"},
		{"把成品", "把成品发给我一下"},
		{"请字开头", "请把最新的文件发给我吧"},
		{"发我文件", "发我文件"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			it := Parse(tc.text)
			if it == nil {
				t.Fatalf("应命中：%s", tc.text)
			}
			if it.Action != ActionSendLatestFile {
				t.Fatalf("Action = %s，期望 %s（%s）", it.Action, ActionSendLatestFile, tc.text)
			}
			if it.Target != "latest" {
				t.Errorf("Target = %q，期望 latest", it.Target)
			}
		})
	}
}

// TestParse_SendLatestFile_Misses 不命中样例（≥3）：宁可漏判走聊天管道。
func TestParse_SendLatestFile_Misses(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"发消息给我", "发消息给我"},      // 名词不在白名单
		{"删除动词", "把文件删了"},       // 无发送动词
		{"方向不是 我", "发文件给他"},     // 方向词必须指向我
		{"提醒句式", "提醒我把报告发给老板"},  // 句首提醒归提醒位，锚定式不吞
		{"天气", "发一下最近的天气"},      // 天气不是产物名词
		{"裸发送", "发送"},           // 无名词
		{"询问句", "最新的报告写得怎么样"},   // 无发送动词
		{"打开文件", "把文件打开"},       // 无发送动词
		{"发红包", "给我发个红包"},       // 红包不是产物名词
		{"改图指代", "把这张图的背景换成海边"}, // 改图域，不越界
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if it := Parse(tc.text); it != nil && it.Action == ActionSendLatestFile {
				t.Fatalf("不应命中产物推送（%s → %s）", tc.text, it.Action)
			}
		})
	}
}

// TestParse_SendLatestFile_YieldsToReminder 提醒句式让位：句首带「提醒」的发送
// 诉求归提醒位（到点执行），产物推送不越界抢答。
func TestParse_SendLatestFile_YieldsToReminder(t *testing.T) {
	it := Parse("提醒我 30分钟后 把最新的报告发给老板")
	if it == nil || it.Action != ActionReminder {
		t.Fatalf("提醒句式应归 ActionReminder，got %+v", it)
	}
}
