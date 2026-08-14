package config

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ── slog 捕获测试基建（T6-1.3 吞错清理日志断言用）──────────────────

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

func logContainsMsg(records []slog.Record, substr string) bool {
	for _, r := range records {
		if strings.Contains(r.Message, substr) {
			return true
		}
	}
	return false
}

// withTempHome 在临时 HOME 下执行 fn（不污染真实配置文件）。
func withTempHome(t *testing.T, fn func(home string)) {
	t.Helper()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})
	fn(tmpHome)
}

// ── config Load 坏 JSON 吞错 → 日志（T6-1.3）──────────────────────

// TestLoad_CorruptConfigJSONLogs 主配置文件是坏 JSON 时，Load 必须记录
// 解析失败日志（忽略文件覆盖，保留默认值）。
func TestLoad_CorruptConfigJSONLogs(t *testing.T) {
	withTempHome(t, func(home string) {
		if err := os.WriteFile(filepath.Join(home, ".gaea_config.json"), []byte("{corrupt json"), 0o644); err != nil {
			t.Fatalf("写坏配置文件: %v", err)
		}
		records := captureLogs(t, func() {
			cfg := Load()
			if cfg == nil {
				t.Fatal("Load 返回 nil")
			}
			if cfg.NovelsDir != filepath.FromSlash("C:/AI/xiaoshuo") {
				t.Errorf("坏 JSON 时应保留默认值, NovelsDir = %q", cfg.NovelsDir)
			}
		})
		if !logContainsMsg(records, "配置文件解析失败") {
			t.Fatalf("未记录配置文件解析失败日志, got %v", records)
		}
	})
}
