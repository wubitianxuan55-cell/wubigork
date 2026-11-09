package boot_test

// P0 注册表地图守卫·boot 运行时普查（2026-11-09 众神殿裁决 action-1）：用 mock
// provider 走完整 boot.Build（唯一注册表），断言运行时注册表内每个工具名都在
// RegistryMap 有契约行——比静态 grep 更硬，能逮 Build 内部新增 reg.Add 与未来
// 装配面漂移。 ExtraTools 侧由 app 包 registrymap_extra_test.go 覆盖。

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
	"github.com/gaea/gaea/internal/gaea/tool"
)

var registryCensusMockOnce sync.Once

func registerRegistryCensusMockKind() {
	registryCensusMockOnce.Do(func() {
		provider.Register("test-mock-registry-census", func(cfg provider.Config) (provider.Provider, error) {
			return testutil.NewMock("mock"), nil
		})
	})
}

func TestRegistryMapCoversBootCensus(t *testing.T) {
	registerRegistryCensusMockKind()
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	// 桌面端形态：Enabled=nil 全量注册内置（gaea_handler.go 同款）。
	cfg.Tools.Enabled = nil
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: "test-mock-registry-census", Model: "m1", ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model: "mock", RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		SessionDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("boot.Build: %v", err)
	}
	var missing []string
	for _, tl := range ctrl.Tools() {
		if _, ok := tool.RegistryMap[tl.Name()]; !ok {
			missing = append(missing, tl.Name())
		}
	}
	if len(missing) > 0 {
		t.Fatalf("boot.Build 运行时注册面有工具缺 P0 契约行: %v", missing)
	}
}
