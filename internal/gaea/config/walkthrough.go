package config

import (
	"log/slog"
	"os"
	"path/filepath"
)

// WalkthroughWorkspaceDir 返回走查沙箱工作区固定路径（可复核，整体删除即清场）。
func WalkthroughWorkspaceDir() string {
	return filepath.Join(os.TempDir(), "gaea-walkthrough", "workspace")
}

// ApplyWalkthroughOverride 在 GAEA_WALKTHROUGH=1 时把 cfg.Workspace 强制指向
// 走查沙箱。真机走查隔离（v4.418，2026-09-26 走查险情后的防复发刀）：强制
// workspace 指向独立沙箱目录，会话/记忆/交付物与用户数据彻底解耦——首条消息
// 自动装配+自动恢复会把回合接进用户最新真实会话，专用走查会话方案对首次装配
// 无效（NewSession 在引擎装配前必然诚实报错），只有从根上换 workspace 才封得住。
// 显式设置才生效，且覆盖用户配置的 workspace（走查隔离优先于一切）。
// 消费方：config.Load()（agent/无头路径）与 app 层 gaeaLoadConfig()（壳内办公
// 引擎自带一套 Default+toml 合并，不经 Load——两处必须同调本助手，2026-09-26
// 实测漏掉壳侧即开关失效）。
// v4.428.1 补漏：只换 workspace 封不住 UserConfigDir 数据面——sin/小说/角色库
// 等的数据根在 %APPDATA%\gaea（os.UserConfigDir），2026-09-29 走查实证沙箱壳
// 直连了用户真实故事（幸而全程只读）。APPDATA 一并重定向到沙箱，此后所有
// os.UserConfigDir 调用（含 whisperDataRoot/chat.db/engines.json）都落在
// Temp\gaea-walkthroughppdata——整目录删除即清场，清场半径与 workspace 一致。
func ApplyWalkthroughOverride(cfg *Config) {
	if cfg == nil || os.Getenv("GAEA_WALKTHROUGH") != "1" {
		return
	}
	dir := WalkthroughWorkspaceDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Warn("创建走查沙箱工作区失败（保持原 workspace）", "error", err)
		return
	}
	cfg.Workspace = dir
	slog.Warn("GAEA_WALKTHROUGH=1：workspace 已强制指向走查沙箱", "dir", dir)
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		sandbox := filepath.Join(filepath.Dir(dir), "appdata")
		if err := os.MkdirAll(sandbox, 0o755); err != nil {
			slog.Warn("创建走查沙箱 APPDATA 失败（UserConfigDir 数据面未隔离！）", "error", err)
			return
		}
		if err := os.Setenv("APPDATA", sandbox); err != nil {
			slog.Warn("重定向 APPDATA 失败（UserConfigDir 数据面未隔离！）", "error", err)
			return
		}
		slog.Warn("GAEA_WALKTHROUGH=1：APPDATA 已重定向到走查沙箱（UserConfigDir 数据面同隔离）", "from", appdata, "to", sandbox)
	}
}
