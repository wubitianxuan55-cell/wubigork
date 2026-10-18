//go:build windows && strata_integration

package app

// strata_integration_test.go — Strata 真实生命周期验证（默认不编译）。
// 使用 `go test -tags strata_integration` 启动 8091 的 qwen36_uncensored 档（目录
// 现存唯一档位），验证 Python 启动入口、/v1/models 探活、默认模型同步与停止；
// CI/日常单测不受影响。

import (
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

func TestStrataQwen36UncensoredLifecycle_Integration(t *testing.T) {
	mgr := modelengine.NewManager("", "")
	a := &App{core: &core{cfg: &config.Config{StrataPath: config.DefaultStrataPath}, engineMgr: mgr}}
	defer func() {
		stop := a.StrataEngineStop()
		t.Logf("stop: %v", stop)
		if stopped, _ := stop["stopped"].(bool); !stopped {
			if reason, _ := stop["reason"].(string); reason != "not-running" {
				t.Errorf("strata qwen36_uncensored profile did not stop cleanly: %v", stop)
			}
		}
	}()

	modelID := strataProfiles[0].ModelID
	res := a.StrataModelStart(modelID)
	t.Logf("start: %v", res)
	if started, _ := res["started"].(bool); !started {
		t.Fatalf("strata qwen36_uncensored model did not start: %v", res)
	}
	st := a.StrataEngineStatus()
	t.Logf("status: %v", st)
	if running, _ := st["running"].(bool); !running {
		t.Fatalf("status did not see the running strata engine: %v", st)
	}
	if profile, _ := st["profile"].(string); profile != "qwen36_uncensored" {
		t.Fatalf("profile = %q, want qwen36_uncensored (%v)", profile, st)
	}
	// 模型启动只刷新完整目录，不替用户改默认模型或功能绑定。
	if _, err := a.RefreshEngineModels("strata"); err != nil {
		t.Fatalf("refresh models: %v", err)
	}
	eng, ok := mgr.GetEngine("strata")
	if !ok || len(eng.Models) != len(strataProfiles) {
		t.Fatalf("model list not refreshed: %+v", eng)
	}
	if eng.DefaultModel != "" {
		t.Fatalf("engine start must not set default model: %+v", eng)
	}
	if got := a.StrataModelStop(modelID); got["stopped"] != true {
		t.Fatalf("model stop failed: %v", got)
	}
	if _, err := a.RefreshEngineModels("strata"); err != nil {
		t.Fatalf("refresh after stop: %v", err)
	}
	eng, _ = mgr.GetEngine("strata")
	for _, m := range eng.Models {
		if m.Status != "stopped" {
			t.Fatalf("model %s still %q after stop", m.ID, m.Status)
		}
	}
}
