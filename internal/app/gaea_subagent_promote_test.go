package app

// GaeaPromoteSubagent 测试（v4.68「保存为新会话」）：投影往返、标题来源、
// ref 校验、原 transcript 不动、重复提升独立、降级策略、play 空间、archive 源。
// 搭建模式沿用 gaea_subagents_test.go 的 writeJSON/writeMessages 助手。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// writePromoteFixture 构造带完整子代理 transcript 的会话（目录形态：
// <root>/.gaea/sessions[/<where>]/，与 sessionDirForPath 校验一致）。
// transcript 含 system 提示、配对工具调用与最终回答——真实落盘结构的忠实缩影。
func writePromoteFixture(t *testing.T, where string) (sessionPath, sessionDir, subDir, ref string) {
	t.Helper()
	root := t.TempDir()
	sessionDir = filepath.Join(root, ".gaea", "sessions", where)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sessionPath = filepath.Join(sessionDir, "s1.jsonl")
	if err := os.WriteFile(sessionPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir = filepath.Join(sessionDir, "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ref = "sa_20260904_090000_0000000001_ab12cd34"
	writeJSON(t, filepath.Join(subDir, ref+".meta.json"), map[string]interface{}{
		"ref": ref, "status": "completed", "title": "成本调研子代理",
		"model": "deepseek-v4-flash",
	})
	writeMessages(t, filepath.Join(subDir, ref+".jsonl"), []provider.Message{
		{Role: provider.RoleSystem, Content: "You are a sub-agent invoked by a parent assistant."},
		{Role: provider.RoleUser, Content: "找出项目里所有调用 format_convert 的地方并总结模式"},
		{Role: provider.RoleAssistant, Content: "先检索引用点。", ReasoningContent: "需要先定位文件",
			ToolCalls: []provider.ToolCall{{ID: "call_1", Name: "grep", Arguments: `{"pattern":"format_convert"}`}}},
		{Role: provider.RoleTool, Name: "grep", ToolCallID: "call_1", Content: "5 处匹配"},
		{Role: provider.RoleAssistant, Content: "共发现 5 处调用，集中在 format_convert.go 与工具注册表。"},
	})
	return sessionPath, sessionDir, subDir, ref
}

// TestGaeaPromoteSubagent_RoundTrip 核心断言：提升后的新会话按真实恢复链
// （Restore = 日志全量重放投影）读回，消息序列与 transcript 忠实等价
// （system 不随迁）；legacy 镜像与会话列表发现同样成立。
func TestGaeaPromoteSubagent_RoundTrip(t *testing.T) {
	sessionPath, sessionDir, _, ref := writePromoteFixture(t, "")
	a := &App{core: &core{}}
	newPath, err := a.GaeaPromoteSubagent(sessionPath, ref)
	if err != nil {
		t.Fatalf("GaeaPromoteSubagent: %v", err)
	}
	if filepath.Dir(newPath) != sessionDir {
		t.Fatalf("新会话应落在同空间会话目录：%s", newPath)
	}
	if !strings.HasSuffix(filepath.Base(newPath), "-deepseek-v4-flash.jsonl") {
		t.Fatalf("文件名应携带 meta.Model 标签：%s", filepath.Base(newPath))
	}

	// 真实恢复链：checkpoint（不存在）+ 事件日志全量重放投影。
	restored, last, err := session.Restore(session.CheckpointPathFor(newPath), session.LogPathFor(newPath))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if last <= 0 {
		t.Fatalf("恢复游标应 > 0：%d", last)
	}
	want := []struct {
		role    provider.Role
		content string
	}{
		{provider.RoleUser, "找出项目里所有调用 format_convert 的地方并总结模式"},
		{provider.RoleAssistant, "先检索引用点。"},
		{provider.RoleTool, "5 处匹配"},
		{provider.RoleAssistant, "共发现 5 处调用，集中在 format_convert.go 与工具注册表。"},
	}
	if len(restored) != len(want) {
		t.Fatalf("恢复消息数 = %d，want %d：%+v", len(restored), len(want), restored)
	}
	for i, w := range want {
		if restored[i].Role != w.role || restored[i].Content != w.content {
			t.Fatalf("消息[%d] = %s/%q，want %s/%q", i, restored[i].Role, restored[i].Content, w.role, w.content)
		}
	}
	for _, m := range restored {
		if m.Role == provider.RoleSystem {
			t.Fatal("子代理 system 提示不应随迁")
		}
	}
	// 工具调用对完整保真：assistant 携带 call_1，tool 结果同 id 配对。
	if len(restored[1].ToolCalls) != 1 || restored[1].ToolCalls[0].ID != "call_1" ||
		restored[1].ToolCalls[0].Name != "grep" {
		t.Fatalf("assistant 工具调用丢失：%+v", restored[1].ToolCalls)
	}
	if restored[2].ToolCallID != "call_1" || restored[2].Name != "grep" {
		t.Fatalf("tool 结果配对丢失：%+v", restored[2])
	}
	if restored[1].ReasoningContent != "需要先定位文件" {
		t.Fatalf("reasoning 丢失：%q", restored[1].ReasoningContent)
	}

	// 投影往返恒等：ToLogEntries 产物投影 == 清洗后序列（写前校验的镜像断言）。
	entries, err := session.ReadLogRepaired(session.LogPathFor(newPath))
	if err != nil {
		t.Fatalf("ReadLogRepaired: %v", err)
	}
	if !promoteMessagesEqual(session.ProjectMessages(entries), restored) {
		t.Fatal("日志投影与恢复结果不等价")
	}
	kinds := make(map[string]int)
	for _, e := range entries {
		kinds[e.Kind]++
	}
	for _, k := range []string{"turn_started", "user_message", "assistant_message", "tool_result", "turn_done"} {
		if kinds[k] == 0 {
			t.Fatalf("日志缺少 kind %s：%v", k, kinds)
		}
	}

	// legacy 镜像与日志等价（列表发现与 legacy 恢复的数据源）。
	mirror, err := session.Load(newPath)
	if err != nil {
		t.Fatalf("Load 镜像: %v", err)
	}
	if !promoteMessagesEqual(mirror.Messages, restored) {
		t.Fatal("legacy 镜像与事件日志投影不等价")
	}

	// 会话列表发现：新会话应出现在 ListSessions（preview/turns 来自镜像）。
	infos, err := gaeaAgent.ListSessions(sessionDir)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	found := false
	for _, in := range infos {
		if in.Path == newPath {
			found = true
			if in.Turns < 1 || !strings.Contains(in.Preview, "format_convert") {
				t.Fatalf("列表条目异常：turns=%d preview=%q", in.Turns, in.Preview)
			}
		}
	}
	if !found {
		t.Fatalf("新会话未出现在会话列表：%+v", infos)
	}
}

// TestGaeaPromoteSubagent_Title：标题取 meta.Title；无 title 的 meta 回退
// transcript 首条 user 消息（截断 120 现有规则）；超长标题按规则截断。
func TestGaeaPromoteSubagent_Title(t *testing.T) {
	sessionPath, sessionDir, subDir, ref := writePromoteFixture(t, "")
	a := &App{core: &core{}}
	newPath, err := a.GaeaPromoteSubagent(sessionPath, ref)
	if err != nil {
		t.Fatalf("GaeaPromoteSubagent: %v", err)
	}
	if got := loadSessionTitles(sessionDir)[filepath.Base(newPath)]; got != "成本调研子代理" {
		t.Fatalf("标题注册表 = %q，want meta.Title", got)
	}

	// meta 无 title：回退首条 user 消息。
	ref2 := "sa_20260904_091000_0000000002_cd34ef56"
	writeJSON(t, filepath.Join(subDir, ref2+".meta.json"), map[string]interface{}{
		"ref": ref2, "status": "completed",
	})
	writeMessages(t, filepath.Join(subDir, ref2+".jsonl"), []provider.Message{
		{Role: provider.RoleUser, Content: strings.Repeat("汉", 200)},
		{Role: provider.RoleAssistant, Content: "回答。"},
	})
	newPath2, err := a.GaeaPromoteSubagent(sessionPath, ref2)
	if err != nil {
		t.Fatalf("GaeaPromoteSubagent(ref2): %v", err)
	}
	title2 := loadSessionTitles(sessionDir)[filepath.Base(newPath2)]
	if len([]rune(title2)) != 120 || !strings.HasSuffix(title2, "…") {
		t.Fatalf("回退标题应为首条 user 消息截断 120：len=%d tail=%q", len([]rune(title2)), title2[len(title2)-3:])
	}
}

// TestGaeaPromoteSubagent_RefValidation：ref 无效 / sessionPath 非法 /
// transcript 缺失 / 运行中 → 明确 error，且不产生任何新会话文件。
func TestGaeaPromoteSubagent_RefValidation(t *testing.T) {
	sessionPath, sessionDir, subDir, ref := writePromoteFixture(t, "")
	a := &App{core: &core{}}

	cases := []struct {
		name        string
		sessionPath string
		ref         string
		wantErr     string
	}{
		{"空会话路径", "", ref, "非法会话路径"},
		{"会话目录外路径", filepath.Join(t.TempDir(), "outside.jsonl"), ref, "非法会话路径"},
		{"空 ref", sessionPath, "", "sa_ 前缀"},
		{"非 sa_ 前缀（mt_ 本地模型工具）", sessionPath, "mt_20260904_090000_0000000001_ab12cd34", "sa_ 前缀"},
		{"路径穿越 ref", sessionPath, "../evil", "sa_ 前缀"},
		{"transcript 缺失", sessionPath, "sa_20260904_092000_0000000003_ee55ff66", "transcript"},
	}
	for _, c := range cases {
		got, err := a.GaeaPromoteSubagent(c.sessionPath, c.ref)
		if err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Fatalf("%s：err=%v，want 含 %q", c.name, err, c.wantErr)
		}
		if got != "" {
			t.Fatalf("%s：失败时不应返回路径 %q", c.name, got)
		}
	}

	// 运行中拒绝提升（transcript 是移动快照）。
	refRun := "sa_20260904_093000_0000000004_aa77bb88"
	writeJSON(t, filepath.Join(subDir, refRun+".meta.json"), map[string]interface{}{
		"ref": refRun, "status": "running",
	})
	writeMessages(t, filepath.Join(subDir, refRun+".jsonl"), []provider.Message{
		{Role: provider.RoleUser, Content: "还在跑"},
	})
	if _, err := a.GaeaPromoteSubagent(sessionPath, refRun); err == nil || !strings.Contains(err.Error(), "运行") {
		t.Fatalf("running 应拒绝：err=%v", err)
	}

	// 以上全部失败路径不得产生任何新会话产物。
	infos, err := gaeaAgent.ListSessions(sessionDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 0 {
		t.Fatalf("校验失败路径不应落盘会话：%+v", infos)
	}
}

// TestGaeaPromoteSubagent_OriginalUntouched：提升是纯复制——源 transcript、
// meta 与原会话文件逐字节不动。
func TestGaeaPromoteSubagent_OriginalUntouched(t *testing.T) {
	sessionPath, _, subDir, ref := writePromoteFixture(t, "")
	a := &App{core: &core{}}
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	before := map[string]string{
		"transcript":   read(filepath.Join(subDir, ref+".jsonl")),
		"meta":         read(filepath.Join(subDir, ref+".meta.json")),
		"session_file": read(sessionPath),
	}
	if _, err := a.GaeaPromoteSubagent(sessionPath, ref); err != nil {
		t.Fatalf("GaeaPromoteSubagent: %v", err)
	}
	for name, p := range map[string]string{
		"transcript":   filepath.Join(subDir, ref+".jsonl"),
		"meta":         filepath.Join(subDir, ref+".meta.json"),
		"session_file": sessionPath,
	} {
		if after := read(p); after != before[name] {
			t.Fatalf("源文件被改动：%s", name)
		}
	}
}

// TestGaeaPromoteSubagent_RepeatIndependent：重复提升同一 ref 每次产生独立
// 新副本（快照语义 + 追问后可再提升最新态），两副本互不别名、内容等价。
func TestGaeaPromoteSubagent_RepeatIndependent(t *testing.T) {
	sessionPath, sessionDir, _, ref := writePromoteFixture(t, "")
	a := &App{core: &core{}}
	p1, err := a.GaeaPromoteSubagent(sessionPath, ref)
	if err != nil {
		t.Fatalf("第一次提升: %v", err)
	}
	p2, err := a.GaeaPromoteSubagent(sessionPath, ref)
	if err != nil {
		t.Fatalf("第二次提升: %v", err)
	}
	if p1 == p2 {
		t.Fatalf("重复提升应产生新副本，实际复用 %s", p1)
	}
	r1, _, err := session.Restore(session.CheckpointPathFor(p1), session.LogPathFor(p1))
	if err != nil {
		t.Fatalf("Restore p1: %v", err)
	}
	r2, _, err := session.Restore(session.CheckpointPathFor(p2), session.LogPathFor(p2))
	if err != nil {
		t.Fatalf("Restore p2: %v", err)
	}
	if !promoteMessagesEqual(r1, r2) {
		t.Fatal("两个独立副本内容应等价")
	}
	if base := filepath.Base(p2); loadSessionTitles(sessionDir)[base] != "成本调研子代理" {
		t.Fatalf("第二个副本标题未注册：%q", loadSessionTitles(sessionDir)[base])
	}
}

// TestGaeaPromoteSubagent_DegradePolicy：无法忠实投影的内容诚实降级——
// 孤立/重复 tool 结果丢弃、未响应工具调用剥离（保留正文）、空 assistant 丢弃；
// 产物仍通过恢复校验且可续聊（无悬空配对）。
func TestGaeaPromoteSubagent_DegradePolicy(t *testing.T) {
	sessionPath, _, subDir, _ := writePromoteFixture(t, "")
	ref := "sa_20260904_094000_0000000005_bb99cc00"
	writeJSON(t, filepath.Join(subDir, ref+".meta.json"), map[string]interface{}{
		"ref": ref, "status": "failed",
	})
	writeMessages(t, filepath.Join(subDir, ref+".jsonl"), []provider.Message{
		{Role: provider.RoleUser, Content: "中断的任务"},
		{Role: provider.RoleTool, Name: "grep", ToolCallID: "call_orphan", Content: "无主结果"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "call_dead", Name: "bash", Arguments: "{}"}}},
		{Role: provider.RoleAssistant, Content: "被中断前的正文。",
			ToolCalls: []provider.ToolCall{{ID: "call_unanswered", Name: "write", Arguments: "{}"}}},
	})
	a := &App{core: &core{}}
	newPath, err := a.GaeaPromoteSubagent(sessionPath, ref)
	if err != nil {
		t.Fatalf("降级提升不应失败: %v", err)
	}
	restored, _, err := session.Restore(session.CheckpointPathFor(newPath), session.LogPathFor(newPath))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if len(restored) != 2 {
		t.Fatalf("降级后应剩 user + 带正文的 assistant，实际 %d 条：%+v", len(restored), restored)
	}
	if restored[0].Content != "中断的任务" || restored[1].Content != "被中断前的正文。" {
		t.Fatalf("内容不符：%+v", restored)
	}
	if len(restored[1].ToolCalls) != 0 {
		t.Fatalf("未响应调用应被剥离：%+v", restored[1].ToolCalls)
	}
	// 降级产物自洽：清洗 → 日志 → 投影 的往返在实现内已校验通过（否则报错）。
	entries, err := session.ReadLogRepaired(session.LogPathFor(newPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Kind == "tool_result" {
			t.Fatal("降级产物不应包含 tool_result 条目")
		}
	}
}

// TestGaeaPromoteSubagent_PlaySpace：play 分区会话的子代理提升落 play 目录，
// 日志行带 play 空间自描述，恢复空间校验通过（fail-closed 不误拒）。
func TestGaeaPromoteSubagent_PlaySpace(t *testing.T) {
	sessionPath, _, _, ref := writePromoteFixture(t, "play")
	a := &App{core: &core{}}
	newPath, err := a.GaeaPromoteSubagent(sessionPath, ref)
	if err != nil {
		t.Fatalf("GaeaPromoteSubagent: %v", err)
	}
	if filepath.Base(filepath.Dir(newPath)) != "play" {
		t.Fatalf("新会话应落在 play 分区：%s", newPath)
	}
	entries, err := session.ReadLogRepaired(session.LogPathFor(newPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 || entries[0].Space != "play" {
		t.Fatalf("日志行应带 play 空间自描述：%+v", entries[:1])
	}
	if _, _, err := session.Restore(session.CheckpointPathFor(newPath), session.LogPathFor(newPath)); err != nil {
		t.Fatalf("play 会话恢复被空间校验拒绝: %v", err)
	}
}

// TestGaeaPromoteSubagent_FromArchive：归档会话的子代理也可提升，新会话落到
// 所属空间目录（而非 archive/），源 transcript 取 archive 内兄弟 subagents/。
func TestGaeaPromoteSubagent_FromArchive(t *testing.T) {
	sessionPath, sessionDir, subDir, ref := writePromoteFixture(t, "")
	archiveDir := filepath.Join(sessionDir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(sessionPath, filepath.Join(archiveDir, "s1.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(subDir, filepath.Join(archiveDir, "subagents")); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{}}
	newPath, err := a.GaeaPromoteSubagent(filepath.Join(archiveDir, "s1.jsonl"), ref)
	if err != nil {
		t.Fatalf("GaeaPromoteSubagent: %v", err)
	}
	if filepath.Dir(newPath) != sessionDir {
		t.Fatalf("归档源的提升应落所属空间目录 %s，实际 %s", sessionDir, filepath.Dir(newPath))
	}
	if _, _, err := session.Restore(session.CheckpointPathFor(newPath), session.LogPathFor(newPath)); err != nil {
		t.Fatalf("Restore: %v", err)
	}
}
