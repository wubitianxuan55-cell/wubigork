package boot_test

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// effortProbe 捕获装配期 provider 构造实收的 Extra（effort 进 provider 的唯一
// 通道；Register 重复注册会 panic，用 sync.Once 保证同进程只注册一次）。
var (
	effortProbeOnce sync.Once
	effortProbeGot  map[string]any
)

// TestBuildAppliesAgentEffortBeforeProviderConstruction 钉死 GA4-01 的顺序契约：
// cfg.Agent.Effort 覆盖必须在 provider 构造前生效——探针 kind 构造时快照
// Extra["effort"]，构造后再赋值对它不可见（旧序「NewProvider 之后才覆盖」
// 本用例必红：探针看到的是空 effort）。
func TestBuildAppliesAgentEffortBeforeProviderConstruction(t *testing.T) {
	effortProbeOnce.Do(func() {
		provider.Register("effort-probe", func(cfg provider.Config) (provider.Provider, error) {
			effortProbeGot = cfg.Extra
			return testutil.NewMock("probe"), nil
		})
	})
	cfg := config.Default()
	cfg.DefaultModel = "probe"
	cfg.Tools.Enabled = nil
	cfg.Agent.Effort = "high"
	cfg.Providers = []config.ProviderEntry{{
		Name: "probe", Kind: "effort-probe", Model: "m1", ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	_, err := boot.Build(context.Background(), boot.Options{
		Model:      "probe",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		SessionDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if effortProbeGot == nil {
		t.Fatal("probe kind was never constructed")
	}
	if got, _ := effortProbeGot["effort"].(string); got != "high" {
		t.Errorf("provider 实收 Extra[%q] = %q, want %q（覆盖晚于构造=静默失效）", "effort", got, "high")
	}
}
