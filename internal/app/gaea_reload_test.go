package app

import (
	"context"
	"os"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// TestGaeaReloadHotLoadsConfig 验证热加载接口：磁盘上的持久化配置被外部
// 修改后，GaeaReload 重新读取并重建 controller，无需重启桌面端即生效；
// 同时返回重建后的工具/技能数量供前端展示。
func TestGaeaReloadHotLoadsConfig(t *testing.T) {
	oldwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldwd) }()
	_ = os.Chdir(t.TempDir())

	// 指向临时用户配置目录，避免污染真实配置（同 TestGaeaConfigPersistRoundTrip）。
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	bridge.SetClient(ai.NewClient(config.Load()))

	// 首次持久化一份办公引擎配置（温度 0.2），模拟用户设置面板保存的结果。
	seed := gaeaConfig.Default()
	seed.DefaultModel = "gaea"
	seed.Providers = []gaeaConfig.ProviderEntry{{
		Name:          "gaea",
		Kind:          "wubigrok",
		Model:         "",
		ContextWindow: 1_000_000,
	}}
	seed.Tools.Enabled = nil
	seed.Sandbox.Bash = "off"
	seed.Agent.Temperature = 0.2
	if err := gaeaConfig.Save(seed); err != nil {
		t.Fatalf("种子配置保存失败: %v", err)
	}

	a := &App{core: &core{
		ctx:    context.Background(),
		cfg:    config.Load(),
		client: ai.NewClient(config.Load()),
	}}

	// 快照全局 gaea 运行时，测试结束恢复，避免污染同包其他测试。
	ga.mu.Lock()
	oldCtrl, oldCfg := ga.ctrl, ga.cfg
	ga.ctrl, ga.cfg = nil, nil
	ga.mu.Unlock()
	defer func() {
		ga.mu.Lock()
		defer ga.mu.Unlock()
		if ga.ctrl != nil && ga.ctrl != oldCtrl {
			ga.ctrl.Close()
		}
		ga.ctrl, ga.cfg = oldCtrl, oldCfg
		// 关闭临时配置目录下的 Hephaestus.db 单例连接，否则 Windows 上
		// t.TempDir 清理会因文件被占用而失败。
		_ = db.CloseDatabase(gaeaConfig.MemoryUserDir())
	}()

	if err := a.GaeaInit(); err != nil {
		t.Fatalf("GaeaInit 失败: %v", err)
	}
	ga.mu.Lock()
	initialTemp := ga.cfg.Agent.Temperature
	ga.mu.Unlock()
	if initialTemp != 0.2 {
		t.Fatalf("初始 Temperature = %v, want 0.2", initialTemp)
	}

	// 模拟外部编辑：直接改磁盘上的持久化配置再保存。
	disk, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("读取磁盘配置失败: %v", err)
	}
	disk.Agent.Temperature = 0.85
	if err := gaeaConfig.Save(disk); err != nil {
		t.Fatalf("磁盘配置保存失败: %v", err)
	}

	res, err := a.GaeaReload()
	if err != nil {
		t.Fatalf("GaeaReload 失败: %v", err)
	}

	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.cfg.Agent.Temperature != 0.85 {
		t.Errorf("热加载后 Temperature = %v, want 0.85", ga.cfg.Agent.Temperature)
	}
	if ga.ctrl == nil {
		t.Fatal("热加载后 controller 为 nil")
	}
	if res.Skills == 0 {
		t.Errorf("热加载结果 Skills = %d, want > 0", res.Skills)
	}
	if res.Tools == 0 {
		t.Errorf("热加载结果 Tools = %d, want > 0", res.Tools)
	}
}
