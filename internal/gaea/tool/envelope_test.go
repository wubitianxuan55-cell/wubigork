package tool

// B2 失败即值测试（dsh-ops 蒸馏 2026-11-10）：信封三字段 roundtrip + 纯加法
// （旧字段不漂移）。盘查实证 envelope.go 此前零测试，本文件一并钉基础契约。

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvelopeNewFieldsRoundTrip(t *testing.T) {
	raw := WrapFailure(&Failure{
		Code:        CodeConflict,
		Err:         errTest("REVISION_CONFLICT: 已被外部修改"),
		Retryable:   true,
		EffectState: EffectNone,
		NextAction:  "重读拿最新 revision 后再改",
	}, "REVISION_CONFLICT: 已被外部修改", map[string]any{"tool": "edit_file"})

	env, ok := ParseEnvelope(raw)
	if !ok {
		t.Fatalf("应可解析: %q", raw)
	}
	if env.OK || env.Success {
		t.Fatalf("失败信封 ok/success 应为 false: %q", raw)
	}
	if env.Code != CodeConflict {
		t.Fatalf("code=%q want %q", env.Code, CodeConflict)
	}
	if !env.Retryable || env.EffectState != EffectNone || env.NextAction == "" {
		t.Fatalf("失败即值三字段缺失: retryable=%v effect=%q next=%q", env.Retryable, env.EffectState, env.NextAction)
	}
}

func TestEnvelopeAdditiveNoDrift(t *testing.T) {
	// 成功信封不得携带失败字段（omitempty 纪律：token 不为成功路径付费）。
	raw := WrapResult(CodeOK, map[string]any{"tool": "read_file"})
	if strings.Contains(raw, "retryable") || strings.Contains(raw, "effect_state") || strings.Contains(raw, "next_action") {
		t.Fatalf("成功信封不应带失败字段: %q", raw)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"ok", "success", "code", "data"} {
		if _, exists := m[k]; !exists {
			t.Errorf("旧字段 %q 不得漂移: %q", k, raw)
		}
	}
	// WrapError 旧口径零变化：无失败三字段。
	legacy := WrapError(CodeExecError, "boom", nil)
	if strings.Contains(legacy, "retryable") {
		t.Fatalf("WrapError 旧口径不得被改写: %q", legacy)
	}
}

func TestParseEnvelopeNonJSON(t *testing.T) {
	if _, ok := ParseEnvelope("blocked: 裸文本前缀路径"); ok {
		t.Fatal("非 JSON 不应解析为信封")
	}
	if _, ok := ParseEnvelope(""); ok {
		t.Fatal("空串不应解析为信封")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
