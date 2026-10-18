//go:build windows && tianxuan_integration

package app

// tianxuan_integration_test.go — 真实引擎生命周期验证（默认不编译）。
// 使用 `go test -tags tianxuan_integration` 才会拉起 8094 的 uncensored 档（目录
// 现存唯一档位），用于验证
// 「模型中心按钮 → 外部启动器 → /v1/models → 默认模型同步 → 停止」全链路；
// CI/日常单测不受影响。

import (
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

func TestTianxuanUncensoredLifecycle_Integration(t *testing.T) {
	mgr := modelengine.NewManager("", "")
	a := &App{core: &core{cfg: &config.Config{TianxuanPath: config.DefaultTianxuanPath}, engineMgr: mgr}}
	defer func() {
		stop := a.TianxuanEngineStop()
		t.Logf("stop: %v", stop)
		if stopped, _ := stop["stopped"].(bool); !stopped {
			if reason, _ := stop["reason"].(string); reason != "not-running" {
				t.Errorf("uncensored profile did not stop cleanly: %v", stop)
			}
		}
	}()

	modelID := tianxuanProfiles[0].ModelID
	res := a.TianxuanModelStart(modelID)
	t.Logf("start: %v", res)
	if started, _ := res["started"].(bool); !started {
		t.Fatalf("uncensored model did not start: %v", res)
	}
	st := a.TianxuanEngineStatus()
	t.Logf("status: %v", st)
	if running, _ := st["running"].(bool); !running {
		t.Fatalf("status did not see the running engine: %v", st)
	}
	if profile, _ := st["profile"].(string); profile != "uncensored" {
		t.Fatalf("profile = %q, want uncensored (%v)", profile, st)
	}
	// 模型启动只刷新完整目录，不替用户改默认模型或功能绑定。
	if _, err := a.RefreshEngineModels("tianxuan"); err != nil {
		t.Fatalf("refresh models: %v", err)
	}
	eng, ok := mgr.GetEngine("tianxuan")
	if !ok || len(eng.Models) != len(tianxuanProfiles) {
		t.Fatalf("model list not refreshed: %+v", eng)
	}
	if eng.DefaultModel != "" {
		t.Fatalf("engine start must not set default model: %+v", eng)
	}
	if got := a.TianxuanModelStop(modelID); got["stopped"] != true {
		t.Fatalf("model stop failed: %v", got)
	}
	if _, err := a.RefreshEngineModels("tianxuan"); err != nil {
		t.Fatalf("refresh after stop: %v", err)
	}
	eng, _ = mgr.GetEngine("tianxuan")
	for _, m := range eng.Models {
		if m.Status != "stopped" {
			t.Fatalf("model %s still %q after stop", m.ID, m.Status)
		}
	}
}
