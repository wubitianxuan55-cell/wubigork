package tool

// Failure 结构化失败测试（B2 失败即值，dsh-ops 蒸馏 2026-11-10）。

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFailureErrorPassesThroughText(t *testing.T) {
	// 前缀文本是兼容层：既有字符串消费方（测试/守卫）依赖 REVISION_CONFLICT
	// 等前缀原样可见。
	f := &Failure{Code: CodeConflict, Err: fmt.Errorf("REVISION_CONFLICT: a.txt 已被修改")}
	if got := f.Error(); got != "REVISION_CONFLICT: a.txt 已被修改" {
		t.Fatalf("文案应原样透传: %q", got)
	}
	var target *Failure
	if !errors.As(error(f), &target) || target.Code != CodeConflict {
		t.Fatal("errors.As 应命中 Failure（含包装链）")
	}
	// 包装链穿透：fmt.Errorf %w 外裹一层仍可 AsFailure。
	wrapped := fmt.Errorf("batch: %w", f)
	if af, ok := AsFailure(wrapped); !ok || af.Code != CodeConflict {
		t.Fatalf("包装链应命中: ok=%v", ok)
	}
	if _, ok := AsFailure(fmt.Errorf("plain")); ok {
		t.Fatal("普通错误不应误报 Failure")
	}
}

func TestFailureNilSafety(t *testing.T) {
	var f *Failure
	if got := f.Error(); got == "" {
		t.Fatal("nil 接收者 Error() 应有兜底文案")
	}
	codeOnly := &Failure{Code: CodeTimeout}
	if got := codeOnly.Error(); got != CodeTimeout {
		t.Fatalf("无底层错误时应回落 code: %q", got)
	}
}

func TestWrapFailureDefaults(t *testing.T) {
	// 站点未填 NextAction → 按码+effect 默认表补；unknown 必须带「先核实」。
	raw := WrapFailure(&Failure{
		Code:        CodeTimeout,
		Retryable:   true,
		EffectState: EffectUnknown,
	}, "command timed out", nil)
	env, _ := ParseEnvelope(raw)
	if env.NextAction == "" || !strings.Contains(env.NextAction, "未知") || !strings.Contains(env.NextAction, "再决定重试") {
		t.Fatalf("timeout+unknown 默认动作应含先核实指引: %q", env.NextAction)
	}
	if env.Code != CodeTimeout {
		t.Fatalf("code=%q want timeout", env.Code)
	}
	// 空 code → exec_error 兜底。
	raw2 := WrapFailure(&Failure{Err: fmt.Errorf("x")}, "x", nil)
	env2, _ := ParseEnvelope(raw2)
	if env2.Code != CodeExecError {
		t.Fatalf("空 code 应兜底 exec_error: %q", env2.Code)
	}
}

func TestDefaultRetryableTable(t *testing.T) {
	for _, c := range []string{CodeTimeout, CodeExecError} {
		if !DefaultRetryable(c) {
			t.Errorf("%s 应默认可重试", c)
		}
	}
	for _, c := range []string{CodeValidationError, CodeDenied, CodeNotFound, CodeConflict, CodeUnknownTool, CodeBlocked} {
		if DefaultRetryable(c) {
			t.Errorf("%s 应默认不可原样重试", c)
		}
	}
}
