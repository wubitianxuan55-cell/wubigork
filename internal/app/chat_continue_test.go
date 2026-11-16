package app

import (
	"net/http"
	"strings"
	"testing"
)

// A 批续写测试：prefill 续写拼接落库 + 准入校验。
func TestChatContinue_PrefillContinuation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sseReply(w, "这是续写段。")
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	topic, err := a.ChatTopicCreate("continue", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	// 线形两轮：q1 → a1(stopped) → q2 → a2；续写锚在 a1（历史=可见路径该轮之前）
	u1, _ := a.chatStore.AppendMessage(topic.ID, "user", "问题一", "")
	a1, _ := a.chatStore.AppendMessage(topic.ID, "assistant", "回答一部分", `{"stopped":true,"answered_by":{"engine":"herdsman"}}`)
	_, _ = a.chatStore.AppendMessage(topic.ID, "user", "问题二", "")
	_, _ = a.chatStore.AppendMessage(topic.ID, "assistant", "回答二", "")

	fixed := "chat_continue_" + strings.ReplaceAll(t.Name(), "/", "_")
	origRunID := newChatStreamRunID
	newChatStreamRunID = func() string { return fixed }
	t.Cleanup(func() { newChatStreamRunID = origRunID })

	events := captureStreamEvents(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatContinue(topic.ID, a1.ID); err != nil {
			t.Fatalf("ChatContinue: %v", err)
		}
	})
	done := terminalStreamEvent(t, events, "done")
	if done["reply"] != "这是续写段。" {
		t.Errorf("done.reply = %v, want 续写段", done["reply"])
	}
	if done["continued"] == nil {
		t.Errorf("done 缺 continued 锚: %v", done)
	}
	merged, err := a.chatStore.GetMessage(a1.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if !strings.Contains(merged.Content, "回答一部分") || !strings.Contains(merged.Content, "这是续写段。") {
		t.Errorf("拼接落库 = %q, want 原文+续写段", merged.Content)
	}
	// extra 不动（stopped 标记保留——准入凭据不因续写消失）
	if !extraFlagTrue(merged.Extra, "stopped") {
		t.Errorf("续写后 extra.stopped 丢失: %s", merged.Extra)
	}
	// 用户行未被动到
	u1Now, _ := a.chatStore.GetMessage(u1.ID)
	if u1Now.Content != "问题一" {
		t.Errorf("用户行被动到: %q", u1Now.Content)
	}
}

func TestChatContinue_AdmissionGuards(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sseReply(w, "不该被调用")
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	topic, err := a.ChatTopicCreate("guard", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	other, err := a.ChatTopicCreate("other", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate other: %v", err)
	}
	u, _ := a.chatStore.AppendMessage(topic.ID, "user", "q", "")
	noStop, _ := a.chatStore.AppendMessage(topic.ID, "assistant", "正常回答", "")
	_, _ = a.chatStore.AppendMessage(topic.ID, "assistant", "停掉的回答", `{"stopped":true}`)
	otherMsg, _ := a.chatStore.AppendMessage(other.ID, "assistant", "别的话题", `{"stopped":true}`)
	empty, _ := a.chatStore.AppendMessage(topic.ID, "assistant", "", `{"stopped":true}`)

	cases := []struct {
		name    string
		topicID string
		msgID   int64
		wantErr string
	}{
		{"未停止行拒绝", topic.ID, noStop.ID, "不是已停止状态"},
		{"跨话题拒绝", topic.ID, otherMsg.ID, "不在话题内"},
		{"用户行拒绝", topic.ID, u.ID, "回答行"},
		{"不存在拒绝", topic.ID, 999999, "消息不存在"},
		{"空正文拒绝", topic.ID, empty.ID, "无正文"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := a.ChatContinue(tc.topicID, tc.msgID); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want 含 %q", err, tc.wantErr)
			}
		})
	}
}
