package tasks

// 任务会话维度测试（v4.180 结构刀）：SessionID 写读往返、非会话入口诚实留空
// （空串 + JSON omitempty 缺省）、事件视图携带 session_id。

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSubmitSpaceSessionRoundtrip(t *testing.T) {
	gdb := openTestDB(t)
	m := New(gdb, nil, Options{})

	// 会话入口：SubmitSpaceSession 显式带会话标识，返回视图原样回带
	tk, err := m.SubmitSpaceSession(KindFileIndex, "会话内索引", map[string]any{"reason": "manual"}, "", "sa_abc123")
	if err != nil {
		t.Fatalf("submit session: %v", err)
	}
	if tk.Space != defaultTaskSpace {
		t.Errorf("SubmitSpaceSession 空 space = %q, want %q", tk.Space, defaultTaskSpace)
	}
	if tk.SessionID != "sa_abc123" {
		t.Errorf("返回视图 SessionID = %q, want sa_abc123", tk.SessionID)
	}

	// Get 往返：列值原样回带
	got, err := m.Get(tk.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SessionID != "sa_abc123" {
		t.Errorf("Get SessionID = %q, want sa_abc123", got.SessionID)
	}

	// List 原样回带（读取端统一列清单）
	list, err := m.ListInSpace(10, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].SessionID != "sa_abc123" {
		t.Fatalf("List SessionID = %+v, want sa_abc123", list)
	}
}

func TestSubmitSessionEmptyHonest(t *testing.T) {
	gdb := openTestDB(t)
	m := New(gdb, nil, Options{})

	// 既有入口 Submit/SubmitSpace：会话归属诚实留空（cron/系统周期任务不造数）
	legacy, err := m.Submit(KindPriceFetchAll, "定时抓取", map[string]any{"cron": true})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if legacy.SessionID != "" {
		t.Errorf("Submit SessionID = %q, want 空串", legacy.SessionID)
	}
	sp, err := m.SubmitSpace(KindPriceFetch, "手动抓取", nil, "work")
	if err != nil {
		t.Fatalf("submit space: %v", err)
	}
	if sp.SessionID != "" {
		t.Errorf("SubmitSpace SessionID = %q, want 空串", sp.SessionID)
	}
	// 空串落库（非 NULL）：Get 读回仍为空串
	got, err := m.Get(legacy.ID)
	if err != nil {
		t.Fatalf("get legacy: %v", err)
	}
	if got.SessionID != "" {
		t.Errorf("Get(legacy) SessionID = %q, want 空串", got.SessionID)
	}
}

func TestSessionIDJSONOmitEmpty(t *testing.T) {
	gdb := openTestDB(t)
	m := New(gdb, nil, Options{})

	// 非会话任务：JSON 缺省 session_id 键（omitempty）
	legacy, err := m.Submit(KindFileIndex, "无会话任务", nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	b, err := json.Marshal(*legacy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "session_id") {
		t.Errorf("空 SessionID 应被 omitempty 缺省，got %s", b)
	}

	// 会话任务：JSON 携带 session_id（随 GaeaTaskList 自动带出，零新绑定）
	tk, err := m.SubmitSpaceSession(KindFileIndex, "会话任务", nil, "work", "sa_json")
	if err != nil {
		t.Fatalf("submit session: %v", err)
	}
	b, err = json.Marshal(*tk)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if !strings.Contains(string(b), `"session_id":"sa_json"`) {
		t.Errorf("JSON 应携带 session_id，got %s", b)
	}
}

func TestSessionTaskEventCarriesSessionID(t *testing.T) {
	gdb := openTestDB(t)
	col := &eventCollector{}
	m := New(gdb, col.add, Options{})

	tk, err := m.SubmitSpaceSession(KindFileIndex, "会话任务", nil, "", "sa_evt")
	if err != nil {
		t.Fatalf("submit session: %v", err)
	}
	ev, ok := col.last()
	if !ok {
		t.Fatalf("入队事件未收到")
	}
	if ev.ID != tk.ID || ev.SessionID != "sa_evt" {
		t.Errorf("事件视图 = %+v, want SessionID sa_evt", ev)
	}
}
