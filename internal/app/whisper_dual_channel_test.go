package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// 众神殿刀3（进度计划/gaea-chat-persona-optimization-20261105.md §1.1①）：搜索
// 入口 raw/llm 双通道。唯一标记经 chatWebSearch 桩注入后：出现在本轮模型输入
// （llmUserMsg 尾部）与 PromptHistory 实发串（本批明确不改的既定口径），
// 不得出现在 raw 通道面——WM.UserText、chat.db 落库原文。双通道≠隔离完成。
func TestWhisperChatWithSearch_DualChannelMarker(t *testing.T) {
	const marker = "ZZSEARCHMARKER7391ZZ"
	var sawMarkerInModel atomic.Bool
	var lastUserMsg atomic.Pointer[string]
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), marker) {
			sawMarkerInModel.Store(true)
		}
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			for i := len(req.Messages) - 1; i >= 0; i-- {
				if req.Messages[i].Role == "user" {
					s := req.Messages[i].Content
					lastUserMsg.Store(&s)
					break
				}
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"今天晴\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	personalityID := "persona_dual_channel"
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	stubChatWebSearch(t, func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: "今天天气如何", Text: "标记段 " + marker + " 结束"}, nil
	})

	if _, err := a.whisperChatCore(context.Background(), "今天天气如何", personalityID, "", false, true /* B7 后附录机制归 lowLatency(voice) 面 */, true, false, nil, nil); err != nil {
		t.Fatalf("whisperChatCore: %v", err)
	}
	if !sawMarkerInModel.Load() {
		t.Fatal("唯一标记应出现在本轮模型输入")
	}
	if !strings.Contains(*lastUserMsg.Load(), marker) {
		t.Fatalf("标记应在最后一条 user 消息（llmUserMsg）: %q", *lastUserMsg.Load())
	}

	// raw 通道面①：工作记忆收原文
	orch := a.getOrCreateOrch(personalityID)
	recent := orch.WM.GetRecent(orch.SessionID)
	if len(recent) == 0 {
		t.Fatal("WM 应有本轮交换")
	}
	if got := recent[len(recent)-1].UserText; got != "今天天气如何" {
		t.Fatalf("WM.UserText 应为用户原文，got %q", got)
	}
	// raw 通道面②：PromptHistory 存原句（净行刀 B3，2026-11-06）——旧「实发
	// 串」口径随净行刀退役：动态块与搜索增广不再以 user 角色跨轮重放（格式
	// 模仿老师+token 浪费），同进程/重启回灌/语音出口三口径一致。增广块只
	// 进本轮 llmUserMsg（上面 lastUserMsg 断言已锁）。
	if len(orch.PromptHistory) == 0 {
		t.Fatal("PromptHistory 应有本轮行")
	}
	if got := orch.PromptHistory[len(orch.PromptHistory)-1].UserText; got != "今天天气如何" {
		t.Fatalf("PromptHistory 应存用户原句（净行契约），got %q", got)
	}
	if strings.Contains(orch.PromptHistory[len(orch.PromptHistory)-1].UserText, marker) {
		t.Fatal("PromptHistory 不得含搜索增广标记（净行契约）")
	}
	time.Sleep(100 * time.Millisecond)
}

// TestChatSendPersona_DualChannelRawDB chat.db 落库面收原文：阻塞 persona 路径
// 触发搜索后，chat.db 用户行仍是原文、不含增广块。
func TestChatSendPersona_DualChannelRawDB(t *testing.T) {
	const marker = "ZZDBMARKER8241ZZ"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"今天晴\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	personalityID := "persona_dual_db"
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	topic, err := a.ChatTopicCreate("双通道落库", personalityID)
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	stubChatWebSearch(t, func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: "今天天气如何", Text: "标记段 " + marker + " 结束"}, nil
	})

	if _, err := a.chatSendPersona(topic.ID, "今天天气如何", personalityID, false, false, false); err != nil {
		t.Fatalf("chatSendPersona: %v", err)
	}
	msgs, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("应落一对消息，got %d", len(msgs))
	}
	if msgs[0].Content != "今天天气如何" {
		t.Fatalf("chat.db 用户行应为原文，got %q", msgs[0].Content)
	}
	for _, m := range msgs {
		if strings.Contains(m.Content, marker) {
			t.Fatalf("搜索增广不得进入 chat.db: %+v", m)
		}
	}
	time.Sleep(100 * time.Millisecond)
}
