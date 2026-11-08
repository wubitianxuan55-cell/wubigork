package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sandbox"
)

// TestBashFailClosed 锁定 2026-11-08 众神殿结论第四步（pan_1794098356822）：
// 请求了 enforce 隔离但平台给不了（Windows 无 WSL2/容器）时——
//   - FailClosed=true：拒绝执行并说明原因（无隔离执行比拒绝更危险）；
//   - FailClosed=false（默认）：维持 warned-unconfined 既有行为，命令照跑；
//   - Mode 非 enforce：与 fail-closed 无关，照常执行。
func TestBashFailClosed(t *testing.T) {
	if sandbox.Available() {
		t.Skip("本机有可用沙箱（WSL2/Seatbelt），拒绝路径无法复现——活体验证需在无隔离环境跑")
	}

	t.Run("fail_closed_refuses_enforce_unavailable", func(t *testing.T) {
		b := bash{sb: sandbox.Spec{Mode: "enforce", FailClosed: true}, workDir: t.TempDir()}
		args, _ := json.Marshal(map[string]string{"command": "echo hi"})
		out, err := b.Execute(context.Background(), args)
		if err == nil {
			t.Fatalf("fail-closed must refuse, got output %q", out)
		}
		if !strings.Contains(err.Error(), "fail-closed") {
			t.Fatalf("refusal must explain the policy, got %v", err)
		}
	})

	t.Run("default_off_keeps_warned_unconfined", func(t *testing.T) {
		b := bash{sb: sandbox.Spec{Mode: "enforce"}, workDir: t.TempDir()}
		args, _ := json.Marshal(map[string]string{"command": "echo failclosed-probe"})
		out, err := b.Execute(context.Background(), args)
		if err != nil || !strings.Contains(out, "failclosed-probe") {
			t.Fatalf("default (FailClosed off) must keep the warned-unconfined behaviour: out=%q err=%v", out, err)
		}
	})

	t.Run("mode_off_unaffected", func(t *testing.T) {
		b := bash{sb: sandbox.Spec{Mode: "off", FailClosed: true}, workDir: t.TempDir()}
		args, _ := json.Marshal(map[string]string{"command": "echo off-probe"})
		out, err := b.Execute(context.Background(), args)
		if err != nil || !strings.Contains(out, "off-probe") {
			t.Fatalf("Mode=off is not an isolation request; fail-closed must not apply: out=%q err=%v", out, err)
		}
	})
}
