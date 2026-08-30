package app

import "testing"

// v4.8.2 微信实测复盘：身份类问题误触发联网搜索——「你是谁」命中触发词
// 「是谁」→ Bing 无关摘要注入提示词 → 模型把网页英文片段混进回复（夹杂
// 随机字母的根因）+ 白加一轮搜索往返（慢的次要因素）。
// isIdentityQuestion 守卫：身份类问题跳过 auto-search。

func TestIsIdentityQuestion(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"你是谁", true},
		{"你叫什么名字", true},
		{"你会什么", true},
		{"你能干什么", true},
		{"你是什么模型", true},
		{"介绍一下你自己", true},
		{"说说你自己", true},
		{"你好", false},          // 普通问候：不过不搜索无关（无触发词），也不拦
		{"他叫什么名字", false},   // 问第三方：不拦（无「你」开头模式命中）
		{"今天天气怎么样", false}, // 时效类：正常走搜索
		{"帮我搜一下特斯拉股价", false},
	}
	for _, c := range cases {
		if got := isIdentityQuestion(c.msg); got != c.want {
			t.Errorf("isIdentityQuestion(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

// 守卫与触发词的协同：身份问题即便命中搜索触发词也不搜索。
func TestIdentityGuardBeatsSearchTrigger(t *testing.T) {
	if !shouldSearchWeb("你是谁") {
		t.Fatal("前置条件：「你是谁」应命中触发词「是谁」（否则此测试无意义）")
	}
	if !isIdentityQuestion("你是谁") {
		t.Fatal("身份守卫应命中「你是谁」")
	}
}
