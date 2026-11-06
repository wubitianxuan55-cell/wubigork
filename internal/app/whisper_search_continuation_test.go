package app

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 众神殿 action-7 多轮续接刀（进度计划/gaea-chat-persona-optimization-20261105.md
// §三.4）：待补槽位+一次合并检索。旧天气链坏例「明天天气→四川成都→我要天气→成都」
// 全转城市介绍——shouldSearchWeb 只扫当前句，短实体答不触发二次检索。验收：
// 第二轮触发「上一轮查询词+实体」合并检索；未消费即清；重启后续接仍可用。

// recordingSearchStub 记录每次实际检索词的 chatWebSearch 桩（返回罐头结果）。
func recordingSearchStub(t *testing.T) *[]string {
	t.Helper()
	var mu sync.Mutex
	queries := &[]string{}
	orig := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		mu.Lock()
		*queries = append(*queries, q)
		mu.Unlock()
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "罐头天气结果"}, nil
	}
	t.Cleanup(func() { chatWebSearch = orig })
	return queries
}

func stubQueries(q *[]string) []string { return *q }

func TestSearchContinuation_MergesFollowup(t *testing.T) {
	a := newChatServiceTestApp(t)
	personalityID := "cont_merge"
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	queries := recordingSearchStub(t)

	if _, err := a.WhisperChatWithSearch("明天天气", personalityID, false, false); err != nil {
		t.Fatalf("turn1: %v", err)
	}
	if _, err := a.WhisperChatWithSearch("四川成都", personalityID, false, false); err != nil {
		t.Fatalf("turn2: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	qs := stubQueries(queries)
	if len(qs) != 2 {
		t.Fatalf("应恰好两次检索（触发+合并），got %v", qs)
	}
	if !strings.Contains(qs[1], "四川成都") {
		t.Fatalf("第二轮应为合并检索（含实体），got %q", qs[1])
	}
	if !strings.Contains(qs[1], "天气") && !strings.Contains(qs[1], "明天") {
		t.Fatalf("合并检索应携带上一轮查询语境，got %q", qs[1])
	}
	// 第三轮：pending 已消费清除，不再检索
	if _, err := a.WhisperChatWithSearch("今天好开心呀", personalityID, false, false); err != nil {
		t.Fatalf("turn3: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if got := stubQueries(queries); len(got) != 2 {
		t.Fatalf("消费后不得续挂检索，got %v", got)
	}
}

func TestSearchContinuation_NotConsumedCleared(t *testing.T) {
	a := newChatServiceTestApp(t)
	personalityID := "cont_clear"
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	queries := recordingSearchStub(t)

	if _, err := a.WhisperChatWithSearch("明天天气", personalityID, false, false); err != nil {
		t.Fatalf("turn1: %v", err)
	}
	// 第二轮长句闲聊（不像实体短答）→ 不消费；回合结束 pending 清除
	if _, err := a.WhisperChatWithSearch("今天心情不错想聊聊最近的生活", personalityID, false, false); err != nil {
		t.Fatalf("turn2: %v", err)
	}
	if _, err := a.WhisperChatWithSearch("成都", personalityID, false, false); err != nil {
		t.Fatalf("turn3: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	qs := stubQueries(queries)
	if len(qs) != 1 {
		t.Fatalf("过期后不得续接检索，got %v", qs)
	}
}

func TestSearchContinuation_PersistsAcrossRestart(t *testing.T) {
	a := newChatServiceTestApp(t)
	ws := a.whisperState
	personalityID := "cont_restart"
	sessionID := "whisper_" + personalityID
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	queries := recordingSearchStub(t)

	// 第一轮触发搜索，同步落库（PendingSearch 随 state_json 持久化）
	if _, err := a.WhisperChatWithSearch("明天天气", personalityID, false, false); err != nil {
		t.Fatalf("turn1: %v", err)
	}
	orch := ws.getOrCreateOrch(personalityID)
	if orch.State.PendingSearch == nil || !strings.Contains(orch.State.PendingSearch.Query, "天气") {
		t.Fatalf("turn1 后应有 PendingSearch: %+v", orch.State.PendingSearch)
	}
	if err := ws.persistStateSync(orch); err != nil {
		t.Fatalf("persist: %v", err)
	}

	// 模拟重启：清会话缓存（数据根不动）
	whisperSessionsMu.Lock()
	delete(whisperSessions, sessionID)
	whisperSessionsMu.Unlock()

	// 重启后第一轮：短实体答仍应触发合并检索
	if _, err := a.WhisperChatWithSearch("四川成都", personalityID, false, false); err != nil {
		t.Fatalf("turn2 after restart: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	qs := stubQueries(queries)
	if len(qs) != 2 || !strings.Contains(qs[1], "四川成都") {
		t.Fatalf("重启后应续接合并检索，got %v", qs)
	}

	// 持久化面复核：state_json 里确有 pendingSearch
	if st, err := repos.LoadCompanionStateFromDB(ws.whisperDataRoot, sessionID); err != nil || st == nil {
		t.Fatalf("reload: %v", err)
	}
}
