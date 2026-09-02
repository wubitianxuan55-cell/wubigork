package intent

import "testing"

// intent_sendfile_v41412_test.go — v4.41.2 真机修复回归：放宽「发给我」尾式、
// 「改完再发」复合请求诚实能力答复位、提醒字样让位。

// TestParse_SendLatestFile_V41412_Hits 放宽后新增命中样例（含指代式）。
func TestParse_SendLatestFile_V41412_Hits(t *testing.T) {
	cases := map[string]string{
		"把它发给我":   "latest",
		"把这个文件发给我": "latest",
		"把那个发我":    "latest",
		"刚才的产物发给我": "latest",
	}
	for text, want := range cases {
		it := Parse(text)
		if it == nil || it.Action != ActionSendLatestFile || it.Target != want {
			t.Fatalf("%q 应命中 ActionSendLatestFile/%s，got %+v", text, want, it)
		}
	}
}

// TestParse_SendLatestFile_ModifyAndSend 复合请求命中后必须打 modify_and_send
// 标记（执行层给诚实能力答复，绝不坠回聊天管道——真机幻觉实证）。
func TestParse_SendLatestFile_ModifyAndSend(t *testing.T) {
	for _, text := range []string{"重新整理后发给我", "改好后发给我", "帮我润色一下这个报告再发给我", "处理后传给我"} {
		it := Parse(text)
		if it == nil || it.Action != ActionSendLatestFile || it.Target != "modify_and_send" {
			t.Fatalf("%q 应命中 modify_and_send，got %+v", text, it)
		}
	}
}

// TestParse_SendLatestFile_LetsReminderWin 提醒字样整体让位提醒位。
func TestParse_SendLatestFile_LetsReminderWin(t *testing.T) {
	for _, text := range []string{"提醒我把报告发给老板", "提醒我发产物给我", "定时把文件发给我"} {
		it := Parse(text)
		if it != nil && it.Action == ActionSendLatestFile {
			t.Fatalf("%q 不应命中产物推送（应归提醒位），got %+v", text, it)
		}
	}
}

// TestParse_SendLatestFile_V41412_Miss 放宽后仍不命中的样例（宁漏勿误）。
func TestParse_SendLatestFile_V41412_Miss(t *testing.T) {
	for _, text := range []string{"别什么都发给我", "直接发给我", "发文件给他", "把文件删了"} {
		if it := Parse(text); it != nil && it.Action == ActionSendLatestFile {
			t.Fatalf("%q 不应命中产物推送，got %+v", text, it)
		}
	}
}
