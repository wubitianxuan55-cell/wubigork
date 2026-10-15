package app

// 阶段十 10.1 量化红线测试（出口判据「低档警告有触发用例」）：警告去重
//（同 feature+model 一次）+ 达标/云端/无标记不警告。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

func TestQuantToolCallWarnOnce(t *testing.T) {
	c := newQuantWarnTestCore(t)

	w1 := c.quantToolCallWarnOnce("chat", "herdsman", "qwen3-8b-iq2_xs")
	if w1 == "" {
		t.Fatal("IQ2 本地首次应警告")
	}
	// 去重：同 feature+model 第二次为空
	if w2 := c.quantToolCallWarnOnce("chat", "herdsman", "qwen3-8b-iq2_xs"); w2 != "" {
		t.Fatalf("同组合应去重: %q", w2)
	}
	// 换 feature 重评一次
	if w3 := c.quantToolCallWarnOnce("agent", "herdsman", "qwen3-8b-iq2_xs"); w3 == "" {
		t.Fatal("换 feature 应重评")
	}
}

func TestQuantToolCallWarnOnce_NoWarnCases(t *testing.T) {
	c := newQuantWarnTestCore(t)
	// Q4 达标下限
	if w := c.quantToolCallWarnOnce("chat", "herdsman", "model-Q4_K_P"); w != "" {
		t.Fatalf("Q4 不应警告: %q", w)
	}
	// 云端引擎（xai 不 IsLocal）
	if w := c.quantToolCallWarnOnce("chat", "xai", "grok-iq2_xs"); w != "" {
		t.Fatalf("云端不应警告: %q", w)
	}
	// 无量化标记
	if w := c.quantToolCallWarnOnce("chat", "herdsman", "Qwen3.6-35B-A3B-MTP"); w != "" {
		t.Fatalf("无标记不应警告: %q", w)
	}
	// 未登记引擎
	if w := c.quantToolCallWarnOnce("chat", "nope", "qwen-iq2_xs"); w != "" {
		t.Fatalf("未登记引擎不应警告: %q", w)
	}
}

func TestQuantWarningMessageReadable(t *testing.T) {
	w := modelengine.ToolCallQuantWarning(modelengine.EngineHerdsman, "model-iq3_m")
	if !strings.Contains(w, "IQ3") || !strings.Contains(w, "Q4") {
		t.Fatalf("警告须点名档位与建议下限: %q", w)
	}
}

func newQuantWarnTestCore(t *testing.T) *core {
	t.Helper()
	c := &core{cfg: &config.Config{}}
	c.engineMgr = modelengine.NewManager("", "")
	return c
}
