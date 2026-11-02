package agent

import (
	"fmt"
	"strings"
	"sync"
)

// task 子代理按调用路由的授权清单（2026-11，dsh model-selection 蒸馏 P2）。
//
// 形态取道不取器：上游 modelSelection/modelDiscovery 分离+创建时授权记录+
// list_subagent_models 发现工具+会话投影采样——gaea 的授权=配置层一次性
// 清单（[agent] subagent_routes），「发现」=task CompactDescription 动态
// 列出授权路由（零新增模型面工具、零注册表增长）。语义照单收两条纪律：
// 「无授权政策时只许父路由」（清单空=路由整体关闭，engine/model 参数一律
// 拒绝）与「显式未授权委派必须失败」（fail-loud 点名可用路由，绝不静默
// 回落——静默降级=模型以为切过去了，本仓既铲过的坑，GaeaSetModel 同纪律）。
//
// 窗口：按调用路由不换压缩窗口（引擎登记表无窗口元数据，刀前核实见 022
// 档 K1）——继承父窗口，切小窗引擎后由 K1 溢出错误自学习兜真实窗口。
// 计价：继承父 pricing（子代理 usage 记账口径不变；跨引擎差价属模型中心
// 统计面，不进子代理账）。

var (
	subRoutesMu sync.Mutex
	subRoutes   []string
)

// SetSubagentRoutes 设定授权路由清单（boot 装配点调用）。空白项剔除、保序
// 去重；nil/空=路由关闭。
func SetSubagentRoutes(routes []string) {
	subRoutesMu.Lock()
	defer subRoutesMu.Unlock()
	seen := make(map[string]bool, len(routes))
	out := make([]string, 0, len(routes))
	for _, r := range routes {
		r = strings.TrimSpace(r)
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	subRoutes = out
}

// SubagentRoutes 返回授权路由快照（CompactDescription 渲染与测试用）。
func SubagentRoutes() []string {
	subRoutesMu.Lock()
	defer subRoutesMu.Unlock()
	out := make([]string, len(subRoutes))
	copy(out, subRoutes)
	return out
}

// AuthorizeSubagentRoute 校验 "engine/model" 形式的按调用路由授权。
func AuthorizeSubagentRoute(route string) error {
	route = strings.TrimSpace(route)
	subRoutesMu.Lock()
	defer subRoutesMu.Unlock()
	if len(subRoutes) == 0 {
		return fmt.Errorf("task engine/model routing is not authorized (configure [agent] subagent_routes in gaea.toml); the sub-agent inherits the parent route")
	}
	for _, r := range subRoutes {
		if r == route {
			return nil
		}
	}
	return fmt.Errorf("route %q is not authorized for sub-agent delegation (authorized: %s)", route, strings.Join(subRoutes, ", "))
}
