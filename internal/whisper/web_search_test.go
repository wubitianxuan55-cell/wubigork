package whisper

import "testing"

func TestCleanSearchQuery(t *testing.T) {
	cases := []struct{ in, want string }{
		{"最近几天天气怎样", "最近几天天气"},
		{"成都今天天气怎么样", "成都今天天气"},
		{"帮我搜索一下最近的新闻", "最近的新闻"},
		{"查一下今天上海天气", "今天上海天气"},
		{"什么是量子纠缠", "量子纠缠"},
		{"如何申请护照", "申请护照"},
		{"今天股价多少", "今天股价"},
		{"请帮我查一下杭州到上海的高铁", "杭州到上海的高铁"},
		{"", ""},
		{"仅仅一个词", "仅仅一个词"},
	}
	for _, c := range cases {
		got := cleanSearchQuery(c.in)
		if got != c.want {
			t.Errorf("cleanSearchQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
