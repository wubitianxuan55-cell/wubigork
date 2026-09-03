package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// TestModelToolRunLifecycle 覆盖 mt_ 记录全生命周期：NewModelToolRun（running
// meta + transcript 首条 user）、UpdateModelToolTitle（参数齐后刷新标题）、
// FinishModelTool（成功/失败各一），以及 meta 字段（kind/title/tool）守恒。
func TestModelToolRunLifecycle(t *testing.T) {
	store := NewSubagentStore(t.TempDir())

	run, err := store.NewModelToolRun("vision · 识别图片 a.png", "vision", spaces.SpaceWork)
	if err != nil {
		t.Fatalf("NewModelToolRun: %v", err)
	}
	if run.Ref == "" || !IsModelToolRef(run.Ref) {
		t.Fatalf("ref = %q, want mt_ prefix", run.Ref)
	}
	meta, err := store.loadMeta(run.Ref)
	if err != nil {
		t.Fatalf("loadMeta: %v", err)
	}
	if meta.Kind != SubagentKindModelTool || meta.Status != SubagentRunning ||
		meta.Tool != "vision" || meta.Title == "" || meta.Space != spaces.SpaceWork {
		t.Fatalf("running meta wrong: %+v", meta)
	}

	// 标题刷新（参数完整后）：transcript 首条 user 与 meta.Title 同步。
	fullLabel := "vision · 识别图片 C:\\x.png（提取图中所有文字）"
	if err := store.UpdateModelToolTitle(run.Ref, fullLabel); err != nil {
		t.Fatalf("UpdateModelToolTitle: %v", err)
	}
	sess, err := session.Load(store.transcriptPathUnlocked(run.Ref))
	if err != nil {
		t.Fatalf("load transcript: %v", err)
	}
	msgs := sess.Messages
	if len(msgs) != 1 || msgs[0].Role != provider.RoleUser || msgs[0].Content != fullLabel {
		t.Fatalf("transcript first user wrong: %+v", msgs)
	}

	// 成功收尾：结果入 assistant 行、meta completed，CreatedAt 保持原值。
	created := meta.CreatedAt
	if err := store.FinishModelTool(run.Ref, "图中有文字“你好”", nil); err != nil {
		t.Fatalf("FinishModelTool: %v", err)
	}
	meta, err = store.loadMeta(run.Ref)
	if err != nil {
		t.Fatalf("loadMeta after finish: %v", err)
	}
	if meta.Status != SubagentCompleted || meta.Title != fullLabel || meta.Tool != "vision" {
		t.Fatalf("completed meta wrong: %+v", meta)
	}
	if !meta.CreatedAt.Equal(created) {
		t.Fatalf("CreatedAt changed: %v -> %v", created, meta.CreatedAt)
	}
	sess, _ = session.Load(store.transcriptPathUnlocked(run.Ref))
	msgs = sess.Messages
	if len(msgs) != 2 || msgs[1].Role != provider.RoleAssistant || !strings.Contains(msgs[1].Content, "你好") {
		t.Fatalf("final transcript wrong: %+v", msgs)
	}

	// 失败收尾：err 进 assistant 行、meta failed。
	run2, err := store.NewModelToolRun("summarize_file", "summarize_file", "")
	if err != nil {
		t.Fatalf("NewModelToolRun #2: %v", err)
	}
	if err := store.FinishModelTool(run2.Ref, "", os.ErrNotExist); err != nil {
		t.Fatalf("FinishModelTool failed path: %v", err)
	}
	meta2, _ := store.loadMeta(run2.Ref)
	if meta2.Status != SubagentFailed {
		t.Fatalf("failed meta status = %q, want failed", meta2.Status)
	}
	if meta2.Space != spaces.SpaceWork {
		t.Fatalf("empty space should normalize to work, got %q", meta2.Space)
	}
}

// TestTrackProgressFlushesBeforeStop 验证 TrackProgress：运行中快照持续落盘、
// stop() 阻塞完成最终 flush，随后终态写不会回退 running。
func TestTrackProgressFlushesBeforeStop(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	run, err := store.PrepareFresh("sys", spaces.SpaceWork)
	if err != nil {
		t.Fatalf("PrepareFresh: %v", err)
	}
	run.Title = "调研 X"
	if err := store.MarkRunning(run); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}
	stop := store.TrackProgress(run, 10*time.Millisecond)
	defer stop()
	run.Session.Add(provider.Message{Role: provider.RoleUser, Content: "调研 X 的背景"})
	run.Session.Add(provider.Message{Role: provider.RoleAssistant, Content: "部分结论"})
	// 等待至少一次 ticker flush。
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		sess, err := session.Load(store.transcriptPathUnlocked(run.Ref))
		if err == nil && len(sess.Messages) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stop() // final flush + 等 goroutine 退出
	if err := store.SaveCompleted(run); err != nil {
		t.Fatalf("SaveCompleted: %v", err)
	}
	meta, err := store.loadMeta(run.Ref)
	if err != nil {
		t.Fatalf("loadMeta: %v", err)
	}
	if meta.Status != SubagentCompleted {
		t.Fatalf("status = %q after finalize, want completed", meta.Status)
	}
	if meta.Title != "调研 X" {
		t.Fatalf("title = %q, want 调研 X", meta.Title)
	}
	sess2, err := session.Load(store.transcriptPathUnlocked(run.Ref))
	if err != nil || len(sess2.Messages) < 2 {
		t.Fatalf("final transcript missing content: %v (%d msgs)", err, len(sess2.Messages))
	}
}

// TestLazySubagentStoreResolvesDir 验证惰性 store 随 dirFn 解析（boot 会话切换
// 语义）：解析器未就绪时不写盘报错，就绪后正常落盘。
func TestLazySubagentStoreResolvesDir(t *testing.T) {
	root := t.TempDir()
	var src func() string
	store := NewLazySubagentStore(func() string {
		if src == nil {
			return ""
		}
		return src()
	})
	if _, err := store.PrepareFresh("sys", spaces.SpaceWork); err != nil {
		t.Fatalf("PrepareFresh with empty dir should not fail (lazy): %v", err)
	}
	dir := filepath.Join(root, "sessions", "subagents")
	src = func() string { return dir }
	run, err := store.PrepareFreshWithTitle("sys", spaces.SpaceWork, "标题")
	if err != nil {
		t.Fatalf("PrepareFreshWithTitle: %v", err)
	}
	run.Session.Add(provider.Message{Role: provider.RoleUser, Content: "任务"})
	if err := store.MarkRunning(run); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}
	if err := store.SaveCompleted(run); err != nil {
		t.Fatalf("SaveCompleted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, run.Ref+".jsonl")); err != nil {
		t.Fatalf("transcript missing: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, run.Ref+".meta.json"))
	if err != nil {
		t.Fatalf("meta missing: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	if meta["title"] != "标题" || meta["kind"] != SubagentKindSubagent {
		t.Fatalf("meta fields wrong: %v", meta)
	}
}
