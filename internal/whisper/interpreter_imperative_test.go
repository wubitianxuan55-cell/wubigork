package whisper

import (
	"testing"
)

// 众神殿臂B刀②（docs/gaea-chat-probe-baseline-20261106.md §二）：短祈使/任务句
// 豁免 EvtCold——「讲个故事」「继续」「唱歌」是办事不是冷淡；真冷淡（嗯/哦/随便）
// 照旧判冷；问句路径不受影响。
func TestImperativeShortMessagesNotCold(t *testing.T) {
	cases := []struct {
		msg  string
		want EventType
	}{
		{"讲个故事", EvtCasualChat},
		{"写首诗", EvtCasualChat},
		{"继续", EvtCasualChat},
		{"唱歌", EvtCasualChat},
		{"逗我笑", EvtCasualChat},
		{"来个笑话", EvtCasualChat},
		{"讲一个短故事", EvtCasualChat},
		{"嗯", EvtCold},
		{"哦", EvtCold},
		{"随便", EvtCold},
		{"你是谁", EvtQuestion},
		{"你会什么", EvtQuestion},
		{"今天天气怎么样", EvtQuestion},
	}
	for _, c := range cases {
		if got := InterpretInput(c.msg, 50).Type; got != c.want {
			t.Errorf("InterpretInput(%q) = %s, want %s", c.msg, got, c.want)
		}
	}
}

func TestIsImperativeOrTask(t *testing.T) {
	for _, m := range []string{"讲个故事", "写首诗", "帮我查一下", "继续", "陪我说说话"} {
		if !IsImperativeOrTask(m) {
			t.Errorf("%q 应命中祈使/任务锚", m)
		}
	}
	for _, m := range []string{"嗯", "哦", "随便", "不聊了", "你真讨厌"} {
		if IsImperativeOrTask(m) {
			t.Errorf("%q 不应命中祈使/任务锚", m)
		}
	}
}
