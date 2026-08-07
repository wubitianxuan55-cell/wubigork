package app

import "testing"

func TestClassifyMainBrainIntent(t *testing.T) {
	cases := []struct {
		msg    string
		module string
		intent string
	}{
		{"帮我把这个标书写了", "office", "create"},
		{"写一份土壤修复方案", "office", "create"},
		{"生成第三章", "novel", "create_chapter"},
		{"写小说章节", "novel", "create_chapter"},
		{"和轻语聊聊天", "whisper", "chat"},
		{"画一张星空图", "imagegen", "generate"},
		{"生成一幅水墨画", "imagegen", "generate"},
		{"今天天气怎么样", "gaea", "chat"},
	}
	for _, tc := range cases {
		module, intent := classifyMainBrainIntent(tc.msg)
		if module != tc.module || intent != tc.intent {
			t.Errorf("%q → (%q,%q), want (%q,%q)", tc.msg, module, intent, tc.module, tc.intent)
		}
	}
}
