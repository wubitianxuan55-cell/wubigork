package contextview

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func entry(seq int64, kind string, payload any) session.LogEntry {
	b, _ := json.Marshal(payload)
	return session.LogEntry{Seq: seq, Ts: 1700000000 + seq, Kind: kind, Payload: b}
}

func headerPayload(system string, tools ...string) any {
	ts := make([]any, 0, len(tools))
	for _, t := range tools {
		ts = append(ts, map[string]any{"name": t, "schema": `{"name":"` + t + `","parameters":{}}`})
	}
	return map[string]any{"system": system, "tools": ts, "window": 1_000_000}
}

func TestFoldEmpty(t *testing.T) {
	tl := FoldTimeline(nil, 1_000_000, 0)
	if !tl.Ok {
		t.Fatal("ok should be true")
	}
	if tl.Window != 1_000_000 {
		t.Fatalf("window = %d, want 1000000", tl.Window)
	}
	if len(tl.Requests) != 0 || len(tl.Nodes) != 0 || len(tl.Events) != 0 {
		t.Fatalf("empty fold should have no requests/nodes/events")
	}
}

// 回归：空会话的四条切片必须序列化成 [] 而不是 null——Go 的 nil 切片序列化
// 为 JSON null，前端 .length / for-of 按数组消费会整页崩（ErrorBoundary 接管）。
func TestFoldEmptySlicesMarshalAsArrays(t *testing.T) {
	cases := map[string]ContextTimeline{
		"fold":      FoldTimeline(nil, 1_000_000, 0),
		"binding":   EmptyTimeline(),
		"retention": FoldTimeline(nil, 0, 200),
	}
	for name, tl := range cases {
		b, err := json.Marshal(tl)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		s := string(b)
		for _, field := range []string{`"requests":[]`, `"events":[]`, `"nodes":[]`, `"archive":[]`, `"files":[]`} {
			if !strings.Contains(s, field) {
				t.Fatalf("%s: %s should marshal as [], got %s", name, field, s)
			}
		}
	}
}

func TestFoldFileActivity(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys", "read_file", "write_file")),
		entry(3, "user_message", map[string]any{"content": "改一下报价单"}),
		entry(4, "tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"报价单.md"}`, "partial": false}),
		entry(5, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "body"}),
		entry(6, "tool_dispatch", map[string]any{"id": "t2", "name": "write_file", "args": `{"path":"报价单.md","content":"..."}`, "partial": false}),
		entry(7, "tool_result", map[string]any{"id": "t2", "name": "write_file", "output": "ok"}),
		entry(8, "tool_dispatch", map[string]any{"id": "t3", "name": "ls", "args": `{"path":"."}`, "partial": false}),
		entry(9, "tool_result", map[string]any{"id": "t3", "name": "ls", "output": "a.md\nb.md"}),
		// 无路径键的工具不进文件活动（诚实不造数）
		entry(10, "tool_dispatch", map[string]any{"id": "t4", "name": "bash", "args": `{"command":"dir"}`, "partial": false}),
		entry(11, "tool_result", map[string]any{"id": "t4", "name": "bash", "output": "..."}),
		// 同一工具+路径+步骤重复调用合并为一条
		entry(12, "tool_dispatch", map[string]any{"id": "t5", "name": "read_file", "args": `{"path":"报价单.md"}`, "partial": false}),
		entry(13, "tool_result", map[string]any{"id": "t5", "name": "read_file", "output": "body2"}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Files) != 3 {
		t.Fatalf("files = %d, want 3（read/write/dir 各一条，bash 不进、重复 read 合并）: %+v", len(tl.Files), tl.Files)
	}
	want := []FileActivity{
		{Tool: "read_file", Action: "read", Path: "报价单.md", Turn: 1, Step: 1},
		{Tool: "write_file", Action: "write", Path: "报价单.md", Turn: 1, Step: 1},
		{Tool: "ls", Action: "dir", Path: ".", Turn: 1, Step: 1},
	}
	for i, w := range want {
		f := tl.Files[i]
		if f.Tool != w.Tool || f.Action != w.Action || f.Path != w.Path {
			t.Errorf("file[%d] = %+v, want %+v", i, f, w)
		}
	}
	// 合并后的 read 条目时间戳刷新到最后一次
	if tl.Files[0].Seq != 12 {
		t.Errorf("merged read seq = %d, want 12（刷新到最后一次调用）", tl.Files[0].Seq)
	}
}

func TestFoldFileActivityFromOutput(t *testing.T) {
	// screen_capture 参数无路径，结果输出携带保存路径（JSON）→ 结果阶段补记。
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys", "screen_capture")),
		entry(3, "user_message", map[string]any{"content": "截个屏"}),
		entry(4, "tool_dispatch", map[string]any{"id": "t1", "name": "screen_capture", "args": `{"region":{"x":0}}`, "partial": false}),
		entry(5, "tool_result", map[string]any{"id": "t1", "name": "screen_capture", "output": `{"path":"shots/20260831.png"}`}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Files) != 1 {
		t.Fatalf("files = %d, want 1（从输出提取）: %+v", len(tl.Files), tl.Files)
	}
	if tl.Files[0].Action != "write" || tl.Files[0].Path != "shots/20260831.png" {
		t.Fatalf("file = %+v, want write shots/20260831.png", tl.Files[0])
	}
}

func TestFoldSingleRequest(t *testing.T) {
	sys := strings.Repeat("系统提示词内容", 20)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys, "read_file", "bash")),
		entry(3, "user_message", map[string]any{"content": "Referenced context:\n\n# file.md\n这是一段很长的项目资料内容用于注入上下文\n\n请帮我总结这份资料并给出建议"}),
		entry(4, "assistant_message", map[string]any{"text": "好的，我来总结。"}),
		entry(5, "tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"a.go"}`, "partial": false}),
		entry(6, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "package a 这是一段足够长的工具输出内容", "truncated": false}),
		entry(7, "usage", map[string]any{
			"promptTokens": 400, "completionTokens": 30,
			"cacheHitTokens": 380, "cacheMissTokens": 20,
			"sessionCacheHitTokens": 380, "sessionCacheMissTokens": 20,
			"turn": 1,
		}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if tl.Window != 1_000_000 {
		t.Fatalf("window should come from header, got %d", tl.Window)
	}
	if len(tl.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(tl.Requests))
	}
	r := tl.Requests[0]
	if r.Turn != 1 || r.Step != 1 {
		t.Fatalf("turn/step = %d/%d, want 1/1", r.Turn, r.Step)
	}
	if r.PromptTokens != 400 {
		t.Fatalf("promptTokens = %d, want 400", r.PromptTokens)
	}
	if r.Estimated {
		t.Fatal("有 usage 事件的请求不应标记 estimated")
	}
	if r.BriefUser != "请帮我总结这份资料并给出建议" {
		t.Fatalf("briefUser = %q, want 请帮我总结这份资料并给出建议", r.BriefUser)
	}
	if r.BriefResp != "read_file {\"path\":\"a.go\"}" {
		t.Fatalf("briefResp = %q, want read_file with args", r.BriefResp)
	}
	// 2.5d 跳转锚点：user 锚=用户消息事件（含注入拆分时的 user 半边）；resp
	// 锚=工具结果节点（结果到达后覆盖派发/assistant 锚）。
	if r.BriefUserSeq != 3 {
		t.Fatalf("briefUserSeq = %d, want 3", r.BriefUserSeq)
	}
	if r.BriefRespSeq != 6 {
		t.Fatalf("briefRespSeq = %d, want 6（工具结果节点）", r.BriefRespSeq)
	}
	if tl.Current.System == 0 || tl.Current.Tools == 0 {
		t.Fatalf("system/tools should be estimated from header: %+v", tl.Current)
	}
	if tl.Current.User == 0 || tl.Current.Inject == 0 {
		t.Fatalf("user/inject should be split: %+v", tl.Current)
	}
	if tl.Current.Assistant == 0 || tl.Current.Tool == 0 {
		t.Fatalf("assistant/tool should be present: %+v", tl.Current)
	}
	if tl.Stats.ToolCalls != 1 || tl.Stats.Steps != 1 || tl.Stats.Turns != 1 || tl.Stats.Injects != 1 {
		t.Fatalf("stats wrong: %+v", tl.Stats)
	}
	if tl.Stats.CacheHitPercent != 95 {
		t.Fatalf("cacheHitPercent = %v, want 95", tl.Stats.CacheHitPercent)
	}
	// 六分类节点齐全
	cats := map[string]bool{}
	for _, n := range tl.Nodes {
		cats[n.Cat] = true
	}
	for _, want := range []string{catSystem, catTools, catUser, catInject, catAssistant, catTool} {
		if !cats[want] {
			t.Fatalf("missing node category %q in %+v", want, cats)
		}
	}
}

func TestFoldBriefJumpAnchorsFallback(t *testing.T) {
	// 2.5d 锚点退化口径：
	//  1. 无工具交换 → resp 锚退化到 assistant 消息节点；
	//  2. header 先于任何消息 → 锚点为 0（前端不渲染跳转）；
	//  3. estimated 关闭路径同样带锚点。
	sys := strings.Repeat("s", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys, "read_file")),
		entry(3, "user_message", map[string]any{"content": "这是一段足够长的用户输入内容"}),
		entry(4, "assistant_message", map[string]any{"text": "直接回答，不调用工具的回答内容"}),
		entry(5, "usage", map[string]any{"promptTokens": 100, "turn": 1}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(tl.Requests))
	}
	r := tl.Requests[0]
	if r.BriefUserSeq != 3 {
		t.Fatalf("briefUserSeq = %d, want 3", r.BriefUserSeq)
	}
	if r.BriefRespSeq != 4 {
		t.Fatalf("briefRespSeq = %d, want 4（无工具时退化 assistant 节点）", r.BriefRespSeq)
	}

	empty := FoldTimeline([]session.LogEntry{
		entry(1, "request_header", headerPayload(sys)),
		entry(2, "turn_done", map[string]any{}),
	}, 1_000_000, 0)
	// 全程无任何用户/助手/工具节点 → 锚点 0（前端不渲染跳转，诚实缺失）。
	if len(empty.Requests) != 1 {
		t.Fatalf("empty requests = %d, want 1（turn_done 估算关闭）", len(empty.Requests))
	}
	if r0 := empty.Requests[0]; r0.BriefUserSeq != 0 || r0.BriefRespSeq != 0 {
		t.Fatalf("无消息时锚点应为 0，got user=%d resp=%d", r0.BriefUserSeq, r0.BriefRespSeq)
	}

	est := FoldTimeline([]session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys)),
		entry(3, "user_message", map[string]any{"content": "旧日志无 usage 的提问内容"}),
		entry(4, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "body"}),
		entry(5, "turn_done", map[string]any{}),
	}, 1_000_000, 0)
	re := est.Requests[0]
	if !re.Estimated {
		t.Fatal("estimated 应为 true")
	}
	if re.BriefUserSeq != 3 || re.BriefRespSeq != 4 {
		t.Fatalf("estimated 关闭路径锚点错误：user=%d resp=%d", re.BriefUserSeq, re.BriefRespSeq)
	}
}

func TestFoldEstimatedCloseOnTurnDone(t *testing.T) {
	// 旧日志/无 usage 提供方：request_header 已开、usage 未到，turn_done 用
	// 估算分类关闭请求（趋势柱仍出现，诚实标注 estimated，不伪造用量）。
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(strings.Repeat("s", 200), "read_file")),
		entry(3, "user_message", map[string]any{"content": "这是一段足够长的用户输入内容"}),
		entry(4, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "body 一段足够长的工具输出"}),
		entry(5, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Requests) != 1 {
		t.Fatalf("requests = %d, want 1（回合末估算关闭）", len(tl.Requests))
	}
	r := tl.Requests[0]
	if !r.Estimated {
		t.Fatal("estimated 应为 true（无 usage 事件）")
	}
	if r.Category.System == 0 || r.Category.Tools == 0 || r.Category.User == 0 {
		t.Fatalf("estimated category should carry system/tools/user: %+v", r.Category)
	}
}

func TestFoldSystemToolsNodesOnlyOnChange(t *testing.T) {
	// request_header 的系统/工具集合只在变化时新增 nodes（每步重复不刷屏）。
	sys := strings.Repeat("s", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys, "read_file", "bash")),
		entry(3, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10, "turn": 1}),
		entry(4, "request_header", headerPayload(sys, "read_file", "bash")), // 同 system + 同工具集 → 不新增
		entry(5, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10, "turn": 1}),
		entry(6, "request_header", headerPayload(sys+"\n追加", "read_file", "bash")), // system 变化 → 新增
		entry(7, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10, "turn": 1}),
		entry(8, "request_header", headerPayload(sys+"\n追加", "read_file", "bash", "grep")), // 工具集变化 → 新增
		entry(9, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10, "turn": 1}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	sysNodes := 0
	toolsNodes := 0
	for _, n := range tl.Nodes {
		if n.Cat == catSystem {
			sysNodes++
		}
		if n.Cat == catTools {
			toolsNodes++
		}
	}
	if sysNodes != 2 {
		t.Fatalf("system nodes = %d, want 2（初版 + 变化版）", sysNodes)
	}
	if toolsNodes != 2 {
		t.Fatalf("tools nodes = %d, want 2（初版 + 工具集变化版）", toolsNodes)
	}
	// 工具集节点文本应含工具名
	found := false
	for _, n := range tl.Nodes {
		if n.Cat == catTools && strings.Contains(n.Text, "grep") {
			found = true
		}
	}
	if !found {
		t.Fatalf("tools node text should list tool names: %+v", tl.Nodes)
	}
}

func TestFoldScalesToActual(t *testing.T) {
	// 估算合计很小，usage 实际 promptTokens 很大 → 等比放大且钳制在 4 倍内。
	sys := strings.Repeat("s", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys)),
		entry(3, "user_message", map[string]any{"content": "hi 一段足够长的用户输入内容"}),
		entry(4, "usage", map[string]any{"promptTokens": 10_000, "completionTokens": 10, "turn": 1}),
	}
	tl := FoldTimeline(entries, 0, 0)
	r := tl.Requests[0]
	if r.Category.Total() <= 0 {
		t.Fatalf("scaled total should be positive: %+v", r.Category)
	}
	if r.Category.Total() > 10_000*2 {
		t.Fatalf("scaled total %d should be clamped near actual", r.Category.Total())
	}
}

func TestFoldLegacyWithoutHeader(t *testing.T) {
	// 旧日志没有 request_header：usage 事件仍生成请求记录（估算分类）。
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "这是一段足够长的用户输入内容"}),
		entry(3, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10, "turn": 1}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(tl.Requests))
	}
	if tl.Requests[0].Category.User == 0 {
		t.Fatalf("legacy request should carry estimated user tokens: %+v", tl.Requests[0].Category)
	}
}

func TestFoldCompaction(t *testing.T) {
	sys := strings.Repeat("s", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys)),
		entry(3, "user_message", map[string]any{"content": "task one 一段足够长的用户输入"}),
		entry(4, "tool_result", map[string]any{"id": "t1", "name": "grep", "output": "many lines of output that will be compacted away later"}),
		entry(5, "usage", map[string]any{"promptTokens": 500, "completionTokens": 20, "turn": 1}),
		entry(6, "compaction_done", map[string]any{"trigger": "ratio", "summary": "summary text", "messages": 5, "archive": []string{"x"}}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if tl.Stats.Compacts != 1 {
		t.Fatalf("compacts = %d, want 1", tl.Stats.Compacts)
	}
	if len(tl.Events) != 1 || tl.Events[0].Kind != "compact" || tl.Events[0].Delta >= 0 {
		t.Fatalf("compact event wrong: %+v", tl.Events)
	}
	// 压缩后 user/tool 节点应进归档、current 回落。
	if len(tl.Archive) == 0 {
		t.Fatal("archive should contain gone nodes")
	}
	if tl.Current.User != 0 || tl.Current.Tool != 0 {
		t.Fatalf("current should drop after compaction: %+v", tl.Current)
	}
	if tl.Current.System == 0 {
		t.Fatalf("system prompt survives compaction: %+v", tl.Current)
	}
	for _, n := range tl.Nodes {
		if n.Gone != nil {
			t.Fatalf("live nodes should not be gone: %+v", n)
		}
	}
}

func TestRetention(t *testing.T) {
	var entries []session.LogEntry
	seq := int64(1)
	for i := 0; i < 5; i++ {
		entries = append(entries, entry(seq, "turn_started", map[string]any{}))
		seq++
		entries = append(entries, entry(seq, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1, "turn": i + 1}))
		seq++
	}
	tl := FoldTimeline(entries, 0, 2)
	if len(tl.Requests) != 2 {
		t.Fatalf("retention: requests = %d, want 2", len(tl.Requests))
	}
	if tl.Requests[0].Turn != 4 {
		t.Fatalf("retention should keep the newest: first turn = %d, want 4", tl.Requests[0].Turn)
	}
}

func TestPruneEvent(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "tool_result", map[string]any{"id": "t1", "name": "bash", "output": "x", "truncated": true}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if tl.Stats.Prunes != 1 {
		t.Fatalf("prunes = %d, want 1", tl.Stats.Prunes)
	}
	if len(tl.Events) != 1 || tl.Events[0].Kind != "prune" {
		t.Fatalf("prune event missing: %+v", tl.Events)
	}
}

// ─── 对比上一步（RequestDelta） ───────────────────────────────────

// TestFoldRequestDeltaFirst：首个请求 First=true，差值=全量构成（基线=空）。
func TestFoldRequestDeltaFirst(t *testing.T) {
	sys := strings.Repeat("s", 120)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys, "read_file")),
		entry(3, "user_message", map[string]any{"content": "请帮我看看这个文件"}),
		entry(4, "usage", map[string]any{"promptTokens": 300, "completionTokens": 10, "turn": 1}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if len(tl.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(tl.Requests))
	}
	d := tl.Requests[0].Delta
	if d == nil {
		t.Fatal("首请求应有 delta")
	}
	if !d.First {
		t.Fatal("首请求应标记 First")
	}
	if d.Approx {
		t.Fatal("未跨压缩不应标记 Approx")
	}
	if d.Items <= 0 || d.Tokens <= 0 {
		t.Fatalf("首请求差值应为全量构成: %+v", d)
	}
	// ByCat 按 |tokens| 降序
	for i := 1; i < len(d.ByCat); i++ {
		ai, aj := abs64(d.ByCat[i-1].Tokens), abs64(d.ByCat[i].Tokens)
		if ai < aj || (ai == aj && d.ByCat[i-1].Cat > d.ByCat[i].Cat) {
			t.Fatalf("ByCat 排序错误: %+v", d.ByCat)
		}
	}
}

// TestFoldRequestDeltaSecond：第二个请求相对第一个的净变化——新增的 user/
// assistant/tool 内容记正增量，system/tools 未变不出现在 ByCat。
// 顺序按 live 日志（user 落盘先于本回合 header）。
func TestFoldRequestDeltaSecond(t *testing.T) {
	sys := strings.Repeat("s", 120)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "第一问"}),
		entry(3, "request_header", headerPayload(sys, "read_file")),
		entry(4, "usage", map[string]any{"promptTokens": 200, "completionTokens": 10, "turn": 1}),
		entry(5, "assistant_message", map[string]any{"text": "回答内容足够长以产生估算 tokens"}),
		entry(6, "tool_dispatch", map[string]any{"id": "t1", "name": "ls", "args": `{}`, "partial": false}),
		entry(7, "tool_result", map[string]any{"id": "t1", "name": "ls", "output": "file-a.go file-b.go"}),
		entry(8, "user_message", map[string]any{"content": "第二问再详细一点"}),
		entry(9, "request_header", headerPayload(sys, "read_file")),
		entry(10, "usage", map[string]any{"promptTokens": 400, "completionTokens": 10, "turn": 2}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if len(tl.Requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(tl.Requests))
	}
	d := tl.Requests[1].Delta
	if d == nil || d.First {
		t.Fatalf("第二请求 delta 错误: %+v", d)
	}
	if d.Approx {
		t.Fatal("未跨压缩不应标记 Approx")
	}
	byCat := map[string]CatDelta{}
	for _, c := range d.ByCat {
		byCat[c.Cat] = c
	}
	// system/tools 两请求间未变，不应进 ByCat
	if _, ok := byCat[catSystem]; ok {
		t.Fatalf("system 未变不应进 ByCat: %+v", d.ByCat)
	}
	if _, ok := byCat[catTools]; ok {
		t.Fatalf("tools 未变不应进 ByCat: %+v", d.ByCat)
	}
	// 上一响应的 assistant 消息 + 工具结果 + 新 user 输入都是正增量
	if c := byCat[catAssistant]; c.Tokens <= 0 || c.Items != 1 {
		t.Fatalf("assistant 增量错误: %+v", c)
	}
	if c := byCat[catTool]; c.Tokens <= 0 || c.Items != 1 {
		t.Fatalf("tool 增量错误: %+v", c)
	}
	if c := byCat[catUser]; c.Tokens <= 0 || c.Items != 1 {
		t.Fatalf("user 增量错误: %+v", c)
	}
	if d.Tokens <= 0 || d.Items != 3 {
		t.Fatalf("合计增量错误: %+v", d)
	}
}

// TestFoldRequestDeltaApprox：两请求之间发生压缩 → 其后首个请求 delta 标
// Approx，且被压分类的 tokens 记负增量（基线含被压内容；新输入比旧短时净
// tokens 为负、项数净 0）。顺序按 live 日志（user 落盘先于本回合 header）。
func TestFoldRequestDeltaApprox(t *testing.T) {
	sys := strings.Repeat("s", 120)
	longAsk := strings.Repeat("这是一段很长很长的用户提问，会被压缩掉。", 6)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": longAsk}),
		entry(3, "request_header", headerPayload(sys, "read_file")),
		entry(4, "usage", map[string]any{"promptTokens": 500, "completionTokens": 10, "turn": 1}),
		entry(5, "compaction_done", map[string]any{"trigger": "ratio", "summary": "摘要", "messages": 3}),
		entry(6, "user_message", map[string]any{"content": "新问题"}),
		entry(7, "request_header", headerPayload(sys, "read_file")),
		entry(8, "usage", map[string]any{"promptTokens": 260, "completionTokens": 10, "turn": 2}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if tl.Stats.Compacts != 1 {
		t.Fatalf("compacts = %d, want 1", tl.Stats.Compacts)
	}
	d := tl.Requests[1].Delta
	if d == nil {
		t.Fatal("应有 delta")
	}
	if !d.Approx {
		t.Fatal("跨压缩请求应标记 Approx")
	}
	byCat := map[string]CatDelta{}
	for _, c := range d.ByCat {
		byCat[c.Cat] = c
	}
	// 旧 user 被压走：净变化 = 新 user − 旧 user，tokens 为负、项数净 0
	c := byCat[catUser]
	if c.Tokens >= 0 || c.Items != 0 {
		t.Fatalf("跨压缩 user 净变化错误（应为净负 tokens、项数 0）: %+v", c)
	}
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// ─── 文件活动行级增量（v4.81，dsh ±added/−removed 同款） ──────────

// TestFoldFileDelta：四写类工具从参数确定性提取 ±行；grep 结果行数为命中
// 近似；非写类/取不到参数诚实留零。
func TestFoldFileDelta(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "tool_dispatch", map[string]any{"id": "w", "name": "write_file", "args": `{"path":"a.go","content":"l1\nl2\nl3"}`, "partial": false}),
		entry(2, "tool_result", map[string]any{"id": "w", "name": "write_file", "output": "ok"}),
		entry(3, "tool_dispatch", map[string]any{"id": "e", "name": "edit_file", "args": `{"path":"b.go","old_string":"x\ny","new_string":"z"}`, "partial": false}),
		entry(4, "tool_result", map[string]any{"id": "e", "name": "edit_file", "output": "ok"}),
		entry(5, "tool_dispatch", map[string]any{"id": "m", "name": "multi_edit", "args": `{"path":"c.go","edits":[{"old_string":"a","new_string":"p\nq"},{"old_string":"r\ns\nt","new_string":"u"}]}`, "partial": false}),
		entry(6, "tool_result", map[string]any{"id": "m", "name": "multi_edit", "output": "ok"}),
		entry(7, "tool_dispatch", map[string]any{"id": "l", "name": "edit_lines", "args": `{"path":"d.go","start_line":5,"end_line":6,"new_content":"p\nq\nr"}`, "partial": false}),
		entry(8, "tool_result", map[string]any{"id": "l", "name": "edit_lines", "output": "ok"}),
		entry(9, "tool_dispatch", map[string]any{"id": "g", "name": "grep", "args": `{"path":"internal","pattern":"x"}`, "partial": false}),
		entry(10, "tool_result", map[string]any{"id": "g", "name": "grep", "output": "hit1\nhit2\nhit3\nhit4"}),
		entry(11, "tool_dispatch", map[string]any{"id": "r", "name": "read_file", "args": `{"path":"e.go"}`, "partial": false}),
		entry(12, "tool_result", map[string]any{"id": "r", "name": "read_file", "output": "正文"}),
	}
	tl := FoldTimeline(entries, 0, 0)
	byPath := map[string]FileActivity{}
	for _, f := range tl.Files {
		byPath[f.Path] = f
	}
	if f := byPath["a.go"]; f.Added != 3 || f.Removed != 0 {
		t.Fatalf("write_file 增量错误: %+v", f)
	}
	if f := byPath["b.go"]; f.Added != 1 || f.Removed != 2 {
		t.Fatalf("edit_file 增量错误: %+v", f)
	}
	if f := byPath["c.go"]; f.Added != 3 || f.Removed != 4 {
		t.Fatalf("multi_edit 增量错误: %+v", f)
	}
	if f := byPath["d.go"]; f.Added != 3 || f.Removed != 2 {
		t.Fatalf("edit_lines 增量错误: %+v", f)
	}
	if f := byPath["internal"]; f.Hits != 4 {
		t.Fatalf("grep 命中行错误: %+v", f)
	}
	if f := byPath["e.go"]; f.Added != 0 || f.Removed != 0 || f.Hits != 0 {
		t.Fatalf("read_file 不应有增量/命中: %+v", f)
	}
}

func TestFoldCostRate(t *testing.T) {
	// 2.5e 成本 hover：fold 透出最近一次 usage 上报的单价（每 1M tokens）。
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys", "read_file")),
		entry(3, "usage", map[string]any{
			"promptTokens": 100, "completionTokens": 10,
			"cacheHitTokens": 80, "cacheMissTokens": 20,
			"input": 2.0, "output": 8.0, "cacheHitPrice": 0.5,
			"currency": "CNY", "turn": 1,
		}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if tl.Rate == nil {
		t.Fatal("有定价 usage 时应透出 Rate")
	}
	if tl.Rate.InputPer1M != 2.0 || tl.Rate.OutputPer1M != 8.0 || tl.Rate.CacheHitPer1M != 0.5 {
		t.Fatalf("rate = %+v", tl.Rate)
	}
	if tl.Rate.Currency != "CNY" {
		t.Fatalf("currency = %q, want CNY", tl.Rate.Currency)
	}
	// 无定价 usage → Rate 为 nil（诚实：费用未估算）
	tl2 := FoldTimeline([]session.LogEntry{
		entry(1, "usage", map[string]any{"promptTokens": 10, "turn": 1}),
	}, 1_000_000, 0)
	if tl2.Rate != nil {
		t.Fatalf("无定价时 Rate 应为 nil: %+v", tl2.Rate)
	}
}
