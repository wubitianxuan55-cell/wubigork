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

// D3（dsh-ops 蒸馏 2026-11-10）子码与 applied/守卫类零效应测试。
func TestBashGuardFailuresAreZeroEffect(t *testing.T) {
	b := bash{workDir: t.TempDir()}
	// 空命令：validation+none（命令从未跑）。
	_, err := b.Execute(context.Background(), []byte(`{"command":""}`))
	f, ok := tool.AsFailure(err)
	if !ok || f.Code != tool.CodeValidationError || f.EffectState != tool.EffectNone {
		t.Fatalf("空命令应 validation+none: %+v", f)
	}
	// 递归删除守卫：validation+none。
	_, err = b.Execute(context.Background(), []byte(`{"command":"rm -rf /"}`))
	if f, ok := tool.AsFailure(err); !ok || f.Code != tool.CodeValidationError || f.EffectState != tool.EffectNone {
		t.Fatalf("递归删除守卫应 validation+none: %+v", f)
	}
}

func TestBashJSONModeFailureGoesErrorChannel(t *testing.T) {
	b := bash{workDir: t.TempDir()}
	out, err := b.Execute(context.Background(), []byte(`{"command":"exit 3","output_format":"json"}`))
	if err == nil {
		t.Fatalf("json 模式非零退出应走错误通道, out=%q", out)
	}
	f, ok := tool.AsFailure(err)
	if !ok {
		t.Fatalf("应为结构化 Failure: %v", err)
	}
	if f.Subcode != "EXIT_NONZERO" || f.EffectState != tool.EffectUnknown || !f.Retryable {
		t.Fatalf("json 失败语义错: %+v", f)
	}
	if !f.HasExitCode || f.ExitCode != 3 {
		t.Fatalf("退出码应结构化: %+v", f)
	}
	// 结构化流保真：stdout/stderr JSON 体作为 result 字符串透传（executeOne
	// 会进 data.detail）。
	if !strings.Contains(out, `"exit_code":3`) {
		t.Fatalf("json 体应保留结构化流: %q", out)
	}
}

func TestBashPlainExitCarriesSubcode(t *testing.T) {
	b := bash{workDir: t.TempDir()}
	_, err := b.Execute(context.Background(), []byte(`{"command":"exit 9"}`))
	f, ok := tool.AsFailure(err)
	if !ok || f.Subcode != "EXIT_NONZERO" || f.ExitCode != 9 {
		t.Fatalf("plain 非零退出应带 EXIT_NONZERO 子码: %+v", f)
	}
}
