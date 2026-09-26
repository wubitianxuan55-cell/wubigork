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
}
