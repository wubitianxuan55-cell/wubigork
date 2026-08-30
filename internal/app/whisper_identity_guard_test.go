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

// v4.9.1 触发词收窄后的协同口径：口语疑问词（是谁/什么/怎么…）已从触发表
// 移除——身份问题根本不再触发搜索（守卫保留为纵深防御，防未来触发表回胖）。
func TestIdentityGuardBeatsSearchTrigger(t *testing.T) {
	if shouldSearchWeb("你是谁") {
		t.Fatal("「你是谁」是口语疑问，收窄后不应触发搜索")
	}
	for _, msg := range []string{"你怎么看", "最近怎么样", "你在哪里", "为什么啊", "介绍一下这个项目"} {
		if shouldSearchWeb(msg) {
			t.Errorf("对话高频词 %q 不应触发搜索（宁漏勿误）", msg)
		}
	}
	for _, msg := range []string{"帮我查一下北京天气", "搜一下沪深300指数", "今天有什么新闻", "查查黄金价格"} {
		if !shouldSearchWeb(msg) {
			t.Errorf("显式动词/硬时效 %q 应触发搜索", msg)
		}
	}
	if !isIdentityQuestion("你是谁") {
		t.Fatal("身份守卫应命中「你是谁」（纵深防御保留）")
	}
}
