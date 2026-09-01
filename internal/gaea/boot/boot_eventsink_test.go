package boot_test

// 3.0 Step 1：事件日志 sink 接线测试（回退开关 session.log_format）。
// event 模式（缺省）：Send 一轮后 <id>.gaea-log.jsonl 落盘、seq 连续、
// turn 生命周期（turn_started/turn_done）入日志、回合边界写入器关闭
// （文件可删除）；显式 "legacy"：不产生事件日志（旧行为不变）。

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// buildEventModeCtrl 以指定 log_format 装配控制器（mock provider，不依赖网络）。
func buildEventModeCtrl(t *testing.T, kind string, logFormat string) (*control.Controller, string) {
	t.Helper()
	chdirTemp(t)
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock",
			testutil.Turn{Text: "好的，收到。"},
			testutil.Turn{Text: "好的，收到。"},
			testutil.Turn{Text: "好的，收到。"},
		), nil
	})
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Session.LogFormat = logFormat
	cfg.Providers = []config.ProviderEntry{{
		Name:          "mock",
		Kind:          kind,
		Model:         "grok-3",
		ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	sessDir := t.TempDir()
	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:      "mock",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		SessionDir: sessDir,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Cleanup(ctrl.Close)
	return ctrl, sessDir
}

// resumeAndSend 恢复固定会话并 Send 一轮（交互路径，回合结束 Emit TurnDone）。
func resumeAndSend(t *testing.T, ctrl *control.Controller, sessDir string) string {
	t.Helper()
	path := filepath.Join(sessDir, "s1.jsonl")
	s := agent.NewSession("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	ctrl.Resume(s, path)
	ctrl.Send("你好")
	return path
}

// waitTurnDone 轮询日志直到出现 turn_done，返回读到的条目（此时日志尚未删除）。
func waitTurnDone(t *testing.T, logPath string) []session.LogEntry {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := session.ReadLogRepaired(logPath)
		if err != nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		for _, e := range entries {
			if e.Kind == "turn_done" {
				return entries
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("turn_done not observed in %s within 10s", logPath)
	return nil
}

// waitFileUnlocked 轮询直到文件可删除（回合边界写入器已关闭）。
func waitFileUnlocked(t *testing.T, logPath string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := os.Remove(logPath); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("log writer not closed after turn_done: %s", logPath)
}

// TestBuildEventLogModeWritesLog：event 模式下事件日志落盘、seq 连续、
// turn 生命周期完整、回合边界写入器关闭。
func TestBuildEventLogModeWritesLog(t *testing.T) {
	ctrl, sessDir := buildEventModeCtrl(t, testKind("test-mock-boot-eventlog"), "event")
	logPath := filepath.Join(sessDir, "s1.gaea-log.jsonl")
	resumeAndSend(t, ctrl, sessDir)

	entries := waitTurnDone(t, logPath)
	waitFileUnlocked(t, logPath)

	if len(entries) == 0 {
		t.Fatal("event log is empty after a turn")
	}
	for i, e := range entries {
		if e.Seq != int64(i+1) {
			t.Fatalf("seq[%d] = %d, want %d", i, e.Seq, i+1)
		}
	}
	hasStart, hasDone := false, false
	for _, e := range entries {
		if e.Kind == "turn_started" {
			hasStart = true
		}
		if e.Kind == "turn_done" {
			hasDone = true
		}
	}
	if !hasStart || !hasDone {
		t.Errorf("log missing turn lifecycle: start=%v done=%v", hasStart, hasDone)
	}
}

// TestBuildDefaultEventLog：缺省 log_format（空）即事件日志——轨迹/上下文
// 看板数据源；显式 "legacy" 才关闭。
func TestBuildDefaultEventLog(t *testing.T) {
	ctrl, sessDir := buildEventModeCtrl(t, testKind("test-mock-boot-default-event"), "")
	path := filepath.Join(sessDir, "s1.jsonl")
	s := agent.NewSession("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	ctrl.Resume(s, path)
	ctrl.Send("你好")

	logPath := filepath.Join(sessDir, "s1.gaea-log.jsonl")
	entries := waitTurnDone(t, logPath)
	waitFileUnlocked(t, logPath)
	if len(entries) == 0 {
		t.Fatal("缺省模式应产生事件日志（轨迹/上下文数据源）")
	}
}

// TestBuildExplicitLegacyNoEventLog：显式 log_format="legacy" 不产生事件日志
// （旧行为整文件重写 JSONL）。
func TestBuildExplicitLegacyNoEventLog(t *testing.T) {
	ctrl, sessDir := buildEventModeCtrl(t, testKind("test-mock-boot-explicit-legacy"), "legacy")
	path := filepath.Join(sessDir, "s1.jsonl")
	s := agent.NewSession("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	ctrl.Resume(s, path)
	ctrl.Send("你好")

	// 等待回合完成（legacy 无日志可轮询，用 Running 状态）
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !ctrl.Running() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	entries, err := os.ReadDir(sessDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "gaea-log.jsonl") {
			t.Fatalf("显式 legacy 模式不应产生事件日志: %s", e.Name())
		}
	}
}
