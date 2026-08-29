package intent

import (
	"testing"
)

// TestParse_Hits 命中用例：正指令必须命中且字段正确。
func TestParse_Hits(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		action Action
		target string
	}{
		{"打开板块", "打开绘梦", ActionNavigate, "imagegen"},
		{"切换到", "切换到造价数据库", ActionNavigate, "cost"},
		{"强动词看看", "看一下记忆中枢", ActionNavigate, "memoryhub"},
		{"回首页", "回到首页", ActionNavigate, "home"},
		{"去首页", "去首页", ActionNavigate, "home"},
		{"微信", "打开微信助手", ActionNavigate, "weixin"},
		{"画一张", "画一张赛博朋克城市夜景", ActionGenerateImage, "赛博朋克城市夜景"},
		{"帮我画", "帮我画一只橘猫", ActionGenerateImage, "一只橘猫"},
		{"画单词", "画猫", ActionGenerateImage, "猫"},
		{"生成图片", "生成一张图片：雪山日出", ActionGenerateImage, "雪山日出"},
		{"状态模型", "现在用什么模型", ActionStatus, "model"},
		{"状态引擎", "引擎状态怎么样", ActionStatus, "model"},
		{"提醒", "提醒我 30分钟后 喝水", ActionReminder, ""},
		{"口语尾巴", "打开绘梦。", ActionNavigate, "imagegen"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			it := Parse(tc.text)
			if it == nil {
				t.Fatalf("应命中 %s", tc.text)
			}
			if it.Action != tc.action {
				t.Errorf("Action = %s，期望 %s", it.Action, tc.action)
			}
			if it.Target != tc.target {
				t.Errorf("Target = %q，期望 %q", it.Target, tc.target)
			}
		})
	}
}

// TestParse_Misses 漏判/误判防线：闲聊绝不触发指令（宁可漏判不可误判）。
func TestParse_Misses(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"纯闲聊", "今天天气怎么样"},
		{"画口语赞美", "画得不错"},
		{"画过时态", "画了半天图"},
		{"画过时态二", "这张画过好几次了"},
		{"打开无板块", "打开文件管理器看看"},
		{"无动词闲聊提板块", "造价数据库真是好用"},
		{"空文本", "  "},
		{"标点闲聊", "画得好！"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if it := Parse(tc.text); it != nil {
				t.Fatalf("不应命中（%s → %s/%s）", tc.text, it.Action, it.Target)
			}
		})
	}
}
