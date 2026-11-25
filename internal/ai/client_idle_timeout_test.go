package ai

import (
	"errors"
	"testing"
	"time"

	gaeacfg "github.com/gaea/gaea/internal/gaea/config"
)

// TestStreamIdleTimeoutResolution 流空闲超时的解析顺序（2026-11-25 紧急修复）：
// 测试注入字段 > gaea 配置 [agent] stream_idle_timeout_secs > 内置默认。
// 回归钉：内置默认必须是分钟级——回到 60s 会再次把推理模型出字前的长思考
// 当成空闲切断（用户实测办公造价档「动不动就流空闲超时：超过 1m0s 无数据」）。
func TestStreamIdleTimeoutResolution(t *testing.T) {
	if defaultStreamIdleTimeout < 5*time.Minute {
		t.Fatalf("内置流空闲超时 %s 过短：推理模型出第一个字前可能静默数分钟", defaultStreamIdleTimeout)
	}

	cfg := gaeacfg.Default()
	gaeacfg.SetLoader(func() (*gaeacfg.Config, error) { return cfg, nil })
	t.Cleanup(func() { gaeacfg.SetLoader(nil) })

	c := &Client{} // NewClient 不预填该字段：零值=未注入
	if got := c.idleTimeout(); got != defaultStreamIdleTimeout {
		t.Fatalf("未配置时应用内置默认 %s，得到 %s", defaultStreamIdleTimeout, got)
	}

	cfg.Agent.StreamIdleTimeoutSecs = 900
	if got := c.idleTimeout(); got != 15*time.Minute {
		t.Fatalf("配置覆盖 900s 应得 15m0s，得到 %s", got)
	}

	c.streamIdleTimeout = 2 * time.Second
	if got := c.idleTimeout(); got != 2*time.Second {
		t.Fatalf("测试注入应最优先，得到 %s", got)
	}

	// 配置读不出：按内置默认，不因配置问题改变超时行为。
	gaeacfg.SetLoader(func() (*gaeacfg.Config, error) { return nil, errors.New("config unavailable") })
	c.streamIdleTimeout = 0
	if got := c.idleTimeout(); got != defaultStreamIdleTimeout {
		t.Fatalf("配置读不出时应回内置默认，得到 %s", got)
	}

	// 错误文案里的时长随生效值走（用户看到的「超过 1m0s」就是它）。
	c.streamIdleTimeout = 90 * time.Second
	if got := c.idleTimeout().String(); got != "1m30s" {
		t.Fatalf("时长格式化 = %q, want 1m30s", got)
	}
}
