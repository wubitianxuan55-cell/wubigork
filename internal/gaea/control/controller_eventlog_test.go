package control

// 3.0 Step 1 运行时接线：事件日志模式下恢复走 Restore（DetectLegacy 迁移 →
// checkpoint + log tail）、压缩后检查点可恢复、模型调用前 flush 检查点
// （fail-closed）、用户消息入日志；legacy 模式（缺省）不产生任何副作用。

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// 验收 3.2：Resume 走 checkpoint + tail 恢复（先 DetectLegacy 迁移，再 Restore）。
func TestEventResumeRestoresCheckpointPlusTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	logPath := session.LogPathFor(path)
	cpPath := session.CheckpointPathFor(path)

	w, err := session.OpenLog(logPath, "")
	if err != nil {
		t.Fatal(err)
	}
	// turn1: user + assistant_message + turn_done（seq 1-3）
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindAssistantMessage, map[string]string{"text": "a1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append("turn_done", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	// 压缩：checkpoint 落在 seq 3（压缩后投影 = system + digest）
	ck := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "<compaction-summary>…"},
	}
	if err := session.WriteCheckpoint(cpPath, 3, ck); err != nil {
		t.Fatal(err)
	}
	// turn2: user + assistant_message（seq 4-5，checkpoint 之后的 tail）
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindAssistantMessage, map[string]string{"text": "a2"}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	exec := agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard)
	c := New(Options{Runner: &fakeStateRunner{}, Executor: exec, Sink: event.Discard, SessionDir: dir, LogFormat: "event"})

	loaded, err := c.ResumeFromDisk(path)
	if err != nil {
		t.Fatalf("ResumeFromDisk: %v", err)
	}
	if !loaded.IsEventMode() {
		t.Fatal("恢复的会话应处于事件日志模式")
	}
	msgs := loaded.Snapshot()
	if len(msgs) != 4 {
		t.Fatalf("messages = %d, want 4（checkpoint 2 + tail 2）", len(msgs))
	}
	if msgs[0].Content != "sys" || msgs[1].Content != "<compaction-summary>…" {
		t.Errorf("checkpoint 段 = %+v", msgs[:2])
	}
	if msgs[2].Role != provider.RoleUser || msgs[2].Content != "u2" {
		t.Errorf("tail user = %+v", msgs[2])
	}
	if msgs[3].Role != provider.RoleAssistant || msgs[3].Content != "a2" {
		t.Errorf("tail assistant = %+v", msgs[3])
	}
	if loaded.LogSeq() != 5 {
		t.Fatalf("log seq = %d, want 5", loaded.LogSeq())
	}
	// 控制器已接管恢复的会话
	if got := c.History(); !reflect.DeepEqual(got, msgs) {
		t.Fatal("控制器会话未接管恢复结果")
	}
}

// 验收 3.3：压缩后检查点可恢复（Snapshot 在回合结束后 flush 检查点，
// 检查点本身即压缩后的完整投影，log tail 为空时直接恢复）。
func TestEventCompactionCheckpointRestorable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	c := New(Options{Runner: &fakeStateRunner{}, Executor: exec, Sink: event.Discard, SessionDir: dir, LogFormat: "event"})
	c.SetSessionPath(path)
	s := exec.Session()
	s.SetLogFormat("event")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "u1"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "a1"})
	// 压缩前的历史日志（seq 1-2）
	w, err := session.OpenLog(session.LogPathFor(path), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindAssistantMessage, map[string]string{"text": "a1"}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	// 压缩：Replace 为压缩后投影（system + digest），随后回合结束 Snapshot
	compacted := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "<digest>…"},
	}
	s.Replace(compacted)
	if err := c.Snapshot(); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	cp, err := session.ReadCheckpoint(session.CheckpointPathFor(path))
	if err != nil || cp == nil {
		t.Fatalf("压缩后应已有检查点: cp=%v err=%v", cp, err)
	}
	if !reflect.DeepEqual(cp.Messages, compacted) {
		t.Fatalf("checkpoint 消息 = %+v, want 压缩后投影 %+v", cp.Messages, compacted)
	}
	// 由该检查点恢复：checkpoint 即完整投影（tail 无新增条目）
	msgs, last, err := session.Restore(session.CheckpointPathFor(path), session.LogPathFor(path))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(msgs, compacted) {
		t.Fatalf("恢复 = %+v, want %+v", msgs, compacted)
	}
	if last != cp.Seq {
		t.Errorf("last = %d, want checkpoint seq %d", last, cp.Seq)
	}
}

// 验收 3.4：模型调用前 flush 检查点落盘（fail-closed：runner.Run 触发时
// 检查点必须已存在）；用户消息先于模型调用写入事件日志。
func TestEventFlushCheckpointBeforeModelCall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	var sawCheckpoint atomic.Bool
	run := &fakeStateRunner{onRun: func() {
		if _, err := os.Stat(session.CheckpointPathFor(path)); err == nil {
			sawCheckpoint.Store(true)
		}
	}}
	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	c := New(Options{Runner: run, Executor: exec, Sink: event.Discard, SessionDir: dir, LogFormat: "event"})
	c.SetSessionPath(path)
	s := exec.Session()
	s.SetLogFormat("event")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "u1"})
	// 历史日志（seq 1-2）：使模型调用前的 flush 有内容可固化
	w, err := session.OpenLog(session.LogPathFor(path), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append("turn_done", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	if err := c.runTurnWithRaw(context.Background(), "继续", "继续"); err != nil {
		t.Fatalf("runTurnWithRaw: %v", err)
	}
	if !sawCheckpoint.Load() {
		t.Fatal("模型调用前检查点未落盘（fail-closed 未生效）")
	}
	// 用户消息已入日志（seq 3，先于回合内事件）
	entries, err := session.ReadLog(session.LogPathFor(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[2].Kind != session.KindUserMessage || entries[2].Seq != 3 {
		t.Fatalf("用户消息未写入日志: %+v", entries)
	}
	// 回合结束 Snapshot 后检查点 seq 前移到 3（含用户消息）
	cp, err := session.ReadCheckpoint(session.CheckpointPathFor(path))
	if err != nil || cp == nil {
		t.Fatalf("回合后检查点缺失: %v", err)
	}
	if cp.Seq != 3 {
		t.Errorf("回合后 checkpoint seq = %d, want 3", cp.Seq)
	}
}

// 验收 4：legacy 模式（缺省）回合全流程不产生事件日志/检查点副作用。
func TestLegacyModeNoEventLogSideEffects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	c := New(Options{Runner: &fakeStateRunner{}, Executor: exec, Sink: event.Discard, SessionDir: dir}) // 缺省 legacy
	c.SetSessionPath(path)
	exec.Session().Add(provider.Message{Role: provider.RoleUser, Content: "u1"})

	if err := c.runTurnWithRaw(context.Background(), "继续", "继续"); err != nil {
		t.Fatalf("runTurnWithRaw: %v", err)
	}
	if session.HasEventLog(path) {
		t.Fatal("legacy 模式不应产生事件日志")
	}
	if _, err := os.Stat(session.CheckpointPathFor(path)); err == nil {
		t.Fatal("legacy 模式不应产生检查点")
	}
	// 镜像 JSONL 正常写出（旧行为）
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("legacy Save 未写镜像: %v", err)
	}
}
