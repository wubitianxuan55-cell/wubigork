package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/characterlib"
)

// ── slog 捕获测试基建（T6-1.3 吞错清理日志断言用）──────────────────

// captureHandler 收集 slog 记录用于测试断言（不向终端输出）。
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func (h *captureHandler) snapshot() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]slog.Record(nil), h.records...)
}

// captureLogs 在 fn 执行期间接管全局 slog 默认 logger，返回期间产生的全部记录。
func captureLogs(t *testing.T, fn func()) []slog.Record {
	t.Helper()
	h := &captureHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	defer slog.SetDefault(orig)
	fn()
	return h.snapshot()
}

func logMsgs(records []slog.Record) []string {
	out := make([]string, 0, len(records))
	for _, r := range records {
		out = append(out, r.Message)
	}
	return out
}

func logContainsMsg(records []slog.Record, substr string) bool {
	for _, r := range records {
		if strings.Contains(r.Message, substr) {
			return true
		}
	}
	return false
}

// ── ChatTopicsList / ChatMessagesList 吞错 → 日志（T6-1.3）──────────

// TestChatTopicsList_StoreErrorLogs 存储不可用（已关闭）时列表失败必须记录日志，
// 并且错误透传给调用方（T6-3.2 不再返回 nil 掩盖失败）。
func TestChatTopicsList_StoreErrorLogs(t *testing.T) {
	a := newChatServiceTestApp(t)
	if err := a.chatStore.Close(); err != nil {
		t.Fatalf("关闭 chatStore: %v", err)
	}
	records := captureLogs(t, func() {
		got, err := a.ChatTopicsList()
		if err == nil {
			t.Errorf("存储不可用时应返回错误, got %+v", got)
		}
	})
	if !logContainsMsg(records, "chat 话题列表读取失败") {
		t.Fatalf("未记录话题列表读取失败日志, got %v", logMsgs(records))
	}
}

// TestChatMessagesList_StoreErrorLogs 存储不可用（已关闭）时消息列表失败必须记录日志，
// 并且错误透传给调用方（T6-3.2）。
func TestChatMessagesList_StoreErrorLogs(t *testing.T) {
	a := newChatServiceTestApp(t)
	if err := a.chatStore.Close(); err != nil {
		t.Fatalf("关闭 chatStore: %v", err)
	}
	records := captureLogs(t, func() {
		got, err := a.ChatMessagesList("t1")
		if err == nil {
			t.Errorf("存储不可用时应返回错误, got %+v", got)
		}
	})
	if !logContainsMsg(records, "chat 消息列表读取失败") {
		t.Fatalf("未记录消息列表读取失败日志, got %v", logMsgs(records))
	}
}

// ── whisper_handler 两处吞错 → 日志（T6-1.3）───────────────────────

// TestWhisperChat_SwallowLogs 数据根目录不可用（被普通文件占位）时，
// 会话状态恢复与轮次追踪落库失败都必须记录日志（不改变对外行为）。
func TestWhisperChat_SwallowLogs(t *testing.T) {
	a := newChatServiceTestApp(t)
	// 破坏 dataRoot：用普通文件占位，使 db.GetDatabase 初始化失败
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("写占位文件: %v", err)
	}
	a.whisperState.whisperDataRoot = blocker

	pid := fmt.Sprintf("swallow-e2e-%d", time.Now().UnixNano())
	records := captureLogs(t, func() {
		if _, err := a.whisperState.WhisperChat("你好", pid, false); err != nil {
			t.Fatalf("WhisperChat 不应因数据根不可用而失败: %v", err)
		}
	})
	if !logContainsMsg(records, "会话状态恢复失败") {
		t.Fatalf("未记录会话状态恢复失败日志, got %v", logMsgs(records))
	}
	if !logContainsMsg(records, "轮次追踪落库失败") {
		t.Fatalf("未记录轮次追踪落库失败日志, got %v", logMsgs(records))
	}
}

// ── characterlib_handler 吞错 → 日志（T6-1.3）─────────────────────

// TestCharacterDelete_AssistantCleanupFailureLogs 删除角色时聊天通道清理
// 失败必须记录日志，而不是静默丢弃。
func TestCharacterDelete_AssistantCleanupFailureLogs(t *testing.T) {
	a := newCharacterLibTestApp(t)
	custom := characterlib.Character{
		ID: "del_ghost_1", Name: "幽灵通道", Kind: characterlib.KindCustom,
		AssistantID: "ghost-not-exist", // 指向不存在的助手 → Delete 报错
	}
	if err := a.charLib.Upsert(&custom); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	records := captureLogs(t, func() {
		if err := a.CharacterDelete("del_ghost_1"); err != nil {
			t.Fatalf("CharacterDelete 不应因清理通道失败而失败: %v", err)
		}
	})
	if !logContainsMsg(records, "角色删除时清理聊天通道失败") {
		t.Fatalf("未记录聊天通道清理失败日志, got %v", logMsgs(records))
	}
}

// ── gaea_ui 注册表坏 JSON 吞错 → 日志（T6-1.3）─────────────────────

// TestLoadPinned_CorruptFileLogs 置顶注册表损坏时按空处理并记录日志。
func TestLoadPinned_CorruptFileLogs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".pinned.json"), []byte("{corrupt json"), 0o644); err != nil {
		t.Fatalf("写坏置顶文件: %v", err)
	}
	records := captureLogs(t, func() {
		m := loadPinned(dir)
		if len(m) != 0 {
			t.Errorf("坏 JSON 应按空处理, got %v", m)
		}
	})
	if !logContainsMsg(records, "置顶注册表解析失败") {
		t.Fatalf("未记录置顶注册表解析失败日志, got %v", logMsgs(records))
	}
}
