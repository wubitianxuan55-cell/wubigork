package builtin

// B2 失败即值——bash 前台失败分类测试（dsh-ops 蒸馏 2026-11-10）：
// 非零退出=exec_error+effect unknown+退出码结构化；超时=timeout 码；
// 取消=不可重试。真 bash 工具级（与 publishguard_wiring_test 同构造）。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func TestBashFailureExitNonZero(t *testing.T) {
	b := bash{workDir: t.TempDir()}
	out, err := b.Execute(context.Background(), []byte(`{"command":"exit 7"}`))
	if err == nil {
		t.Fatalf("非零退出应为错误, out=%q", out)
	}
	f, ok := tool.AsFailure(err)
	if !ok {
		t.Fatalf("应为结构化 Failure: %v", err)
	}
	if f.Code != tool.CodeExecError || f.EffectState != tool.EffectUnknown || !f.Retryable {
		t.Fatalf("非零退出失败语义错: code=%q effect=%q retryable=%v", f.Code, f.EffectState, f.Retryable)
	}
	if !f.HasExitCode || f.ExitCode != 7 {
		t.Fatalf("退出码应结构化: has=%v code=%d", f.HasExitCode, f.ExitCode)
	}
	if !strings.Contains(err.Error(), "command exited") {
		t.Fatalf("旧文案兼容层缺失: %v", err)
	}
}

func TestBashFailureCancellationNotRetryable(t *testing.T) {
	b := bash{workDir: t.TempDir()}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()
	_, err := b.Execute(ctx, []byte(`{"command":"sleep 30"}`))
	if err == nil {
		t.Fatal("取消应产生错误")
	}
	f, ok := tool.AsFailure(err)
	if !ok {
		t.Fatalf("应为结构化 Failure: %v", err)
	}
	if f.Retryable {
		t.Fatal("回合取消不可原样重试")
	}
	if f.EffectState != tool.EffectUnknown {
		t.Fatalf("中断副作用应 unknown: %q", f.EffectState)
	}
}
