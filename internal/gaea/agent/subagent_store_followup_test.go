package agent

// v4.66 追问失败写回 meta：RunFollowUp 后台失败的原因摘要经 SubagentStore
// 落 meta sidecar（followUpError），随 transcript/轮询链路带出，前端据此把
// 乐观气泡转失败态（不再永久「等待中」）。用例覆盖：
//   - RecordFollowUpError 记录/清除 + CreatedAt/Title/Kind/Status 守恒；
//   - 无 meta 不造幽灵记录、mt_/非法 ref 拒绝、超长摘要截断；
//   - RunFollowUp 失败路：终态 failed + 摘要落 meta（且终态不被 running
//     快照回写覆盖——stop 先于终态写的 v4.64.0 回归钉子）；
//   - RunFollowUp 成功路：开跑清旧摘要，完成后 meta completed 无残留。

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// errProvider 的 Stream 恒错：模拟追问后台执行失败（模型掉线/上下文超限…）。
type errProvider struct{ name string }

func (p *errProvider) Name() string { return p.name }
func (p *errProvider) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	return nil, context.DeadlineExceeded
}
func (p *errProvider) Chat(_ context.Context, _ provider.Request) (*provider.Completion, error) {
	return nil, context.DeadlineExceeded
}

// seedCompletedRun 造一个已完结的 sa_ 运行（带 transcript + meta），返回 ref。
func seedCompletedRun(t *testing.T, store *SubagentStore) string {
	t.Helper()
	run, err := store.PrepareFreshWithTitle("sys", spaces.SpaceWork, "调研子代理续跑")
	if err != nil {
		t.Fatalf("PrepareFreshWithTitle: %v", err)
	}
	run.Session.Add(provider.Message{Role: provider.RoleUser, Content: "调研子代理续跑的背景"})
	run.Session.Add(provider.Message{Role: provider.RoleAssistant, Content: "结论：可以续跑。"})
	if err := store.MarkRunning(run); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}
	if err := store.SaveCompleted(run); err != nil {
		t.Fatalf("SaveCompleted: %v", err)
	}
	return run.Ref
}

// TestRecordFollowUpError_Conservation：失败摘要记录/清除不破坏 meta 既有
// 字段（CreatedAt/Title/Kind/Status 守恒——与终态写同一读改写管道）。
func TestRecordFollowUpError_Conservation(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	ref := seedCompletedRun(t, store)

	before, err := store.loadMeta(ref)
	if err != nil {
		t.Fatalf("loadMeta: %v", err)
	}

	// 记录失败摘要：字段落盘，其余字段原值保留。
	if err := store.RecordFollowUpError(ref, "provider 掉线：context deadline exceeded"); err != nil {
		t.Fatalf("RecordFollowUpError: %v", err)
	}
	after, err := store.loadMeta(ref)
	if err != nil {
		t.Fatalf("loadMeta after record: %v", err)
	}
	if after.FollowUpError == "" || !strings.Contains(after.FollowUpError, "provider 掉线") {
		t.Fatalf("FollowUpError = %q, want the failure summary", after.FollowUpError)
	}
	if !after.CreatedAt.Equal(before.CreatedAt) || after.Title != before.Title ||
		after.Kind != before.Kind || after.Status != before.Status {
		t.Fatalf("meta fields not conserved: before=%+v after=%+v", before, after)
	}

	// 清除（开跑语义）：摘要归零，其余字段仍守恒。
	if err := store.RecordFollowUpError(ref, ""); err != nil {
		t.Fatalf("RecordFollowUpError(clear): %v", err)
	}
	cleared, _ := store.loadMeta(ref)
	if cleared.FollowUpError != "" {
		t.Fatalf("FollowUpError = %q after clear, want empty", cleared.FollowUpError)
	}
	if !cleared.CreatedAt.Equal(before.CreatedAt) || cleared.Title != before.Title {
		t.Fatalf("clear broke conserved fields: %+v", cleared)
	}
}

// TestRecordFollowUpError_Guards：无 meta 不落幽灵记录；mt_/非法 ref 拒绝；
// 超长摘要按 rune 截断。
func TestRecordFollowUpError_Guards(t *testing.T) {
	dir := t.TempDir()
	store := NewSubagentStore(dir)

	// 无 meta：报错且不创建文件（避免空状态幽灵记录进分工列表）。
	if err := store.RecordFollowUpError("sa_20990101_000000_0000000001_deadbeef", "boom"); err == nil {
		t.Fatal("missing meta should error")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("no meta file should be created, got %d entries", len(entries))
	}

	// mt_ / 非法字符 ref：拒绝（追问只适用于 sa_ 真子代理）。
	if err := store.RecordFollowUpError("mt_20990101_000000_0000000001_deadbeef", "boom"); err == nil {
		t.Fatal("mt_ ref should be rejected")
	}
	if err := store.RecordFollowUpError("sa_bad/../escape", "boom"); err == nil {
		t.Fatal("path-traversal ref should be rejected")
	}

	// 超长摘要截断（300 rune 上限 + 省略号）。
	ref := seedCompletedRun(t, store)
	long := strings.Repeat("汉", 500)
	if err := store.RecordFollowUpError(ref, long); err != nil {
		t.Fatalf("RecordFollowUpError(long): %v", err)
	}
	meta, _ := store.loadMeta(ref)
	if got := len([]rune(meta.FollowUpError)); got != followUpErrorMaxRunes {
		t.Fatalf("summary len = %d runes, want %d", got, followUpErrorMaxRunes)
	}
	if !strings.HasSuffix(meta.FollowUpError, "…") {
		t.Fatalf("truncated summary should end with ellipsis: %q", meta.FollowUpError)
	}
}

// followUpTaskTool 组装带 transcript store 的 TaskTool（追问管道同款）。
func followUpTaskTool(t *testing.T, store *SubagentStore, prov provider.LLMProvider) *TaskTool {
	t.Helper()
	task := NewTaskTool(prov, nil, tool.NewRegistry(), 20, 0, 0.0, "", "sys", &stubGate{})
	return task.WithTranscripts(store)
}

// followUpSink 兜底空汇（FollowUpSink 对 nil onText 已安全）。
func followUpSink(ref string) event.Sink { return FollowUpSink(ref, nil) }

// TestRunFollowUp_FailureRecorded：后台失败 → 终态 failed + 失败摘要落 meta、
// CreatedAt/Title 守恒；且 RunFollowUp 返回后终态不被 running 快照回写覆盖
// （v4.64.0 回归：defer stop() 晚于终态写，meta 卡 running、再追问被拒）。
func TestRunFollowUp_FailureRecorded(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	ref := seedCompletedRun(t, store)
	before, _ := store.loadMeta(ref)

	task := followUpTaskTool(t, store, &errProvider{name: "dead"})
	runErr := task.RunFollowUp(context.Background(), ref, "追问：再展开第二点", followUpSink(ref))
	if runErr == nil {
		t.Fatal("RunFollowUp with dead provider should fail")
	}

	meta, err := store.loadMeta(ref)
	if err != nil {
		t.Fatalf("loadMeta: %v", err)
	}
	if meta.Status != SubagentFailed {
		t.Fatalf("status = %q after failed follow-up, want failed (running snapshot must not overwrite terminal state)", meta.Status)
	}
	if !strings.Contains(meta.FollowUpError, runErr.Error()) {
		t.Fatalf("FollowUpError = %q, want the runner error %q", meta.FollowUpError, runErr.Error())
	}
	if !meta.CreatedAt.Equal(before.CreatedAt) || meta.Title != before.Title || meta.Kind != before.Kind {
		t.Fatalf("meta identity not conserved: before=%+v after=%+v", before, meta)
	}
}

// TestRunFollowUp_SuccessClearsStaleError：上次失败摘要残留时重试成功 →
// 开跑清旧摘要，完成后 meta completed 且无残留（失败态只属最近一次）。
func TestRunFollowUp_SuccessClearsStaleError(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	ref := seedCompletedRun(t, store)
	if err := store.RecordFollowUpError(ref, "上一枪的失败"); err != nil {
		t.Fatalf("seed stale error: %v", err)
	}

	task := followUpTaskTool(t, store, &mockProvider{name: "ok", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "第二点的补充说明。"},
		{Type: provider.ChunkDone},
	}})
	if err := task.RunFollowUp(context.Background(), ref, "追问：再展开第二点", followUpSink(ref)); err != nil {
		t.Fatalf("RunFollowUp: %v", err)
	}

	meta, err := store.loadMeta(ref)
	if err != nil {
		t.Fatalf("loadMeta: %v", err)
	}
	if meta.Status != SubagentCompleted {
		t.Fatalf("status = %q after successful follow-up, want completed (running snapshot must not overwrite terminal state)", meta.Status)
	}
	if meta.FollowUpError != "" {
		t.Fatalf("FollowUpError = %q, want cleared by the successful retry", meta.FollowUpError)
	}
}

// TestRunFollowUp_MetaRefreshed：追问运行会刷新 UpdatedAt（轮询自校正依据），
// 且 transcript 带回追问的 user 消息（前端乐观气泡清场依据）。
func TestRunFollowUp_MetaRefreshed(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	ref := seedCompletedRun(t, store)
	before, _ := store.loadMeta(ref)
	time.Sleep(5 * time.Millisecond) // UpdatedAt 时间分辨率余量

	task := followUpTaskTool(t, store, &mockProvider{name: "ok", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "好的。"},
		{Type: provider.ChunkDone},
	}})
	if err := task.RunFollowUp(context.Background(), ref, "追问：还有吗", followUpSink(ref)); err != nil {
		t.Fatalf("RunFollowUp: %v", err)
	}
	meta, _ := store.loadMeta(ref)
	if !meta.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf("UpdatedAt not refreshed: %v -> %v", before.UpdatedAt, meta.UpdatedAt)
	}
	sess, err := session.Load(store.transcriptPathUnlocked(ref))
	if err != nil {
		t.Fatalf("load transcript: %v", err)
	}
	found := false
	for _, m := range sess.Messages {
		if m.Role == provider.RoleUser && m.Content == "追问：还有吗" {
			found = true
		}
	}
	if !found {
		t.Fatalf("follow-up prompt missing from transcript: %+v", sess.Messages)
	}
}
