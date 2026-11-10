package boot

// A2 动态工具目录装配（dsh-ops 蒸馏 2026-11-10）：[agent] dynamic_tools 三态。
//
//   auto（缺省）：仅当 MCP 工具 schema 超阈值时把 mcp__* 转为可发现、其余全
//     常驻——标准会话零变化（工具面 10-15K 且前缀缓存稳定命中；激活一次=整段
//     前缀缓存失效，长会话默认全开可能净亏——REA 式 60K MCP schema 的重载面
//     才是无争议收益，auto 精确命中该场景）；
//   on：激进两级（核心+build-internal meta 常驻，其余可发现）——A/B 评测用；
//   off：关闭。
//
// discover_tools 仅在目录开启时注册（关闭时零 schema 增量、工具面哈希不变，
// 前缀缓存不受扰）。

import (
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// dynamicMCPThresholdBytes 是 auto 模式的 MCP schema 字节阈值（≈8K tokens，
// cache_shape 的 4B/tok 口径）。标准面 10-15K 不触发；REA 式 240KB 触发。
const dynamicMCPThresholdBytes = 32 * 1024

// dynamicCoreResident 是 "on" 模式的核心常驻清单（工作区读写/命令/网络/任务
// 协议族——日常主链路零发现成本）。
var dynamicCoreResident = []string{
	"read_file", "write_file", "edit_file", "multi_edit", "edit_lines", "move_file", "replace",
	"grep", "ls", "bash", "bash_output", "kill_shell", "wait", "read_spill",
	"web_fetch", "web_search", "todo_write", "complete_step",
}

// residentToolsFor 按模式算常驻名单（nil/空=目录关闭）。在注册面定稿后调用
// （boot.Build 尾段）；discover_tools 由调用方注册后追加。
func residentToolsFor(mode string, reg *tool.Registry) []string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "off":
		return nil
	case "on":
		seen := map[string]bool{}
		var out []string
		add := func(n string) {
			if _, ok := reg.Get(n); ok && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
		for _, n := range dynamicCoreResident {
			add(n)
		}
		// build-internal meta 族全常驻（ask/exit_plan_mode/request_permission/
		// task/run_skill/send_message 等宿主协议——漏挂会卡住协议性路径）。
		for _, n := range reg.Names() {
			if e, ok := tool.RegistryMap[n]; ok && e.Source == "build-internal" {
				add(n)
			}
		}
		return out
	default: // auto / 空 / 未知值（未知值如实回落 auto 并记日志由装配点负责）
		mcpBytes := 0
		for _, s := range reg.Schemas() {
			if strings.HasPrefix(s.Name, "mcp__") {
				mcpBytes += len(s.Name) + len(s.Description) + len(s.Parameters)
			}
		}
		if mcpBytes <= dynamicMCPThresholdBytes {
			return nil
		}
		// 重载面：非 MCP 全常驻（含隐藏可调面——Names 不滤 hidden），MCP 可发现。
		var out []string
		for _, n := range reg.Names() {
			if !strings.HasPrefix(n, "mcp__") {
				out = append(out, n)
			}
		}
		return out
	}
}

// wireDynamicTools 在注册面定稿后装配目录：算名单→注册 discover_tools→
// 装进 runner。返回是否开启（记档用）。
func wireDynamicTools(mode string, reg *tool.Registry, executor *agent.AgentRunner) bool {
	resident := residentToolsFor(mode, reg)
	if len(resident) == 0 {
		return false
	}
	reg.Add(agent.NewDiscoverToolsTool())
	executor.SetResidentTools(append(resident, agent.DiscoverToolName))
	return true
}
