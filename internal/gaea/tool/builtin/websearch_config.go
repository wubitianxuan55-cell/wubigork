package builtin

import (
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/netclient"
)

// searchCfg holds the runtime search configuration injected by boot.
// nil means all API-based engines are disabled; only public SearXNG works.
var searchCfg *config.SearchConfig

// searchEngineOrderCfg 是 config 驱动的引擎优先级（[search] engine_order）。
// 空 = 使用默认序 defaultSearchEngineOrder（与改造前硬编码顺序一致）。
var searchEngineOrderCfg []string

// searchProxy holds the runtime network proxy injected by boot. web_search
// requests honour the same auto/env/custom/off proxy modes as web_fetch.
var searchProxy netclient.ProxySpec

// SetSearchConfig injects the search configuration from boot assembly.
// Call once before any web_search tool execution.
func SetSearchConfig(cfg config.SearchConfig) {
	cp := cfg
	searchCfg = &cp
}

// SetSearchEngineOrder 注入搜索引擎优先级（3.0 Step 3d #1：config 可配序）。
// kinds 为 SearchEngineKinds() 中的子集；空或 nil 恢复默认序。
// 由 boot 从 [search] engine_order 装配；切换引擎顺序只改配置、代码零改动。
func SetSearchEngineOrder(kinds []string) {
	searchEngineOrderCfg = append([]string(nil), kinds...)
}

// SetSearchProxy injects the network proxy used by web_search requests.
func SetSearchProxy(spec netclient.ProxySpec) {
	searchProxy = spec
}
