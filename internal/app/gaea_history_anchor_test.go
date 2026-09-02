package app

// v4.34.0 线A：GaeaHistory 子代理锚点合并测试。
// 覆盖：mergeSubagentAnchors 纯函数（中插槽位换算含 tool dispatch 条目 /
// 越界丢弃 / 无锚点原样 / 同位保序 / 偏移对齐 / 负偏移宁漏勿误）与
// GaeaHistory 真实接线（磁盘事件日志 → 锚点 → 插入位置与 SubagentRef）。

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// anchorMsgs 是槽位换算测试的公共 fixture：
// user → assistant(2 个工具调用) → tool → assistant。
// GaeaHistory 展开槽位：user=1、assistant=1+2、tool=1、assistant=1，共 6 条。
func anchorMsgs() []provider.Message {
	return []provider.Message{
		{Role: provider.RoleUser, Content: "跑个子代理"},
		{Role: provider.RoleAssistant, Content: "开工", ToolCalls: []provider.ToolCall{
			{ID: "t1", Name: "task", Arguments: "{}"},
			{ID: "t2", Name: "read_file", Arguments: "{}"},
		}},
		{Role: provider.RoleTool, Content: "done", ToolCallID: "t1", Name: "task"},
		{Role: provider.RoleAssistant, Content: "收工"},
	}
}

// roles 提取 Role 序列便于断言插入位置。
func roles(msgs []HistoryMessage) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Role)
	}
	return out
}

// 中插位置正确：锚点在 assistant+ToolCalls 消息之后时，槽位换算必须把两条
// tool dispatch 条目算进去（第 2 条 provider 消息产出 3 条槽位 → 插在下标 4）。
func TestMergeSubagentAnchorsMidInsertAfterToolCalls(t *testing.T) {
	msgs := anchorMsgs()
	anchors := []session.SubagentAnchor{
		{Text: "子答复", Ref: "sa_1", ParentToolID: "t1", AfterMsgIndex: 2},
	}
	got := mergeSubagentAnchors(msgs, anchors, 0)
	wantRoles := []string{"user", "assistant", "tool", "tool", "assistant", "tool_result", "assistant"}
	if !reflect.DeepEqual(roles(got), wantRoles) {
		t.Fatalf("roles = %v, want %v", roles(got), wantRoles)
	}
	sub := got[4]
	if sub.Content != "子答复" || sub.SubagentRef != "sa_1" || sub.ToolID != "" || sub.ToolName != "" {
		t.Fatalf("锚点条目 = %+v, want 纯 assistant 气泡携带 subagentRef", sub)
	}
}

// K 越界丢弃：换算位置超出 provider 消息数 → 宁漏勿误，原样返回。
func TestMergeSubagentAnchorsDropsOutOfRange(t *testing.T) {
	msgs := anchorMsgs()
	anchors := []session.SubagentAnchor{
		{Text: "迟到", Ref: "sa_x", AfterMsgIndex: 99},
	}
	got := mergeSubagentAnchors(msgs, anchors, 0)
	if !reflect.DeepEqual(got, buildHistoryMessages(msgs)) {
		t.Fatalf("越界锚点未丢弃: %+v", got)
	}
}

// 无锚点原样返回（与既有构建逻辑逐条一致）。
func TestMergeSubagentAnchorsNoAnchorsIdentity(t *testing.T) {
	msgs := anchorMsgs()
	got := mergeSubagentAnchors(msgs, nil, 0)
	if !reflect.DeepEqual(got, buildHistoryMessages(msgs)) {
		t.Fatalf("无锚点输出漂移:\n got %+v\nwant %+v", got, buildHistoryMessages(msgs))
	}
}

// 多锚点同 K 按日志序依次排列。
func TestMergeSubagentAnchorsSameKPreservesOrder(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "u"},
		{Role: provider.RoleAssistant, Content: "a"},
	}
	anchors := []session.SubagentAnchor{
		{Text: "第一个", Ref: "sa_1", AfterMsgIndex: 1},
		{Text: "第二个", Ref: "sa_2", AfterMsgIndex: 1},
	}
	got := mergeSubagentAnchors(msgs, anchors, 0)
	wantRoles := []string{"user", "assistant", "assistant", "assistant"}
	if !reflect.DeepEqual(roles(got), wantRoles) {
		t.Fatalf("roles = %v, want %v", roles(got), wantRoles)
	}
	if got[1].Content != "第一个" || got[1].SubagentRef != "sa_1" {
		t.Fatalf("got[1] = %+v, want 第一个/sa_1", got[1])
	}
	if got[2].Content != "第二个" || got[2].SubagentRef != "sa_2" {
		t.Fatalf("got[2] = %+v, want 第二个/sa_2（同 K 保序）", got[2])
	}
}

// 偏移对齐：事件日志模式的 provider 历史比日志投影多出检查点携带的 system
// 提示消息（日志从不投影 system prompt）→ offset=1 时锚点 K=1 应插在 user
// 消息之后，而非 system 之后。
func TestMergeSubagentAnchorsOffsetAlignsSystemPrompt(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "system prompt"},
		{Role: provider.RoleUser, Content: "u"},
		{Role: provider.RoleAssistant, Content: "a"},
	}
	anchors := []session.SubagentAnchor{{Text: "子答复", Ref: "sa_1", AfterMsgIndex: 1}}
	got := mergeSubagentAnchors(msgs, anchors, 1)
	wantRoles := []string{"system", "user", "assistant", "assistant"}
	if !reflect.DeepEqual(roles(got), wantRoles) {
		t.Fatalf("roles = %v, want %v", roles(got), wantRoles)
	}
	if got[2].SubagentRef != "sa_1" {
		t.Fatalf("got[2] = %+v, want 锚点条目", got[2])
	}
}

// 负偏移（compaction 吞掉早期消息）：换算位置为负的锚点丢弃且不阻塞后续
// 锚点（宁漏勿误）。
func TestMergeSubagentAnchorsNegativeOffsetDropsStale(t *testing.T) {
	msgs := anchorMsgs() // 4 条 provider 消息
	anchors := []session.SubagentAnchor{
		{Text: "被压缩吞掉的", Ref: "sa_old", AfterMsgIndex: 1}, // 1-5 = -4 → 丢弃
		{Text: "存活", Ref: "sa_new", AfterMsgIndex: 7},     // 7-5 = 2 → 插在第 2 条消息槽位之后
	}
	got := mergeSubagentAnchors(msgs, anchors, -5)
	// K=7 → 换算位置 2 = 第 2 条 provider 消息（assistant + 2 条 dispatch，
	// 共 3 槽）之后 → 下标 4。
	wantRoles := []string{"user", "assistant", "tool", "tool", "assistant", "tool_result", "assistant"}
	if !reflect.DeepEqual(roles(got), wantRoles) {
		t.Fatalf("roles = %v, want %v", roles(got), wantRoles)
	}
	if got[4].Content != "存活" || got[4].SubagentRef != "sa_new" {
		t.Fatalf("got[4] = %+v, want 存活/sa_new", got[4])
	}
}

// mustRaw 序列化 payload（日志写入端同形状）。
func mustRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestGaeaHistoryMergesSubagentAnchors 真实接线：轻量控制器 + 磁盘事件日志。
// 日志投影 3 条消息（user/assistant+toolcall/tool），provider 历史多一条
// system 提示（offset=1）；锚点 K=1 必须落在 user 消息之后，携带 subagentRef，
// 其余条目与既有 GaeaHistory 构建逐条一致。GaeaResumeSession 复用 GaeaHistory，
// 自动生效。
func TestGaeaHistoryMergesSubagentAnchors(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws, "")

	// 内存会话：system 提示 + user + assistant(工具调用) + tool 结果。
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "跑个子代理"})
	s.Add(provider.Message{
		Role:    provider.RoleAssistant,
		Content: "开工",
		ToolCalls: []provider.ToolCall{
			{ID: "t1", Name: "task", Arguments: "{}"},
		},
	})
	s.Add(provider.Message{Role: provider.RoleTool, Content: "done", ToolCallID: "t1", Name: "task"})
	path := filepath.Join(sessionDir, "sa-session.jsonl")

	// 磁盘事件日志：user_message 之后落 subagent_message（与运行期事件序一致）。
	logPath := session.LogPathFor(path)
	w, err := session.OpenLog(logPath, "", "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	entries := []session.LogEntry{
		{Kind: "turn_started", Payload: mustRaw(t, map[string]string{})},
		{Kind: session.KindUserMessage, Payload: mustRaw(t, map[string]string{"content": "跑个子代理"})},
		{Kind: session.KindSubagentMessage, Payload: mustRaw(t, map[string]string{
			"text": "子代理答复：任务完成", "ref": "sa_t1", "parentId": "t1",
		})},
		{Kind: session.KindAssistantMessage, Payload: mustRaw(t, map[string]any{
			"id": "t1", "text": "开工",
			"tool_calls": []map[string]string{{"id": "t1", "name": "task", "args": "{}"}},
		})},
		{Kind: session.KindToolResult, Payload: mustRaw(t, map[string]string{
			"id": "t1", "name": "task", "output": "done",
		})},
	}
	for i, e := range entries {
		if _, err := w.AppendRaw(e.Kind, e.Payload); err != nil {
			t.Fatalf("append entry %d: %v", i, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// 轻量控制器（不走真实引擎 boot）：GaeaHistory 只用到 History/SessionPath。
	exec := agent.New(nil, nil, s, agent.Options{}, event.Discard)
	ctrl := control.New(control.Options{Runner: noopStateRunner{}, Executor: exec, Sink: event.Discard})
	ctrl.Resume(s, path)
	ga.ctrl = ctrl
	defer ctrl.Close()

	a := &App{}
	hist := a.GaeaHistory()
	wantRoles := []string{"system", "user", "assistant", "assistant", "tool", "tool_result"}
	if !reflect.DeepEqual(roles(hist), wantRoles) {
		t.Fatalf("roles = %v, want %v", roles(hist), wantRoles)
	}
	sub := hist[2]
	if sub.Content != "子代理答复：任务完成" || sub.SubagentRef != "sa_t1" {
		t.Fatalf("锚点气泡 = %+v, want 子代理答复/sa_t1（插在 user 之后）", sub)
	}
	if hist[3].Content != "开工" || hist[4].ToolID != "t1" || hist[5].ToolOutput != "done" {
		t.Fatalf("既有构建条目漂移: %+v", hist)
	}
}
