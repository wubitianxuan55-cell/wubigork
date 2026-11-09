package app

import "testing"

// TestLlamaEnsureRequiresExplicitProfile 锁死「引擎启动不捆绑默认模型」：
// 空档位直接返回 model-required，不允许隐式拉起 qwen35。
func TestLlamaEnsureRequiresExplicitProfile(t *testing.T) {
	a := &App{}
	got := a.llamaEnsure("")
	if started, _ := got["started"].(bool); started {
		t.Fatalf("空 profile 不应启动模型: %#v", got)
	}
	if got["reason"] != "model-required" {
		t.Fatalf("reason = %#v, want model-required", got["reason"])
	}
}
