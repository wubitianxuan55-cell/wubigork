package intent

import (
	"testing"
)

// TestParse_EditImageHits 对话式改图命中样例（v4.9）：编辑动词 × 图片指代
// 双门槛同时满足；Target = 去掉指代前缀的编辑指令全文（剥不动回原文）。
func TestParse_EditImageHits(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		target string
	}{
		{"换背景", "把这张图的背景换成海边", "背景换成海边"},
		{"去路人", "去掉图里的路人", "去掉图里的路人"},
		{"刚才那张", "刚才那张图调成黑白", "调成黑白"},
		{"词头改图带冒号", "改图：加上帽子", "加上帽子"},
		{"帮我变成动漫", "帮我把这张图变成动漫风", "变成动漫风"},
		{"图中人物", "把图中的人物变成卡通", "人物变成卡通"},
		{"上一张重绘", "上一张图重绘成水彩风", "重绘成水彩风"},
		{"P成风格", "帮我把这张图P成赛博朋克风", "P成赛博朋克风"},
		{"指代它", "把它变成像素风", "变成像素风"},
		{"修图词头带逗号", "修图，把天空调亮一点", "把天空调亮一点"},
		{"那个图换天空", "把那个图的天空换成晚霞", "天空换成晚霞"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			it := Parse(tc.text)
			if it == nil {
				t.Fatalf("应命中改图：%s", tc.text)
			}
			if it.Action != ActionEditImage {
				t.Fatalf("Action = %s，期望 %s", it.Action, ActionEditImage)
			}
			if it.Target != tc.target {
				t.Errorf("Target = %q，期望 %q", it.Target, tc.target)
			}
			if it.Text != tc.text {
				t.Errorf("Text = %q，期望原文 %q", it.Text, tc.text)
			}
		})
	}
}

// TestParse_EditImageMisses 改图误判防线：双门槛缺一即不命中；「画一张」
// 归生图（reImage 先命中序不变）。
func TestParse_EditImageMisses(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"无编辑动词", "这张图好看吗"},
		{"普通聊天", "今天天气怎么样"},
		{"背景非图义", "把背景音乐关掉"},
		{"裸删不在动词表", "把它删了吧"},
		{"屏幕不是图", "把屏幕调亮一点"},
		{"闲聊提到图", "我给你发过一张图，你还记得吗"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if it := Parse(tc.text); it != nil && it.Action == ActionEditImage {
				t.Fatalf("不应命中改图（%s → %s/%s）", tc.text, it.Action, it.Target)
			}
		})
	}

	// 「画一张图」必须落生图——改图不得误吞（命中序回归护栏）。
	if it := Parse("画一张图"); it == nil || it.Action != ActionGenerateImage {
		t.Fatalf("画一张图应归生图，实际 %+v", it)
	}
	if it := Parse("画一张海边风景"); it == nil || it.Action != ActionGenerateImage {
		t.Fatalf("画一张海边风景应归生图，实际 %+v", it)
	}
}

// TestExtractEditImageQuery 指代前缀剥离的边界：剥完只剩标点 = 空串
// （调用方按未命中处理，宁漏勿误）；无前缀回原文。
func TestExtractEditImageQuery(t *testing.T) {
	cases := []struct{ in, want string }{
		{"把这张图的背景换成海边", "背景换成海边"},
		{"请帮我把这张图调成黑白的", "调成黑白的"},
		{"去掉图里的路人", "去掉图里的路人"}, // 句中指代保留：整句即编辑指令
		{"改图。", ""},                 // 只剩标点 → 空
		{"背景换成海边", "换成海边"},
	}
	for _, tc := range cases {
		if got := extractEditImageQuery(tc.in); got != tc.want {
			t.Errorf("extractEditImageQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
