package app

// 工具装配面面板缝测试（2026-11-21）：绑定形状 + 引擎未初始化的诚实缺省。

import (
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// TestGaeaToolSurfaceUninitialized 引擎未装配：Tools 空、Mode=配置默认档
//（前端落静态清单兜底的判据）。
func TestGaeaToolSurfaceUninitialized(t *testing.T) {
	a := &App{}
	view := a.GaeaToolSurface()
	if view.Board != "gaea" || view.Mode != "standard" || len(view.Tools) != 0 {
		t.Fatalf("未初始化面形状漂移: %+v", view)
	}
	wx := a.WenxueToolSurface()
	if wx.Board != "wenxue" || wx.Mode != "" || len(wx.Tools) != 0 {
		t.Fatalf("图文未初始化面形状漂移: %+v", wx)
	}
}

// TestGaeaToolSurfaceModeFollowsConfig 配置档=cost 时 Mode 随登记档（引擎未装配
// 场景取配置默认；已装配场景由 gaeaBuildController 写 ga.runMode——见守卫矩阵）。
func TestGaeaToolSurfaceModeFollowsConfig(t *testing.T) {
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Office: gaeaConfig.OfficeConfig{RunMode: "cost"}}
	defer func() { ga.cfg = oldCfg }()
	view := (&App{}).GaeaToolSurface()
	if view.Mode != "cost" {
		t.Fatalf("Mode 应随配置档 cost，got %q", view.Mode)
	}
	if !strings.EqualFold(view.Board, "gaea") {
		t.Fatalf("Board 应为 gaea，got %q", view.Board)
	}
}
