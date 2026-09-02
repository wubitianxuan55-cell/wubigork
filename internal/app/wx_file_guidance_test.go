package app

import (
	"strings"
	"testing"
)

// v4.41.1 文件消息直通聊天补丁：注入标记判定与引导追加。
// 真机实证背景——文件提取正文过意图路由会被正文碎片误触发导航
// （评审报告含「编程」+「看看/打开」被劫持回「打开编程」）。

func TestIsWxFileInjectMsg(t *testing.T) {
	withHeader := "[用户发来文件 报告.docx（424 KB）]\n正文…提到打开编程与看看代码"
	if !isWxFileInjectMsg(withHeader) {
		t.Fatal("文件注入消息未被识别")
	}
	plain := "把刚才的报告发我"
	if isWxFileInjectMsg(plain) {
		t.Fatal("普通消息被误判为文件消息")
	}
	noHeader := "用户发来文件这几个字出现在句子中间"
	if isWxFileInjectMsg(noHeader) {
		t.Fatal("无注入头标记不应命中")
	}
}

func TestApplyWxFileGuidance(t *testing.T) {
	raw := "[用户发来文件 a.docx（1 KB）]\n内容"
	got := applyWxFileGuidance(raw)
	if !strings.HasPrefix(got, raw) {
		t.Fatal("引导追加不得改动原文")
	}
	if !strings.Contains(got, "询问用户需要做什么") || !strings.Contains(got, "不要把文件内容当作") {
		t.Fatal("引导缺少「确认+询问+不执行」三要素")
	}
	// 生产链路（微信回调）每条消息只调用一次；函数非幂等（逐次追加）是已知
	// 且无害的实现口径——锁死该口径，防未来被无意改成静默去重。
	if twice := applyWxFileGuidance(got); strings.Count(twice, "询问用户需要做什么") != 2 {
		t.Fatal("重复调用行为与实现口径不符（应逐次追加）")
	}
	if plain := applyWxFileGuidance("普通聊天"); plain != "普通聊天" {
		t.Fatal("非文件消息不得追加引导")
	}
}
